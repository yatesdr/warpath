package www

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"shingocore/engine"
	"shingocore/fleet"
	"shingocore/fleet/seerrds"
	"shingocore/rds"
	"shingocore/store"

	"github.com/google/uuid"
)

// nodeSceneInfo holds parsed scene data for a node location, used in the template.
type nodeSceneInfo struct {
	PointName string
	Tasks     string
	BoundMap  string
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
			var props []rds.SceneProperty
			if err := json.Unmarshal([]byte(sp.PropertiesJSON), &props); err == nil {
				if p, ok := rds.FindProperty(props, "bindRobotMap"); ok {
					info.BoundMap = p.StringValue
				}
				if p, ok := rds.FindProperty(props, "binTask"); ok {
					info.Tasks = parseNodeTasks(p.StringValue)
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
	// Scene sync requires the Seer RDS adapter
	adapter, ok := h.engine.Fleet().(*seerrds.Adapter)
	if !ok {
		log.Printf("node sync: fleet backend does not support scene sync")
		http.Redirect(w, r, "/nodes", http.StatusSeeOther)
		return
	}
	scene, err := adapter.RDSClient().GetScene()
	if err != nil {
		log.Printf("node sync: fleet error: %v", err)
		http.Redirect(w, r, "/nodes", http.StatusSeeOther)
		return
	}

	// Phase 1: Sync all scene points (same as handleSceneSync)
	locationSet := make(map[string]string) // instanceName -> areaName
	pointsTotal := 0
	for _, area := range scene.Areas {
		h.engine.DB().DeleteScenePointsByArea(area.Name)

		for _, ap := range area.LogicalMap.AdvancedPoints {
			label := ""
			if p, ok := rds.FindProperty(ap.Property, "label"); ok {
				label = p.StringValue
			}
			propsJSON, _ := json.Marshal(ap.Property)
			sp := &store.ScenePoint{
				AreaName:       area.Name,
				InstanceName:   ap.InstanceName,
				ClassName:      ap.ClassName,
				Label:          label,
				PosX:           ap.Pos.X,
				PosY:           ap.Pos.Y,
				PosZ:           ap.Pos.Z,
				Dir:            ap.Dir,
				PropertiesJSON: string(propsJSON),
			}
			h.engine.DB().UpsertScenePoint(sp)
			pointsTotal++
		}

		for _, blg := range area.LogicalMap.BinLocationsList {
			for _, bin := range blg.BinLocationList {
				locationSet[bin.InstanceName] = area.Name
				propsJSON, _ := json.Marshal(bin.Property)
				sp := &store.ScenePoint{
					AreaName:       area.Name,
					InstanceName:   bin.InstanceName,
					ClassName:      bin.ClassName,
					PointName:      bin.PointName,
					GroupName:      bin.GroupName,
					PosX:           bin.Pos.X,
					PosY:           bin.Pos.Y,
					PosZ:           bin.Pos.Z,
					PropertiesJSON: string(propsJSON),
				}
				h.engine.DB().UpsertScenePoint(sp)
				pointsTotal++
			}
		}
	}

	// Phase 2: Create nodes for locations not yet in DB
	created := 0
	for instanceName, areaName := range locationSet {
		if _, err := h.engine.DB().GetNodeByVendorLocation(instanceName); err == nil {
			continue
		}
		node := &store.Node{
			Name:           instanceName,
			VendorLocation: instanceName,
			NodeType:       "storage",
			Zone:           areaName,
			Capacity:       1,
			Enabled:        true,
		}
		if err := h.engine.DB().CreateNode(node); err != nil {
			continue
		}
		h.engine.NodeState().RefreshNodeMeta(node.ID)
		h.engine.Events.Emit(engine.Event{Type: engine.EventNodeUpdated, Payload: engine.NodeUpdatedEvent{
			NodeID: node.ID, NodeName: node.Name, Action: "created",
		}})
		created++
	}

	// Phase 3: Delete nodes not present in current scene
	deleted := 0
	nodes, _ := h.engine.DB().ListNodes()
	for _, n := range nodes {
		if _, inScene := locationSet[n.VendorLocation]; !inScene {
			h.engine.DB().DeleteNode(n.ID)
			h.engine.Events.Emit(engine.Event{Type: engine.EventNodeUpdated, Payload: engine.NodeUpdatedEvent{
				NodeID: n.ID, NodeName: n.Name, Action: "deleted",
			}})
			deleted++
		}
	}

	// Phase 4: Update zones on remaining nodes
	if deleted > 0 || created > 0 {
		nodes, _ = h.engine.DB().ListNodes()
	}
	for _, n := range nodes {
		if zone, ok := locationSet[n.VendorLocation]; ok && n.Zone != zone {
			n.Zone = zone
			h.engine.DB().UpdateNode(n)
		}
	}

	locationNames := make([]string, 0, len(locationSet))
	for k := range locationSet {
		locationNames = append(locationNames, k)
	}
	log.Printf("node sync: %d scene points, locations=%v, created %d, deleted %d nodes", pointsTotal, locationNames, created, deleted)
	http.Redirect(w, r, "/nodes", http.StatusSeeOther)
}

func (h *Handlers) handleSceneSync(w http.ResponseWriter, r *http.Request) {
	// Scene sync requires the Seer RDS adapter
	adapter, ok := h.engine.Fleet().(*seerrds.Adapter)
	if !ok {
		log.Printf("scene sync: fleet backend does not support scene sync")
		http.Redirect(w, r, "/nodes", http.StatusSeeOther)
		return
	}
	scene, err := adapter.RDSClient().GetScene()
	if err != nil {
		log.Printf("scene sync: fleet error: %v", err)
		http.Redirect(w, r, "/nodes", http.StatusSeeOther)
		return
	}

	// Build location-to-area lookup for node zone updates
	locationArea := make(map[string]string)

	total := 0
	for _, area := range scene.Areas {
		// Clear existing points for this area before re-sync
		h.engine.DB().DeleteScenePointsByArea(area.Name)

		// Persist advanced points (LocationMark, ActionPoint, ChargePoint)
		for _, ap := range area.LogicalMap.AdvancedPoints {
			label := ""
			if p, ok := rds.FindProperty(ap.Property, "label"); ok {
				label = p.StringValue
			}
			propsJSON, _ := json.Marshal(ap.Property)
			sp := &store.ScenePoint{
				AreaName:       area.Name,
				InstanceName:   ap.InstanceName,
				ClassName:      ap.ClassName,
				Label:          label,
				PosX:           ap.Pos.X,
				PosY:           ap.Pos.Y,
				PosZ:           ap.Pos.Z,
				Dir:            ap.Dir,
				PropertiesJSON: string(propsJSON),
			}
			if err := h.engine.DB().UpsertScenePoint(sp); err != nil {
				log.Printf("scene sync: upsert point %s: %v", ap.InstanceName, err)
			}
			total++
		}

		// Persist bin locations
		for _, blg := range area.LogicalMap.BinLocationsList {
			for _, bin := range blg.BinLocationList {
				locationArea[bin.InstanceName] = area.Name
				propsJSON, _ := json.Marshal(bin.Property)
				sp := &store.ScenePoint{
					AreaName:       area.Name,
					InstanceName:   bin.InstanceName,
					ClassName:      bin.ClassName,
					PointName:      bin.PointName,
					GroupName:      bin.GroupName,
					PosX:           bin.Pos.X,
					PosY:           bin.Pos.Y,
					PosZ:           bin.Pos.Z,
					PropertiesJSON: string(propsJSON),
				}
				if err := h.engine.DB().UpsertScenePoint(sp); err != nil {
					log.Printf("scene sync: upsert bin %s: %v", bin.InstanceName, err)
				}
				total++
			}
		}
	}

	// Update node zones from bin locations
	nodes, _ := h.engine.DB().ListNodes()
	for _, node := range nodes {
		if node.VendorLocation == "" || node.Zone != "" {
			continue
		}
		if zone, ok := locationArea[node.VendorLocation]; ok {
			node.Zone = zone
			h.engine.DB().UpdateNode(node)
			h.engine.Events.Emit(engine.Event{Type: engine.EventNodeUpdated, Payload: engine.NodeUpdatedEvent{
				NodeID: node.ID, NodeName: node.Name, Action: "updated",
			}})
		}
	}

	log.Printf("scene sync: persisted %d scene points", total)
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

func (h *Handlers) handleTestOrder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FromNodeID int64 `json:"from_node_id"`
		ToNodeID   int64 `json:"to_node_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.FromNodeID == req.ToNodeID {
		h.jsonError(w, "source and destination must be different", http.StatusBadRequest)
		return
	}

	sourceNode, err := h.engine.DB().GetNode(req.FromNodeID)
	if err != nil {
		h.jsonError(w, "source node not found", http.StatusNotFound)
		return
	}
	destNode, err := h.engine.DB().GetNode(req.ToNodeID)
	if err != nil {
		h.jsonError(w, "destination node not found", http.StatusNotFound)
		return
	}

	edgeUUID := "test-" + uuid.New().String()[:8]

	order := &store.Order{
		EdgeUUID:     edgeUUID,
		ClientID:     "shingocore",
		FactoryID:    h.engine.AppConfig().FactoryID,
		OrderType:    "move",
		Status:       "pending",
		SourceNodeID: &sourceNode.ID,
		DestNodeID:   &destNode.ID,
		PickupNode:   sourceNode.Name,
		DeliveryNode: destNode.Name,
		PayloadDesc:  "test order from shingo core",
	}
	if err := h.engine.DB().CreateOrder(order); err != nil {
		h.jsonError(w, "failed to create order: "+err.Error(), http.StatusInternalServerError)
		return
	}
	h.engine.DB().UpdateOrderStatus(order.ID, "pending", "test order created")

	vendorOrderID := fmt.Sprintf("sg-%d-%s", order.ID, uuid.New().String()[:8])
	fleetReq := fleet.TransportOrderRequest{
		OrderID:    vendorOrderID,
		ExternalID: edgeUUID,
		FromLoc:    sourceNode.VendorLocation,
		ToLoc:      destNode.VendorLocation,
	}

	if _, err := h.engine.Fleet().CreateTransportOrder(fleetReq); err != nil {
		h.engine.DB().UpdateOrderStatus(order.ID, "failed", err.Error())
		h.jsonError(w, "fleet dispatch failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.engine.DB().UpdateOrderVendor(order.ID, vendorOrderID, "CREATED", "")
	h.engine.DB().UpdateOrderStatus(order.ID, "dispatched", "vendor order "+vendorOrderID)

	h.engine.Events.Emit(engine.Event{
		Type: engine.EventOrderDispatched,
		Payload: engine.OrderDispatchedEvent{
			OrderID:       order.ID,
			VendorOrderID: vendorOrderID,
			SourceNode:    sourceNode.Name,
			DestNode:      destNode.Name,
		},
	})

	h.jsonOK(w, map[string]any{
		"order_id":        order.ID,
		"vendor_order_id": vendorOrderID,
		"from":            sourceNode.Name,
		"to":              destNode.Name,
	})
}
