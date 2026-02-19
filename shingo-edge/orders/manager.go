package orders

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"shingoedge/store"

	"github.com/google/uuid"
)

// Manager handles the order lifecycle state machine.
type Manager struct {
	db        *store.DB
	emitter   EventEmitter
	namespace string
	lineID    string
}

// NewManager creates an order manager.
func NewManager(db *store.DB, emitter EventEmitter, namespace, lineID string) *Manager {
	return &Manager{
		db:        db,
		emitter:   emitter,
		namespace: namespace,
		lineID:    lineID,
	}
}

// CreateRetrieveOrder creates a new retrieve order and enqueues it to the outbox.
func (m *Manager) CreateRetrieveOrder(payloadID *int64, retrieveEmpty bool, quantity float64, deliveryNode, stagingNode, loadType string, templateID *int64, autoConfirm bool) (*store.Order, error) {
	orderUUID := uuid.New().String()

	orderID, err := m.db.CreateOrder(orderUUID, TypeRetrieve,
		payloadID, retrieveEmpty,
		quantity, deliveryNode, stagingNode, "", loadType, templateID, autoConfirm)
	if err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	// Look up payload description for message
	var payloadDesc string
	if payloadID != nil {
		if p, err := m.db.GetPayload(*payloadID); err == nil {
			payloadDesc = p.Description
		}
	}

	// Build outbound message
	msg := OrderMessage{
		Namespace:     m.namespace,
		LineID:        m.lineID,
		OrderUUID:     orderUUID,
		OrderType:     TypeRetrieve,
		PayloadDesc:   payloadDesc,
		RetrieveEmpty: retrieveEmpty,
		Quantity:      quantity,
		DeliveryNode:  deliveryNode,
		StagingNode:   stagingNode,
		LoadType:      loadType,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
	}
	payload, _ := json.Marshal(msg)

	if _, err := m.db.EnqueueOutbox("orders", payload, "order_request"); err != nil {
		log.Printf("enqueue order %s: %v", orderUUID, err)
	}

	m.emitter.EmitOrderCreated(orderID, orderUUID, TypeRetrieve)

	return m.db.GetOrder(orderID)
}

// CreateStoreOrder creates a new store order (for returning payloads to warehouse).
func (m *Manager) CreateStoreOrder(payloadID *int64, quantity float64, pickupNode string, templateID *int64) (*store.Order, error) {
	orderUUID := uuid.New().String()

	orderID, err := m.db.CreateOrder(orderUUID, TypeStore,
		payloadID, false,
		quantity, "", "", pickupNode, "", templateID, false)
	if err != nil {
		return nil, fmt.Errorf("create store order: %w", err)
	}

	m.emitter.EmitOrderCreated(orderID, orderUUID, TypeStore)
	return m.db.GetOrder(orderID)
}

// CreateMoveOrder creates a new move order (e.g., quality hold).
func (m *Manager) CreateMoveOrder(payloadID *int64, quantity float64, pickupNode, deliveryNode string, templateID *int64) (*store.Order, error) {
	orderUUID := uuid.New().String()

	orderID, err := m.db.CreateOrder(orderUUID, TypeMove,
		payloadID, false,
		quantity, deliveryNode, "", pickupNode, "", templateID, false)
	if err != nil {
		return nil, fmt.Errorf("create move order: %w", err)
	}

	// Look up payload description for message
	var payloadDesc string
	if payloadID != nil {
		if p, err := m.db.GetPayload(*payloadID); err == nil {
			payloadDesc = p.Description
		}
	}

	// Enqueue outbound message
	msg := OrderMessage{
		Namespace:    m.namespace,
		LineID:       m.lineID,
		OrderUUID:    orderUUID,
		OrderType:    TypeMove,
		PayloadDesc:  payloadDesc,
		Quantity:     quantity,
		DeliveryNode: deliveryNode,
		PickupNode:   pickupNode,
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
	}
	msgPayload, _ := json.Marshal(msg)
	if _, err := m.db.EnqueueOutbox("orders", msgPayload, "order_request"); err != nil {
		log.Printf("enqueue move order %s: %v", orderUUID, err)
	}

	m.emitter.EmitOrderCreated(orderID, orderUUID, TypeMove)
	return m.db.GetOrder(orderID)
}

