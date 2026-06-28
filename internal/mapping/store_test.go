package mapping

import (
	"fmt"
	"sync"
	"testing"
)

func TestStoreCRUD(t *testing.T) {
	store := NewStore()
	first := store.Create(mustParseMapping(t, "first"))
	second := store.Create(mustParseMapping(t, "second"))

	if store.Count() != 2 {
		t.Fatalf("count = %d, want 2", store.Count())
	}

	mappings := store.List()
	if len(mappings) != 2 {
		t.Fatalf("list len = %d, want 2", len(mappings))
	}
	if mappings[0].ID() != first.ID() || mappings[1].ID() != second.ID() {
		t.Fatalf("list order = [%s, %s], want insertion order [%s, %s]", mappings[0].ID(), mappings[1].ID(), first.ID(), second.ID())
	}

	updated, ok := store.Replace(first.ID(), mustParseMapping(t, "updated"))
	if !ok {
		t.Fatalf("Replace() ok = false, want true")
	}
	if updated.ID() != first.ID() {
		t.Fatalf("updated id = %q, want %q", updated.ID(), first.ID())
	}
	if updated.Sequence() != first.Sequence() {
		t.Fatalf("updated sequence = %d, want %d", updated.Sequence(), first.Sequence())
	}

	got, ok := store.Get(first.ID())
	if !ok {
		t.Fatalf("Get() ok = false, want true")
	}
	if got.Name() != "updated" {
		t.Fatalf("name = %q, want updated", got.Name())
	}

	if !store.Delete(first.ID()) {
		t.Fatalf("Delete() ok = false, want true")
	}
	if _, ok := store.Get(first.ID()); ok {
		t.Fatalf("Get() after delete ok = true, want false")
	}
	if store.Delete(first.ID()) {
		t.Fatalf("second Delete() ok = true, want false")
	}
}

func TestStoreConcurrentAccess(t *testing.T) {
	store := NewStore()

	var wg sync.WaitGroup
	errCh := make(chan error, 200)
	for i := 0; i < 100; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()

			created := store.Create(mustParseMapping(t, fmt.Sprintf("mapping-%d", i)))
			if _, ok := store.Get(created.ID()); !ok {
				errCh <- fmt.Errorf("created mapping %s not found", created.ID())
				return
			}

			_ = store.List()
			if i%2 == 0 {
				store.Delete(created.ID())
			}
			_ = store.List()
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Error(err)
	}
}

func TestStoreRangeIteratesSnapshotWithoutExposingListCopy(t *testing.T) {
	store := NewStore()
	first := store.Create(mustParseMapping(t, "first"))
	second := store.Create(mustParseMapping(t, "second"))

	var ids []string
	store.Range(func(stub Mapping) bool {
		ids = append(ids, stub.ID())
		return true
	})

	if len(ids) != 2 || ids[0] != first.ID() || ids[1] != second.ID() {
		t.Fatalf("range ids = %#v, want [%s %s]", ids, first.ID(), second.ID())
	}

	var limited []string
	store.Range(func(stub Mapping) bool {
		limited = append(limited, stub.ID())
		return false
	})
	if len(limited) != 1 || limited[0] != first.ID() {
		t.Fatalf("limited range ids = %#v, want first id only", limited)
	}
}

func TestStoreRangeCandidatesUsesExactIndexesAndFallback(t *testing.T) {
	store := NewStore()
	exactURL := store.Create(parseMappingJSON(t, `{
	  "name": "exact-url",
	  "request": {"method": "GET", "url": "/resource?id=1"},
	  "response": {"status": 200}
	}`))
	exactPath := store.Create(parseMappingJSON(t, `{
	  "name": "exact-path",
	  "request": {"method": "ANY", "urlPath": "/resource"},
	  "response": {"status": 200}
	}`))
	regex := store.Create(parseMappingJSON(t, `{
	  "name": "regex",
	  "request": {"method": "GET", "urlPathPattern": "/resource/.*"},
	  "response": {"status": 200}
	}`))
	store.Create(parseMappingJSON(t, `{
	  "name": "other",
	  "request": {"method": "POST", "urlPath": "/other"},
	  "response": {"status": 200}
	}`))

	var ids []string
	store.RangeCandidates("GET", "/resource?id=1", "/resource", func(stub Mapping) bool {
		ids = append(ids, stub.ID())
		return true
	})

	want := []string{exactURL.ID(), exactPath.ID(), regex.ID()}
	if fmt.Sprint(ids) != fmt.Sprint(want) {
		t.Fatalf("candidate ids = %v, want %v", ids, want)
	}
}

func mustParseMapping(t *testing.T, name string) Mapping {
	t.Helper()

	stub, err := ParseJSON([]byte(fmt.Sprintf(`{
	  "name": %q,
	  "request": {"method": "GET", "url": "/%s"},
	  "response": {"status": 200}
	}`, name, name)))
	if err != nil {
		t.Fatalf("ParseJSON(): %v", err)
	}
	return stub
}

func parseMappingJSON(t *testing.T, body string) Mapping {
	t.Helper()

	stub, err := ParseJSON([]byte(body))
	if err != nil {
		t.Fatalf("ParseJSON(): %v", err)
	}
	return stub
}
