package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/sentinel-monitoring/sentinel/internal/models"
)

func (s *Store) ListSites(tenantScope string) ([]models.Site, error) {
	q := `SELECT id, name, tenant_id, primary_host, created_at, updated_at FROM sites`
	var args []interface{}
	if tenantScope != "" {
		q += ` WHERE tenant_id = ?`
		args = append(args, tenantScope)
	}
	q += ` ORDER BY name COLLATE NOCASE ASC`
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Site
	for rows.Next() {
		site, err := scanSiteRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *site)
	}
	return out, rows.Err()
}

func (s *Store) GetSite(id string) (*models.Site, error) {
	row := s.db.QueryRow(`SELECT id, name, tenant_id, primary_host, created_at, updated_at FROM sites WHERE id = ?`, id)
	site, err := scanSiteRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return site, err
}

func (s *Store) CreateSite(site *models.Site) error {
	now := time.Now().UTC()
	site.ID = newID()
	site.CreatedAt = now
	site.UpdatedAt = now
	_, err := s.db.Exec(`
		INSERT INTO sites (id, name, tenant_id, primary_host, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		site.ID, site.Name, nullString(site.TenantID), site.PrimaryHost,
		formatTime(site.CreatedAt), formatTime(site.UpdatedAt),
	)
	return err
}

func (s *Store) UpdateSite(site *models.Site) error {
	site.UpdatedAt = time.Now().UTC()
	res, err := s.db.Exec(`
		UPDATE sites SET name = ?, primary_host = ?, updated_at = ? WHERE id = ?`,
		site.Name, site.PrimaryHost, formatTime(site.UpdatedAt), site.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("site not found")
	}
	return nil
}

func (s *Store) DeleteSite(id string) error {
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM monitors WHERE site_id = ?`, id).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return fmt.Errorf("site has monitors")
	}
	_, err := s.db.Exec(`DELETE FROM sites WHERE id = ?`, id)
	return err
}

func (s *Store) CountMonitorsForSite(siteID string) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM monitors WHERE site_id = ?`, siteID).Scan(&n)
	return n, err
}

func (s *Store) ListSitesForExport(tenantScope string) ([]models.Site, error) {
	return s.ListSites(tenantScope)
}

func (s *Store) InsertSiteImport(site *models.Site) error {
	if site.ID == "" {
		site.ID = newID()
	}
	if site.CreatedAt.IsZero() {
		site.CreatedAt = time.Now().UTC()
	}
	site.UpdatedAt = time.Now().UTC()
	_, err := s.db.Exec(`
		INSERT INTO sites (id, name, tenant_id, primary_host, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		site.ID, site.Name, nullString(site.TenantID), site.PrimaryHost,
		formatTime(site.CreatedAt), formatTime(site.UpdatedAt),
	)
	return err
}

func (s *Store) ReplaceSiteImport(site *models.Site) error {
	site.UpdatedAt = time.Now().UTC()
	res, err := s.db.Exec(`
		UPDATE sites SET name = ?, tenant_id = ?, primary_host = ?, updated_at = ? WHERE id = ?`,
		site.Name, nullString(site.TenantID), site.PrimaryHost, formatTime(site.UpdatedAt), site.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("site not found")
	}
	return nil
}

func scanSiteRow(row interface {
	Scan(dest ...any) error
}) (*models.Site, error) {
	var id, name, primaryHost, createdAt, updatedAt string
	var tenantID sql.NullString
	if err := row.Scan(&id, &name, &tenantID, &primaryHost, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	site := &models.Site{
		ID:          id,
		Name:        name,
		TenantID:    nullableString(tenantID),
		PrimaryHost: primaryHost,
	}
	if t, err := parseTime(createdAt); err == nil {
		site.CreatedAt = t
	}
	if t, err := parseTime(updatedAt); err == nil {
		site.UpdatedAt = t
	}
	return site, nil
}
