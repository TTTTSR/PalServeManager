package handlers

import (
	"encoding/json"
	"net/http"

	"palworldserve/config"
)

// PanelConfigHandler 管理面板自身配置。
type PanelConfigHandler struct {
	cfg    *config.Config
	onSave func(*config.Config) error
}

// NewPanelConfigHandler 创建一个新的 PanelConfigHandler。
func NewPanelConfigHandler(cfg *config.Config, onSave func(*config.Config) error) *PanelConfigHandler {
	return &PanelConfigHandler{cfg: cfg, onSave: onSave}
}

// GetConfig 返回当前面板配置。
func (h *PanelConfigHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.cfg)
}

// UpdateConfig 更新面板配置。
func (h *PanelConfigHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	// 逐个字段更新
	applyString(updates, "host", &h.cfg.Host)
	applyInt(updates, "port", &h.cfg.Port)
	applyString(updates, "palServerDir", &h.cfg.PalServerDir)
	applyString(updates, "palServerBinary", &h.cfg.PalServerBinary)
	applyString(updates, "steamCmdPath", &h.cfg.SteamCMDPath)
	applyString(updates, "restApiHost", &h.cfg.RestAPIHost)
	applyInt(updates, "restApiPort", &h.cfg.RestAPIPort)
	applyInt(updates, "serverPort", &h.cfg.ServerPort)
	applyInt(updates, "maxPlayers", &h.cfg.MaxPlayers)
	applyString(updates, "extraArgs", &h.cfg.ExtraArgs)
	applyString(updates, "publicIp", &h.cfg.PublicIP)
	applyBool(updates, "usePublicLobby", &h.cfg.UsePublicLobby)

	if h.onSave != nil {
		if err := h.onSave(h.cfg); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "Failed to save config: " + err.Error(),
			})
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Config updated. Some changes require restart.",
		"config":  h.cfg,
	})
}

func applyString(m map[string]interface{}, key string, target *string) {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			*target = s
		}
	}
}

func applyInt(m map[string]interface{}, key string, target *int) {
	if v, ok := m[key]; ok {
		if f, ok := v.(float64); ok {
			*target = int(f)
		}
	}
}

func applyBool(m map[string]interface{}, key string, target *bool) {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			*target = b
		}
	}
}
