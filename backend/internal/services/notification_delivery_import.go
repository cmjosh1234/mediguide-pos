package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"

	"mediguide/internal/models"

	"google.golang.org/api/bigquery/v2"
	"google.golang.org/api/option"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var fcmExportTable = regexp.MustCompile(`^[a-z][a-z0-9-]{4,62}\.[A-Za-z_][A-Za-z0-9_]*\.[A-Za-z_][A-Za-z0-9_]*$`)
var fcmProviderMessageID = regexp.MustCompile(`^[A-Za-z0-9_%:+.=-]{1,255}$`)

type FirebaseDeliveryImportEvent struct {
	MessageID      string    `json:"message_id"`
	EventTimestamp time.Time `json:"event_timestamp"`
	SDKPlatform    string    `json:"sdk_platform"`
	Event          string    `json:"event"`
}

// Only trusted Firebase export data can confirm delivery. This is deliberately
// not a client-facing endpoint. Replays cannot inflate counts or downgrade opens.
func (s NotificationOutboxService) ImportFirebaseDeliveryEvent(project string, event FirebaseDeliveryImportEvent) (int, error) {
	if event.Event != "MESSAGE_DELIVERED" {
		return 0, nil
	}
	platform := strings.ToLower(event.SDKPlatform)
	if !oneOf(platform, "android", "ios") || !fcmProviderMessageID.MatchString(event.MessageID) ||
		event.EventTimestamp.IsZero() || event.EventTimestamp.After(s.now().Add(5*time.Minute)) {
		return 0, ErrNotificationInvalid
	}
	count := 0
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		var deliveries []models.NotificationDelivery
		providerID := "projects/" + project + "/messages/" + event.MessageID
		if err := tx.Model(&models.NotificationDelivery{}).
			Joins("JOIN firebase_devices fd ON fd.id = notification_deliveries.firebase_device_id AND fd.user_id = notification_deliveries.user_id AND LOWER(fd.platform) = ?", platform).
			Where("notification_deliveries.channel = 'push' AND (notification_deliveries.provider_message_id = ? OR notification_deliveries.provider_message_id = ?)", providerID, event.MessageID).
			Clauses(clause.Locking{Strength: "UPDATE", Table: clause.Table{Name: "notification_deliveries"}}).
			Find(&deliveries).Error; err != nil {
			return err
		}
		// Firebase IDs are not guaranteed globally unique. A collision spanning
		// multiple installations is unknown delivery until instance-ID matching is
		// available; never credit the wrong installation. Digest members share one.
		devices := map[string]bool{}
		for _, delivery := range deliveries {
			if delivery.FirebaseDeviceID != nil {
				devices[delivery.FirebaseDeviceID.String()] = true
			}
		}
		if len(devices) > 1 {
			return nil
		}
		for _, delivery := range deliveries {
			if event.EventTimestamp.Before(delivery.CreatedAt.Add(-5 * time.Minute)) {
				continue
			}
			hash := sha256.Sum256([]byte(providerID + ":" + platform + ":" + event.EventTimestamp.UTC().Format(time.RFC3339Nano)))
			audit := models.NotificationDeliveryEvent{DeliveryID: delivery.ID, UserID: delivery.UserID, EventID: "fcm-" + hex.EncodeToString(hash[:]), EventType: "delivered", OccurredAt: event.EventTimestamp}
			created := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&audit)
			if created.Error != nil {
				return created.Error
			}
			if created.RowsAffected == 0 {
				continue
			}
			updates := map[string]any{"updated_at": s.now()}
			if delivery.DeliveredAt == nil || event.EventTimestamp.Before(*delivery.DeliveredAt) {
				updates["delivered_at"] = event.EventTimestamp.UTC()
			}
			if oneOf(delivery.State, "queued", "attempted", "accepted") {
				updates["state"] = "delivered"
			}
			if err := tx.Model(&models.NotificationDelivery{}).Where("id = ?", delivery.ID).Updates(updates).Error; err != nil {
				return err
			}
			count++
		}
		return nil
	})
	return count, err
}

