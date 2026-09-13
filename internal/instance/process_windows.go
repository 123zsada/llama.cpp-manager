//go:build windows

package instance

import (
	"os/exec"
	"strconv"
	"syscall"
)

const createNoWindow = 0x08000000

func sysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createNoWindow,
	}
}

// killProcessTree 在 Windows 上使用 taskkill 结束整个进程树。
func killProcessTree(pid int) {
	c := exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(pid))
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
	_ = c.Run()
}
