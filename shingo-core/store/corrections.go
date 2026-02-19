package store

import (
	"database/sql"
	"time"
)

type Correction struct {
	ID             int64     `json:"id"`
	CorrectionType string    `json:"correction_type"`
	NodeID         int64     `json:"node_id"`
	MaterialID     *int64    `json:"material_id,omitempty"`
	InventoryID    *int64    `json:"inventory_id,omitempty"`
	PayloadID      *int64    `json:"payload_id,omitempty"`
	ManifestItemID *int64    `json:"manifest_item_id,omitempty"`
	CatID          string    `json:"cat_id"`
	Description    string    `json:"description"`
	Quantity       float64   `json:"quantity"`
	Reason         string    `json:"reason"`
	Actor          string    `json:"actor"`
	CreatedAt      time.Time `json:"created_at"`
}

func (db *DB) CreateCorrection(c *Correction) error {
	var matID, invID, plID, miID any
	if c.MaterialID != nil {
		matID = *c.MaterialID
	}
	if c.InventoryID != nil {
		invID = *c.InventoryID
	}
	if c.PayloadID != nil {
		plID = *c.PayloadID
	}
	if c.ManifestItemID != nil {
		miID = *c.ManifestItemID
	}
	result, err := db.Exec(db.Q(`INSERT INTO corrections (correction_type, node_id, material_id, inventory_id, payload_id, manifest_item_id, cat_id, description, quantity, reason, actor) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`),
		c.CorrectionType, c.NodeID, matID, invID, plID, miID, c.CatID, c.Description, c.Quantity, c.Reason, c.Actor)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	c.ID = id
	return nil
}

func (db *DB) ListCorrections(limit int) ([]*Correction, error) {
	rows, err := db.Query(db.Q(`SELECT id, correction_type, node_id, material_id, inventory_id, payload_id, manifest_item_id, cat_id, description, quantity, reason, actor, created_at FROM corrections ORDER BY id DESC LIMIT ?`), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var corrections []*Correction
	for rows.Next() {
		var c Correction
		var createdAt any
		var materialID, inventoryID, payloadID, manifestItemID sql.NullInt64
		if err := rows.Scan(&c.ID, &c.CorrectionType, &c.NodeID, &materialID, &inventoryID, &payloadID, &manifestItemID, &c.CatID, &c.Description, &c.Quantity, &c.Reason, &c.Actor, &createdAt); err != nil {
			return nil, err
		}
		if materialID.Valid {
			c.MaterialID = &materialID.Int64
		}
		if inventoryID.Valid {
			c.InventoryID = &inventoryID.Int64
		}
		if payloadID.Valid {
			c.PayloadID = &payloadID.Int64
		}
		if manifestItemID.Valid {
			c.ManifestItemID = &manifestItemID.Int64
		}
		c.CreatedAt = parseTime(createdAt)
		corrections = append(corrections, &c)
	}
	return corrections, rows.Err()
}

func (db *DB) ListCorrectionsByNode(nodeID int64, limit int) ([]*Correction, error) {
	rows, err := db.Query(db.Q(`SELECT id, correction_type, node_id, material_id, inventory_id, payload_id, manifest_item_id, cat_id, description, quantity, reason, actor, created_at FROM corrections WHERE node_id = ? ORDER BY id DESC LIMIT ?`), nodeID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var corrections []*Correction
	for rows.Next() {
		var c Correction
		var createdAt any
		var materialID, inventoryID, payloadID, manifestItemID sql.NullInt64
		if err := rows.Scan(&c.ID, &c.CorrectionType, &c.NodeID, &materialID, &inventoryID, &payloadID, &manifestItemID, &c.CatID, &c.Description, &c.Quantity, &c.Reason, &c.Actor, &createdAt); err != nil {
			return nil, err
		}
		if materialID.Valid {
			c.MaterialID = &materialID.Int64
		}
		if inventoryID.Valid {
			c.InventoryID = &inventoryID.Int64
		}
		if payloadID.Valid {
			c.PayloadID = &payloadID.Int64
		}
		if manifestItemID.Valid {
			c.ManifestItemID = &manifestItemID.Int64
		}
		c.CreatedAt = parseTime(createdAt)
		corrections = append(corrections, &c)
	}
	return corrections, rows.Err()
}
