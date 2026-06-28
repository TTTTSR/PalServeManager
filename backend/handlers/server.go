package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"palworldserve/services"
)

// ServerHandler 处理服务器控制相关的端点。
type ServerHandler struct {
	pm          *services.ProcessManager
	restClient  *services.RestAPIClient
	steamcmd    *services.SteamCMDService
}

// NewServerHandler 创建一个新的 ServerHandler。
func NewServerHandler(pm *services.ProcessManager, rest *services.RestAPIClient, steamcmd *services.SteamCMDService) *ServerHandler {
	return &ServerHandler{pm: pm, restClient: rest, steamcmd: steamcmd}
}

// GetStatus 返回当前服务器状态。
func (h *ServerHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	info := h.pm.GetInfo()
	writeJSON(w, http.StatusOK, info)
}

// Start 启动 Palworld 服务器。
func (h *ServerHandler) Start(w http.ResponseWriter, r *http.Request) {
	log := services.Log("Handler")
	if h.pm.IsRunning() {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "Server is already running"})
		return
	}

	log.Info("收到启动服务器请求")
	go func() {
		if err := h.pm.Start(); err != nil {
			log.Error("启动服务器失败: %v", err)
		}
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{
		"message": "Server start initiated",
		"status":  string(services.StatusStarting),
	})
}

// Stop 停止 Palworld 服务器（通过 REST API 优雅关闭）。
func (h *ServerHandler) Stop(w http.ResponseWriter, r *http.Request) {
	log := services.Log("Handler")
	if !h.pm.IsRunning() {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "Server is not running"})
		return
	}

	log.Info("收到停止服务器请求")

	go func() {
		if err := h.pm.Stop(); err != nil {
			log.Error("停止服务器失败: %v", err)
		}
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{
		"message": "Server stop initiated (via REST API)",
		"status":  string(services.StatusStopping),
	})
}

// Restart 重启 Palworld 服务器。
func (h *ServerHandler) Restart(w http.ResponseWriter, r *http.Request) {
	log := services.Log("Handler")
	log.Info("收到重启服务器请求")
	if h.pm.IsRunning() && h.restClient != nil {
		h.restClient.Announce("Server is restarting. Please reconnect in a few minutes.")
		h.restClient.SaveGame()
	}

	go func() {
		if err := h.pm.Restart(); err != nil {
			log.Error("重启服务器失败: %v", err)
		}
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{
		"message": "Server restart initiated",
	})
}

