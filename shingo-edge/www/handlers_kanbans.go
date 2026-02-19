package www

import (
	"net/http"
	"strconv"

	"shingoedge/store"
)

func (h *Handlers) handleKanbans(w http.ResponseWriter, r *http.Request) {
	db := h.engine.DB()

	lines, _ := db.ListProductionLines()

	// Determine active line from query param (0 = all lines)
	var activeLineID int64
	if lineParam := r.URL.Query().Get("line"); lineParam != "" {
		if id, err := strconv.ParseInt(lineParam, 10, 64); err == nil {
			// Validate line exists
			for _, l := range lines {
				if l.ID == id {
					activeLineID = id
					break
				}
			}
		}
	}

	var activeOrders []store.Order
	if activeLineID > 0 {
		activeOrders, _ = db.ListActiveOrdersByLine(activeLineID)
	} else {
		activeOrders, _ = db.ListActiveOrders()
	}

	knownNodes, _ := db.ListKnownNodes()
	anomalies, rpMap := loadAnomalyData(h)

	data := map[string]interface{}{
		"Page":              "kanbans",
		"Lines":             lines,
		"ActiveLineID":      activeLineID,
		"ActiveOrders":      activeOrders,
		"KnownNodes":        knownNodes,
		"Anomalies":         anomalies,
		"ReportingPointMap": rpMap,
	}

	h.renderTemplate(w, "kanbans.html", data)
}
