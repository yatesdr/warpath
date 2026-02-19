package store

// KanbanTemplate defines a user-configurable E-Kanban order template.
type KanbanTemplate struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	OrderType   string `json:"order_type"`
	Payload     string `json:"payload"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
}

func (db *DB) ListKanbanTemplates() ([]KanbanTemplate, error) {
	rows, err := db.Query(`SELECT id, name, order_type, payload, description, created_at FROM kanban_templates ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var templates []KanbanTemplate
	for rows.Next() {
		var t KanbanTemplate
		if err := rows.Scan(&t.ID, &t.Name, &t.OrderType, &t.Payload, &t.Description, &t.CreatedAt); err != nil {
			return nil, err
		}
		templates = append(templates, t)
	}
	return templates, rows.Err()
}

func (db *DB) GetKanbanTemplate(id int64) (*KanbanTemplate, error) {
	t := &KanbanTemplate{}
	err := db.QueryRow(`SELECT id, name, order_type, payload, description, created_at FROM kanban_templates WHERE id = ?`, id).
		Scan(&t.ID, &t.Name, &t.OrderType, &t.Payload, &t.Description, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (db *DB) CreateKanbanTemplate(name, orderType, payload, description string) (int64, error) {
	res, err := db.Exec(`INSERT INTO kanban_templates (name, order_type, payload, description) VALUES (?, ?, ?, ?)`,
		name, orderType, payload, description)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (db *DB) UpdateKanbanTemplate(id int64, name, orderType, payload, description string) error {
	_, err := db.Exec(`UPDATE kanban_templates SET name=?, order_type=?, payload=?, description=? WHERE id=?`,
		name, orderType, payload, description, id)
	return err
}

func (db *DB) DeleteKanbanTemplate(id int64) error {
	_, err := db.Exec(`DELETE FROM kanban_templates WHERE id=?`, id)
	return err
}
