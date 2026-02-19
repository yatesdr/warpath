package store

import "time"

// OutboxMessage is a queued outbound message.
type OutboxMessage struct {
	ID        int64      `json:"id"`
	Payload   []byte     `json:"payload"`
	MsgType   string     `json:"msg_type"`
	Retries   int        `json:"retries"`
	CreatedAt time.Time  `json:"created_at"`
	SentAt    *time.Time `json:"sent_at"`
}

func (db *DB) EnqueueOutbox(payload []byte, msgType string) (int64, error) {
	res, err := db.Exec(`INSERT INTO outbox (topic, payload, msg_type) VALUES ('orders', ?, ?)`, payload, msgType)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (db *DB) ListPendingOutbox(limit int) ([]OutboxMessage, error) {
	rows, err := db.Query(`SELECT id, payload, msg_type, retries, created_at FROM outbox WHERE sent_at IS NULL ORDER BY id LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var msgs []OutboxMessage
	for rows.Next() {
		var m OutboxMessage
		var createdAt string
		if err := rows.Scan(&m.ID, &m.Payload, &m.MsgType, &m.Retries, &createdAt); err != nil {
			return nil, err
		}
		m.CreatedAt = scanTime(createdAt)
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

func (db *DB) AckOutbox(id int64) error {
	_, err := db.Exec(`UPDATE outbox SET sent_at = datetime('now','localtime') WHERE id = ?`, id)
	return err
}

func (db *DB) IncrementOutboxRetries(id int64) error {
	_, err := db.Exec(`UPDATE outbox SET retries = retries + 1 WHERE id = ?`, id)
	return err
}
