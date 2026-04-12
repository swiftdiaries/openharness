package lite

import (
	"context"
	"errors"
	"testing"

	"github.com/swiftdiaries/openharness/harness"
)

func TestLiteChannelRouterSendOutbound(t *testing.T) {
	var called bool
	fn := func(_ context.Context, channelName string, msg harness.Message) error {
		called = true
		if channelName != "slack-general" {
			t.Errorf("channelName = %q, want %q", channelName, "slack-general")
		}
		if msg.Content != "hello" {
			t.Errorf("msg.Content = %q, want %q", msg.Content, "hello")
		}
		return nil
	}

	r := NewLiteChannelRouterWithSender(fn)
	ctx := context.Background()

	// Register a channel first to prove it doesn't affect send.
	_, err := r.RegisterWebhook(ctx, "t1", "slack", harness.ChannelConfig{
		Name:      "slack-general",
		Direction: "outbound",
	})
	if err != nil {
		t.Fatalf("RegisterWebhook: %v", err)
	}

	err = r.SendOutbound(ctx, "t1", "slack-general", harness.Message{Content: "hello"})
	if err != nil {
		t.Fatalf("SendOutbound: %v", err)
	}
	if !called {
		t.Error("SendFunc was not called")
	}

	// Also test that a nil-sender router is a no-op.
	r2 := NewLiteChannelRouter()
	err = r2.SendOutbound(ctx, "t1", "slack-general", harness.Message{Content: "hello"})
	if err != nil {
		t.Fatalf("SendOutbound (nil sender): %v", err)
	}
}

func TestLiteChannelRouterHandleInbound(t *testing.T) {
	r := NewLiteChannelRouter()
	ctx := context.Background()

	_, err := r.HandleInbound(ctx, "slack", nil)
	if err == nil {
		t.Fatal("expected error from HandleInbound, got nil")
	}
	if !errors.Is(err, harness.ErrNotFound) {
		t.Errorf("error = %v, want wrapping ErrNotFound", err)
	}
}

func TestLiteChannelRouterRegisterAndList(t *testing.T) {
	r := NewLiteChannelRouter()
	ctx := context.Background()

	// Empty tenant returns nil.
	list, err := r.ListChannels(ctx, "t1")
	if err != nil {
		t.Fatalf("ListChannels (empty): %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected 0 channels, got %d", len(list))
	}

	// Register two channels for the same tenant.
	_, err = r.RegisterWebhook(ctx, "t1", "slack", harness.ChannelConfig{
		Name:      "slack-general",
		Direction: "outbound",
	})
	if err != nil {
		t.Fatalf("RegisterWebhook(slack): %v", err)
	}

	_, err = r.RegisterWebhook(ctx, "t1", "email", harness.ChannelConfig{
		Name:      "ops-alerts",
		Direction: "outbound",
	})
	if err != nil {
		t.Fatalf("RegisterWebhook(email): %v", err)
	}

	list, err = r.ListChannels(ctx, "t1")
	if err != nil {
		t.Fatalf("ListChannels: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 channels, got %d", len(list))
	}

	// Verify channel info is correct.
	byType := make(map[string]harness.ChannelInfo)
	for _, ch := range list {
		byType[ch.Type] = ch
	}

	slack, ok := byType["slack"]
	if !ok {
		t.Fatal("missing slack channel")
	}
	if slack.Name != "slack-general" {
		t.Errorf("slack.Name = %q, want %q", slack.Name, "slack-general")
	}
	if !slack.Enabled {
		t.Error("slack channel should be enabled")
	}

	email, ok := byType["email"]
	if !ok {
		t.Fatal("missing email channel")
	}
	if email.Name != "ops-alerts" {
		t.Errorf("email.Name = %q, want %q", email.Name, "ops-alerts")
	}

	// Different tenant should still be empty.
	list, err = r.ListChannels(ctx, "t2")
	if err != nil {
		t.Fatalf("ListChannels (t2): %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected 0 channels for t2, got %d", len(list))
	}
}

func TestLiteChannelRouterUnregister(t *testing.T) {
	r := NewLiteChannelRouter()
	ctx := context.Background()

	// Register then unregister.
	_, err := r.RegisterWebhook(ctx, "t1", "slack", harness.ChannelConfig{
		Name:      "slack-general",
		Direction: "outbound",
	})
	if err != nil {
		t.Fatalf("RegisterWebhook: %v", err)
	}

	err = r.UnregisterWebhook(ctx, "t1", "slack")
	if err != nil {
		t.Fatalf("UnregisterWebhook: %v", err)
	}

	// List should be empty now.
	list, err := r.ListChannels(ctx, "t1")
	if err != nil {
		t.Fatalf("ListChannels: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected 0 channels after unregister, got %d", len(list))
	}
}

func TestLiteChannelRouterUnregisterNotFound(t *testing.T) {
	r := NewLiteChannelRouter()
	ctx := context.Background()

	// Unregister from a tenant with no channels at all.
	err := r.UnregisterWebhook(ctx, "t1", "slack")
	if err == nil {
		t.Fatal("expected error for unregistering nonexistent channel")
	}
	if !errors.Is(err, harness.ErrNotFound) {
		t.Errorf("error = %v, want wrapping ErrNotFound", err)
	}

	// Register one type, then try to unregister a different type.
	_, err = r.RegisterWebhook(ctx, "t1", "email", harness.ChannelConfig{
		Name:      "ops-alerts",
		Direction: "outbound",
	})
	if err != nil {
		t.Fatalf("RegisterWebhook: %v", err)
	}

	err = r.UnregisterWebhook(ctx, "t1", "slack")
	if err == nil {
		t.Fatal("expected error for unregistering nonexistent channel type")
	}
	if !errors.Is(err, harness.ErrNotFound) {
		t.Errorf("error = %v, want wrapping ErrNotFound", err)
	}
}
