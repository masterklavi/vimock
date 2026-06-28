package scenario

import (
	"sync"
	"testing"

	"vimock/internal/mapping"
)

func TestSelectAndTransitionStartsAtStartedAndAdvances(t *testing.T) {
	store := NewStore()
	first := parseMapping(t, `{
	  "scenarioName": "login",
	  "requiredScenarioState": "Started",
	  "newScenarioState": "Authenticated",
	  "request": {
	    "method": "GET",
	    "urlPath": "/login"
	  },
	  "response": {
	    "status": 401,
	    "body": "first"
	  },
	  "priority": 1
	}`)
	second := parseMapping(t, `{
	  "scenarioName": "login",
	  "requiredScenarioState": "Authenticated",
	  "request": {
	    "method": "GET",
	    "urlPath": "/login"
	  },
	  "response": {
	    "status": 200,
	    "body": "second"
	  },
	  "priority": 1
	}`)
	candidates := []mapping.Mapping{first, second}

	selected, ok := store.SelectAndTransition(candidates, compareByPriorityAndSequence)
	if !ok {
		t.Fatalf("first selection ok = false, want true")
	}
	if string(selected.Response().Body) != "first" {
		t.Fatalf("first body = %q, want first", selected.Response().Body)
	}
	if state := store.State("login"); state != "Authenticated" {
		t.Fatalf("state = %q, want Authenticated", state)
	}

	selected, ok = store.SelectAndTransition(candidates, compareByPriorityAndSequence)
	if !ok {
		t.Fatalf("second selection ok = false, want true")
	}
	if string(selected.Response().Body) != "second" {
		t.Fatalf("second body = %q, want second", selected.Response().Body)
	}
}

func TestResetReturnsScenarioToStarted(t *testing.T) {
	store := NewStore()
	stub := parseMapping(t, `{
	  "scenarioName": "order",
	  "requiredScenarioState": "Started",
	  "newScenarioState": "Done",
	  "request": {
	    "method": "GET",
	    "urlPath": "/order"
	  },
	  "response": {
	    "body": "done"
	  }
	}`)

	store.MappingCreated(stub)
	store.SelectAndTransition([]mapping.Mapping{stub}, compareByPriorityAndSequence)
	if state := store.State("order"); state != "Done" {
		t.Fatalf("state = %q, want Done", state)
	}

	store.Reset()
	if state := store.State("order"); state != Started {
		t.Fatalf("state after reset = %q, want %s", state, Started)
	}
}

func TestDeletingLastScenarioMappingRemovesState(t *testing.T) {
	store := NewStore()
	stub := parseMapping(t, `{
	  "scenarioName": "single",
	  "newScenarioState": "Changed",
	  "request": {
	    "method": "GET",
	    "urlPath": "/single"
	  },
	  "response": {
	    "body": "changed"
	  }
	}`)

	store.MappingCreated(stub)
	store.SelectAndTransition([]mapping.Mapping{stub}, compareByPriorityAndSequence)
	if state := store.State("single"); state != "Changed" {
		t.Fatalf("state = %q, want Changed", state)
	}

	store.MappingDeleted(stub)
	if state := store.State("single"); state != Started {
		t.Fatalf("state after delete = %q, want %s", state, Started)
	}
}

func TestUpdatingMappingInSameScenarioKeepsState(t *testing.T) {
	store := NewStore()
	oldStub := parseMapping(t, `{
	  "scenarioName": "same",
	  "newScenarioState": "Changed",
	  "request": {
	    "method": "GET",
	    "urlPath": "/same"
	  },
	  "response": {
	    "body": "old"
	  }
	}`)
	newStub, err := parseMapping(t, `{
	  "scenarioName": "same",
	  "request": {
	    "method": "GET",
	    "urlPath": "/same"
	  },
	  "response": {
	    "body": "new"
	  }
	}`).WithID(oldStub.ID())
	if err != nil {
		t.Fatalf("WithID() error = %v", err)
	}

	store.MappingCreated(oldStub)
	store.SelectAndTransition([]mapping.Mapping{oldStub}, compareByPriorityAndSequence)
	if state := store.State("same"); state != "Changed" {
		t.Fatalf("state = %q, want Changed", state)
	}

	store.MappingUpdated(oldStub, newStub)
	if state := store.State("same"); state != "Changed" {
		t.Fatalf("state after same-scenario update = %q, want Changed", state)
	}
}

func TestSelectAndTransitionIsConcurrentSafe(t *testing.T) {
	store := NewStore()
	first := parseMapping(t, `{
	  "scenarioName": "concurrent",
	  "requiredScenarioState": "Started",
	  "newScenarioState": "Done",
	  "request": {
	    "method": "GET",
	    "urlPath": "/concurrent"
	  },
	  "response": {
	    "body": "first"
	  },
	  "priority": 1
	}`)
	done := parseMapping(t, `{
	  "scenarioName": "concurrent",
	  "requiredScenarioState": "Done",
	  "newScenarioState": "Done",
	  "request": {
	    "method": "GET",
	    "urlPath": "/concurrent"
	  },
	  "response": {
	    "body": "done"
	  },
	  "priority": 1
	}`)
	candidates := []mapping.Mapping{first, done}

	const requests = 32
	var wg sync.WaitGroup
	results := make(chan string, requests)
	for range requests {
		wg.Add(1)
		go func() {
			defer wg.Done()
			selected, ok := store.SelectAndTransition(candidates, compareByPriorityAndSequence)
			if !ok {
				results <- "none"
				return
			}
			results <- string(selected.Response().Body)
		}()
	}
	wg.Wait()
	close(results)

	var firstCount int
	for result := range results {
		if result == "first" {
			firstCount++
		}
	}
	if firstCount != 1 {
		t.Fatalf("first response count = %d, want 1", firstCount)
	}
}

func parseMapping(t *testing.T, body string) mapping.Mapping {
	t.Helper()

	stub, err := mapping.ParseJSON([]byte(body))
	if err != nil {
		t.Fatalf("parse mapping: %v", err)
	}
	return stub
}

func compareByPriorityAndSequence(left, right mapping.Mapping) int {
	if left.Priority() != right.Priority() {
		return left.Priority() - right.Priority()
	}
	if left.Sequence() < right.Sequence() {
		return -1
	}
	if left.Sequence() > right.Sequence() {
		return 1
	}
	return 0
}
