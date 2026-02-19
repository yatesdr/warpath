package www

import (
	"net/http"
)

func (h *Handlers) handleManualOrder(w http.ResponseWriter, r *http.Request) {
	db := h.engine.DB()

	payloads, _ := db.ListPayloads()
	nodes, _ := db.ListLocationNodes()
	anomalies, rpMap := loadAnomalyData(h)

	data := map[string]interface{}{
		"Page":              "manual-order",
		"Payloads":          payloads,
		"Nodes":             nodes,
		"Anomalies":         anomalies,
		"ReportingPointMap": rpMap,
	}

	h.renderTemplate(w, "manual-order.html", data)
}