// ImportFirebaseDeliveryExport uses a seven-day overlap because Firebase export
// is delayed. The audit-event unique index makes repeated windows idempotent.
func (s NotificationOutboxService) ImportFirebaseDeliveryExport(ctx context.Context, table, appName, encodedCredentials string) (int, error) {
	if !fcmExportTable.MatchString(table) || strings.TrimSpace(appName) == "" {
		return 0, ErrNotificationInvalid
	}
	project := strings.Split(table, ".")[0]
	if s.Firebase != nil && s.Firebase.Project != "" && s.Firebase.Project != project {
		return 0, ErrNotificationInvalid
	}
	appNames := []*bigquery.QueryParameterValue{}
	for _, name := range strings.Split(appName, ",") {
		name = strings.TrimSpace(name)
		if name == "" || len(name) > 255 || len(appNames) >= 10 {
			return 0, ErrNotificationInvalid
		}
		appNames = append(appNames, &bigquery.QueryParameterValue{Value: name})
	}
	options := []option.ClientOption{option.WithScopes(bigquery.BigqueryScope)}
	if encodedCredentials != "" {
		raw, _, err := decodeFirebaseServiceAccount(encodedCredentials)
		if err != nil {
			return 0, err
		}
		options = append(options, option.WithCredentialsJSON(raw))
	}
	client, err := bigquery.NewService(ctx, options...)
	if err != nil {
		return 0, err
	}
	query := fmt.Sprintf("SELECT message_id, FORMAT_TIMESTAMP('%%Y-%%m-%%dT%%H:%%M:%%E*SZ', event_timestamp, 'UTC') AS event_timestamp, sdk_platform FROM `%s` WHERE _PARTITIONTIME >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 7 DAY) AND event = 'MESSAGE_DELIVERED' AND app_name IN UNNEST(@app_names)", table)
	useLegacySQL := false
	response, err := client.Jobs.Query(project, &bigquery.QueryRequest{Query: query, UseLegacySql: &useLegacySQL,
		ParameterMode: "NAMED", QueryParameters: []*bigquery.QueryParameter{{Name: "app_names", ParameterType: &bigquery.QueryParameterType{Type: "ARRAY", ArrayType: &bigquery.QueryParameterType{Type: "STRING"}}, ParameterValue: &bigquery.QueryParameterValue{ArrayValues: appNames}}},
		MaximumBytesBilled: 1 << 30, TimeoutMs: 10000, MaxResults: 1000}).Context(ctx).Do()
	if err != nil {
		return 0, err
	}
	rows := response.Rows
	token := response.PageToken
	complete := response.JobComplete
	matched := 0
	for {
		if complete {
			for _, row := range rows {
				if len(row.F) != 3 {
					return matched, fmt.Errorf("unexpected Firebase export schema")
				}
				value := func(i int) string { return fmt.Sprint(row.F[i].V) }
				at, err := time.Parse(time.RFC3339Nano, value(1))
				if err != nil {
					return matched, err
				}
				count, err := s.ImportFirebaseDeliveryEvent(project, FirebaseDeliveryImportEvent{MessageID: value(0), EventTimestamp: at, SDKPlatform: value(2), Event: "MESSAGE_DELIVERED"})
				if err != nil {
					return matched, err
				}
				matched += count
			}
			if token == "" {
				return matched, nil
			}
		}
		if response.JobReference == nil {
			return matched, fmt.Errorf("Firebase export query has no job reference")
		}
		page, err := client.Jobs.GetQueryResults(project, response.JobReference.JobId).Location(response.JobReference.Location).
			PageToken(token).MaxResults(1000).TimeoutMs(10000).Context(ctx).Do()
		if err != nil {
			return matched, err
		}
		rows = page.Rows
		token = page.PageToken
		complete = page.JobComplete
	}
}
