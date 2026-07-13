package main

import (
	"database/sql"
	"fmt"
	"time"
)

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
	_, err := database.Exec(`DELETE FROM auth_sessions WHERE expires_at <= ?`, time.Now().Format(time.RFC3339))
	return err
}
