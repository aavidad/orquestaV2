package orquestaappcodexstack

import (
	"context"
	"fmt"
	"strings"

	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

type CodexStackObserveAppDirectorGoalExecutorV0 struct {
	stack *StackV0
}

var _ orquestamcp.MCPTransportObserveAppDirectorGoalTimeoutSnapshotExecutorV0 = CodexStackObserveAppDirectorGoalExecutorV0{}

func NewCodexStackObserveAppDirectorGoalExecutorV0(
	stack *StackV0,
) CodexStackObserveAppDirectorGoalExecutorV0 {
	return CodexStackObserveAppDirectorGoalExecutorV0{stack: stack}
}

func (executor CodexStackObserveAppDirectorGoalExecutorV0) Execute(
	ctx context.Context,
	input orquestamcp.MCPObserveAppDirectorGoalToolInputV0,
) (orquestamcp.MCPObserveAppDirectorGoalToolResultV0, error) {
	if executor.stack == nil {
		return orquestamcp.MCPObserveAppDirectorGoalToolResultV0{}, fmt.Errorf("stack requerido")
	}
	result, err := executor.stack.ObserveAppDirectorGoalV0(
		ctx,
		orquestamcp.ToObserveAppDirectorGoalRequestV0(input),
	)
	if err != nil {
		if publicResult, ok := orquestamcp.NewMCPObserveAppDirectorGoalErrorResultFromErrorV0(input, err); ok {
			return executor.withPartialSnapshotAfterObserveErrorV0(ctx, input, publicResult), nil
		}
		return orquestamcp.MCPObserveAppDirectorGoalToolResultV0{}, err
	}
	return executor.withMaterializedRefsV0(ctx, input, orquestamcp.NewMCPObserveAppDirectorGoalResultV0(input, result)), nil
}

func (executor CodexStackObserveAppDirectorGoalExecutorV0) withPartialSnapshotAfterObserveErrorV0(
	ctx context.Context,
	input orquestamcp.MCPObserveAppDirectorGoalToolInputV0,
	publicResult orquestamcp.MCPObserveAppDirectorGoalToolResultV0,
) orquestamcp.MCPObserveAppDirectorGoalToolResultV0 {
	snapshot, err := executor.ObserveAppDirectorGoalTimeoutSnapshotV0(ctx, input)
	if err != nil {
		return publicResult
	}
	return orquestamcp.NewMCPObserveAppDirectorGoalTimeoutResultWithPartialV0(publicResult, snapshot)
}

func (executor CodexStackObserveAppDirectorGoalExecutorV0) ObserveAppDirectorGoalTimeoutSnapshotV0(
	ctx context.Context,
	input orquestamcp.MCPObserveAppDirectorGoalToolInputV0,
) (orquestamcp.MCPObserveAppDirectorGoalToolResultV0, error) {
	if executor.stack == nil {
		return orquestamcp.MCPObserveAppDirectorGoalToolResultV0{}, fmt.Errorf("stack requerido")
	}
	result, err := orquestamcp.NewMCPObserveAppDirectorGoalToolExecutorV0(executor.stack.Ports).
		ObserveAppDirectorGoalTimeoutSnapshotV0(ctx, input)
	if err != nil {
		return result, err
	}
	result = executor.withProcessRefsV0(ctx, input, result)
	return executor.withMaterializedRefsV0(ctx, input, result), nil
}

func (executor CodexStackObserveAppDirectorGoalExecutorV0) withProcessRefsV0(
	ctx context.Context,
	input orquestamcp.MCPObserveAppDirectorGoalToolInputV0,
	result orquestamcp.MCPObserveAppDirectorGoalToolResultV0,
) orquestamcp.MCPObserveAppDirectorGoalToolResultV0 {
	runRef := strings.TrimSpace(input.RunRef)
	if runRef == "" || executor.stack == nil || executor.stack.Stores.ProcessRegistry == nil {
		return result
	}
	lister, ok := executor.stack.Stores.ProcessRegistry.(orquestacionnucleoapp.AgentProcessRegistryListPortV0)
	if !ok {
		return result
	}
	records, err := lister.ListAgentProcessesV0(ctx, orquestacionnucleoapp.AgentProcessRegistryListFilterV0{RunID: runRef})
	if err != nil {
		return result
	}
	refs := append([]string(nil), result.ProcessRefs...)
	evidence := append([]string(nil), result.EvidenceRefs...)
	for _, record := range records {
		if ref := strings.TrimSpace(record.ProcessRef); ref != "" {
			refs = append(refs, ref)
		}
		evidence = append(evidence, record.EvidenceRefs...)
	}
	result.ProcessRefs = compactCodexStackStringsV0(refs)
	result.EvidenceRefs = compactCodexStackStringsV0(evidence)
	return result
}

func (executor CodexStackObserveAppDirectorGoalExecutorV0) withMaterializedRefsV0(
	ctx context.Context,
	input orquestamcp.MCPObserveAppDirectorGoalToolInputV0,
	result orquestamcp.MCPObserveAppDirectorGoalToolResultV0,
) orquestamcp.MCPObserveAppDirectorGoalToolResultV0 {
	if executor.stack == nil {
		return result
	}
	runRef := strings.TrimSpace(result.RunRef)
	if runRef == "" {
		runRef = strings.TrimSpace(input.RunRef)
	}
	if runRef == "" {
		return result
	}
	store := executor.stack.Ports.GoalStateStore
	if store == nil {
		store = executor.stack.Stores.AppGoalStateStore
	}
	if store == nil {
		return result
	}
	state, err := store.LoadGoalWorkStateV0(ctx, runRef)
	if err != nil {
		return result
	}
	refs, ok, err := (stackGoalMaterializedRefsSourceV0{
		Config: ConfigV0{Codex: executor.stack.Codex},
	}).ResolveDirectorGoalMaterializedRefsV0(ctx, state)
	if err != nil || !ok {
		return result
	}
	return orquestamcp.EnrichMCPObserveAppDirectorGoalWithMaterializedRefsV0(result, refs)
}
