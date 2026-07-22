//go:build !linux

package bubblewrap

import (
	"context"

	"orquesta/internal/ports"
)

type Adapter struct{}

func New(Config) (*Adapter, error) { return nil, &Error{Code: CodeUnavailable} }

func (*Adapter) PolicyIdentity() PolicyIdentity { return PolicyIdentity{} }

func (*Adapter) Attest(context.Context, ports.TestAttestationRun) (ports.TestAttestationResult, error) {
	return ports.TestAttestationResult{}, &Error{Code: CodeUnavailable}
}

func (*Adapter) Close() error { return nil }
