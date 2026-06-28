package files

import "sync"

type Store interface {
	Get(name string) ([]byte, bool)
	Put(name string, data []byte)
}

type MemoryStore struct {
	mu    sync.RWMutex
	files map[string][]byte
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		files: make(map[string][]byte),
	}
}

func (s *MemoryStore) Get(name string) ([]byte, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, ok := s.files[name]
	if !ok {
		return nil, false
	}
	return cloneBytes(data), true
}

func (s *MemoryStore) Put(name string, data []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.files[name] = cloneBytes(data)
}

func cloneBytes(data []byte) []byte {
	if data == nil {
		return nil
	}
	clone := make([]byte, len(data))
	copy(clone, data)
	return clone
}
