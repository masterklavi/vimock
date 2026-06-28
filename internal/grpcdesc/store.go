package grpcdesc

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

const (
	KindDescriptorSet = "descriptor-set"
	KindProtoSource   = "proto-source"
)

type Store struct {
	mu         sync.RWMutex
	blobs      map[string]blob
	active     Registry
	generation uint64
}

type blob struct {
	Name      string
	Kind      string
	Data      []byte
	UpdatedAt time.Time
}

type FileInfo struct {
	Name      string    `json:"name"`
	Kind      string    `json:"kind"`
	Size      int       `json:"size"`
	Loadable  bool      `json:"loadable"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Registry struct {
	Generation int      `json:"generation"`
	Files      int      `json:"files"`
	Services   []string `json:"services"`
	Messages   []string `json:"messages"`

	files *protoregistry.Files
	types *protoregistry.Types
}

type TypeResolver interface {
	protoregistry.MessageTypeResolver
	protoregistry.ExtensionTypeResolver
}

func NewStore() *Store {
	return &Store{
		blobs:  make(map[string]blob),
		active: Registry{Files: 0},
	}
}

func (s *Store) Put(name string, data []byte) (bool, error) {
	name, kind, err := ValidateFileName(name)
	if err != nil {
		return false, err
	}
	if err := validateBlob(kind, data); err != nil {
		return false, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	_, exists := s.blobs[name]
	s.blobs[name] = blob{
		Name:      name,
		Kind:      kind,
		Data:      cloneBytes(data),
		UpdatedAt: time.Now().UTC(),
	}
	return exists, nil
}

func (s *Store) PutLegacy(name string, data []byte) bool {
	if s == nil {
		return false
	}
	if _, kind, err := ValidateFileName(name); err != nil || kind == KindProtoSource {
		return false
	}
	if err := validateBlob(KindDescriptorSet, data); err != nil {
		return false
	}
	_, err := s.Put(name, data)
	return err == nil
}

func (s *Store) Delete(name string) bool {
	name, _, err := ValidateFileName(name)
	if err != nil {
		return false
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.blobs[name]; !ok {
		return false
	}
	delete(s.blobs, name)
	return true
}

func (s *Store) List() ([]FileInfo, Registry) {
	if s == nil {
		return nil, Registry{}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	files := make([]FileInfo, 0, len(s.blobs))
	for _, stored := range s.blobs {
		files = append(files, FileInfo{
			Name:      stored.Name,
			Kind:      stored.Kind,
			Size:      len(stored.Data),
			Loadable:  stored.Kind == KindDescriptorSet,
			UpdatedAt: stored.UpdatedAt,
		})
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name < files[j].Name
	})

	return files, cloneRegistry(s.active)
}

func (s *Store) Reset() (Registry, error) {
	if s == nil {
		return Registry{}, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	combined := &descriptorpb.FileDescriptorSet{}
	for _, name := range sortedBlobNames(s.blobs) {
		stored := s.blobs[name]
		if stored.Kind != KindDescriptorSet {
			continue
		}
		parsed, err := parseDescriptorSet(stored.Data)
		if err != nil {
			return Registry{}, fmt.Errorf("%s: %w", name, err)
		}
		combined.File = append(combined.File, parsed.File...)
	}

	registry, err := buildRegistry(combined)
	if err != nil {
		return Registry{}, err
	}
	s.generation++
	registry.Generation = int(s.generation)
	s.active = registry
	return cloneRegistry(registry), nil
}

func (s *Store) Active() Registry {
	if s == nil {
		return Registry{}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return cloneRegistry(s.active)
}

func (r Registry) FindService(name string) (protoreflect.ServiceDescriptor, bool) {
	if r.files == nil {
		return nil, false
	}

	descriptor, err := r.files.FindDescriptorByName(protoreflect.FullName(name))
	if err != nil {
		return nil, false
	}
	service, ok := descriptor.(protoreflect.ServiceDescriptor)
	return service, ok
}

func (r Registry) FindMessageType(name string) (protoreflect.MessageType, bool) {
	if r.types == nil {
		return nil, false
	}

	messageType, err := r.types.FindMessageByName(protoreflect.FullName(name))
	if err != nil {
		return nil, false
	}
	return messageType, true
}

func (r Registry) TypeResolver() TypeResolver {
	if r.types == nil {
		return protoregistry.GlobalTypes
	}
	return r.types
}

func ValidateFileName(name string) (string, string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", "", fmt.Errorf("descriptor file name is required")
	}
	if name == "." || name == ".." || strings.ContainsAny(name, `/\`) || strings.ContainsRune(name, 0) {
		return "", "", fmt.Errorf("invalid descriptor file name %q", name)
	}

	extension := strings.ToLower(filepath.Ext(name))
	switch extension {
	case ".dsc", ".desc":
		return name, KindDescriptorSet, nil
	case ".proto":
		return name, KindProtoSource, nil
	default:
		return "", "", fmt.Errorf("descriptor file %q must have .dsc, .desc or .proto extension", name)
	}
}

func validateBlob(kind string, data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("descriptor file body must not be empty")
	}

	switch kind {
	case KindDescriptorSet:
		_, err := parseDescriptorSet(data)
		return err
	case KindProtoSource:
		if !utf8.Valid(data) {
			return fmt.Errorf("proto source must be valid UTF-8")
		}
		return nil
	default:
		return fmt.Errorf("unsupported descriptor kind %q", kind)
	}
}

func parseDescriptorSet(data []byte) (*descriptorpb.FileDescriptorSet, error) {
	var descriptorSet descriptorpb.FileDescriptorSet
	if err := proto.Unmarshal(data, &descriptorSet); err != nil {
		return nil, fmt.Errorf("parse FileDescriptorSet: %w", err)
	}
	if len(descriptorSet.File) == 0 {
		return nil, fmt.Errorf("FileDescriptorSet must contain at least one file")
	}
	return &descriptorSet, nil
}

func buildRegistry(descriptorSet *descriptorpb.FileDescriptorSet) (Registry, error) {
	if descriptorSet == nil || len(descriptorSet.File) == 0 {
		return Registry{
			files: &protoregistry.Files{},
			types: &protoregistry.Types{},
		}, nil
	}

	files, err := protodesc.NewFiles(descriptorSet)
	if err != nil {
		return Registry{}, fmt.Errorf("build descriptor registry: %w", err)
	}

	types := &protoregistry.Types{}
	var services []string
	var messages []string
	var fileCount int
	var registerErr error
	files.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		fileCount++
		for i := 0; i < file.Services().Len(); i++ {
			services = append(services, string(file.Services().Get(i).FullName()))
		}
		registerMessages(types, file.Messages(), &messages, &registerErr)
		return registerErr == nil
	})
	if registerErr != nil {
		return Registry{}, registerErr
	}

	sort.Strings(services)
	sort.Strings(messages)
	return Registry{
		Files:    fileCount,
		Services: services,
		Messages: messages,
		files:    files,
		types:    types,
	}, nil
}

func registerMessages(types *protoregistry.Types, descriptors protoreflect.MessageDescriptors, messages *[]string, registerErr *error) {
	for i := 0; i < descriptors.Len(); i++ {
		descriptor := descriptors.Get(i)
		*messages = append(*messages, string(descriptor.FullName()))
		if err := types.RegisterMessage(dynamicpb.NewMessageType(descriptor)); err != nil {
			*registerErr = fmt.Errorf("register message %s: %w", descriptor.FullName(), err)
			return
		}
		registerMessages(types, descriptor.Messages(), messages, registerErr)
		if *registerErr != nil {
			return
		}
	}
}

func sortedBlobNames(blobs map[string]blob) []string {
	names := make([]string, 0, len(blobs))
	for name := range blobs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func cloneRegistry(source Registry) Registry {
	return Registry{
		Generation: source.Generation,
		Files:      source.Files,
		Services:   append([]string(nil), source.Services...),
		Messages:   append([]string(nil), source.Messages...),
		files:      source.files,
		types:      source.types,
	}
}

func cloneBytes(source []byte) []byte {
	if source == nil {
		return nil
	}
	clone := make([]byte, len(source))
	copy(clone, source)
	return clone
}
