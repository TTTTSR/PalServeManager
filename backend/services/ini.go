package services

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// DefaultPalWorldSettings 包含所有已知的 Palworld 设置及其默认值。
var DefaultPalWorldSettings = map[string]string{
	"Difficulty":                           "None",
	"DayTimeSpeedRate":                     "1.000000",
	"NightTimeSpeedRate":                   "1.000000",
	"ExpRate":                              "1.000000",
	"PalCaptureRate":                       "1.000000",
	"PalSpawnNumRate":                      "1.000000",
	"PalDamageRateAttack":                  "1.000000",
	"PalDamageRateDefense":                 "1.000000",
	"PlayerDamageRateAttack":               "1.000000",
	"PlayerDamageRateDefense":              "1.000000",
	"PlayerStomachDecreaceRate":            "1.000000",
	"PlayerStaminaDecreaceRate":            "1.000000",
	"PlayerAutoHPRegeneRate":               "1.000000",
	"PlayerAutoHpRegeneRateInSleep":        "1.000000",
	"PalStomachDecreaceRate":               "1.000000",
	"PalStaminaDecreaceRate":               "1.000000",
	"PalAutoHPRegeneRate":                  "1.000000",
	"PalAutoHpRegeneRateInSleep":           "1.000000",
	"BuildObjectDamageRate":                "1.000000",
	"BuildObjectDeteriorationDamageRate":   "1.000000",
	"CollectionDropRate":                   "1.000000",
	"CollectionObjectHpRate":               "1.000000",
	"CollectionObjectRespawnSpeedRate":     "1.000000",
	"EnemyDropItemRate":                    "1.000000",
	"DeathPenalty":                         "All",
	"bHardcore":                            "False",
	"bPalLost":                             "False",
	"bEnablePlayerToPlayerDamage":          "False",
	"bEnableFriendlyFire":                  "False",
	"bEnableInvaderEnemy":                  "True",
	"bIsMultiplay":                         "True",
	"bIsPvP":                               "False",
	"bFastTravel":                          "True",
	"bEnableDefenseOtherGuildPlayer":        "False",
	"bInvisibleOtherGuildBaseCampAreaFX":    "False",
	"bBuildAreaLimit":                       "True",
	"bShowPlayerList":                       "True",
	"bIsUseBackupSaveData":                  "True",
	"bEnableNonLoginPenalty":                "True",
	"CoopPlayerMaxNum":                      "4",
	"ServerPlayerMaxNum":                    "32",
	"ServerName":                            "Default Palworld Server",
	"ServerDescription":                     "",
	"AdminPassword":                         "",
	"ServerPassword":                        "",
	"PublicPort":                            "8211",
	"PublicIP":                              "",
	"RESTAPIEnabled":                        "True",
	"RESTAPIPort":                           "8212",
	"Region":                                "",
	"bUseAuth":                              "True",
	"BanListURL":                            "https://api.palworldgame.com/api/banlist.txt",
	"AllowConnectPlatform":                  "Steam",
	"GuildPlayerMaxNum":                     "20",
	"PalEggDefaultHatchingTime":             "1.000000",
	"BaseCampMaxNum":                        "128",
	"BaseCampMaxNumInGuild":                 "4",
	"BaseCampWorkerMaxNum":                  "15",
	"DropItemMaxNum":                        "3000",
	"DropItemAliveMaxHours":                 "1.000000",
	"bAutoResetGuildNoOnlinePlayers":        "False",
	"AutoResetGuildTimeNoOnlinePlayers":     "72.000000",
	"WorkSpeedRate":                         "1.000000",
	"bIsUseBackupSaveData_Custom":           "True",
}

