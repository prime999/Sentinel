package logpath

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidatePathAllowlist(t *testing.T) {
	ok := []string{
		"/var/log/nginx/error.log",
		"/var/log/php8.2-fpm.log",
		"/var/www/app/storage/logs/laravel.log",
		"/srv/myapp/logs/app.log",
		"/var/www/*/storage/logs/*.log",
	}
	for _, p := range ok {
		if err := ValidatePath(p); err != nil {
			t.Errorf("ValidatePath(%q): %v", p, err)
		}
	}
}

func TestValidatePathDenylist(t *testing.T) {
	bad := []string{
		"/etc/shadow",
		"/etc/nginx/nginx.conf",
		"/root/.bashrc",
		"/home/user/.ssh/id_rsa",
		"/var/www/app/.env",
		"/var/log/../etc/shadow",
		"relative/path.log",
		"/var/www/app/config.php",
		"/tmp/secret.pem",
	}
	for _, p := range bad {
		if err := ValidatePath(p); err == nil {
			t.Errorf("ValidatePath(%q) should reject", p)
		}
	}
}

func TestValidateResolvedFileRegularOnly(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "x.log")
	// Put under allowlisted-looking path by validating denylist on temp — use /var/log style via symlink? 
	// Direct: create file and check non-regular rejection with a fake path under /var/log that we can't create.
	// Instead test directory rejection with ValidateResolvedFile on temp dir using a path that fails allowlist.
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateResolvedFile("/var/log/nginx", info); err == nil {
		t.Fatal("expected reject for directory")
	}
	_ = f
}
