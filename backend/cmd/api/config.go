package main

import (
	"fmt"
	"net"
	"os"
	"strings"
)

func validateRuntimeConfig() error {
	if !strings.EqualFold(strings.TrimSpace(os.Getenv("PROGRESS_TRACKER_ENV")), "production") {
		return nil
	}
	if !secureCookiesEnabled() {
		return fmt.Errorf("PROGRESS_TRACKER_SECURE_COOKIES=true is required in production")
	}
	if strings.TrimSpace(os.Getenv("PROGRESS_TRACKER_ALLOWED_ORIGINS")) == "" {
		return fmt.Errorf("PROGRESS_TRACKER_ALLOWED_ORIGINS is required in production")
	}
	publicURL := strings.TrimSpace(os.Getenv("PROGRESS_TRACKER_PUBLIC_URL"))
	if !strings.HasPrefix(publicURL, "https://") {
		return fmt.Errorf("PROGRESS_TRACKER_PUBLIC_URL must use HTTPS in production")
	}
	if strings.TrimSpace(os.Getenv("PROGRESS_TRACKER_SMTP_HOST")) == "" {
		return fmt.Errorf("PROGRESS_TRACKER_SMTP_HOST is required in production")
	}
	return nil
}

func developmentTimerSpeedEnabled() bool {
	if value := strings.TrimSpace(os.Getenv("PROGRESS_TRACKER_DEV_TIMER_SPEED")); value != "" {
		return strings.EqualFold(value, "true")
	}

	host := strings.TrimSpace(os.Getenv("PROGRESS_TRACKER_HOST"))
	if host == "" {
		host = "127.0.0.1"
	}
	return host == "localhost" || net.ParseIP(host).IsLoopback()
}

func developmentActionTokensEnabled() bool {
	if value := strings.TrimSpace(os.Getenv("PROGRESS_TRACKER_DEV_ACTION_TOKENS")); value != "" {
		return strings.EqualFold(value, "true")
	}
	return developmentTimerSpeedEnabled()
}
