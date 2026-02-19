package www

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"shingocore/engine"
	"shingocore/fleet"
	"shingocore/store"
)

// nodeSceneInfo holds parsed scene data for a node location, used in the template.
type nodeSceneInfo struct {
	PointName string
	Tasks     string
	BoundMap  string
}

// sceneProperty is a minimal representation of a scene point property for template rendering.
type sceneProperty struct {
	Key         string `json:"key"`
	StringValue string `json:"stringValue,omitempty"`
}

func findSceneProperty(props []sceneProperty, key string) (string, bool) {
	for _, p := range props {
		if p.Key == key {
			return p.StringValue, true
		}
	}
	return "", false
}

// parseNodeTasks extracts task names from a binTask JSON property value.
// Input is like: [{"Load":{}},{"Unload":{}}]  →  "Load, Unload"
func parseNodeTasks(jsonStr string) string {
	var tasks []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &tasks); err != nil {
		return ""
	}
	var names []string
	for _, t := range tasks {
		for k := range t {
			names = append(names, k)
		}
	}
	return strings.Join(names, ", ")
}

func (h *Handlers) handleNodes(w http.ResponseWriter, r *http.Request) {
	nodes, _ := h.engine.DB().ListNodes()
	states, _ := h.engine.NodeState().GetAllNodeStates()

	// Build count map and collect distinct zones
	counts := make(map[int64]int, len(nodes))
	zoneSet := map[string]bool{}
	for _, n := range nodes {
		if st, ok := states[n.ID]; ok {
			counts[n.ID] = st.ItemCount
		}
		if n.Zone != "" {
			zoneSet[n.Zone] = true
		}
	}
	zones := make([]string, 0, len(zoneSet))
	for z := range zoneSet {
		zones = append(zones, z)
	}

	// Build scene data for template
	scenePoints, _ := h.engine.DB().ListScenePoints()
	nodeLabels := make(map[string]string)
	nodeInfo := make(map[string]*nodeSceneInfo)
	mapGroups := make(map[string][]*store.ScenePoint)
	for _, sp := range scenePoints {
		if sp.ClassName == "GeneralLocation" {
			nodeLabels[sp.InstanceName] = sp.Label
			info := &nodeSceneInfo{PointName: sp.PointName}
			var props []sceneProperty
			if err := json.Unmarshal([]byte(sp.PropertiesJSON), &props); err == nil {
				if v, ok := findSceneProperty(props, "bindRobotMap"); ok {
					info.BoundMap = v
				}
				if v, ok := findSceneProperty(props, "binTask"); ok {
					info.Tasks = parseNodeTasks(v)
				}
			}
			nodeInfo[sp.InstanceName] = info
		} else {
			mapGroups[sp.ClassName] = append(mapGroups[sp.ClassName], sp)
		}
	}

	data := map[string]any{
		"Page":          "nodes",
		"Nodes":         nodes,
		"Counts":        counts,
		"Zones":         zones,
		"Authenticated": h.isAuthenticated(r),
		"NodeLabels":    nodeLabels,
		"NodeInfo":      nodeInfo,
		"MapGroups":     mapGroups,
		"MapClassOrder": []string{"ActionPoint", "ChargePoint", "LocationMark"},
	}
	h.render(w, "nodes.html", data)
}

func (h *Handlers) handleNodeCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	capacity, _ := strconv.Atoi(r.FormValue("capacity"))
	node := &store.Node{
		Name:           r.FormValue("name"),
		VendorLocation: r.FormValue("vendor_location"),
		NodeType:       r.FormValue("node_type"),
		Zone:           r.FormValue("zone"),
		Capacity:       capacity,
		Enabled:        r.FormValue("enabled") == "on",
	}

	if err := h.engine.DB().CreateNode(node); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.engine.NodeState().RefreshNodeMeta(node.ID)
	h.engine.Events.Emit(engine.Event{Type: engine.EventNodeUpdated, Payload: engine.NodeUpdatedEvent{
		NodeID: node.ID, NodeName: node.Name, Action: "created",
	}})

	http.Redirect(w, r, "/nodes", http.StatusSeeOther)
}

func (h *Handlers) handleNodeUpdate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid node id", http.StatusBadRequest)
		return
	}

	node, err := h.engine.DB().GetNode(id)
	if err != nil {
		http.Error(w, "node not found", http.StatusNotFound)
		return
	}

	capacity, _ := strconv.Atoi(r.FormValue("capacity"))
	node.Name = r.FormValue("name")
	node.VendorLocation = r.FormValue("vendor_location")
	node.NodeType = r.FormValue("node_type")
	node.Zone = r.FormValue("zone")
	node.Capacity = capacity
	node.Enabled = r.FormValue("enabled") == "on"

	if err := h.engine.DB().UpdateNode(node); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.engine.NodeState().RefreshNodeMeta(node.ID)
	h.engine.Events.Emit(engine.Event{Type: engine.EventNodeUpdated, Payload: engine.NodeUpdatedEvent{
		NodeID: node.ID, NodeName: node.Name, Action: "updated",
	}})

	http.Redirect(w, r, "/nodes", http.StatusSeeOther)
}

