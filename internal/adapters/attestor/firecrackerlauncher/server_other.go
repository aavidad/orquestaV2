//go:build !linux

package firecrackerlauncher

import "context"

type Server struct{}

func NewServer(config Config) (*Server, error) {
	if err := validateConfig(config); err != nil {
		return nil, err
	}
	return nil, launcherError(CodeUnavailable)
}

func (*Server) Serve(context.Context) error {
	return launcherError(CodeUnavailable)
}

func (*Server) Close() error {
	return nil
}
