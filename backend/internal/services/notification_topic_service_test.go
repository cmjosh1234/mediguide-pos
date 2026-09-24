package services

import (
	"encoding/json"
	"testing"
	"time"

	"mediguide/internal/models"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func publishOutbreakForTest(t *testing.T, service OutbreakAdminService, id uuid.UUID, lock int, status string) *OutbreakAdminDTO {
	t.Helper()
	author, reviewer, publisher := OutbreakActor{ID: uuid.New()}, OutbreakActor{ID: uuid.New()}, OutbreakActor{ID: uuid.New()}
	if _, err := service.TransitionOutbreak(author, id, "submit", TransitionInput{LockVersion: lock}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.TransitionOutbreak(reviewer, id, "approve", TransitionInput{LockVersion: lock + 1}); err != nil {
		t.Fatal(err)
	}
	published, err := service.TransitionOutbreak(publisher, id, "publish", TransitionInput{LockVersion: lock + 2, OperationalStatus: status})
	if err != nil {
		t.Fatal(err)
	}
	return published
}

func topicJobsForTest(t *testing.T, db *gorm.DB) []models.NotificationTopicJob {
	t.Helper()
	var jobs []models.NotificationTopicJob
	if err := db.Order("created_at").Find(&jobs).Error; err != nil {
		t.Fatal(err)
	}
	return jobs
}

func topicPayloadForTest(t *testing.T, job models.NotificationTopicJob) NotificationDeliveryPayload {
	t.Helper()
	var payload NotificationDeliveryPayload
	if err := json.Unmarshal(job.PayloadJSON, &payload); err != nil {
		t.Fatal(err)
	}
	return payload
}

func TestPublishingAnOutbreakAlertsTopicSubscribersOnce(t *testing.T) {
	service := outbreakAdminTestService(t)
	author := OutbreakActor{ID: uuid.New()}
	item, err := service.CreateOutbreak(author, validOutbreakDraftInput(time.Now().UTC()))
	if err != nil {
		t.Fatal(err)
	}
	published := publishOutbreakForTest(t, service, item.ID, 1, "active")

	jobs := topicJobsForTest(t, service.DB)
	if len(jobs) != 1 || jobs[0].Topic != PublicOutbreakTopic || jobs[0].SourceID != item.ID || jobs[0].Status != "pending" || jobs[0].ExpiresAt == nil {
		t.Fatalf("expected one pending outbreak alert, got %#v", jobs)
	}
	payload := topicPayloadForTest(t, jobs[0])
	if payload.Title != "Outbreak alert: Ebola response" || payload.Body != "Public health response" || !payload.PublicContent {
		t.Fatalf("unexpected alert text: %#v", payload)
	}
	// validOutbreakDraftInput is critical, so the alert must interrupt.
	if payload.Priority != "urgent" || payload.AndroidChannel != "mediguide_emergency" {
		t.Fatalf("critical outbreak alert is not urgent: %#v", payload)
	}
	if payload.Action.Type != NotificationActionOutbreak || payload.Action.ResourceID == nil || *payload.Action.ResourceID != item.ID.String() {
		t.Fatalf("alert does not open the outbreak: %#v", payload.Action)
	}

	corrected, err := service.CorrectOutbreak(author, item.ID, TransitionInput{LockVersion: published.LockVersion, Reason: "Correct case definition"})
	if err != nil {
		t.Fatal(err)
	}
	publishOutbreakForTest(t, service, corrected.ID, corrected.LockVersion, "active")
	if jobs := topicJobsForTest(t, service.DB); len(jobs) != 1 {
		t.Fatalf("a republished correction alerted subscribers again: %d jobs", len(jobs))
	}
}

func TestPublishingAClosedOutbreakRecordDoesNotAlert(t *testing.T) {
	service := outbreakAdminTestService(t)
	item, err := service.CreateOutbreak(OutbreakActor{ID: uuid.New()}, validOutbreakDraftInput(time.Now().UTC()))
	if err != nil {
		t.Fatal(err)
	}
	publishOutbreakForTest(t, service, item.ID, 1, "closed")
	if jobs := topicJobsForTest(t, service.DB); len(jobs) != 0 {
		t.Fatalf("closed outbreak record alerted subscribers: %#v", jobs)
	}
}

func TestPublishingAnOutbreakUpdateAlertsTopicSubscribers(t *testing.T) {
	service := outbreakAdminTestService(t)
	item, err := service.CreateOutbreak(OutbreakActor{ID: uuid.New()}, validOutbreakDraftInput(time.Now().UTC()))
	if err != nil {
		t.Fatal(err)
	}
	publishOutbreakForTest(t, service, item.ID, 1, "active")
	author, reviewer, publisher := OutbreakActor{ID: uuid.New()}, OutbreakActor{ID: uuid.New()}, OutbreakActor{ID: uuid.New()}
	title := "Two new districts report cases"
	update, err := service.CreateUpdate(author, item.ID, ChildContentInput{Title: &title})
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range []struct {
		actor  OutbreakActor
		action string
		lock   int
	}{{author, "submit", 1}, {reviewer, "approve", 2}, {publisher, "publish", 3}} {
		if _, err := service.TransitionUpdate(step.actor, item.ID, update.ID, step.action, TransitionInput{LockVersion: step.lock}); err != nil {
			t.Fatalf("%s: %v", step.action, err)
		}
	}

	jobs := topicJobsForTest(t, service.DB)
	if len(jobs) != 2 || jobs[1].SourceType != "outbreak_update" || jobs[1].SourceID != update.ID {
		t.Fatalf("expected the update to be announced, got %#v", jobs)
	}
	payload := topicPayloadForTest(t, jobs[1])
	if payload.Title != "Ebola response" || payload.Body != title || payload.AndroidChannel != "mediguide_updates" {
		t.Fatalf("unexpected update alert: %#v", payload)
	}
	if payload.CollapseKey != "outbreak-"+item.ID.String() {
		t.Fatalf("update alert should replace the outbreak's earlier alert on the device: %q", payload.CollapseKey)
	}
}

func TestTopicWorkerRetriesThenDeliversAndDropsExpiredAlerts(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.NotificationTopicJob{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	mock := &firebaseMockMessenger{results: []firebaseMockResult{{err: mockFirebaseError("unavailable")}, {id: "messages/topic-1"}}}
	worker := NotificationTopicService{DB: db, Firebase: &FirebaseService{DB: db, Project: "test-project", Messenger: mock}, WorkerID: "worker-1", Now: func() time.Time { return now }}
	alert := outbreakTopicAlert{SourceType: "outbreak", SourceID: uuid.New(), OutbreakID: uuid.New(), Title: "Outbreak alert: Cholera", Body: "Boil drinking water."}
	if err := enqueueOutbreakTopicAlertTx(db, alert, now); err != nil {
		t.Fatal(err)
	}
	// Queuing the same publish twice must not send twice.
	if err := enqueueOutbreakTopicAlertTx(db, alert, now); err != nil {
		t.Fatal(err)
	}

	result, err := worker.ProcessBatch(t.Context())
	if err != nil || result.Claimed != 1 || result.Retried != 1 {
		t.Fatalf("first attempt: %#v %v", result, err)
	}
	if len(mock.messages) != 1 || mock.messages[0].Topic != PublicOutbreakTopic || mock.messages[0].Notification.Title != "Outbreak alert: Cholera" {
		t.Fatalf("unexpected topic message: %#v", mock.messages)
	}
	if result, _ := worker.ProcessBatch(t.Context()); result.Claimed != 0 {
		t.Fatalf("retry ran before its backoff: %#v", result)
	}

	now = now.Add(time.Hour)
	result, err = worker.ProcessBatch(t.Context())
	if err != nil || result.Accepted != 1 {
		t.Fatalf("retry: %#v %v", result, err)
	}
	jobs := topicJobsForTest(t, db)
	if jobs[0].Status != "accepted" || jobs[0].ProviderMessageID == nil || *jobs[0].ProviderMessageID != "messages/topic-1" || jobs[0].AttemptCount != 2 {
		t.Fatalf("unexpected accepted job: %#v", jobs[0])
	}

	stale := outbreakTopicAlert{SourceType: "outbreak", SourceID: uuid.New(), OutbreakID: uuid.New(), Title: "Old alert"}
	if err := enqueueOutbreakTopicAlertTx(db, stale, now.Add(-25*time.Hour)); err != nil {
		t.Fatal(err)
	}
	result, err = worker.ProcessBatch(t.Context())
	if err != nil || result.Failed != 1 || len(mock.messages) != 2 {
		t.Fatalf("expired alert was sent: %#v %v messages=%d", result, err, len(mock.messages))
	}
}

func TestTruncateRunesKeepsAlertsWithinLimits(t *testing.T) {
	if got := truncateRunes("Kasese", 10); got != "Kasese" {
		t.Fatalf("short text changed: %q", got)
	}
	if got := truncateRunes("Ebola response in Kasese", 10); got != "Ebola res…" {
		t.Fatalf("long text: %q", got)
	}
}
