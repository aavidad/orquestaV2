//go:build !linux

package firecrackerlauncher

import (
	"context"
	"io"
	"os"
)

func NewSealedInput([]byte, int64) (*os.File, string, error) {
	return nil, "", launcherError(CodeUnavailable)
}

func NewSealedInputFromReader(io.Reader, int64, int64) (*os.File, string, error) {
	return nil, "", launcherError(CodeUnavailable)
}

func NewSealedInputFromFileContext(context.Context, *os.File, int64, int64) (*os.File, string, error) {
	return nil, "", launcherError(CodeUnavailable)
}

func NewOutputDescriptor(uint64) (*os.File, error) {
	return nil, launcherError(CodeUnavailable)
}
