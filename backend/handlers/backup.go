package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"palworldserve/services"
)

// BackupHandler 管理存档备份与恢复。
type BackupHandler struct {
	pm         *services.ProcessManager
	restClient *services.RestAPIClient
	saveDir    string
	backupDir  string
}

// NewBackupHandler 创建一个新的 BackupHandler。
func NewBackupHandler(pm *services.ProcessManager, restClient *services.RestAPIClient, saveDir, backupDir string) *BackupHandler {
	return &BackupHandler{pm: pm, restClient: restClient, saveDir: saveDir, backupDir: backupDir}
}

// List 返回备份列表。
func (h *BackupHandler) List(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir(h.backupDir)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"backups": []string{}})
		return
	}
	var backups []string
	for _, e := range entries {
		if e.IsDir() {
			backups = append(backups, e.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(backups)))
	writeJSON(w, http.StatusOK, map[string]interface{}{"backups": backups})
}

// Create 创建新备份。
func (h *BackupHandler) Create(w http.ResponseWriter, r *http.Request) {
	if !h.pm.IsInstalled() {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "服务器未安装"})
		return
	}

	// 若在运行，先调用 REST API 存档
	if h.pm.IsRunning() && h.restClient != nil {
		h.restClient.SaveGame()
		time.Sleep(2 * time.Second) // 等落盘
	}

	name := time.Now().Format("2006-01-02_15-04-05")
	dest := filepath.Join(h.backupDir, name)
	if err := copyDir(h.saveDir, dest); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "备份失败: " + err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "备份完成", "name": name})
}

// Restore 恢复指定备份。
func (h *BackupHandler) Restore(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请指定备份名称"})
		return
	}

	src := filepath.Join(h.backupDir, req.Name)
	if _, err := os.Stat(src); os.IsNotExist(err) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "备份不存在"})
		return
	}

	// 记下是否在运行
	wasRunning := h.pm.IsRunning()

	// 停服
	if wasRunning {
		h.restClient.Announce("正在恢复存档...")
		if err := h.pm.Stop(); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "停止服务器失败: " + err.Error()})
			return
		}
	}

	// 清空当前存档
	os.RemoveAll(h.saveDir)
	os.MkdirAll(h.saveDir, 0755)

	// 复制备份到存档目录
	if err := copyDir(src, h.saveDir); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "恢复失败: " + err.Error()})
		return
	}

	// 之前运行则重启
	if wasRunning {
		go func() {
			time.Sleep(2 * time.Second)
			h.pm.Start()
		}()
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": fmt.Sprintf("已恢复至 %s", req.Name)})
}

// copyDir 递归复制目录内容。
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	os.MkdirAll(filepath.Dir(dst), 0755)
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
