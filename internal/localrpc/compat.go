package localrpc

import (
	"context"
	"time"
)

const (
	defaultTimeout   = 5 * time.Second
	defaultStartWait = 8 * time.Second
)

type ServerInfo = State
type ExecuteRequest = ExecRequest
type ExecuteResponse = ExecResponse

func SaveServerInfo(info ServerInfo) error {
	return SaveState("", &info)
}

func LoadServerInfo() (*ServerInfo, error) {
	return LoadState("")
}

func Ping(ctx context.Context, addr string) error {
	_, err := NewClient(addr, nil).Ping(ctx)
	return err
}

func Execute(ctx context.Context, addr string, payload ExecuteRequest) (*ExecuteResponse, error) {
	return NewClient(addr, nil).Exec(ctx, &payload)
}

func DefaultTimeout() time.Duration {
	return defaultTimeout
}

func DefaultStartWait() time.Duration {
	return defaultStartWait
}
