package www

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"shingo/protocol"
	"shingocore/engine"
	"shingocore/fleet"
	"shingocore/fleet/seerrds"
	"shingocore/rds"
	"shingocore/store"

	"github.com/google/uuid"
)

// --- Page ---

func (h *Handlers) handleTestOrders(w http.ResponseWriter, r *http.Request) {
	nodes, _ := h.engine.DB().ListNodes()
	payloadTypes, _ := h.engine.DB().ListPayloadTypes()
	data := map[string]any{
		"Page":          "test-orders",
		"Authenticated": h.isAuthenticated(r),
		"Nodes":         nodes,
		"PayloadTypes":  payloadTypes,
	}
	h.render(w, "test-orders.html", data)
}

// --- Section A: Kafka Order APIs ---

func (h *Handlers) apiTestOrderSubmit(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OrderType       string  `json:"order_type"`
		PickupNode      string  `json:"pickup_node"`
		DeliveryNode    string  `json:"delivery_node"`
		PayloadTypeCode string  `json:"payload_type_code"`
		Quantity        float64 `json:"quantity"`
		Priority        int     `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.OrderType == "" {
		h.jsonError(w, "order_type is required", http.StatusBadRequest)
		return
	}
	if req.Quantity <= 0 {
		req.Quantity = 1
	}

	cfg := h.engine.AppConfig()
	orderUUID := "test-" + uuid.New().String()[:8]

	src := protocol.Address{Role: protocol.RoleEdge, Station: "core-test"}
	dst := protocol.Address{Role: protocol.RoleCore, Station: cfg.Messaging.StationID}

	orderReq := &protocol.OrderRequest{
		OrderUUID:       orderUUID,
		OrderType:       req.OrderType,
		PayloadTypeCode: req.PayloadTypeCode,
		Quantity:        req.Quantity,
		DeliveryNode:    req.DeliveryNode,
		PickupNode:      req.PickupNode,
		Priority:        req.Priority,
		PayloadDesc:     "test order from shingo core",
	}

	env, err := protocol.NewEnvelope(protocol.TypeOrderRequest, src, dst, orderReq)
	if err != nil {
		h.jsonError(w, "build envelope: "+err.Error(), http.StatusInternalServerError)
		return
	}

	data, err := env.Encode()
	if err != nil {
		h.jsonError(w, "encode envelope: "+err.Error(), http.StatusInternalServerError)
		return
	}

	topic := cfg.Messaging.OrdersTopic
	log.Printf("test-orders: publishing %s to %s: %s", env.Type, topic, string(data))

	if err := h.engine.MsgClient().Publish(topic, data); err != nil {
		h.jsonError(w, "publish failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.jsonOK(w, map[string]any{
		"order_uuid":  orderUUID,
		"envelope_id": env.ID,
	})
}

func (h *Handlers) apiTestOrderCancel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OrderUUID string `json:"order_uuid"`
		Reason    string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.OrderUUID == "" {
		h.jsonError(w, "order_uuid is required", http.StatusBadRequest)
		return
	}
	if req.Reason == "" {
		req.Reason = "cancelled via test page"
	}

	cfg := h.engine.AppConfig()
	src := protocol.Address{Role: protocol.RoleEdge, Station: "core-test"}
	dst := protocol.Address{Role: protocol.RoleCore, Station: cfg.Messaging.StationID}

	cancelReq := &protocol.OrderCancel{
		OrderUUID: req.OrderUUID,
		Reason:    req.Reason,
	}

	env, err := protocol.NewEnvelope(protocol.TypeOrderCancel, src, dst, cancelReq)
	if err != nil {
		h.jsonError(w, "build envelope: "+err.Error(), http.StatusInternalServerError)
		return
	}

	data, err := env.Encode()
	if err != nil {
		h.jsonError(w, "encode envelope: "+err.Error(), http.StatusInternalServerError)
		return
	}

	topic := cfg.Messaging.OrdersTopic
	log.Printf("test-orders: publishing %s to %s: %s", env.Type, topic, string(data))

	if err := h.engine.MsgClient().Publish(topic, data); err != nil {
		h.jsonError(w, "publish failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.jsonOK(w, map[string]string{"status": "cancel sent", "order_uuid": req.OrderUUID})
}

func (h *Handlers) apiTestOrderReceipt(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OrderUUID   string  `json:"order_uuid"`
		ReceiptType string  `json:"receipt_type"`
		FinalCount  float64 `json:"final_count"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.OrderUUID == "" {
		h.jsonError(w, "order_uuid is required", http.StatusBadRequest)
		return
	}
	if req.ReceiptType == "" {
		req.ReceiptType = "full"
	}

	cfg := h.engine.AppConfig()
	src := protocol.Address{Role: protocol.RoleEdge, Station: "core-test"}
	dst := protocol.Address{Role: protocol.RoleCore, Station: cfg.Messaging.StationID}

	receiptReq := &protocol.OrderReceipt{
		OrderUUID:   req.OrderUUID,
		ReceiptType: req.ReceiptType,
		FinalCount:  req.FinalCount,
	}

	env, err := protocol.NewEnvelope(protocol.TypeOrderReceipt, src, dst, receiptReq)
	if err != nil {
		h.jsonError(w, "build envelope: "+err.Error(), http.StatusInternalServerError)
		return
	}

	data, err := env.Encode()
	if err != nil {
		h.jsonError(w, "encode envelope: "+err.Error(), http.StatusInternalServerError)
		return
	}

	topic := cfg.Messaging.OrdersTopic
	log.Printf("test-orders: publishing %s to %s: %s", env.Type, topic, string(data))

	if err := h.engine.MsgClient().Publish(topic, data); err != nil {
		h.jsonError(w, "publish failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.jsonOK(w, map[string]string{"status": "receipt sent", "order_uuid": req.OrderUUID})
}

func (h *Handlers) apiTestOrdersList(w http.ResponseWriter, r *http.Request) {
	orders, err := h.engine.DB().ListOrdersByStation("core-test", 50)
	if err != nil {
		h.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.jsonOK(w, orders)
}

func (h *Handlers) apiTestOrderDetail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if err != nil {
		h.jsonError(w, "invalid id", http.StatusBadRequest)
		return
	}
	order, err := h.engine.DB().GetOrder(id)
	if err != nil {
		h.jsonError(w, "order not found", http.StatusNotFound)
		return
	}
	history, _ := h.engine.DB().ListOrderHistory(id)
	h.jsonOK(w, map[string]any{"order": order, "history": history})
}

// --- Section B: Direct-to-RDS Order APIs ---

func (h *Handlers) apiDirectOrderSubmit(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FromNodeID int64 `json:"from_node_id"`
		ToNodeID   int64 `json:"to_node_id"`
		Priority   int   `json:"priority"`
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
		StationID:    "core-direct",
		OrderType:    "move",
		Status:       "pending",
		PickupNode:   sourceNode.Name,
		DeliveryNode: destNode.Name,
		Priority:     req.Priority,
		PayloadDesc:  "direct test order from shingo core",
	}
	if err := h.engine.DB().CreateOrder(order); err != nil {
		h.jsonError(w, "failed to create order: "+err.Error(), http.StatusInternalServerError)
		return
	}
	h.engine.DB().UpdateOrderStatus(order.ID, "pending", "direct test order created")

	vendorOrderID := fmt.Sprintf("sg-%d-%s", order.ID, uuid.New().String()[:8])
	fleetReq := fleet.TransportOrderRequest{
		OrderID:    vendorOrderID,
		ExternalID: edgeUUID,
		FromLoc:    sourceNode.VendorLocation,
		ToLoc:      destNode.VendorLocation,
		Priority:   req.Priority,
	}

	log.Printf("test-orders: direct fleet request: %+v", fleetReq)

	if _, err := h.engine.Fleet().CreateTransportOrder(fleetReq); err != nil {
		log.Printf("test-orders: direct fleet error: %v", err)
		h.engine.DB().UpdateOrderStatus(order.ID, "failed", err.Error())
		h.jsonError(w, "fleet dispatch failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("test-orders: direct fleet dispatched: order=%d vendor=%s", order.ID, vendorOrderID)

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

func (h *Handlers) apiDirectOrdersList(w http.ResponseWriter, r *http.Request) {
	orders, err := h.engine.DB().ListOrdersByStation("core-direct", 50)
	if err != nil {
		h.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.jsonOK(w, orders)
}

// --- Section C: Direct RDS Robot Command APIs ---

func (h *Handlers) apiTestCommandSubmit(w http.ResponseWriter, r *http.Request) {
	adapter, ok := h.engine.Fleet().(*seerrds.Adapter)
	if !ok {
		h.jsonError(w, "fleet backend does not support direct RDS commands", http.StatusNotImplemented)
		return
	}

	var req struct {
		CommandType string `json:"command_type"`
		RobotID     string `json:"robot_id"`
		Location    string `json:"location"`
		ConfigID    string `json:"config_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.CommandType == "" || req.RobotID == "" {
		h.jsonError(w, "command_type and robot_id are required", http.StatusBadRequest)
		return
	}

	orderID := "tc-" + uuid.New().String()[:8]
	blockID := orderID + "-b1"

	block := rds.Block{
		BlockID:  blockID,
		Location: req.Location,
	}
	if req.CommandType == "jack" || req.CommandType == "unjack" {
		if req.ConfigID == "" {
			h.jsonError(w, "config_id is required for jack/unjack commands", http.StatusBadRequest)
			return
		}
		block.PostAction = &rds.PostAction{ConfigID: req.ConfigID}
	}

	rdsReq := &rds.SetOrderRequest{
		ID:       orderID,
		Vehicle:  req.RobotID,
		Blocks:   []rds.Block{block},
		Complete: true,
	}

	log.Printf("test-commands: submitting %s to RDS: robot=%s loc=%s order=%s", req.CommandType, req.RobotID, req.Location, orderID)

	if err := adapter.RDSClient().CreateOrder(rdsReq); err != nil {
		log.Printf("test-commands: RDS error: %v", err)
		h.jsonError(w, "RDS command failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("test-commands: RDS order created: %s", orderID)

	tc := &store.TestCommand{
		CommandType:   req.CommandType,
		RobotID:       req.RobotID,
		VendorOrderID: orderID,
		VendorState:   "CREATED",
		Location:      req.Location,
		ConfigID:      req.ConfigID,
	}
	if err := h.engine.DB().CreateTestCommand(tc); err != nil {
		log.Printf("test-commands: db save error: %v", err)
	}

	h.jsonOK(w, map[string]any{
		"id":              tc.ID,
		"vendor_order_id": orderID,
	})
}

func (h *Handlers) apiTestCommandStatus(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if err != nil {
		h.jsonError(w, "invalid id", http.StatusBadRequest)
		return
	}

	tc, err := h.engine.DB().GetTestCommand(id)
	if err != nil {
		h.jsonError(w, "command not found", http.StatusNotFound)
		return
	}

	var rdsDetail *rds.OrderDetail
	if tc.CompletedAt == nil {
		adapter, ok := h.engine.Fleet().(*seerrds.Adapter)
		if ok && tc.VendorOrderID != "" {
			detail, err := adapter.RDSClient().GetOrderDetails(tc.VendorOrderID)
			if err == nil {
				rdsDetail = detail
				newState := string(detail.State)
				if newState != tc.VendorState {
					h.engine.DB().UpdateTestCommandStatus(id, newState, "")
					tc.VendorState = newState
				}
				if detail.State.IsTerminal() {
					h.engine.DB().CompleteTestCommand(id)
				}
			}
		}
	}

	h.jsonOK(w, map[string]any{
		"command":    tc,
		"rds_detail": rdsDetail,
	})
}

func (h *Handlers) apiTestCommandsList(w http.ResponseWriter, r *http.Request) {
	cmds, err := h.engine.DB().ListTestCommands(50)
	if err != nil {
		h.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.jsonOK(w, cmds)
}

// --- Shared Helper APIs ---

func (h *Handlers) apiTestRobots(w http.ResponseWriter, r *http.Request) {
	rl, ok := h.engine.Fleet().(fleet.RobotLister)
	if !ok {
		h.jsonError(w, "fleet backend does not support robot listing", http.StatusNotImplemented)
		return
	}
	robots, err := rl.GetRobotsStatus()
	if err != nil {
		h.jsonError(w, "fleet error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	h.jsonOK(w, robots)
}

func (h *Handlers) apiTestScenePoints(w http.ResponseWriter, r *http.Request) {
	points, err := h.engine.DB().ListScenePoints()
	if err != nil {
		h.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.jsonOK(w, points)
}
