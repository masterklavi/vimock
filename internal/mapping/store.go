package mapping

import (
	"sync"
	"sync/atomic"
)

type snapshot struct {
	mappings []Mapping
	byID     map[string]Mapping
}

// Store keeps mappings in memory and publishes immutable snapshots for lock-free reads.
type Store struct {
	mu       sync.Mutex
	byID     map[string]Mapping
	order    []string
	nextSeq  uint64
	snapshot atomic.Value
}

func NewStore() *Store {
	store := &Store{
		byID:  make(map[string]Mapping),
		order: make([]string, 0),
	}
	store.snapshot.Store(snapshot{
		mappings: []Mapping{},
		byID:     map[string]Mapping{},
	})
	return store
}

func (s *Store) List() []Mapping {
	current := s.loadSnapshot()
	mappings := make([]Mapping, len(current.mappings))
	copy(mappings, current.mappings)
	return mappings
}

func (s *Store) Range(yield func(Mapping) bool) {
	current := s.loadSnapshot()
	for _, mapping := range current.mappings {
		if !yield(mapping) {
			return
		}
	}
}

func (s *Store) Get(id string) (Mapping, bool) {
	current := s.loadSnapshot()
	mapping, ok := current.byID[id]
	return mapping, ok
}

func (s *Store) Create(mapping Mapping) Mapping {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.byID[mapping.id]; ok {
		mapping.sequence = existing.sequence
	} else {
		s.nextSeq++
		mapping.sequence = s.nextSeq
		s.order = append(s.order, mapping.id)
	}

	s.byID[mapping.id] = mapping
	s.publishLocked()
	return mapping
}

func (s *Store) Replace(id string, mapping Mapping) (Mapping, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.byID[id]
	if !ok {
		return Mapping{}, false
	}

	mapping.id = id
	mapping.raw = cloneRawMap(mapping.raw)
	mapping.raw["id"] = mustMarshalRaw(id)
	mapping.sequence = existing.sequence

	s.byID[id] = mapping
	s.publishLocked()
	return mapping, true
}

func (s *Store) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.byID[id]; !ok {
		return false
	}

	delete(s.byID, id)
	for i, orderedID := range s.order {
		if orderedID == id {
			s.order = append(s.order[:i], s.order[i+1:]...)
			break
		}
	}

	s.publishLocked()
	return true
}

func (s *Store) Count() int {
	return len(s.loadSnapshot().mappings)
}

func (s *Store) loadSnapshot() snapshot {
	return s.snapshot.Load().(snapshot)
}

func (s *Store) publishLocked() {
	mappings := make([]Mapping, 0, len(s.order))
	byID := make(map[string]Mapping, len(s.byID))

	for _, id := range s.order {
		mapping, ok := s.byID[id]
		if !ok {
			continue
		}
		mappings = append(mappings, mapping)
		byID[id] = mapping
	}

	s.snapshot.Store(snapshot{
		mappings: mappings,
		byID:     byID,
	})
}
