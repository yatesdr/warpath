package engine

// plcEmitter adapts the engine's EventBus to the plc.EventEmitter interface.
type plcEmitter struct {
	bus *EventBus
}

func (e *plcEmitter) EmitCounterRead(rpID int64, plcName, tagName string, value int64) {
	e.bus.Emit(Event{Type: EventCounterRead, Payload: CounterReadEvent{
		ReportingPointID: rpID, PLCName: plcName, TagName: tagName, Value: value,
	}})
}

func (e *plcEmitter) EmitCounterDelta(rpID, lineID, jobStyleID, delta, newCount int64) {
	e.bus.Emit(Event{Type: EventCounterDelta, Payload: CounterDeltaEvent{
		ReportingPointID: rpID, LineID: lineID, JobStyleID: jobStyleID, Delta: delta, NewCount: newCount,
	}})
}

func (e *plcEmitter) EmitCounterAnomaly(snapshotID, rpID int64, plcName, tagName string, oldVal, newVal int64, anomalyType string) {
	e.bus.Emit(Event{Type: EventCounterAnomaly, Payload: CounterAnomalyEvent{
		SnapshotID: snapshotID, ReportingPointID: rpID,
		PLCName: plcName, TagName: tagName,
		OldValue: oldVal, NewValue: newVal, AnomalyType: anomalyType,
	}})
}

func (e *plcEmitter) EmitPLCConnected(plcName string) {
	e.bus.Emit(Event{Type: EventPLCConnected, Payload: PLCEvent{PLCName: plcName}})
}

func (e *plcEmitter) EmitPLCDisconnected(plcName string, err error) {
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	e.bus.Emit(Event{Type: EventPLCDisconnected, Payload: PLCEvent{PLCName: plcName, Error: errStr}})
}

func (e *plcEmitter) EmitPLCHealthAlert(plcName string, errMsg string) {
	e.bus.Emit(Event{Type: EventPLCHealthAlert, Payload: PLCHealthAlertEvent{PLCName: plcName, Error: errMsg}})
}

func (e *plcEmitter) EmitPLCHealthRecover(plcName string) {
	e.bus.Emit(Event{Type: EventPLCHealthRecover, Payload: PLCHealthRecoverEvent{PLCName: plcName}})
}

func (e *plcEmitter) EmitWarLinkConnected() {
	e.bus.Emit(Event{Type: EventWarLinkConnected, Payload: WarLinkEvent{Connected: true}})
}

func (e *plcEmitter) EmitWarLinkDisconnected(err error) {
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	e.bus.Emit(Event{Type: EventWarLinkDisconnected, Payload: WarLinkEvent{Connected: false, Error: errStr}})
}

// orderEmitter adapts the engine's EventBus to the orders.EventEmitter interface.
type orderEmitter struct {
	bus *EventBus
}

func (e *orderEmitter) EmitOrderCreated(orderID int64, orderUUID, orderType string) {
	e.bus.Emit(Event{Type: EventOrderCreated, Payload: OrderCreatedEvent{
		OrderID: orderID, OrderUUID: orderUUID, OrderType: orderType,
	}})
}

func (e *orderEmitter) EmitOrderStatusChanged(orderID int64, orderUUID, orderType, oldStatus, newStatus, eta string) {
	e.bus.Emit(Event{Type: EventOrderStatusChanged, Payload: OrderStatusChangedEvent{
		OrderID: orderID, OrderUUID: orderUUID, OrderType: orderType, OldStatus: oldStatus, NewStatus: newStatus, ETA: eta,
	}})
}

func (e *orderEmitter) EmitOrderCompleted(orderID int64, orderUUID, orderType string) {
	e.bus.Emit(Event{Type: EventOrderCompleted, Payload: OrderCompletedEvent{
		OrderID: orderID, OrderUUID: orderUUID, OrderType: orderType,
	}})
}

// changeoverEmitter adapts the engine's EventBus to the changeover.EventEmitter interface.
type changeoverEmitter struct {
	bus *EventBus
}

func (e *changeoverEmitter) EmitChangeoverStarted(lineID int64, fromJobStyle, toJobStyle string) {
	e.bus.Emit(Event{Type: EventChangeoverStarted, Payload: ChangeoverStartedEvent{
		LineID: lineID, FromJobStyle: fromJobStyle, ToJobStyle: toJobStyle,
	}})
}

func (e *changeoverEmitter) EmitChangeoverStateChanged(lineID int64, fromJobStyle, toJobStyle, oldState, newState string) {
	e.bus.Emit(Event{Type: EventChangeoverStateChanged, Payload: ChangeoverStateChangedEvent{
		LineID: lineID, FromJobStyle: fromJobStyle, ToJobStyle: toJobStyle, OldState: oldState, NewState: newState,
	}})
}

func (e *changeoverEmitter) EmitChangeoverCompleted(lineID int64, fromJobStyle, toJobStyle string) {
	e.bus.Emit(Event{Type: EventChangeoverCompleted, Payload: ChangeoverCompletedEvent{
		LineID: lineID, FromJobStyle: fromJobStyle, ToJobStyle: toJobStyle,
	}})
}
