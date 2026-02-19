package store

// ChangeoverLog records a changeover state transition.
type ChangeoverLog struct {
	ID           int64  `json:"id"`
	FromJobStyle string `json:"from_job_style"`
	ToJobStyle   string `json:"to_job_style"`
	State        string `json:"state"`
	Detail       string `json:"detail"`
	Operator     string `json:"operator"`
	LineID       int64  `json:"line_id"`
	CreatedAt    string `json:"created_at"`
}

func (db *DB) InsertChangeoverLog(fromJobStyle, toJobStyle, state, detail, operator string, lineID int64) (int64, error) {
	res, err := db.Exec(`INSERT INTO changeover_log (from_job_style, to_job_style, state, detail, operator, line_id) VALUES (?, ?, ?, ?, ?, ?)`,
		fromJobStyle, toJobStyle, state, detail, operator, lineID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (db *DB) ListChangeoverLog(limit int) ([]ChangeoverLog, error) {
	rows, err := db.Query(`SELECT id, from_job_style, to_job_style, state, detail, operator, COALESCE(line_id, 0), created_at FROM changeover_log ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanChangeoverLogs(rows)
}

func (db *DB) ListChangeoverLogByLine(lineID int64, limit int) ([]ChangeoverLog, error) {
	rows, err := db.Query(`SELECT id, from_job_style, to_job_style, state, detail, operator, COALESCE(line_id, 0), created_at FROM changeover_log WHERE line_id = ? ORDER BY created_at DESC LIMIT ?`, lineID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanChangeoverLogs(rows)
}

// ListCurrentChangeoverLog returns entries from the most recent changeover session for a line.
func (db *DB) ListCurrentChangeoverLog(lineID int64) ([]ChangeoverLog, error) {
	rows, err := db.Query(`SELECT id, from_job_style, to_job_style, state, detail, operator, COALESCE(line_id, 0), created_at
		FROM changeover_log
		WHERE line_id = ? AND id >= COALESCE(
			(SELECT MAX(id) FROM changeover_log WHERE line_id = ? AND state = 'stopping'), 0)
		ORDER BY created_at ASC`, lineID, lineID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanChangeoverLogs(rows)
}

func scanChangeoverLogs(rows interface{ Next() bool; Scan(...interface{}) error; Err() error }) ([]ChangeoverLog, error) {
	var logs []ChangeoverLog
	for rows.Next() {
		var l ChangeoverLog
		if err := rows.Scan(&l.ID, &l.FromJobStyle, &l.ToJobStyle, &l.State, &l.Detail, &l.Operator, &l.LineID, &l.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}
