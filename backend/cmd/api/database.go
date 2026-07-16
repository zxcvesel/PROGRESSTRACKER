package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

func openDatabase() (*sql.DB, error) {
	dbPath := os.Getenv("PROGRESS_TRACKER_DB_PATH")
	if dbPath == "" {
		dbPath = "data/progress.db"
	}

	if dbPath != ":memory:" {
		directory := filepath.Dir(dbPath)
		if err := os.MkdirAll(directory, 0o700); err != nil {
			return nil, err
		}
		if directory != "." {
			if err := os.Chmod(directory, 0o700); err != nil {
				return nil, err
			}
		}
	}

	database, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	if err := configureDatabase(database, dbPath); err != nil {
		database.Close()
		return nil, err
	}
	if dbPath != ":memory:" {
		if err := os.Chmod(dbPath, 0o600); err != nil {
			database.Close()
			return nil, err
		}
	}

	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT NOT NULL UNIQUE COLLATE NOCASE,
			name TEXT NOT NULL DEFAULT '',
			password_hash TEXT NOT NULL,
			created_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS auth_sessions (
			token_hash TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			created_at TEXT NOT NULL,
			expires_at TEXT NOT NULL,
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS entries (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			date TEXT NOT NULL,
			category TEXT NOT NULL,
			minutes INTEGER NOT NULL CHECK(minutes BETWEEN 1 AND 1440),
			note TEXT NOT NULL DEFAULT ''
		);`,
		`CREATE TABLE IF NOT EXISTS goals (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			total_days INTEGER NOT NULL CHECK(total_days BETWEEN 1 AND 3650),
			daily_target_minutes INTEGER NOT NULL CHECK(daily_target_minutes BETWEEN 1 AND 1440),
			active_weekdays TEXT NOT NULL,
			start_date TEXT NOT NULL,
			created_at TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active', 'completed'))
		);`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			goal_id INTEGER NOT NULL,
			started_at TEXT NOT NULL,
			ended_at TEXT NOT NULL,
			duration_minutes INTEGER NOT NULL CHECK(duration_minutes BETWEEN 1 AND 1440),
			notes TEXT NOT NULL DEFAULT '',
			tags TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			FOREIGN KEY(goal_id) REFERENCES goals(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS daily_progress (
			goal_id INTEGER NOT NULL,
			date TEXT NOT NULL,
			total_minutes INTEGER NOT NULL DEFAULT 0 CHECK(total_minutes >= 0),
			target_minutes INTEGER NOT NULL CHECK(target_minutes BETWEEN 1 AND 1440),
			is_completed INTEGER NOT NULL DEFAULT 0 CHECK(is_completed IN (0, 1)),
			PRIMARY KEY(goal_id, date),
			FOREIGN KEY(goal_id) REFERENCES goals(id) ON DELETE CASCADE
		);`,
	}
	for _, query := range queries {
		if _, err := database.Exec(query); err != nil {
			database.Close()
			return nil, err
		}
	}
	if err := addColumnIfMissing(database, "goals", "user_id", "user_id INTEGER NOT NULL DEFAULT 1"); err != nil {
		database.Close()
		return nil, err
	}
	if err := addColumnIfMissing(database, "entries", "user_id", "user_id INTEGER NOT NULL DEFAULT 1"); err != nil {
		database.Close()
		return nil, err
	}
	if err := runDatabaseMigrations(database); err != nil {
		database.Close()
		return nil, err
	}
	if err := cleanupExpiredAuthSessions(database); err != nil {
		database.Close()
		return nil, err
	}
	return database, nil
}

type databaseMigration struct {
	version int
	name    string
	SQL     string
}

