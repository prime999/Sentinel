//go:build !linux

package main

import "fmt"

func readJournalLines(unit string, n int) ([]string, error) {
	return nil, fmt.Errorf("journald not supported on this platform")
}
