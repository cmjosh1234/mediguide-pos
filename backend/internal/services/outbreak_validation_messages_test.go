package services

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Administrators see these messages in the dashboard, so each rejection has to
// say what is wrong, not just that the operation was invalid.
func TestOutbreakRejectionsExplainWhatToFix(t *testing.T) {
	service := outbreakAdminTestService(t)
	service.AllowedExternalHosts = []string{"who.int", "health.go.ug"}
	now := time.Now().UTC().Add(-time.Hour)
	actor := OutbreakActor{ID: uuid.New()}

	cases := []struct {
		name   string
		mutate func(*OutbreakInput)
		want   string
	}{
		{"missing title", func(in *OutbreakInput) { in.Title = ptr("  ") }, "Title is required"},
		{"missing disease", func(in *OutbreakInput) { in.DiseaseID = nil }, "Disease is required"},
		{"unapproved source domain", func(in *OutbreakInput) { in.SourceURL = ptr("https://google.com") }, "google.com is not an approved domain"},
		{"insecure source url", func(in *OutbreakInput) { in.SourceURL = ptr("http://who.int/x") }, "must be a full link starting with https://"},
		{"verified before data", func(in *OutbreakInput) { in.LastVerifiedAt = ptr(now.Add(-2 * time.Hour)) }, "Last verified can't be earlier than Data as of"},
		{"metric without source", func(in *OutbreakInput) {
			in.Metrics = &[]OutbreakMetric{{Key: "cases", Label: "Cases", Value: "2", AsOf: now, SortOrder: 1}}
		}, "Metric 1: source is required"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := validOutbreakDraftInput(now)
			tc.mutate(&input)
			_, err := service.CreateOutbreak(actor, input)
			var validation *OutbreakValidationError
			if !errors.Is(err, ErrOutbreakInvalid) || !errors.As(err, &validation) || !strings.Contains(validation.Message, tc.want) {
				t.Fatalf("want message containing %q, got %v", tc.want, err)
			}
		})
	}
}

func TestOutbreakWorkflowRejectionsExplainWhy(t *testing.T) {
	service := outbreakAdminTestService(t)
	now := time.Now().UTC()
	author := OutbreakActor{ID: uuid.New()}
	item, err := service.CreateOutbreak(author, validOutbreakDraftInput(now))
	if err != nil {
		t.Fatal(err)
	}
	expect := func(action string, actor OutbreakActor, lock int, want string) {
		t.Helper()
		_, err := service.TransitionOutbreak(actor, item.ID, action, TransitionInput{LockVersion: lock, Reason: "why"})
		if !errors.Is(err, ErrOutbreakInvalid) || err.Error() == ErrOutbreakInvalid.Error() || !strings.Contains(err.Error(), want) {
			t.Fatalf("%s: want message containing %q, got %v", action, want, err)
		}
	}
	expect("withdraw", author, 1, "Only a published outbreak can be withdrawn. This outbreak is draft")
	expect("approve", author, 1, "Only an outbreak in review can be approved")
	if _, err = service.TransitionOutbreak(author, item.ID, "submit", TransitionInput{LockVersion: 1}); err != nil {
		t.Fatal(err)
	}
	expect("approve", author, 2, "You created this outbreak, so a different reviewer has to approve it")
	expect("publish", author, 2, "has to be approved before it can be published")
}

func TestOutbreakUpdateStatusMovesFreelyAmongPublicStatuses(t *testing.T) {
	service := outbreakAdminTestService(t)
	now := time.Now().UTC()
	author := OutbreakActor{ID: uuid.New()}
	reviewer := OutbreakActor{ID: uuid.New()}
	publisher := OutbreakActor{ID: uuid.New()}
	item, err := service.CreateOutbreak(author, validOutbreakDraftInput(now))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.TransitionOutbreak(author, item.ID, "submit", TransitionInput{LockVersion: 1}); err != nil {
		t.Fatal(err)
	}
	item, err = service.TransitionOutbreak(reviewer, item.ID, "approve", TransitionInput{LockVersion: 2})
	if err != nil {
		t.Fatal(err)
	}
	item, err = service.TransitionOutbreak(publisher, item.ID, "publish", TransitionInput{LockVersion: 3, OperationalStatus: "active"})
	if err != nil || item.Status != "active" {
		t.Fatalf("publish: %#v %v", item, err)
	}

	// A reason is required.
	if _, err = service.TransitionOutbreak(reviewer, item.ID, "update_status", TransitionInput{LockVersion: item.LockVersion, OperationalStatus: "monitoring"}); !errors.Is(err, ErrOutbreakInvalid) {
		t.Fatalf("missing reason accepted: %v", err)
	}

	item, err = service.TransitionOutbreak(reviewer, item.ID, "update_status", TransitionInput{LockVersion: item.LockVersion, OperationalStatus: "monitoring", Reason: "case count declining"})
	if err != nil || item.Status != "monitoring" {
		t.Fatalf("active->monitoring: %#v %v", item, err)
	}
	item, err = service.TransitionOutbreak(reviewer, item.ID, "update_status", TransitionInput{LockVersion: item.LockVersion, OperationalStatus: "contained", Reason: "no new cases"})
	if err != nil || item.Status != "contained" {
		t.Fatalf("monitoring->contained: %#v %v", item, err)
	}
	item, err = service.TransitionOutbreak(reviewer, item.ID, "update_status", TransitionInput{LockVersion: item.LockVersion, OperationalStatus: "active", Reason: "new cluster identified"})
	if err != nil || item.Status != "active" {
		t.Fatalf("contained->active: %#v %v", item, err)
	}
	item, err = service.TransitionOutbreak(reviewer, item.ID, "update_status", TransitionInput{LockVersion: item.LockVersion, OperationalStatus: "closed", Reason: "response concluded"})
	if err != nil || item.Status != "closed" {
		t.Fatalf("active->closed: %#v %v", item, err)
	}

	// closed is terminal for update_status (withdraw from closed is unaffected by
	// this feature and stays allowed, as it already was).
	if _, err = service.TransitionOutbreak(reviewer, item.ID, "update_status", TransitionInput{LockVersion: item.LockVersion, OperationalStatus: "active", Reason: "reopen"}); !errors.Is(err, ErrOutbreakInvalid) {
		t.Fatalf("status change from closed accepted: %v", err)
	}
}

func TestOutbreakUpdateStatusRejectsFromDraftAndSameTarget(t *testing.T) {
	service := outbreakAdminTestService(t)
	now := time.Now().UTC()
	author := OutbreakActor{ID: uuid.New()}
	item, err := service.CreateOutbreak(author, validOutbreakDraftInput(now))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.TransitionOutbreak(author, item.ID, "update_status", TransitionInput{LockVersion: 1, OperationalStatus: "monitoring", Reason: "x"}); !errors.Is(err, ErrOutbreakInvalid) {
		t.Fatalf("status change from draft accepted: %v", err)
	}
}
