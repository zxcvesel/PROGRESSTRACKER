package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	_ "modernc.org/sqlite"
)

func TestSessionDateMigrationMergesDuplicates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "migration.db")
	database, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if err := configureDatabase(database, path); err != nil {
		t.Fatal(err)
	}

	_, err = database.Exec(`
		CREATE TABLE schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TEXT NOT NULL
		);
		INSERT INTO schema_migrations (version, name, applied_at)
		VALUES
			(1, 'existing schema', '2026-07-13T00:00:00Z'),
			(3, 'not part of this isolated test', '2026-07-13T00:00:00Z'),
			(4, 'not part of this isolated test', '2026-07-13T00:00:00Z'),
			(5, 'not part of this isolated test', '2026-07-13T00:00:00Z');
		CREATE TABLE sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			goal_id INTEGER NOT NULL,
			started_at TEXT NOT NULL,
			ended_at TEXT NOT NULL,
			duration_minutes INTEGER NOT NULL,
			notes TEXT NOT NULL DEFAULT '',
			tags TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		);
		INSERT INTO sessions (goal_id, started_at, ended_at, duration_minutes, notes, tags, created_at)
		VALUES
			(7, '2026-07-12T10:00:00+03:00', '2026-07-12T10:05:00+03:00', 5, 'first', 'Go', '2026-07-12T10:05:00+03:00'),
			(7, '2026-07-12T11:00:00+03:00', '2026-07-12T11:07:00+03:00', 7, 'second', 'API', '2026-07-12T11:07:00+03:00');
	`)
	if err != nil {
		t.Fatal(err)
	}

	if err := runDatabaseMigrations(database); err != nil {
		t.Fatal(err)
	}

	var count int
	var minutes int
	var date string
	var notes string
	if err := database.QueryRow(`
		SELECT COUNT(*), duration_minutes, session_date, notes
		FROM sessions
		WHERE goal_id = 7
	`).Scan(&count, &minutes, &date, &notes); err != nil {
		t.Fatal(err)
	}
	if count != 1 || minutes != 12 || date != "2026-07-12" {
		t.Fatalf("merged session = count %d, minutes %d, date %q", count, minutes, date)
	}
	if notes != "first\n\nsecond" {
		t.Fatalf("merged notes = %q", notes)
	}

	_, err = database.Exec(`
		INSERT INTO sessions (
			goal_id, started_at, ended_at, duration_minutes,
			notes, tags, created_at, session_date
		)
		VALUES (7, ?, ?, 1, '', '', ?, '2026-07-12')
	`, "2026-07-12T12:00:00+03:00", "2026-07-12T12:01:00+03:00", "2026-07-12T12:01:00+03:00")
	if err == nil {
		t.Fatal("database accepted a duplicate daily session")
	}
}

func TestDatabaseFileUsesPrivatePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not expose Unix permission bits")
	}

	path := filepath.Join(t.TempDir(), "private", "progress.db")
	t.Setenv("PROGRESS_TRACKER_DB_PATH", path)
	database, err := openDatabase()
	if err != nil {
		t.Fatal(err)
	}
	database.Close()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if permissions := info.Mode().Perm(); permissions != 0600 {
		t.Fatalf("database permissions = %o, want 600", permissions)
	}
}

func TestOpenDatabaseAppliesMigrationsAndReopensCleanly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "migrations", "progress.db")
	t.Setenv("PROGRESS_TRACKER_DB_PATH", path)

	database, err := openDatabase()
	if err != nil {
		t.Fatal(err)
	}
	var migrationCount int
	if err := database.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&migrationCount); err != nil {
		database.Close()
		t.Fatal(err)
	}
	if migrationCount != len(databaseMigrations) {
		database.Close()
		t.Fatalf("migration count = %d, want %d", migrationCount, len(databaseMigrations))
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := openDatabase()
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	var integrity string
	if err := reopened.QueryRow(`PRAGMA integrity_check`).Scan(&integrity); err != nil {
		t.Fatal(err)
	}
	if integrity != "ok" {
		t.Fatalf("integrity check = %q", integrity)
	}
}
