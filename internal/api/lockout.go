package api

import (
	"strings"
	"time"
)

const (
	loginFailLimit  = 5
	loginFailWindow = 15 * time.Minute
)

func loginFailKey(username string) string {
	return "login-fail:" + strings.ToLower(strings.TrimSpace(username))
}
