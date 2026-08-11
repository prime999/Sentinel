package checker

import (
	"testing"

	"github.com/sentinel-monitoring/sentinel/internal/models"
)

func TestSSLExpirySeverityBands(t *testing.T) {
	cases := []struct {
		days int
		want models.MonitorStatus
	}{
		{31, models.StatusUp},
		{30, models.StatusDegraded},
		{8, models.StatusDegraded},
		{7, models.StatusDown},
		{0, models.StatusDown},
	}
	for _, tc := range cases {
		got := sslExpiryStatus(tc.days, nil)
		if got != tc.want {
			t.Fatalf("days=%d status=%s want %s", tc.days, got, tc.want)
		}
	}
	if sslExpiryStatus(40, []string{"chain_error"}) != models.StatusDegraded {
		t.Fatal("non-expiry issues should stay degraded")
	}
}
