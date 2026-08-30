// Package tools exposes the neutral, public catalog contract for connector
// manifests. It owns no execution, transport, authorization, or lifecycle.
package tools

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"

	"orquesta/internal/governance"
	"orquesta/internal/identity"
	"orquesta/internal/tooling"
)

var (
	ErrSpecEncoding      = errors.New("toolsdk.spec_encoding_invalid")
	ErrSpecInvalid       = errors.New("toolsdk.spec_invalid")
	ErrSpecDuplicate     = errors.New("toolsdk.spec_duplicate")
	ErrDiscoveryInvalid  = errors.New("toolsdk.discovery_invalid")
	ErrNamespaceTooLarge = errors.New("toolsdk.namespace_too_large")
)

type Permission string
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

// ResourceCost is the public mirror of Orquesta's integer resource
// vocabulary. Currency is empty unless MoneyMicros is positive.
type ResourceCost struct {
	Tokens       int64  `json:"tokens"`
	MoneyMicros  int64  `json:"money_micros"`
	Currency     string `json:"currency,omitempty"`
	ActiveTimeNS int64  `json:"active_time_ns"`
	ProcessSlots int64  `json:"process_slots"`
	DiskBytes    int64  `json:"disk_bytes"`
}

type CostContract struct {
	Mode    CostMode     `json:"mode"`
	Maximum ResourceCost `json:"maximum"`
}

type OutputDelivery struct {
	MaxBytes    int64 `json:"max_bytes"`
	InlineBytes int64 `json:"inline_bytes"`
}

// Spec is the connector-facing representation of one exact tool version. It
// intentionally exposes no handler, process, path, credential, or provider.
type Spec struct {
	ID           string            `json:"id"`
	Version      string            `json:"version"`
	InputSchema  json.RawMessage   `json:"input_schema"`
	OutputSchema json.RawMessage   `json:"output_schema"`
	Permissions  []Permission      `json:"permissions"`
	Cost         CostContract      `json:"cost"`
	Output       OutputDelivery    `json:"output"`
	Idempotency  IdempotencyPolicy `json:"idempotency"`
	Receipt      ReceiptPolicy     `json:"receipt"`
}

type Registration struct {
	Spec   Spec   `json:"spec"`
	Digest string `json:"digest"`
}

// Catalog is a read-only public projection of the canonical internal tool
// registry. All semantic validation is delegated to that registry.
type Catalog struct {
	registry *tooling.Registry
}

// NewCatalog validates all connector specs atomically. It never leaves a
// partial catalog after an invalid or duplicate entry.
func NewCatalog(specs ...Spec) (*Catalog, error) {
	internal := make([]tooling.CapabilitySpec, len(specs))
	for index, spec := range specs {
		internal[index] = toInternal(spec)
	}
	registry, err := tooling.NewRegistry(internal...)
	if err != nil {
		return nil, publicRegistryError(err)
	}
	return &Catalog{registry: registry}, nil
}

// DecodeSpec accepts exactly one strict JSON object, rejects unknown fields,
// and returns the same canonical form NewCatalog stores.
func DecodeSpec(encoded []byte) (Spec, error) {
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var source Spec
	if err := decoder.Decode(&source); err != nil {
		return Spec{}, ErrSpecEncoding
	}
	if err := rejectTrailingJSON(decoder); err != nil {
		return Spec{}, ErrSpecEncoding
	}
	catalog, err := NewCatalog(source)
	if err != nil {
		return Spec{}, err
	}
	registration, ok := catalog.Lookup(source.ID, source.Version)
	if !ok {
		return Spec{}, ErrSpecInvalid
	}
	return registration.Spec, nil
}

// Lookup resolves one exact version and never selects latest implicitly.
func (catalog *Catalog) Lookup(id, version string) (Registration, bool) {
	if catalog == nil || catalog.registry == nil {
		return Registration{}, false
	}
	registration, ok := catalog.registry.Lookup(id, version)
	if !ok {
		return Registration{}, false
	}
	return fromInternal(registration), true
}

// List returns detached registrations in canonical ID and numeric-version
// order.
func (catalog *Catalog) List() []Registration {
	if catalog == nil || catalog.registry == nil {
		return nil
	}
	internal := catalog.registry.List()
	result := make([]Registration, len(internal))
	for index, registration := range internal {
		result[index] = fromInternal(registration)
	}
	return result
}

