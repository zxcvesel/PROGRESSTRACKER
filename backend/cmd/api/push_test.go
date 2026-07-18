package main

import (
	"context"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
)

type fakePushCall struct {
	Payload      pushPayload
	Subscription storedPushSubscription
	Topic        string
}

type fakePushSender struct {
	Status int
	Err    error
	Calls  []fakePushCall
}

func (sender *fakePushSender) Send(_ context.Context, payload []byte, subscription storedPushSubscription, topic string) (int, error) {
	var decoded pushPayload
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return 0, err
	}
	sender.Calls = append(sender.Calls, fakePushCall{
		Payload:      decoded,
		Subscription: subscription,
		Topic:        topic,
	})
	return sender.Status, sender.Err
}

func TestPushSubscriptionRoutes(t *testing.T) {
	setupTestDatabase(t)
	config := testVAPIDConfig(t)
	pushNotifications = newPushNotificationService(config, &fakePushSender{Status: http.StatusCreated})
	router := newRouter()
	cookie := registerAPIUser(t, router, "push-api@example.com")
	subscription := validTestSubscription(t, "https://fcm.googleapis.com/fcm/send/api-subscription")

	publicKey := apiRequest(t, router, http.MethodGet, "/push/public-key", "", cookie)
	if publicKey.Code != http.StatusOK {
		t.Fatalf("public key status = %d, body = %s", publicKey.Code, publicKey.Body.String())
	}
	var publicKeyResponse map[string]string
	decodeResponse(t, publicKey, &publicKeyResponse)
	if len(publicKeyResponse) != 1 || publicKeyResponse["publicKey"] != config.PublicKey {
		t.Fatalf("public key response = %#v", publicKeyResponse)
	}

	body, err := json.Marshal(subscription)
	if err != nil {
		t.Fatal(err)
	}
	subscribe := apiRequest(t, router, http.MethodPost, "/push/subscriptions", string(body), cookie)
	if subscribe.Code != http.StatusNoContent || subscribe.Body.Len() != 0 {
		t.Fatalf("subscribe status = %d, body = %q", subscribe.Code, subscribe.Body.String())
	}

	var storedEndpoint string
	var storedP256dh string
	var storedAuth string
	var originalUserID int
	if err := db.QueryRow(`
		SELECT user_id, endpoint, p256dh, auth
		FROM push_subscriptions
		WHERE endpoint = ?
	`, subscription.Endpoint).Scan(&originalUserID, &storedEndpoint, &storedP256dh, &storedAuth); err != nil {
		t.Fatal(err)
	}
	if storedEndpoint != subscription.Endpoint || storedP256dh != subscription.Keys.P256dh || storedAuth != subscription.Keys.Auth {
		t.Fatal("stored subscription did not match submitted subscription")
	}
	if _, err := db.Exec(`
		INSERT INTO users (id, email, name, password_hash, created_at, email_verified, timezone)
		VALUES (999, 'other-push@example.com', 'Other User', 'test-hash', ?, 1, 'UTC')
	`, time.Now().UTC().Format(time.RFC3339)); err != nil {
		t.Fatal(err)
	}
	otherToken, err := createAuthSession(999)
	if err != nil {
		t.Fatal(err)
	}
	conflict := apiRequest(t, router, http.MethodPost, "/push/subscriptions", string(body), &http.Cookie{
		Name: authCookieName, Value: otherToken,
	})
	if conflict.Code != http.StatusConflict {
		t.Fatalf("cross-account subscription status = %d, body = %s", conflict.Code, conflict.Body.String())
	}
	var storedUserID int
	if err := db.QueryRow(`SELECT user_id FROM push_subscriptions WHERE endpoint = ?`, subscription.Endpoint).Scan(&storedUserID); err != nil {
		t.Fatal(err)
	}
	if storedUserID != originalUserID {
		t.Fatalf("subscription owner = %d, want %d", storedUserID, originalUserID)
	}

	unsubscribeBody := fmt.Sprintf(`{"endpoint":%q}`, subscription.Endpoint)
	unsubscribe := apiRequest(t, router, http.MethodDelete, "/push/subscriptions", unsubscribeBody, cookie)
	if unsubscribe.Code != http.StatusNoContent || unsubscribe.Body.Len() != 0 {
		t.Fatalf("unsubscribe status = %d, body = %q", unsubscribe.Code, unsubscribe.Body.String())
	}
	var remaining int
	if err := db.QueryRow(`SELECT COUNT(*) FROM push_subscriptions WHERE endpoint = ?`, subscription.Endpoint).Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if remaining != 0 {
		t.Fatalf("subscriptions after unsubscribe = %d, want 0", remaining)
	}
}