func (h *Handlers) handleNodeSyncFleet(w http.ResponseWriter, r *http.Request) {
	syncer, ok := h.engine.Fleet().(fleet.SceneSyncer)
	if !ok {
		log.Printf("node sync: fleet backend does not support scene sync")
		http.Redirect(w, r, "/nodes", http.StatusSeeOther)
		return
	}
	areas, err := syncer.GetSceneAreas()
	if err != nil {
		log.Printf("node sync: fleet error: %v", err)
		http.Redirect(w, r, "/nodes", http.StatusSeeOther)
		return
	}
	pointsTotal, locationSet := h.engine.SyncScenePoints(areas)
	created, deleted := h.engine.SyncFleetNodes(locationSet)
	log.Printf("node sync: %d scene points, created %d, deleted %d nodes", pointsTotal, created, deleted)
	http.Redirect(w, r, "/nodes", http.StatusSeeOther)
}

func (h *Handlers) handleSceneSync(w http.ResponseWriter, r *http.Request) {
	syncer, ok := h.engine.Fleet().(fleet.SceneSyncer)
	if !ok {
		log.Printf("scene sync: fleet backend does not support scene sync")
		http.Redirect(w, r, "/nodes", http.StatusSeeOther)
		return
	}
	areas, err := syncer.GetSceneAreas()
	if err != nil {
		log.Printf("scene sync: fleet error: %v", err)
		http.Redirect(w, r, "/nodes", http.StatusSeeOther)
		return
	}
	total, locationSet := h.engine.SyncScenePoints(areas)
	h.engine.UpdateNodeZones(locationSet, false)
	log.Printf("scene sync: %d points synced", total)
	http.Redirect(w, r, "/nodes", http.StatusSeeOther)
}

func (h *Handlers) handleNodeDelete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid node id", http.StatusBadRequest)
		return
	}

	node, err := h.engine.DB().GetNode(id)
	if err != nil {
		http.Error(w, "node not found", http.StatusNotFound)
		return
	}

	if err := h.engine.DB().DeleteNode(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.engine.Events.Emit(engine.Event{Type: engine.EventNodeUpdated, Payload: engine.NodeUpdatedEvent{
		NodeID: id, NodeName: node.Name, Action: "deleted",
	}})

	http.Redirect(w, r, "/nodes", http.StatusSeeOther)
}

func (h *Handlers) apiNodeOccupancy(w http.ResponseWriter, r *http.Request) {
	np, ok := h.engine.Fleet().(fleet.NodeOccupancyProvider)
	if !ok {
		h.jsonError(w, "fleet backend does not support occupancy status", http.StatusNotImplemented)
		return
	}
	locations, err := np.GetNodeOccupancy()
	if err != nil {
		h.jsonError(w, "fleet error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	nodes, _ := h.engine.DB().ListNodes()

	// Build lookup maps
	locMap := make(map[string]bool, len(locations))
	for _, loc := range locations {
		locMap[loc.ID] = loc.Occupied
	}

	nodeVendor := make(map[string]string, len(nodes))
	for _, n := range nodes {
		if n.VendorLocation != "" {
			nodeVendor[n.VendorLocation] = n.Name
		}
	}

	type entry struct {
		LocationID    string `json:"location_id"`
		NodeName      string `json:"node_name"`
		FleetOccupied *bool  `json:"fleet_occupied"`
		InShinGo      bool   `json:"in_shingo"`
		Discrepancy   string `json:"discrepancy"`
	}

	var results []entry

	// Locations in fleet
	for _, loc := range locations {
		e := entry{
			LocationID:    loc.ID,
			FleetOccupied: &loc.Occupied,
			InShinGo:      nodeVendor[loc.ID] != "",
			NodeName:      nodeVendor[loc.ID],
		}
		if !e.InShinGo {
			e.Discrepancy = "fleet_only"
		}
		results = append(results, e)
	}

	// Nodes in ShinGo but not in fleet
	for _, n := range nodes {
		if n.VendorLocation == "" {
			continue
		}
		if _, ok := locMap[n.VendorLocation]; !ok {
			results = append(results, entry{
				LocationID:  n.VendorLocation,
				NodeName:    n.Name,
				InShinGo:    true,
				Discrepancy: "shingo_only",
			})
		}
	}

	h.jsonOK(w, results)
}