func (catalog *Catalog) Digest() string {
	if catalog == nil || catalog.registry == nil {
		return ""
	}
	return catalog.registry.Digest()
}

// ErrorCode returns a stable public code for SDK contract failures.
func ErrorCode(err error) string {
	switch {
	case errors.Is(err, ErrSpecEncoding):
		return ErrSpecEncoding.Error()
	case errors.Is(err, ErrSpecInvalid):
		return ErrSpecInvalid.Error()
	case errors.Is(err, ErrSpecDuplicate):
		return ErrSpecDuplicate.Error()
	case errors.Is(err, ErrDiscoveryInvalid):
		return ErrDiscoveryInvalid.Error()
	case errors.Is(err, ErrNamespaceTooLarge):
		return ErrNamespaceTooLarge.Error()
	default:
		return ""
	}
}

func publicRegistryError(err error) error {
	switch tooling.ErrorCode(err) {
	case tooling.ErrorSpecInvalid:
		return ErrSpecInvalid
	case tooling.ErrorSpecDuplicate:
		return ErrSpecDuplicate
	default:
		return err
	}
}

func rejectTrailingJSON(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return ErrSpecEncoding
		}
		return err
	}
	return nil
}

func toInternal(source Spec) tooling.CapabilitySpec {
	permissions := make([]identity.Permission, len(source.Permissions))
	for index, permission := range source.Permissions {
		permissions[index] = identity.Permission(permission)
	}
	return tooling.CapabilitySpec{
		ID: source.ID, Version: source.Version,
		InputSchema:  append(json.RawMessage(nil), source.InputSchema...),
		OutputSchema: append(json.RawMessage(nil), source.OutputSchema...),
		Permissions:  permissions,
		Cost: tooling.CostContract{
			Mode: tooling.CostMode(source.Cost.Mode),
			Maximum: governance.ResourceVector{
				Tokens: source.Cost.Maximum.Tokens, MoneyMicros: source.Cost.Maximum.MoneyMicros,
				Currency: governance.Currency(source.Cost.Maximum.Currency), ActiveTimeNS: source.Cost.Maximum.ActiveTimeNS,
				ProcessSlots: source.Cost.Maximum.ProcessSlots, DiskBytes: source.Cost.Maximum.DiskBytes,
			},
		},
		Output: tooling.OutputDelivery{
			MaxBytes: source.Output.MaxBytes, InlineBytes: source.Output.InlineBytes,
		},
		Idempotency: tooling.IdempotencyPolicy(source.Idempotency),
		Receipt:     tooling.ReceiptPolicy(source.Receipt),
	}
}

func fromInternal(source tooling.Registration) Registration {
	permissions := make([]Permission, len(source.Spec.Permissions))
	for index, permission := range source.Spec.Permissions {
		permissions[index] = Permission(permission)
	}
	return Registration{
		Digest: source.Digest,
		Spec: Spec{
			ID: source.Spec.ID, Version: source.Spec.Version,
			InputSchema:  append(json.RawMessage(nil), source.Spec.InputSchema...),
			OutputSchema: append(json.RawMessage(nil), source.Spec.OutputSchema...),
			Permissions:  permissions,
			Cost: CostContract{
				Mode: CostMode(source.Spec.Cost.Mode),
				Maximum: ResourceCost{
					Tokens: source.Spec.Cost.Maximum.Tokens, MoneyMicros: source.Spec.Cost.Maximum.MoneyMicros,
					Currency: string(source.Spec.Cost.Maximum.Currency), ActiveTimeNS: source.Spec.Cost.Maximum.ActiveTimeNS,
					ProcessSlots: source.Spec.Cost.Maximum.ProcessSlots, DiskBytes: source.Spec.Cost.Maximum.DiskBytes,
				},
			},
			Output: OutputDelivery{
				MaxBytes: source.Spec.Output.MaxBytes, InlineBytes: source.Spec.Output.InlineBytes,
			},
			Idempotency: IdempotencyPolicy(source.Spec.Idempotency),
			Receipt:     ReceiptPolicy(source.Spec.Receipt),
		},
	}
}
