//go:build !linux

package codex

import (
	"context"
	"errors"
)

func platformSupervisorCommand(context.Context, supervisorEnvelope) (supervisorCommand, error) {
	return supervisorCommand{}, errors.New(CodeControlUnsupported)
}

func platformRunLocalSupervisor() error {
	return errors.New(CodeControlUnsupported)
}