// TransitionOrder moves an order to a new status with validation.
func (m *Manager) TransitionOrder(orderID int64, newStatus, detail string) error {
	order, err := m.db.GetOrder(orderID)
	if err != nil {
		return fmt.Errorf("get order: %w", err)
	}

	if !IsValidTransition(order.Status, newStatus) {
		return fmt.Errorf("invalid transition from %s to %s", order.Status, newStatus)
	}

	// Store orders require count confirmation before submitting
	if order.OrderType == TypeStore && newStatus == StatusSubmitted && !order.CountConfirmed {
		return fmt.Errorf("store order requires count confirmation before submitting")
	}

	oldStatus := order.Status
	if err := m.db.UpdateOrderStatus(orderID, newStatus); err != nil {
		return fmt.Errorf("update order status: %w", err)
	}
	if err := m.db.InsertOrderHistory(orderID, oldStatus, newStatus, detail); err != nil {
		log.Printf("insert order history: %v", err)
	}

	m.emitter.EmitOrderStatusChanged(orderID, order.UUID, order.OrderType, oldStatus, newStatus)

	if IsTerminal(newStatus) {
		m.emitter.EmitOrderCompleted(orderID, order.UUID, order.OrderType)
	}

	return nil
}

// AbortOrder cancels a non-terminal order and enqueues a cancel message.
func (m *Manager) AbortOrder(orderID int64) error {
	order, err := m.db.GetOrder(orderID)
	if err != nil {
		return fmt.Errorf("get order: %w", err)
	}
	if IsTerminal(order.Status) {
		return fmt.Errorf("order is already in terminal state: %s", order.Status)
	}

	if err := m.TransitionOrder(orderID, StatusCancelled, "aborted by operator"); err != nil {
		return err
	}

	msg := OrderMessage{
		Namespace: m.namespace,
		LineID:    m.lineID,
		OrderUUID: order.UUID,
		OrderType: order.OrderType,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	payload, _ := json.Marshal(msg)
	if _, err := m.db.EnqueueOutbox("orders", payload, "order_cancel"); err != nil {
		log.Printf("enqueue order cancel %s: %v", order.UUID, err)
	}

	return nil
}

// RedirectOrder changes the delivery node of a non-terminal order and enqueues a redirect message.
func (m *Manager) RedirectOrder(orderID int64, newDeliveryNode string) (*store.Order, error) {
	order, err := m.db.GetOrder(orderID)
	if err != nil {
		return nil, fmt.Errorf("get order: %w", err)
	}
	if IsTerminal(order.Status) {
		return nil, fmt.Errorf("order is already in terminal state: %s", order.Status)
	}

	if err := m.db.UpdateOrderDeliveryNode(orderID, newDeliveryNode); err != nil {
		return nil, fmt.Errorf("update delivery node: %w", err)
	}

	msg := OrderMessage{
		Namespace:    m.namespace,
		LineID:       m.lineID,
		OrderUUID:    order.UUID,
		OrderType:    order.OrderType,
		DeliveryNode: newDeliveryNode,
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
	}
	payload, _ := json.Marshal(msg)
	if _, err := m.db.EnqueueOutbox("orders", payload, "redirect_request"); err != nil {
		log.Printf("enqueue redirect %s: %v", order.UUID, err)
	}

	return m.db.GetOrder(orderID)
}

// SubmitOrder transitions a queued order to submitted and enqueues it.
func (m *Manager) SubmitOrder(orderID int64) error {
	order, err := m.db.GetOrder(orderID)
	if err != nil {
		return err
	}

	if err := m.TransitionOrder(orderID, StatusSubmitted, "submitted to dispatch"); err != nil {
		return err
	}

	// For store orders, build and enqueue the storage waybill
	if order.OrderType == TypeStore {
		msg := StorageWaybillMessage{
			Namespace:   m.namespace,
			LineID:      m.lineID,
			OrderUUID:   order.UUID,
			OrderType:   TypeStore,
			PayloadDesc: order.PayloadDesc,
			PickupNode:  order.PickupNode,
			FinalCount:  ptrFloat64(order.FinalCount),
			Timestamp:   time.Now().UTC().Format(time.RFC3339),
		}
		payload, _ := json.Marshal(msg)
		if _, err := m.db.EnqueueOutbox("orders", payload, "storage_waybill"); err != nil {
			log.Printf("enqueue storage waybill %s: %v", order.UUID, err)
		}
	}

	return nil
}

// ConfirmDelivery sends a delivery receipt and transitions to confirmed.
func (m *Manager) ConfirmDelivery(orderID int64, finalCount float64) error {
	order, err := m.db.GetOrder(orderID)
	if err != nil {
		return err
	}

	if order.Status != StatusDelivered {
		return fmt.Errorf("order must be in delivered status to confirm, got %s", order.Status)
	}

	// Update final count
	if err := m.db.UpdateOrderFinalCount(orderID, finalCount, true); err != nil {
		return err
	}

	// Enqueue delivery receipt
	receipt := DeliveryReceiptMessage{
		Namespace:   m.namespace,
		LineID:      m.lineID,
		OrderUUID:   order.UUID,
		ReceiptType: "confirmed",
		FinalCount:  finalCount,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}
	payload, _ := json.Marshal(receipt)
	if _, err := m.db.EnqueueOutbox("orders", payload, "delivery_receipt"); err != nil {
		log.Printf("enqueue delivery receipt %s: %v", order.UUID, err)
	}

	return m.TransitionOrder(orderID, StatusConfirmed, fmt.Sprintf("confirmed with count %.0f", finalCount))
}

// HandleDispatchReply processes an inbound reply from central dispatch.
func (m *Manager) HandleDispatchReply(orderUUID, replyType, waybillID, eta, statusDetail string) error {
	order, err := m.db.GetOrderByUUID(orderUUID)
	if err != nil {
		return fmt.Errorf("order %s not found: %w", orderUUID, err)
	}

	switch replyType {
	case "ack":
		return m.TransitionOrder(order.ID, StatusAcknowledged, statusDetail)
	case "waybill":
		if err := m.db.UpdateOrderWaybill(order.ID, waybillID, eta); err != nil {
			return err
		}
		return m.TransitionOrder(order.ID, StatusInTransit, fmt.Sprintf("waybill %s, ETA %s", waybillID, eta))
	case "update":
		// Status update with ETA, no state change
		if eta != "" {
			if err := m.db.UpdateOrderWaybill(order.ID, waybillID, eta); err != nil {
				return err
			}
		}
		return nil
	case "delivered":
		if err := m.TransitionOrder(order.ID, StatusDelivered, statusDetail); err != nil {
			return err
		}
		// Auto-confirm if enabled
		if order.AutoConfirm {
			return m.ConfirmDelivery(order.ID, order.Quantity)
		}
		return nil
	default:
		return fmt.Errorf("unknown reply type: %s", replyType)
	}
}

// OrderMessage is the outbound order request JSON.
type OrderMessage struct {
	Namespace     string      `json:"namespace"`
	LineID        string      `json:"line_id"`
	OrderUUID     string      `json:"order_uuid"`
	OrderType     string      `json:"order_type"`
	PayloadDesc   string      `json:"payload_desc,omitempty"`
	RetrieveEmpty bool        `json:"retrieve_empty,omitempty"`
	Quantity      float64     `json:"quantity"`
	DeliveryNode  string      `json:"delivery_node,omitempty"`
	StagingNode   string      `json:"staging_node,omitempty"`
	PickupNode    string      `json:"pickup_node,omitempty"`
	LoadType      string      `json:"load_type,omitempty"`
	TemplateData  interface{} `json:"template_data,omitempty"`
	Timestamp     string      `json:"timestamp"`
}

// StorageWaybillMessage is the outbound storage waybill JSON.
type StorageWaybillMessage struct {
	Namespace   string  `json:"namespace"`
	LineID      string  `json:"line_id"`
	OrderUUID   string  `json:"order_uuid"`
	OrderType   string  `json:"order_type"`
	PayloadDesc string  `json:"payload_desc,omitempty"`
	PickupNode  string  `json:"pickup_node"`
	FinalCount  float64 `json:"final_count"`
	Timestamp   string  `json:"timestamp"`
}

// DeliveryReceiptMessage is the outbound delivery receipt JSON.
type DeliveryReceiptMessage struct {
	Namespace   string  `json:"namespace"`
	LineID      string  `json:"line_id"`
	OrderUUID   string  `json:"order_uuid"`
	ReceiptType string  `json:"receipt_type"`
	FinalCount  float64 `json:"final_count"`
	Timestamp   string  `json:"timestamp"`
}

func ptrFloat64(p *float64) float64 {
	if p != nil {
		return *p
	}
	return 0
}
