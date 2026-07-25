//go:build !linux

package codex

import (
	"context"
	"errors"
)

func platformSupervisorCommand(context.Context, supervisorEnvelope, *codexCgroupLeaf) (supervisorCommand, error) {
	return supervisorCommand{}, errors.New(CodeControlUnsupported)
}

func platformRunLocalSupervisor() error {
	return errors.New(CodeControlUnsupported)
}
