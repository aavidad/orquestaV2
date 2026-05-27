package orquestaoperatormcpclient

import (
	"context"
	"errors"
	"time"

	operator "orquesta/modulos/orquesta-operator-mcp"
)

const DefaultOperatorMCPClientTimeoutV0 = 30 * time.Second

type OperatorMCPClientContextFactoryV0 func() context.Context

func (connector OperatorMCPClientConnectorV0) callContextV0() (context.Context, context.CancelFunc) {
	parent := context.Background()
	if connector.contextFactory != nil {
		if configured := connector.contextFactory(); configured != nil {
			parent = configured
		}
	}
	return context.WithTimeout(parent, connector.timeout)
}

func normalizeTimeoutV0(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return DefaultOperatorMCPClientTimeoutV0
	}
	return timeout
}

func connectorErrorCodeV0(ctx context.Context, err error) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded), errors.Is(ctx.Err(), context.DeadlineExceeded):
		return operator.ErrOperatorMCPTimeoutV0
	case errors.Is(err, context.Canceled), errors.Is(ctx.Err(), context.Canceled):
		return operator.ErrOperatorMCPCancelledV0
	}
	if code, ok := operator.PublicOperatorMCPErrorCodeV0(err); ok {
		if normalized := normalizePublicErrorCodeV0(code); normalized != "" {
			return normalized
		}
	}
	return operator.ErrOperatorMCPPortErrorV0
}
