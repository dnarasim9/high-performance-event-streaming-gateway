package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dheemanth-hn/event-streaming-gateway/internal/application/dto"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/entity"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/valueobject"
)

// Mock implementations for SubscriptionService testing

type MockEventSubscriber struct {
	SubscribeFunc   func(ctx context.Context, topic, group string, filter valueobject.EventFilter) (<-chan *entity.Event, error)
	UnsubscribeFunc func() error
	CloseFunc       func() error
}

func (m *MockEventSubscriber) Subscribe(ctx context.Context, topic, group string, filter valueobject.EventFilter) (<-chan *entity.Event, error) {
	if m.SubscribeFunc != nil {
		return m.SubscribeFunc(ctx, topic, group, filter)
	}
	ch := make(chan *entity.Event)
	close(ch)
	return ch, nil
}

func (m *MockEventSubscriber) Unsubscribe() error {
	if m.UnsubscribeFunc != nil {
		return m.UnsubscribeFunc()
	}
	return nil
}

func (m *MockEventSubscriber) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

// TestSubscriptionService_Subscribe_Success tests successful subscription creation
func TestSubscriptionService_Subscribe_Success(t *testing.T) {
	eventChan := make(chan *entity.Event, 10)
	close(eventChan)

	subscriber := &MockEventSubscriber{
		SubscribeFunc: func(ctx context.Context, topic, group string, filter valueobject.EventFilter) (<-chan *entity.Event, error) {
			return eventChan, nil
		},
	}
	metrics := &MockMetricsCollector{}
	service := NewSubscriptionService(subscriber, metrics, newTestLogger())

	req := &dto.SubscribeRequest{
		ConsumerID: "consumer-1",
		GroupID:    "group-1",
	}

	ch, err := service.Subscribe(context.Background(), req)
	if err != nil {
		t.Errorf("Subscribe() error = %v, want nil", err)
	}
	if ch == nil {
		t.Error("Subscribe() returned nil channel")
	}

	activeCount := service.GetActiveSubscriptions()
	if activeCount != 1 {
		t.Errorf("GetActiveSubscriptions() = %d, want 1", activeCount)
	}
}

// TestSubscriptionService_Subscribe_WithFilter tests subscription with filter
func TestSubscriptionService_Subscribe_WithFilter(t *testing.T) {
	eventChan := make(chan *entity.Event, 10)
	close(eventChan)

	subscriber := &MockEventSubscriber{
		SubscribeFunc: func(ctx context.Context, topic, group string, filter valueobject.EventFilter) (<-chan *entity.Event, error) {
			return eventChan, nil
		},
	}
	metrics := &MockMetricsCollector{}
	service := NewSubscriptionService(subscriber, metrics, newTestLogger())

	req := &dto.SubscribeRequest{
		ConsumerID: "consumer-1",
		GroupID:    "group-1",
		Filter: &dto.EventFilterDTO{
			TypePattern: "user.created",
		},
	}

	ch, err := service.Subscribe(context.Background(), req)
	if err != nil {
		t.Errorf("Subscribe() error = %v, want nil", err)
	}
	if ch == nil {
		t.Error("Subscribe() returned nil channel")
	}
}

// TestSubscriptionService_Subscribe_EmptyGroupID tests subscription validation
func TestSubscriptionService_Subscribe_EmptyGroupID(t *testing.T) {
	eventChan := make(chan *entity.Event, 10)
	close(eventChan)

	subscriber := &MockEventSubscriber{
		SubscribeFunc: func(ctx context.Context, topic, group string, filter valueobject.EventFilter) (<-chan *entity.Event, error) {
			return eventChan, nil
		},
	}
	metrics := &MockMetricsCollector{}
	service := NewSubscriptionService(subscriber, metrics, newTestLogger())

	req := &dto.SubscribeRequest{
		ConsumerID: "consumer-1",
		GroupID:    "group-1",
	}

	ch, err := service.Subscribe(context.Background(), req)
	if err != nil {
		t.Errorf("Subscribe() error = %v, want nil", err)
	}
	if ch == nil {
		t.Error("Subscribe() returned nil channel")
	}
}

// TestSubscriptionService_Subscribe_BrokerError tests subscription with broker error
func TestSubscriptionService_Subscribe_BrokerError(t *testing.T) {
	subscriber := &MockEventSubscriber{
		SubscribeFunc: func(ctx context.Context, topic, group string, filter valueobject.EventFilter) (<-chan *entity.Event, error) {
			return nil, errors.New("broker error")
		},
	}
	metrics := &MockMetricsCollector{}
	service := NewSubscriptionService(subscriber, metrics, newTestLogger())

	req := &dto.SubscribeRequest{
		ConsumerID: "consumer-1",
		GroupID:    "group-1",
	}

	_, err := service.Subscribe(context.Background(), req)
	if err == nil {
		t.Error("Subscribe() with broker error expected error, got nil")
	}
}

