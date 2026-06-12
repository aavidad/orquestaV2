package main

import (
	"context"

	orquestaserver "orquesta/modulos/orquesta-server"
)

type externalBridgeLifecycleRuntimeV0 interface {
	MarkExternalBridgeLifecycleV0(context.Context, orquestaserver.ExternalBridgeLifecycleUpdateV0)
}

func externalBridgeRuntimeObserverV0(
	runtime externalBridgeLifecycleRuntimeV0,
) externalBridgeLoopObserverV0 {
	if runtime == nil {
		return nil
	}
	return func(ctx context.Context, event externalBridgeLoopEventV0) {
		runtime.MarkExternalBridgeLifecycleV0(ctx, orquestaserver.ExternalBridgeLifecycleUpdateV0{
			Component:    event.Component,
			Status:       event.Status,
			TickNumber:   event.TickNumber,
			TickActive:   event.TickActive,
			Success:      event.Success,
			ErrorCode:    event.ErrorCode,
			StopReason:   event.StopReason,
			Filters:      event.FilterSummary,
			Counters:     event.Counters,
			EvidenceRefs: event.EvidenceRefs,
			OccurredAt:   event.OccurredAt,
		})
	}
}

func markExternalBridgeConfigBlockedV0(
	ctx context.Context,
	runtime externalBridgeLifecycleRuntimeV0,
	err error,
) {
	if runtime == nil || err == nil {
		return
	}
	runtime.MarkExternalBridgeLifecycleV0(ctx, orquestaserver.ExternalBridgeLifecycleUpdateV0{
		Component: "opes_bridge_loop",
		Status:    "blocked",
		ErrorCode: compactExternalBridgeErrorCodeV0(err),
	})
}
