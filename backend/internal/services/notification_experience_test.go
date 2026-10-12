package services

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"mediguide/internal/models"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

func experienceCampaign(t *testing.T, s NotificationService, title string, priority string, expires *time.Time) models.NotificationCampaign {
	t.Helper()
	action, _ := json.Marshal(NotificationAction{Type: NotificationActionInternalRoute, Route: stringPointer("/notifications"), Parameters: map[string]string{}})
	campaign := models.NotificationCampaign{
		CampaignVariablesJSON: datatypes.JSON([]byte(`{}`)), DispatchSnapshotJSON: datatypes.JSON([]byte(`{}`)),
		ChannelsJSON: datatypes.JSON([]byte(`["push","in-app"]`)), AudienceCountriesJSON: datatypes.JSON([]byte(`[]`)), AudienceRolesJSON: datatypes.JSON([]byte(`[]`)), Name: title, Type: "update", Status: "queued", RenderedTitle: title, RenderedBody: "Read the update", Priority: priority,
		Timezone: "UTC", IdempotencyKey: uuid.NewString(), LockVersion: 1, ExpiresAt: expires,
		AudienceDefinitionJSON: datatypes.JSON([]byte(`{"all_eligible":true}`)), ActionSnapshotJSON: datatypes.JSON(action), RequestedChannelsJSON: datatypes.JSON([]byte(`["push","in-app"]`))}
	if err := s.DB.Create(&campaign).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := s.prepareCampaignDispatch(s.DB, &campaign); err != nil {
		t.Fatal(err)
	}
	if err := s.DB.Model(&models.NotificationOutboxJob{}).Where("campaign_id = ?", campaign.ID).Update("status", "pending").Error; err != nil {
		t.Fatal(err)
	}
	return campaign
}

