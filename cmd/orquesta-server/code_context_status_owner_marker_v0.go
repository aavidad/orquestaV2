package main

import (
	"context"
	"strings"
	"time"

	orquestacontext "orquesta/modulos/orquesta-context"
	orquestamcp "orquesta/modulos/orquesta-mcp"
)

type serverCodebaseStatusOwnerMarkerExecutorV0 struct {
	Inner    orquestamcp.MCPTransportCodebaseStatusExecutorV0
	Leases   orquestacontext.CodeContextToolLeaseListPortV0
	Observer serverCodeContextToolOwnerObserverV0
	Clock    func() time.Time
}

func serverCodebaseStatusExecutorWithOwnerMarkersV0(
	inner orquestamcp.MCPTransportCodebaseStatusExecutorV0,
	leases orquestacontext.CodeContextToolLeaseListPortV0,
	observer serverCodeContextToolOwnerObserverV0,
	clock func() time.Time,
) orquestamcp.MCPTransportCodebaseStatusExecutorV0 {
	if inner == nil || leases == nil || observer == nil {
		return inner
	}
	return serverCodebaseStatusOwnerMarkerExecutorV0{
		Inner:    inner,
		Leases:   leases,
		Observer: observer,
		Clock:    clock,
	}
}

func (executor serverCodebaseStatusOwnerMarkerExecutorV0) Execute(
	ctx context.Context,
	input orquestamcp.MCPCodebaseStatusToolInputV0,
) (orquestamcp.MCPCodebaseStatusToolResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	inner := executor.Inner
	if inner == nil {
		inner = orquestamcp.MCPCodebaseStatusToolExecutorV0{}
	}
	if executor.Leases == nil || executor.Observer == nil {
		return inner.Execute(ctx, input)
	}
	observations, err := executor.ownerMarkerObservationsV0(ctx, input)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return orquestamcp.MCPCodebaseStatusToolResultV0{}, ctxErr
		}
		return inner.Execute(ctx, input)
	}
	if len(observations) > 0 {
		input.Observations = append(
			append([]orquestamcp.MCPCodebaseStatusToolObservationInputV0{}, input.Observations...),
			observations...,
		)
	}
	return inner.Execute(ctx, input)
}

func (executor serverCodebaseStatusOwnerMarkerExecutorV0) ownerMarkerObservationsV0(
	ctx context.Context,
	input orquestamcp.MCPCodebaseStatusToolInputV0,
) ([]orquestamcp.MCPCodebaseStatusToolObservationInputV0, error) {
	filter := orquestacontext.CodeContextToolLeaseListFilterV0{
		RepositoryRef: strings.TrimSpace(input.RepositoryRef),
		ToolRef:       strings.TrimSpace(input.ToolRef),
	}
	if !input.IncludeTerminal {
		filter.Status = orquestacontext.CodeContextToolLeaseStatusActiveV0
	}
	leases, err := executor.Leases.ListCodeContextToolLeasesV0(ctx, filter)
	if err != nil {
		return nil, err
	}
	now := executor.observedAtV0(input)
	out := make([]orquestamcp.MCPCodebaseStatusToolObservationInputV0, 0, len(leases))
	for _, lease := range leases {
		observation, observeErr := executor.Observer.ObserveCodeContextToolOwnerV0(ctx, lease, now)
		if observeErr != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, ctxErr
			}
			continue
		}
		if strings.TrimSpace(observation.Lease.LeaseRef) == "" {
			continue
		}
		out = append(out, serverCodebaseStatusOwnerMarkerObservationInputV0(observation))
	}
	return out, nil
}

func (executor serverCodebaseStatusOwnerMarkerExecutorV0) observedAtV0(
	input orquestamcp.MCPCodebaseStatusToolInputV0,
) time.Time {
	if value := strings.TrimSpace(input.ObservedAt); value != "" {
		if parsed, err := time.Parse(time.RFC3339, value); err == nil {
			return parsed.UTC()
		}
	}
	if executor.Clock != nil {
		return executor.Clock().UTC()
	}
	return time.Now().UTC()
}

func serverCodebaseStatusOwnerMarkerObservationInputV0(
	observation orquestacontext.CodeContextToolLeaseObservationV0,
) orquestamcp.MCPCodebaseStatusToolObservationInputV0 {
	return orquestamcp.MCPCodebaseStatusToolObservationInputV0{
		ObservationRef: observation.ObservationRef,
		LeaseRef:       observation.Lease.LeaseRef,
		ObservedAt:     observation.ObservedAt,
		LastRequestAt:  observation.LastRequestAt,
		CPUPercent:     observation.CPUPercent,
		ActiveRequests: observation.ActiveRequests,
		CPUHighPercent: observation.CPUHighPercent,
		EvidenceRefs:   append([]string{}, observation.EvidenceRefs...),
	}
}
