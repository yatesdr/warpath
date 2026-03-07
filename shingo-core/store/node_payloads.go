package store

import "fmt"

func (db *DB) AssignPayloadToNode(nodeID, payloadID int64) error {
	_, err := db.Exec(db.Q(`INSERT INTO node_payloads (node_id, payload_id) VALUES (?, ?) ON CONFLICT DO NOTHING`), nodeID, payloadID)
	return err
}

func (db *DB) UnassignPayloadFromNode(nodeID, payloadID int64) error {
	_, err := db.Exec(db.Q(`DELETE FROM node_payloads WHERE node_id=? AND payload_id=?`), nodeID, payloadID)
	return err
}

func (db *DB) ListPayloadsForNode(nodeID int64) ([]*Payload, error) {
	rows, err := db.Query(db.Q(fmt.Sprintf(`
		SELECT %s FROM payloads
		WHERE id IN (SELECT payload_id FROM node_payloads WHERE node_id=?)
		ORDER BY code`, payloadSelectCols)), nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPayloads(rows)
}

func (db *DB) ListNodesForPayload(payloadID int64) ([]*Node, error) {
	rows, err := db.Query(db.Q(fmt.Sprintf(`
		SELECT %s %s
		WHERE n.id IN (SELECT np.node_id FROM node_payloads np WHERE np.payload_id=?)
		ORDER BY n.name`, nodeSelectCols, nodeFromClause)), payloadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNodes(rows)
}

// GetEffectivePayloads returns payload templates for a node, walking up the parent
// chain until a non-empty set is found. Returns nil (all payloads) if no ancestor has payloads.
func (db *DB) GetEffectivePayloads(nodeID int64) ([]*Payload, error) {
	cur := nodeID
	for {
		ps, err := db.ListPayloadsForNode(cur)
		if err != nil {
			return nil, err
		}
		if len(ps) > 0 {
			return ps, nil
		}
		node, err := db.GetNode(cur)
		if err != nil {
			return nil, nil
		}
		if node.ParentID == nil {
			return nil, nil
		}
		cur = *node.ParentID
	}
}

// SetNodePayloads replaces all payload template assignments for a node.
func (db *DB) SetNodePayloads(nodeID int64, payloadIDs []int64) error {
	if _, err := db.Exec(db.Q(`DELETE FROM node_payloads WHERE node_id=?`), nodeID); err != nil {
		return err
	}
	for _, pID := range payloadIDs {
		if _, err := db.Exec(db.Q(`INSERT INTO node_payloads (node_id, payload_id) VALUES (?, ?)`), nodeID, pID); err != nil {
			return err
		}
	}
	return nil
}

