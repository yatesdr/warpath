package store

import "time"

// Payload event type constants.
const (
	PayloadEventCreated     = "created"
	PayloadEventMoved       = "moved"
	PayloadEventClaimed     = "claimed"
	PayloadEventUnclaimed   = "unclaimed"
	PayloadEventLoaded      = "loaded"
	PayloadEventDepleted    = "depleted"
	PayloadEventFlagged     = "flagged"
	PayloadEventMaintenance = "maintenance"
	PayloadEventRetired     = "retired"
	PayloadEventTagScanned  = "tag_scanned"
	PayloadEventTagMismatch = "tag_mismatch"
)

type PayloadEvent struct {
	ID        int64     `json:"id"`
	PayloadID int64     `json:"payload_id"`
	EventType string    `json:"event_type"`
	Detail    string    `json:"detail"`
	Actor     string    `json:"actor"`
	CreatedAt time.Time `json:"created_at"`
}

func (db *DB) CreatePayloadEvent(evt *PayloadEvent) error {
	_, err := db.Exec(db.Q(`INSERT INTO payload_events (payload_id, event_type, detail, actor) VALUES (?, ?, ?, ?)`),
		evt.PayloadID, evt.EventType, evt.Detail, evt.Actor)
	return err
}

// logPayloadEvent is a fire-and-forget helper for lifecycle event logging.
func (db *DB) logPayloadEvent(payloadID int64, eventType, detail string) {
	_ = db.CreatePayloadEvent(&PayloadEvent{
		PayloadID: payloadID,
		EventType: eventType,
		Detail:    detail,
		Actor:     "system",
	})
}

func (db *DB) ListPayloadEvents(payloadID int64, limit int) ([]*PayloadEvent, error) {
	rows, err := db.Query(db.Q(`SELECT id, payload_id, event_type, detail, actor, created_at FROM payload_events WHERE payload_id = ? ORDER BY created_at DESC LIMIT ?`), payloadID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*PayloadEvent
	for rows.Next() {
		e := &PayloadEvent{}
		var createdAt any
		if err := rows.Scan(&e.ID, &e.PayloadID, &e.EventType, &e.Detail, &e.Actor, &createdAt); err != nil {
			return nil, err
		}
		e.CreatedAt = parseTime(createdAt)
		events = append(events, e)
	}
	return events, rows.Err()
}