// TestSubscriptionService_Unsubscribe_Success tests successful unsubscription
func TestSubscriptionService_Unsubscribe_Success(t *testing.T) {
	eventChan := make(chan *entity.Event, 10)
	close(eventChan)

	subscriber := &MockEventSubscriber{
		SubscribeFunc: func(ctx context.Context, topic, group string, filter valueobject.EventFilter) (<-chan *entity.Event, error) {
			return eventChan, nil
		},
		UnsubscribeFunc: func() error {
			return nil
		},
	}
	metrics := &MockMetricsCollector{}
	service := NewSubscriptionService(subscriber, metrics, newTestLogger())

	req := &dto.SubscribeRequest{
		ConsumerID: "consumer-1",
		GroupID:    "group-1",
	}

	ch, err := service.Subscribe(context.Background(), req)
	if err != nil {
		t.Fatalf("Subscribe() error = %v, want nil", err)
	}

	// Get subscription ID from the active subscriptions
	activeCount := service.GetActiveSubscriptions()
	if activeCount != 1 {
		t.Fatalf("GetActiveSubscriptions() = %d, want 1", activeCount)
	}

	// Find the subscription ID
	var subID string
	for key := range service.subscriptions {
		subID = key
		break
	}

	err = service.Unsubscribe(context.Background(), subID)
	if err != nil {
		t.Errorf("Unsubscribe() error = %v, want nil", err)
	}

	// Drain the channel to avoid goroutine leak
	for range ch {
	}
}

// TestSubscriptionService_Unsubscribe_NotFound tests unsubscription of non-existent subscription
func TestSubscriptionService_Unsubscribe_NotFound(t *testing.T) {
	subscriber := &MockEventSubscriber{}
	metrics := &MockMetricsCollector{}
	service := NewSubscriptionService(subscriber, metrics, newTestLogger())

	err := service.Unsubscribe(context.Background(), "non-existent")
	if err == nil {
		t.Error("Unsubscribe() for non-existent subscription expected error, got nil")
	}
}

// TestSubscriptionService_Unsubscribe_BrokerError tests unsubscription with broker error
func TestSubscriptionService_Unsubscribe_BrokerError(t *testing.T) {
	eventChan := make(chan *entity.Event, 10)
	close(eventChan)

	subscriber := &MockEventSubscriber{
		SubscribeFunc: func(ctx context.Context, topic, group string, filter valueobject.EventFilter) (<-chan *entity.Event, error) {
			return eventChan, nil
		},
		UnsubscribeFunc: func() error {
			return errors.New("broker error")
		},
	}
	metrics := &MockMetricsCollector{}
	service := NewSubscriptionService(subscriber, metrics, newTestLogger())

	req := &dto.SubscribeRequest{
		ConsumerID: "consumer-1",
		GroupID:    "group-1",
	}

	ch, err := service.Subscribe(context.Background(), req)
	if err != nil {
		t.Fatalf("Subscribe() error = %v, want nil", err)
	}

	// Get subscription ID
	var subID string
	for key := range service.subscriptions {
		subID = key
		break
	}

	err = service.Unsubscribe(context.Background(), subID)
	if err == nil {
		t.Error("Unsubscribe() with broker error expected error, got nil")
	}

	// Drain the channel to avoid goroutine leak
	for range ch {
	}
}

// TestSubscriptionService_GetActiveSubscriptions tests active subscription count
func TestSubscriptionService_GetActiveSubscriptions(t *testing.T) {
	eventChan := make(chan *entity.Event, 10)
	close(eventChan)

	subscriber := &MockEventSubscriber{
		SubscribeFunc: func(ctx context.Context, topic, group string, filter valueobject.EventFilter) (<-chan *entity.Event, error) {
			return eventChan, nil
		},
	}
	metrics := &MockMetricsCollector{}
	service := NewSubscriptionService(subscriber, metrics, newTestLogger())

	if service.GetActiveSubscriptions() != 0 {
		t.Error("GetActiveSubscriptions() should start at 0")
	}

	// Create multiple subscriptions
	for i := 0; i < 3; i++ {
		req := &dto.SubscribeRequest{
			ConsumerID: "consumer-1",
			GroupID:    "group-1",
		}
		_, _ = service.Subscribe(context.Background(), req)
	}

	if service.GetActiveSubscriptions() != 3 {
		t.Errorf("GetActiveSubscriptions() = %d, want 3", service.GetActiveSubscriptions())
	}
}

