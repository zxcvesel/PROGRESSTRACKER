package main

import (
	"database/sql"
	"log"
	"time"
)

var db *sql.DB

const (
	authTokenLifetime = 30 * 24 * time.Hour
	authCookieName    = "progress_tracker_session"
	passwordHashName  = "argon2id"
	passwordKeyBytes  = 32
	passwordSaltBytes = 16
	passwordMemory    = 64 * 1024
	passwordTime      = 3
	passwordThreads   = 1
	legacyHashName    = "pbkdf2_sha256"
	legacyRounds      = 120000
)

var authRateLimiter = newRateLimiter(12, 10*time.Minute)

func main() {
	if err := validateRuntimeConfig(); err != nil {
		log.Fatal(err)
	}
	database, err := openDatabase()
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()
	db = database
	if err := refreshDailyProgressForAllGoals(); err != nil {
		log.Fatal(err)
	}
	if err := runServer(newRouter()); err != nil {
		log.Fatal(err)
	}
}
