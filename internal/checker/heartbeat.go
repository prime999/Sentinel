package checker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sentinel-monitoring/sentinel/internal/models"
)

func heartbeatGraceSeconds(m *models.Monitor) int {
	if m.Config != "" {
		var cfg models.HeartbeatConfig
		if err := json.Unmarshal([]byte(m.Config), &cfg); err == nil && cfg.GraceSeconds > 0 {
			return cfg.GraceSeconds
		}
	}
	if m.IntervalSeconds > 0 {
		return m.IntervalSeconds
	}
	return 60
}

func (c *Checker) probeHeartbeat(_ context.Context, m *models.Monitor) *models.CheckResult {
	now := time.Now().UTC()
	grace := time.Duration(heartbeatGraceSeconds(m)) * time.Second

	result := &models.CheckResult{
		MonitorID:      m.ID,
		Status:         models.StatusUp,
		ResponseTimeMs: 0,
		CheckedAt:      now,
	}

	if m.LastCheckedAt == nil {
		result.Status = models.StatusDown
		result.Error = "no heartbeat received yet"
		return result
	}

	elapsed := now.Sub(*m.LastCheckedAt)
	if elapsed > grace {
		result.Status = models.StatusDown
		result.Error = fmt.Sprintf("heartbeat overdue by %s (grace %s)", elapsed.Round(time.Second), grace)
	}

	return result
}
