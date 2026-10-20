package cache

import (
	"encoding/json"
	"sync"
	"time"
)

type memItem struct {
	saved time.Time
	data  []byte
}

// Mem is an in-memory store for consumers' tests.
type Mem struct {
	mu    sync.Mutex
	now   func() time.Time
	items map[string]memItem
}

var _ Store = (*Mem)(nil)

// NewMem creates a Mem; now supplies the save time (nil → time.Now).
func NewMem(now func() time.Time) *Mem {
	if now == nil {
		now = time.Now
	}
	return &Mem{now: now, items: map[string]memItem{}}
}

func (m *Mem) Load(key string, v any) (time.Time, bool, error) {
	if err := checkKey(key); err != nil {
		return time.Time{}, false, err
	}
	m.mu.Lock()
	it, ok := m.items[key]
	m.mu.Unlock()
	if !ok || json.Unmarshal(it.data, v) != nil {
		return time.Time{}, false, nil
	}
	return it.saved, true, nil
}

func (m *Mem) Save(key string, v any) error {
	if err := checkKey(key); err != nil {
		return err
	}
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	m.mu.Lock()
	m.items[key] = memItem{saved: m.now(), data: b}
	m.mu.Unlock()
	return nil
}
