// Package tooling owns the transport-neutral catalog of tools. It describes
// what may be invoked; application remains responsible for authorization,
// budgets, effects, receipts, and Goal lifecycle.
package tooling

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"strings"

	"orquesta/internal/governance"
	"orquesta/internal/identity"
)

const (
	ErrorSpecInvalid    = "tooling.spec_invalid"
	ErrorSpecDuplicate  = "tooling.spec_duplicate"
	ErrorSpecNotFound   = "tooling.spec_not_found"
	ErrorPayloadInvalid = "tooling.payload_invalid"
)

type CostMode string
type IdempotencyPolicy string
type ReceiptPolicy string

const (
	CostMaximum CostMode = "maximum"

	IdempotencyRequired      IdempotencyPolicy = "required"
	IdempotencyReadReexecute IdempotencyPolicy = "read_reexecute"

	ReceiptApplication ReceiptPolicy = "application"
	ReceiptObservation ReceiptPolicy = "observation"
)

type CostContract struct {
	Mode    CostMode                  `json:"mode"`
	Maximum governance.ResourceVector `json:"maximum"`
}

type OutputDelivery struct {
	MaxBytes    int64 `json:"max_bytes"`
	InlineBytes int64 `json:"inline_bytes"`
}

type CapabilitySpec struct {
	ID           string                `json:"id"`
	Version      string                `json:"version"`
	InputSchema  json.RawMessage       `json:"input_schema"`
	OutputSchema json.RawMessage       `json:"output_schema"`
	Permissions  []identity.Permission `json:"permissions"`
	Cost         CostContract          `json:"cost"`
	Output       OutputDelivery        `json:"output"`
	Idempotency  IdempotencyPolicy     `json:"idempotency"`
	Receipt      ReceiptPolicy         `json:"receipt"`
}

type Registration struct {
	Spec   CapabilitySpec `json:"spec"`
	Digest string         `json:"digest"`
}

type Registry struct {
	entries []Registration
	byKey   map[string]int
	digest  string
}

func NewRegistry(specs ...CapabilitySpec) (*Registry, error) {
	entries := make([]Registration, 0, len(specs))
	seen := make(map[string]struct{}, len(specs))
	for _, source := range specs {
		spec, err := canonicalSpec(source)
		if err != nil {
			return nil, err
		}
		key := registryKey(spec.ID, spec.Version)
		if _, duplicate := seen[key]; duplicate {
			return nil, contractError(ErrorSpecDuplicate, "identity")
		}
		seen[key] = struct{}{}
		entries = append(entries, Registration{Spec: spec, Digest: specDigest(spec)})
	}
	sort.Slice(entries, func(left, right int) bool {
		if entries[left].Spec.ID != entries[right].Spec.ID {
			return entries[left].Spec.ID < entries[right].Spec.ID
		}
		leftRevision, _ := strconv.ParseUint(entries[left].Spec.Version, 10, 64)
		rightRevision, _ := strconv.ParseUint(entries[right].Spec.Version, 10, 64)
		return leftRevision < rightRevision
	})
	byKey := make(map[string]int, len(entries))
	for index, entry := range entries {
		byKey[registryKey(entry.Spec.ID, entry.Spec.Version)] = index
	}
	return &Registry{entries: entries, byKey: byKey, digest: registryDigest(entries)}, nil
}

func (registry *Registry) Lookup(id, version string) (Registration, bool) {
	if registry == nil {
		return Registration{}, false
	}
	index, ok := registry.byKey[registryKey(id, version)]
	if !ok {
		return Registration{}, false
	}
	return cloneRegistration(registry.entries[index]), true
}

func (registry *Registry) List() []Registration {
	if registry == nil {
		return nil
	}
	result := make([]Registration, len(registry.entries))
	for index, entry := range registry.entries {
		result[index] = cloneRegistration(entry)
	}
	return result
}

func (registry *Registry) Digest() string {
	if registry == nil {
		return ""
	}
	return registry.digest
}

func (registry *Registry) ValidateInput(id, version string, payload json.RawMessage) (json.RawMessage, error) {
	return registry.validatePayload(id, version, payload, true)
}

func (registry *Registry) ValidateOutput(id, version string, payload json.RawMessage) (json.RawMessage, error) {
	return registry.validatePayload(id, version, payload, false)
}

func (registry *Registry) validatePayload(id, version string, payload json.RawMessage, input bool) (json.RawMessage, error) {
	if registry == nil {
		return nil, contractError(ErrorSpecNotFound, "identity")
	}
	index, ok := registry.byKey[registryKey(id, version)]
	if !ok {
		return nil, contractError(ErrorSpecNotFound, "identity")
	}
	schema := registry.entries[index].Spec.OutputSchema
	if input {
		schema = registry.entries[index].Spec.InputSchema
	} else if int64(len(payload)) > registry.entries[index].Spec.Output.MaxBytes {
		return nil, contractError(ErrorPayloadInvalid, "payload")
	}
	canonical, err := validatePayloadAgainstSchema(schema, payload)
	if err != nil {
		return nil, contractError(ErrorPayloadInvalid, "payload")
	}
	return canonical, nil
}

