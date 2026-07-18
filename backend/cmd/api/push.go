package main

import (
	"context"
	"crypto/elliptic"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
)

const (
	pushWorkerInterval = 15 * time.Second
	pushRequestTimeout = 15 * time.Second
	maxPushEndpointLen = 2048
	maxP256dhKeyLen    = 128
	maxAuthKeyLen      = 64
)

var allowedPushHosts = map[string]struct{}{
	"fcm.googleapis.com":                {},
	"push.services.mozilla.com":         {},
	"updates.push.services.mozilla.com": {},
	"web.push.apple.com":                {},
}

type vapidConfig struct {
	Subject    string
	PublicKey  string
	PrivateKey string
}

type pushSubscriptionRequest struct {
	Endpoint string               `json:"endpoint"`
	Keys     pushSubscriptionKeys `json:"keys"`
}

type pushSubscriptionKeys struct {
	P256dh string `json:"p256dh"`
	Auth   string `json:"auth"`
}

type unsubscribePushRequest struct {
	Endpoint string `json:"endpoint"`
}

type storedPushSubscription struct {
	ID       int
	UserID   int
	Endpoint string
	P256dh   string
	Auth     string
}

type pushPayload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Tag   string `json:"tag"`
}

type pushSender interface {
	Send(context.Context, []byte, storedPushSubscription, string) (int, error)
}

type webPushSender struct {
	config vapidConfig
	client webpush.HTTPClient
}

func (sender webPushSender) Send(ctx context.Context, payload []byte, subscription storedPushSubscription, topic string) (int, error) {
	client := sender.client
	if client == nil {
		client = &http.Client{Timeout: pushRequestTimeout}
	}
	response, err := webpush.SendNotificationWithContext(ctx, payload, &webpush.Subscription{
		Endpoint: subscription.Endpoint,
		Keys: webpush.Keys{
			P256dh: subscription.P256dh,
			Auth:   subscription.Auth,
		},
	}, &webpush.Options{
		HTTPClient:      client,
		Subscriber:      sender.config.Subject,
		VAPIDPublicKey:  sender.config.PublicKey,
		VAPIDPrivateKey: sender.config.PrivateKey,
		TTL:             24 * 60 * 60,
		Urgency:         webpush.UrgencyNormal,
		Topic:           topic,
	})
	if response == nil {
		return 0, err
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4<<10))
	return response.StatusCode, err
}

type pushNotificationService struct {
	config vapidConfig
	sender pushSender
}

var pushNotifications *pushNotificationService

func newPushNotificationService(config vapidConfig, sender pushSender) *pushNotificationService {
	return &pushNotificationService{config: config, sender: sender}
}

func configuredVAPIDValues() (vapidConfig, error) {
	config := vapidConfig{
		Subject:    strings.TrimSpace(os.Getenv("PROGRESS_TRACKER_VAPID_SUBJECT")),
		PublicKey:  strings.TrimSpace(os.Getenv("PROGRESS_TRACKER_VAPID_PUBLIC_KEY")),
		PrivateKey: strings.TrimSpace(os.Getenv("PROGRESS_TRACKER_VAPID_PRIVATE_KEY")),
	}
	if config.Subject == "" || config.PublicKey == "" || config.PrivateKey == "" {
		return vapidConfig{}, fmt.Errorf("PROGRESS_TRACKER_VAPID_SUBJECT, PROGRESS_TRACKER_VAPID_PUBLIC_KEY, and PROGRESS_TRACKER_VAPID_PRIVATE_KEY are required in production")
	}
	if err := validateVAPIDConfig(config); err != nil {
		return vapidConfig{}, err
	}
	return config, nil
}

func loadPushConfig() (vapidConfig, error) {
	configured := strings.TrimSpace(os.Getenv("PROGRESS_TRACKER_VAPID_SUBJECT")) != "" ||
		strings.TrimSpace(os.Getenv("PROGRESS_TRACKER_VAPID_PUBLIC_KEY")) != "" ||
		strings.TrimSpace(os.Getenv("PROGRESS_TRACKER_VAPID_PRIVATE_KEY")) != ""
	if configured || strings.EqualFold(strings.TrimSpace(os.Getenv("PROGRESS_TRACKER_ENV")), "production") {
		return configuredVAPIDValues()
	}

	privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		return vapidConfig{}, fmt.Errorf("generate ephemeral VAPID keys: %w", err)
	}
	return vapidConfig{
		Subject:    "mailto:notifications@progress-tracker.local",
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}, nil
}