var databaseMigrations = []databaseMigration{
	{
		version: 1,
		name:    "indexes and data constraints",
		SQL: `
			CREATE INDEX IF NOT EXISTS idx_auth_sessions_user_id ON auth_sessions(user_id);
			CREATE INDEX IF NOT EXISTS idx_auth_sessions_expires_at ON auth_sessions(expires_at);
			CREATE INDEX IF NOT EXISTS idx_entries_user_date ON entries(user_id, date DESC);
			CREATE INDEX IF NOT EXISTS idx_goals_user_status ON goals(user_id, status);
			CREATE INDEX IF NOT EXISTS idx_sessions_goal_ended_at ON sessions(goal_id, ended_at DESC);
			CREATE INDEX IF NOT EXISTS idx_daily_progress_date ON daily_progress(date);

			CREATE TRIGGER IF NOT EXISTS validate_goal_insert
			BEFORE INSERT ON goals
			WHEN trim(NEW.title) = '' OR length(NEW.title) > 120
				OR NEW.total_days NOT BETWEEN 1 AND 3650
				OR NEW.daily_target_minutes NOT BETWEEN 1 AND 1440
				OR NEW.status NOT IN ('active', 'completed')
			BEGIN SELECT RAISE(ABORT, 'invalid goal data'); END;

			CREATE TRIGGER IF NOT EXISTS validate_goal_update
			BEFORE UPDATE ON goals
			WHEN trim(NEW.title) = '' OR length(NEW.title) > 120
				OR NEW.total_days NOT BETWEEN 1 AND 3650
				OR NEW.daily_target_minutes NOT BETWEEN 1 AND 1440
				OR NEW.status NOT IN ('active', 'completed')
			BEGIN SELECT RAISE(ABORT, 'invalid goal data'); END;

			CREATE TRIGGER IF NOT EXISTS validate_session_insert
			BEFORE INSERT ON sessions
			WHEN NEW.duration_minutes NOT BETWEEN 1 AND 1440
			BEGIN SELECT RAISE(ABORT, 'invalid session duration'); END;

			CREATE TRIGGER IF NOT EXISTS validate_session_update
			BEFORE UPDATE ON sessions
			WHEN NEW.duration_minutes NOT BETWEEN 1 AND 1440
			BEGIN SELECT RAISE(ABORT, 'invalid session duration'); END;

			CREATE TRIGGER IF NOT EXISTS validate_entry_insert
			BEFORE INSERT ON entries
			WHEN NEW.minutes NOT BETWEEN 1 AND 1440
			BEGIN SELECT RAISE(ABORT, 'invalid entry duration'); END;

			CREATE TRIGGER IF NOT EXISTS validate_entry_update
			BEFORE UPDATE ON entries
			WHEN NEW.minutes NOT BETWEEN 1 AND 1440
			BEGIN SELECT RAISE(ABORT, 'invalid entry duration'); END;
		`,
	},
	{
		version: 2,
		name:    "one session per goal and calendar day",
		SQL: `
			ALTER TABLE sessions ADD COLUMN session_date TEXT NOT NULL DEFAULT '';
			UPDATE sessions
			SET session_date = substr(ended_at, 1, 10)
			WHERE session_date = '';

			UPDATE sessions AS keeper
			SET duration_minutes = MIN(1440, (
					SELECT COALESCE(SUM(duplicate.duration_minutes), keeper.duration_minutes)
					FROM sessions AS duplicate
					WHERE duplicate.goal_id = keeper.goal_id
						AND duplicate.session_date = keeper.session_date
				)),
				started_at = (
					SELECT MIN(duplicate.started_at)
					FROM sessions AS duplicate
					WHERE duplicate.goal_id = keeper.goal_id
						AND duplicate.session_date = keeper.session_date
				),
				ended_at = (
					SELECT MAX(duplicate.ended_at)
					FROM sessions AS duplicate
					WHERE duplicate.goal_id = keeper.goal_id
						AND duplicate.session_date = keeper.session_date
				),
				notes = COALESCE((
					SELECT GROUP_CONCAT(NULLIF(duplicate.notes, ''), char(10) || char(10))
					FROM sessions AS duplicate
					WHERE duplicate.goal_id = keeper.goal_id
						AND duplicate.session_date = keeper.session_date
				), ''),
				tags = COALESCE((
					SELECT GROUP_CONCAT(NULLIF(duplicate.tags, ''), ',')
					FROM sessions AS duplicate
					WHERE duplicate.goal_id = keeper.goal_id
						AND duplicate.session_date = keeper.session_date
				), '')
			WHERE keeper.id IN (
				SELECT MIN(id)
				FROM sessions
				GROUP BY goal_id, session_date
			);

			DELETE FROM sessions
			WHERE id NOT IN (
				SELECT MIN(id)
				FROM sessions
				GROUP BY goal_id, session_date
			);

			CREATE UNIQUE INDEX idx_sessions_goal_session_date
			ON sessions(goal_id, session_date);

			CREATE TRIGGER validate_session_date_insert
			BEFORE INSERT ON sessions
			WHEN length(NEW.session_date) != 10
				OR date(NEW.session_date) IS NULL
				OR NEW.session_date != substr(NEW.ended_at, 1, 10)
			BEGIN SELECT RAISE(ABORT, 'invalid session date'); END;

			CREATE TRIGGER validate_session_date_update
			BEFORE UPDATE ON sessions
			WHEN length(NEW.session_date) != 10
				OR date(NEW.session_date) IS NULL
				OR NEW.session_date != substr(NEW.ended_at, 1, 10)
			BEGIN SELECT RAISE(ABORT, 'invalid session date'); END;
		`,
	},
	{
		version: 3,
		name:    "server managed active timers",
		SQL: `
			CREATE TABLE active_timers (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				user_id INTEGER NOT NULL UNIQUE,
				goal_id INTEGER NOT NULL UNIQUE,
				session_date TEXT NOT NULL,
				state TEXT NOT NULL CHECK(state IN ('running', 'paused', 'finished')),
				started_at TEXT NOT NULL,
				last_resumed_at TEXT NOT NULL,
				accumulated_seconds REAL NOT NULL DEFAULT 0 CHECK(accumulated_seconds >= 0),
				target_seconds INTEGER NOT NULL CHECK(target_seconds BETWEEN 1 AND 86400),
				speed_multiplier REAL NOT NULL DEFAULT 1 CHECK(speed_multiplier > 0 AND speed_multiplier <= 5),
				updated_at TEXT NOT NULL,
				FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
				FOREIGN KEY(goal_id) REFERENCES goals(id) ON DELETE CASCADE
			);
			CREATE INDEX idx_active_timers_goal_id ON active_timers(goal_id);
		`,
	},
	{
		version: 4,
		name:    "account action tokens",
		SQL: `
			ALTER TABLE users ADD COLUMN email_verified INTEGER NOT NULL DEFAULT 1;
			CREATE TABLE action_tokens (
				token_hash TEXT PRIMARY KEY,
				user_id INTEGER NOT NULL,
				kind TEXT NOT NULL CHECK(kind IN ('verify_email', 'reset_password')),
				created_at TEXT NOT NULL,
				expires_at TEXT NOT NULL,
				FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
			);
			CREATE INDEX idx_action_tokens_user_kind ON action_tokens(user_id, kind);
			CREATE INDEX idx_action_tokens_expires_at ON action_tokens(expires_at);
		`,
	},
	{
		version: 5,
		name:    "account timezone",
		SQL: `
			ALTER TABLE users ADD COLUMN timezone TEXT NOT NULL DEFAULT 'UTC';
		`,
	},
}

