package scenario

import (
	"sync"

	"vimock/internal/mapping"
)

const Started = "Started"

type StateStore interface {
	SelectAndTransition(candidates []mapping.Mapping, compare func(left, right mapping.Mapping) int) (mapping.Mapping, bool)
}

type Store struct {
	mu      sync.Mutex
	states  map[string]string
	members map[string]map[string]struct{}
}

func NewStore() *Store {
	return &Store{
		states:  make(map[string]string),
		members: make(map[string]map[string]struct{}),
	}
}

func (s *Store) SelectAndTransition(candidates []mapping.Mapping, compare func(left, right mapping.Mapping) int) (mapping.Mapping, bool) {
	if s == nil {
		return selectBest(candidates, compare)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	selected, found := s.selectBestMatchingStateLocked(candidates, compare)
	if found {
		s.transitionLocked(selected)
	}
	return selected, found
}

func (s *Store) State(name string) string {
	if s == nil {
		return Started
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.stateLocked(name)
}

func (s *Store) Reset() {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	clear(s.states)
}

func (s *Store) MappingCreated(stub mapping.Mapping) {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.addMappingLocked(stub)
}

func (s *Store) MappingUpdated(oldStub, newStub mapping.Mapping) {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if oldStub.ScenarioName() == newStub.ScenarioName() {
		s.addMappingLocked(newStub)
		return
	}

	s.removeMappingLocked(oldStub)
	s.addMappingLocked(newStub)
}

func (s *Store) MappingDeleted(stub mapping.Mapping) {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.removeMappingLocked(stub)
}

func (s *Store) selectBestMatchingStateLocked(candidates []mapping.Mapping, compare func(left, right mapping.Mapping) int) (mapping.Mapping, bool) {
	var selected mapping.Mapping
	var found bool
	for _, stub := range candidates {
		if !s.matchesStateLocked(stub) {
			continue
		}
		if !found || compare(stub, selected) < 0 {
			selected = stub
			found = true
		}
	}
	return selected, found
}

func (s *Store) matchesStateLocked(stub mapping.Mapping) bool {
	scenarioDefinition := stub.Scenario()
	if scenarioDefinition.Name == "" || scenarioDefinition.RequiredState == "" {
		return true
	}
	return s.stateLocked(scenarioDefinition.Name) == scenarioDefinition.RequiredState
}

func (s *Store) transitionLocked(stub mapping.Mapping) {
	scenarioDefinition := stub.Scenario()
	if scenarioDefinition.Name == "" || scenarioDefinition.NewState == "" {
		return
	}
	if scenarioDefinition.RequiredState != "" && s.stateLocked(scenarioDefinition.Name) != scenarioDefinition.RequiredState {
		return
	}
	s.states[scenarioDefinition.Name] = scenarioDefinition.NewState
}

func (s *Store) stateLocked(name string) string {
	if state, ok := s.states[name]; ok && state != "" {
		return state
	}
	return Started
}

func (s *Store) addMappingLocked(stub mapping.Mapping) {
	name := stub.ScenarioName()
	if name == "" {
		return
	}

	if s.members[name] == nil {
		s.members[name] = make(map[string]struct{})
	}
	s.members[name][stub.ID()] = struct{}{}
}

func (s *Store) removeMappingLocked(stub mapping.Mapping) {
	name := stub.ScenarioName()
	if name == "" {
		return
	}

	scenarioMembers := s.members[name]
	if scenarioMembers == nil {
		delete(s.states, name)
		return
	}

	delete(scenarioMembers, stub.ID())
	if len(scenarioMembers) == 0 {
		delete(s.members, name)
		delete(s.states, name)
	}
}

func selectBest(candidates []mapping.Mapping, compare func(left, right mapping.Mapping) int) (mapping.Mapping, bool) {
	var selected mapping.Mapping
	var found bool
	for _, stub := range candidates {
		if !found || compare(stub, selected) < 0 {
			selected = stub
			found = true
		}
	}
	return selected, found
}
