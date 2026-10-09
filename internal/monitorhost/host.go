package monitorhost

import (
	"net/url"
	"strings"
)

// NormalizeHost extracts a hostname from a monitor URL or bare host (matches MonitorForm logic).
func NormalizeHost(raw string) string {
	trimmed := strings.TrimSpace(strings.ToLower(raw))
	if trimmed == "" {
		return ""
	}
	withScheme := trimmed
	if !strings.Contains(withScheme, "://") {
		withScheme = "https://" + withScheme
	}
	u, err := url.Parse(withScheme)
	if err != nil || u.Hostname() == "" {
		s := strings.TrimPrefix(strings.TrimPrefix(trimmed, "https://"), "http://")
		s = strings.Split(s, "/")[0]
		s = strings.Split(s, ":")[0]
		return strings.TrimSuffix(s, ".")
	}
	return strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
}
