package server

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"vimock/internal/delay"
	"vimock/internal/grpcdesc"
	"vimock/internal/matcher"
	"vimock/internal/recording"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

const (
	grpcContentType        = "application/grpc"
	grpcFrameHeaderSize    = 5
	maxGRPCRequestBodySize = 64 << 20

	grpcStatusOK              = 0
	grpcStatusCanceled        = 1
	grpcStatusUnknown         = 2
	grpcStatusInvalidArgument = 3
	grpcStatusNotFound        = 5
	grpcStatusPermission      = 7
	grpcStatusUnimplemented   = 12
	grpcStatusInternal        = 13
	grpcStatusUnavailable     = 14
	grpcStatusUnauthenticated = 16

	grpcStatusNameHeader   = "grpc-status-name"
	grpcStatusReasonHeader = "grpc-status-reason"
)

type grpcMethod struct {
	Service protoreflect.ServiceDescriptor
	Method  protoreflect.MethodDescriptor
}

func isGRPCRequest(r *http.Request) bool {
	return r.Method == http.MethodPost && strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), grpcContentType)
}

func (a runtimeAPI) serveGRPC(w http.ResponseWriter, r *http.Request) {
	if a.descriptors == nil {
		writeGRPCError(w, grpcStatusUnimplemented, "gRPC descriptor registry is not configured")
		return
	}

	method, registry, ok := a.findGRPCMethod(r.URL.Path)
	if !ok {
		writeGRPCError(w, grpcStatusUnimplemented, "No matching gRPC service method found")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxGRPCRequestBodySize)
	defer r.Body.Close()

	framedBody, err := io.ReadAll(r.Body)
	if err != nil {
		writeGRPCError(w, grpcStatusInternal, fmt.Sprintf("read gRPC request: %v", err))
		return
	}
	requestPayload, err := decodeUnaryGRPCFrame(framedBody)
	if err != nil {
		writeGRPCError(w, grpcStatusInternal, err.Error())
		return
	}

	requestJSON, err := decodeProtoJSON(requestPayload, method.Method.Input(), registry)
	if err != nil {
		writeGRPCError(w, grpcStatusInternal, err.Error())
		return
	}

	bodyContext := matcher.NewBodyContext(requestJSON)
	matchRequest := r.Clone(r.Context())
	matchRequest.Header = grpcMatcherHeaders(r.Header)
	stub, matched := a.findMatch(matchRequest, bodyContext)
	if !matched {
		writeGRPCError(w, grpcStatusUnimplemented, "No matching stub mapping found for gRPC request")
		return
	}

	definition := stub.Response()
	if err := a.sleep(r.Context(), delay.InitialDuration(definition, nil)); err != nil {
		return
	}
	if definition.ProxyBaseURL != "" {
		writeGRPCError(w, grpcStatusUnimplemented, "gRPC proxy is not implemented")
		return
	}

	rendered, err := a.renderer.Render(definition, bodyContext)
	if err != nil {
		writeGRPCError(w, grpcStatusInternal, err.Error())
		return
	}

	if code, reason, ok := grpcStatusFromHeaders(rendered.Headers); ok {
		if code != grpcStatusOK {
			a.recordGRPCServeEvent(r, requestJSON, rendered.Status, rendered.Headers, nil)
			writeGRPCErrorWithHeaders(w, rendered.Headers, code, reason)
			return
		}
	} else if code, reason, ok := grpcStatusFromHTTP(rendered.Status); ok {
		a.recordGRPCServeEvent(r, requestJSON, rendered.Status, rendered.Headers, nil)
		writeGRPCErrorWithHeaders(w, rendered.Headers, code, reason)
		return
	}

	responsePayload, err := encodeProtoJSON(rendered.Body, method.Method.Output(), registry)
	if err != nil {
		writeGRPCErrorWithHeaders(w, rendered.Headers, grpcStatusInternal, err.Error())
		return
	}

	a.recordGRPCServeEvent(r, requestJSON, rendered.Status, rendered.Headers, rendered.Body)
	writeGRPCMessage(w, rendered.Headers, responsePayload)
}

