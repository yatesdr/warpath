package www

import (
	"net/http"
	"strconv"

	"shingoedge/store"
)

func (h *Handlers) handleMaterial(w http.ResponseWriter, r *http.Request) {
	db := h.engine.DB()

	lines, _ := db.ListProductionLines()

	// Determine active line from query param or default to first
	var activeLine *store.ProductionLine
	if lineParam := r.URL.Query().Get("line"); lineParam != "" {
		if lineID, err := strconv.ParseInt(lineParam, 10, 64); err == nil {
			for i := range lines {
				if lines[i].ID == lineID {
					activeLine = &lines[i]
					break
				}
			}
		}
	}
	if activeLine == nil && len(lines) > 0 {
		activeLine = &lines[0]
	}

	var activeLineID int64
	var activeStyleName string
	var payloads []store.Payload

	if activeLine != nil {
		activeLineID = activeLine.ID
		if activeLine.ActiveJobStyleID != nil {
			js, err := db.GetJobStyle(*activeLine.ActiveJobStyleID)
			if err == nil {
				activeStyleName = js.Name
				payloads, _ = db.ListPayloadsByJobStyle(js.ID)
			}
		}
	}

	if payloads == nil && activeLine != nil {
		// No active style set — show all payloads for this line's styles
		styles, _ := db.ListJobStylesByLine(activeLineID)
		for _, s := range styles {
			sp, _ := db.ListPayloadsByJobStyle(s.ID)
			payloads = append(payloads, sp...)
		}
	}

	if payloads == nil {
		payloads, _ = db.ListPayloads()
	}

	anomalies, rpMap := loadAnomalyData(h)

	data := map[string]interface{}{
		"Page":              "material",
		"Lines":             lines,
		"ActiveLineID":      activeLineID,
		"Payloads":          payloads,
		"ActiveJobStyle":    activeStyleName,
		"Anomalies":         anomalies,
		"ReportingPointMap": rpMap,
	}

	h.renderTemplate(w, "material.html", data)
}
