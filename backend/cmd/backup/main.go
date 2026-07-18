package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

func main() {
	databasePath := flag.String("db", "data/progress.db", "path to the SQLite database")
	outputPath := flag.String("out", "", "backup file path")
	checkPath := flag.String("check", "", "verify an existing SQLite backup")
	flag.Parse()
	if *checkPath != "" {
		if err := verifySQLiteDatabase(*checkPath); err != nil {
			log.Fatal(err)
		}
		absCheck, err := filepath.Abs(*checkPath)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(absCheck)
		return
	}
	if *outputPath == "" {
		log.Fatal("-out is required unless -check is used")
	}

	absOutput, err := createBackup(*databasePath, *outputPath)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(absOutput)
}

func createBackup(databasePath string, outputPath string) (string, error) {
	absDatabase, err := filepath.Abs(databasePath)
	if err != nil {
		return "", err
	}
	absOutput, err := filepath.Abs(outputPath)
	if err != nil {
		return "", err
	}
	if absDatabase == absOutput {
		return "", fmt.Errorf("backup path must differ from database path")
	}
	if err := os.MkdirAll(filepath.Dir(absOutput), 0o700); err != nil {
		return "", err
	}
	if err := os.Remove(absOutput); err != nil && !os.IsNotExist(err) {
		return "", err
	}
	database, err := sql.Open("sqlite", absDatabase)
	if err != nil {
		return "", err
	}
	defer database.Close()
	quotedOutput := strings.ReplaceAll(filepath.ToSlash(absOutput), "'", "''")
	if _, err := database.Exec("VACUUM INTO '" + quotedOutput + "'"); err != nil {
		return "", err
	}
	if err := os.Chmod(absOutput, 0o600); err != nil {
		return "", err
	}
	if err := verifySQLiteDatabase(absOutput); err != nil {
		return "", fmt.Errorf("verify backup: %w", err)
	}
	return absOutput, nil
}

func verifySQLiteDatabase(databasePath string) error {
	absDatabase, err := filepath.Abs(databasePath)
	if err != nil {
		return err
	}
	info, err := os.Stat(absDatabase)
	if err != nil {
		return err
	}
	if info.Size() == 0 {
		return fmt.Errorf("database file is empty")
	}

	database, err := sql.Open("sqlite", absDatabase)
	if err != nil {
		return err
	}
	defer database.Close()

	var result string
	if err := database.QueryRow("PRAGMA integrity_check").Scan(&result); err != nil {
		return err
	}
	if result != "ok" {
		return fmt.Errorf("SQLite integrity check failed: %s", result)
	}
	return nil
}
