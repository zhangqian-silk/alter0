//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

const (
	processQueryLimitedInformation = 0x1000
	processStillActive             = 259
)

var (
	kernel32DLL     = syscall.NewLazyDLL("kernel32.dll")
	openProcessProc = kernel32DLL.NewProc("OpenProcess")
	closeHandleProc = kernel32DLL.NewProc("CloseHandle")
	getExitCodeProc = kernel32DLL.NewProc("GetExitCodeProcess")
)

func processExists(pid int) bool {
	handle, _, _ := openProcessProc.Call(
		uintptr(processQueryLimitedInformation),
		uintptr(0),
		uintptr(uint32(pid)),
	)
	if handle == 0 {
		return false
	}
	defer closeHandleProc.Call(handle)

	var exitCode uint32
	result, _, _ := getExitCodeProc.Call(handle, uintptr(unsafe.Pointer(&exitCode)))
	if result == 0 {
		return false
	}
	return exitCode == processStillActive
}
