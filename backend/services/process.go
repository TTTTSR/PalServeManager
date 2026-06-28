package services

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ServerStatus 表示 Palworld 服务器的当前状态。
type ServerStatus string

const (
	StatusRunning      ServerStatus = "running"
	StatusStopped      ServerStatus = "stopped"
	StatusStarting     ServerStatus = "starting"
	StatusStopping     ServerStatus = "stopping"
	StatusNotInstalled ServerStatus = "not_installed"
	StatusInstalling   ServerStatus = "installing"
	StatusError        ServerStatus = "error"
)

// ProcessInfo 包含运行中服务器进程的信息。
type ProcessInfo struct {
	PID             int          `json:"pid"`
	Status          ServerStatus `json:"status"`
	Installed       bool         `json:"installed"`
	SteamCMDReady   bool         `json:"steamcmdReady"`
	StartTime       *time.Time   `json:"startTime,omitempty"`
	Uptime          string       `json:"uptime,omitempty"`
	Port            int          `json:"port"`
	MaxPlayers      int          `json:"maxPlayers"`
}

// ProcessManager 管理 Palworld 服务器进程的生命周期。
type ProcessManager struct {
	mu sync.Mutex

	palServerDir    string
	palServerBinary string
	serverPort      int
	maxPlayers      int
	extraArgs       string
	publicIP        string
	usePublicLobby  bool

	cmd       *exec.Cmd
	pid       int
	startTime *time.Time
	status    ServerStatus

	// steamcmdChecker 检查 SteamCMD 是否安装（可选回调）
	steamcmdChecker func() bool

	// 用于启动后 API 可达性检查
	restClient  *RestAPIClient
	managerAddr string
}

// NewProcessManager 创建一个新的 ProcessManager。
func NewProcessManager(serverDir, binary string, port, maxPlayers int, extraArgs, publicIP string, usePublicLobby bool) *ProcessManager {
	return &ProcessManager{
		palServerDir:    serverDir,
		palServerBinary: binary,
		serverPort:      port,
		maxPlayers:      maxPlayers,
		extraArgs:       extraArgs,
		publicIP:        publicIP,
		usePublicLobby:  usePublicLobby,
		status:          StatusStopped,
	}
}

