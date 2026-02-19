package www

import (
	"net/http"
)

func (h *Handlers) handleDashboard(w http.ResponseWriter, r *http.Request) {
	activeOrders, _ := h.engine.DB().ListActiveOrders()
	nodes, _ := h.engine.DB().ListNodes()

	// Count orders by status
	statusCounts := map[string]int{}
	for _, o := range activeOrders {
		statusCounts[o.Status]++
	}

	// Node stats
	enabledNodes := 0
	for _, n := range nodes {
		if n.Enabled {
			enabledNodes++
		}
	}

	// Fleet health check
	fleetOK := false
	if err := h.engine.Fleet().Ping(); err == nil {
		fleetOK = true
	}

	msgOK := h.engine.MsgClient().IsConnected()

	data := map[string]any{
		"Page":         "dashboard",
		"ActiveOrders": activeOrders,
		"StatusCounts": statusCounts,
		"TotalOrders":  len(activeOrders),
		"TotalNodes":   len(nodes),
		"EnabledNodes": enabledNodes,
		"FleetOK":      fleetOK,
		"MessagingOK":  msgOK,
		"Authenticated": h.isAuthenticated(r),
	}
	h.render(w, "dashboard.html", data)
}
