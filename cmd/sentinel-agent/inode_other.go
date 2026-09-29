//go:build !linux

package main

import "os"

func inodeFromFileInfo(info os.FileInfo) uint64 {
	return 0
}