// Start 启动 Palworld 服务器。
func (pm *ProcessManager) Start() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	log := Log("Server")

	if !pm.IsInstalled() {
		return fmt.Errorf("server is not installed. Please install the server first")
	}

	if pm.status == StatusRunning {
		return fmt.Errorf("server is already running (PID: %d)", pm.pid)
	}

	pm.status = StatusStarting

	binaryPath := filepath.Join(pm.palServerDir, pm.palServerBinary)
	args := []string{
		fmt.Sprintf("-port=%d", pm.serverPort),
		fmt.Sprintf("-players=%d", pm.maxPlayers),
	}

	if pm.usePublicLobby {
		args = append(args, "-publiclobby")
	}
	if pm.publicIP != "" {
		args = append(args, fmt.Sprintf("-publicip=%s", pm.publicIP))
	}
	if pm.extraArgs != "" {
		args = append(args, strings.Fields(pm.extraArgs)...)
	}

	log.Info("启动服务器: %s %v", binaryPath, args)

	pm.cmd = exec.Command(binaryPath, args...)
	pm.cmd.Dir = pm.palServerDir

	// 时间戳日志（PalServer 标准路径）
	palLogDir := filepath.Join(pm.palServerDir, "Pal", "Saved", "Logs")
	os.MkdirAll(palLogDir, 0755)
	logPath := filepath.Join(palLogDir, fmt.Sprintf("server-%s.log", time.Now().Format("2006-01-02-150405")))
	palLogFile, err := os.Create(logPath)
	if err != nil {
		pm.status = StatusError
		log.Error("创建日志文件失败: %v", err)
		return fmt.Errorf("failed to create log file: %w", err)
	}

	// 固定名称控制台日志（管理器 logs 目录，每次启动截断）
	mgrLogDir := LogDir()
	os.MkdirAll(mgrLogDir, 0755)
	consolePath := filepath.Join(mgrLogDir, "server-console.log")
	consoleFile, err := os.OpenFile(consolePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		palLogFile.Close()
		pm.status = StatusError
		log.Error("创建控制台日志文件失败: %v", err)
		return fmt.Errorf("failed to create console log: %w", err)
	}

	log.Info("服务器日志: %s", logPath)
	log.Info("控制台日志: %s", consolePath)

	// Tee: 同时写入 PalServer 时间戳日志 + 控制台日志
	pm.cmd.Stdout = io.MultiWriter(palLogFile, consoleFile)
	pm.cmd.Stderr = io.MultiWriter(palLogFile, consoleFile)

	if err := pm.cmd.Start(); err != nil {
		pm.status = StatusError
		palLogFile.Close()
		consoleFile.Close()
		log.Error("启动失败: %v", err)
		return fmt.Errorf("failed to start server: %w", err)
	}

	pm.pid = pm.cmd.Process.Pid
	now := time.Now()
	pm.startTime = &now
	pm.status = StatusRunning

	log.Info("服务器启动成功 (PID: %d, 端口: %d, 最大玩家: %d)", pm.pid, pm.serverPort, pm.maxPlayers)

	// 捕获变量供 goroutine 使用（避免竞态）
	restClient := pm.restClient
	managerAddr := pm.managerAddr
	gamePort := pm.serverPort

	go func() {
		err := pm.cmd.Wait()
		palLogFile.Close()
		consoleFile.Close()
		pm.mu.Lock()
		defer pm.mu.Unlock()
		if pm.status == StatusRunning {
			if err != nil {
				log.Error("服务器进程异常退出 (PID: %d): %v", pm.pid, err)
				pm.status = StatusError
				BroadcastStatusGlobal("error")
			} else {
				log.Info("服务器进程正常退出 (PID: %d)", pm.pid)
				pm.status = StatusStopped
				BroadcastStatusGlobal("stopped")
			}
			pm.pid = 0
		}
	}()

	// 启动后 API 可达性检查 + WebSocket 广播 running
	go func() {
		pm.runHealthChecks(gamePort, restClient, managerAddr)
		BroadcastStatusGlobal("running")
	}()

	return nil
}

// Stop 停止 Palworld 服务器。
// 优先通过 REST API 优雅关闭，失败时回退到信号/强制终止。
func (pm *ProcessManager) Stop() error {
	pm.mu.Lock()

	log := Log("Server")

	if pm.status != StatusRunning {
		pm.mu.Unlock()
		return fmt.Errorf("server is not running")
	}

	log.Info("正在停止服务器 (PID: %d)...", pm.pid)
	pm.status = StatusStopping

	if pm.cmd == nil || pm.cmd.Process == nil {
		pm.status = StatusStopped
		pm.mu.Unlock()
		log.Info("服务器已停止（无活动进程）")
		return nil
	}

	restClient := pm.restClient
	cmd := pm.cmd
	pid := pm.pid
	pm.mu.Unlock()

	// Step 1: 通过 REST API 优雅关闭
	restUsed := false
	if restClient != nil {
		log.Info("通过 REST API 发送优雅关闭命令...")
		if err := restClient.Announce("Server is shutting down..."); err != nil {
			log.Info("  发送公告失败: %v (继续...)", err)
		}
		if err := restClient.SaveGame(); err != nil {
			log.Info("  保存游戏失败: %v (继续...)", err)
		}
		if err := restClient.Shutdown(30, "Server shutting down in 30 seconds"); err != nil {
			log.Error("  REST API 关闭命令失败: %v，回退到信号方式", err)
		} else {
			restUsed = true
			log.Info("  REST API 关闭命令已发送，等待服务器自行退出...")
		}
	} else {
		log.Info("REST API 未配置，使用信号方式关闭...")
	}

	// Step 2: 如果 REST API 不可用，发送 SIGINT 作为备用
	if !restUsed {
		if err := signalProcess(cmd.Process, os.Interrupt); err != nil {
			pm.mu.Lock()
			pm.status = StatusStopped
			pm.pid = 0
			pm.mu.Unlock()
			log.Info("服务器已停止")
			return nil
		}
	}

	// Step 3: 等待进程退出
	timeout := 30 * time.Second
	if restUsed {
		timeout = 90 * time.Second // REST API 倒计时 + 存档需要更多时间
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-done:
		log.Info("服务器已优雅停止")
	case <-time.After(timeout):
		log.Info("优雅停止超时 (%v)，强制终止 (PID: %d)", timeout, pid)
		killProcess(cmd.Process)
		<-done
		log.Info("服务器已强制终止")
	}

	pm.mu.Lock()
	pm.status = StatusStopped
	pm.pid = 0
	pm.cmd = nil
	pm.mu.Unlock()

	BroadcastStatusGlobal("stopped")
	return nil
}

