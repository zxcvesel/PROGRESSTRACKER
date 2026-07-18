package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"net/url"
	"os"
	"strings"
	"time"
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
	if strings.ContainsAny(to, "\r\n") || strings.ContainsAny(from, "\r\n") {
		return fmt.Errorf("SMTP address contains invalid characters")
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

	address := net.JoinHostPort(host, port)
	connection, err := (&net.Dialer{Timeout: 10 * time.Second}).Dial("tcp", address)
	if err != nil {
		return err
	}
	defer connection.Close()
	if err := connection.SetDeadline(time.Now().Add(20 * time.Second)); err != nil {
		return err
	}

	client, err := smtp.NewClient(connection, host)
	if err != nil {
		return err
	}
	defer client.Close()

	if available, _ := client.Extension("STARTTLS"); available {
		if err := client.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}); err != nil {
			return err
		}
	} else if strings.EqualFold(strings.TrimSpace(os.Getenv("PROGRESS_TRACKER_ENV")), "production") {
		return fmt.Errorf("SMTP server does not support STARTTLS")
	}

	if username != "" {
		if err := client.Auth(smtp.PlainAuth("", username, password, host)); err != nil {
			return err
		}
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(message); err != nil {
		writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}
