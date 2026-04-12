package lite

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/swiftdiaries/openharness/harness"
)

// Compile-time interface check.
var _ harness.ChannelRouter = (*LiteChannelRouter)(nil)

// SendFunc is an injectable function for outbound message delivery.
// Used in testing or when the Lite edition needs to route messages
// (e.g., to a log sink). If nil, SendOutbound is a no-op.
type SendFunc func(ctx context.Context, channelName string, msg harness.Message) error

// LiteChannelRouter implements ChannelRouter for the Lite (desktop) edition.
// It maintains an in-memory registry of channels and supports outbound-only
// delivery. Inbound channels are not supported in Lite.
type LiteChannelRouter struct {
	mu       sync.RWMutex
	channels map[string]map[string]harness.ChannelInfo // tenantID -> channelType -> info
	counter  atomic.Int64
	sendFn   SendFunc
}

// NewLiteChannelRouter returns a LiteChannelRouter with an empty registry
// and no send function (SendOutbound is a no-op).
func NewLiteChannelRouter() *LiteChannelRouter {
	return &LiteChannelRouter{
		channels: make(map[string]map[string]harness.ChannelInfo),
	}
}

// NewLiteChannelRouterWithSender returns a LiteChannelRouter with an empty
// registry and the given send function for outbound delivery.
func NewLiteChannelRouterWithSender(fn SendFunc) *LiteChannelRouter {
	return &LiteChannelRouter{
		channels: make(map[string]map[string]harness.ChannelInfo),
		sendFn:   fn,
	}
}

// HandleInbound always returns an error — inbound channels are not supported
// in the Lite edition.
func (r *LiteChannelRouter) HandleInbound(_ context.Context, _ string, _ json.RawMessage) (harness.InboundResult, error) {
	return harness.InboundResult{}, fmt.Errorf("inbound channels not supported in Lite edition: %w", harness.ErrNotFound)
}

// SendOutbound delivers a message via the configured SendFunc. If no SendFunc
// is set, the call is a no-op and returns nil (suitable for the desktop app
// where channels just log).
func (r *LiteChannelRouter) SendOutbound(ctx context.Context, _ string, channelName string, msg harness.Message) error {
	if r.sendFn != nil {
		return r.sendFn(ctx, channelName, msg)
	}
	return nil
}

// RegisterWebhook stores a ChannelInfo entry in the in-memory registry.
// It returns an empty webhook URL since there are no real webhooks in Lite.
func (r *LiteChannelRouter) RegisterWebhook(_ context.Context, tenantID, channelType string, cfg harness.ChannelConfig) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	tenant, ok := r.channels[tenantID]
	if !ok {
		tenant = make(map[string]harness.ChannelInfo)
		r.channels[tenantID] = tenant
	}

	id := fmt.Sprintf("ch-%d", r.counter.Add(1))
	_ = id // ID generated but not exposed — used internally for uniqueness

	tenant[channelType] = harness.ChannelInfo{
		Name:      cfg.Name,
		Type:      channelType,
		Direction: cfg.Direction,
		Enabled:   true,
	}

	return "", nil
}

// UnregisterWebhook removes a channel from the in-memory registry.
// Returns ErrNotFound if the channel is not registered.
func (r *LiteChannelRouter) UnregisterWebhook(_ context.Context, tenantID, channelType string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	tenant, ok := r.channels[tenantID]
	if !ok {
		return fmt.Errorf("%w: no channels for tenant %q", harness.ErrNotFound, tenantID)
	}

	if _, exists := tenant[channelType]; !exists {
		return fmt.Errorf("%w: channel type %q not registered for tenant %q", harness.ErrNotFound, channelType, tenantID)
	}

	delete(tenant, channelType)
	return nil
}

// ListChannels returns all ChannelInfo entries for the given tenant.
func (r *LiteChannelRouter) ListChannels(_ context.Context, tenantID string) ([]harness.ChannelInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tenant, ok := r.channels[tenantID]
	if !ok {
		return nil, nil
	}

	result := make([]harness.ChannelInfo, 0, len(tenant))
	for _, info := range tenant {
		result = append(result, info)
	}
	return result, nil
}
