package api

import (
	"log/slog"
	"strings"
)

// auditActor returns the attempted identity for unauthenticated security events.
func auditActor(identity string) string {
	identity = strings.TrimSpace(identity)
	if identity == "" {
		return "unknown"
	}
	return identity
}

// recordSecurityEvent writes a structured warn log and an audit_log row.
func (s *Server) recordSecurityEvent(msg, actor, action, resource, detail string, logAttrs ...any) {
	slog.Warn(msg, logAttrs...)
	if s.store != nil {
		_ = s.store.InsertAudit(actor, action, resource, detail)
	}
}
