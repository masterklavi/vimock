package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"

	"vimock/internal/files"
	"vimock/internal/grpcdesc"
	"vimock/internal/mapping"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

const (
	pdmServiceName = "pdm_api_gateway.v1.MCProduct"
	pdmMethodName  = "WarehousesByNomenclature"
	pdmFullMethod  = "/" + pdmServiceName + "/" + pdmMethodName
)

func TestGRPCRuntimeServesPDMFixture(t *testing.T) {
	descriptorStore := newLoadedDescriptorStore(t)
	registry := descriptorStore.Active()
	method := mustGRPCMethod(t, registry, pdmServiceName, pdmMethodName)

	mappingStore := mapping.NewStore()
	mappingStore.Create(mustParseMappingFile(t, "../../testdata/grpc_mapping.json"))
	handler := NewHandlerWithStoresAndDescriptors(nil, mappingStore, files.NewMemoryStore(), descriptorStore)

	input := newDynamicMessageFromJSON(t, method.Input(), registry, `{"guids":["b27ed95d-3717-4538-9be6-a7136b8ad52f"]}`)
	resp := requestGRPC(t, handler, pdmFullMethod, input)
	output := decodeGRPCResponse(t, resp, method.Output(), registry)

	var body map[string]any
	outputJSON := marshalDynamicMessageJSON(t, output, registry)
	if err := json.Unmarshal(outputJSON, &body); err != nil {
		t.Fatalf("decode output JSON %s: %v", outputJSON, err)
	}

	warehouses := body["warehouses"].([]any)
	if len(warehouses) != 1 {
		t.Fatalf("warehouses len = %d, want 1", len(warehouses))
	}
	item := warehouses[0].(map[string]any)
	if item["nomenclature_guid"] != "b27ed95d-3717-4538-9be6-a7136b8ad52f" {
		t.Fatalf("nomenclature_guid = %v", item["nomenclature_guid"])
	}
	warehouseGuids := item["warehouses_guid"].([]any)
	if len(warehouseGuids) != 1 || warehouseGuids[0] != "00000000-0000-0000-0000-050258290258" {
		t.Fatalf("warehouses_guid = %v", warehouseGuids)
	}
}

func TestGRPCRuntimeSupportsNonOKStatusHeader(t *testing.T) {
	descriptorStore := newLoadedDescriptorStore(t)
	registry := descriptorStore.Active()
	method := mustGRPCMethod(t, registry, pdmServiceName, pdmMethodName)

	mappingStore := mapping.NewStore()
	mappingStore.Create(mustParseMappingJSON(t, `{
	  "request": {
	    "method": "POST",
	    "urlPath": "/pdm_api_gateway.v1.MCProduct/WarehousesByNomenclature"
	  },
	  "response": {
	    "status": 200,
	    "headers": {
	      "grpc-status-name": "NOT_FOUND",
	      "grpc-status-reason": "missing warehouse"
	    }
	  }
	}`))
	handler := NewHandlerWithStoresAndDescriptors(nil, mappingStore, files.NewMemoryStore(), descriptorStore)

	input := newDynamicMessageFromJSON(t, method.Input(), registry, `{"guids":["missing"]}`)
	resp := requestGRPC(t, handler, pdmFullMethod, input)
	assertGRPCStatus(t, resp, grpcStatusNotFound, "missing warehouse")
}

func TestGRPCRuntimeReturnsUnimplementedForUnmatchedRequest(t *testing.T) {
	descriptorStore := newLoadedDescriptorStore(t)
	registry := descriptorStore.Active()
	method := mustGRPCMethod(t, registry, pdmServiceName, pdmMethodName)

	handler := NewHandlerWithStoresAndDescriptors(nil, mapping.NewStore(), files.NewMemoryStore(), descriptorStore)

	input := newDynamicMessageFromJSON(t, method.Input(), registry, `{"guids":["missing"]}`)
	resp := requestGRPC(t, handler, pdmFullMethod, input)
	assertGRPCStatus(t, resp, grpcStatusUnimplemented, "No matching stub mapping found for gRPC request")
}

func TestGRPCRuntimeConvertsBinaryMetadataForHeaderMatching(t *testing.T) {
	descriptorStore := newLoadedDescriptorStore(t)
	registry := descriptorStore.Active()
	method := mustGRPCMethod(t, registry, pdmServiceName, pdmMethodName)

	mappingStore := mapping.NewStore()
	mappingStore.Create(mustParseMappingJSON(t, `{
	  "request": {
	    "method": "POST",
	    "urlPath": "/pdm_api_gateway.v1.MCProduct/WarehousesByNomenclature",
	    "headers": {
	      "trace-bin": {
	        "equalTo": "[1, 2, 3]"
	      }
	    }
	  },
	  "response": {
	    "status": 200,
	    "body": "{\"warehouses\": []}"
	  }
	}`))
	handler := NewHandlerWithStoresAndDescriptors(nil, mappingStore, files.NewMemoryStore(), descriptorStore)

	input := newDynamicMessageFromJSON(t, method.Input(), registry, `{"guids":["b27ed95d-3717-4538-9be6-a7136b8ad52f"]}`)
	resp := requestGRPCWithHeaders(t, handler, pdmFullMethod, input, map[string]string{
		"trace-bin": "AQID",
	})
	_ = decodeGRPCResponse(t, resp, method.Output(), registry)
}