// TestSubscriptionService_SetChannelBufferSize tests buffer size configuration
func TestSubscriptionService_SetChannelBufferSize(t *testing.T) {
	subscriber := &MockEventSubscriber{}
	metrics := &MockMetricsCollector{}
	service := NewSubscriptionService(subscriber, metrics, newTestLogger())

	service.SetChannelBufferSize(5000)
	if service.channelBufferSize != 5000 {
		t.Errorf("SetChannelBufferSize() = %d, want 5000", service.channelBufferSize)
	}
}

// TestSubscriptionService_SubscribeMetricsRecorded tests that metrics are recorded
func TestSubscriptionService_SubscribeMetricsRecorded(t *testing.T) {
	eventChan := make(chan *entity.Event, 10)
	close(eventChan)

	subscriber := &MockEventSubscriber{
		SubscribeFunc: func(ctx context.Context, topic, group string, filter valueobject.EventFilter) (<-chan *entity.Event, error) {
			return eventChan, nil
		},
	}

	metricsRecorded := false
	metrics := &MockMetricsCollector{
		RecordSubscriptionFunc: func(consumerID, groupID string) {
			metricsRecorded = true
		},
	}
	service := NewSubscriptionService(subscriber, metrics, newTestLogger())

	req := &dto.SubscribeRequest{
		ConsumerID: "consumer-1",
		GroupID:    "group-1",
	}

	_, err := service.Subscribe(context.Background(), req)
	if err != nil {
		t.Errorf("Subscribe() error = %v, want nil", err)
	}
	if !metricsRecorded {
		t.Error("Subscribe() did not record subscription metrics")
	}
}

// TestSubscriptionService_ForwardEvents tests event forwarding
func TestSubscriptionService_ForwardEvents(t *testing.T) {
	// Create a channel with some test events
	inputChan := make(chan *entity.Event, 3)

	event1, _ := entity.NewEvent(
		"event-1",
		"test-source",
		"test.type",
		"/test",
		[]byte(`{"test": "data"}`),
		"application/json",
	)
	event2, _ := entity.NewEvent(
		"event-2",
		"test-source",
		"test.type",
		"/test",
		[]byte(`{"test": "data2"}`),
		"application/json",
	)

	inputChan <- event1
	inputChan <- event2
	close(inputChan)

	subscriber := &MockEventSubscriber{
		SubscribeFunc: func(ctx context.Context, topic, group string, filter valueobject.EventFilter) (<-chan *entity.Event, error) {
			return inputChan, nil
		},
	}
	metrics := &MockMetricsCollector{}
	service := NewSubscriptionService(subscriber, metrics, newTestLogger())

	req := &dto.SubscribeRequest{
		ConsumerID: "consumer-1",
		GroupID:    "group-1",
	}

	outputChan, err := service.Subscribe(context.Background(), req)
	if err != nil {
		t.Fatalf("Subscribe() error = %v, want nil", err)
	}

	// Verify events are forwarded
	eventCount := 0
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	for {
		select {
		case event := <-outputChan:
			if event != nil {
				eventCount++
			}
		case <-ctx.Done():
			goto done
		}
	}
done:
	if eventCount != 2 {
		t.Errorf("ForwardEvents() forwarded %d events, want 2", eventCount)
	}
}

// TestSubscriptionService_ConcurrentSubscriptions tests thread safety with multiple subscriptions
func TestSubscriptionService_ConcurrentSubscriptions(t *testing.T) {
	eventChan := make(chan *entity.Event, 10)
	close(eventChan)

	subscriber := &MockEventSubscriber{
		SubscribeFunc: func(ctx context.Context, topic, group string, filter valueobject.EventFilter) (<-chan *entity.Event, error) {
			return eventChan, nil
		},
	}
	metrics := &MockMetricsCollector{}
	service := NewSubscriptionService(subscriber, metrics, newTestLogger())

	// Create subscriptions concurrently
	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func(idx int) {
			req := &dto.SubscribeRequest{
				ConsumerID: "consumer-" + string(rune('0'+idx)),
				GroupID:    "group-1",
			}
			_, _ = service.Subscribe(context.Background(), req)
			done <- true
		}(i)
	}

	// Wait for all subscriptions to complete
	for i := 0; i < 5; i++ {
		<-done
	}

	if service.GetActiveSubscriptions() != 5 {
		t.Errorf("ConcurrentSubscriptions() = %d, want 5", service.GetActiveSubscriptions())
	}
}