// Restart 停止然后重新启动服务器。
func (pm *ProcessManager) Restart() error {
	log := Log("Server")
	log.Info("开始重启服务器...")
	if pm.GetInfo().Status == StatusRunning {
		if err := pm.Stop(); err != nil {
			log.Error("重启失败（停止阶段）: %v", err)
			return fmt.Errorf("failed to stop server during restart: %w", err)
		}
		// 等待端口释放
		time.Sleep(3 * time.Second)
	}
	return pm.Start()
}

// GetInfo 返回当前服务器进程的信息。
func (pm *ProcessManager) GetInfo() *ProcessInfo {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	installed := pm.IsInstalled()

	status := pm.status
	if !installed {
		status = StatusNotInstalled
	}

	steamcmdReady := false
	if pm.steamcmdChecker != nil {
		steamcmdReady = pm.steamcmdChecker()
	}

	info := &ProcessInfo{
		PID:           pm.pid,
		Status:        status,
		Installed:     installed,
		SteamCMDReady: steamcmdReady,
		StartTime:     nil,
		Uptime:        "",
		Port:          pm.serverPort,
		MaxPlayers:    pm.maxPlayers,
	}

	if pm.startTime != nil && pm.status == StatusRunning {
		info.StartTime = pm.startTime
		info.Uptime = formatDuration(time.Since(*pm.startTime))
	}

	return info
}

// IsInstalled 检查 Palworld 服务器二进制文件是否存在。
func (pm *ProcessManager) IsInstalled() bool {
	binaryPath := filepath.Join(pm.palServerDir, pm.palServerBinary)
	_, err := os.Stat(binaryPath)
	return err == nil
}

// SetInstalling 将状态设置为安装中（用于安装期间的用户界面反馈）。
func (pm *ProcessManager) SetInstalling() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.status = StatusInstalling
}

// SetInstalled 在安装完成后更新状态。
func (pm *ProcessManager) SetInstalled() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.status = StatusStopped
}

// SetSteamCMDChecker 设置 SteamCMD 检测回调。
func (pm *ProcessManager) SetSteamCMDChecker(checker func() bool) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.steamcmdChecker = checker
}

// SetRestClient 注入 REST API 客户端，用于启动后可达性检查。
func (pm *ProcessManager) SetRestClient(client *RestAPIClient) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.restClient = client
}

// SetManagerAddr 设置控制面板自身监听地址，用于自检 API。
func (pm *ProcessManager) SetManagerAddr(addr string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.managerAddr = addr
}

// IsRunning 如果服务器当前正在运行则返回 true。
func (pm *ProcessManager) IsRunning() bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.status == StatusRunning
}

// UpdateConfig 更新服务器启动参数（在下次重启时生效）。
func (pm *ProcessManager) UpdateConfig(port, maxPlayers int, extraArgs, publicIP string, usePublicLobby bool) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.serverPort = port
	pm.maxPlayers = maxPlayers
	pm.extraArgs = extraArgs
	pm.publicIP = publicIP
	pm.usePublicLobby = usePublicLobby
}

