package handlers

import (
	"net/http"
	"time"

	"palworldserve/services"
)

// LogHandler 处理服务器日志相关的端点。
type LogHandler struct {
	monitor *services.Monitor
}

// NewLogHandler 创建一个新的 LogHandler。
func NewLogHandler(monitor *services.Monitor) *LogHandler {
	return &LogHandler{monitor: monitor}
}

// GetLogs 返回最近的服务器日志条目。
func (h *LogHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	lines := parseIntParam(r, "lines", 100)
	if lines < 1 {
		lines = 1
	}
	if lines > 1000 {
		lines = 1000
	}

	logs, err := h.monitor.GetLogs(lines)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"logs":  []string{},
			"error": err.Error(),
			"time":  time.Now(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"logs":  logs,
		"count": len(logs),
		"time":  time.Now(),
	})
}