func newLoadedDescriptorStore(t *testing.T) *grpcdesc.Store {
	t.Helper()

	data, err := os.ReadFile("../../testdata/mc_product.dsc")
	if err != nil {
		t.Fatalf("read descriptor: %v", err)
	}
	store := grpcdesc.NewStore()
	if _, err := store.Put("mc_product.dsc", data); err != nil {
		t.Fatalf("put descriptor: %v", err)
	}
	if _, err := store.Reset(); err != nil {
		t.Fatalf("reset descriptor registry: %v", err)
	}
	return store
}

func mustParseMappingFile(t *testing.T, path string) mapping.Mapping {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read mapping %s: %v", path, err)
	}
	return mustParseMappingBytes(t, data)
}

func mustParseMappingJSON(t *testing.T, body string) mapping.Mapping {
	t.Helper()
	return mustParseMappingBytes(t, []byte(body))
}

func mustParseMappingBytes(t *testing.T, body []byte) mapping.Mapping {
	t.Helper()

	stub, err := mapping.ParseJSON(body)
	if err != nil {
		t.Fatalf("parse mapping: %v", err)
	}
	return stub
}

func mustGRPCMethod(t *testing.T, registry grpcdesc.Registry, serviceName, methodName string) protoreflect.MethodDescriptor {
	t.Helper()

	service, ok := registry.FindService(serviceName)
	if !ok {
		t.Fatalf("service %s not found in descriptor registry", serviceName)
	}
	method := service.Methods().ByName(protoreflect.Name(methodName))
	if method == nil {
		t.Fatalf("method %s/%s not found in descriptor registry", serviceName, methodName)
	}
	return method
}

func newDynamicMessageFromJSON(t *testing.T, descriptor protoreflect.MessageDescriptor, registry grpcdesc.Registry, body string) *dynamicpb.Message {
	t.Helper()

	message := dynamicpb.NewMessage(descriptor)
	if err := (protojson.UnmarshalOptions{
		Resolver: registry.TypeResolver(),
	}).Unmarshal([]byte(body), message); err != nil {
		t.Fatalf("unmarshal dynamic JSON %s: %v", body, err)
	}
	return message
}

func marshalDynamicMessageJSON(t *testing.T, message *dynamicpb.Message, registry grpcdesc.Registry) []byte {
	t.Helper()

	body, err := protojson.MarshalOptions{
		UseProtoNames: true,
		Resolver:      registry.TypeResolver(),
	}.Marshal(message)
	if err != nil {
		t.Fatalf("marshal dynamic JSON: %v", err)
	}
	return body
}

func requestGRPC(t *testing.T, handler http.Handler, path string, message proto.Message) *httptest.ResponseRecorder {
	t.Helper()
	return requestGRPCWithHeaders(t, handler, path, message, nil)
}

func requestGRPCWithHeaders(t *testing.T, handler http.Handler, path string, message proto.Message, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	payload, err := proto.Marshal(message)
	if err != nil {
		t.Fatalf("marshal request protobuf: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(encodeGRPCFrame(payload)))
	req.Header.Set("Content-Type", grpcContentType)
	for name, value := range headers {
		req.Header.Set(name, value)
	}
	req.ProtoMajor = 2
	req.ProtoMinor = 0

	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	return resp
}

func decodeGRPCResponse(t *testing.T, resp *httptest.ResponseRecorder, descriptor protoreflect.MessageDescriptor, registry grpcdesc.Registry) *dynamicpb.Message {
	t.Helper()

	assertGRPCStatus(t, resp, grpcStatusOK, "")
	payload, err := decodeUnaryGRPCFrame(resp.Body.Bytes())
	if err != nil {
		t.Fatalf("decode response frame: %v", err)
	}

	message := dynamicpb.NewMessage(descriptor)
	if err := proto.Unmarshal(payload, message); err != nil {
		t.Fatalf("unmarshal response protobuf: %v", err)
	}
	return message
}

func assertGRPCStatus(t *testing.T, resp *httptest.ResponseRecorder, wantCode int, wantMessage string) {
	t.Helper()

	if resp.Code != http.StatusOK {
		t.Fatalf("http status = %d, want 200: %s", resp.Code, resp.Body.String())
	}
	result := resp.Result()
	defer result.Body.Close()

	gotCode := result.Trailer.Get("Grpc-Status")
	if gotCode == "" {
		gotCode = result.Header.Get("Grpc-Status")
	}
	if gotCode != strconv.Itoa(wantCode) {
		t.Fatalf("grpc-status = %q, want %d; headers=%v trailers=%v", gotCode, wantCode, result.Header, result.Trailer)
	}
	if wantMessage == "" {
		return
	}
	gotMessage := result.Trailer.Get("Grpc-Message")
	if gotMessage == "" {
		gotMessage = result.Header.Get("Grpc-Message")
	}
	decodedMessage, err := url.QueryUnescape(gotMessage)
	if err != nil {
		t.Fatalf("decode grpc-message %q: %v", gotMessage, err)
	}
	if !strings.Contains(decodedMessage, wantMessage) {
		t.Fatalf("grpc-message = %q, want containing %q", decodedMessage, wantMessage)
	}
}
