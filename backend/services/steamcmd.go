package services

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

// BuildInfo 保存 SteamCMD 构建信息。
type BuildInfo struct {
	AppID      string     `json:"appId"`
	BuildID    string     `json:"buildId"`
	Branch     string     `json:"branch"`
	Version    string     `json:"version"`
	LastUpdate *time.Time `json:"lastUpdate,omitempty"`
}

// UpdateResult 保存更新操作的结果。
type UpdateResult struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	BuildID  string `json:"buildId,omitempty"`
	Duration string `json:"duration"`
	Output   string `json:"output,omitempty"`
}

// SteamCMDService 管理 Palworld 服务器的 SteamCMD 操作。
type SteamCMDService struct {
	mu           sync.Mutex
	steamcmdPath string
	appID        string
	serverDir    string
	lastUpdate   *time.Time
	isUpdating     bool
	currentBuild   string
	currentVersion string
}

// NewSteamCMDService 创建一个新的 SteamCMDService。
func NewSteamCMDService(steamcmdPath, appID, serverDir string) *SteamCMDService {
	return &SteamCMDService{
		steamcmdPath: steamcmdPath,
		appID:        appID,
		serverDir:    serverDir,
	}
}

// CheckForUpdate 检查是否有更新的构建版本可用，但不进行下载。
func (s *SteamCMDService) CheckForUpdate() (*BuildInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log := Log("SteamCMD")
	log.Info("检查更新 (appid=%s)", s.appID)

	cmd := exec.Command(s.steamcmdPath,
		"+login", "anonymous",
		"+app_info_print", s.appID,
		"+quit",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Error("检查更新失败: %v", err)
		log.Output("steamcmd 输出", string(output))
		return nil, fmt.Errorf("steamcmd check failed: %w\nOutput: %s", err, string(output))
	}

	latest := s.parseBuildInfo(string(output))
	if latest == nil {
		latest = &BuildInfo{AppID: s.appID}
	}

	log.Info("检查更新完成 (buildid=%s)", latest.BuildID)
	return latest, nil
}

// Update 运行 steamcmd 来更新 Palworld 服务器。
func (s *SteamCMDService) Update(validate bool) (*UpdateResult, error) {
	s.mu.Lock()
	if s.isUpdating {
		s.mu.Unlock()
		return nil, fmt.Errorf("an update is already in progress")
	}
	s.isUpdating = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.isUpdating = false
		s.mu.Unlock()
	}()

	log := Log("SteamCMD")
	log.Info("开始更新/安装 (appid=%s, dir=%s, validate=%v)", s.appID, s.serverDir, validate)

	startTime := time.Now()

	args := []string{
		"+login", "anonymous",
		"+app_update", s.appID,
	}
	if validate {
		args = append(args, "validate")
	}
	args = append(args, "+quit")

	cmd := exec.Command(s.steamcmdPath, args...)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)
	duration := time.Since(startTime).Round(time.Second).String()

	result := &UpdateResult{
		Duration: duration,
		Output:   outputStr,
	}

	if err != nil {
		result.Success = false
		result.Message = fmt.Sprintf("Update failed: %v", err)
		log.Error("更新/安装失败 (耗时 %s): %v", duration, err)
		log.Output("steamcmd 输出", outputStr)
		return result, nil
	}

	buildID := s.parseManifestBuildID()
	result.BuildID = buildID
	result.Success = true
	result.Message = "Update completed successfully"

	log.Info("更新/安装完成 (buildid=%s, 耗时 %s)", buildID, duration)

	if buildID != "" {
		s.mu.Lock()
		s.currentBuild = buildID
		now := time.Now()
		s.lastUpdate = &now
		s.mu.Unlock()
	}

	return result, nil
}

// GetInfo 返回当前安装的信息。
func (s *SteamCMDService) GetInfo() *BuildInfo {
	info := &BuildInfo{AppID: s.appID}

	s.mu.Lock()
	info.BuildID = s.currentBuild
	info.Version = s.currentVersion
	if s.lastUpdate != nil {
		info.LastUpdate = s.lastUpdate
	}
	s.mu.Unlock()

	return info
}

// RefreshInfo 通过 REST API 获取当前运行的服务端版本，更新 internal state。
func (s *SteamCMDService) RefreshInfo(restClient *RestAPIClient) {
	if restClient == nil {
		return
	}
	serverInfo, err := restClient.GetInfo()
	if err != nil {
		return
	}

	s.mu.Lock()
	s.currentVersion = serverInfo.Version
	s.mu.Unlock()
}

// IsUpdating 如果有更新正在进行，返回 true。
func (s *SteamCMDService) IsUpdating() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.isUpdating
}

func (s *SteamCMDService) parseBuildInfo(output string) *BuildInfo {
	re := regexp.MustCompile(`"buildid"\s*"(\d+)"`)
	matches := re.FindStringSubmatch(output)
	info := &BuildInfo{
		AppID: s.appID,
	}
	if len(matches) >= 2 {
		info.BuildID = matches[1]
	}

	branchRe := regexp.MustCompile(`"branch"\s*"([^"]+)"`)
	branchMatches := branchRe.FindStringSubmatch(output)
	if len(branchMatches) >= 2 {
		info.Branch = branchMatches[1]
	}

	return info
}

func (s *SteamCMDService) parseManifestBuildID() string {
	manifestPath := fmt.Sprintf("%s/steamapps/appmanifest_%s.acf", s.serverDir, s.appID)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return ""
	}

	re := regexp.MustCompile(`"buildid"\s*"(\d+)"`)
	matches := re.FindStringSubmatch(string(data))
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// readManifest 从 Valve ACF 清单文件中读取键值对。
func readManifest(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	re := regexp.MustCompile(`"([^"]+)"\s*"([^"]*)"`)
	matches := re.FindAllStringSubmatch(string(data), -1)
	for _, m := range matches {
		if len(m) >= 3 {
			result[strings.TrimSpace(m[1])] = strings.TrimSpace(m[2])
		}
	}
	return result, nil
}

// SteamcmdPath 返回当前 SteamCMD 的路径。
func (s *SteamCMDService) SteamcmdPath() string {
	return s.steamcmdPath
}

// IsSteamCMDInstalled 检查 SteamCMD 是否已安装。
func (s *SteamCMDService) IsSteamCMDInstalled() bool {
	if _, err := os.Stat(s.steamcmdPath); err == nil {
		return true
	}
	return s.FindSteamCMD()
}

// FindSteamCMD 在系统中搜索 SteamCMD 并更新路径（Debian 专用）。
func (s *SteamCMDService) FindSteamCMD() bool {
	// Debian/Ubuntu 通过 apt 安装的默认路径
	for _, p := range []string{"/usr/games/steamcmd", "/usr/bin/steamcmd"} {
		if _, err := os.Stat(p); err == nil {
			s.steamcmdPath = p
			return true
		}
	}
	if path, err := exec.LookPath("steamcmd"); err == nil {
		s.steamcmdPath = path
		return true
	}
	return false
}