// SettingCategory 将设置映射到 UI 分类。
var SettingCategory = map[string]string{
	"ServerName":          "Server",
	"ServerDescription":   "Server",
	"ServerPassword":      "Server",
	"AdminPassword":       "Server",
	"PublicPort":          "Server",
	"PublicIP":            "Server",
	"RESTAPIEnabled":         "Server",
	"RESTAPIPort":            "Server",
	"Region":              "Server",
	"bUseAuth":            "Server",
	"BanListURL":          "Server",
	"AllowConnectPlatform": "Server",
	"ServerPlayerMaxNum":  "Multiplayer",
	"bIsMultiplay":        "Multiplayer",
	"bIsPvP":              "Multiplayer",
	"bEnablePlayerToPlayerDamage": "Multiplayer",
	"bEnableFriendlyFire": "Multiplayer",
	"CoopPlayerMaxNum":    "Multiplayer",
	"GuildPlayerMaxNum":   "Multiplayer",
	"bShowPlayerList":     "Multiplayer",
	"DayTimeSpeedRate":    "Time",
	"NightTimeSpeedRate":  "Time",
	"ExpRate":             "Player",
	"Difficulty":          "Player",
	"DeathPenalty":        "Player",
	"bHardcore":           "Player",
	"PlayerDamageRateAttack":  "Player",
	"PlayerDamageRateDefense": "Player",
	"PlayerStomachDecreaceRate":    "Player",
	"PlayerStaminaDecreaceRate":    "Player",
	"PlayerAutoHPRegeneRate":       "Player",
	"PlayerAutoHpRegeneRateInSleep": "Player",
	"bPalLost":            "Player",
	"PalCaptureRate":      "Pal",
	"PalSpawnNumRate":     "Pal",
	"PalDamageRateAttack":  "Pal",
	"PalDamageRateDefense": "Pal",
	"PalStomachDecreaceRate":       "Pal",
	"PalStaminaDecreaceRate":       "Pal",
	"PalAutoHPRegeneRate":          "Pal",
	"PalAutoHpRegeneRateInSleep":   "Pal",
	"PalEggDefaultHatchingTime":    "Pal",
	"WorkSpeedRate":       "Pal",
	"BuildObjectDamageRate":              "Building",
	"BuildObjectDeteriorationDamageRate": "Building",
	"bBuildAreaLimit":              "Building",
	"CollectionDropRate":               "World",
	"CollectionObjectHpRate":           "World",
	"CollectionObjectRespawnSpeedRate": "World",
	"EnemyDropItemRate":       "World",
	"DropItemMaxNum":          "World",
	"DropItemAliveMaxHours":   "World",
	"bEnableInvaderEnemy":     "World",
	"BaseCampMaxNum":          "Base",
	"BaseCampMaxNumInGuild":   "Base",
	"BaseCampWorkerMaxNum":    "Base",
	"bInvisibleOtherGuildBaseCampAreaFX": "Base",
	"bAutoResetGuildNoOnlinePlayers":     "Base",
	"AutoResetGuildTimeNoOnlinePlayers":  "Base",
}

// PalWorldSettings 表示解析后的服务器配置。
type PalWorldSettings struct {
	Settings map[string]string `json:"settings"`
	FilePath string            `json:"filePath"`
}

