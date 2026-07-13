// Package orquestaconfig provides pure primitives for configuration documents.
package orquestaconfig

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
)

const DefaultMaxDocumentBytesV0 = 1 << 20

var (
	ErrDocumentTooLargeV0 = errors.New("orquesta-config: document exceeds byte limit")
	ErrInvalidTargetV0    = errors.New("orquesta-config: decode target must be a non-nil pointer")
)

type CanonicalDocumentV0 struct {
	Bytes    []byte
	Revision string
}

func EncodeCanonicalJSONV0(value any) (CanonicalDocumentV0, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return CanonicalDocumentV0{}, fmt.Errorf("orquesta-config: canonical encode: %w", err)
	}
	return CanonicalDocumentFromBytesV0(data), nil
}

func CanonicalDocumentFromBytesV0(data []byte) CanonicalDocumentV0 {
	canonical := append([]byte(nil), data...)
	return CanonicalDocumentV0{Bytes: canonical, Revision: RevisionSHA256V0(canonical)}
}

func RevisionSHA256V0(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// DecodeStrictBoundedJSONV0 is pure and rejects unknown fields and trailing values.
func DecodeStrictBoundedJSONV0(data []byte, target any, maxBytes int) error {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxDocumentBytesV0
	}
	if len(data) > maxBytes {
		return ErrDocumentTooLargeV0
	}
	targetValue := reflect.ValueOf(target)
	if !targetValue.IsValid() || targetValue.Kind() != reflect.Pointer || targetValue.IsNil() {
		return ErrInvalidTargetV0
	}
	// Decode into an isolated candidate. Callers can safely retain target when
	// syntax, unknown-field, trailing-value, or semantic validation fails.
	candidate := reflect.New(targetValue.Elem().Type())
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(candidate.Interface()); err != nil {
		return fmt.Errorf("orquesta-config: strict decode: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("orquesta-config: multiple JSON values")
		}
		return fmt.Errorf("orquesta-config: trailing JSON: %w", err)
	}
	targetValue.Elem().Set(candidate.Elem())
	return nil
}
