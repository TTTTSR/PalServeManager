//go:build windows

package services

import (
	"os"
)

// signalProcess 向进程发送信号。在 Windows 上，使用 os.Kill。
func signalProcess(p *os.Process, _ os.Signal) error {
	return p.Kill()
}

// killProcess 强制终止进程。在 Windows 上，使用 os.Kill。
func killProcess(p *os.Process) error {
	return p.Kill()
}
