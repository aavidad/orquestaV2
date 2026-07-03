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
		"checkpoint temprano dentro del write-set autorizado",
		"checkpoint_started.txt",
		"No vuelques salidas gigantes de herramientas",
		"max_text_bytes=16384",
		"rg --max-count",
		"rg --files | head",
		"El recibo terminal debe declarar artifact_paths",
		"materialized_artifacts",
		"checklist.expected_refs",
		"rework_plan_refs",
		"Evidencia requerida: evidence-ref-required",
		CodexGoalResultMarkerV0,
		CodexGoalResultSchemaV0,
		CodexGoalResultFileNameForGoalRefV0("goal-ref-001"),
		"Materializa primero el directorio del write-set",
		"goal_ref",
		"schema_version",
		"status/estado debe ser terminal explicito",
		"usa literalmente los test_ref",
		"usa literalmente los artifact_ref",
		"\"artifact_paths\":[]",
		"\"materialized_artifacts\"",
		"\"checklist\"",
		"lista todas las rutas relativas de ficheros creados",
		"separa artefactos validos de borradores recuperables",
		"checklist.missing_refs",
		"si escribiste algo fuera del write-set, no lo ocultes",
		"status blocked con summary out_of_scope_artifacts",
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
		!packet.DirectionContract.RequireEarlyCheckpoint ||
		packet.DirectionContract.EarlyCheckpointFile != "checkpoint_started.txt" ||
		packet.DirectionContract.ToolOutputPolicy.MaxTextBytes != CodexGoalToolOutputMaxBytesV0 ||
		!packet.DirectionContract.ToolOutputPolicy.RequireBoundedCommands ||
		!packet.DirectionContract.ToolOutputPolicy.DurableEvidenceRequired ||
		len(packet.DirectionContract.AllowedWriteSet) != 1 ||
		len(packet.DirectionContract.RequiredArtifactContracts) != 1 ||
		packet.RequiredTests[0].Command != "go test -count=1 ./modulos/orquesta-goal" {
		t.Fatalf("packet no conserva gobierno: %+v", packet)
	}
	for _, want := range []string{"artifact_paths", "materialized_artifacts", "checklist", "evidence_refs"} {
		if !hasGoalStringForTestV0(packet.DirectionContract.RequiredTerminalFields, want) {
			t.Fatalf("direction_contract no exige %s: %+v", want, packet.DirectionContract)
		}
	}
	if !hasGoalStringForTestV0(packet.DirectionContract.ToolOutputPolicy.BoundedCommandHints, "rg --max-count") {
		t.Fatalf("direction_contract sin hints de comandos acotados: %+v", packet.DirectionContract.ToolOutputPolicy)
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

func TestBuildCodexGoalPromptV0DomainWorkDelegaReceiptEnAdaptadorV0(t *testing.T) {
	spec := validCodexGoalSpecV0()
	spec.WorkProfileKind = "domain_work"
	spec.ClosurePolicy.RequireDomainReceipt = true

	packet, issues := BuildCodexGoalStartPacketV0(spec)

	if len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
	for _, want := range []string{
		"En trabajos domain_work",
		"materializa el artefacto en el write-set",
		"deja submit_artifact/receipt al adaptador externo de Orquesta",
		"no bloquees por no tener conector REST/MCP",
		"domain_receipt_refs vacio",
		"No pongas public_blocker, pending, ready=false",
	} {
		if !strings.Contains(packet.Prompt, want) {
			t.Fatalf("prompt no contiene %q:\n%s", want, packet.Prompt)
		}
	}
}

func TestBuildCodexGoalPromptV0NoTrataMarkdownWriteSetComoDirectorioV0(t *testing.T) {
	spec := validCodexGoalSpecV0()
	spec.WriteSet = []orquestagoal.GoalWriteScopeV0{{Path: "external/opes/INCIDENCIA_TEST.md"}}

	packet, issues := BuildCodexGoalStartPacketV0(spec)

	if len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
	for _, forbidden := range []string{
		"external/opes/INCIDENCIA_TEST.md/docs/" + CodexGoalResultFileNameV0,
		"Materializa primero el directorio del write-set",
	} {
		if strings.Contains(packet.Prompt, forbidden) {
			t.Fatalf("prompt contiene ruta/instruccion prohibida %q:\n%s", forbidden, packet.Prompt)
		}
	}
	for _, want := range []string{
		"external/opes/INCIDENCIA_TEST.md es un archivo Markdown final",
		"no crees un directorio con ese nombre",
	} {
		if !strings.Contains(packet.Prompt, want) {
			t.Fatalf("prompt no contiene %q:\n%s", want, packet.Prompt)
		}
	}
}

func TestBuildCodexGoalPromptV0UsaSiguienteWriteSetDirectorioParaResultadoDurableV0(t *testing.T) {
	spec := validCodexGoalSpecV0()
	spec.WriteSet = []orquestagoal.GoalWriteScopeV0{
		{Path: "external/opes/INCIDENCIA_TEST.md"},
		{Path: "external/opes/control_audio"},
	}

	packet, issues := BuildCodexGoalStartPacketV0(spec)

	if len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
	if strings.Contains(packet.Prompt, "external/opes/INCIDENCIA_TEST.md/docs/"+CodexGoalResultFileNameV0) {
		t.Fatalf("prompt cuelga resultado bajo markdown:\n%s", packet.Prompt)
	}
	if !strings.Contains(packet.Prompt, "external/opes/control_audio/docs/"+CodexGoalResultFileNameForGoalRefV0(spec.GoalRef)) {
		t.Fatalf("prompt no usa write-set directorio para resultado durable:\n%s", packet.Prompt)
	}
}

func TestBuildCodexGoalPromptV0UsaResultadoDurableUnicoPorGoalRefV0(t *testing.T) {
	spec := validCodexGoalSpecV0()
	spec.GoalRef = "goal-ref-autoprogramming-backlog-srv-task-022-a54b0a70"
	spec.WriteSet = []orquestagoal.GoalWriteScopeV0{{Path: "modulos/orquesta-server"}}

	packet, issues := BuildCodexGoalStartPacketV0(spec)

	if len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
	want := "modulos/orquesta-server/docs/" + CodexGoalResultFileNameForGoalRefV0(spec.GoalRef)
	if !strings.Contains(packet.Prompt, want) ||
		strings.Contains(packet.Prompt, "modulos/orquesta-server/docs/"+CodexGoalResultFileNameV0+" ") {
		t.Fatalf("prompt=%s want=%s", packet.Prompt, want)
	}
}

func TestCodexGoalResultFileNameForGoalRefV0NormalizaNombreV0(t *testing.T) {
	got := CodexGoalResultFileNameForGoalRefV0(" Goal Ref/APG 005: cierre ")
	if got != "orquesta_goal_result_goal-ref-apg-005-cierre.json" ||
		!CodexGoalResultFileNameLooksValidV0(got) ||
		!CodexGoalResultFileNameLooksValidV0(CodexGoalResultFileNameV0) {
		t.Fatalf("got=%q", got)
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
		receipt: CodexGoalStartReceiptV0{
			IssueCode:    "codex_app_server_control_socket_missing",
			EvidenceRefs: []string{"evidence-ref-codex-app-server-control-socket-missing"},
		},
		err: errors.New("backend unavailable"),
	}
	launcher := CodexGoalLauncherV0{Starter: starter}

	receipt, err := launcher.LaunchGoalWorkV0(context.Background(), validCodexGoalSpecV0())

	if err == nil ||
		receipt.Status != orquestagoal.GoalStatusInvalidV0 ||
		!hasGoalIssueCodeForTestV0(receipt.Issues, "codex_app_server_control_socket_missing") ||
		!hasGoalStringForTestV0(receipt.EvidenceRefs, "evidence-ref-codex-app-server-control-socket-missing") {
		t.Fatalf("receipt=%+v err=%v", receipt, err)
	}
}

func TestCodexGoalLauncherV0ClasificaErrorBackendSinIssueCodeV0(t *testing.T) {
	cases := []struct {
		name    string
		message string
		want    string
	}{
		{
			name: "reset stdio",
			message: `Node.js[2796823]: void node::ResetStdio() at ../src/node.cc:751
Assertion failed: !(err != 0) || (err == -1 && (*__errno_location ()) == 1)`,
			want: "codex_app_server_wrapper_stdio_failed",
		},
		{
			name:    "tmux exited",
			message: "tmux session exited before app-server socket was ready",
			want:    "codex_app_server_tmux_session_exited",
		},
		{
			name:    "bwrap permission",
			message: "bwrap: Can't mkdir parents for /tmp/orquesta-runtime: Permission denied",
			want:    "codex_app_server_permission_denied",
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			starter := &recordingCodexGoalStarterV0{
				err: errors.New(tt.message),
			}
			launcher := CodexGoalLauncherV0{Starter: starter}

			receipt, err := launcher.LaunchGoalWorkV0(context.Background(), validCodexGoalSpecV0())

			if err == nil ||
				receipt.Status != orquestagoal.GoalStatusInvalidV0 ||
				!hasGoalIssueCodeForTestV0(receipt.Issues, tt.want) {
				t.Fatalf("receipt=%+v err=%v want_issue=%q", receipt, err, tt.want)
			}
		})
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
			ArtifactPaths:   []string{" docs/orquesta_goal_result_v0.json "},
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
		len(result.ArtifactPaths) != 1 ||
		result.ArtifactPaths[0] != "docs/orquesta_goal_result_v0.json" ||
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

func TestCodexGoalObserverV0ConservaArtefactosMaterializadosChecklistYRework(t *testing.T) {
	observer := CodexGoalObserverV0{Observer: &recordingCodexGoalObserverV0{
		receipt: CodexGoalObservationReceiptV0{
			Status:  orquestagoal.GoalStatusBlockedV0,
			GoalRef: "goal-ref-001",
			MaterializedArtifacts: []orquestagoal.GoalMaterializedArtifactV0{{
				ArtifactRef:  "artifact-ref-tema-001",
				Path:         "temas/tema_001.md",
				ArtifactType: "markdown",
				Status:       orquestagoal.GoalMaterializedArtifactStatusPartialV0,
				EvidenceRefs: []string{"evidence-ref-artifact-partial"},
			}},
			Checklist: orquestagoal.GoalWorkChecklistV0{
				ExpectedRefs:  []string{"check-ref-texto-publicable"},
				MissingRefs:   []string{"check-ref-texto-publicable"},
				EvidenceRefs:  []string{"evidence-ref-checklist"},
				CompletedRefs: []string{"check-ref-fuentes"},
			},
			ReworkPlanRefs: []string{"rework-plan-ref-tema-001"},
		},
	}}

	result, err := observer.ObserveGoalWorkV0(context.Background(), orquestagoal.GoalObservationRequestV0{GoalRef: "goal-ref-001"})

	if err != nil ||
		result.Status != orquestagoal.GoalStatusBlockedV0 ||
		len(result.MaterializedArtifacts) != 1 ||
		result.MaterializedArtifacts[0].Status != orquestagoal.GoalMaterializedArtifactStatusPartialV0 ||
		len(result.Checklist.MissingRefs) != 1 ||
		len(result.ReworkPlanRefs) != 1 ||
		result.ReworkPlanRefs[0] != "rework-plan-ref-tema-001" {
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
		ClosurePolicy: orquestagoal.GoalClosurePolicyV0{
			RequireRequiredTests:                 true,
			RequireArtifactPaths:                 true,
			RequireMaterializedArtifacts:         true,
			RequireChecklist:                     true,
			RequireReworkPlanForPartialArtifacts: true,
			RequiredEvidenceRefs:                 []string{"evidence-ref-required"},
		},
		ReworkPolicy: orquestagoal.GoalReworkPolicyV0{PreferNewGoal: true, PreserveArtifacts: true},
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

func TestCodexGoalObserverV0ClasificaErrorBackendSinIssueCodeV0(t *testing.T) {
	observer := CodexGoalObserverV0{Observer: &recordingCodexGoalObserverV0{
		err: errors.New("codex app-server failed: Operation not permitted (os error 1)"),
	}}

	result, err := observer.ObserveGoalWorkV0(context.Background(), orquestagoal.GoalObservationRequestV0{
		GoalRef:         "goal-ref-001",
		ExternalGoalRef: "external-goal-ref-001",
	})

	if err == nil ||
		result.Status != orquestagoal.GoalStatusInvalidV0 ||
		!hasGoalIssueCodeForTestV0(result.Issues, "codex_app_server_operation_not_permitted") {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestCodexGoalObserverV0ClasificaQuotaFilesystemBackendSinIssueCodeV0(t *testing.T) {
	observer := CodexGoalObserverV0{Observer: &recordingCodexGoalObserverV0{
		err: errors.New("rollout writer failed: Quota exceeded (os error 122)"),
	}}

	result, err := observer.ObserveGoalWorkV0(context.Background(), orquestagoal.GoalObservationRequestV0{
		GoalRef:         "goal-ref-001",
		ExternalGoalRef: "external-goal-ref-001",
	})

	if err == nil ||
		result.Status != orquestagoal.GoalStatusInvalidV0 ||
		!hasGoalIssueCodeForTestV0(result.Issues, "codex_app_server_storage_quota_exceeded") {
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

func hasGoalStringForTestV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
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
