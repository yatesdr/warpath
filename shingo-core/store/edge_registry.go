package store

import (
	"encoding/json"
	"time"
)

// EdgeRegistration represents a registered edge node.
type EdgeRegistration struct {
	ID             int64     `json:"id"`
	NodeID         string    `json:"node_id"`
	FactoryID      string    `json:"factory_id"`
	Hostname       string    `json:"hostname"`
	Version        string    `json:"version"`
	LineIDs        []string  `json:"line_ids"`
	RegisteredAt   time.Time `json:"registered_at"`
	LastHeartbeat  *time.Time `json:"last_heartbeat"`
	Status         string    `json:"status"`
}

// RegisterEdge upserts an edge registration. If the node_id already exists,
// it updates the record and resets status to active.
func (db *DB) RegisterEdge(nodeID, factoryID, hostname, version string, lineIDs []string) error {
	lineJSON, _ := json.Marshal(lineIDs)

	_, err := db.Exec(db.Q(`
		INSERT INTO edge_registry (node_id, factory_id, hostname, version, line_ids, registered_at, status)
		VALUES (?, ?, ?, ?, ?, datetime('now','localtime'), 'active')
		ON CONFLICT(node_id) DO UPDATE SET
			factory_id = excluded.factory_id,
			hostname = excluded.hostname,
			version = excluded.version,
			line_ids = excluded.line_ids,
			registered_at = excluded.registered_at,
			status = 'active'
	`), nodeID, factoryID, hostname, version, string(lineJSON))
	return err
}

// UpdateHeartbeat updates the last_heartbeat timestamp and sets status to active.
func (db *DB) UpdateHeartbeat(nodeID string) error {
	_, err := db.Exec(db.Q(`
		UPDATE edge_registry
		SET last_heartbeat = datetime('now','localtime'), status = 'active'
		WHERE node_id = ?
	`), nodeID)
	return err
}

// ListEdges returns all registered edges.
func (db *DB) ListEdges() ([]EdgeRegistration, error) {
	rows, err := db.Query(db.Q(`
		SELECT id, node_id, factory_id, hostname, version, line_ids, registered_at, last_heartbeat, status
		FROM edge_registry ORDER BY node_id
	`))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edges []EdgeRegistration
	for rows.Next() {
		var e EdgeRegistration
		var lineJSON string
		var regAt, hbAt *string
		if err := rows.Scan(&e.ID, &e.NodeID, &e.FactoryID, &e.Hostname, &e.Version, &lineJSON, &regAt, &hbAt, &e.Status); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(lineJSON), &e.LineIDs)
		if regAt != nil {
			if t, err := time.Parse("2006-01-02 15:04:05", *regAt); err == nil {
				e.RegisteredAt = t
			}
		}
		if hbAt != nil {
			if t, err := time.Parse("2006-01-02 15:04:05", *hbAt); err == nil {
				e.LastHeartbeat = &t
			}
		}
		edges = append(edges, e)
	}
	return edges, rows.Err()
}

// MarkStaleEdges sets status to "stale" for edges whose last_heartbeat is older than the given threshold.
func (db *DB) MarkStaleEdges(threshold time.Duration) (int64, error) {
	cutoff := time.Now().Add(-threshold).Format("2006-01-02 15:04:05")
	result, err := db.Exec(db.Q(`
		UPDATE edge_registry
		SET status = 'stale'
		WHERE status = 'active'
		  AND last_heartbeat IS NOT NULL
		  AND last_heartbeat < ?
	`), cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
