package store

import (
	"encoding/json"
	"fmt"
	"time"
)

// ManifestEntry describes a single item in a bin's manifest JSON.
type ManifestEntry struct {
	CatID    string `json:"catid"`
	Quantity int64  `json:"qty"`
	LotCode  string `json:"lot_code,omitempty"`
	Notes    string `json:"notes,omitempty"`
}

// BinManifest is the parsed form of a bin's manifest JSON field.
type BinManifest struct {
	Items []ManifestEntry `json:"items"`
}

// SetBinManifest populates a bin's contents from a payload template.
func (db *DB) SetBinManifest(binID int64, manifestJSON string, payloadCode string, uopRemaining int) error {
	_, err := db.Exec(db.Q(`UPDATE bins SET payload_code=?, manifest=?, uop_remaining=?, manifest_confirmed=0, updated_at=datetime('now') WHERE id=?`),
		payloadCode, manifestJSON, uopRemaining, binID)
	return err
}

// ConfirmBinManifest marks a bin's manifest as confirmed by an operator.
func (db *DB) ConfirmBinManifest(binID int64) error {
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	_, err := db.Exec(db.Q(`UPDATE bins SET manifest_confirmed=1, loaded_at=?, updated_at=datetime('now') WHERE id=?`),
		now, binID)
	return err
}

// ClearBinManifest empties a bin's manifest (bin is now empty).
func (db *DB) ClearBinManifest(binID int64) error {
	_, err := db.Exec(db.Q(`UPDATE bins SET payload_code='', manifest=NULL, uop_remaining=0, manifest_confirmed=0, loaded_at=NULL, updated_at=datetime('now') WHERE id=?`),
		binID)
	return err
}

// ParseManifest parses a bin's manifest JSON into a BinManifest struct.
func (b *Bin) ParseManifest() (*BinManifest, error) {
	if b.Manifest == nil || *b.Manifest == "" {
		return &BinManifest{}, nil
	}
	var m BinManifest
	if err := json.Unmarshal([]byte(*b.Manifest), &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	return &m, nil
}

// GetBinManifest fetches a bin and parses its manifest.
func (db *DB) GetBinManifest(binID int64) (*BinManifest, error) {
	bin, err := db.GetBin(binID)
	if err != nil {
		return nil, err
	}
	return bin.ParseManifest()
}

// FindSourceBinFIFO finds the best unclaimed bin at an enabled storage node
// matching the given payload code, using FIFO ordering.
func (db *DB) FindSourceBinFIFO(payloadCode string) (*Bin, error) {
	row := db.QueryRow(db.Q(fmt.Sprintf(`%s
		WHERE b.payload_code = ?
		  AND n.enabled = 1
		  AND n.is_synthetic = 0
		  AND b.claimed_by IS NULL
		  AND b.locked = 0
		  AND b.manifest_confirmed = 1
		  AND b.status NOT IN ('staged', 'maintenance', 'flagged', 'retired', 'quality_hold')
		ORDER BY COALESCE(b.loaded_at, b.created_at) ASC
		LIMIT 1`, binJoinQuery)), payloadCode)
	return scanBin(row)
}

// FindStorageDestination finds the best storage node for a bin.
// Prefers nodes with existing bins of the same payload code, then empty nodes.
func (db *DB) FindStorageDestination(payloadCode string) (*Node, error) {
	// Try consolidation: storage nodes with bins of same payload code
	row := db.QueryRow(db.Q(fmt.Sprintf(`
		SELECT %s %s WHERE n.id = (
			SELECT sn.id
			FROM nodes sn
			JOIN bins match_b ON match_b.node_id = sn.id AND match_b.payload_code = ?
			LEFT JOIN bins total_b ON total_b.node_id = sn.id
			WHERE sn.enabled = 1 AND sn.is_synthetic = 0
			GROUP BY sn.id
			HAVING COUNT(DISTINCT total_b.id) < 1
			ORDER BY COUNT(DISTINCT match_b.id) DESC
			LIMIT 1
		)`, nodeSelectCols, nodeFromClause)), payloadCode)
	n, err := scanNode(row)
	if err == nil {
		return n, nil
	}

	// Fall back to emptiest storage node (no bins)
	row = db.QueryRow(db.Q(fmt.Sprintf(`
		SELECT %s %s WHERE n.id = (
			SELECT sn.id
			FROM nodes sn
			LEFT JOIN bins sb ON sb.node_id = sn.id
			WHERE sn.enabled = 1 AND sn.is_synthetic = 0
			GROUP BY sn.id
			HAVING COUNT(sb.id) < 1
			ORDER BY COUNT(sb.id) ASC
			LIMIT 1
		)`, nodeSelectCols, nodeFromClause)))
	return scanNode(row)
}

// DecrementBinUOP reduces the uop_remaining on a bin.
func (db *DB) DecrementBinUOP(binID int64, delta int) error {
	_, err := db.Exec(db.Q(`UPDATE bins SET uop_remaining = MAX(0, uop_remaining - ?), updated_at=datetime('now') WHERE id=?`),
		delta, binID)
	return err
}

// SetBinManifestFromTemplate sets a bin's manifest from a payload template's
// manifest items and marks it as confirmed.
func (db *DB) SetBinManifestFromTemplate(binID int64, payloadCode string, uopCapacity int) error {
	// Look up the payload template
	p, err := db.GetPayloadByCode(payloadCode)
	if err != nil {
		return fmt.Errorf("payload template %q: %w", payloadCode, err)
	}

	// Get the template manifest items
	items, err := db.ListPayloadManifest(p.ID)
	if err != nil {
		return fmt.Errorf("payload manifest: %w", err)
	}

	// Build manifest JSON from template items
	manifest := BinManifest{Items: make([]ManifestEntry, len(items))}
	for i, item := range items {
		manifest.Items[i] = ManifestEntry{
			CatID:    item.PartNumber,
			Quantity: item.Quantity,
		}
	}
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}

	uop := uopCapacity
	if uop == 0 {
		uop = p.UOPCapacity
	}

	return db.SetBinManifest(binID, string(manifestJSON), payloadCode, uop)
}
