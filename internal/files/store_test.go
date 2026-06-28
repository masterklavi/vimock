package files

import (
	"bytes"
	"testing"
)

func TestMemoryStorePutGetCopiesBytes(t *testing.T) {
	store := NewMemoryStore()
	original := []byte{0x01, 0x02, 0x03}
	store.Put("payload.bin", original)
	original[0] = 0xff

	got, ok := store.Get("payload.bin")
	if !ok {
		t.Fatalf("Get() ok = false, want true")
	}
	if !bytes.Equal(got, []byte{0x01, 0x02, 0x03}) {
		t.Fatalf("stored bytes = %v, want original copy", got)
	}

	got[1] = 0xff
	gotAgain, ok := store.Get("payload.bin")
	if !ok {
		t.Fatalf("second Get() ok = false, want true")
	}
	if !bytes.Equal(gotAgain, []byte{0x01, 0x02, 0x03}) {
		t.Fatalf("stored bytes after mutating result = %v, want unchanged", gotAgain)
	}
}

func TestMemoryStoreGetMissing(t *testing.T) {
	store := NewMemoryStore()
	if _, ok := store.Get("missing.bin"); ok {
		t.Fatalf("Get() ok = true, want false")
	}
}