func (a runtimeAPI) recordGRPCServeEvent(r *http.Request, requestJSON []byte, responseStatus int, responseHeaders http.Header, responseJSON []byte) {
	if a.recorder == nil {
		return
	}
	a.recorder.AddServeEvent(recording.ServeEvent{
		Method:          http.MethodPost,
		URL:             r.URL.Path,
		Path:            r.URL.Path,
		RequestHeaders:  r.Header,
		RequestBody:     requestJSON,
		ResponseStatus:  responseStatus,
		ResponseHeaders: responseHeaders,
		ResponseBody:    responseJSON,
		Source:          recording.SourceStub,
		Protocol:        recording.ProtocolGRPC,
	})
}

func (a runtimeAPI) findGRPCMethod(path string) (grpcMethod, grpcdesc.Registry, bool) {
	serviceName, methodName, ok := splitGRPCPath(path)
	if !ok {
		return grpcMethod{}, grpcdesc.Registry{}, false
	}

	registry := a.descriptors.Active()
	service, ok := registry.FindService(serviceName)
	if !ok {
		return grpcMethod{}, registry, false
	}
	method := service.Methods().ByName(protoreflect.Name(methodName))
	if method == nil {
		return grpcMethod{}, registry, false
	}
	return grpcMethod{Service: service, Method: method}, registry, true
}

func splitGRPCPath(path string) (string, string, bool) {
	path = strings.TrimPrefix(path, "/")
	serviceName, methodName, ok := strings.Cut(path, "/")
	if !ok || serviceName == "" || methodName == "" || strings.Contains(methodName, "/") {
		return "", "", false
	}
	return serviceName, methodName, true
}

func decodeUnaryGRPCFrame(body []byte) ([]byte, error) {
	if len(body) < grpcFrameHeaderSize {
		return nil, fmt.Errorf("gRPC request frame is too short")
	}
	if body[0] != 0 {
		return nil, fmt.Errorf("compressed gRPC messages are not supported")
	}

	size := int(binary.BigEndian.Uint32(body[1:grpcFrameHeaderSize]))
	end := grpcFrameHeaderSize + size
	if size < 0 || end > len(body) {
		return nil, fmt.Errorf("gRPC request frame length exceeds body size")
	}
	if end != len(body) {
		return nil, fmt.Errorf("only unary gRPC requests are supported")
	}
	return body[grpcFrameHeaderSize:end], nil
}

func encodeGRPCFrame(payload []byte) []byte {
	frame := make([]byte, grpcFrameHeaderSize+len(payload))
	binary.BigEndian.PutUint32(frame[1:grpcFrameHeaderSize], uint32(len(payload)))
	copy(frame[grpcFrameHeaderSize:], payload)
	return frame
}

func decodeProtoJSON(payload []byte, descriptor protoreflect.MessageDescriptor, registry grpcdesc.Registry) ([]byte, error) {
	message := dynamicpb.NewMessage(descriptor)
	if err := proto.Unmarshal(payload, message); err != nil {
		return nil, fmt.Errorf("decode gRPC request protobuf: %w", err)
	}

	body, err := protojson.MarshalOptions{
		UseProtoNames:   true,
		EmitUnpopulated: false,
		Resolver:        registry.TypeResolver(),
	}.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("encode gRPC request JSON: %w", err)
	}
	return body, nil
}

func encodeProtoJSON(body []byte, descriptor protoreflect.MessageDescriptor, registry grpcdesc.Registry) ([]byte, error) {
	message := dynamicpb.NewMessage(descriptor)
	if len(bytes.TrimSpace(body)) > 0 {
		if err := (protojson.UnmarshalOptions{
			Resolver: registry.TypeResolver(),
		}).Unmarshal(body, message); err != nil {
			return nil, fmt.Errorf("encode gRPC response protobuf: %w", err)
		}
	}

	payload, err := proto.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("marshal gRPC response protobuf: %w", err)
	}
	return payload, nil
}

func grpcStatusFromHeaders(headers http.Header) (int, string, bool) {
	statusName := strings.TrimSpace(headerValue(headers, grpcStatusNameHeader))
	if statusName == "" {
		return 0, "", false
	}

	code, ok := grpcStatusCodeByName(statusName)
	if !ok {
		return grpcStatusInternal, "unsupported grpc-status-name: " + statusName, true
	}
	return code, headerValue(headers, grpcStatusReasonHeader), true
}

