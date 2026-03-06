package store

import "fmt"

// SetNodeBinTypes replaces all bin type assignments for a node.
func (db *DB) SetNodeBinTypes(nodeID int64, binTypeIDs []int64) error {
	if _, err := db.Exec(db.Q(`DELETE FROM node_bin_types WHERE node_id=?`), nodeID); err != nil {
		return err
	}
	for _, btID := range binTypeIDs {
		if _, err := db.Exec(db.Q(`INSERT INTO node_bin_types (node_id, bin_type_id) VALUES (?, ?)`), nodeID, btID); err != nil {
			return err
		}
	}
	return nil
}

// ListBinTypesForNode returns the directly assigned bin types for a node.
func (db *DB) ListBinTypesForNode(nodeID int64) ([]*BinType, error) {
	rows, err := db.Query(db.Q(fmt.Sprintf(`
		SELECT %s FROM bin_types
		WHERE id IN (SELECT bin_type_id FROM node_bin_types WHERE node_id=?)
		ORDER BY code`, binTypeSelectCols)), nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBinTypes(rows)
}

// GetEffectiveBinTypes returns bin types for a node based on its bin_type_mode property:
//   - "all": no restrictions (returns nil)
//   - "specific": returns directly assigned bin types
//   - "" / "inherit": walks parent chain until a non-empty set is found
func (db *DB) GetEffectiveBinTypes(nodeID int64) ([]*BinType, error) {
	mode := db.GetNodeProperty(nodeID, "bin_type_mode")
	switch mode {
	case "all":
		return nil, nil
	case "specific":
		return db.ListBinTypesForNode(nodeID)
	default: // "" or "inherit"
		cur := nodeID
		for {
			bts, err := db.ListBinTypesForNode(cur)
			if err != nil {
				return nil, err
			}
			if len(bts) > 0 {
				return bts, nil
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
}
