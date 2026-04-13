package cost

import (
	"os"
	"testing"
	"time"
)

func TestRecordAndCheckBudget(t *testing.T) {
	dir, err := os.MkdirTemp("", "cost-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	tracker, err := NewTracker(dir, 1.00, 10.00)
	if err != nil {
		t.Fatal(err)
	}

	// Initially budget should be allowed
	ok, reason := tracker.CheckBudget()
	if !ok {
		t.Fatalf("expected budget allowed, got: %s", reason)
	}

	// Record some cost
	err = tracker.Record(Record{
		Timestamp:    time.Now(),
		Provider:     "openrouter",
		Model:        "test-model",
		InputTokens:  1000,
		OutputTokens: 500,
		CostUSD:      0.50,
		SessionID:    "test-session",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Still under budget
	ok, _ = tracker.CheckBudget()
	if !ok {
		t.Fatal("expected budget still allowed after $0.50")
	}

	// Push over daily limit
	err = tracker.Record(Record{
		Timestamp:    time.Now(),
		Provider:     "openrouter",
		Model:        "test-model",
		InputTokens:  2000,
		OutputTokens: 1000,
		CostUSD:      0.60,
		SessionID:    "test-session",
	})
	if err != nil {
		t.Fatal(err)
	}

	ok, reason = tracker.CheckBudget()
	if ok {
		t.Fatal("expected budget exceeded after $1.10 daily spend")
	}
	if reason == "" {
		t.Fatal("expected a reason string for budget exceeded")
	}
}