func grpcStatusCodeByName(name string) (int, bool) {
	normalized := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(name), "-", "_"))
	switch normalized {
	case "OK":
		return grpcStatusOK, true
	case "CANCELED", "CANCELLED":
		return grpcStatusCanceled, true
	case "UNKNOWN":
		return grpcStatusUnknown, true
	case "INVALID_ARGUMENT":
		return grpcStatusInvalidArgument, true
	case "NOT_FOUND":
		return grpcStatusNotFound, true
	case "PERMISSION_DENIED":
		return grpcStatusPermission, true
	case "UNIMPLEMENTED":
		return grpcStatusUnimplemented, true
	case "INTERNAL":
		return grpcStatusInternal, true
	case "UNAVAILABLE":
		return grpcStatusUnavailable, true
	case "UNAUTHENTICATED":
		return grpcStatusUnauthenticated, true
	default:
		return 0, false
	}
}

func grpcStatusFromHTTP(status int) (int, string, bool) {
	switch status {
	case http.StatusOK:
		return grpcStatusOK, "", false
	case http.StatusBadRequest:
		return grpcStatusInternal, http.StatusText(status), true
	case http.StatusUnauthorized:
		return grpcStatusUnauthenticated, http.StatusText(status), true
	case http.StatusForbidden:
		return grpcStatusPermission, http.StatusText(status), true
	case http.StatusNotFound:
		return grpcStatusUnimplemented, http.StatusText(status), true
	case http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return grpcStatusUnavailable, http.StatusText(status), true
	default:
		if status >= 400 {
			return grpcStatusUnknown, http.StatusText(status), true
		}
		return grpcStatusOK, "", false
	}
}

func writeGRPCMessage(w http.ResponseWriter, headers http.Header, payload []byte) {
	prepareGRPCResponse(w, headers)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(encodeGRPCFrame(payload))
	setGRPCTrailers(w, grpcStatusOK, "")
}

func writeGRPCError(w http.ResponseWriter, code int, message string) {
	writeGRPCErrorWithHeaders(w, nil, code, message)
}

func writeGRPCErrorWithHeaders(w http.ResponseWriter, headers http.Header, code int, message string) {
	prepareGRPCResponse(w, headers)
	w.WriteHeader(http.StatusOK)
	setGRPCTrailers(w, code, message)
}

func prepareGRPCResponse(w http.ResponseWriter, headers http.Header) {
	target := w.Header()
	for name, values := range headers {
		if isInternalGRPCMappingHeader(name) {
			continue
		}
		for _, value := range values {
			target.Add(name, value)
		}
	}

	target.Set("Content-Type", grpcContentType)
	target.Set("Trailer", "Grpc-Status, Grpc-Message")
}

func setGRPCTrailers(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Grpc-Status", strconv.Itoa(code))
	if message != "" {
		w.Header().Set("Grpc-Message", encodeGRPCMessage(message))
	}
}

func isInternalGRPCMappingHeader(name string) bool {
	switch strings.ToLower(name) {
	case grpcStatusNameHeader, grpcStatusReasonHeader, "content-length":
		return true
	default:
		return false
	}
}

func headerValue(headers http.Header, name string) string {
	if len(headers) == 0 {
		return ""
	}
	if value := headers.Get(name); value != "" {
		return value
	}
	for candidate, values := range headers {
		if strings.EqualFold(candidate, name) && len(values) > 0 {
			return values[0]
		}
	}
	return ""
}

func grpcMatcherHeaders(headers http.Header) http.Header {
	result := make(http.Header, len(headers))
	for name, values := range headers {
		copied := make([]string, 0, len(values))
		for _, value := range values {
			if strings.HasSuffix(strings.ToLower(name), "-bin") {
				if decoded, ok := decodeGRPCBinaryHeader(value); ok {
					value = formatByteArray(decoded)
				}
			}
			copied = append(copied, value)
		}
		result[name] = copied
	}
	return result
}

func decodeGRPCBinaryHeader(value string) ([]byte, bool) {
	if decoded, err := base64.StdEncoding.DecodeString(value); err == nil {
		return decoded, true
	}
	if decoded, err := base64.RawStdEncoding.DecodeString(value); err == nil {
		return decoded, true
	}
	return nil, false
}

func formatByteArray(data []byte) string {
	if len(data) == 0 {
		return "[]"
	}
	parts := make([]string, 0, len(data))
	for _, value := range data {
		signed := int(value)
		if signed > 127 {
			signed -= 256
		}
		parts = append(parts, strconv.Itoa(signed))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func encodeGRPCMessage(message string) string {
	return strings.ReplaceAll(url.QueryEscape(message), "+", "%20")
}
