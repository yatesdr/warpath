package engine

import (
	"fmt"
	"time"

	"shingocore/dispatch"
	"shingocore/messaging"
	"shingocore/store"
)

func (e *Engine) wireEventHandlers() {
	// When an order is dispatched, track it in the tracker
	e.Events.SubscribeTypes(func(evt Event) {
		ev := evt.Payload.(OrderDispatchedEvent)
		if e.tracker == nil {
			return
		}
		// On redirect, the order may already have an old vendor order ID tracked.
		// Look up the order and untrack the old ID if it differs from the new one.
		if order, err := e.db.GetOrder(ev.OrderID); err == nil && order.VendorOrderID != "" && order.VendorOrderID != ev.VendorOrderID {
			e.tracker.Untrack(order.VendorOrderID)
			e.logFn("engine: untracked old vendor order %s for order %d (redirect)", order.VendorOrderID, ev.OrderID)
		}
		e.tracker.Track(ev.VendorOrderID)
		e.logFn("engine: tracking vendor order %s for order %d", ev.VendorOrderID, ev.OrderID)
	}, EventOrderDispatched)

	// When the fleet reports a status change, update our order and notify ShinGo Edge
	e.Events.SubscribeTypes(func(evt Event) {
		ev := evt.Payload.(OrderStatusChangedEvent)
		e.handleVendorStatusChange(ev)
	}, EventOrderStatusChanged)

	// When an order fails, log it
	e.Events.SubscribeTypes(func(evt Event) {
		ev := evt.Payload.(OrderFailedEvent)
		e.logFn("engine: order %d failed: %s - %s", ev.OrderID, ev.ErrorCode, ev.Detail)
		e.db.AppendAudit("order", ev.OrderID, "failed", "", ev.Detail, "system")
	}, EventOrderFailed)

	// When an order is completed, update inventory and audit
	e.Events.SubscribeTypes(func(evt Event) {
		ev := evt.Payload.(OrderCompletedEvent)
		e.logFn("engine: order %d completed", ev.OrderID)
		e.db.AppendAudit("order", ev.OrderID, "completed", "", "", "system")
		e.handleOrderCompleted(ev)
	}, EventOrderCompleted)

	// When an order is cancelled, audit it
	e.Events.SubscribeTypes(func(evt Event) {
		ev := evt.Payload.(OrderCancelledEvent)
		e.logFn("engine: order %d cancelled: %s", ev.OrderID, ev.Reason)
		e.db.AppendAudit("order", ev.OrderID, "cancelled", "", ev.Reason, "system")
	}, EventOrderCancelled)

	// When an order is received, audit it
	e.Events.SubscribeTypes(func(evt Event) {
		ev := evt.Payload.(OrderReceivedEvent)
		e.logFn("engine: order %d received from %s: %s %s -> %s", ev.OrderID, ev.ClientID, ev.OrderType, ev.PayloadTypeCode, ev.DeliveryNode)
		e.db.AppendAudit("order", ev.OrderID, "received", "", fmt.Sprintf("%s %s from %s", ev.OrderType, ev.PayloadTypeCode, ev.ClientID), "system")
	}, EventOrderReceived)

	// Payload changes: audit
	e.Events.SubscribeTypes(func(evt Event) {
		ev := evt.Payload.(PayloadChangedEvent)
		e.db.AppendAudit("payload", ev.PayloadID, ev.Action, "", fmt.Sprintf("type=%s node=%d", ev.PayloadTypeCode, ev.NodeID), "system")
	}, EventPayloadChanged)

	// Node updates: audit
	e.Events.SubscribeTypes(func(evt Event) {
		ev := evt.Payload.(NodeUpdatedEvent)
		e.db.AppendAudit("node", ev.NodeID, ev.Action, "", ev.NodeName, "system")
	}, EventNodeUpdated)

	// Corrections: audit
	e.Events.SubscribeTypes(func(evt Event) {
		ev := evt.Payload.(CorrectionAppliedEvent)
		e.db.AppendAudit("correction", ev.CorrectionID, ev.CorrectionType, "", ev.Reason, ev.Actor)
	}, EventCorrectionApplied)
}