// LoadSettings 读取并解析 PalWorldSettings.ini 文件。
func LoadSettings(configDir string) (*PalWorldSettings, error) {
	iniPath := filepath.Join(configDir, "PalWorldSettings.ini")
	settings := &PalWorldSettings{
		Settings: make(map[string]string),
		FilePath: iniPath,
	}

	// 使用默认值初始化
	for k, v := range DefaultPalWorldSettings {
		settings.Settings[k] = v
	}

	// 尝试读取实际的配置文件
	data, err := os.ReadFile(iniPath)
	if err != nil {
		if os.IsNotExist(err) {
			// 检查 DefaultPalWorldSettings.ini
			defaultPath := filepath.Join(filepath.Dir(configDir), "..", "..", "..", "DefaultPalWorldSettings.ini")
			// 更具体的路径：<PalServerDir>/DefaultPalWorldSettings.ini
			altDefaultPath := filepath.Join(configDir, "..", "..", "..", "DefaultPalWorldSettings.ini")
			data2, err2 := os.ReadFile(defaultPath)
			if err2 != nil {
				data2, err2 = os.ReadFile(altDefaultPath)
				if err2 != nil {
					return settings, nil // 返回默认值
				}
			}
			data = data2
		} else {
			return settings, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// 解析 OptionSettings 行
	parseSettings(string(data), settings.Settings)
	return settings, nil
}

// Save 将设置写回 PalWorldSettings.ini。
func (s *PalWorldSettings) Save() error {
	content := generateINI(s.Settings)
	return os.WriteFile(s.FilePath, []byte(content), 0644)
}

// parseSettings 解析 OptionSettings=(...) 行。
func parseSettings(content string, settings map[string]string) {
	// 查找 OptionSettings=(...) 块
	re := regexp.MustCompile(`OptionSettings=\(([^)]*)\)`)
	matches := re.FindStringSubmatch(content)
	if len(matches) < 2 {
		return
	}

	optionsStr := matches[1]

	// 解析 key=value 键值对
	// 处理带引号的字符串和不带引号的值
	keyRe := regexp.MustCompile(`(\w+)=("([^"]*)"|([^,]+))`)
	keyMatches := keyRe.FindAllStringSubmatch(optionsStr, -1)

	for _, m := range keyMatches {
		if len(m) < 3 {
			continue
		}
		key := m[1]
		var value string
		if m[3] != "" || (len(m) > 2 && strings.HasPrefix(m[2], "\"")) {
			// 带引号的值
			value = m[3]
		} else {
			// 不带引号的值
			value = strings.TrimSpace(m[4])
		}
		settings[key] = value
	}
}

// generateINI 生成 PalWorldSettings.ini 的内容。
func generateINI(settings map[string]string) string {
	var parts []string

	// 对键进行排序以保持输出一致性——重要设置优先
	importantKeys := []string{
		"ServerName", "ServerDescription", "ServerPassword", "AdminPassword",
		"PublicIP", "PublicPort", "RESTAPIEnabled", "RESTAPIPort",
		"ServerPlayerMaxNum", "bIsMultiplay", "bIsPvP", "bEnablePlayerToPlayerDamage",
	}

	// 先添加重要的键
	added := make(map[string]bool)
	for _, k := range importantKeys {
		if v, ok := settings[k]; ok {
			parts = append(parts, formatSetting(k, v))
			added[k] = true
		}
	}

	// 按排序添加剩余的键
	var remaining []string
	for k := range settings {
		if !added[k] {
			remaining = append(remaining, k)
		}
	}
	sort.Strings(remaining)
	for _, k := range remaining {
		parts = append(parts, formatSetting(k, settings[k]))
	}

	optionsLine := "OptionSettings=(" + strings.Join(parts, ",") + ")\n"
	return "[/Script/Pal.PalGameWorldSettings]\n" + optionsLine
}

// formatSetting 为 INI 文件格式化单个 key=value 键值对。
func formatSetting(key, value string) string {
	switch key {
	case "ServerName", "ServerDescription", "ServerPassword", "AdminPassword",
		"PublicIP", "Region", "BanListURL", "AllowConnectPlatform":
		return fmt.Sprintf(`%s="%s"`, key, value)
	default:
		// 数值和布尔值不需要引号
		if value == "True" || value == "False" || value == "None" ||
			value == "All" || value == "Item" || value == "ItemAndEquipment" {
			return fmt.Sprintf("%s=%s", key, value)
		}
		// 检查是否为数字
		if _, err := strconv.ParseFloat(value, 64); err == nil {
			return fmt.Sprintf("%s=%s", key, value)
		}
		// 默认不使用引号
		return fmt.Sprintf("%s=%s", key, value)
	}
}

// GetCategories 返回按 UI 分类分组的设置。
func (s *PalWorldSettings) GetCategories() map[string]map[string]string {
	categories := make(map[string]map[string]string)
	for key, value := range s.Settings {
		cat, ok := SettingCategory[key]
		if !ok {
			cat = "Other"
		}
		if categories[cat] == nil {
			categories[cat] = make(map[string]string)
		}
		categories[cat][key] = value
	}
	return categories
}

// UpdateSettings 一次性更新多个设置。
func (s *PalWorldSettings) UpdateSettings(updates map[string]string) {
	for k, v := range updates {
		s.Settings[k] = v
	}
}
