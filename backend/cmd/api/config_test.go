package main

import (
	"testing"

	webpush "github.com/SherClockHolmes/webpush-go"
)

func setValidProductionConfig(t *testing.T) {
	t.Helper()
	t.Setenv("PROGRESS_TRACKER_ENV", "production")
	t.Setenv("PROGRESS_TRACKER_SECURE_COOKIES", "true")
	t.Setenv("PROGRESS_TRACKER_ALLOWED_ORIGINS", "https://progress.example.com")
	t.Setenv("PROGRESS_TRACKER_PUBLIC_URL", "https://progress.example.com")
	t.Setenv("PROGRESS_TRACKER_SMTP_HOST", "smtp.example.com")
	t.Setenv("PROGRESS_TRACKER_SMTP_PORT", "587")
	t.Setenv("PROGRESS_TRACKER_SMTP_FROM", "no-reply@progress.example.com")
	t.Setenv("PROGRESS_TRACKER_DEV_TIMER_SPEED", "false")
	t.Setenv("PROGRESS_TRACKER_DEV_ACTION_TOKENS", "false")
	privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PROGRESS_TRACKER_VAPID_SUBJECT", "mailto:notifications@progress.example.com")
	t.Setenv("PROGRESS_TRACKER_VAPID_PUBLIC_KEY", publicKey)
	t.Setenv("PROGRESS_TRACKER_VAPID_PRIVATE_KEY", privateKey)
}

func TestValidateRuntimeConfigAcceptsSecureProductionConfig(t *testing.T) {
	setValidProductionConfig(t)
	if err := validateRuntimeConfig(); err != nil {
		t.Fatalf("valid production config rejected: %v", err)
	}
}

func TestValidateRuntimeConfigRejectsUnsafeProductionConfig(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{"insecure origin", "PROGRESS_TRACKER_ALLOWED_ORIGINS", "http://progress.example.com"},
		{"insecure public URL", "PROGRESS_TRACKER_PUBLIC_URL", "http://progress.example.com"},
		{"missing SMTP host", "PROGRESS_TRACKER_SMTP_HOST", ""},
		{"missing SMTP sender", "PROGRESS_TRACKER_SMTP_FROM", ""},
		{"invalid SMTP port", "PROGRESS_TRACKER_SMTP_PORT", "70000"},
		{"development timer", "PROGRESS_TRACKER_DEV_TIMER_SPEED", "true"},
		{"development action tokens", "PROGRESS_TRACKER_DEV_ACTION_TOKENS", "true"},
		{"missing VAPID public key", "PROGRESS_TRACKER_VAPID_PUBLIC_KEY", ""},
		{"invalid VAPID subject", "PROGRESS_TRACKER_VAPID_SUBJECT", "http://progress.example.com"},
		{"invalid VAPID private key", "PROGRESS_TRACKER_VAPID_PRIVATE_KEY", "not-base64url"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setValidProductionConfig(t)
			t.Setenv(test.key, test.value)
			if err := validateRuntimeConfig(); err == nil {
				t.Fatal("unsafe production config was accepted")
			}
		})
	}
}
