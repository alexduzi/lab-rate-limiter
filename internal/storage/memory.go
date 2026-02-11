package storage

import (
	"context"
	"sync"
	"time"
)

type memoryEntry struct {
	counter      int64
	windowStart  time.Time
	blockedUntil *time.Time
}

type MemoryStorage struct {
	mu   sync.Mutex
	data map[string]*memoryEntry
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		data: make(map[string]*memoryEntry),
	}
}

func (m *MemoryStorage) Increment(ctx context.Context, key string, window time.Duration) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()

	entry, exists := m.data[key]
	if !exists {
		m.data[key] = &memoryEntry{
			counter:     1,
			windowStart: now,
		}
		return 1, nil
	}

	if now.Sub(entry.windowStart) >= window {
		entry.counter = 1
		entry.windowStart = now
	} else {
		entry.counter++
	}

	return entry.counter, nil
}

func (m *MemoryStorage) IsBlocked(ctx context.Context, key string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, exists := m.data[key]
	if !exists {
		return false, nil
	}

	if entry.blockedUntil == nil {
		return false, nil
	}

	if time.Now().Before(*entry.blockedUntil) {
		return true, nil
	}

	entry.blockedUntil = nil
	return false, nil
}

func (m *MemoryStorage) Block(ctx context.Context, key string, duration time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	blockedUntil := time.Now().Add(duration)

	entry, exists := m.data[key]
	if !exists {
		m.data[key] = &memoryEntry{
			blockedUntil: &blockedUntil,
		}
		return nil
	}

	entry.blockedUntil = &blockedUntil
	return nil
}

func (m *MemoryStorage) Reset(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.data, key)
	return nil
}

func (m *MemoryStorage) Close() error {
	return nil
}
