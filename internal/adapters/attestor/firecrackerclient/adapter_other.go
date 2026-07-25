//go:build !linux

package firecrackerclient

import (
	"context"

	"orquesta/internal/ports"
)

type Adapter struct{}

func New(Config) (*Adapter, error) {
	return nil, contractError(codeUnavailable)
}

func (*Adapter) PolicyIdentity() PolicyIdentity { return PolicyIdentity{} }

func (*Adapter) Attest(
	context.Context,
	ports.TestAttestationRun,
) (ports.TestAttestationResult, error) {
	return ports.TestAttestationResult{}, contractError(codeUnavailable)
}

func (*Adapter) Close() error { return nil }
