//go:build !windows

package services

import (
	"os"
	"syscall"
)

// signalProcess 向进程发送信号。在 Unix 上，发送 SIGTERM。
func signalProcess(p *os.Process, _ os.Signal) error {
	return p.Signal(syscall.SIGTERM)
}

// killProcess 强制终止进程。在 Unix 上，发送 SIGKILL。
func killProcess(p *os.Process) error {
	return p.Signal(syscall.SIGKILL)
}
