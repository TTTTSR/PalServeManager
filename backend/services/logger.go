package services

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Logger 提供同时输出到终端和文件的日志功能。
type Logger struct {
	mu       sync.Mutex
	file     *os.File
	stdout   *log.Logger
	combined *log.Logger
	category string
}

var (
	globalLogger  *Logger
	globalLogDir  string
	logOnce       sync.Once
)

// InitLogger 初始化全局日志器。
// logDir 是日志文件存放目录，传空则默认为可执行文件同级的 logs/ 目录。
func InitLogger(logDir string) {
	logOnce.Do(func() {
		if logDir == "" {
			exePath, _ := os.Executable()
			logDir = filepath.Join(filepath.Dir(exePath), "logs")
		}

		globalLogDir = logDir
		os.MkdirAll(logDir, 0755)

		logName := fmt.Sprintf("manager-%s.log", time.Now().Format("2006-01-02"))
		logPath := filepath.Join(logDir, logName)

		file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			// 无法打开文件则只输出到终端
			globalLogger = &Logger{
				stdout:   log.New(os.Stdout, "", 0),
				combined: log.New(os.Stdout, "", 0),
			}
			log.Printf("[Logger] 警告: 无法创建日志文件 %s: %v", logPath, err)
			return
		}

		multi := io.MultiWriter(os.Stdout, file)
		globalLogger = &Logger{
			file:     file,
			stdout:   log.New(os.Stdout, "", 0),
			combined: log.New(multi, "", 0),
		}

		globalLogger.combined.Printf("%s [System] === Palworld Manager 启动 ===", time.Now().Format("2006-01-02 15:04:05"))
		globalLogger.combined.Printf("%s [System] 日志文件: %s", time.Now().Format("2006-01-02 15:04:05"), logPath)
	})
}

// Log 返回带指定分类标签的日志器。
func Log(category string) *Logger {
	if globalLogger == nil {
		// 未初始化时的回退：仅输出到终端
		stdout := log.New(os.Stdout, "", 0)
		return &Logger{stdout: stdout, combined: stdout, category: category}
	}
	return &Logger{
		stdout:   globalLogger.stdout,
		combined: globalLogger.combined,
		file:     globalLogger.file,
		category: category,
	}
}

// Info 记录一条普通信息。
func (l *Logger) Info(format string, v ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	ts := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, v...)
	l.combined.Printf("%s [%s] %s", ts, l.category, msg)
}

// Error 记录一条错误信息。
func (l *Logger) Error(format string, v ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	ts := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, v...)
	l.combined.Printf("%s [%s] ERROR: %s", ts, l.category, msg)
}

// Output 记录原始命令输出，超过 8000 字符则截断。
func (l *Logger) Output(label, raw string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	ts := time.Now().Format("2006-01-02 15:04:05")
	if raw == "" {
		l.combined.Printf("%s [%s] %s (无输出)", ts, l.category, label)
		return
	}
	const maxLen = 8000
	if len(raw) > maxLen {
		l.combined.Printf("%s [%s] %s (共 %d 字符，已截断):\n%s\n%s [%s] ... (剩余 %d 字符未显示)",
			ts, l.category, label, len(raw), raw[:maxLen], ts, l.category, len(raw)-maxLen)
	} else {
		l.combined.Printf("%s [%s] %s:\n%s", ts, l.category, label, raw)
	}
}

// Request 记录一条 HTTP 请求日志。
func (l *Logger) Request(method, path string, status int, duration time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	ts := time.Now().Format("2006-01-02 15:04:05")
	statusText := fmt.Sprintf("%d", status)
	if status >= 400 {
		statusText = fmt.Sprintf("%d ✗", status)
	}
	l.combined.Printf("%s [%s] %s %s → %s (%v)", ts, l.category, method, path, statusText, duration.Round(time.Millisecond))
}

// LogDir 返回日志文件存放目录。
func LogDir() string {
	if globalLogDir != "" {
		return globalLogDir
	}
	exePath, _ := os.Executable()
	return filepath.Join(filepath.Dir(exePath), "logs")
}

// Close 关闭日志文件。
func (l *Logger) Close() {
	if l.file != nil {
		l.file.Close()
	}
}

// CloseGlobal 关闭全局日志器文件。
func CloseGlobal() {
	if globalLogger != nil && globalLogger.file != nil {
		globalLogger.file.Close()
	}
}
