package store

import (
	"database/sql"
	"fmt"
	"time"
)

type Bin struct {
	ID          int64     `json:"id"`
	BinTypeID   int64     `json:"bin_type_id"`
	Label       string    `json:"label"`
	Description string    `json:"description"`
	NodeID      *int64    `json:"node_id,omitempty"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	// Joined fields
	BinTypeCode string `json:"bin_type_code"`
	NodeName    string `json:"node_name"`
}

const binJoinQuery = `SELECT b.id, b.bin_type_id, b.label, b.description, b.node_id, b.status, b.created_at, b.updated_at,
	bt.code, COALESCE(n.name, '')
	FROM bins b
	JOIN bin_types bt ON bt.id = b.bin_type_id
	LEFT JOIN nodes n ON n.id = b.node_id`

func scanBin(row interface{ Scan(...any) error }) (*Bin, error) {
	var b Bin
	var nodeID sql.NullInt64
	var createdAt, updatedAt any
	err := row.Scan(&b.ID, &b.BinTypeID, &b.Label, &b.Description, &nodeID, &b.Status,
		&createdAt, &updatedAt, &b.BinTypeCode, &b.NodeName)
	if err != nil {
		return nil, err
	}
	if nodeID.Valid {
		b.NodeID = &nodeID.Int64
	}
	b.CreatedAt = parseTime(createdAt)
	b.UpdatedAt = parseTime(updatedAt)
	return &b, nil
}

func scanBins(rows *sql.Rows) ([]*Bin, error) {
	var bins []*Bin
	for rows.Next() {
		b, err := scanBin(rows)
		if err != nil {
			return nil, err
		}
		bins = append(bins, b)
	}
	return bins, rows.Err()
}

func (db *DB) CreateBin(b *Bin) error {
	result, err := db.Exec(db.Q(`INSERT INTO bins (bin_type_id, label, description, node_id, status) VALUES (?, ?, ?, ?, ?)`),
		b.BinTypeID, b.Label, b.Description, nullableInt64(b.NodeID), b.Status)
	if err != nil {
		return fmt.Errorf("create bin: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("create bin last id: %w", err)
	}
	b.ID = id
	return nil
}

func (db *DB) UpdateBin(b *Bin) error {
	_, err := db.Exec(db.Q(`UPDATE bins SET bin_type_id=?, label=?, description=?, node_id=?, status=?, updated_at=datetime('now','localtime') WHERE id=?`),
		b.BinTypeID, b.Label, b.Description, nullableInt64(b.NodeID), b.Status, b.ID)
	return err
}

func (db *DB) DeleteBin(id int64) error {
	_, err := db.Exec(db.Q(`DELETE FROM bins WHERE id=?`), id)
	return err
}

func (db *DB) GetBin(id int64) (*Bin, error) {
	row := db.QueryRow(db.Q(fmt.Sprintf(`%s WHERE b.id=?`, binJoinQuery)), id)
	return scanBin(row)
}

func (db *DB) GetBinByLabel(label string) (*Bin, error) {
	row := db.QueryRow(db.Q(fmt.Sprintf(`%s WHERE b.label=?`, binJoinQuery)), label)
	return scanBin(row)
}

func (db *DB) ListBins() ([]*Bin, error) {
	rows, err := db.Query(db.Q(fmt.Sprintf(`%s ORDER BY b.id DESC`, binJoinQuery)))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBins(rows)
}

func (db *DB) ListBinsByNode(nodeID int64) ([]*Bin, error) {
	rows, err := db.Query(db.Q(fmt.Sprintf(`%s WHERE b.node_id=? ORDER BY b.id DESC`, binJoinQuery)), nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBins(rows)
}

func (db *DB) CountBinsByNode(nodeID int64) (int, error) {
	var count int
	err := db.QueryRow(db.Q(`SELECT COUNT(*) FROM bins WHERE node_id=?`), nodeID).Scan(&count)
	return count, err
}

// CountBinsByAllNodes returns a map of node_id -> bin count for all nodes that have bins.
func (db *DB) CountBinsByAllNodes() (map[int64]int, error) {
	rows, err := db.Query(`SELECT node_id, COUNT(*) FROM bins WHERE node_id IS NOT NULL GROUP BY node_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := make(map[int64]int)
	for rows.Next() {
		var nodeID int64
		var count int
		if err := rows.Scan(&nodeID, &count); err != nil {
			return nil, err
		}
		counts[nodeID] = count
	}
	return counts, rows.Err()
}

// MoveBin moves a bin to a new node.
func (db *DB) MoveBin(binID, toNodeID int64) error {
	_, err := db.Exec(db.Q(`UPDATE bins SET node_id=?, updated_at=datetime('now','localtime') WHERE id=?`), toNodeID, binID)
	return err
}

// ListAvailableBins returns bins not currently assigned to a payload (available for new payloads).
func (db *DB) ListAvailableBins() ([]*Bin, error) {
	rows, err := db.Query(db.Q(fmt.Sprintf(`%s WHERE b.id NOT IN (SELECT bin_id FROM payloads WHERE bin_id IS NOT NULL) ORDER BY b.id`, binJoinQuery)))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBins(rows)
}

// ListBinsByType returns all bins of a given bin type.
func (db *DB) ListBinsByType(binTypeID int64) ([]*Bin, error) {
	rows, err := db.Query(db.Q(fmt.Sprintf(`%s WHERE b.bin_type_id=? ORDER BY b.label`, binJoinQuery)), binTypeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBins(rows)
}
