//go:build windows
// +build windows

package main

import (
	"syscall"
	"unsafe"
)

var (
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	getDiskFreeSpaceExW  = kernel32.NewProc("GetDiskFreeSpaceExW")
)

// getDiskSpace returns available disk space in bytes for a given path (Windows)
func getDiskSpace(path string) (uint64, error) {
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}

	var freeBytesAvailable uint64
	var totalBytes uint64
	var totalFreeBytes uint64

	ret, _, err := getDiskFreeSpaceExW.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&totalFreeBytes)),
	)

	if ret == 0 {
		return 0, err
	}

	return freeBytesAvailable, nil
}