// SendCommand 通过 REST API 向服务器发送命令。
func (h *ServerHandler) SendCommand(w http.ResponseWriter, r *http.Request) {
	if !h.pm.IsRunning() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error":   "服务器未运行",
			"message": "请先启动幻兽帕鲁服务器后再发送命令",
		})
		return
	}
	if h.restClient == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "REST API not configured"})
		return
	}

	var req struct {
		Action   string `json:"action"`
		Message  string `json:"message"`
		UserID   string `json:"userid"`
		WaitTime int    `json:"waittime"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	var result string
	var err error

	switch strings.ToLower(req.Action) {
	case "info":
		info, e := h.restClient.GetInfo()
		if e != nil {
			err = e
		} else {
			result = fmt.Sprintf("Version: %s, Server: %s, World: %s", info.Version, info.Servername, info.WorldGUID)
		}
	case "metrics":
		metrics, e := h.restClient.GetMetrics()
		if e != nil {
			err = e
		} else {
			result = fmt.Sprintf("FPS: %d, Players: %d, Uptime: %d, Days: %d, Bases: %d",
				metrics.CurrentFPS, metrics.PlayerCount, metrics.Uptime, metrics.Days, metrics.BaseCount)
		}
	case "players":
		players, e := h.restClient.GetPlayers()
		if e != nil {
			err = e
		} else {
			var lines []string
			for _, p := range players {
				lines = append(lines, fmt.Sprintf("%s (UID: %s, Level: %d, Ping: %.0f)", p.Name, p.UserID, p.Level, p.Ping))
			}
			result = strings.Join(lines, "\n")
			if result == "" {
				result = "No players connected"
			}
		}
	case "announce":
		if req.Message == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "message is required for announce"})
			return
		}
		err = h.restClient.Announce(req.Message)
		result = "Announcement sent"
	case "save":
		err = h.restClient.SaveGame()
		result = "World saved"
	case "shutdown":
		waittime := req.WaitTime
		if waittime <= 0 {
			waittime = 60
		}
		msg := req.Message
		if msg == "" {
			msg = "Server is shutting down"
		}
		err = h.restClient.Shutdown(waittime, msg)
		result = fmt.Sprintf("Shutdown initiated with %ds countdown", waittime)
	case "kick":
		if req.UserID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "userid is required for kick"})
			return
		}
		msg := req.Message
		if msg == "" {
			msg = "You have been kicked"
		}
		err = h.restClient.KickPlayer(req.UserID, msg)
		result = fmt.Sprintf("Player %s kicked", req.UserID)
	case "ban":
		if req.UserID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "userid is required for ban"})
			return
		}
		msg := req.Message
		if msg == "" {
			msg = "You have been banned"
		}
		err = h.restClient.BanPlayer(req.UserID, msg)
		result = fmt.Sprintf("Player %s banned", req.UserID)
	case "unban":
		if req.UserID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "userid is required for unban"})
			return
		}
		err = h.restClient.UnbanPlayer(req.UserID)
		result = fmt.Sprintf("Player %s unbanned", req.UserID)
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Unknown action. Supported: info, metrics, players, announce, save, shutdown, kick, ban, unban",
		})
		return
	}

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":  "Command failed",
			"detail": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"action": req.Action,
		"output": result,
	})
}

// GetPlayers 通过 REST API 返回已连接玩家列表。
func (h *ServerHandler) GetPlayers(w http.ResponseWriter, r *http.Request) {
	log := services.Log("Handler")
	if !h.pm.IsRunning() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error":   "服务器未运行",
			"message": "请先启动幻兽帕鲁服务器后再查看玩家列表",
		})
		return
	}
	if h.restClient == nil {
		log.Info("GetPlayers: REST API 未配置，返回 503")
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "REST API not configured"})
		return
	}

	log.Info("GetPlayers: 请求玩家列表...")
	players, err := h.restClient.GetPlayers()
	if err != nil {
		log.Error("GetPlayers: 获取玩家列表失败: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":  "Failed to get player list",
			"detail": err.Error(),
		})
		return
	}

	if players == nil {
		players = []services.Player{}
	}

	log.Info("GetPlayers: 成功，%d 名玩家在线", len(players))
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"players": players,
		"count":   len(players),
	})
}

// GetServerInfo 通过 REST API 返回服务器信息。
func (h *ServerHandler) GetServerInfo(w http.ResponseWriter, r *http.Request) {
	log := services.Log("Handler")
	if !h.pm.IsRunning() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error":   "服务器未运行",
			"message": "请先启动幻兽帕鲁服务器后再查看服务器信息",
		})
		return
	}
	if h.restClient == nil {
		log.Info("GetServerInfo: REST API 未配置，返回 503")
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "REST API not configured"})
		return
	}

	log.Info("GetServerInfo: 请求服务器信息...")
	info, err := h.restClient.GetInfo()
	if err != nil {
		log.Error("GetServerInfo: 获取服务器信息失败: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":  "Failed to get server info",
			"detail": err.Error(),
		})
		return
	}

	log.Info("GetServerInfo: 成功 (版本=%s, 名称=%s)", info.Version, info.Servername)
	writeJSON(w, http.StatusOK, info)
}

// Install 通过 SteamCMD 安装 Palworld 服务器。
func (h *ServerHandler) Install(w http.ResponseWriter, r *http.Request) {
	log := services.Log("Handler")
	if h.pm.IsInstalled() {
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": "服务器已安装，请使用更新功能",
		})
		return
	}

	if !h.steamcmd.IsSteamCMDInstalled() {
		log.Error("安装请求被拒绝: SteamCMD 未安装")
		writeJSON(w, http.StatusPreconditionFailed, map[string]string{
			"error": "未检测到 SteamCMD，请先在系统中安装 SteamCMD",
		})
		return
	}

	if h.steamcmd.IsUpdating() {
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": "安装/更新操作正在进行中",
		})
		return
	}

	h.pm.SetInstalling()
	log.Info("开始安装 Palworld 服务器...")

	go func() {
		result, err := h.steamcmd.Update(true)
		if err != nil || (result != nil && !result.Success) {
			msg := "安装失败"
			if result != nil {
				msg = result.Message
			}
			log.Error("安装失败: %s", msg)
			return
		}
		h.pm.SetInstalled()
		log.Info("安装完成 (Build: %s)", result.BuildID)
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{
		"message": "安装已开始，通过 SteamCMD 下载 Palworld 服务器（约 20GB），需要几分钟。",
	})
}
