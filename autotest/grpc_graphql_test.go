package autotest

import (
	"bytes"
	"encoding/binary"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

const (
	pdmServiceName = "pdm_api_gateway.v1.MCProduct"
	pdmMethodName  = "WarehousesByNomenclature"
	pdmFullPath    = "/" + pdmServiceName + "/" + pdmMethodName
)

func TestBlackBoxGRPCDescriptorUploadAndUnaryRuntime(t *testing.T) {
	s := requireTarget(t)
	descriptorSet := readTestdata(t, "mc_product.dsc")
	uploadLegacyFileAtPath(t, s, "grpc/mc_product.dsc", "mc_product.dsc", descriptorSet)

	protoName := strings.ReplaceAll("autotest_"+uniqueName(t)+".proto", "-", "_")
	protoResp, protoBody := s.request(t, http.MethodPut, "/__admin/ext/grpc/descriptors/"+url.PathEscape(protoName), []byte(`syntax = "proto3"; package autotest; message Ping { string id = 1; }`), map[string]string{"Content-Type": "text/plain"})
	if protoResp.StatusCode != http.StatusCreated && protoResp.StatusCode != http.StatusOK {
		t.Fatalf("proto upload status = %d, want 201/200: %s", protoResp.StatusCode, protoBody)
	}
	defer s.request(t, http.MethodDelete, "/__admin/ext/grpc/descriptors/"+url.PathEscape(protoName), nil, nil)

	descResp, descBody := s.request(t, http.MethodPut, "/__admin/ext/grpc/descriptors/mc_product.dsc", descriptorSet, map[string]string{"Content-Type": "application/octet-stream"})
	if descResp.StatusCode != http.StatusCreated && descResp.StatusCode != http.StatusOK {
		t.Fatalf("descriptor upload status = %d, want 201/200: %s", descResp.StatusCode, descBody)
	}
	resetResp, resetBody := s.request(t, http.MethodPost, "/__admin/ext/grpc/reset", nil, nil)
	expectStatus(t, resetResp, resetBody, http.StatusOK)

	listResp, listBody := s.request(t, http.MethodGet, "/__admin/ext/grpc/descriptors", nil, nil)
	expectStatus(t, listResp, listBody, http.StatusOK)
	if !bytes.Contains(listBody, []byte("mc_product.dsc")) || !bytes.Contains(listBody, []byte(protoName)) {
		t.Fatalf("descriptor list does not include uploaded files: %s", listBody)
	}

	mapping := readTestdata(t, "grpc_mapping.json")
	createMappingRaw(t, s, mapping)

	files, types := buildProtoRegistry(t, descriptorSet)
	method := findMethod(t, files, pdmServiceName, pdmMethodName)
	request := dynamicMessageFromJSON(t, method.Input(), types, `{"guids":["b27ed95d-3717-4538-9be6-a7136b8ad52f"]}`)
	resp, body := requestGRPC(t, s, pdmFullPath, request)
	expectStatus(t, resp, body, http.StatusOK)
	assertGRPCTrailer(t, resp, "0", "")
	responsePayload := decodeGRPCFrame(t, body)
	responseMessage := dynamicpb.NewMessage(method.Output())
	if err := proto.Unmarshal(responsePayload, responseMessage); err != nil {
		t.Fatalf("unmarshal gRPC response: %v", err)
	}
	responseJSON, err := protojson.MarshalOptions{UseProtoNames: true, Resolver: types}.Marshal(responseMessage)
	if err != nil {
		t.Fatalf("marshal gRPC response JSON: %v", err)
	}
	if !bytes.Contains(responseJSON, []byte("00000000-0000-0000-0000-050258290258")) {
		t.Fatalf("gRPC response JSON = %s", responseJSON)
	}

	createMapping(t, s, map[string]any{
		"priority": 0,
		"request": map[string]any{
			"method":  "POST",
			"urlPath": pdmFullPath,
			"bodyPatterns": []any{
				map[string]any{"matchesJsonPath": "$.guids[?(@ == 'missing')]"},
			},
		},
		"response": map[string]any{
			"status": 200,
			"headers": map[string]any{
				"grpc-status-name":   "NOT_FOUND",
				"grpc-status-reason": "missing warehouse",
			},
		},
	})
	missingRequest := dynamicMessageFromJSON(t, method.Input(), types, `{"guids":["missing"]}`)
	missingResp, missingBody := requestGRPC(t, s, pdmFullPath, missingRequest)
	expectStatus(t, missingResp, missingBody, http.StatusOK)
	assertGRPCTrailer(t, missingResp, "5", "missing%20warehouse")
}

func TestBlackBoxGraphQLSemanticMatcher(t *testing.T) {
	s := requireTarget(t)
	suffix := uniqueName(t)
	path := "/autotest/" + suffix + "/graphql"

	createMapping(t, s, map[string]any{
		"name":       "autotest-graphql-" + suffix,
		"persistent": true,
		"priority":   1,
		"request": map[string]any{
			"method":  "POST",
			"urlPath": path,
			"customMatcher": map[string]any{
				"name": "graphql-body-matcher",
				"parameters": map[string]any{
					"query":         "query GetHero($episode: Episode) { hero(episode: $episode) { name age friends { name } } }",
					"variables":     map[string]any{"episode": "JEDI"},
					"operationName": "GetHero",
				},
			},
		},
		"response": map[string]any{
			"status": 200,
			"jsonBody": map[string]any{
				"data": map[string]any{
					"hero": map[string]any{"name": "Luke Skywalker", "age": 19},
				},
			},
		},
	})

	positiveResp, positiveBody := s.requestJSON(t, http.MethodPost, path, map[string]any{
		"operationName": "GetHero",
		"variables":     map[string]any{"episode": "JEDI"},
		"query":         "query GetHero($episode: Episode) { hero(episode: $episode) { friends { name } age name } }",
	})
	expectStatus(t, positiveResp, positiveBody, http.StatusOK)
	if !bytes.Contains(positiveBody, []byte("Luke Skywalker")) {
		t.Fatalf("GraphQL response body = %s", positiveBody)
	}

	wrongVariablesResp, wrongVariablesBody := s.requestJSON(t, http.MethodPost, path, map[string]any{
		"operationName": "GetHero",
		"variables":     map[string]any{"episode": "EMPIRE"},
		"query":         "query GetHero($episode: Episode) { hero(episode: $episode) { age name friends { name } } }",
	})
	expectStatus(t, wrongVariablesResp, wrongVariablesBody, http.StatusNotFound)

	wrongOperationResp, wrongOperationBody := s.requestJSON(t, http.MethodPost, path, map[string]any{
		"operationName": "OtherHero",
		"variables":     map[string]any{"episode": "JEDI"},
		"query":         "query GetHero($episode: Episode) { hero(episode: $episode) { age name friends { name } } }",
	})
	expectStatus(t, wrongOperationResp, wrongOperationBody, http.StatusNotFound)
}

func readTestdata(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("../testdata/" + name)
	if err != nil {
		t.Fatalf("read testdata %s: %v", name, err)
	}
	return data
}

func buildProtoRegistry(t *testing.T, descriptorSet []byte) (*protoregistry.Files, *protoregistry.Types) {
	t.Helper()
	var set descriptorpb.FileDescriptorSet
	if err := proto.Unmarshal(descriptorSet, &set); err != nil {
		t.Fatalf("unmarshal descriptor set: %v", err)
	}
	files, err := protodesc.NewFiles(&set)
	if err != nil {
		t.Fatalf("build file registry: %v", err)
	}
	types := &protoregistry.Types{}
	files.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		registerMessages(t, types, file.Messages())
		return true
	})
	return files, types
}

