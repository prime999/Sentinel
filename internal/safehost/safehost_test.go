package safehost

import (
	"net"
	"testing"
)

func TestIsBlockedIP(t *testing.T) {
	cases := []struct {
		ip      string
		blocked bool
	}{
		{"127.0.0.1", true},
		{"10.0.0.1", true},
		{"192.168.1.1", true},
		{"172.16.0.1", true},
		{"169.254.169.254", true},
		{"100.64.1.1", true},
		{"0.0.0.0", true},
		{"::1", true},
		{"fc00::1", true},
		{"fe80::1", true},
		{"8.8.8.8", false},
		{"1.1.1.1", false},
	}
	for _, tc := range cases {
		ip := net.ParseIP(tc.ip)
		if got := IsBlockedIP(ip); got != tc.blocked {
			t.Errorf("IsBlockedIP(%s)=%v want %v", tc.ip, got, tc.blocked)
		}
	}
}

func TestValidateHTTPURL(t *testing.T) {
	if err := ValidateHTTPURL("http://127.0.0.1/"); err == nil {
		t.Fatal("expected block for loopback")
	}
	if err := ValidateHTTPURL("http://10.0.0.5/admin"); err == nil {
		t.Fatal("expected block for private")
	}
	if err := ValidateHTTPURL("ftp://example.com/"); err == nil {
		t.Fatal("expected scheme error")
	}
}
