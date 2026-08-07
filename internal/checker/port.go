package checker

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/sentinel-monitoring/sentinel/internal/models"
	"github.com/sentinel-monitoring/sentinel/internal/safehost"
)

func (c *Checker) probePort(ctx context.Context, m *models.Monitor) *models.CheckResult {
	start := time.Now()
	result := &models.CheckResult{
		MonitorID: m.ID,
		Status:    models.StatusDown,
		CheckedAt: start,
	}

	host := strings.TrimSpace(m.URL)
	port := 0
	if m.Port != nil {
		port = *m.Port
	}
	if host == "" || port == 0 {
		result.Error = "host and port are required for port monitors"
		return result
	}
	if err := safehost.ValidateHostname(host); err != nil {
		result.Error = err.Error()
		return result
	}

	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	dialCtx := ctx
	if m.TimeoutMs > 0 {
		var cancel context.CancelFunc
		dialCtx, cancel = context.WithTimeout(ctx, time.Duration(m.TimeoutMs)*time.Millisecond)
		defer cancel()
	}

	conn, err := safehost.ControlDialContext(dialCtx, "tcp", addr)
	elapsed := int(time.Since(start).Milliseconds())
	result.ResponseTimeMs = elapsed
	tcpMs := elapsed
	result.TCPMs = &tcpMs

	details := models.PortDetails{Host: host, Port: port, Open: err == nil}
	if b, e := json.Marshal(details); e == nil {
		result.Details = string(b)
	}

	if err != nil {
		result.Error = fmt.Sprintf("port %d closed or unreachable: %v", port, err)
		return result
	}
	_ = conn.Close()

	if m.SlowThresholdMs > 0 && elapsed > m.SlowThresholdMs {
		result.Status = models.StatusDegraded
	} else {
		result.Status = models.StatusUp
	}
	return result
}