type contractFailure struct {
	code  string
	field string
}

func (failure *contractFailure) Error() string { return failure.code + ":" + failure.field }

func ErrorCode(err error) string {
	var failure *contractFailure
	if errors.As(err, &failure) {
		return failure.code
	}
	return ""
}

func contractError(code, field string) error { return &contractFailure{code: code, field: field} }

func canonicalSpec(source CapabilitySpec) (CapabilitySpec, error) {
	if !validToolID(source.ID) {
		return CapabilitySpec{}, contractError(ErrorSpecInvalid, "id")
	}
	if _, err := parseVersion(source.Version); err != nil {
		return CapabilitySpec{}, contractError(ErrorSpecInvalid, "version")
	}
	input, err := canonicalObjectSchema(source.InputSchema)
	if err != nil {
		return CapabilitySpec{}, contractError(ErrorSpecInvalid, "input_schema")
	}
	output, err := canonicalObjectSchema(source.OutputSchema)
	if err != nil {
		return CapabilitySpec{}, contractError(ErrorSpecInvalid, "output_schema")
	}
	permissions := append([]identity.Permission(nil), source.Permissions...)
	if len(permissions) == 0 {
		return CapabilitySpec{}, contractError(ErrorSpecInvalid, "permissions")
	}
	sort.Slice(permissions, func(left, right int) bool { return permissions[left] < permissions[right] })
	for index, permission := range permissions {
		if identity.ValidatePermission(permission) != nil || index > 0 && permission == permissions[index-1] {
			return CapabilitySpec{}, contractError(ErrorSpecInvalid, "permissions")
		}
	}
	if source.Cost.Mode != CostMaximum || governance.ValidateResourceVector(source.Cost.Maximum) != nil {
		return CapabilitySpec{}, contractError(ErrorSpecInvalid, "cost")
	}
	if source.Output.MaxBytes <= 0 || source.Output.InlineBytes < 0 || source.Output.InlineBytes > source.Output.MaxBytes || source.Cost.Maximum.DiskBytes < source.Output.MaxBytes {
		return CapabilitySpec{}, contractError(ErrorSpecInvalid, "output")
	}
	validReplay := source.Idempotency == IdempotencyRequired && source.Receipt == ReceiptApplication
	validRead := source.Idempotency == IdempotencyReadReexecute && source.Receipt == ReceiptObservation
	if !validReplay && !validRead {
		return CapabilitySpec{}, contractError(ErrorSpecInvalid, "receipt_policy")
	}
	source.InputSchema = input
	source.OutputSchema = output
	source.Permissions = permissions
	return source, nil
}

func validToolID(id string) bool {
	if id == "" || len(id) > 200 || strings.TrimSpace(id) != id {
		return false
	}
	segments := strings.Split(id, ".")
	if len(segments) < 2 {
		return false
	}
	for _, segment := range segments {
		if segment == "" || len(segment) > 64 || segment[0] < 'a' || segment[0] > 'z' {
			return false
		}
		for _, character := range segment[1:] {
			if character < 'a' || character > 'z' {
				if character < '0' || character > '9' {
					if character != '_' && character != '-' {
						return false
					}
				}
			}
		}
	}
	return true
}

func parseVersion(version string) (uint64, error) {
	revision, err := strconv.ParseUint(version, 10, 64)
	if err != nil || revision == 0 || strconv.FormatUint(revision, 10) != version {
		return 0, errors.New("version_invalid")
	}
	return revision, nil
}

func registryKey(id, version string) string { return id + "\x00" + version }

func cloneRegistration(source Registration) Registration {
	source.Spec.InputSchema = append(json.RawMessage(nil), source.Spec.InputSchema...)
	source.Spec.OutputSchema = append(json.RawMessage(nil), source.Spec.OutputSchema...)
	source.Spec.Permissions = append([]identity.Permission(nil), source.Spec.Permissions...)
	return source
}

func specDigest(spec CapabilitySpec) string {
	encoded, err := json.Marshal(spec)
	if err != nil {
		panic("canonical tool spec cannot fail JSON encoding: " + err.Error())
	}
	digest := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func registryDigest(entries []Registration) string {
	digest := sha256.New()
	var size [8]byte
	for _, entry := range entries {
		binary.BigEndian.PutUint64(size[:], uint64(len(entry.Digest)))
		_, _ = digest.Write(size[:])
		_, _ = digest.Write([]byte(entry.Digest))
	}
	return "sha256:" + hex.EncodeToString(digest.Sum(nil))
}
