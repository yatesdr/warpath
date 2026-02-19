package store

import (
	"database/sql"
	"fmt"
	"strings"

	"shingocore/config"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
	dialect Dialect
	driver  string
}

func Open(cfg *config.DatabaseConfig) (*DB, error) {
	switch cfg.Driver {
	case "sqlite":
		return openSQLite(cfg.SQLite.Path)
	case "postgres":
		return openPostgres(&cfg.Postgres)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Driver)
	}
}

func openSQLite(path string) (*DB, error) {
	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on", path)
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	sqlDB.SetMaxOpenConns(1)
	db := &DB{DB: sqlDB, dialect: sqliteDialect{}, driver: "sqlite"}
	if err := db.migrateRenames(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("migrate renames sqlite: %w", err)
	}
	if err := db.migrate(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("migrate sqlite: %w", err)
	}
	return db, nil
}

func openPostgres(cfg *config.PostgresConfig) (*DB, error) {
	dsn := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.Database, cfg.User, cfg.Password, cfg.SSLMode)
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	db := &DB{DB: sqlDB, dialect: postgresDialect{}, driver: "postgres"}
	if err := db.migrateRenames(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("migrate renames postgres: %w", err)
	}
	if err := db.migrate(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("migrate postgres: %w", err)
	}
	return db, nil
}

func (db *DB) Dialect() Dialect { return db.dialect }
func (db *DB) Driver() string   { return db.driver }

// Q rewrites ? placeholders and datetime literals for PostgreSQL, passes through for SQLite.
func (db *DB) Q(query string) string {
	if db.driver == "postgres" {
		query = strings.ReplaceAll(query, "datetime('now','localtime')", "NOW()")
		return Rebind(query)
	}
	return query
}

// migrateRenames idempotently renames old RDS-specific columns to vendor-neutral names.
// SQLite 3.25+ and PostgreSQL both support ALTER TABLE RENAME COLUMN.
func (db *DB) migrateRenames() error {
	renames := []struct{ table, oldCol, newCol string }{
		{"nodes", "rds_location", "vendor_location"},
		{"orders", "rds_order_id", "vendor_order_id"},
		{"orders", "rds_state", "vendor_state"},
	}
	for _, r := range renames {
		if db.columnExists(r.table, r.oldCol) {
			_, err := db.Exec(fmt.Sprintf(`ALTER TABLE %s RENAME COLUMN %s TO %s`, r.table, r.oldCol, r.newCol))
			if err != nil {
				return fmt.Errorf("rename %s.%s: %w", r.table, r.oldCol, err)
			}
		}
	}
	// Rename index idempotently (drop old, new one created by schema)
	if db.driver == "postgres" {
		db.Exec(`DROP INDEX IF EXISTS idx_orders_rds`)
	}
	return nil
}

// columnExists checks if a column exists in a table.
func (db *DB) columnExists(table, column string) bool {
	switch db.driver {
	case "sqlite":
		rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
		if err != nil {
			return false
		}
		defer rows.Close()
		for rows.Next() {
			var cid int
			var name, typ string
			var notnull int
			var dflt sql.NullString
			var pk int
			if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
				return false
			}
			if name == column {
				return true
			}
		}
		return false
	case "postgres":
		var exists bool
		db.QueryRow(`SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name=$1 AND column_name=$2)`, table, column).Scan(&exists)
		return exists
	}
	return false
}

func (db *DB) migrate() error {
	var schema string
	switch db.driver {
	case "sqlite":
		schema = schemaSQLite
	case "postgres":
		schema = schemaPostgres
	default:
		return fmt.Errorf("no schema for driver: %s", db.driver)
	}
	_, err := db.Exec(schema)
	return err
}
