package store

import "time"

// AdminUser is a user who can access the setup page.
type AdminUser struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

func (db *DB) GetAdminUser(username string) (*AdminUser, error) {
	u := &AdminUser{}
	var createdAt string
	err := db.QueryRow(`SELECT id, username, password_hash, created_at FROM admin_users WHERE username = ?`, username).
		Scan(&u.ID, &u.Username, &u.PasswordHash, &createdAt)
	if err != nil {
		return nil, err
	}
	u.CreatedAt = scanTime(createdAt)
	return u, nil
}

func (db *DB) CreateAdminUser(username, passwordHash string) (int64, error) {
	res, err := db.Exec(`INSERT INTO admin_users (username, password_hash) VALUES (?, ?)`, username, passwordHash)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (db *DB) UpdateAdminPassword(username, passwordHash string) error {
	_, err := db.Exec(`UPDATE admin_users SET password_hash = ? WHERE username = ?`, passwordHash, username)
	return err
}

func (db *DB) AdminUserExists() (bool, error) {
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM admin_users`).Scan(&count)
	return count > 0, err
}
