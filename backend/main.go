package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/gorilla/mux"
	"github.com/robfig/cron/v3"

	"palworldserve/config"
	"palworldserve/handlers"
	"palworldserve/middleware"
	"palworldserve/services"
)

//go:embed frontend-dist/*
var frontendFiles embed.FS

func main() {
	// 确定配置文件路径
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		exePath, _ := os.Executable()
		configPath = filepath.Join(filepath.Dir(exePath), "config.json")
	}

	// 加载配置
	cfg, err := config.Load(configPath)
	if err != nil {
		cfg = config.DefaultConfig()
	}

	// 初始化日志系统（终端 + 文件双输出）
	logDir := cfg.LogDir
	if logDir == "" {
		exePath, _ := os.Executable()
		logDir = filepath.Join(filepath.Dir(exePath), "logs")
	}
	services.InitLogger(logDir)
	defer services.CloseGlobal()

	svcLog := services.Log("System")
	svcLog.Info("Palworld Server Manager 正在初始化...")

	// 初始化服务
	processManager := services.NewProcessManager(
		cfg.PalServerDir,
		cfg.PalServerBinary,
		cfg.ServerPort,
		cfg.MaxPlayers,
		cfg.ExtraArgs,
		cfg.PublicIP,
		cfg.UsePublicLobby,
	)

	steamcmdService := services.NewSteamCMDService(
		cfg.SteamCMDPath,
		cfg.PalServerAppID,
		cfg.PalServerDir,
	)

	// 从 PalServer 配置文件读取 AdminPassword
	restPassword := ""
	if settings, err := services.LoadSettings(cfg.ConfigDir); err == nil {
		restPassword = settings.Settings["AdminPassword"]
	}

	restClient := services.NewRestAPIClient(
		cfg.RestAPIHost,
		cfg.RestAPIPort,
		restPassword,
	)

	// 注入 SteamCMD 检测回调
	processManager.SetSteamCMDChecker(func() bool { return steamcmdService.IsSteamCMDInstalled() })

	// 注入 REST 客户端和管理器地址，用于启动后 API 可达性检查
	processManager.SetRestClient(restClient)
	processManager.SetManagerAddr(fmt.Sprintf("127.0.0.1:%d", cfg.Port))

	// 启动后台状态轮询
	processManager.StartPolling()

	monitor := services.NewMonitor(processManager, restClient, cfg.PalServerDir)

	// 初始化处理器
	serverHandler := handlers.NewServerHandler(processManager, restClient, steamcmdService)
	configHandler := handlers.NewConfigHandler(cfg.ConfigDir)
	updateHandler := handlers.NewUpdateHandler(steamcmdService, processManager)
	monitorHandler := handlers.NewMonitorHandler(monitor)