func TestPushRoutesRequireVerifiedUser(t *testing.T) {
	setupTestDatabase(t)
	config := testVAPIDConfig(t)
	pushNotifications = newPushNotificationService(config, &fakePushSender{Status: http.StatusCreated})
	router := newRouter()

	if _, err := db.Exec(`
		INSERT INTO users (id, email, name, password_hash, created_at, email_verified, timezone)
		VALUES (2, 'unverified-push@example.com', 'Push User', 'test-hash', ?, 0, 'UTC')
	`, time.Now().UTC().Format(time.RFC3339)); err != nil {
		t.Fatal(err)
	}
	token, err := createAuthSession(2)
	if err != nil {
		t.Fatal(err)
	}
	cookie := &http.Cookie{Name: authCookieName, Value: token}

	response := apiRequest(t, router, http.MethodGet, "/push/public-key", "", cookie)
	if response.Code != http.StatusForbidden {
		t.Fatalf("unverified public key status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestValidatePushSubscription(t *testing.T) {
	valid := validTestSubscription(t, "https://web.push.apple.com/Q/test")
	if message := validatePushSubscription(valid); message != "" {
		t.Fatalf("valid subscription rejected: %s", message)
	}

	for _, endpoint := range []string{
		"http://fcm.googleapis.com/fcm/send/test",
		"https://fcm.googleapis.com.evil.example/fcm/send/test",
		"https://evil.googleapis.com/fcm/send/test",
		"https://fcm.googleapis.com:444/fcm/send/test",
		"https://updates.push.services.mozilla.com/test#fragment",
	} {
		t.Run(endpoint, func(t *testing.T) {
			candidate := valid
			candidate.Endpoint = endpoint
			if message := validatePushSubscription(candidate); message != "invalid push subscription endpoint" {
				t.Fatalf("validation message = %q", message)
			}
		})
	}

	tests := []struct {
		name     string
		p256dh   string
		auth     string
		expected string
	}{
		{"malformed p256dh", "not+base64", valid.Keys.Auth, "invalid push subscription p256dh key"},
		{"oversized p256dh", strings.Repeat("A", maxP256dhKeyLen+1), valid.Keys.Auth, "invalid push subscription p256dh key"},
		{"invalid p256 point", base64.RawURLEncoding.EncodeToString(append([]byte{4}, make([]byte, 64)...)), valid.Keys.Auth, "invalid push subscription p256dh key"},
		{"short auth", valid.Keys.P256dh, base64.RawURLEncoding.EncodeToString(make([]byte, 15)), "invalid push subscription auth key"},
		{"oversized auth", valid.Keys.P256dh, strings.Repeat("A", maxAuthKeyLen+1), "invalid push subscription auth key"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := valid
			candidate.Keys.P256dh = test.p256dh
			candidate.Keys.Auth = test.auth
			if message := validatePushSubscription(candidate); message != test.expected {
				t.Fatalf("validation message = %q, want %q", message, test.expected)
			}
		})
	}
}

func TestLoadPushConfigGeneratesEphemeralKeysOutsideProduction(t *testing.T) {
	t.Setenv("PROGRESS_TRACKER_ENV", "development")
	t.Setenv("PROGRESS_TRACKER_VAPID_SUBJECT", "")
	t.Setenv("PROGRESS_TRACKER_VAPID_PUBLIC_KEY", "")
	t.Setenv("PROGRESS_TRACKER_VAPID_PRIVATE_KEY", "")

	config, err := loadPushConfig()
	if err != nil {
		t.Fatal(err)
	}
	if err := validateVAPIDConfig(config); err != nil {
		t.Fatalf("ephemeral config is invalid: %v", err)
	}
}

func TestValidateVAPIDConfigRejectsMismatchedKeys(t *testing.T) {
	first := testVAPIDConfig(t)
	second := testVAPIDConfig(t)
	first.PrivateKey = second.PrivateKey
	if err := validateVAPIDConfig(first); err == nil {
		t.Fatal("mismatched VAPID keys were accepted")
	}
}

func TestPushWorkerNotifiesCompletedTimerOnce(t *testing.T) {
	setupTestDatabase(t)
	now := time.Date(2026, 7, 18, 10, 0, 0, 0, time.UTC)
	goalID := insertPushTestGoal(t, now.In(time.Local).Format(time.DateOnly), 30)
	insertTestPushSubscription(t, "https://fcm.googleapis.com/fcm/send/timer-once")
	_, err := db.Exec(`
		INSERT INTO active_timers (
			user_id, goal_id, session_date, state, started_at, last_resumed_at,
			accumulated_seconds, target_seconds, speed_multiplier, updated_at
		)
		VALUES (1, ?, ?, 'running', ?, ?, 0, 60, 1, ?)
	`, goalID, now.In(time.Local).Format(time.DateOnly), now.Add(-2*time.Minute).Format(time.RFC3339Nano),
		now.Add(-2*time.Minute).Format(time.RFC3339Nano), now.Add(-2*time.Minute).Format(time.RFC3339Nano))
	if err != nil {
		t.Fatal(err)
	}

	sender := &fakePushSender{Status: http.StatusCreated}
	service := newPushNotificationService(testVAPIDConfig(t), sender)
	if err := service.runCycle(context.Background(), now); err != nil {
		t.Fatal(err)
	}
	if err := service.runCycle(context.Background(), now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if len(sender.Calls) != 1 {
		t.Fatalf("push sends = %d, want 1", len(sender.Calls))
	}
	if sender.Calls[0].Topic != fmt.Sprintf("goal-%d", goalID) || sender.Calls[0].Payload.Tag != fmt.Sprintf("goal-%d-complete", goalID) {
		t.Fatalf("timer collapse values = topic %q, tag %q", sender.Calls[0].Topic, sender.Calls[0].Payload.Tag)
	}
	var notifiedAt string
	if err := db.QueryRow(`SELECT completion_notified_at FROM active_timers WHERE goal_id = ?`, goalID).Scan(&notifiedAt); err != nil {
		t.Fatal(err)
	}
	if notifiedAt == "" {
		t.Fatal("timer completion was not claimed")
	}
}

func TestPushWorkerNotifiesIncompleteDailyTargetOnceAfterLocal20(t *testing.T) {
	setupTestDatabase(t)
	if _, err := db.Exec(`UPDATE users SET timezone = 'Europe/Moscow' WHERE id = 1`); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 18, 17, 1, 0, 0, time.UTC)
	insertPushTestGoal(t, "2026-07-18", 30)
	insertTestPushSubscription(t, "https://updates.push.services.mozilla.com/wpush/v2/daily-once")

	sender := &fakePushSender{Status: http.StatusCreated}
	service := newPushNotificationService(testVAPIDConfig(t), sender)
	if err := service.runCycle(context.Background(), now); err != nil {
		t.Fatal(err)
	}
	if err := service.runCycle(context.Background(), now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if len(sender.Calls) != 1 {
		t.Fatalf("push sends = %d, want 1", len(sender.Calls))
	}
	if sender.Calls[0].Topic != "daily-reminder" || sender.Calls[0].Payload.Tag != "daily-reminder" {
		t.Fatalf("daily collapse values = topic %q, tag %q", sender.Calls[0].Topic, sender.Calls[0].Payload.Tag)
	}
	var claims int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM daily_notification_claims WHERE user_id = 1 AND local_date = '2026-07-18'
	`).Scan(&claims); err != nil {
		t.Fatal(err)
	}
	if claims != 1 {
		t.Fatalf("daily claims = %d, want 1", claims)
	}
}

func TestPushWorkerSkipsCompletedDailyTarget(t *testing.T) {
	setupTestDatabase(t)
	if _, err := db.Exec(`UPDATE users SET timezone = 'Europe/Moscow' WHERE id = 1`); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 18, 17, 1, 0, 0, time.UTC)
	goalID := insertPushTestGoal(t, "2026-07-18", 30)
	insertTestPushSubscription(t, "https://web.push.apple.com/Q/daily-complete")
	if _, err := db.Exec(`
		INSERT INTO sessions (goal_id, started_at, ended_at, duration_minutes, notes, tags, created_at, session_date)
		VALUES (?, ?, ?, 30, '', '', ?, '2026-07-18')
	`, goalID, now.Add(-time.Hour).Format(time.RFC3339), now.Format(time.RFC3339), now.Format(time.RFC3339)); err != nil {
		t.Fatal(err)
	}

	sender := &fakePushSender{Status: http.StatusCreated}
	service := newPushNotificationService(testVAPIDConfig(t), sender)
	if err := service.runCycle(context.Background(), now); err != nil {
		t.Fatal(err)
	}
	if len(sender.Calls) != 0 {
		t.Fatalf("push sends = %d, want 0", len(sender.Calls))
	}
}

func TestPushWorkerRemovesExpiredSubscription(t *testing.T) {
	setupTestDatabase(t)
	now := time.Date(2026, 7, 18, 10, 0, 0, 0, time.UTC)
	goalID := insertPushTestGoal(t, now.In(time.Local).Format(time.DateOnly), 30)
	insertTestPushSubscription(t, "https://fcm.googleapis.com/fcm/send/expired")
	if _, err := db.Exec(`
		INSERT INTO active_timers (
			user_id, goal_id, session_date, state, started_at, last_resumed_at,
			accumulated_seconds, target_seconds, speed_multiplier, updated_at
		)
		VALUES (1, ?, ?, 'finished', ?, ?, 60, 60, 1, ?)
	`, goalID, now.In(time.Local).Format(time.DateOnly), now.Format(time.RFC3339Nano),
		now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}

	sender := &fakePushSender{Status: http.StatusGone}
	service := newPushNotificationService(testVAPIDConfig(t), sender)
	if err := service.runCycle(context.Background(), now); err != nil {
		t.Fatal(err)
	}
	var subscriptions int
	if err := db.QueryRow(`SELECT COUNT(*) FROM push_subscriptions`).Scan(&subscriptions); err != nil {
		t.Fatal(err)
	}
	if subscriptions != 0 {
		t.Fatalf("expired subscriptions = %d, want 0", subscriptions)
	}
}

func testVAPIDConfig(t *testing.T) vapidConfig {
	t.Helper()
	privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		t.Fatal(err)
	}
	return vapidConfig{
		Subject:    "mailto:notifications@progress.example.com",
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}
}

func validTestSubscription(t *testing.T, endpoint string) pushSubscriptionRequest {
	t.Helper()
	_, x, y, err := elliptic.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return pushSubscriptionRequest{
		Endpoint: endpoint,
		Keys: pushSubscriptionKeys{
			P256dh: base64.RawURLEncoding.EncodeToString(elliptic.Marshal(elliptic.P256(), x, y)),
			Auth:   base64.RawURLEncoding.EncodeToString(make([]byte, 16)),
		},
	}
}

func insertTestPushSubscription(t *testing.T, endpoint string) {
	t.Helper()
	subscription := validTestSubscription(t, endpoint)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := db.Exec(`
		INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth, created_at, updated_at)
		VALUES (1, ?, ?, ?, ?, ?)
	`, subscription.Endpoint, subscription.Keys.P256dh, subscription.Keys.Auth, now, now); err != nil {
		t.Fatal(err)
	}
}

func insertPushTestGoal(t *testing.T, startDate string, targetMinutes int) int {
	t.Helper()
	result, err := db.Exec(`
		INSERT INTO goals (
			title, description, total_days, daily_target_minutes,
			active_weekdays, start_date, created_at, status, user_id
		)
		VALUES ('Push goal', '', 30, ?, '1,2,3,4,5,6,7', ?, ?, 'active', 1)
	`, targetMinutes, startDate, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	return int(id)
}