func validateVAPIDConfig(config vapidConfig) error {
	if len(config.Subject) > 256 {
		return fmt.Errorf("PROGRESS_TRACKER_VAPID_SUBJECT is invalid")
	}
	subject, err := url.Parse(config.Subject)
	if err != nil || subject.Fragment != "" {
		return fmt.Errorf("PROGRESS_TRACKER_VAPID_SUBJECT is invalid")
	}
	switch subject.Scheme {
	case "mailto":
		if subject.Opaque == "" || strings.ContainsAny(subject.Opaque, "\r\n") {
			return fmt.Errorf("PROGRESS_TRACKER_VAPID_SUBJECT is invalid")
		}
	case "https":
		if subject.Host == "" || subject.User != nil {
			return fmt.Errorf("PROGRESS_TRACKER_VAPID_SUBJECT is invalid")
		}
	default:
		return fmt.Errorf("PROGRESS_TRACKER_VAPID_SUBJECT must use mailto or HTTPS")
	}

	publicKey, err := decodeBoundedBase64URL(config.PublicKey, maxP256dhKeyLen)
	if err != nil || len(publicKey) != 65 || publicKey[0] != 4 {
		return fmt.Errorf("PROGRESS_TRACKER_VAPID_PUBLIC_KEY is invalid")
	}
	x, y := elliptic.Unmarshal(elliptic.P256(), publicKey)
	if x == nil || y == nil {
		return fmt.Errorf("PROGRESS_TRACKER_VAPID_PUBLIC_KEY is invalid")
	}

	privateKey, err := decodeBoundedBase64URL(config.PrivateKey, maxAuthKeyLen)
	if err != nil || len(privateKey) != 32 {
		return fmt.Errorf("PROGRESS_TRACKER_VAPID_PRIVATE_KEY is invalid")
	}
	privateValue := new(big.Int).SetBytes(privateKey)
	if privateValue.Sign() <= 0 || privateValue.Cmp(elliptic.P256().Params().N) >= 0 {
		return fmt.Errorf("PROGRESS_TRACKER_VAPID_PRIVATE_KEY is invalid")
	}
	derivedX, derivedY := elliptic.P256().ScalarBaseMult(privateKey)
	if subtle.ConstantTimeCompare(publicKey, elliptic.Marshal(elliptic.P256(), derivedX, derivedY)) != 1 {
		return fmt.Errorf("PROGRESS_TRACKER_VAPID keys do not form a valid pair")
	}
	return nil
}

func pushPublicKeyHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := currentVerifiedUserFromRequest(w, r); !ok {
		return
	}
	if pushNotifications == nil {
		writeError(w, "push notifications are unavailable", http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, map[string]string{"publicKey": pushNotifications.config.PublicKey}, http.StatusOK)
}

func subscribePushHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := currentVerifiedUserFromRequest(w, r)
	if !ok {
		return
	}
	var request pushSubscriptionRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if message := validatePushSubscription(request); message != "" {
		writeError(w, message, http.StatusBadRequest)
		return
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	result, err := db.Exec(`
		INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(endpoint) DO UPDATE SET
			p256dh = excluded.p256dh,
			auth = excluded.auth,
			updated_at = excluded.updated_at
		WHERE push_subscriptions.user_id = excluded.user_id
	`, user.ID, request.Endpoint, request.Keys.P256dh, request.Keys.Auth, now, now)
	if err != nil {
		writeError(w, "failed to save push subscription", http.StatusInternalServerError)
		return
	}
	updated, err := result.RowsAffected()
	if err != nil {
		writeError(w, "failed to save push subscription", http.StatusInternalServerError)
		return
	}
	if updated != 1 {
		writeError(w, "push subscription belongs to another account", http.StatusConflict)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func unsubscribePushHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := currentVerifiedUserFromRequest(w, r)
	if !ok {
		return
	}
	var request unsubscribePushRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if message := validatePushEndpoint(request.Endpoint); message != "" {
		writeError(w, message, http.StatusBadRequest)
		return
	}
	if _, err := db.Exec(`DELETE FROM push_subscriptions WHERE user_id = ? AND endpoint = ?`, user.ID, request.Endpoint); err != nil {
		writeError(w, "failed to remove push subscription", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func validatePushSubscription(subscription pushSubscriptionRequest) string {
	if message := validatePushEndpoint(subscription.Endpoint); message != "" {
		return message
	}
	p256dh, err := decodeBoundedBase64URL(subscription.Keys.P256dh, maxP256dhKeyLen)
	if err != nil || len(p256dh) != 65 || p256dh[0] != 4 {
		return "invalid push subscription p256dh key"
	}
	x, y := elliptic.Unmarshal(elliptic.P256(), p256dh)
	if x == nil || y == nil {
		return "invalid push subscription p256dh key"
	}
	auth, err := decodeBoundedBase64URL(subscription.Keys.Auth, maxAuthKeyLen)
	if err != nil || len(auth) != 16 {
		return "invalid push subscription auth key"
	}
	return ""
}

func validatePushEndpoint(endpoint string) string {
	if endpoint == "" || len(endpoint) > maxPushEndpointLen || strings.TrimSpace(endpoint) != endpoint {
		return "invalid push subscription endpoint"
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme != "https" || parsed.User != nil || parsed.Host == "" || parsed.Fragment != "" {
		return "invalid push subscription endpoint"
	}
	if parsed.Port() != "" && parsed.Port() != "443" {
		return "invalid push subscription endpoint"
	}
	if _, allowed := allowedPushHosts[strings.ToLower(parsed.Hostname())]; !allowed {
		return "invalid push subscription endpoint"
	}
	return ""
}

func decodeBoundedBase64URL(value string, maximum int) ([]byte, error) {
	if value == "" || len(value) > maximum || strings.Contains(value, "=") {
		return nil, errors.New("invalid base64url value")
	}
	return base64.RawURLEncoding.Strict().DecodeString(value)
}

func (service *pushNotificationService) run(ctx context.Context) {
	ticker := time.NewTicker(pushWorkerInterval)
	defer ticker.Stop()
	for {
		if err := service.runCycle(ctx, time.Now().UTC()); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("push notification worker: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (service *pushNotificationService) runCycle(ctx context.Context, now time.Time) error {
	return errors.Join(
		service.notifyCompletedTimers(ctx, now.UTC()),
		service.notifyIncompleteDailyTargets(ctx, now.UTC()),
	)
}

func (service *pushNotificationService) notifyCompletedTimers(ctx context.Context, now time.Time) error {
	rows, err := db.Query(`
		SELECT id, user_id, goal_id, session_date, state, started_at, last_resumed_at,
			accumulated_seconds, target_seconds, speed_multiplier
		FROM active_timers
		WHERE completion_notified_at IS NULL
	`)
	if err != nil {
		return err
	}
	var timers []activeTimerRecord
	for rows.Next() {
		var timer activeTimerRecord
		if err := rows.Scan(
			&timer.ID, &timer.UserID, &timer.GoalID, &timer.SessionDate, &timer.State,
			&timer.StartedAt, &timer.LastResumedAt, &timer.AccumulatedSeconds,
			&timer.TargetSeconds, &timer.SpeedMultiplier,
		); err != nil {
			rows.Close()
			return err
		}
		timers = append(timers, timer)
	}
	if err := rows.Close(); err != nil {
		return err
	}

	var sendErrors []error
	for _, timer := range timers {
		state, _, err := timerState(timer, now)
		if err != nil {
			sendErrors = append(sendErrors, fmt.Errorf("read timer %d: %w", timer.ID, err))
			continue
		}
		if state.State != "finished" {
			continue
		}
		claimedAt := now.Format(time.RFC3339Nano)
		result, err := db.Exec(`
			UPDATE active_timers
			SET state = 'finished', accumulated_seconds = target_seconds,
				updated_at = ?, completion_notified_at = ?
			WHERE id = ? AND completion_notified_at IS NULL
		`, claimedAt, claimedAt, timer.ID)
		if err != nil {
			sendErrors = append(sendErrors, fmt.Errorf("claim timer %d: %w", timer.ID, err))
			continue
		}
		claimed, err := result.RowsAffected()
		if err != nil || claimed != 1 {
			continue
		}
		tag := fmt.Sprintf("goal-%d-complete", timer.GoalID)
		if err := service.sendToUser(ctx, timer.UserID, pushPayload{
			Title: "Timer complete",
			Body:  "Your practice timer reached its target.",
			Tag:   tag,
		}, fmt.Sprintf("goal-%d", timer.GoalID)); err != nil {
			sendErrors = append(sendErrors, err)
		}
	}
	return errors.Join(sendErrors...)
}

func (service *pushNotificationService) notifyIncompleteDailyTargets(ctx context.Context, now time.Time) error {
	rows, err := db.Query(`
		SELECT users.id, users.timezone
		FROM users
		WHERE users.email_verified = 1
			AND EXISTS (SELECT 1 FROM push_subscriptions WHERE push_subscriptions.user_id = users.id)
	`)
	if err != nil {
		return err
	}
	type userTimezone struct {
		ID       int
		Timezone string
	}
	var users []userTimezone
	for rows.Next() {
		var user userTimezone
		if err := rows.Scan(&user.ID, &user.Timezone); err != nil {
			rows.Close()
			return err
		}
		users = append(users, user)
	}
	if err := rows.Close(); err != nil {
		return err
	}

	var sendErrors []error
	for _, user := range users {
		location, err := time.LoadLocation(user.Timezone)
		if err != nil {
			location = time.UTC
		}
		localNow := now.In(location)
		if localNow.Hour() < 20 {
			continue
		}
		localDate := localNow.Format(time.DateOnly)
		incomplete, err := userHasIncompleteDailyTarget(user.ID, localDate, isoWeekday(localNow))
		if err != nil {
			sendErrors = append(sendErrors, fmt.Errorf("check daily target for user %d: %w", user.ID, err))
			continue
		}
		if !incomplete {
			continue
		}
		result, err := db.Exec(`
			INSERT OR IGNORE INTO daily_notification_claims (user_id, local_date, claimed_at)
			VALUES (?, ?, ?)
		`, user.ID, localDate, now.Format(time.RFC3339Nano))
		if err != nil {
			sendErrors = append(sendErrors, fmt.Errorf("claim daily notification for user %d: %w", user.ID, err))
			continue
		}
		claimed, err := result.RowsAffected()
		if err != nil || claimed != 1 {
			continue
		}
		if err := service.sendToUser(ctx, user.ID, pushPayload{
			Title: "Daily target incomplete",
			Body:  "You still have practice left for today.",
			Tag:   "daily-reminder",
		}, "daily-reminder"); err != nil {
			sendErrors = append(sendErrors, err)
		}
	}
	return errors.Join(sendErrors...)
}

func userHasIncompleteDailyTarget(userID int, localDate string, weekday int) (bool, error) {
	var incomplete int
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM goals
			WHERE user_id = ?
				AND status = 'active'
				AND date(?) BETWEEN date(start_date) AND date(start_date, '+' || (total_days - 1) || ' days')
				AND instr(',' || active_weekdays || ',', ',' || ? || ',') > 0
				AND COALESCE((
					SELECT SUM(duration_minutes)
					FROM sessions
					WHERE sessions.goal_id = goals.id AND sessions.session_date = ?
				), 0) < daily_target_minutes
		)
	`, userID, localDate, weekday, localDate).Scan(&incomplete)
	return incomplete == 1, err
}

func isoWeekday(value time.Time) int {
	if value.Weekday() == time.Sunday {
		return 7
	}
	return int(value.Weekday())
}

func (service *pushNotificationService) sendToUser(ctx context.Context, userID int, payload pushPayload, topic string) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	subscriptions, err := loadPushSubscriptions(userID)
	if err != nil {
		return err
	}
	var sendErrors []error
	for _, subscription := range subscriptions {
		status, err := service.sender.Send(ctx, encoded, subscription, topic)
		if status == http.StatusNotFound || status == http.StatusGone {
			if _, deleteErr := db.Exec(`DELETE FROM push_subscriptions WHERE id = ?`, subscription.ID); deleteErr != nil {
				sendErrors = append(sendErrors, fmt.Errorf("remove expired push subscription %d: %w", subscription.ID, deleteErr))
			}
			continue
		}
		if err != nil {
			sendErrors = append(sendErrors, fmt.Errorf("send push subscription %d for user %d: %w", subscription.ID, userID, err))
			continue
		}
		if status < 200 || status >= 300 {
			sendErrors = append(sendErrors, fmt.Errorf("push subscription %d for user %d returned status %d", subscription.ID, userID, status))
		}
	}
	return errors.Join(sendErrors...)
}

func loadPushSubscriptions(userID int) ([]storedPushSubscription, error) {
	rows, err := db.Query(`
		SELECT id, user_id, endpoint, p256dh, auth
		FROM push_subscriptions
		WHERE user_id = ?
		ORDER BY id
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var subscriptions []storedPushSubscription
	for rows.Next() {
		var subscription storedPushSubscription
		if err := rows.Scan(&subscription.ID, &subscription.UserID, &subscription.Endpoint, &subscription.P256dh, &subscription.Auth); err != nil {
			return nil, err
		}
		subscriptions = append(subscriptions, subscription)
	}
	return subscriptions, rows.Err()
}
