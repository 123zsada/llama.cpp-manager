//go:build !windows

package main

import "fmt"

// showFatalError 在非 Windows 平台输出到标准错误。
func showFatalError(title, message string) {
	fmt.Printf("%s: %s\n", title, message)
}
