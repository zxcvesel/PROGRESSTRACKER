package main

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
)

func validateRuntimeConfig() error {
	if !strings.EqualFold(strings.TrimSpace(os.Getenv("PROGRESS_TRACKER_ENV")), "production") {
		return nil
	}
	if !secureCookiesEnabled() {
		return fmt.Errorf("PROGRESS_TRACKER_SECURE_COOKIES=true is required in production")
	}
	allowedOrigins := strings.TrimSpace(os.Getenv("PROGRESS_TRACKER_ALLOWED_ORIGINS"))
	if allowedOrigins == "" {
		return fmt.Errorf("PROGRESS_TRACKER_ALLOWED_ORIGINS is required in production")
	}
	for _, origin := range strings.Split(allowedOrigins, ",") {
		parsed, err := url.Parse(strings.TrimSpace(origin))
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
			return fmt.Errorf("PROGRESS_TRACKER_ALLOWED_ORIGINS must contain HTTPS origins only")
		}
	}
	publicURL := strings.TrimSpace(os.Getenv("PROGRESS_TRACKER_PUBLIC_URL"))
	parsedPublicURL, err := url.Parse(publicURL)
	if err != nil || parsedPublicURL.Scheme != "https" || parsedPublicURL.Host == "" || parsedPublicURL.RawQuery != "" || parsedPublicURL.Fragment != "" {
		return fmt.Errorf("PROGRESS_TRACKER_PUBLIC_URL must use HTTPS in production")
	}
	if strings.TrimSpace(os.Getenv("PROGRESS_TRACKER_SMTP_HOST")) == "" {
		return fmt.Errorf("PROGRESS_TRACKER_SMTP_HOST is required in production")
	}
	if strings.TrimSpace(os.Getenv("PROGRESS_TRACKER_SMTP_FROM")) == "" {
		return fmt.Errorf("PROGRESS_TRACKER_SMTP_FROM is required in production")
	}
	smtpPort := strings.TrimSpace(os.Getenv("PROGRESS_TRACKER_SMTP_PORT"))
	if smtpPort == "" {
		smtpPort = "587"
	}
	port, err := strconv.Atoi(smtpPort)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("PROGRESS_TRACKER_SMTP_PORT must be a valid port")
	}
	if developmentTimerSpeedEnabled() {
		return fmt.Errorf("PROGRESS_TRACKER_DEV_TIMER_SPEED must be false in production")
	}
	if developmentActionTokensEnabled() {
		return fmt.Errorf("PROGRESS_TRACKER_DEV_ACTION_TOKENS must be false in production")
	}
	return nil
}

func developmentTimerSpeedEnabled() bool {
	// Manual timer acceleration is intentionally hidden from the product UI.
	// It can only be enabled explicitly for isolated development tests.
	return strings.EqualFold(strings.TrimSpace(os.Getenv("PROGRESS_TRACKER_DEV_TIMER_SPEED")), "true")
}

func loopbackDevelopmentHost() bool {
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
	return loopbackDevelopmentHost()
}
