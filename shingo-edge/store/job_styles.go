package store

import (
	"database/sql"
	"encoding/json"
	"time"
)

// JobStyle represents a product/recipe style that maps to a BOM.
type JobStyle struct {
	ID          int64    `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	// CatIDs maps to the DB column "cat_id" (singular), which stores a JSON array
	// of catalog identifiers despite the singular column name.
	CatIDs    []string  `json:"cat_ids"`
	LineID    int64     `json:"line_id"`
	CreatedAt time.Time `json:"created_at"`
}

// scanCatIDs parses the "cat_id" TEXT column (singular name, stores JSON array) into a []string.
// Supports JSON arrays and legacy single-value strings.
func scanCatIDs(raw sql.NullString) []string {
	if !raw.Valid || raw.String == "" {
		return []string{}
	}
	var ids []string
	if err := json.Unmarshal([]byte(raw.String), &ids); err != nil {
		// Legacy: treat as single value
		return []string{raw.String}
	}
	return ids
}

// marshalCatIDs serializes []string to a JSON array string for the cat_id column.
func marshalCatIDs(ids []string) string {
	if len(ids) == 0 {
		return "[]"
	}
	b, _ := json.Marshal(ids)
	return string(b)
}

func scanJobStyle(scanner interface{ Scan(...interface{}) error }) (JobStyle, error) {
	var s JobStyle
	var rawCat sql.NullString
	var createdAt string
	if err := scanner.Scan(&s.ID, &s.Name, &s.Description, &rawCat, &s.LineID, &createdAt); err != nil {
		return s, err
	}
	s.CatIDs = scanCatIDs(rawCat)
	s.CreatedAt = scanTime(createdAt)
	return s, nil
}

func (db *DB) ListJobStyles() ([]JobStyle, error) {
	rows, err := db.Query(`SELECT id, name, description, cat_id, COALESCE(line_id, 0) as line_id, created_at FROM job_styles ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var styles []JobStyle
	for rows.Next() {
		s, err := scanJobStyle(rows)
		if err != nil {
			return nil, err
		}
		styles = append(styles, s)
	}
	return styles, rows.Err()
}

func (db *DB) ListJobStylesByLine(lineID int64) ([]JobStyle, error) {
	rows, err := db.Query(`SELECT id, name, description, cat_id, COALESCE(line_id, 0) as line_id, created_at FROM job_styles WHERE line_id = ? ORDER BY name`, lineID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var styles []JobStyle
	for rows.Next() {
		s, err := scanJobStyle(rows)
		if err != nil {
			return nil, err
		}
		styles = append(styles, s)
	}
	return styles, rows.Err()
}

func (db *DB) GetJobStyleByName(name string) (*JobStyle, error) {
	s, err := scanJobStyle(db.QueryRow(`SELECT id, name, description, cat_id, COALESCE(line_id, 0) as line_id, created_at FROM job_styles WHERE name = ?`, name))
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (db *DB) GetJobStyle(id int64) (*JobStyle, error) {
	s, err := scanJobStyle(db.QueryRow(`SELECT id, name, description, cat_id, COALESCE(line_id, 0) as line_id, created_at FROM job_styles WHERE id = ?`, id))
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (db *DB) CreateJobStyle(name, description string, catIDs []string, lineID int64) (int64, error) {
	res, err := db.Exec(`INSERT INTO job_styles (name, description, cat_id, line_id) VALUES (?, ?, ?, ?)`, name, description, marshalCatIDs(catIDs), lineID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (db *DB) UpdateJobStyle(id int64, name, description string, catIDs []string, lineID int64) error {
	_, err := db.Exec(`UPDATE job_styles SET name=?, description=?, cat_id=?, line_id=? WHERE id=?`, name, description, marshalCatIDs(catIDs), lineID, id)
	return err
}

func (db *DB) DeleteJobStyle(id int64) error {
	_, err := db.Exec(`DELETE FROM job_styles WHERE id=?`, id)
	return err
}
