package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config 保存所有应用配置。
type Config struct {
	// 服务器设置
	Host string `json:"host"`
	Port int    `json:"port"`

	// Palworld 服务器路径
	PalServerDir     string `json:"palServerDir"`
	PalServerBinary  string `json:"palServerBinary"`
	ConfigDir        string `json:"configDir"`
	SaveDir          string `json:"saveDir"`
	LogDir           string `json:"logDir"`
	PanelLogDir      string `json:"panelLogDir"`

	// SteamCMD 更新管理
	SteamCMDPath  string `json:"steamCmdPath"`
	PalServerAppID string `json:"palServerAppId"`

	// REST API（替代已弃用的 RCON）
	RestAPIHost     string `json:"restApiHost"`
	RestAPIPort     int    `json:"restApiPort"`
	RestAPIPassword string `json:"restApiPassword"`

	// 服务器启动参数
	ServerPort      int    `json:"serverPort"`
	MaxPlayers      int    `json:"maxPlayers"`
	ExtraArgs       string `json:"extraArgs"`
	PublicIP        string `json:"publicIp"`
	UsePublicLobby  bool   `json:"usePublicLobby"`

	// 认证
	AdminToken string `json:"adminToken"`

	// 定时任务
	AutoRestartEnabled  bool   `json:"autoRestartEnabled"`
	AutoRestartCron     string `json:"autoRestartCron"`
	AutoUpdateEnabled   bool   `json:"autoUpdateEnabled"`
	AutoUpdateCron      string `json:"autoUpdateCron"`
	AutoSaveEnabled     bool   `json:"autoSaveEnabled"`
	AutoSaveCron        string `json:"autoSaveCron"`
}

// DefaultConfig 返回一个适用于 Linux 的合理默认配置。
func DefaultConfig() *Config {
	return &Config{
		Host:              "0.0.0.0",
		Port:              8080,
		PalServerDir:      "/home/steam/.steam/steam/steamapps/common/PalServer",
		PalServerBinary:   "PalServer.sh",
		ConfigDir:         "",
		SaveDir:           "",
		LogDir:            "",
		PanelLogDir:       "/opt/palworld-manager/logs",
		SteamCMDPath:      "/usr/games/steamcmd",
		PalServerAppID:    "2394010",
		RestAPIHost:       "127.0.0.1",
		RestAPIPort:       8212,
		RestAPIPassword:   "",
		ServerPort:        8211,
		MaxPlayers:        32,
		ExtraArgs:         "-useperfthreads -NoAsyncLoadingThread -UseMultithreadForDS",
		PublicIP:          "",
		UsePublicLobby:    false,
		AdminToken:        "admin",
		AutoRestartEnabled: false,
		AutoRestartCron:   "0 */6 * * *",
		AutoUpdateEnabled:  false,
		AutoUpdateCron:    "0 4 * * *",
		AutoSaveEnabled:   false,
		AutoSaveCron:      "*/15 * * * *",
	}
}

// Load 从 JSON 文件读取配置。如果文件不存在，则使用默认值创建文件并返回默认配置。
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// 写入默认配置
			if saveErr := Save(path, cfg); saveErr != nil {
				return cfg, saveErr
			}
			return cfg, nil
		}
		return cfg, err
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return cfg, err
	}

	// ConfigDir 固定为 <PalServerDir>/Pal/Saved/Config/LinuxServer
	cfg.ConfigDir = filepath.Join(cfg.PalServerDir, "Pal", "Saved", "Config", "LinuxServer")
	if cfg.SaveDir == "" {
		cfg.SaveDir = filepath.Join(cfg.PalServerDir, "Pal", "Saved", "SaveGames")
	}
	if cfg.LogDir == "" {
		cfg.LogDir = filepath.Join(cfg.PalServerDir, "Pal", "Saved", "Logs")
	}

	return cfg, nil
}

// Save 将配置写入 JSON 文件。
func Save(path string, cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

