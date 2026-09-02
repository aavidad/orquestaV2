package bootstrap

import (
	"context"
	"testing"
	"time"

	statesqlite "orquesta/internal/adapters/state/sqlite"
	"orquesta/internal/application"
	"orquesta/internal/config"
	"orquesta/internal/ports"
)

type lifecycleOnlyAgent struct{ AgentAdapter }

func (lifecycleOnlyAgent) Inspect(context.Context, ports.AgentEnvironmentInspectRequest) (ports.AgentEnvironmentInspectReceipt, error) {
	return ports.AgentEnvironmentInspectReceipt{}, nil
}
func (lifecycleOnlyAgent) Quiesce(context.Context, ports.AgentQuiesceRequest) (ports.AgentQuiesceReceipt, error) {
	return ports.AgentQuiesceReceipt{}, nil
}
func (lifecycleOnlyAgent) Preserve(context.Context, ports.AgentPreserveRequest) (ports.AgentPreserveReceipt, error) {
	return ports.AgentPreserveReceipt{}, nil
}
func (lifecycleOnlyAgent) Close(context.Context, ports.AgentCloseRequest) (ports.AgentCloseReceipt, error) {
	return ports.AgentCloseReceipt{}, nil
}

type completeLifecycleAgent struct{ lifecycleOnlyAgent }

type expiredContinuationWriterAgent struct{ AgentAdapter }

func (expiredContinuationWriterAgent) PrepareExpiredAgentLaunchContinuationV41(
	context.Context, ports.AgentLaunchRequest, application.ExpiredAgentLaunchContinuationCausalBindingV41,
) (application.ExpiredAgentLaunchContinuationPreparationV41, error) {
	return application.ExpiredAgentLaunchContinuationPreparationV41{}, nil
}
func (expiredContinuationWriterAgent) IssueExpiredAgentLaunchContinuationV41(
	context.Context, ports.AgentLaunchRequest, application.ExpiredAgentLaunchContinuationCausalBindingV41,
	application.ExpiredAgentLaunchContinuationIssuanceV41,
) (application.ExpiredAgentLaunchContinuationRecordV41, error) {
	return application.ExpiredAgentLaunchContinuationRecordV41{}, nil
}

func (completeLifecycleAgent) ReconcileQuiesce(context.Context, ports.AgentQuiesceRequest) (ports.AgentQuiesceReceipt, error) {
	return ports.AgentQuiesceReceipt{}, nil
}
func (completeLifecycleAgent) ReconcilePreserve(context.Context, ports.AgentPreserveRequest) (ports.AgentPreserveReceipt, error) {
	return ports.AgentPreserveReceipt{}, nil
}
func (completeLifecycleAgent) ReconcileClose(context.Context, ports.AgentCloseRequest) (ports.AgentCloseReceipt, error) {
	return ports.AgentCloseReceipt{}, nil
}

func TestBuildOrchestratorDependenciesComposesAgentLifecycleAllOrNothing(t *testing.T) {
	snapshot, err := config.Resolve(config.ResolveOptions{})
	if err != nil {
		t.Fatal(err)
	}
	setup := buildSetup{snapshot: snapshot}
	repository := &statesqlite.Repository{}
	builder := func(context.Context, ports.AgentPreserveReceipt, application.GoalRecord, time.Time) (application.ComprobantePreservacionEntornoAgente, error) {
		return application.ComprobantePreservacionEntornoAgente{}, nil
	}
	dependencies := func(agent AgentAdapter, preservation application.AgentEnvironmentPreservationBuilder) application.Dependencies {
		return buildOrchestratorDependencies(
			setup, repository, nil, agent, nil, ports.AgentCapabilities{}, nil,
			buildTestAttestorComposition{},
			executionRuntimeComposition{environmentPreservationBuilder: preservation},
		)
	}

	if got := dependencies(lifecycleOnlyAgent{}, builder).AgentLifecycle; got != nil {
		t.Fatalf("lifecycle without reconciler composed fallback: %+v", got)
	}
	if got := dependencies(completeLifecycleAgent{}, nil).AgentLifecycle; got != nil {
		t.Fatalf("lifecycle without preservation builder composed fallback: %+v", got)
	}

	agent := completeLifecycleAgent{}
	got := dependencies(agent, builder).AgentLifecycle
	if got == nil || got.Store != repository || got.Physical == nil || got.Reconciler == nil || got.BuildPreservation == nil {
		t.Fatalf("complete lifecycle composition = %+v", got)
	}
}

func TestBuildOrchestratorDependenciesComposesExpiredContinuationWriterWithoutLaunchFallback(t *testing.T) {
	snapshot, err := config.Resolve(config.ResolveOptions{})
	if err != nil {
		t.Fatal(err)
	}
	repository := &statesqlite.Repository{}
	agent := expiredContinuationWriterAgent{}
	dependencies := buildOrchestratorDependencies(
		buildSetup{snapshot: snapshot}, repository, nil, agent, nil,
		ports.AgentCapabilities{}, nil, buildTestAttestorComposition{},
	)
	composition := dependencies.ExpiredLaunchContinuation
	if composition == nil || composition.Source != repository || composition.Store != repository ||
		composition.Writer == nil || composition.SessionAuthoritySource != repository ||
		composition.SessionAuthenticationMethod != "execution_token" {
		t.Fatalf("expired continuation composition=%+v", composition)
	}
}
