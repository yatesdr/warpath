package store

import "fmt"

func (db *DB) AssignBlueprintToNode(nodeID, blueprintID int64) error {
	_, err := db.Exec(db.Q(`INSERT INTO node_blueprints (node_id, blueprint_id) VALUES (?, ?) ON CONFLICT DO NOTHING`), nodeID, blueprintID)
	return err
}

func (db *DB) UnassignBlueprintFromNode(nodeID, blueprintID int64) error {
	_, err := db.Exec(db.Q(`DELETE FROM node_blueprints WHERE node_id=? AND blueprint_id=?`), nodeID, blueprintID)
	return err
}

func (db *DB) ListBlueprintsForNode(nodeID int64) ([]*Blueprint, error) {
	rows, err := db.Query(db.Q(fmt.Sprintf(`
		SELECT %s FROM blueprints
		WHERE id IN (SELECT blueprint_id FROM node_blueprints WHERE node_id=?)
		ORDER BY code`, blueprintSelectCols)), nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBlueprints(rows)
}

func (db *DB) ListNodesForBlueprint(blueprintID int64) ([]*Node, error) {
	rows, err := db.Query(db.Q(fmt.Sprintf(`
		SELECT %s %s
		WHERE n.id IN (SELECT nb.node_id FROM node_blueprints nb WHERE nb.blueprint_id=?)
		ORDER BY n.name`, nodeSelectCols, nodeFromClause)), blueprintID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNodes(rows)
}

// GetEffectiveBlueprints returns blueprints for a node, walking up the parent
// chain until a non-empty set is found. Returns nil (all blueprints) if no ancestor has blueprints.
func (db *DB) GetEffectiveBlueprints(nodeID int64) ([]*Blueprint, error) {
	cur := nodeID
	for {
		bps, err := db.ListBlueprintsForNode(cur)
		if err != nil {
			return nil, err
		}
		if len(bps) > 0 {
			return bps, nil
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

// SetNodeBlueprints replaces all blueprint assignments for a node.
func (db *DB) SetNodeBlueprints(nodeID int64, blueprintIDs []int64) error {
	if _, err := db.Exec(db.Q(`DELETE FROM node_blueprints WHERE node_id=?`), nodeID); err != nil {
		return err
	}
	for _, bpID := range blueprintIDs {
		if _, err := db.Exec(db.Q(`INSERT INTO node_blueprints (node_id, blueprint_id) VALUES (?, ?)`), nodeID, bpID); err != nil {
			return err
		}
	}
	return nil
}
