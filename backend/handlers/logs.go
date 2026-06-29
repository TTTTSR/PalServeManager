package handlers

import (
	"bufio"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"palworldserve/services"
)

// LogHandler 处理服务器日志相关的端点。
type LogHandler struct {
	monitor  *services.Monitor
	panelLog string // 管理面板自身日志目录
}

// NewLogHandler 创建一个新的 LogHandler。
func NewLogHandler(monitor *services.Monitor) *LogHandler {
	return &LogHandler{monitor: monitor, panelLog: services.LogDir()}
}

// GetLogs 返回最近的日志条目。source=panel 返回面板日志，否则返回服务器日志。
func (h *LogHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	lines := parseIntParam(r, "lines", 100)
	if lines < 1 { lines = 1 }
	if lines > 1000 { lines = 1000 }

	source := r.URL.Query().Get("source")
	var logs []string
	var err error

	if source == "panel" {
		logs, err = h.getPanelLogs(lines)
	} else {
		logs, err = h.monitor.GetLogs(lines)
	}

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

func (h *LogHandler) getPanelLogs(lines int) ([]string, error) {
	// 查找最新日志文件
	entries, err := os.ReadDir(h.panelLog)
	if err != nil {
		return nil, err
	}
	var latest string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".log" {
			latest = filepath.Join(h.panelLog, e.Name())
		}
	}
	if latest == "" {
		return []string{}, nil
	}
	f, err := os.Open(latest)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var result []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	for scanner.Scan() {
		result = append(result, scanner.Text())
	}
	if len(result) > lines {
		result = result[len(result)-lines:]
	}
	return result, scanner.Err()
}
