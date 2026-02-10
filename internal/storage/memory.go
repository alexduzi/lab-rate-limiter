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
	data sync.Map
	mu   sync.RWMutex
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		data: sync.Map{},
	}
}

func (m *MemoryStorage) Increment(ctx context.Context, key string, window time.Duration) (int64, error) {
	now := time.Now()

	value, exists := m.data.Load(key)

	if !exists {
		entry := &memoryEntry{
			counter:     1,
			windowStart: now,
		}
		m.data.Store(key, entry)
		return 1, nil
	}

	entry := value.(*memoryEntry)

	// Se passou a janela, reseta
	if now.Sub(entry.windowStart) >= window {
		entry.counter = 1
		entry.windowStart = now
	} else {
		entry.counter++
	}

	m.data.Store(key, entry)
	return entry.counter, nil
}

func (m *MemoryStorage) IsBlocked(ctx context.Context, key string) (bool, error) {
	value, exists := m.data.Load(key)
	if !exists {
		return false, nil
	}

	entry := value.(*memoryEntry)

	if entry.blockedUntil == nil {
		return false, nil
	}

	if time.Now().Before(*entry.blockedUntil) {
		return true, nil
	}

	// Bloqueio expirou
	entry.blockedUntil = nil
	m.data.Store(key, entry)
	return false, nil
}

func (m *MemoryStorage) Block(ctx context.Context, key string, duration time.Duration) error {
	blockedUntil := time.Now().Add(duration)

	value, exists := m.data.Load(key)

	if !exists {
		entry := &memoryEntry{
			blockedUntil: &blockedUntil,
		}
		m.data.Store(key, entry)
		return nil
	}

	entry := value.(*memoryEntry)
	entry.blockedUntil = &blockedUntil
	m.data.Store(key, entry)

	return nil
}

func (m *MemoryStorage) Reset(ctx context.Context, key string) error {
	m.data.Delete(key)
	return nil
}

func (m *MemoryStorage) Close() error {
	return nil
}
