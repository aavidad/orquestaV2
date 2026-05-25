package orquestaapprunner

import (
	"context"
	"errors"
	"testing"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestPrepareAppOrchestrationV0DeclaraRutaPreviewCompatibilidad(t *testing.T) {
	prepared, err := PrepareAppOrchestrationV0(PrepareAppOrchestrationRequestV0{
		AppSpec: validFactoryAppSpecForRunnerTestV0(t),
	})
	if err != nil {
		t.Fatalf("PrepareAppOrchestrationV0: %v", err)
	}
	if prepared.RoutePolicy.Mode != AppRunnerRouteModePreviewCompatV0 ||
		prepared.RoutePolicy.PreferredEntrypoint != AppRunnerPreferredEntrypointDirectorV0 ||
		prepared.RoutePolicy.LegacyEntrypoint != AppRunnerLegacyEntrypointPrepareV0 {
		t.Fatalf("route_policy=%+v", prepared.RoutePolicy)
	}
	if !runnerStringInSetV0(prepared.EvidenceRefs, AppRunnerPreviewCompatibilityEvidenceV0) {
		t.Fatalf("evidence_refs=%v", prepared.EvidenceRefs)
	}
}

func TestRunPreparedAppOrchestrationV0BloqueaSiRequiereDirectorV2(t *testing.T) {
	prepared := validPreparedLargeAppForRunTestV0(t)
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	request := validRunPreparedRequestForTestV0(prepared)
	request.RequireDirectorV2 = true

	_, err := RunPreparedAppOrchestrationV0(
		context.Background(),
		request,
		validRunPreparedPortsForTestV0(store, sink, ledger),
	)
	var issue AppRunnerIssueV0
	if !errors.As(err, &issue) || issue.Field != AppRunnerDirectorV2RequiredFieldV0 {
		t.Fatalf("issue=%+v err=%v", issue, err)
	}
}
