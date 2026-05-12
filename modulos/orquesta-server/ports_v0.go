package orquestaserver

import (
	"context"
	"time"

	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

type SupervisorPortV0 interface {
	RunGlobalSupervisorV0(
		context.Context,
		orquestarunsupervisor.RunSupervisorCommandV0,
	) (orquestarunsupervisor.RunSupervisorResultV0, error)
}

type StateStorePortV0 interface {
	SaveServerStateV0(context.Context, StateV0) error
	LoadServerStateV0(context.Context) (StateV0, error)
}

type ClockPortV0 interface {
	Now() time.Time
}

type SystemClockV0 struct{}

func (SystemClockV0) Now() time.Time {
	return time.Now().UTC()
}
