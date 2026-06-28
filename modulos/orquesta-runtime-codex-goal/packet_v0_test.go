package orquestaruntimecodexgoal

import (
	"context"
	"errors"
	"strings"
	"testing"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

func TestBuildCodexGoalStartPacketV0IncluyeContratoDeDireccion(t *testing.T) {
	packet, issues := BuildCodexGoalStartPacketV0(validCodexGoalSpecV0())
	if len(issues) != 0 {
		t.Fatalf("issues=%v", issues)
	}
	for _, expected := range []string{
		"Director operativo interno",
		"Orquesta gobierna desde fuera",
		"request-ref-001",
		"implementation",
		"docs/estado_actual_2026-05-17.md",
		"kind=doc",
		"modulos/orquesta-goal",
		"test_ref=test-ref-goal",
		"go test -count=1 ./modulos/orquesta-goal",
		"criterio-ref-goal-first",
		"artifact-ref-goal-summary",
		"Materializa cada artefacto requerido dentro de un write-set autorizado",
		"<artifact_type>.json",
		"Evidencia requerida: evidence-ref-required",
		CodexGoalResultMarkerV0,
		CodexGoalResultFileNameV0,
		"goal_ref",
		"usa literalmente los test_ref",
		"usa literalmente los artifact_ref",
		"Resultado estructurado obligatorio",
		"Los command de Tests requeridos son parte del contrato neutral acotado",
	} {
		if !strings.Contains(packet.Prompt, expected) {
			t.Fatalf("prompt no contiene %q:\n%s", expected, packet.Prompt)
		}
	}
	if packet.SchemaVersion != CodexGoalStartPacketSchemaV0 {
		t.Fatalf("schema=%q", packet.SchemaVersion)
	}
	if packet.RequestRef != "request-ref-001" ||
		packet.ProjectRef != "project-ref-orquesta" ||
		packet.WorkKind != "idle_self_improvement" ||
		packet.WorkProfileKind != "implementation" ||
		len(packet.AcceptanceCriteria) != 1 ||
		packet.AcceptanceCriteria[0] != "criterio-ref-goal-first" ||
		len(packet.ArtifactContracts) != 1 ||
		packet.ArtifactContracts[0].ArtifactRef != "artifact-ref-goal-summary" ||
		len(packet.EvidenceRefs) != 1 ||
		packet.EvidenceRefs[0] != "evidence-ref-input" ||
		!packet.ClosurePolicy.RequireRequiredTests ||
		!packet.ReworkPolicy.PreferNewGoal ||
		packet.Budget.MaxSubgoals != 2 ||
		len(packet.RequiredTests) != 1 ||
		packet.RequiredTests[0].Command != "go test -count=1 ./modulos/orquesta-goal" {
		t.Fatalf("packet no conserva gobierno: %+v", packet)
	}
}

func TestBuildCodexGoalPromptV0IncluyePropositoDeContextRefs(t *testing.T) {
	spec := validCodexGoalSpecV0()
	spec.ContextRefs = append(spec.ContextRefs, orquestagoal.GoalContextRefV0{
		Kind:    "input_field_value",
		Ref:     "input-field-output_contract-value-abc123",
		Purpose: `Campo input_fields.output_contract inlineado de forma acotada: {"artifact_type":"content_block"}`,
	})

	packet, issues := BuildCodexGoalStartPacketV0(spec)

	if len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
	for _, want := range []string{
		"input-field-output_contract-value-abc123 kind=input_field_value",
		`Campo input_fields.output_contract inlineado de forma acotada`,
		`"artifact_type":"content_block"`,
	} {
		if !strings.Contains(packet.Prompt, want) {
			t.Fatalf("prompt no contiene %q:\n%s", want, packet.Prompt)
		}
	}
}

func TestBuildCodexGoalStartPacketV0RechazaPromptDemasiadoGrande(t *testing.T) {
	spec := validCodexGoalSpecV0()
	spec.Objective = strings.Repeat("x", CodexGoalMaxPromptBytesV0)

	packet, issues := BuildCodexGoalStartPacketV0(spec)

	if len(issues) != 1 ||
		issues[0].Code != ErrCodexGoalPromptTooLargeV0 ||
		issues[0].Field != "prompt" ||
		packet.SchemaVersion != "" {
		t.Fatalf("packet=%+v issues=%+v", packet, issues)
	}
}

func TestCodexGoalLauncherV0RechazaSpecInvalido(t *testing.T) {
	launcher := CodexGoalLauncherV0{Starter: fakeCodexGoalStarterV0{}}
	receipt, err := launcher.LaunchGoalWorkV0(context.Background(), orquestagoal.GoalWorkSpecV0{})
	if err == nil {
		t.Fatal("expected error")
	}
	if receipt.Status != orquestagoal.GoalStatusInvalidV0 {
		t.Fatalf("receipt=%+v", receipt)
	}
}

func TestCodexGoalLauncherV0LlamaStarterInyectado(t *testing.T) {
	starter := &recordingCodexGoalStarterV0{}
	launcher := CodexGoalLauncherV0{Starter: starter}
	receipt, err := launcher.LaunchGoalWorkV0(context.Background(), validCodexGoalSpecV0())
	if err != nil {
		t.Fatalf("launch: %v", err)
	}
	if !starter.called {
		t.Fatal("starter no llamado")
	}
	if receipt.Status != orquestagoal.GoalStatusAcceptedV0 {
		t.Fatalf("receipt=%+v", receipt)
	}
	if receipt.ExternalGoalRef != "external-goal-ref-001" {
		t.Fatalf("external=%q", receipt.ExternalGoalRef)
	}
}

func TestCodexGoalLauncherV0PreservaIssueCodeDeBackendV0(t *testing.T) {
	starter := &recordingCodexGoalStarterV0{
		receipt: CodexGoalStartReceiptV0{IssueCode: "codex_app_server_control_socket_missing"},
		err:     errors.New("backend unavailable"),
	}
	launcher := CodexGoalLauncherV0{Starter: starter}

	receipt, err := launcher.LaunchGoalWorkV0(context.Background(), validCodexGoalSpecV0())

	if err == nil ||
		receipt.Status != orquestagoal.GoalStatusInvalidV0 ||
		!hasGoalIssueCodeForTestV0(receipt.Issues, "codex_app_server_control_socket_missing") {
		t.Fatalf("receipt=%+v err=%v", receipt, err)
	}
}

func TestBuildCodexGoalObservationRequestV0ValidaRefs(t *testing.T) {
	packet, issues := BuildCodexGoalObservationRequestV0(orquestagoal.GoalObservationRequestV0{
		GoalRef:         "goal-ref-001",
		ExternalGoalRef: "external-goal-ref-001",
	})
	if len(issues) != 0 {
		t.Fatalf("issues=%v", issues)
	}
	if packet.SchemaVersion != CodexGoalObservationRequestSchemaV0 ||
		packet.GoalRef != "goal-ref-001" ||
		packet.ExternalGoalRef != "external-goal-ref-001" {
		t.Fatalf("packet=%+v", packet)
	}

	_, issues = BuildCodexGoalObservationRequestV0(orquestagoal.GoalObservationRequestV0{
		ExternalGoalRef: "/tmp/external-goal",
	})
	if len(issues) == 0 {
		t.Fatal("expected issues")
	}
}

func TestCodexGoalObserverV0LlamaObserverInyectado(t *testing.T) {
	backend := &recordingCodexGoalObserverV0{
		receipt: CodexGoalObservationReceiptV0{
			Status:          orquestagoal.GoalStatusCompleteV0,
			GoalRef:         "goal-ref-001",
			ExternalGoalRef: "external-goal-ref-001",
			Summary:         "cierre observado",
			EvidenceRefs:    []string{"evidence-ref-observed"},
			RequiredTestResults: []orquestagoal.GoalRequiredTestResultV0{{
				TestRef: "test-ref-goal",
				Status:  "passed",
			}},
		},
	}
	observer := CodexGoalObserverV0{Observer: backend}

	result, err := observer.ObserveGoalWorkV0(context.Background(), orquestagoal.GoalObservationRequestV0{
		GoalRef:         "goal-ref-001",
		ExternalGoalRef: "external-goal-ref-001",
	})

	if err != nil {
		t.Fatalf("observe: %v", err)
	}
	if !backend.called ||
		backend.lastRequest.GoalRef != "goal-ref-001" ||
		result.Status != orquestagoal.GoalStatusCompleteV0 ||
		result.GoalRef != "goal-ref-001" ||
		result.ExternalGoalRef != "external-goal-ref-001" ||
		len(result.EvidenceRefs) != 1 {
		t.Fatalf("called=%v request=%+v result=%+v", backend.called, backend.lastRequest, result)
	}
}

func TestCodexGoalObserverV0RechazaObserverAusente(t *testing.T) {
	var observer CodexGoalObserverV0

	result, err := observer.ObserveGoalWorkV0(context.Background(), orquestagoal.GoalObservationRequestV0{GoalRef: "goal-ref-001"})

	if err == nil || result.Status != orquestagoal.GoalStatusInvalidV0 ||
		!hasGoalIssueCodeForTestV0(result.Issues, ErrCodexGoalObserverMissingV0) {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestCodexGoalObserverV0RechazaGoalRefCruzado(t *testing.T) {
	observer := CodexGoalObserverV0{Observer: &recordingCodexGoalObserverV0{
		receipt: CodexGoalObservationReceiptV0{
			Status:  orquestagoal.GoalStatusCompleteV0,
			GoalRef: "goal-ref-otro",
		},
	}}

	result, err := observer.ObserveGoalWorkV0(context.Background(), orquestagoal.GoalObservationRequestV0{GoalRef: "goal-ref-001"})

	if err == nil || result.Status != orquestagoal.GoalStatusInvalidV0 ||
		!hasGoalIssueFieldForTestV0(result.Issues, "goal_ref") {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestCodexGoalObserverV0RechazaRefsInvalidasDelBackend(t *testing.T) {
	observer := CodexGoalObserverV0{Observer: &recordingCodexGoalObserverV0{
		receipt: CodexGoalObservationReceiptV0{
			Status:       orquestagoal.GoalStatusCompleteV0,
			GoalRef:      "goal-ref-001",
			EvidenceRefs: []string{"/tmp/evidence"},
		},
	}}

	result, err := observer.ObserveGoalWorkV0(context.Background(), orquestagoal.GoalObservationRequestV0{GoalRef: "goal-ref-001"})

	if err == nil || result.Status != orquestagoal.GoalStatusInvalidV0 ||
		!hasGoalIssueFieldForTestV0(result.Issues, "evidence_refs") {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func validCodexGoalSpecV0() orquestagoal.GoalWorkSpecV0 {
	return orquestagoal.GoalWorkSpecV0{
		GoalRef:            "goal-ref-001",
		RequestRef:         "request-ref-001",
		ProjectRef:         "project-ref-orquesta",
		WorkKind:           "idle_self_improvement",
		WorkProfileKind:    "implementation",
		Objective:          "Programar el contrato goal-first.",
		DirectorKind:       orquestagoal.GoalDirectorKindCodexGoalV0,
		ContextRefs:        []orquestagoal.GoalContextRefV0{{Kind: "doc", Ref: "docs/estado_actual_2026-05-17.md", Required: true}},
		RuleRefs:           []orquestagoal.GoalRuleRefV0{{Ref: "AGENTS.md", Enforcement: orquestagoal.GoalRuleEnforcementAdvisoryV0}},
		WriteSet:           []orquestagoal.GoalWriteScopeV0{{Path: "modulos/orquesta-goal"}},
		RequiredTests:      []orquestagoal.GoalRequiredTestV0{{TestRef: "test-ref-goal", Command: "go test -count=1 ./modulos/orquesta-goal"}},
		AcceptanceCriteria: []string{"criterio-ref-goal-first"},
		ArtifactContracts:  []orquestagoal.GoalArtifactContractV0{{ArtifactRef: "artifact-ref-goal-summary", ArtifactType: "summary", Required: true}},
		EvidenceRefs:       []string{"evidence-ref-input"},
		Budget:             orquestagoal.GoalBudgetV0{MaxSubgoals: 2},
		ClosurePolicy:      orquestagoal.GoalClosurePolicyV0{RequireRequiredTests: true, RequiredEvidenceRefs: []string{"evidence-ref-required"}},
		ReworkPolicy:       orquestagoal.GoalReworkPolicyV0{PreferNewGoal: true, PreserveArtifacts: true},
	}
}

type fakeCodexGoalStarterV0 struct{}

func (fakeCodexGoalStarterV0) StartCodexGoalV0(context.Context, CodexGoalStartPacketV0) (CodexGoalStartReceiptV0, error) {
	return CodexGoalStartReceiptV0{}, nil
}

type recordingCodexGoalStarterV0 struct {
	called  bool
	receipt CodexGoalStartReceiptV0
	err     error
}

func (starter *recordingCodexGoalStarterV0) StartCodexGoalV0(context.Context, CodexGoalStartPacketV0) (CodexGoalStartReceiptV0, error) {
	starter.called = true
	if starter.err != nil {
		return starter.receipt, starter.err
	}
	if starter.receipt.Status != "" || starter.receipt.IssueCode != "" {
		return starter.receipt, nil
	}
	return CodexGoalStartReceiptV0{
		Status:          orquestagoal.GoalStatusAcceptedV0,
		ExternalGoalRef: "external-goal-ref-001",
		EvidenceRefs:    []string{"evidence-ref-started"},
	}, nil
}

type recordingCodexGoalObserverV0 struct {
	called      bool
	lastRequest CodexGoalObservationRequestV0
	receipt     CodexGoalObservationReceiptV0
	err         error
}

func (observer *recordingCodexGoalObserverV0) ObserveCodexGoalV0(
	_ context.Context,
	request CodexGoalObservationRequestV0,
) (CodexGoalObservationReceiptV0, error) {
	observer.called = true
	observer.lastRequest = request
	if observer.err != nil {
		return observer.receipt, observer.err
	}
	return observer.receipt, nil
}

func TestCodexGoalObserverV0PropagaErrorBackend(t *testing.T) {
	observer := CodexGoalObserverV0{Observer: &recordingCodexGoalObserverV0{
		err: errors.New("backend rejected"),
	}}

	result, err := observer.ObserveGoalWorkV0(context.Background(), orquestagoal.GoalObservationRequestV0{GoalRef: "goal-ref-001"})

	if err == nil || result.Status != orquestagoal.GoalStatusInvalidV0 ||
		!hasGoalIssueCodeForTestV0(result.Issues, ErrCodexGoalObservationRejectedV0) {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestCodexGoalObserverV0PreservaIssueCodeDeBackendV0(t *testing.T) {
	observer := CodexGoalObserverV0{Observer: &recordingCodexGoalObserverV0{
		err: errors.New("backend unavailable"),
		receipt: CodexGoalObservationReceiptV0{
			IssueCode:       "codex_app_server_control_socket_missing",
			ExternalGoalRef: "external-goal-ref-001",
		},
	}}

	result, err := observer.ObserveGoalWorkV0(context.Background(), orquestagoal.GoalObservationRequestV0{
		GoalRef:         "goal-ref-001",
		ExternalGoalRef: "external-goal-ref-001",
	})

	if err == nil ||
		result.Status != orquestagoal.GoalStatusInvalidV0 ||
		result.ExternalGoalRef != "external-goal-ref-001" ||
		!hasGoalIssueCodeForTestV0(result.Issues, "codex_app_server_control_socket_missing") {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func hasGoalIssueCodeForTestV0(issues []orquestagoal.GoalWorkIssueV0, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

func hasGoalIssueFieldForTestV0(issues []orquestagoal.GoalWorkIssueV0, field string) bool {
	for _, issue := range issues {
		if issue.Field == field {
			return true
		}
	}
	return false
}
