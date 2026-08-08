package store

import (
	"database/sql"
	"strings"
	"time"

	"github.com/sentinel-monitoring/sentinel/internal/models"
)

func (s *Store) CreateIncident(inc *models.Incident) error {
	inc.ID = newID()
	_, err := s.db.Exec(`
		INSERT INTO incidents (id, monitor_id, type, message, started_at, resolved_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		inc.ID, inc.MonitorID, string(inc.Type), inc.Message,
		formatTime(inc.StartedAt), nil,
	)
	return err
}

func (s *Store) ResolveOpenIncidents(monitorID string, incidentType models.IncidentType, resolvedAt time.Time) error {
	_, err := s.db.Exec(`
		UPDATE incidents SET resolved_at = ?
		WHERE monitor_id = ? AND type = ? AND resolved_at IS NULL`,
		formatTime(resolvedAt), monitorID, string(incidentType),
	)
	return err
}

func (s *Store) HasOpenIncident(monitorID string, incidentType models.IncidentType) (bool, error) {
	inc, err := s.GetOpenIncident(monitorID, incidentType)
	if err != nil {
		return false, err
	}
	return inc != nil, nil
}

func (s *Store) ResolveIncident(id string, resolvedAt time.Time) error {
	_, err := s.db.Exec(`UPDATE incidents SET resolved_at = ? WHERE id = ?`, formatTime(resolvedAt), id)
	return err
}

func (s *Store) GetOpenIncident(monitorID string, incidentType models.IncidentType) (*models.Incident, error) {
	row := s.db.QueryRow(`
		SELECT id, monitor_id, type, message, started_at, resolved_at
		FROM incidents
		WHERE monitor_id = ? AND type = ? AND resolved_at IS NULL
		ORDER BY started_at DESC LIMIT 1`,
		monitorID, string(incidentType),
	)
	return scanIncident(row)
}

func scanIncident(row interface {
	Scan(dest ...any) error
}) (*models.Incident, error) {
	var inc models.Incident
	var incType string
	var startedAt string
	var resolvedAt sql.NullString

	err := row.Scan(&inc.ID, &inc.MonitorID, &incType, &inc.Message, &startedAt, &resolvedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	inc.Type = models.IncidentType(incType)
	if t, err := parseTime(startedAt); err == nil {
		inc.StartedAt = t
	}
	if resolvedAt.Valid && resolvedAt.String != "" {
		if t, err := parseTime(resolvedAt.String); err == nil {
			inc.ResolvedAt = &t
		}
	}
	return &inc, nil
}

func (s *Store) ListIncidents(limit int, openOnly bool) ([]models.IncidentListItem, error) {
	return s.ListIncidentsScoped(limit, openOnly, "")
}

func (s *Store) ListIncidentsByTenant(limit int, openOnly bool, tenantID string) ([]models.IncidentListItem, error) {
	if tenantID == "" {
		return []models.IncidentListItem{}, nil
	}
	return s.ListIncidentsScoped(limit, openOnly, tenantID)
}

func (s *Store) ListIncidentsScoped(limit int, openOnly bool, tenantID string) ([]models.IncidentListItem, error) {
	if limit <= 0 {
		limit = 100
	}
	q := `
		SELECT i.id, i.monitor_id, i.type, i.message, i.started_at, i.resolved_at,
			COALESCE(m.name, pt.name, '') AS monitor_name
		FROM incidents i
		LEFT JOIN monitors m ON m.id = i.monitor_id
		LEFT JOIN performance_targets pt ON pt.id = i.monitor_id`
	var conds []string
	var args []any
	if openOnly {
		conds = append(conds, `i.resolved_at IS NULL`)
	}
	if tenantID != "" {
		conds = append(conds, `(m.tenant_id = ? OR pt.tenant_id = ?)`)
		args = append(args, tenantID, tenantID)
	}
	if len(conds) > 0 {
		q += ` WHERE ` + strings.Join(conds, ` AND `)
	}
	q += ` ORDER BY i.started_at DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.IncidentListItem
	for rows.Next() {
		var item models.IncidentListItem
		var incType string
		var startedAt string
		var resolvedAt sql.NullString
		if err := rows.Scan(
			&item.ID, &item.MonitorID, &incType, &item.Message, &startedAt, &resolvedAt, &item.MonitorName,
		); err != nil {
			return nil, err
		}
		item.Type = models.IncidentType(incType)
		item.StartedAt, _ = parseTime(startedAt)
		if resolvedAt.Valid && resolvedAt.String != "" {
			if t, err := parseTime(resolvedAt.String); err == nil {
				item.ResolvedAt = &t
			}
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) ListIncidentsByMonitor(monitorID string, limit, offset int) ([]models.IncidentListItem, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.db.Query(`
		SELECT i.id, i.monitor_id, i.type, i.message, i.started_at, i.resolved_at,
			COALESCE(m.name, '') AS monitor_name
		FROM incidents i
		LEFT JOIN monitors m ON m.id = i.monitor_id
		WHERE i.monitor_id = ?
		ORDER BY i.started_at DESC
		LIMIT ? OFFSET ?`,
		monitorID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.IncidentListItem
	for rows.Next() {
		var item models.IncidentListItem
		var incType string
		var startedAt string
		var resolvedAt sql.NullString
		if err := rows.Scan(
			&item.ID, &item.MonitorID, &incType, &item.Message, &startedAt, &resolvedAt, &item.MonitorName,
		); err != nil {
			return nil, err
		}
		item.Type = models.IncidentType(incType)
		item.StartedAt, _ = parseTime(startedAt)
		if resolvedAt.Valid && resolvedAt.String != "" {
			if t, err := parseTime(resolvedAt.String); err == nil {
				item.ResolvedAt = &t
			}
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) CountIncidentsByMonitor(monitorID string) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM incidents WHERE monitor_id = ?`, monitorID).Scan(&n)
	return n, err
}

func (s *Store) GetLastSlowAlertAt(monitorID string) (*time.Time, error) {
	var startedAt string
	err := s.db.QueryRow(`
		SELECT started_at FROM incidents
		WHERE monitor_id = ? AND type = ?
		ORDER BY started_at DESC LIMIT 1`,
		monitorID, string(models.IncidentSlow),
	).Scan(&startedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	t, err := parseTime(startedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
