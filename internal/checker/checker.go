package checker

import (
	"context"

	"github.com/sentinel-monitoring/sentinel/internal/models"
	"github.com/sentinel-monitoring/sentinel/internal/store"
)

type Checker struct {
	store *store.Store
}

func New(s *store.Store) *Checker {
	return &Checker{store: s}
}

func (c *Checker) Probe(ctx context.Context, m *models.Monitor) *models.CheckResult {
	switch m.Type {
	case models.MonitorPort:
		return c.probePort(ctx, m)
	case models.MonitorSSL:
		return c.probeSSL(ctx, m)
	case models.MonitorDNS:
		return c.probeDNS(ctx, m)
	case models.MonitorHeartbeat:
		return c.probeHeartbeat(ctx, m)
	default:
		return c.probeHTTP(ctx, m)
	}
}
