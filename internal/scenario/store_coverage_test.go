package scenario

import (
	"testing"

	"vimock/internal/mapping"
)

func TestMappingUpdatedDifferentScenarioAndNilStoreBranches(t *testing.T) {
	store := NewStore()
	oldStub := parseMapping(t, `{
	  "scenarioName": "old",
	  "newScenarioState": "OldState",
	  "request": {"method": "GET", "urlPath": "/old"},
	  "response": {"body": "old"}
	}`)
	newStub := parseMapping(t, `{
	  "scenarioName": "new",
	  "newScenarioState": "NewState",
	  "request": {"method": "GET", "urlPath": "/new"},
	  "response": {"body": "new"}
	}`)
	store.MappingCreated(oldStub)
	store.SelectAndTransition([]mapping.Mapping{oldStub}, compareByPriorityAndSequence)
	store.MappingUpdated(oldStub, newStub)
	store.SelectAndTransition([]mapping.Mapping{newStub}, compareByPriorityAndSequence)
	if state := store.State("old"); state != Started {
		t.Fatalf("old state = %q", state)
	}
	if state := store.State("new"); state != "NewState" {
		t.Fatalf("new state = %q", state)
	}

	var nilStore *Store
	nilStore.MappingCreated(oldStub)
	nilStore.MappingUpdated(oldStub, newStub)
	nilStore.MappingDeleted(oldStub)
	nilStore.Reset()
	if state := nilStore.State("missing"); state != Started {
		t.Fatalf("nil state = %q", state)
	}
}

func TestSelectBestNilStore(t *testing.T) {
	first := parseMapping(t, `{"request":{"method":"GET","urlPath":"/"},"response":{"body":"first"},"priority":2}`)
	second := parseMapping(t, `{"request":{"method":"GET","urlPath":"/"},"response":{"body":"second"},"priority":1}`)
	var store *Store
	selected, ok := store.SelectAndTransition([]mapping.Mapping{first, second}, compareByPriorityAndSequence)
	if !ok || string(selected.Response().Body) != "second" {
		t.Fatalf("selected = %s ok=%v", selected.Response().Body, ok)
	}
}
