package mapping

import "testing"

func TestStoreHasScenariosAndCandidateStop(t *testing.T) {
	store := NewStore()
	if store.HasScenarios() {
		t.Fatal("empty store HasScenarios = true")
	}
	stub := parseMappingJSON(t, `{
	  "scenarioName": "flow",
	  "request": {"method": "GET", "urlPath": "/flow"},
	  "response": {"status": 200}
	}`)
	store.Create(stub)
	if !store.HasScenarios() {
		t.Fatal("store HasScenarios = false after scenario mapping")
	}
	calls := 0
	store.RangeCandidates("GET", "/flow", "/flow", func(Mapping) bool {
		calls++
		return false
	})
	if calls != 1 {
		t.Fatalf("candidate calls = %d, want 1", calls)
	}
}
