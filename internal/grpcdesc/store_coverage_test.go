package grpcdesc

import (
	"os"
	"testing"

	"google.golang.org/protobuf/reflect/protoregistry"
)

func TestPutLegacyAndProtoSourceBranches(t *testing.T) {
	data, err := os.ReadFile("../../testdata/mc_product.dsc")
	if err != nil {
		t.Fatalf("read descriptor: %v", err)
	}
	store := NewStore()
	if !store.PutLegacy("legacy.dsc", data) {
		t.Fatal("PutLegacy valid descriptor = false")
	}
	if store.PutLegacy("schema.proto", []byte("syntax = \"proto3\";")) {
		t.Fatal("PutLegacy proto source = true, want false")
	}
	if store.PutLegacy("bad.dsc", []byte("bad")) {
		t.Fatal("PutLegacy bad descriptor = true, want false")
	}
	var nilStore *Store
	if nilStore.PutLegacy("legacy.dsc", data) {
		t.Fatal("nil Store PutLegacy = true, want false")
	}

	exists, err := store.Put("schema.proto", []byte("syntax = \"proto3\";"))
	if err != nil || exists {
		t.Fatalf("Put proto source exists=%v err=%v", exists, err)
	}
	files, _ := store.List()
	if len(files) != 2 {
		t.Fatalf("files len = %d, want 2", len(files))
	}
	if _, err := store.Reset(); err != nil {
		t.Fatalf("Reset(): %v", err)
	}
}

func TestRegistryLookupAndResolverBranches(t *testing.T) {
	var empty Registry
	if _, ok := empty.FindService("missing.Service"); ok {
		t.Fatal("empty FindService ok = true")
	}
	if _, ok := empty.FindMessageType("missing.Message"); ok {
		t.Fatal("empty FindMessageType ok = true")
	}
	if empty.TypeResolver() != protoregistry.GlobalTypes {
		t.Fatal("empty TypeResolver should use GlobalTypes")
	}

	data, err := os.ReadFile("../../testdata/mc_product.dsc")
	if err != nil {
		t.Fatalf("read descriptor: %v", err)
	}
	store := NewStore()
	if _, err := store.Put("mc_product.dsc", data); err != nil {
		t.Fatalf("Put(): %v", err)
	}
	registry, err := store.Reset()
	if err != nil {
		t.Fatalf("Reset(): %v", err)
	}
	if _, ok := registry.FindService("missing.Service"); ok {
		t.Fatal("missing service ok = true")
	}
	if _, ok := registry.FindService("pdm_api_gateway.v1.MCProduct"); !ok {
		t.Fatal("service not found")
	}
	if _, ok := registry.FindMessageType("missing.Message"); ok {
		t.Fatal("missing message ok = true")
	}
	if _, ok := registry.FindMessageType(registry.Messages[0]); !ok {
		t.Fatal("known message not found")
	}
	if registry.TypeResolver() == nil {
		t.Fatal("TypeResolver() = nil")
	}
}

func TestValidateBlobBranches(t *testing.T) {
	if err := validateBlob("unknown", []byte("x")); err == nil {
		t.Fatal("unknown kind error = nil")
	}
	if err := validateBlob(KindProtoSource, []byte{0xff}); err == nil {
		t.Fatal("invalid UTF-8 proto error = nil")
	}
	if err := validateBlob(KindDescriptorSet, nil); err == nil {
		t.Fatal("empty descriptor error = nil")
	}
}

func TestDeleteListActiveNilBranches(t *testing.T) {
	var store *Store
	files, registry := store.List()
	if files != nil || registry.Files != 0 {
		t.Fatalf("nil list = %v %+v", files, registry)
	}
	if registry := store.Active(); registry.Files != 0 {
		t.Fatalf("nil active = %+v", registry)
	}
	if _, err := store.Reset(); err != nil {
		t.Fatalf("nil reset: %v", err)
	}
}
