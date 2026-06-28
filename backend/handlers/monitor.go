package handlers

import (
	"net/http"

	"palworldserve/services"
)

// MonitorHandler 处理监控和统计相关的端点。
type MonitorHandler struct {
	monitor *services.Monitor
}

// NewMonitorHandler 创建一个新的 MonitorHandler。
func NewMonitorHandler(monitor *services.Monitor) *MonitorHandler {
	return &MonitorHandler{monitor: monitor}
}

// GetStats 返回系统与服务器的综合统计信息。
func (h *MonitorHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.monitor.GetCombinedStats()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to get stats: " + err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

// GetSystemStats 返回系统级资源使用情况。
func (h *MonitorHandler) GetSystemStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.monitor.GetSystemStats()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to get system stats: " + err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

// GetServerStats 返回服务器级资源使用情况和玩家数量。
func (h *MonitorHandler) GetServerStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.monitor.GetServerStats()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to get server stats: " + err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, stats)
}
