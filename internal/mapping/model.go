package mapping

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// Mapping is a WireMock stub mapping with parsed top-level fields and preserved raw JSON.
type Mapping struct {
	id         string
	name       string
	persistent bool
	raw        map[string]json.RawMessage
	sequence   uint64
}

func ParseJSON(data []byte) (Mapping, error) {
	return ParseJSONWithID(data, "")
}

func ParseJSONWithID(data []byte, overrideID string) (Mapping, error) {
	raw, err := decodeObject(data)
	if err != nil {
		return Mapping{}, err
	}

	id, err := stringField(raw, "id")
	if err != nil {
		return Mapping{}, err
	}
	if id != "" {
		if err := ValidateID(id); err != nil {
			return Mapping{}, fmt.Errorf("id: %w", err)
		}
	}

	if overrideID != "" {
		if err := ValidateID(overrideID); err != nil {
			return Mapping{}, err
		}
		id = overrideID
	}
	if id == "" {
		id, err = NewID()
		if err != nil {
			return Mapping{}, fmt.Errorf("generate id: %w", err)
		}
	}

	name, err := stringField(raw, "name")
	if err != nil {
		return Mapping{}, err
	}
	persistent, err := boolField(raw, "persistent")
	if err != nil {
		return Mapping{}, err
	}
	if err := validateObjectField(raw, "request", true); err != nil {
		return Mapping{}, err
	}
	if err := validateObjectField(raw, "response", true); err != nil {
		return Mapping{}, err
	}
	if err := validateObjectField(raw, "metadata", false); err != nil {
		return Mapping{}, err
	}

	mapping := Mapping{
		id:         id,
		name:       name,
		persistent: persistent,
		raw:        cloneRawMap(raw),
	}
	mapping.raw["id"] = mustMarshalRaw(id)

	return mapping, nil
}

func (m Mapping) ID() string {
	return m.id
}

func (m Mapping) Name() string {
	return m.name
}

func (m Mapping) Persistent() bool {
	return m.persistent
}

func (m Mapping) Sequence() uint64 {
	return m.sequence
}

func (m Mapping) WithID(id string) (Mapping, error) {
	if err := ValidateID(id); err != nil {
		return Mapping{}, err
	}

	next := m
	next.id = id
	next.raw = cloneRawMap(m.raw)
	next.raw["id"] = mustMarshalRaw(id)
	return next, nil
}

func (m Mapping) MarshalJSON() ([]byte, error) {
	raw := cloneRawMap(m.raw)
	raw["id"] = mustMarshalRaw(m.id)
	return json.Marshal(raw)
}

func decodeObject(data []byte) (map[string]json.RawMessage, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()

	var raw map[string]json.RawMessage
	if err := decoder.Decode(&raw); err != nil {
		return nil, fmt.Errorf("mapping body must be a valid JSON object: %w", err)
	}
	if raw == nil {
		return nil, fmt.Errorf("mapping body must be a JSON object")
	}
	if decoder.More() {
		return nil, fmt.Errorf("mapping body must contain a single JSON object")
	}

	var extra json.RawMessage
	if err := decoder.Decode(&extra); err == nil {
		return nil, fmt.Errorf("mapping body must contain a single JSON object")
	}

	return raw, nil
}

func stringField(raw map[string]json.RawMessage, field string) (string, error) {
	value, ok := raw[field]
	if !ok || len(value) == 0 || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
		return "", nil
	}

	var parsed string
	if err := json.Unmarshal(value, &parsed); err != nil {
		return "", fmt.Errorf("%s must be a string", field)
	}
	return parsed, nil
}

func boolField(raw map[string]json.RawMessage, field string) (bool, error) {
	value, ok := raw[field]
	if !ok || len(value) == 0 || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
		return false, nil
	}

	var parsed bool
	if err := json.Unmarshal(value, &parsed); err != nil {
		return false, fmt.Errorf("%s must be a boolean", field)
	}
	return parsed, nil
}

func validateObjectField(raw map[string]json.RawMessage, field string, required bool) error {
	value, ok := raw[field]
	if !ok {
		if required {
			return fmt.Errorf("%s is required", field)
		}
		return nil
	}

	var object map[string]json.RawMessage
	if err := json.Unmarshal(value, &object); err != nil || object == nil {
		return fmt.Errorf("%s must be a JSON object", field)
	}
	return nil
}

func cloneRawMap(source map[string]json.RawMessage) map[string]json.RawMessage {
	clone := make(map[string]json.RawMessage, len(source))
	for key, value := range source {
		clone[key] = cloneRaw(value)
	}
	return clone
}

func cloneRaw(source json.RawMessage) json.RawMessage {
	if source == nil {
		return nil
	}
	clone := make([]byte, len(source))
	copy(clone, source)
	return clone
}

func mustMarshalRaw(value any) json.RawMessage {
	raw, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return raw
}

func NewID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}

	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	var out [36]byte
	hex.Encode(out[0:8], b[0:4])
	out[8] = '-'
	hex.Encode(out[9:13], b[4:6])
	out[13] = '-'
	hex.Encode(out[14:18], b[6:8])
	out[18] = '-'
	hex.Encode(out[19:23], b[8:10])
	out[23] = '-'
	hex.Encode(out[24:36], b[10:16])

	return string(out[:]), nil
}

func ValidateID(id string) error {
	if !IsValidID(id) {
		return fmt.Errorf("%s is not a valid UUID", id)
	}
	return nil
}

func IsValidID(id string) bool {
	if len(id) != 36 {
		return false
	}
	for i := 0; i < len(id); i++ {
		switch i {
		case 8, 13, 18, 23:
			if id[i] != '-' {
				return false
			}
		default:
			if !isHex(id[i]) {
				return false
			}
		}
	}
	return true
}

func isHex(b byte) bool {
	return (b >= '0' && b <= '9') ||
		(b >= 'a' && b <= 'f') ||
		(b >= 'A' && b <= 'F')
}