func (e *Engine) handleVendorStatusChange(ev OrderStatusChangedEvent) {
	order, err := e.db.GetOrder(ev.OrderID)
	if err != nil {
		e.logFn("engine: get order %d for status change: %v", ev.OrderID, err)
		return
	}

	// Update robot ID if we got one
	if ev.RobotID != "" && order.RobotID == "" {
		e.db.UpdateOrderVendor(order.ID, order.VendorOrderID, ev.NewStatus, ev.RobotID)

		// Send waybill to ShinGo Edge
		reply := messaging.NewEnvelope("waybill", order.ClientID, e.cfg.FactoryID, messaging.WaybillReply{
			OrderUUID: order.EdgeUUID,
			WaybillID: order.VendorOrderID,
			RobotID:   ev.RobotID,
		})
		data, _ := reply.Encode()
		topic := messaging.DispatchTopic(e.cfg.Messaging.DispatchTopicPrefix, order.ClientID)
		e.db.EnqueueOutbox(topic, data, "waybill", order.ClientID)
	}

	newStatus := e.fleet.MapState(ev.NewStatus)
	if newStatus == order.Status {
		return
	}

	e.db.UpdateOrderStatus(order.ID, newStatus, fmt.Sprintf("fleet: %s -> %s", ev.OldStatus, ev.NewStatus))
	e.db.UpdateOrderVendor(order.ID, order.VendorOrderID, ev.NewStatus, ev.RobotID)

	// Send status update to ShinGo Edge
	reply := messaging.NewEnvelope("update", order.ClientID, e.cfg.FactoryID, messaging.UpdateReply{
		OrderUUID: order.EdgeUUID,
		Status:    newStatus,
		Detail:    fmt.Sprintf("fleet state: %s", ev.NewStatus),
	})
	data, _ := reply.Encode()
	topic := messaging.DispatchTopic(e.cfg.Messaging.DispatchTopicPrefix, order.ClientID)
	e.db.EnqueueOutbox(topic, data, "update", order.ClientID)

	// Handle terminal states
	if e.fleet.IsTerminalState(ev.NewStatus) {
		switch newStatus {
		case dispatch.StatusDelivered:
			e.handleOrderDelivered(order)
		case dispatch.StatusFailed:
			e.db.UpdateOrderStatus(order.ID, dispatch.StatusFailed, "fleet order failed")
			e.Events.Emit(Event{Type: EventOrderFailed, Payload: OrderFailedEvent{
				OrderID:   order.ID,
				EdgeUUID:  order.EdgeUUID,
				ClientID:  order.ClientID,
				ErrorCode: "fleet_failed",
				Detail:    "fleet order failed",
			}})
		case dispatch.StatusCancelled:
			e.db.UpdateOrderStatus(order.ID, dispatch.StatusCancelled, "fleet order stopped")
		}
	}
}

func (e *Engine) handleOrderDelivered(order *store.Order) {
	e.db.UpdateOrderStatus(order.ID, dispatch.StatusDelivered, "payload delivered")

	// Send delivered notification to ShinGo Edge
	reply := messaging.NewEnvelope("delivered", order.ClientID, e.cfg.FactoryID, messaging.DeliveredReply{
		OrderUUID:   order.EdgeUUID,
		DeliveredAt: time.Now().Format(time.RFC3339),
	})
	data, _ := reply.Encode()
	topic := messaging.DispatchTopic(e.cfg.Messaging.DispatchTopicPrefix, order.ClientID)
	e.db.EnqueueOutbox(topic, data, "delivered", order.ClientID)
}

// handleOrderCompleted moves payloads from source to dest after ShinGo Edge confirms physical receipt.
func (e *Engine) handleOrderCompleted(ev OrderCompletedEvent) {
	order, err := e.db.GetOrder(ev.OrderID)
	if err != nil {
		e.logFn("engine: get order %d for completion: %v", ev.OrderID, err)
		return
	}

	if order.SourceNodeID == nil || order.DestNodeID == nil {
		return
	}

	payloads, _ := e.db.ListPayloadsByClaimedOrder(order.ID)
	for _, p := range payloads {
		e.nodeState.MovePayload(p.ID, *order.DestNodeID)
		e.Events.Emit(Event{Type: EventPayloadChanged, Payload: PayloadChangedEvent{
			Action:          "moved",
			PayloadID:       p.ID,
			PayloadTypeCode: p.PayloadTypeName,
			FromNodeID:      *order.SourceNodeID,
			ToNodeID:        *order.DestNodeID,
			NodeID:          *order.DestNodeID,
		}})
	}
}
