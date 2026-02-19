package www

import (
	"net/http"
)

func (h *Handlers) handleDiagnostics(w http.ResponseWriter, r *http.Request) {
	auditLog, _ := h.engine.DB().ListAuditLog(50)

	fleetOK := false
	fleetName := h.engine.Fleet().Name()
	if err := h.engine.Fleet().Ping(); err == nil {
		fleetOK = true
	}

	msgOK := h.engine.MsgClient().IsConnected()
	trackerCount := 0
	if t := h.engine.Tracker(); t != nil {
		trackerCount = t.ActiveCount()
	}

	data := map[string]any{
		"Page":          "diagnostics",
		"AuditLog":      auditLog,
		"FleetOK":       fleetOK,
		"FleetName":     fleetName,
		"MessagingOK":   msgOK,
		"PollerActive":  trackerCount,
		"SSEClients":    h.eventHub.ClientCount(),
		"Authenticated": h.isAuthenticated(r),
	}
	h.render(w, "diagnostics.html", data)
}
