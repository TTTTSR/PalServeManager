package services

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
)

// SystemStats 保存系统资源使用信息。
type SystemStats struct {
	CPUUsage    float64 `json:"cpuUsage"`
	MemoryTotal uint64  `json:"memoryTotal"`
	MemoryUsed  uint64  `json:"memoryUsed"`
	MemoryUsage float64 `json:"memoryUsage"`
	DiskTotal   uint64  `json:"diskTotal"`
	DiskUsed    uint64  `json:"diskUsed"`
	DiskUsage   float64 `json:"diskUsage"`
	Uptime      string  `json:"uptime"`
}

// ServerStats 保存 Palworld 服务器资源使用情况。
type ServerStats struct {
	PID         int     `json:"pid"`
	CPUUsage    float64 `json:"cpuUsage"`
	MemoryUsed  uint64  `json:"memoryUsed"`
	MemoryUsage float64 `json:"memoryUsage"`
	PlayerCount int     `json:"playerCount"`
}

// CombinedStats 组合系统和服务器统计数据。
type CombinedStats struct {
	System    SystemStats `json:"system"`
	Server    ServerStats `json:"server"`
	Timestamp time.Time   `json:"timestamp"`
}

// Monitor 提供系统和服务器监控功能。
type Monitor struct {
	processManager *ProcessManager
	restClient     *RestAPIClient
	serverDir      string
}

// NewMonitor 创建一个新的 Monitor。
func NewMonitor(pm *ProcessManager, rest *RestAPIClient, serverDir string) *Monitor {
	return &Monitor{
		processManager: pm,
		restClient:     rest,
		serverDir:      serverDir,
	}
}

// GetCombinedStats 返回系统和服务器统计数据。
func (m *Monitor) GetCombinedStats() (*CombinedStats, error) {
	sysStats, err := m.GetSystemStats()
	if err != nil {
		sysStats = &SystemStats{}
	}

	serverStats, err := m.GetServerStats()
	if err != nil {
		serverStats = &ServerStats{}
	}

	return &CombinedStats{
		System:    *sysStats,
		Server:    *serverStats,
		Timestamp: time.Now(),
	}, nil
}

// GetSystemStats 返回系统资源使用情况。
func (m *Monitor) GetSystemStats() (*SystemStats, error) {
	stats := &SystemStats{}

	// CPU 使用率
	cpuPercent, err := cpu.Percent(time.Second, false)
	if err == nil && len(cpuPercent) > 0 {
		stats.CPUUsage = roundFloat(cpuPercent[0], 1)
	}

	// 内存
	memInfo, err := mem.VirtualMemory()
	if err == nil {
		stats.MemoryTotal = memInfo.Total
		stats.MemoryUsed = memInfo.Used
		stats.MemoryUsage = roundFloat(memInfo.UsedPercent, 1)
	}

	// 服务器目录的磁盘使用情况
	diskInfo, err := disk.Usage(m.serverDir)
	if err == nil {
		stats.DiskTotal = diskInfo.Total
		stats.DiskUsed = diskInfo.Used
		stats.DiskUsage = roundFloat(diskInfo.UsedPercent, 1)
	}

	// 系统运行时间
	stats.Uptime = getSystemUptime()

	return stats, nil
}

// GetServerStats 返回 Palworld 服务器的资源使用情况和玩家数量。
func (m *Monitor) GetServerStats() (*ServerStats, error) {
	info := m.processManager.GetInfo()
	stats := &ServerStats{
		PID: info.PID,
	}

	if info.Status != StatusRunning || info.PID == 0 {
		return stats, nil
	}

	// 获取进程级别的统计数据
	proc, err := process.NewProcess(int32(info.PID))
	if err != nil {
		return stats, nil
	}

	// CPU 使用率
	cpuPercent, err := proc.CPUPercent()
	if err == nil {
		stats.CPUUsage = roundFloat(cpuPercent, 1)
	}

	// 内存使用情况
	memInfo, err := proc.MemoryInfo()
	if err == nil {
		stats.MemoryUsed = memInfo.RSS
		// 获取系统总内存以计算百分比
		sysMem, err := mem.VirtualMemory()
		if err == nil && sysMem.Total > 0 {
			stats.MemoryUsage = roundFloat(float64(memInfo.RSS)/float64(sysMem.Total)*100, 1)
		}
	}

	// 通过 REST API 获取玩家数量（仅服务器运行时）
	if info.Status == StatusRunning && m.restClient != nil {
		count, err := m.restClient.GetPlayerCount()
		if err == nil {
			stats.PlayerCount = count
		}
	}

	return stats, nil
}

// GetLogs 获取最近的服务器日志条目。
func (m *Monitor) GetLogs(lines int) ([]string, error) {
	return m.processManager.GetLogs(lines)
}

func roundFloat(val float64, precision int) float64 {
	format := fmt.Sprintf("%%.%df", precision)
	str := fmt.Sprintf(format, val)
	var result float64
	fmt.Sscanf(str, "%f", &result)
	return result
}

func getSystemUptime() string {
	// 在 Linux 上读取 /proc/uptime
	if runtime.GOOS == "linux" {
		data, err := os.ReadFile("/proc/uptime")
		if err == nil {
			parts := strings.Fields(string(data))
			if len(parts) > 0 {
				var seconds float64
				fmt.Sscanf(parts[0], "%f", &seconds)
				d := time.Duration(seconds) * time.Second
				return formatDuration(d)
			}
		}
	}
	return "N/A"
}
