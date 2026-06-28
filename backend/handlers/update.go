package handlers

import (
	"net/http"

	"palworldserve/services"
)

// UpdateHandler 处理服务器更新相关的端点。
type UpdateHandler struct {
	steamcmd *services.SteamCMDService
	pm       *services.ProcessManager
}

// NewUpdateHandler 创建一个新的 UpdateHandler。
func NewUpdateHandler(steamcmd *services.SteamCMDService, pm *services.ProcessManager) *UpdateHandler {
	return &UpdateHandler{steamcmd: steamcmd, pm: pm}
}

// CheckUpdate 检查是否有可用更新。
func (h *UpdateHandler) CheckUpdate(w http.ResponseWriter, r *http.Request) {
	// 从 REST API 获取当前运行版本
	h.steamcmd.RefreshInfo(h.pm.GetRestClient())
	current := h.steamcmd.GetInfo()
	latest, err := h.steamcmd.CheckForUpdate()
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"current": current,
			"error":   err.Error(),
		})
		return
	}

	updateAvailable := false
	if current.BuildID != "" && latest.BuildID != "" && current.BuildID != latest.BuildID {
		updateAvailable = true
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"current":         current,
		"latest":          latest,
		"updateAvailable": updateAvailable,
	})
}

// Update 通过 SteamCMD 运行服务器更新。
func (h *UpdateHandler) Update(w http.ResponseWriter, r *http.Request) {
	if h.steamcmd.IsUpdating() {
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": "An update is already in progress",
		})
		return
	}

	validate := r.URL.Query().Get("validate") == "true"

	// 如果服务器正在运行，警告需要先停止
	if h.pm.IsRunning() {
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": "Server must be stopped before updating. Stop the server first.",
		})
		return
	}

	// 在后台运行更新
	go func() {
		result, err := h.steamcmd.Update(validate)
		if err != nil {
			println("Update error:", err.Error())
		} else {
			println("Update completed:", result.Message)
			if result.BuildID != "" {
				println("New build ID:", result.BuildID)
			}
		}
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{
		"message": "Update initiated. This may take several minutes.",
	})
}

// GetUpdateStatus 返回当前更新信息。
func (h *UpdateHandler) GetUpdateStatus(w http.ResponseWriter, r *http.Request) {
	info := h.steamcmd.GetInfo()
	isUpdating := h.steamcmd.IsUpdating()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"currentBuild": info,
		"isUpdating":   isUpdating,
	})
}
