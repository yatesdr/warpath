package www

import "shingoedge/store"

// loadAnomalyData loads unconfirmed anomalies and builds a reporting point map
// for display in the global anomaly popover. Used by all page handlers.
func loadAnomalyData(h *Handlers) ([]store.CounterSnapshot, map[int64]map[string]string) {
	db := h.engine.DB()
	anomalies, _ := db.ListUnconfirmedAnomalies()
	reportingPoints, _ := db.ListReportingPoints()

	rpMap := make(map[int64]map[string]string)
	for _, rp := range reportingPoints {
		rpMap[rp.ID] = map[string]string{
			"PLCName": rp.PLCName,
			"TagName": rp.TagName,
		}
	}

	return anomalies, rpMap
}
