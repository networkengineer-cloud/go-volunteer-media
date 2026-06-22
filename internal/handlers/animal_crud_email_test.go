package handlers

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
)

func TestBuildQuarantineEmailBody(t *testing.T) {
	start := time.Date(2026, 6, 22, 0, 0, 0, 0, time.UTC)
	a := &models.Animal{
		Name:                      "Rex",
		QuarantineStartDate:       &start,
		QuarantineIncidentDetails: "Bit a volunteer on the hand during leashing.",
	}
	title, body := buildQuarantineEmail(a)
	if title != "🚨 Bite Quarantine: Rex" {
		t.Errorf("unexpected title: %q", title)
	}
	if !strings.Contains(body, "Rex has been placed in bite quarantine") {
		t.Errorf("body missing intro: %q", body)
	}
	if !strings.Contains(body, "June 22, 2026") {
		t.Errorf("body missing formatted start date: %q", body)
	}
	if !strings.Contains(body, "Bit a volunteer on the hand during leashing.") {
		t.Errorf("body missing incident details: %q", body)
	}
}

func TestSendQuarantineNotificationEmail_NilServiceNoPanic(t *testing.T) {
	// nil email service must be a safe no-op
	sendQuarantineNotificationEmail(context.Background(), nil, nil, &models.Animal{Name: "Rex"})
}