func TestRoutineDigestCombinesUpdatesAndUrgentAlertsRemainImmediate(t *testing.T) {
	s := notificationTestService(t)
	var user models.User
	s.DB.First(&user)
	now := time.Now().UTC()
	enabled := true
	clock := now.Add(30 * time.Minute).Format("15:04")
	if _, err := s.UpdatePreferences(user.ID, NotificationPreferencesInput{RoutineDigest: &enabled, DigestTime: &clock}); err != nil {
		t.Fatal(err)
	}
	experienceCampaign(t, s, "Update one", "normal", nil)
	experienceCampaign(t, s, "Update two", "low", nil)
	experienceCampaign(t, s, "Urgent alert", "urgent", nil)
	mock := &firebaseMockMessenger{results: []firebaseMockResult{{id: "messages/urgent"}, {id: "messages/digest"}}}
	firebase := &FirebaseService{DB: s.DB, Project: "test-project", Messenger: mock}
	worker := NotificationOutboxService{DB: s.DB, Firebase: firebase, WorkerID: "test-worker", MaxConcurrency: 1, Now: func() time.Time { return now }}
	now = now.Add(time.Second)
	if _, err := worker.ProcessBatch(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(mock.messages) != 1 || mock.messages[0].Notification.Title != "Urgent alert" {
		t.Fatalf("routine messages interrupted: %#v", mock.messages)
	}
	if result, err := worker.ProcessDigests(t.Context()); err != nil || result.Claimed != 0 {
		t.Fatalf("early digest: %#v %v", result, err)
	}
	now = now.Add(31 * time.Minute)
	result, err := worker.ProcessDigests(t.Context())
	if err != nil || result.Accepted != 1 || len(mock.messages) != 2 {
		t.Fatalf("digest: %#v %v messages=%d", result, err, len(mock.messages))
	}
	if mock.messages[1].Notification.Body != "2 routine updates are ready in your inbox." {
		t.Fatalf("wrong summary: %#v", mock.messages[1])
	}
	if result, err := worker.ProcessDigests(t.Context()); err != nil || result.Claimed != 0 {
		t.Fatalf("digest was sent twice: %#v %v", result, err)
	}
	var inboxCount int64
	s.DB.Model(&models.Notification{}).Count(&inboxCount)
	if inboxCount != 3 {
		t.Fatalf("digest lost inbox items: %d", inboxCount)
	}
}

func TestDigestRechecksOptOutAndExpiresOldMembers(t *testing.T) {
	for _, optOut := range []bool{false, true} {
		t.Run(map[bool]string{false: "expired", true: "opted-out"}[optOut], func(t *testing.T) {
			s := notificationTestService(t)
			var user models.User
			s.DB.First(&user)
			now := time.Now().UTC()
			clock := now.Add(30 * time.Minute).Format("15:04")
			yes := true
			if _, err := s.UpdatePreferences(user.ID, NotificationPreferencesInput{RoutineDigest: &yes, DigestTime: &clock}); err != nil {
				t.Fatal(err)
			}
			expires := now.Add(5 * time.Minute)
			if optOut {
				expires = now.Add(24 * time.Hour)
			}
			experienceCampaign(t, s, "Old update", "normal", &expires)
			if optOut {
				no := false
				if _, err := s.UpdatePreferences(user.ID, NotificationPreferencesInput{PushEnabled: &no}); err != nil {
					t.Fatal(err)
				}
			}
			mock := &firebaseMockMessenger{}
			worker := NotificationOutboxService{DB: s.DB, Firebase: &FirebaseService{DB: s.DB, Project: "test-project", Messenger: mock}, WorkerID: "test", Now: func() time.Time { return now }}
			now = now.Add(31 * time.Minute)
			if _, err := worker.ProcessDigests(t.Context()); err != nil {
				t.Fatal(err)
			}
			if len(mock.messages) != 0 {
				t.Fatal("expired or opted-out digest was sent")
			}
			var job models.NotificationOutboxJob
			s.DB.Where("channel = 'push'").First(&job)
			if job.Status != "failed" {
				t.Fatalf("member lifecycle not closed: %#v", job)
			}
		})
	}
}

func TestNotificationInterestsPersistAndFilterWithoutBlockingUrgentAlerts(t *testing.T) {
	s := notificationTestService(t)
	var user models.User
	s.DB.First(&user)
	region := uuid.New()
	otherRegion := uuid.New()
	document := models.GuidelineDocument{Title: "Malaria", ProgramArea: "Malaria"}
	s.DB.Create(&document)
	outbreak := models.Outbreak{Title: "Local", RegionID: &otherRegion}
	s.DB.Create(&outbreak)
	interests := models.NotificationInterests{ProgramAreas: []string{"HIV"}, RegionIDs: []string{region.String()}, FollowedGuidelineIDs: []string{document.ID.String()}, FollowedOnly: true}
	prefs, err := s.UpdatePreferences(user.ID, NotificationPreferencesInput{Interests: &interests})
	if err != nil {
		t.Fatal(err)
	}
	if len(prefs.Interests.FollowedGuidelineIDs) != 1 {
		t.Fatal("interests not persisted")
	}
	no := false
	if _, err := s.UpdatePreferences(user.ID, NotificationPreferencesInput{ProductAnnouncements: &no}); err != nil {
		t.Fatal(err)
	}
	var settings models.NotificationPreferenceSettings
	s.DB.Where("user_id = ?", user.ID).First(&settings)
	for _, action := range []NotificationAction{{Type: NotificationActionGuideline, ResourceID: stringPointer(document.ID.String())}, {Type: NotificationActionOutbreak, ResourceID: stringPointer(outbreak.ID.String())}} {
		matches, err := notificationMatchesInterests(s.DB, settings, action, "normal")
		if err != nil || matches != (action.Type == NotificationActionGuideline) {
			t.Fatalf("routine interest filter: %v %v", matches, err)
		}
		matches, err = notificationMatchesInterests(s.DB, settings, action, "urgent")
		if err != nil || !matches {
			t.Fatalf("urgent blocked: %v %v", matches, err)
		}
	}
	bad := models.NotificationInterests{RegionIDs: []string{"not-a-region"}}
	if _, err := s.UpdatePreferences(user.ID, NotificationPreferencesInput{Interests: &bad}); err != ErrNotificationInvalid {
		t.Fatalf("invalid interests accepted: %v", err)
	}
}

func TestFirebaseDeliveryImportIsReplaySafeAndPreservesClicks(t *testing.T) {
	s := notificationTestService(t)
	campaign := experienceCampaign(t, s, "Report", "normal", nil)
	var job models.NotificationOutboxJob
	s.DB.Where("campaign_id = ? AND channel = 'push'", campaign.ID).First(&job)
	var delivery models.NotificationDelivery
	s.DB.Where("outbox_job_id = ?", job.ID).First(&delivery)
	now := time.Now().UTC()
	provider := "projects/test-project/messages/123:abc%def"
	s.DB.Model(&delivery).Updates(map[string]any{"state": "accepted", "provider_message_id": provider, "accepted_at": now})
	worker := NotificationOutboxService{DB: s.DB, Now: func() time.Time { return now.Add(time.Minute) }}
	event := FirebaseDeliveryImportEvent{MessageID: "123:abc%def", EventTimestamp: now, SDKPlatform: "ANDROID", Event: "MESSAGE_DELIVERED"}
	if count, err := worker.ImportFirebaseDeliveryEvent("test-project", event); err != nil || count != 1 {
		t.Fatalf("import: %d %v", count, err)
	}
	if count, err := worker.ImportFirebaseDeliveryEvent("test-project", event); err != nil || count != 0 {
		t.Fatalf("replay: %d %v", count, err)
	}
	if _, err := worker.RecordDeliveryEvent(job.UserID, delivery.ID, "clicked", NotificationDeliveryEventInput{EventID: "clicked-after-delivery"}); err != nil {
		t.Fatal(err)
	}
	event.EventTimestamp = now.Add(30 * time.Second)
	if _, err := worker.ImportFirebaseDeliveryEvent("test-project", event); err != nil {
		t.Fatal(err)
	}
	s.DB.First(&delivery, "id = ?", delivery.ID)
	if delivery.State != "clicked" || delivery.DeliveredAt == nil {
		t.Fatalf("import downgraded click: %#v", delivery)
	}
	event.SDKPlatform = "IOS"
	if count, err := worker.ImportFirebaseDeliveryEvent("test-project", event); err != nil || count != 0 {
		t.Fatalf("wrong platform matched: %d %v", count, err)
	}
	if count, err := worker.ImportFirebaseDeliveryEvent("other-project", FirebaseDeliveryImportEvent{MessageID: event.MessageID, EventTimestamp: event.EventTimestamp, SDKPlatform: "ANDROID", Event: event.Event}); err != nil || count != 0 {
		t.Fatalf("wrong project matched: %d %v", count, err)
	}
}

func TestDigestDisabledReleasesPendingJobsForImmediateDelivery(t *testing.T) {
	s := notificationTestService(t)
	var user models.User
	s.DB.First(&user)
	yes := true
	if _, err := s.UpdatePreferences(user.ID, NotificationPreferencesInput{RoutineDigest: &yes}); err != nil {
		t.Fatal(err)
	}
	experienceCampaign(t, s, "Pending routine update", "normal", nil)
	no := false
	if _, err := s.UpdatePreferences(user.ID, NotificationPreferencesInput{RoutineDigest: &no}); err != nil {
		t.Fatal(err)
	}
	var job models.NotificationOutboxJob
	s.DB.Where("channel = 'push'").First(&job)
	if job.DigestID != nil {
		t.Fatal("turning digest off stranded a pending update")
	}
}

func TestNotificationMetricsExposeDueWorkAndImportHealth(t *testing.T) {
	s := notificationTestService(t)
	experienceCampaign(t, s, "Due", "normal", nil)
	worker := NotificationOutboxService{DB: s.DB, Now: func() time.Time { return time.Now().UTC().Add(time.Minute) }}
	health := &NotificationImportHealth{}
	health.Record(nil)
	recorder := httptest.NewRecorder()
	worker.MetricsHandler(true, health)(recorder, httptest.NewRequest("GET", "/metrics", nil))
	if recorder.Code != 200 || !strings.Contains(recorder.Body.String(), "mediguide_notification_due_jobs 2") || !strings.Contains(recorder.Body.String(), "mediguide_notification_delivery_import_enabled 1") {
		t.Fatalf("unexpected metrics: %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestLateScheduledDigestMemberReopensCompletedBatch(t *testing.T) {
	s := notificationTestService(t)
	var user models.User
	s.DB.First(&user)
	now := time.Now().UTC()
	clock := now.Add(30 * time.Minute).Format("15:04")
	yes := true
	if _, err := s.UpdatePreferences(user.ID, NotificationPreferencesInput{RoutineDigest: &yes, DigestTime: &clock}); err != nil {
		t.Fatal(err)
	}
	experienceCampaign(t, s, "First", "normal", nil)
	late := experienceCampaign(t, s, "Late", "normal", nil)
	if err := s.DB.Model(&models.NotificationOutboxJob{}).Where("campaign_id = ?", late.ID).Update("status", "held").Error; err != nil {
		t.Fatal(err)
	}
	mock := &firebaseMockMessenger{results: []firebaseMockResult{{id: "messages/first"}, {id: "messages/late"}}}
	worker := NotificationOutboxService{DB: s.DB, Firebase: &FirebaseService{DB: s.DB, Project: "test-project", Messenger: mock}, WorkerID: "digest", Now: func() time.Time { return now }}
	now = now.Add(31 * time.Minute)
	if result, err := worker.ProcessDigests(t.Context()); err != nil || result.Accepted != 1 {
		t.Fatalf("first batch: %#v %v", result, err)
	}
	if err := s.DB.Model(&models.NotificationOutboxJob{}).Where("campaign_id = ?", late.ID).Update("status", "pending").Error; err != nil {
		t.Fatal(err)
	}
	if err := reactivateNotificationDigests(s.DB, late.ID, now); err != nil {
		t.Fatal(err)
	}
	if result, err := worker.ProcessDigests(t.Context()); err != nil || result.Accepted != 1 || len(mock.messages) != 2 {
		t.Fatalf("late member stranded: %#v %v", result, err)
	}
	if mock.messages[0].Data["message_id"] == mock.messages[1].Data["message_id"] {
		t.Fatal("late digest members reused the deduplication ID")
	}

}

func TestFirebaseDeliveryImportLeavesCrossDeviceCollisionsUnknown(t *testing.T) {
	s := notificationTestService(t)
	var user models.User
	s.DB.First(&user)
	other := models.FirebaseDevice{UserID: user.ID, InstallationID: uuid.NewString(), RegistrationToken: uuid.NewString(), Platform: "android", NotificationsEnabled: true, LastSeenAt: time.Now().UTC()}
	if err := s.DB.Create(&other).Error; err != nil {
		t.Fatal(err)
	}
	campaign := experienceCampaign(t, s, "Collision", "normal", nil)
	now := time.Now().UTC()
	provider := "projects/test-project/messages/colliding-id"
	if err := s.DB.Model(&models.NotificationDelivery{}).Where("campaign_id = ? AND channel = 'push'", campaign.ID).Updates(map[string]any{"state": "accepted", "accepted_at": now, "provider_message_id": provider}).Error; err != nil {
		t.Fatal(err)
	}
	worker := NotificationOutboxService{DB: s.DB}
	count, err := worker.ImportFirebaseDeliveryEvent("test-project", FirebaseDeliveryImportEvent{MessageID: "colliding-id", SDKPlatform: "ANDROID", EventTimestamp: now, Event: "MESSAGE_DELIVERED"})
	if err != nil || count != 0 {
		t.Fatalf("ambiguous delivery credited: %d %v", count, err)
	}
	var delivered int64
	s.DB.Model(&models.NotificationDelivery{}).Where("campaign_id = ? AND delivered_at IS NOT NULL", campaign.ID).Count(&delivered)
	if delivered != 0 {
		t.Fatal("ambiguous provider receipt changed delivery counts")
	}
}
