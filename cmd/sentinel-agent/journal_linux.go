//go:build linux

package main

import (
	"bufio"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

var (
	journalMu   sync.Mutex
	journalSeen = map[string]map[string]bool{}
)

func readJournalLines(unit string, n int) ([]string, error) {
	if n < 1 {
		n = 20
	}
	if n > 100 {
		n = 100
	}
	cmd := exec.Command("journalctl", "-u", unit, "-n", strconv.Itoa(n), "--no-pager", "-o", "short-iso")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	journalMu.Lock()
	prev := journalSeen[unit]
	if prev == nil {
		prev = map[string]bool{}
	}
	journalMu.Unlock()

	var lines []string
	seen := map[string]bool{}
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || prev[line] {
			continue
		}
		seen[line] = true
		lines = append(lines, line)
	}
	journalMu.Lock()
	merged := map[string]bool{}
	for k := range prev {
		merged[k] = true
	}
	for k := range seen {
		merged[k] = true
	}
	if len(merged) > 500 {
		merged = seen
	}
	journalSeen[unit] = merged
	journalMu.Unlock()
	return lines, nil
}
