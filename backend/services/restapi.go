package services

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// RestAPIClient 提供与 Palworld REST API 交互的方法。
// REST API 替代已弃用的 RCON 接口，默认运行在 8212 端口。
// 官方文档：https://docs.palworldgame.com/category/rest-api/
type RestAPIClient struct {
	baseURL  string
	username string
	password string
	client   *http.Client
}

// ServerInfo 表示 GET /v1/api/info 的响应。
type ServerInfo struct {
	Version   string `json:"version"`
	Servername string `json:"servername"`
	ServerName string `json:"serverName"`
	WorldGUID string `json:"worldguid"`
}

// ServerMetrics 表示 GET /v1/api/metrics 的响应。
type ServerMetrics struct {
	ServerFPS   int     `json:"serverfps"`
	CurrentFPS  int     `json:"currentfps"`
	PlayerCount int     `json:"playercount"`
	Uptime      int64   `json:"uptime"`
	FrameTime   float64 `json:"frametime"`
	Days        int     `json:"days"`
	MaxFPS      int     `json:"maxfps"`
	BaseCount   int     `json:"basecount"`
}

// Player 表示 GET /v1/api/players 返回的已连接玩家。
type Player struct {
	Name     string  `json:"name"`
	UserID   string  `json:"userid"`
	SteamID  string  `json:"steamid"`
	PlayerID string  `json:"playerid"`
	IP       string  `json:"ip"`
	Ping     float64 `json:"ping"`
	Level    int     `json:"level"`
	Location string  `json:"location"`
}

// RestResponse 封装常用的 API 响应字段。
type RestResponse struct {
	StatusCode int
	Body       []byte
}

