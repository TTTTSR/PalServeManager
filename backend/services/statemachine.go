package services

import "time"

var allowedTransitions = map[ServerStatus]map[ServerStatus]bool{
	StatusNotInstalled: {StatusInstalling: true},
	StatusInstalling:   {StatusStopped: true, StatusNotInstalled: true},
	StatusStopped:      {StatusStarting: true, StatusUpdating: true},
	StatusStarting:     {StatusRunning: true, StatusStopped: true},
	StatusRunning:      {StatusStopping: true, StatusStopped: true},
	StatusStopping:     {StatusStopped: true},
	StatusUpdating:     {StatusStopped: true},
}

// StartPolling 后台轮询检测服务端真实状态，兜底状态机
func (pm *ProcessManager) StartPolling() {
	go func() {
		time.Sleep(15 * time.Second)
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			pm.pollServerStatus()
		}
	}()
}

func (pm *ProcessManager) pollServerStatus() {
	pm.mu.Lock()
	status := pm.status
	restClient := pm.restClient
	pm.mu.Unlock()

	if status == StatusStarting || status == StatusStopping ||
		status == StatusInstalling || status == StatusUpdating {
		return
	}
	if restClient == nil {
		return
	}

	apiAlive := restClient.Ping() == nil

	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.status == StatusRunning && !apiAlive {
		Log("Server").Info("轮询: running -> stopped")
		pm.setStatus(StatusStopped)
	} else if pm.status == StatusStopped && apiAlive {
		Log("Server").Info("轮询: stopped -> running")
		pm.setStatus(StatusRunning)
	}
	_ = status
}
