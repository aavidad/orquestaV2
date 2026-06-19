package orquestamcp

import (
	"context"
	"net/http"
)

func runSupervisorExecutionContextV0(r *http.Request) context.Context {
	if r == nil {
		return context.Background()
	}
	return context.WithoutCancel(r.Context())
}
