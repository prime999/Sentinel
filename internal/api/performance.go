package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/sentinel-monitoring/sentinel/internal/models"
)

func periodSince(period string) time.Time {
	switch period {
	case "7d":
		return time.Now().AddDate(0, 0, -7)
	case "30d":
		return time.Now().AddDate(0, 0, -30)
	default:
		return time.Now().Add(-24 * time.Hour)
	}
}

func (s *Server) handleGetFleetPerformance(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	period := r.URL.Query().Get("period")
	since := periodSince(period)

	var fleet *models.FleetPerformance
	var err error
	if isPlatformAdmin(user) {
		customerFilter := strings.TrimSpace(r.URL.Query().Get("customer"))
		if customerFilter != "" {
			fleet, err = s.store.GetFleetPerformanceByTenant(since, customerFilter)
		} else {
			fleet, err = s.store.GetFleetPerformance(since)
		}
	} else if user.TenantID != "" {
		fleet, err = s.store.GetFleetPerformanceByTenant(since, user.TenantID)
	} else {
		fleet = &models.FleetPerformance{Monitors: []models.MonitorPerformance{}, Timeline: []models.FleetTimelinePoint{}}
	}
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	fleet.Period = period
	if fleet.Period == "" {
		fleet.Period = "24h"
	}
	if fleet.Monitors == nil {
		fleet.Monitors = []models.MonitorPerformance{} // json: services
	}
	if fleet.Timeline == nil {
		fleet.Timeline = []models.FleetTimelinePoint{}
	}
	jsonOK(w, fleet)
}
