package handlers

import (
	"encoding/json"
	"net/http"

	"palworldserve/services"
)

// ConfigHandler 处理配置管理相关的端点。
type ConfigHandler struct {
	settings   *services.PalWorldSettings
	configDir  string
}

// NewConfigHandler 创建一个新的 ConfigHandler。
func NewConfigHandler(configDir string) *ConfigHandler {
	settings, err := services.LoadSettings(configDir)
	if err != nil {
		settings = &services.PalWorldSettings{
			Settings: services.DefaultPalWorldSettings,
		}
	}
	return &ConfigHandler{
		settings:  settings,
		configDir: configDir,
	}
}

// GetConfig 返回所有服务器设置。
func (h *ConfigHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"settings":   h.settings.Settings,
		"categories": h.settings.GetCategories(),
		"filePath":   h.settings.FilePath,
	})
}

// UpdateConfig 更新一个或多个服务器设置。
func (h *ConfigHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	var updates map[string]string
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	if len(updates) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "No settings provided"})
		return
	}

	// 验证设置
	for key := range updates {
		if _, ok := services.DefaultPalWorldSettings[key]; !ok {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "Unknown setting: " + key,
			})
			return
		}
	}

	h.settings.UpdateSettings(updates)

	// 保存到文件
	if err := h.settings.Save(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to save settings: " + err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":  "Settings updated successfully",
		"settings": h.settings.Settings,
	})
}

// ResetConfig 将设置重置为默认值。
func (h *ConfigHandler) ResetConfig(w http.ResponseWriter, r *http.Request) {
	h.settings.Settings = make(map[string]string)
	for k, v := range services.DefaultPalWorldSettings {
		h.settings.Settings[k] = v
	}

	if err := h.settings.Save(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to reset settings: " + err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Settings reset to defaults",
	})
}

// GetDefaultSettings 返回默认设置模板。
func (h *ConfigHandler) GetDefaultSettings(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"settings":   services.DefaultPalWorldSettings,
		"categories": getDefaultCategories(),
	})
}

func getDefaultCategories() map[string]map[string]string {
	categories := make(map[string]map[string]string)
	for key, value := range services.DefaultPalWorldSettings {
		cat, ok := services.SettingCategory[key]
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
