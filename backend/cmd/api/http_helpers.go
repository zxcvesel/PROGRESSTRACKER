package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
)

func writeJSON(w http.ResponseWriter, value any, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("failed to write response: %v", err)
	}
}

func writeError(w http.ResponseWriter, message string, status int) {
	writeJSON(w, ErrorResponse{Error: message}, status)
}

func rateLimitKey(r *http.Request, action string) string {
	return action + ":" + clientAddress(r)
}

func clientAddress(r *http.Request) string {
	remote := strings.TrimSpace(r.RemoteAddr)
	if strings.EqualFold(strings.TrimSpace(os.Getenv("PROGRESS_TRACKER_TRUST_PROXY")), "true") {
		if realIP := net.ParseIP(strings.TrimSpace(r.Header.Get("X-Real-IP"))); realIP != nil {
			remote = realIP.String()
		}
	}
	if host, _, err := net.SplitHostPort(remote); err == nil {
		remote = host
	}
	if remote == "" {
		remote = "unknown"
	}
	return remote
}

func addColumnIfMissing(database *sql.DB, table string, column string, definition string) error {
	rows, err := database.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var columnType string
		var notNull int
		var defaultValue sql.NullString
		var primaryKey int
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return err
		}
		if name == column {
			return rows.Err()
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	_, err = database.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", table, definition))
	return err
}
