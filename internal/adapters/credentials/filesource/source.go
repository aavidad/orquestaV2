// Package filesource reads one callback-scoped JSON credential from a pinned,
// private file. It is an ingress adapter: it never persists or returns material.
package filesource

import (
	"context"
	"path/filepath"
	"reflect"
	"strings"

	"orquesta/internal/credentials"
)

// MaximumMaterialBytes is the non-configurable allocation ceiling. A caller's
// explicit maximum may be lower, but never raise this process safety limit.
const MaximumMaterialBytes uint64 = 16 << 20

// Source is a secure, immutable file-backed credential material source.
type Source struct {
	path     string
	ownerUID uint32
	maxBytes uint64
	hooks    sourceHooks
}

type sourceHooks struct {
	afterFirstMetadata func()
	beforeReopen       func()
}

// New constructs a source for one absolute canonical path and exact owner.
// The file itself is not opened until WithSecret so rotation by atomic rename
// remains possible between uses.
func New(path string, ownerUID uint32, maxBytes uint64) (*Source, error) {
	if path == "" || strings.TrimSpace(path) != path || strings.IndexByte(path, 0) >= 0 ||
		!filepath.IsAbs(path) || filepath.Clean(path) != path || filepath.Base(path) == string(filepath.Separator) ||
		maxBytes == 0 || maxBytes > MaximumMaterialBytes {
		return nil, credentials.NewError(credentials.ErrorInvalidRequest, "material_source")
	}
	return &Source{path: path, ownerUID: ownerUID, maxBytes: maxBytes}, nil
}

func (*Source) String() string   { return "[REDACTED]" }
func (*Source) GoString() string { return "filesource.Source{[REDACTED]}" }

func nilContext(ctx context.Context) bool {
	if ctx == nil {
		return true
	}
	value := reflect.ValueOf(ctx)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

var _ credentials.MaterialSource = (*Source)(nil)