// NewRestAPIClient 创建一个新的 REST API 客户端。
// host：Palworld 服务器主机地址（例如 "127.0.0.1"）
// port：REST API 端口（默认 8212）
// adminPassword：PalWorldSettings.ini 中的 AdminPassword
func NewRestAPIClient(host string, port int, adminPassword string) *RestAPIClient {
	return &RestAPIClient{
		baseURL:  fmt.Sprintf("http://%s:%d", host, port),
		username: "admin",
		password: adminPassword,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SetPassword 更新管理员密码（用于配置变更时）。
func (c *RestAPIClient) SetPassword(password string) {
	c.password = password
}

// basicAuth 返回 Basic Auth 请求头的值。
func (c *RestAPIClient) basicAuth() string {
	auth := c.username + ":" + c.password
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}

// doGet 执行一个带认证的 GET 请求，并解码 JSON 响应。
func (c *RestAPIClient) doGet(path string, result interface{}) error {
	log := Log("RestAPI")
	req, err := http.NewRequest("GET", c.baseURL+path, nil)
	if err != nil {
		log.Error("创建请求失败 (%s): %v", path, err)
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", c.basicAuth())

	resp, err := c.client.Do(req)
	if err != nil {
		log.Error("请求失败 (%s): %v", path, err)
		return fmt.Errorf("REST API request failed: %w", err)
	}
	defer resp.Body.Close()

	// 先读取完整响应体
	rawBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		log.Error("读取响应体失败 (%s): %v", path, readErr)
		return fmt.Errorf("failed to read response body: %w", readErr)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		log.Error("REST API 认证失败 (401)，请检查 AdminPassword")
		return fmt.Errorf("REST API authentication failed (status 401) - check AdminPassword")
	}
	if resp.StatusCode != http.StatusOK {
		log.Error("REST API 返回错误 (%s): HTTP %d\n响应体: %s", path, resp.StatusCode, string(rawBody))
		return fmt.Errorf("REST API returned status %d: %s", resp.StatusCode, string(rawBody))
	}

	// 记录成功响应（限制长度）
	if len(rawBody) > 500 {
		log.Info("%s → 200 (响应 %d 字节)", path, len(rawBody))
	} else {
		log.Info("%s → 200: %s", path, string(rawBody))
	}

	if result != nil {
		if err := json.Unmarshal(rawBody, result); err != nil {
			log.Error("JSON 解码失败 (%s): %v\n原始响应: %s", path, err, string(rawBody))
			return fmt.Errorf("failed to decode REST API response: %w\nRaw body: %s", err, string(rawBody))
		}
	}
	return nil
}

// doPost 执行一个带认证的 POST 请求，并附带 JSON 请求体。
func (c *RestAPIClient) doPost(path string, body interface{}, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequest("POST", c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", c.basicAuth())
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("REST API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("REST API authentication failed (status 401) - check AdminPassword")
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("REST API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil && resp.Body != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil && err != io.EOF {
			return fmt.Errorf("failed to decode REST API response: %w", err)
		}
	}
	return nil
}

// Ping 检查 REST API 是否可达。
func (c *RestAPIClient) Ping() error {
	return c.doGet("/v1/api/info", nil)
}

// GetInfo 返回服务器信息（版本、名称、世界 GUID）。
func (c *RestAPIClient) GetInfo() (*ServerInfo, error) {
	var info ServerInfo
	if err := c.doGet("/v1/api/info", &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// GetMetrics 返回服务器指标（FPS、玩家数量、运行时间等）。
func (c *RestAPIClient) GetMetrics() (*ServerMetrics, error) {
	var metrics ServerMetrics
	if err := c.doGet("/v1/api/metrics", &metrics); err != nil {
		return nil, err
	}
	return &metrics, nil
}

// GetPlayers 返回已连接玩家的列表。
// 注意：Palworld REST API 返回 {"players": [...]} 而非直接数组。
func (c *RestAPIClient) GetPlayers() ([]Player, error) {
	var wrapper struct {
		Players []Player `json:"players"`
	}
	if err := c.doGet("/v1/api/players", &wrapper); err != nil {
		return nil, err
	}
	return wrapper.Players, nil
}

// GetPlayerCount 返回当前已连接玩家的数量。
func (c *RestAPIClient) GetPlayerCount() (int, error) {
	metrics, err := c.GetMetrics()
	if err != nil {
		// 备用方案：从玩家列表统计数量
		players, err2 := c.GetPlayers()
		if err2 != nil {
			return 0, fmt.Errorf("failed to get player count (metrics: %v, players: %v)", err, err2)
		}
		return len(players), nil
	}
	return metrics.PlayerCount, nil
}

// GetSettings 从 REST API 返回当前服务器设置。
func (c *RestAPIClient) GetSettings() (map[string]interface{}, error) {
	var settings map[string]interface{}
	if err := c.doGet("/v1/api/settings", &settings); err != nil {
		return nil, err
	}
	return settings, nil
}

// Announce 向所有已连接玩家发送广播消息。
func (c *RestAPIClient) Announce(message string) error {
	body := map[string]string{"message": message}
	return c.doPost("/v1/api/announce", body, nil)
}

// SaveGame 强制执行一次即时世界存档。
func (c *RestAPIClient) SaveGame() error {
	return c.doPost("/v1/api/save", nil, nil)
}

// Shutdown 通过倒计时优雅地关闭服务器。
func (c *RestAPIClient) Shutdown(waittime int, message string) error {
	body := map[string]interface{}{
		"waittime": waittime,
		"message":  message,
	}
	return c.doPost("/v1/api/shutdown", body, nil)
}

// ForceStop 立即停止服务器（无倒计时）。
func (c *RestAPIClient) ForceStop() error {
	return c.doPost("/v1/api/stop", nil, nil)
}

// KickPlayer 根据用户 ID 踢出玩家。
func (c *RestAPIClient) KickPlayer(userID, message string) error {
	body := map[string]string{
		"userid":  userID,
		"message": message,
	}
	return c.doPost("/v1/api/kick", body, nil)
}

// BanPlayer 根据用户 ID 封禁玩家。
func (c *RestAPIClient) BanPlayer(userID, message string) error {
	body := map[string]string{
		"userid":  userID,
		"message": message,
	}
	return c.doPost("/v1/api/ban", body, nil)
}

// UnbanPlayer 根据用户 ID 解封玩家。
func (c *RestAPIClient) UnbanPlayer(userID string) error {
	body := map[string]string{"userid": userID}
	return c.doPost("/v1/api/unban", body, nil)
}
