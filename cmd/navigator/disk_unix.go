//go:build linux || darwin || freebsd
// +build linux darwin freebsd

package main

import "syscall"

// getDiskSpace returns available disk space in bytes for a given path (Unix/Linux/macOS)
func getDiskSpace(path string) (uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, err
	}
	// Available bytes = available blocks * block size
	return stat.Bavail * uint64(stat.Bsize), nil
}
