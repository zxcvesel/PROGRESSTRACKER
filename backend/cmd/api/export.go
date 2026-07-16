package main

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"time"
)

type accountExport struct {
	ExportedAt string       `json:"exportedAt"`
	User       User         `json:"user"`
	Goals      []GoalExport `json:"goals"`
}

type GoalExport struct {
	Goal     Goal      `json:"goal"`
	Sessions []Session `json:"sessions"`
}

func exportAccountHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := currentVerifiedUserFromRequest(w, r)
	if !ok {
		return
	}
	goals, err := loadGoals(user.ID)
	if err != nil {
		writeError(w, "failed to export account", http.StatusInternalServerError)
		return
	}

	export := accountExport{ExportedAt: time.Now().UTC().Format(time.RFC3339), User: user, Goals: []GoalExport{}}
	for _, goal := range goals {
		sessions, err := loadAllSessions(goal.ID)
		if err != nil {
			writeError(w, "failed to export account", http.StatusInternalServerError)
			return
		}
		export.Goals = append(export.Goals, GoalExport{Goal: goal, Sessions: sessions})
	}

	if r.URL.Query().Get("format") == "csv" {
		writeAccountCSV(w, export)
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="progress-tracker-export.json"`)
	writeJSON(w, export, http.StatusOK)
}

func loadAllSessions(goalID int) ([]Session, error) {
	rows, err := db.Query(`
		SELECT id, goal_id, started_at, ended_at, duration_minutes, notes, tags, created_at
		FROM sessions WHERE goal_id = ? ORDER BY ended_at DESC, id DESC
	`, goalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	sessions := []Session{}
	for rows.Next() {
		session, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}
	return sessions, rows.Err()
}

func writeAccountCSV(w http.ResponseWriter, export accountExport) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="progress-tracker-sessions.csv"`)
	w.WriteHeader(http.StatusOK)
	writer := csv.NewWriter(w)
	_ = writer.Write([]string{"goal", "date", "started_at", "ended_at", "duration_minutes", "notes", "tags"})
	for _, item := range export.Goals {
		for _, session := range item.Sessions {
			_ = writer.Write([]string{
				item.Goal.Title,
				sessionDateString(session.EndedAt),
				session.StartedAt,
				session.EndedAt,
				strconv.Itoa(session.DurationMinutes),
				session.Notes,
				tagsToString(session.Tags),
			})
		}
	}
	writer.Flush()
}