func registerMessages(t *testing.T, types *protoregistry.Types, messages protoreflect.MessageDescriptors) {
	t.Helper()
	for i := 0; i < messages.Len(); i++ {
		message := messages.Get(i)
		if err := types.RegisterMessage(dynamicpb.NewMessageType(message)); err != nil {
			t.Fatalf("register message %s: %v", message.FullName(), err)
		}
		registerMessages(t, types, message.Messages())
	}
}

func findMethod(t *testing.T, files *protoregistry.Files, serviceName, methodName string) protoreflect.MethodDescriptor {
	t.Helper()
	descriptor, err := files.FindDescriptorByName(protoreflect.FullName(serviceName))
	if err != nil {
		t.Fatalf("find service %s: %v", serviceName, err)
	}
	service, ok := descriptor.(protoreflect.ServiceDescriptor)
	if !ok {
		t.Fatalf("descriptor %s = %T, want service", serviceName, descriptor)
	}
	method := service.Methods().ByName(protoreflect.Name(methodName))
	if method == nil {
		t.Fatalf("method %s/%s not found", serviceName, methodName)
	}
	return method
}

func dynamicMessageFromJSON(t *testing.T, descriptor protoreflect.MessageDescriptor, resolver *protoregistry.Types, body string) *dynamicpb.Message {
	t.Helper()
	message := dynamicpb.NewMessage(descriptor)
	if err := (protojson.UnmarshalOptions{Resolver: resolver}).Unmarshal([]byte(body), message); err != nil {
		t.Fatalf("unmarshal dynamic JSON %s: %v", body, err)
	}
	return message
}

func requestGRPC(t *testing.T, s *target, path string, message proto.Message) (*http.Response, []byte) {
	t.Helper()
	payload, err := proto.Marshal(message)
	if err != nil {
		t.Fatalf("marshal gRPC request: %v", err)
	}
	return s.request(t, http.MethodPost, path, encodeGRPCFrame(payload), map[string]string{"Content-Type": "application/grpc"})
}

func encodeGRPCFrame(payload []byte) []byte {
	frame := make([]byte, 5+len(payload))
	binary.BigEndian.PutUint32(frame[1:5], uint32(len(payload)))
	copy(frame[5:], payload)
	return frame
}

func decodeGRPCFrame(t *testing.T, body []byte) []byte {
	t.Helper()
	if len(body) < 5 {
		t.Fatalf("gRPC frame is too short: %v", body)
	}
	if body[0] != 0 {
		t.Fatalf("compressed gRPC response is not supported in test")
	}
	length := int(binary.BigEndian.Uint32(body[1:5]))
	if 5+length != len(body) {
		t.Fatalf("gRPC frame length = %d, body len = %d", length, len(body))
	}
	return body[5:]
}

func assertGRPCTrailer(t *testing.T, resp *http.Response, wantStatus, wantMessage string) {
	t.Helper()
	status := resp.Trailer.Get("Grpc-Status")
	if status == "" {
		status = resp.Header.Get("Grpc-Status")
	}
	if status != wantStatus {
		t.Fatalf("Grpc-Status = %q, want %q; headers=%v trailers=%v", status, wantStatus, resp.Header, resp.Trailer)
	}
	message := resp.Trailer.Get("Grpc-Message")
	if message == "" {
		message = resp.Header.Get("Grpc-Message")
	}
	if wantMessage != "" && message != wantMessage {
		t.Fatalf("Grpc-Message = %q, want %q", message, wantMessage)
	}
	if wantMessage == "" && message != "" {
		t.Fatalf("Grpc-Message = %q, want empty", message)
	}
}
