package www

import (
	"net/http"
	"strconv"

	"shingoedge/changeover"
	"shingoedge/store"
)

func (h *Handlers) handleChangeover(w http.ResponseWriter, r *http.Request) {
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
	var fromJob, toJob, state string
	var active bool
	var jobStyles []store.JobStyle
	var activeJobStyleName string

	if activeLine != nil {
		activeLineID = activeLine.ID
		m := h.engine.ChangeoverMachine(activeLine.ID)
		if m != nil {
			fromJob, toJob, state, active = m.Info()
		}
		jobStyles, _ = db.ListJobStylesByLine(activeLine.ID)

		// Resolve active job style name for the "From" field
		if activeLine.ActiveJobStyleID != nil {
			for _, js := range jobStyles {
				if js.ID == *activeLine.ActiveJobStyleID {
					activeJobStyleName = js.Name
					break
				}
			}
		}
	}

	var changeoverLog []store.ChangeoverLog
	if activeLineID > 0 {
		changeoverLog, _ = db.ListCurrentChangeoverLog(activeLineID)
	}

	anomalies, rpMap := loadAnomalyData(h)

	data := map[string]interface{}{
		"Page":              "changeover",
		"Lines":             lines,
		"ActiveLineID":      activeLineID,
		"JobStyles":         jobStyles,
		"ActiveJobStyle":    activeJobStyleName,
		"ChangeoverLog":     changeoverLog,
		"Anomalies":         anomalies,
		"ReportingPointMap": rpMap,
		"Changeover": map[string]interface{}{
			"Active":       active,
			"FromJobStyle": fromJob,
			"ToJobStyle":   toJob,
			"State":        state,
			"StateIndex":   changeover.StateIndex(state),
		},
	}

	h.renderTemplate(w, "changeover.html", data)
}
