package repository

import (
	"context"
	"fmt"
	"sync"

	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/entity"
)

// InMemoryConsumerRepository is a simple in-memory implementation of ConsumerRepository.
// It is thread-safe for concurrent access.
type InMemoryConsumerRepository struct {
	sync.RWMutex
	consumers map[string]*entity.Consumer
}

// NewInMemoryConsumerRepository creates a new InMemoryConsumerRepository.
func NewInMemoryConsumerRepository() *InMemoryConsumerRepository {
	return &InMemoryConsumerRepository{
		consumers: make(map[string]*entity.Consumer),
	}
}

// Save stores or updates a consumer in the repository.
func (r *InMemoryConsumerRepository) Save(ctx context.Context, consumer *entity.Consumer) error {
	r.Lock()
	defer r.Unlock()
	if r.consumers == nil {
		r.consumers = make(map[string]*entity.Consumer)
	}
	r.consumers[consumer.ID()] = consumer
	return nil
}

// Get retrieves a consumer by its ID.
func (r *InMemoryConsumerRepository) Get(ctx context.Context, id string) (*entity.Consumer, error) {
	r.RLock()
	defer r.RUnlock()
	if consumer, exists := r.consumers[id]; exists {
		return consumer, nil
	}
	return nil, fmt.Errorf("consumer not found")
}

// List retrieves all consumers, optionally filtered by group ID.
func (r *InMemoryConsumerRepository) List(ctx context.Context, groupID string) ([]*entity.Consumer, error) {
	r.RLock()
	defer r.RUnlock()
	var result []*entity.Consumer
	for _, consumer := range r.consumers {
		if groupID == "" || consumer.GroupID() == groupID {
			result = append(result, consumer)
		}
	}
	return result, nil
}

// Delete removes a consumer from the repository.
func (r *InMemoryConsumerRepository) Delete(ctx context.Context, id string) error {
	r.Lock()
	defer r.Unlock()
	delete(r.consumers, id)
	return nil
}
