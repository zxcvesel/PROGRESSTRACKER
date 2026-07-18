package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestCreateBackupProducesReadableSQLiteCopy(t *testing.T) {
	directory := t.TempDir()
	source := filepath.Join(directory, "source.db")
	destination := filepath.Join(directory, "backups", "copy.db")
	database, err := sql.Open("sqlite", source)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`CREATE TABLE sample (value TEXT); INSERT INTO sample VALUES ('saved');`); err != nil {
		t.Fatal(err)
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := createBackup(source, destination); err != nil {
		t.Fatal(err)
	}
	backup, err := sql.Open("sqlite", destination)
	if err != nil {
		t.Fatal(err)
	}
	defer backup.Close()
	var value string
	if err := backup.QueryRow(`SELECT value FROM sample`).Scan(&value); err != nil {
		t.Fatal(err)
	}
	if value != "saved" {
		t.Fatalf("backup value = %q", value)
	}
	if err := verifySQLiteDatabase(destination); err != nil {
		t.Fatalf("verify backup: %v", err)
	}
}

func TestVerifySQLiteDatabaseRejectsCorruptFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "corrupt.db")
	if err := os.WriteFile(path, []byte("not a sqlite database"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verifySQLiteDatabase(path); err == nil {
		t.Fatal("corrupt database passed integrity verification")
	}
}
