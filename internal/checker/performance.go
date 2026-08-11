package checker

import (
	"context"

	"github.com/sentinel-monitoring/sentinel/internal/models"
)

func (c *Checker) ProbePerformance(ctx context.Context, t *models.PerformanceTarget) *models.PerformanceResult {
	m := &models.Monitor{
		ID:              t.ID,
		Type:            models.MonitorHTTP,
		Name:            t.Name,
		URL:             t.URL,
		Method:          t.Method,
		ExpectedStatus:  200,
		IntervalSeconds: t.IntervalSeconds,
		TimeoutMs:       t.TimeoutMs,
		SlowThresholdMs: t.SlowThresholdMs,
		FollowRedirects: t.FollowRedirects,
		Enabled:         true,
	}
	if m.Method == "" {
		m.Method = "GET"
	}

	cr := c.probeHTTP(ctx, m)
	status := cr.Status
	if status == models.StatusDown {
		status = models.StatusDegraded
	} else if status == models.StatusUp && t.SlowThresholdMs > 0 && cr.ResponseTimeMs > t.SlowThresholdMs {
		status = models.StatusDegraded
	}
	return &models.PerformanceResult{
		TargetID:       t.ID,
		Status:         status,
		StatusCode:     cr.StatusCode,
		ResponseTimeMs: cr.ResponseTimeMs,
		DNSMs:          cr.DNSMs,
		TCPMs:          cr.TCPMs,
		TLSMs:          cr.TLSMs,
		TTFBMs:         cr.TTFBMs,
		Error:          cr.Error,
		CheckedAt:      cr.CheckedAt,
	}
}
