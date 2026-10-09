package store

import (
	"database/sql"
	"strings"
	"time"

	"github.com/sentinel-monitoring/sentinel/internal/monitorhost"
)

func (s *Store) migrateV29() error {
	if _, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS sites (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			tenant_id TEXT,
			primary_host TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_sites_tenant ON sites(tenant_id);
	`); err != nil {
		return err
	}
	if err := s.addColumnIfMissing("monitors", "site_id", `
		ALTER TABLE monitors ADD COLUMN site_id TEXT REFERENCES sites(id) ON DELETE SET NULL`); err != nil {
		return err
	}
	if _, err := s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_monitors_site_id ON monitors(site_id)`); err != nil {
		return err
	}
	return s.backfillSitesFromMonitors()
}

func (s *Store) backfillSitesFromMonitors() error {
	rows, err := s.db.Query(`
		SELECT id, type, name, url, tenant_id, site_id FROM monitors
		WHERE type IN ('http', 'ssl', 'dns', 'port') AND url IS NOT NULL AND url != ''`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type groupKey struct {
		tenant string
		host   string
	}
	groups := map[groupKey][]struct {
		id, name string
	}{}

	for rows.Next() {
		var id, mtype, name, rawURL string
		var tenantID, siteID sql.NullString
		if err := rows.Scan(&id, &mtype, &name, &rawURL, &tenantID, &siteID); err != nil {
			return err
		}
		if siteID.Valid && siteID.String != "" {
			continue
		}
		host := monitorhost.NormalizeHost(rawURL)
		if host == "" {
			continue
		}
		t := ""
		if tenantID.Valid {
			t = tenantID.String
		}
		k := groupKey{tenant: t, host: host}
		groups[k] = append(groups[k], struct{ id, name string }{id, name})
	}
	if err := rows.Err(); err != nil {
		return err
	}

	now := time.Now().UTC()
	for k, mons := range groups {
		if len(mons) == 0 {
			continue
		}
		siteName := k.host
		nameCount := map[string]int{}
		for _, m := range mons {
			if n := strings.TrimSpace(m.name); n != "" {
				nameCount[n]++
			}
		}
		bestName := ""
		bestN := 0
		for n, c := range nameCount {
			if c > bestN {
				bestN = c
				bestName = n
			}
		}
		if bestName != "" {
			siteName = bestName
		}

		siteID := newID()
		var tenant interface{}
		if k.tenant != "" {
			tenant = k.tenant
		}
		if _, err := s.db.Exec(`
			INSERT INTO sites (id, name, tenant_id, primary_host, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?)`,
			siteID, siteName, tenant, k.host, formatTime(now), formatTime(now),
		); err != nil {
			return err
		}
		for _, m := range mons {
			if _, err := s.db.Exec(`UPDATE monitors SET site_id = ? WHERE id = ?`, siteID, m.id); err != nil {
				return err
			}
		}
	}
	return nil
}