// GetLogs 返回服务器日志文件的尾部内容。
func (pm *ProcessManager) GetLogs(lines int) ([]string, error) {
	logDir := filepath.Join(pm.palServerDir, "Pal", "Saved", "Logs")
	entries, err := os.ReadDir(logDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read log directory: %w", err)
	}

	// 查找最新的日志文件
	var latestLog string
	var latestTime time.Time
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), "server-") && strings.HasSuffix(entry.Name(), ".log") {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			if info.ModTime().After(latestTime) {
				latestTime = info.ModTime()
				latestLog = filepath.Join(logDir, entry.Name())
			}
		}
	}

	if latestLog == "" {
		latestLog = filepath.Join(logDir, "Pal.log")
	}

	return tailFile(latestLog, lines)
}

// tailFile 读取文件的最后 N 行。
func tailFile(path string, n int) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lineBuf []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		lineBuf = append(lineBuf, scanner.Text())
	}

	if len(lineBuf) > n {
		lineBuf = lineBuf[len(lineBuf)-n:]
	}

	return lineBuf, scanner.Err()
}

// runHealthChecks 在服务器启动后检查所有 API 可达性。
func (pm *ProcessManager) runHealthChecks(gamePort int, restClient *RestAPIClient, managerAddr string) {
	log := Log("Server")
	log.Info("=== 开始 API 可达性检查 ===")

	pass, fail := 0, 0

	// ---------- A. Palworld 服务端 ----------
	log.Info("[1/2] Palworld 服务端 API:")

	// 等待 TCP 端口
	tcpAddr := fmt.Sprintf("127.0.0.1:%d", gamePort)
	tcpStart := time.Now()
	tcpOK := false
	for i := 0; i < 20; i++ {
		conn, err := net.DialTimeout("tcp", tcpAddr, 2*time.Second)
		if err == nil {
			conn.Close()
			tcpOK = true
			break
		}
		time.Sleep(3 * time.Second)
	}
	if tcpOK {
		log.Info("  ✓ TCP 端口 %d 已可达 (%.1fs)", gamePort, time.Since(tcpStart).Seconds())
		pass++
	} else {
		log.Error("  ✗ TCP 端口 %d 不可达 (超时 60s)", gamePort)
		fail++
	}

	// 等待 REST API
	if restClient != nil {
		restStart := time.Now()
		restOK := false
		for i := 0; i < 10; i++ {
			if err := restClient.Ping(); err == nil {
				restOK = true
				break
			}
			time.Sleep(3 * time.Second)
		}
		if restOK {
			log.Info("  ✓ REST API /v1/api/info 正常 (%.1fs)", time.Since(restStart).Seconds())
			pass++
		} else {
			log.Error("  ✗ REST API /v1/api/info 不可达 (超时 30s)")
			fail++
		}
	} else {
		log.Info("  - REST API 未配置，跳过检查")
	}

	// ---------- B. 控制面板后端 ----------
	log.Info("[2/2] 控制面板后端 API:")
	managerAPIs := []string{
		"/api/health",
		"/api/server/status",
		"/api/server/players",
		"/api/server/info",
		"/api/config",
		"/api/config/defaults",
		"/api/update/check",
		"/api/update/status",
		"/api/monitor/stats",
		"/api/monitor/system",
		"/api/monitor/server",
		"/api/logs",
		"/api/schedule",
	}

	httpClient := &http.Client{Timeout: 5 * time.Second}
	baseURL := fmt.Sprintf("http://%s", managerAddr)

	for _, path := range managerAPIs {
		url := baseURL + path
		resp, err := httpClient.Get(url)
		if err != nil {
			log.Error("  ✗ GET %s → 连接失败: %v", path, err)
			fail++
			continue
		}
		resp.Body.Close()
		if resp.StatusCode >= 500 {
			log.Error("  ✗ GET %s → %d", path, resp.StatusCode)
			fail++
		} else {
			log.Info("  ✓ GET %s → %d", path, resp.StatusCode)
			pass++
		}
	}

	// ---------- 汇总 ----------
	total := pass + fail
	if fail == 0 {
		log.Info("=== 检查完成: 全部通过 (%d/%d) ===", total, total)
	} else {
		log.Error("=== 检查完成: %d 通过, %d 失败 (%d 项) ===", pass, fail, total)
	}
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
