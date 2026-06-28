package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

	// 推导未显式设置的路径
	if cfg.ConfigDir == "" {
		cfg.ConfigDir = findConfigDir(cfg.PalServerDir)
	} else {
		// 用户指定了 ConfigDir，验证是否存在
		if _, err := os.Stat(filepath.Join(cfg.ConfigDir, "PalWorldSettings.ini")); os.IsNotExist(err) {
			fmt.Printf("[Config] 警告: 指定的 ConfigDir 中未找到 PalWorldSettings.ini: %s\n", cfg.ConfigDir)
		}
	}
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

// findConfigDir 在 PalServerDir 下自动搜索 PalWorldSettings.ini 所在目录。
func findConfigDir(palServerDir string) string {
	// 候选路径，按优先级排列
	candidates := []string{
		filepath.Join(palServerDir, "Pal", "Saved", "Config", "LinuxServer"),
		filepath.Join(palServerDir, "Pal", "Saved", "Config", "WindowsServer"),
		filepath.Join(palServerDir, "Saved", "Config", "LinuxServer"),
		filepath.Join(palServerDir, "Saved", "Config", "WindowsServer"),
	}

	// 先检查候选路径
	for _, dir := range candidates {
		iniPath := filepath.Join(dir, "PalWorldSettings.ini")
		if _, err := os.Stat(iniPath); err == nil {
			fmt.Printf("[Config] 自动发现配置目录: %s\n", dir)
			return dir
		}
	}

	// 候选路径都不存在，递归搜索（限制深度避免扫描整个游戏目录）
	fmt.Printf("[Config] 候选路径均未找到 PalWorldSettings.ini，正在搜索 %s ...\n", palServerDir)
	var found string
	filepath.Walk(palServerDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || found != "" {
			return nil
		}
		// 限制深度：最多 8 层
		rel, _ := filepath.Rel(palServerDir, path)
		if rel != "." && strings.Count(rel, string(filepath.Separator)) >= 8 {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if info.Name() == "PalWorldSettings.ini" {
			found = filepath.Dir(path)
			fmt.Printf("[Config] 已找到配置目录: %s\n", found)
		}
		return nil
	})

	if found != "" {
		return found
	}

	// 兜底：返回默认路径（后续操作会报错提示用户）
	fallback := filepath.Join(palServerDir, "Pal", "Saved", "Config", "LinuxServer")
	fmt.Printf("[Config] 未找到 PalWorldSettings.ini，使用默认路径: %s\n", fallback)
	return fallback
}
