package main

import (
	"fmt"
	"net/smtp"
	"net/url"
	"os"
	"strings"
)

func sendAccountActionEmail(to string, kind string, token string) error {
	host := strings.TrimSpace(os.Getenv("PROGRESS_TRACKER_SMTP_HOST"))
	if host == "" {
		return nil
	}
	port := strings.TrimSpace(os.Getenv("PROGRESS_TRACKER_SMTP_PORT"))
	if port == "" {
		port = "587"
	}
	username := os.Getenv("PROGRESS_TRACKER_SMTP_USERNAME")
	password := os.Getenv("PROGRESS_TRACKER_SMTP_PASSWORD")
	from := strings.TrimSpace(os.Getenv("PROGRESS_TRACKER_SMTP_FROM"))
	if from == "" {
		from = username
	}
	if from == "" {
		return fmt.Errorf("SMTP sender is not configured")
	}

	publicURL := strings.TrimSuffix(strings.TrimSpace(os.Getenv("PROGRESS_TRACKER_PUBLIC_URL")), "/")
	if publicURL == "" {
		publicURL = "http://127.0.0.1:5173"
	}
	parameter := "verifyToken"
	subject := "Verify your Progress Tracker account"
	if kind == "reset_password" {
		parameter = "resetToken"
		subject = "Reset your Progress Tracker password"
	}
	link := publicURL + "/?" + parameter + "=" + url.QueryEscape(token)
	body := "Open this link to continue:\r\n\r\n" + link + "\r\n\r\nIf you did not request this action, ignore this message."
	message := []byte("To: " + to + "\r\n" +
		"From: " + from + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" + body)

	var auth smtp.Auth
	if username != "" {
		auth = smtp.PlainAuth("", username, password, host)
	}
	return smtp.SendMail(host+":"+port, auth, from, []string{to}, message)
}
