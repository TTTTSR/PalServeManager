package handlers

import (
	"encoding/json"
	"net/http"

	"palworldserve/config"
)

// ScheduleHandler 处理定时任务相关的端点。
type ScheduleHandler struct {
	cfg     *config.Config
	onSave  func(*config.Config) error
}

// NewScheduleHandler 创建一个新的 ScheduleHandler。
func NewScheduleHandler(cfg *config.Config, onSave func(*config.Config) error) *ScheduleHandler {
	return &ScheduleHandler{cfg: cfg, onSave: onSave}
}

// GetSchedule 返回当前定时任务配置。
func (h *ScheduleHandler) GetSchedule(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"autoRestart": map[string]interface{}{
			"enabled": h.cfg.AutoRestartEnabled,
			"cron":    h.cfg.AutoRestartCron,
		},
		"autoUpdate": map[string]interface{}{
			"enabled": h.cfg.AutoUpdateEnabled,
			"cron":    h.cfg.AutoUpdateCron,
		},
		"autoSave": map[string]interface{}{
			"enabled": h.cfg.AutoSaveEnabled,
			"cron":    h.cfg.AutoSaveCron,
		},
	})
}

// UpdateSchedule 更新定时任务配置。
func (h *ScheduleHandler) UpdateSchedule(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AutoRestart *struct {
			Enabled *bool   `json:"enabled"`
			Cron    *string `json:"cron"`
		} `json:"autoRestart"`
		AutoUpdate *struct {
			Enabled *bool   `json:"enabled"`
			Cron    *string `json:"cron"`
		} `json:"autoUpdate"`
		AutoSave *struct {
			Enabled *bool   `json:"enabled"`
			Cron    *string `json:"cron"`
		} `json:"autoSave"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	changed := false

	if req.AutoRestart != nil {
		if req.AutoRestart.Enabled != nil {
			h.cfg.AutoRestartEnabled = *req.AutoRestart.Enabled
			changed = true
		}
		if req.AutoRestart.Cron != nil {
			h.cfg.AutoRestartCron = *req.AutoRestart.Cron
			changed = true
		}
	}

	if req.AutoUpdate != nil {
		if req.AutoUpdate.Enabled != nil {
			h.cfg.AutoUpdateEnabled = *req.AutoUpdate.Enabled
			changed = true
		}
		if req.AutoUpdate.Cron != nil {
			h.cfg.AutoUpdateCron = *req.AutoUpdate.Cron
			changed = true
		}
	}

	if req.AutoSave != nil {
		if req.AutoSave.Enabled != nil {
			h.cfg.AutoSaveEnabled = *req.AutoSave.Enabled
			changed = true
		}
		if req.AutoSave.Cron != nil {
			h.cfg.AutoSaveCron = *req.AutoSave.Cron
			changed = true
		}
	}

	if !changed {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "No schedule changes provided"})
		return
	}

	if h.onSave != nil {
		if err := h.onSave(h.cfg); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "Failed to save schedule: " + err.Error(),
			})
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":  "Schedule updated successfully",
		"schedule": h.cfg,
	})
}
