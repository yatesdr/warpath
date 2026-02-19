package changeover

// EventEmitter is the interface the changeover package uses to emit events.
type EventEmitter interface {
	EmitChangeoverStarted(lineID int64, fromJobStyle, toJobStyle string)
	EmitChangeoverStateChanged(lineID int64, fromJobStyle, toJobStyle, oldState, newState string)
	EmitChangeoverCompleted(lineID int64, fromJobStyle, toJobStyle string)
}