logHandler := handlers.NewLogHandler(monitor, cfg.PanelLogDir)
	// 存档备份
	backupDir := cfg.BackupDir
	if backupDir == "" {
		exePath, _ := os.Executable()
		backupDir = filepath.Join(filepath.Dir(exePath), "Saved")
	}
	os.MkdirAll(backupDir, 0755)
	backupHandler := handlers.NewBackupHandler(processManager, restClient, cfg.SaveDir, backupDir)

	scheduleSaveCallback := func(newCfg *config.Config) error {
		return config.Save(configPath, newCfg)
	}
	scheduleHandler := handlers.NewScheduleHandler(cfg, scheduleSaveCallback)
	panelConfigHandler := handlers.NewPanelConfigHandler(cfg, scheduleSaveCallback)

	// 初始化 WebSocket Hub
	wsHub := services.GetWSHub()

	// 设置 API 路由器
	api := mux.NewRouter()
	api.Use(middleware.CORS)
	api.Use(middleware.RequestLogging())

	api.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSONStatic(w, http.StatusOK, map[string]string{"status": "ok"})
	}).Methods("GET")

	// 服务器控制
	api.HandleFunc("/server/status", serverHandler.GetStatus).Methods("GET")
	api.HandleFunc("/server/start", serverHandler.Start).Methods("POST")
	api.HandleFunc("/server/stop", serverHandler.Stop).Methods("POST")
	api.HandleFunc("/server/restart", serverHandler.Restart).Methods("POST")
	api.HandleFunc("/server/install", serverHandler.Install).Methods("POST")
	api.HandleFunc("/server/command", serverHandler.SendCommand).Methods("POST")
	api.HandleFunc("/server/players", serverHandler.GetPlayers).Methods("GET")
	api.HandleFunc("/server/info", serverHandler.GetServerInfo).Methods("GET")

	// 配置管理
	api.HandleFunc("/config", configHandler.GetConfig).Methods("GET")
	api.HandleFunc("/config", configHandler.UpdateConfig).Methods("PUT")
	api.HandleFunc("/config/reset", configHandler.ResetConfig).Methods("POST")
	api.HandleFunc("/config/defaults", configHandler.GetDefaultSettings).Methods("GET")

	// 更新管理
	api.HandleFunc("/update/check", updateHandler.CheckUpdate).Methods("GET")
	api.HandleFunc("/update/run", updateHandler.Update).Methods("POST")
	api.HandleFunc("/update/status", updateHandler.GetUpdateStatus).Methods("GET")

	// 监控管理
	api.HandleFunc("/monitor/stats", monitorHandler.GetStats).Methods("GET")
	api.HandleFunc("/monitor/system", monitorHandler.GetSystemStats).Methods("GET")
	api.HandleFunc("/monitor/server", monitorHandler.GetServerStats).Methods("GET")

	// 日志
	api.HandleFunc("/logs", logHandler.GetLogs).Methods("GET")

	// 计划任务
	api.HandleFunc("/schedule", scheduleHandler.GetSchedule).Methods("GET")
	api.HandleFunc("/schedule", scheduleHandler.UpdateSchedule).Methods("PUT")
	// 存档备份
	api.HandleFunc("/backup/list", backupHandler.List).Methods("GET")
	api.HandleFunc("/backup/create", backupHandler.Create).Methods("POST")
	api.HandleFunc("/backup/restore", backupHandler.Restore).Methods("POST")

	// 面板配置
	api.HandleFunc("/panel-config", panelConfigHandler.GetConfig).Methods("GET")
	api.HandleFunc("/panel-config", panelConfigHandler.UpdateConfig).Methods("PUT")

	// WebSocket 状态推送
	api.HandleFunc("/ws", wsHub.HandleWS)

	// 主路由器 - 合并 API 和前端
	mainRouter := mux.NewRouter()

	// 挂载 API（去除 /api 前缀后转发给子路由）
	mainRouter.PathPrefix("/api").Handler(http.StripPrefix("/api", api))

	// 提供前端服务（无认证）
	frontendHandler := http.FileServer(http.FS(mustSub(frontendFiles, "frontend-dist")))
	mainRouter.PathPrefix("/").Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 如果是未匹配的 API 请求，让其正常返回 404
		w.Header().Set("Access-Control-Allow-Origin", "*")
		// 尝试提供文件；如果未找到，则提供 index.html 以支持 SPA 路由
		frontendHandler.ServeHTTP(w, r)
	}))

	// 设置定时任务
	cronScheduler := cron.New()

	schedLog := services.Log("Scheduler")
	if cfg.AutoRestartEnabled && cfg.AutoRestartCron != "" {
		cronScheduler.AddFunc(cfg.AutoRestartCron, func() {
			schedLog.Info("执行自动重启...")
			if processManager.IsRunning() {
				if restClient != nil {
					restClient.Announce("Scheduled restart starting. Server will be back shortly.")
					restClient.SaveGame()
				}
				if err := processManager.Restart(); err != nil {
					schedLog.Error("自动重启失败: %v", err)
				} else {
					schedLog.Info("自动重启完成")
				}
			}
		})
	}

	if cfg.AutoUpdateEnabled && cfg.AutoUpdateCron != "" {
		cronScheduler.AddFunc(cfg.AutoUpdateCron, func() {
			schedLog.Info("执行自动更新检查...")
			current := steamcmdService.GetInfo()
			latest, err := steamcmdService.CheckForUpdate()
			if err != nil {
				schedLog.Error("更新检查失败: %v", err)
				return
			}
			if current.BuildID != latest.BuildID && latest.BuildID != "" {
				schedLog.Info("有可用更新，正在停止服务器...")
				if processManager.IsRunning() {
					if restClient != nil {
						restClient.Announce("Server update available. Restarting for update.")
						restClient.SaveGame()
					}
					processManager.Stop()
				}
				schedLog.Info("运行 SteamCMD 更新...")
				result, _ := steamcmdService.Update(true)
				if result != nil {
					schedLog.Info("更新结果: %s", result.Message)
				}
				schedLog.Info("重新启动服务器...")
				processManager.Start()
			} else {
				schedLog.Info("无可用更新")
			}
		})
	}

	if cfg.AutoSaveEnabled && cfg.AutoSaveCron != "" {
		cronScheduler.AddFunc(cfg.AutoSaveCron, func() {
			if processManager.IsRunning() && restClient != nil {
				if err := restClient.SaveGame(); err != nil {
					schedLog.Error("自动存档失败: %v", err)
				} else {
					schedLog.Info("自动存档完成")
				}
			}
		})
	}

	cronScheduler.Start()

	// 启动 HTTP 服务器
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	svcLog.Info("监听地址: %s", addr)
	svcLog.Info("PalServer 目录: %s", cfg.PalServerDir)
	svcLog.Info("配置目录: %s", cfg.ConfigDir)
	svcLog.Info("自动重启: %v (cron: %s)", cfg.AutoRestartEnabled, cfg.AutoRestartCron)
	svcLog.Info("自动更新: %v (cron: %s)", cfg.AutoUpdateEnabled, cfg.AutoUpdateCron)
	svcLog.Info("服务器启动完成，等待请求...")

	// 管理面板关闭时只停自身，不联动关闭幻兽帕鲁服务器
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		svcLog.Info("收到关闭信号，管理面板正在退出...")
		if processManager.IsRunning() {
			svcLog.Info("幻兽帕鲁服务器保持运行 (PID 可能已变更)，管理面板重启后可重新关联")
		}
		cronScheduler.Stop()
		os.Exit(0)
	}()

	if err := http.ListenAndServe(addr, mainRouter); err != nil {
		svcLog.Error("HTTP 服务器错误: %v", err)
		log.Fatalf("Server error: %v", err)
	}
}

func writeJSONStatic(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func mustSub(efs embed.FS, dir string) fs.FS {
	sub, err := fs.Sub(efs, dir)
	if err != nil {
		panic(fmt.Sprintf("embedded filesystem missing directory %s: %v", dir, err))
	}
	return sub
}
