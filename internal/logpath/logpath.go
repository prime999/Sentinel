package logpath

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ErrRejectedPolicy is returned when a path is not an approved log location.
var ErrRejectedPolicy = fmt.Errorf("path not allowed: Sentinel can only monitor approved log locations")

var sensitiveBasenames = regexp.MustCompile(`(?i)^(id_rsa|id_ed25519|id_ecdsa|shadow|passwd|sudoers|wp-config\.php|config\.php|credentials)$`)
var sensitiveExt = regexp.MustCompile(`(?i)\.(pem|key|sqlite|db|env)$`)
var envFile = regexp.MustCompile(`(?i)^\.env(\..+)?$`)

// ValidatePath checks that path is an absolute allowed log path (no open).
// Glob characters * and ? are allowed in the path for pattern sources; each
// expanded match must be validated again with ValidateResolvedFile.
func ValidatePath(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("%w: path required", ErrRejectedPolicy)
	}
	if strings.Contains(path, "\x00") {
		return fmt.Errorf("%w: invalid path", ErrRejectedPolicy)
	}
	if !filepath.IsAbs(path) {
		return fmt.Errorf("%w: path must be absolute", ErrRejectedPolicy)
	}
	if strings.Contains(path, "..") {
		return fmt.Errorf("%w: path must not contain ..", ErrRejectedPolicy)
	}
	cleaned := filepath.Clean(path)
	if err := checkDenylist(cleaned); err != nil {
		return err
	}
	if !matchesAllowlist(cleaned) {
		return fmt.Errorf("%w", ErrRejectedPolicy)
	}
	return nil
}

// ValidateResolvedFile validates a concrete file after EvalSymlinks and Stat.
func ValidateResolvedFile(resolved string, info os.FileInfo) error {
	if info == nil {
		return fmt.Errorf("%w: missing file info", ErrRejectedPolicy)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%w: not a regular file", ErrRejectedPolicy)
	}
	resolved = filepath.Clean(resolved)
	if !filepath.IsAbs(resolved) {
		return fmt.Errorf("%w: resolved path must be absolute", ErrRejectedPolicy)
	}
	if err := checkDenylist(resolved); err != nil {
		return err
	}
	if !matchesAllowlist(resolved) {
		return fmt.Errorf("%w", ErrRejectedPolicy)
	}
	return nil
}

// ResolveAndValidate cleans, resolves symlinks when the file exists, and validates.
func ResolveAndValidate(path string) (string, error) {
	if err := ValidatePath(path); err != nil {
		return "", err
	}
	cleaned := filepath.Clean(strings.TrimSpace(path))
	if strings.ContainsAny(cleaned, "*?[") {
		// Glob pattern — validate pattern shape only; expanders re-check each file.
		return cleaned, nil
	}
	resolved, err := filepath.EvalSymlinks(cleaned)
	if err != nil {
		if os.IsNotExist(err) {
			// File may appear later; still enforce allowlist on cleaned path.
			return cleaned, nil
		}
		return "", fmt.Errorf("%w: %v", ErrRejectedPolicy, err)
	}
	info, err := os.Lstat(resolved)
	if err != nil {
		if os.IsNotExist(err) {
			return resolved, nil
		}
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("%w: refusing dangling symlink", ErrRejectedPolicy)
	}
	info, err = os.Stat(resolved)
	if err != nil {
		if os.IsNotExist(err) {
			return resolved, nil
		}
		return "", err
	}
	if err := ValidateResolvedFile(resolved, info); err != nil {
		return "", err
	}
	return resolved, nil
}

func checkDenylist(path string) error {
	lower := strings.ToLower(path)
	blockedPrefixes := []string{
		"/etc/", "/root/", "/proc/", "/sys/", "/dev/",
	}
	for _, p := range blockedPrefixes {
		if path == strings.TrimSuffix(p, "/") || strings.HasPrefix(path, p) {
			return fmt.Errorf("%w", ErrRejectedPolicy)
		}
	}
	if strings.Contains(lower, "/.ssh/") {
		return fmt.Errorf("%w", ErrRejectedPolicy)
	}
	base := filepath.Base(path)
	if sensitiveBasenames.MatchString(base) || envFile.MatchString(base) {
		return fmt.Errorf("%w", ErrRejectedPolicy)
	}
	if sensitiveExt.MatchString(base) && !strings.HasSuffix(lower, ".log") {
		return fmt.Errorf("%w", ErrRejectedPolicy)
	}
	return nil
}

func matchesAllowlist(path string) bool {
	lower := strings.ToLower(path)

	if strings.HasPrefix(path, "/var/log/") || path == "/var/log" {
		return true
	}

	appRoots := []string{"/var/www/", "/srv/"}
	for _, root := range appRoots {
		if !strings.HasPrefix(path, root) {
			continue
		}
		// Must be under a logs/log/storage/logs directory segment.
		if strings.Contains(lower, "/storage/logs/") ||
			strings.Contains(lower, "/logs/") ||
			strings.Contains(lower, "/log/") {
			return true
		}
	}
	return false
}

const MaxGlobMatches = 20

// ExpandGlob expands a glob under allowlisted dirs and validates each match.
func ExpandGlob(pattern string) ([]string, error) {
	if err := ValidatePath(pattern); err != nil {
		return nil, err
	}
	cleaned := filepath.Clean(strings.TrimSpace(pattern))
	if !strings.ContainsAny(cleaned, "*?[") {
		resolved, err := ResolveAndValidate(cleaned)
		if err != nil {
			return nil, err
		}
		return []string{resolved}, nil
	}
	matches, err := filepath.Glob(cleaned)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRejectedPolicy, err)
	}
	var out []string
	for _, m := range matches {
		resolved, err := ResolveAndValidate(m)
		if err != nil {
			continue
		}
		out = append(out, resolved)
		if len(out) >= MaxGlobMatches {
			break
		}
	}
	return out, nil
}

// OpenLogFile opens a validated log path read-only (O_RDONLY). Caller must close.
func OpenLogFile(path string) (*os.File, string, error) {
	resolved, err := ResolveAndValidate(path)
	if err != nil {
		return nil, "", err
	}
	if strings.ContainsAny(resolved, "*?[") {
		return nil, "", fmt.Errorf("%w: expand glob before open", ErrRejectedPolicy)
	}
	info, err := os.Lstat(resolved)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, resolved, fmt.Errorf("file missing: %w", err)
		}
		if os.IsPermission(err) {
			return nil, resolved, fmt.Errorf("permission denied: %w", err)
		}
		return nil, resolved, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, resolved, fmt.Errorf("%w: refusing to open symlink directly", ErrRejectedPolicy)
	}
	f, err := os.OpenFile(resolved, os.O_RDONLY, 0)
	if err != nil {
		if os.IsPermission(err) {
			return nil, resolved, fmt.Errorf("permission denied: %w", err)
		}
		return nil, resolved, err
	}
	st, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, resolved, err
	}
	if err := ValidateResolvedFile(resolved, st); err != nil {
		f.Close()
		return nil, resolved, err
	}
	return f, resolved, nil
}

// LooksLikeLogName returns true for common log filenames (advisory; denylist still wins).
func LooksLikeLogName(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	if strings.HasSuffix(base, ".log") || strings.Contains(base, ".log.") {
		return true
	}
	for _, n := range []string{"error.log", "access.log", "syslog", "messages", "daemon.log"} {
		if base == n {
			return true
		}
	}
	return strings.HasPrefix(base, "laravel-")
}
