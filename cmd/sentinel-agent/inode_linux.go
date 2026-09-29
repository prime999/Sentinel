//go:build linux

package main

import (
	"os"
	"syscall"
)

func inodeFromFileInfo(info os.FileInfo) uint64 {
	if info == nil {
		return 0
	}
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		return st.Ino
	}
	return 0
}
