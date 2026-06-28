package grpcdesc

import (
	"slices"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestStorePutListResetAndDeleteDescriptorSet(t *testing.T) {
	store := NewStore()
	descriptorSet := testDescriptorSetBytes(t)

	replaced, err := store.Put("service.dsc", descriptorSet)
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	if replaced {
		t.Fatalf("replaced = true, want false")
	}

	files, registry := store.List()
	if len(files) != 1 {
		t.Fatalf("files len = %d, want 1", len(files))
	}
	if files[0].Name != "service.dsc" || files[0].Kind != KindDescriptorSet || files[0].Size != len(descriptorSet) {
		t.Fatalf("file info = %+v, want descriptor-set service.dsc size %d", files[0], len(descriptorSet))
	}
	if !files[0].Loadable {
		t.Fatalf("file loadable = false, want true")
	}
	if registry.Files != 0 {
		t.Fatalf("registry files before reset = %d, want 0", registry.Files)
	}

	registry, err = store.Reset()
	if err != nil {
		t.Fatalf("Reset() error = %v", err)
	}
	if registry.Files != 1 {
		t.Fatalf("registry files = %d, want 1", registry.Files)
	}
	if !slices.Contains(registry.Services, "vimock.grpc.test.PingService") {
		t.Fatalf("services = %v, want PingService", registry.Services)
	}
	if !slices.Contains(registry.Messages, "vimock.grpc.test.PingRequest") {
		t.Fatalf("messages = %v, want PingRequest", registry.Messages)
	}
	service, ok := registry.FindService("vimock.grpc.test.PingService")
	if !ok || string(service.FullName()) != "vimock.grpc.test.PingService" {
		t.Fatalf("FindService() = %v, %v, want PingService", service, ok)
	}
	messageType, ok := registry.FindMessageType("vimock.grpc.test.PingRequest")
	if !ok || string(messageType.Descriptor().FullName()) != "vimock.grpc.test.PingRequest" {
		t.Fatalf("FindMessageType() = %v, %v, want PingRequest", messageType, ok)
	}

	replaced, err = store.Put("service.dsc", descriptorSet)
	if err != nil {
		t.Fatalf("replace Put() error = %v", err)
	}
	if !replaced {
		t.Fatalf("replaced = false, want true")
	}

	if !store.Delete("service.dsc") {
		t.Fatalf("Delete() = false, want true")
	}
	registry, err = store.Reset()
	if err != nil {
		t.Fatalf("Reset() after delete error = %v", err)
	}
	if registry.Files != 0 {
		t.Fatalf("registry files after delete = %d, want 0", registry.Files)
	}
}

func TestStoreRejectsInvalidDescriptors(t *testing.T) {
	store := NewStore()

	tests := []struct {
		name string
		body []byte
	}{
		{name: "bad.txt", body: []byte("ignored")},
		{name: "bad.dsc", body: []byte("not a descriptor set")},
		{name: "../bad.dsc", body: testDescriptorSetBytes(t)},
		{name: "empty.dsc", body: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := store.Put(tt.name, tt.body); err == nil {
				t.Fatalf("Put(%q) error = nil, want error", tt.name)
			}
		})
	}
}

func TestStoreAcceptsProtoSourceButDoesNotActivateIt(t *testing.T) {
	store := NewStore()
	if _, err := store.Put("service.proto", []byte(`syntax = "proto3";`)); err != nil {
		t.Fatalf("Put proto source error = %v", err)
	}

	files, registry := store.List()
	if len(files) != 1 || files[0].Kind != KindProtoSource {
		t.Fatalf("files = %+v, want one proto source", files)
	}
	if files[0].Loadable {
		t.Fatalf("proto source loadable = true, want false until proto compilation is implemented")
	}
	if _, err := store.Reset(); err != nil {
		t.Fatalf("Reset() with proto source error = %v", err)
	}
	if registry.Files != 0 {
		t.Fatalf("registry files before reset = %d, want 0", registry.Files)
	}
	if active := store.Active(); active.Files != 0 {
		t.Fatalf("active files = %d, want 0", active.Files)
	}
}

func testDescriptorSetBytes(t *testing.T) []byte {
	t.Helper()

	fieldName := "name"
	fieldNumber := int32(1)
	requestName := "PingRequest"
	responseName := "PingResponse"
	serviceName := "PingService"
	methodName := "Ping"
	inputType := ".vimock.grpc.test.PingRequest"
	outputType := ".vimock.grpc.test.PingResponse"

	descriptorSet := &descriptorpb.FileDescriptorSet{
		File: []*descriptorpb.FileDescriptorProto{
			{
				Name:    proto.String("ping.proto"),
				Package: proto.String("vimock.grpc.test"),
				Syntax:  proto.String("proto3"),
				MessageType: []*descriptorpb.DescriptorProto{
					{
						Name: proto.String(requestName),
						Field: []*descriptorpb.FieldDescriptorProto{
							{
								Name:     proto.String(fieldName),
								Number:   proto.Int32(fieldNumber),
								Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
								Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
								JsonName: proto.String(fieldName),
							},
						},
					},
					{
						Name: proto.String(responseName),
						Field: []*descriptorpb.FieldDescriptorProto{
							{
								Name:     proto.String("message"),
								Number:   proto.Int32(1),
								Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
								Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
								JsonName: proto.String("message"),
							},
						},
					},
				},
				Service: []*descriptorpb.ServiceDescriptorProto{
					{
						Name: proto.String(serviceName),
						Method: []*descriptorpb.MethodDescriptorProto{
							{
								Name:       proto.String(methodName),
								InputType:  proto.String(inputType),
								OutputType: proto.String(outputType),
							},
						},
					},
				},
			},
		},
	}

	data, err := proto.Marshal(descriptorSet)
	if err != nil {
		t.Fatalf("marshal descriptor set: %v", err)
	}
	return data
}
