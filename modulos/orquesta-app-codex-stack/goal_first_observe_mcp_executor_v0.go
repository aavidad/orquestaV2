package orquestaappcodexstack

import (
	"context"
	"fmt"
	"strings"

	orquestaestadovivo "orquesta/modulos/orquesta-estado-vivo"
	orquestagoal "orquesta/modulos/orquesta-goal"
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
	return orquestamcp.WithMCPObserveAppDirectorGoalCausalVerdictV0(
		ctx,
		executor.estadoVivoSourceForObserveV0(),
		input,
		executor.withMaterializedRefsV0(ctx, input, orquestamcp.NewMCPObserveAppDirectorGoalResultV0(input, result)),
	), nil
}

func (executor CodexStackObserveAppDirectorGoalExecutorV0) estadoVivoSourceForObserveV0() orquestaestadovivo.FuenteEvidenciaEstadoPortV0 {
	if executor.stack == nil {
		return nil
	}
	codex := executor.stack.Codex
	if codex.SnapshotSource == nil {
		codex.SnapshotSource = executor.stack.CodexSnapshotSource
	}
	return estadoVivoSourceV0(ConfigV0{
		Stores: executor.stack.Stores,
		Codex:  codex,
	})
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
	observeExecutor := orquestamcp.NewMCPObserveAppDirectorGoalToolExecutorV0(executor.stack.Ports)
	observeExecutor.EstadoVivoSource = executor.estadoVivoSourceForObserveV0()
	result, err := observeExecutor.ObserveAppDirectorGoalTimeoutSnapshotV0(ctx, input)
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
	coordinator := executor.stack.goalFirstObservationCoordinatorV0()
	if coordinator == nil {
		return result
	}
	release, lockErr := coordinator.acquireV0(ctx, runRef)
	if lockErr != nil {
		return result
	}
	defer release()
	state, err := store.LoadGoalWorkStateV0(ctx, runRef)
	if err != nil {
		return result
	}
	refs, ok, err := (stackGoalMaterializedRefsSourceV0{
		Config:               ConfigV0{Codex: executor.stack.Codex},
		GoalStateStore:       store,
		GoalClosureValidator: executor.stack.Ports.GoalClosureValidator,
		// Esta proyeccion forma parte del comando observe y conserva la
		// reparacion dentro del mismo coordinador por run.
		RepairMissingTerminalReceipt: true,
	}).ResolveDirectorGoalMaterializedRefsV0(ctx, state)
	if err != nil || !ok {
		return result
	}
	if repaired, loadErr := store.LoadGoalWorkStateV0(ctx, runRef); loadErr == nil {
		if repaired.LastResult != nil &&
			repaired.LastClosure != nil &&
			goalFirstStringSliceContainsV0(repaired.LastResult.EvidenceRefs, goalFirstRepairReceiptAttemptedEvidenceRefV0) {
			result = codexStackObserveResultWithGoalStateV0(result, repaired)
		}
	}
	return orquestamcp.EnrichMCPObserveAppDirectorGoalWithMaterializedRefsV0(result, refs)
}

func codexStackObserveResultWithGoalStateV0(
	result orquestamcp.MCPObserveAppDirectorGoalToolResultV0,
	state orquestagoal.GoalWorkStateV0,
) orquestamcp.MCPObserveAppDirectorGoalToolResultV0 {
	partial, err := orquestamcp.NewMCPObserveAppDirectorGoalPartialResultFromStateV0(
		orquestamcp.MCPObserveAppDirectorGoalToolInputV0{RunRef: result.RunRef},
		state,
	)
	if err != nil {
		return result
	}
	result.GoalStatus = partial.GoalStatus
	result.ResultRef = partial.ResultRef
	result.ClosureStatus = partial.ClosureStatus
	result.ClosureAccepted = partial.ClosureAccepted
	result.ClosureNeedsRework = partial.ClosureNeedsRework
	result.Summary = partial.Summary
	result.ArtifactRefs = compactCodexStackStringsV0(append(result.ArtifactRefs, partial.ArtifactRefs...))
	result.DomainReceiptRefs = compactCodexStackStringsV0(append(result.DomainReceiptRefs, partial.DomainReceiptRefs...))
	result.EvidenceRefs = compactCodexStackStringsV0(append(result.EvidenceRefs, partial.EvidenceRefs...))
	result.ClosureIssues = append(result.ClosureIssues, partial.ClosureIssues...)
	result.RecommendedAction = partial.RecommendedAction
	return result
}
