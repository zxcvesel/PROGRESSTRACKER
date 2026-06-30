package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"

	_ "modernc.org/sqlite"
)

var db *sql.DB

func main() {
	database, err := openDatabase()
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	db = database

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /entries", entriesHandler)
	mux.HandleFunc("POST /entries", createEntryHandler)
	mux.HandleFunc("GET /stats", statsHandler)

	log.Println("Backend is running on http://localhost:8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := map[string]string{
		"status": "ok",
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "failed to write response", http.StatusInternalServerError)
	}
}

type Entry struct {
	ID       int    `json:"id"`
	Date     string `json:"date"`
	Category string `json:"category"`
	Minutes  int    `json:"minutes"`
	Note     string `json:"note"`
}

type Stats struct {
	TotalEntries int `json:"totalEntries"`
	TotalMinutes int `json:"totalMinutes"`
}

func openDatabase() (*sql.DB, error) {
	if err := os.MkdirAll("data", 0755); err != nil {
		return nil, err
	}

	database, err := sql.Open("sqlite", "data/progress.db")
	if err != nil {
		return nil, err
	}

	createTableQuery := `
		CREATE TABLE IF NOT EXISTS entries (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			date TEXT NOT NULL,
			category TEXT NOT NULL,
			minutes INTEGER NOT NULL,
			note TEXT NOT NULL DEFAULT ''
		);
	`

	if _, err := database.Exec(createTableQuery); err != nil {
		database.Close()
		return nil, err
	}

	return database, nil
}

func entriesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	rows, err := db.Query(`
		SELECT id, date, category, minutes, note
		FROM entries
		ORDER BY date DESC, id DESC
	`)
	if err != nil {
		http.Error(w, "failed to load entries", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	entries := []Entry{}
	for rows.Next() {
		var entry Entry
		if err := rows.Scan(&entry.ID, &entry.Date, &entry.Category, &entry.Minutes, &entry.Note); err != nil {
			http.Error(w, "failed to read entry", http.StatusInternalServerError)
			return
		}

		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "failed to load entries", http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(entries); err != nil {
		http.Error(w, "failed to write response", http.StatusInternalServerError)
	}
}

func createEntryHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var entry Entry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if entry.Date == "" || entry.Category == "" || entry.Minutes <= 0 {
		http.Error(w, "date, category, and positive minutes are required", http.StatusBadRequest)
		return
	}

	result, err := db.Exec(`
		INSERT INTO entries (date, category, minutes, note)
		VALUES (?, ?, ?, ?)
	`, entry.Date, entry.Category, entry.Minutes, entry.Note)
	if err != nil {
		http.Error(w, "failed to save entry", http.StatusInternalServerError)
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		http.Error(w, "failed to read created entry", http.StatusInternalServerError)
		return
	}

	entry.ID = int(id)

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(entry); err != nil {
		http.Error(w, "failed to write response", http.StatusInternalServerError)
	}
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var stats Stats
	err := db.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(minutes), 0)
		FROM entries
	`).Scan(&stats.TotalEntries, &stats.TotalMinutes)
	if err != nil {
		http.Error(w, "failed to load stats", http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(stats); err != nil {
		http.Error(w, "failed to write response", http.StatusInternalServerError)
	}
}
