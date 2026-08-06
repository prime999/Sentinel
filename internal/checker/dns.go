package checker

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/sentinel-monitoring/sentinel/internal/models"
)

var defaultDNSRecords = []string{"A", "AAAA", "MX", "TXT", "NS", "CNAME"}

func (c *Checker) probeDNS(ctx context.Context, m *models.Monitor) *models.CheckResult {
	start := time.Now()
	result := &models.CheckResult{
		MonitorID: m.ID,
		Status:    models.StatusUp,
		CheckedAt: start,
	}

	host := strings.TrimSpace(strings.TrimSuffix(m.URL, "."))
	if host == "" {
		result.Status = models.StatusDown
		result.Error = "hostname is required for DNS monitors"
		return result
	}

	recordTypes := parseDNSRecordTypes(m.Config)
	records := make(map[string][]string)
	var lookupErr error

	for _, rt := range recordTypes {
		vals, err := lookupDNS(ctx, host, rt)
		if err != nil {
			lookupErr = err
			continue
		}
		sort.Strings(vals)
		records[rt] = vals
	}

	result.ResponseTimeMs = int(time.Since(start).Milliseconds())
	dnsMs := result.ResponseTimeMs
	result.DNSMs = &dnsMs

	if lookupErr != nil && len(records) == 0 {
		result.Status = models.StatusDown
		result.Error = lookupErr.Error()
		return result
	}

	prevRaw, _ := c.store.GetSnapshot(m.ID, "dns")
	var prev map[string][]string
	if prevRaw != "" {
		_ = json.Unmarshal([]byte(prevRaw), &prev)
	}

	var changes []models.DNSRecordChange
	if prev != nil {
		for _, rt := range recordTypes {
			before := strings.Join(prev[rt], ", ")
			after := strings.Join(records[rt], ", ")
			if before != after {
				changes = append(changes, models.DNSRecordChange{
					Type: rt, Before: before, After: after,
				})
			}
		}
	}

	details := models.DNSDetails{Records: records, Changes: changes}
	if b, err := json.Marshal(details); err == nil {
		result.Details = string(b)
	}

	if len(changes) > 0 {
		var msgs []string
		for _, ch := range changes {
			msgs = append(msgs, fmt.Sprintf("%s changed from %q to %q", ch.Type, ch.Before, ch.After))
		}
		result.Error = strings.Join(msgs, "; ")
	}

	newSnap, _ := json.Marshal(records)
	_ = c.store.SaveSnapshot(m.ID, "dns", string(newSnap))

	return result
}

func parseDNSRecordTypes(config string) []string {
	if config == "" {
		return defaultDNSRecords
	}
	var cfg struct {
		DNSRecords []string `json:"dns_records"`
	}
	if err := json.Unmarshal([]byte(config), &cfg); err == nil && len(cfg.DNSRecords) > 0 {
		return cfg.DNSRecords
	}
	parts := strings.Split(config, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(strings.ToUpper(p))
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return defaultDNSRecords
	}
	return out
}

func lookupDNS(ctx context.Context, host, recordType string) ([]string, error) {
	resolver := net.Resolver{}
	switch recordType {
	case "A":
		ips, err := resolver.LookupIP(ctx, "ip4", host)
		if err != nil {
			return nil, err
		}
		var out []string
		for _, ip := range ips {
			out = append(out, ip.String())
		}
		return out, nil
	case "AAAA":
		ips, err := resolver.LookupIP(ctx, "ip6", host)
		if err != nil {
			return nil, err
		}
		var out []string
		for _, ip := range ips {
			out = append(out, ip.String())
		}
		return out, nil
	case "MX":
		mxs, err := resolver.LookupMX(ctx, host)
		if err != nil {
			return nil, err
		}
		var out []string
		for _, mx := range mxs {
			out = append(out, fmt.Sprintf("%d %s", mx.Pref, mx.Host))
		}
		return out, nil
	case "TXT":
		return resolver.LookupTXT(ctx, host)
	case "NS":
		nss, err := resolver.LookupNS(ctx, host)
		if err != nil {
			return nil, err
		}
		var out []string
		for _, ns := range nss {
			out = append(out, strings.TrimSuffix(ns.Host, "."))
		}
		return out, nil
	case "CNAME":
		cname, err := resolver.LookupCNAME(ctx, host)
		if err != nil {
			return nil, err
		}
		return []string{strings.TrimSuffix(cname, ".")}, nil
	default:
		return nil, fmt.Errorf("unsupported record type: %s", recordType)
	}
}