func configureDatabase(database *sql.DB, path string) error {
	database.SetMaxOpenConns(1)
	database.SetMaxIdleConns(1)

	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA busy_timeout = 5000",
	}
	if path != ":memory:" {
		pragmas = append(pragmas, "PRAGMA journal_mode = WAL")
	}
	for _, pragma := range pragmas {
		if _, err := database.Exec(pragma); err != nil {
			return fmt.Errorf("configure database: %w", err)
		}
	}
	return database.Ping()
}

func runDatabaseMigrations(database *sql.DB) error {
	if _, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TEXT NOT NULL
		)
	`); err != nil {
		return fmt.Errorf("create migration table: %w", err)
	}

	for _, migration := range databaseMigrations {
		var applied int
		if err := database.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, migration.version).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %d: %w", migration.version, err)
		}
		if applied > 0 {
			continue
		}

		tx, err := database.Begin()
		if err != nil {
			return fmt.Errorf("begin migration %d: %w", migration.version, err)
		}
		if _, err := tx.Exec(migration.SQL); err != nil {
			tx.Rollback()
			return fmt.Errorf("apply migration %d: %w", migration.version, err)
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, ?)`,
			migration.version, migration.name, time.Now().UTC().Format(time.RFC3339)); err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %d: %w", migration.version, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %d: %w", migration.version, err)
		}
	}
	return nil
}

func cleanupExpiredAuthSessions(database *sql.DB) error {
	now := time.Now().Format(time.RFC3339)
	if _, err := database.Exec(`DELETE FROM auth_sessions WHERE expires_at <= ?`, now); err != nil {
		return err
	}
	_, err := database.Exec(`DELETE FROM action_tokens WHERE expires_at <= ?`, now)
	return err
}
