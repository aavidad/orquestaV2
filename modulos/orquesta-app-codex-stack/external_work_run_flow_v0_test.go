package orquestaappcodexstack

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

func TestCodexStackV0ExternalWorkRunCreaRunSinDirectorInicial(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())

	result := postExternalWorkRunStackLegacyV0(t, stack)
	if result.RoutePolicy != orquestamcp.MCPExternalWorkRunRoutePolicyLegacyDirectorLoopV0 ||
		result.DirectorExecutionMode != orquestamcp.MCPExternalWorkRunDirectorExecutionModeLegacyLoopV0 {
		t.Fatalf("external-work debe declararse legacy hasta migracion Goal-first: %+v", result)
	}
	run, err := stack.Stores.RunStore.LoadRunV0(context.Background(), result.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if string(run.CurrentPhase) != "programacion" ||
		len(run.DirectorQuestions) != 1 ||
		len(run.StartedAgents) != 0 {
		t.Fatalf("run=%+v", run)
	}

	ranking := postRunQueuePriorityStackV0(t, stack, orquestamcp.MCPRunQueuePriorityToolInputV0{
		Action:   "rank",
		QueueRef: DefaultRunQueueRefV0,
	})
	if len(ranking.Ranked) != 1 ||
		ranking.Ranked[0].RunRef != result.RunRef ||
		ranking.Ranked[0].AppRef != "opes" {
		t.Fatalf("ranking=%+v result=%+v", ranking, result)
	}
}

func TestCodexStackV0ExternalWorkRunLegacyOptInRequiereModoExplicito(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())

	result := postExternalWorkRunStackRawV0(t, stack, http.StatusBadRequest, defaultExternalWorkRunChangeForTestV0())
	if result.Estado != orquestamcp.MCPExternalWorkRunEstadoErrorV0 ||
		result.RoutePolicy != orquestamcp.MCPExternalWorkRunRoutePolicyGoalFirstV0 ||
		result.DirectorExecutionMode != orquestamcp.MCPExternalWorkRunDirectorExecutionModeGoalFirstV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != orquestamcp.MCPExternalWorkRunLegacyDirectorModeRequiredV0 ||
		result.Errores[0].Field != "director_execution_mode" ||
		!codexStackStringInSetForTestV0(result.NextActions, orquestamcp.MCPExternalWorkRunNextActionDoNotFallbackLegacyV0) ||
		!codexStackStringInSetForTestV0(result.NextActions, orquestamcp.MCPExternalWorkRunNextActionEnableLegacyOptInV0) {
		t.Fatalf("result=%+v", result)
	}
	ranking := postRunQueuePriorityStackV0(t, stack, orquestamcp.MCPRunQueuePriorityToolInputV0{
		Action:   "rank",
		QueueRef: DefaultRunQueueRefV0,
	})
	if len(ranking.Ranked) != 0 {
		t.Fatalf("sin modo legacy no debe encolar: ranking=%+v", ranking)
	}
}

func TestCodexStackV0ExternalWorkRunModoLegacySinOptInNoDegrada(t *testing.T) {
	config := codexStackBaseConfigForTestV0(t, newFakeCodexStackRuntimeV0(), nil, nil)
	config.AllowLegacyExternalWorkRun = false
	stack, err := BuildStackV0(config)
	if err != nil {
		t.Fatalf("BuildStackV0: %v", err)
	}

	result := postExternalWorkRunStackRawWithModeV0(
		t,
		stack,
		http.StatusBadRequest,
		defaultExternalWorkRunChangeForTestV0(),
		orquestamcp.MCPExternalWorkRunDirectorExecutionModeLegacyLoopV0,
	)
	if result.Estado != orquestamcp.MCPExternalWorkRunEstadoErrorV0 ||
		result.RoutePolicy != orquestamcp.MCPExternalWorkRunRoutePolicyGoalFirstV0 ||
		result.DirectorExecutionMode != orquestamcp.MCPExternalWorkRunDirectorExecutionModeGoalFirstV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != orquestamcp.MCPExternalWorkRunLegacyDirectorLoopOptInRequiredV0 ||
		result.Errores[0].Field != "legacy_director_loop_opt_in" {
		t.Fatalf("result=%+v", result)
	}
}

func TestCodexStackV0ExternalWorkRunSinBackendGoalNoDegradaLegacyPorDefecto(t *testing.T) {
	config := codexStackBaseConfigForTestV0(t, newFakeCodexStackRuntimeV0(), nil, nil)
	config.AllowLegacyExternalWorkRun = false
	stack, err := BuildStackV0(config)
	if err != nil {
		t.Fatalf("BuildStackV0: %v", err)
	}

	result := postExternalWorkRunStackRawV0(t, stack, http.StatusBadRequest, defaultExternalWorkRunChangeForTestV0())
	if result.Estado != orquestamcp.MCPExternalWorkRunEstadoErrorV0 ||
		result.RoutePolicy != orquestamcp.MCPExternalWorkRunRoutePolicyGoalFirstV0 ||
		result.DirectorExecutionMode != orquestamcp.MCPExternalWorkRunDirectorExecutionModeGoalFirstV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != orquestamcp.MCPExternalWorkRunGoalBackendRequiredV0 ||
		!codexStackStringInSetForTestV0(result.NextActions, orquestamcp.MCPExternalWorkRunNextActionConfigureGoalBackendV0) ||
		!codexStackStringInSetForTestV0(result.NextActions, orquestamcp.MCPExternalWorkRunNextActionDoNotFallbackLegacyV0) {
		t.Fatalf("result sin backend goal inesperado=%+v", result)
	}
	ranking := postRunQueuePriorityStackV0(t, stack, orquestamcp.MCPRunQueuePriorityToolInputV0{
		Action:   "rank",
		QueueRef: DefaultRunQueueRefV0,
	})
	if len(ranking.Ranked) != 0 {
		t.Fatalf("sin backend goal no debe encolar legacy: ranking=%+v", ranking)
	}
}

func TestCodexStackV0ExternalWorkRunModoNoLegacyNoActivaLegacyAunqueOptInDisponible(t *testing.T) {
	config := codexStackBaseConfigForTestV0(t, newFakeCodexStackRuntimeV0(), nil, nil)
	config.AllowLegacyExternalWorkRun = true
	stack, err := BuildStackV0(config)
	if err != nil {
		t.Fatalf("BuildStackV0: %v", err)
	}

	result := postExternalWorkRunStackRawWithModeV0(
		t,
		stack,
		http.StatusBadRequest,
		defaultExternalWorkRunChangeForTestV0(),
		"goal_first",
	)
	if result.Estado != orquestamcp.MCPExternalWorkRunEstadoErrorV0 ||
		result.RoutePolicy != orquestamcp.MCPExternalWorkRunRoutePolicyGoalFirstV0 ||
		result.DirectorExecutionMode != orquestamcp.MCPExternalWorkRunDirectorExecutionModeGoalFirstV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != orquestamcp.MCPExternalWorkRunLegacyDirectorModeRequiredV0 ||
		!codexStackStringInSetForTestV0(result.NextActions, orquestamcp.MCPExternalWorkRunNextActionDoNotFallbackLegacyV0) {
		t.Fatalf("result=%+v", result)
	}
	ranking := postRunQueuePriorityStackV0(t, stack, orquestamcp.MCPRunQueuePriorityToolInputV0{
		Action:   "rank",
		QueueRef: DefaultRunQueueRefV0,
	})
	if len(ranking.Ranked) != 0 {
		t.Fatalf("modo no legacy no debe encolar legacy: ranking=%+v", ranking)
	}
}

func TestCodexStackV0ExternalWorkRunConBackendGoalArrancaGoalFirstSinColaLegacy(t *testing.T) {
	launcher := &goalFirstQueueLauncherForTestV0{}
	goalStates := newGoalFirstQueueStateStoreForTestV0()
	stack := mustBuildCodexStackWithGoalBackendForTestV0(
		t,
		newFakeCodexStackRuntimeV0(),
		launcher,
		&goalFirstQueueObserverForTestV0{},
		goalStates,
	)

	result := postExternalWorkRunStackV0(t, stack)
	if result.RoutePolicy != orquestamcp.MCPExternalWorkRunRoutePolicyGoalFirstV0 ||
		result.DirectorExecutionMode != orquestamcp.MCPExternalWorkRunDirectorExecutionModeGoalFirstV0 ||
		result.GoalRef == "" ||
		result.DirectorQuestionRef != "" ||
		!codexStackStringInSetForTestV0(result.NextActions, orquestamcp.MCPExternalWorkRunNextActionObserverRequiredV0) ||
		!codexStackStringInSetForTestV0(result.NextActions, orquestamcp.MCPExternalWorkRunNextActionObserveGoalV0) ||
		!codexStackStringInSetForTestV0(result.NextActions, orquestamcp.MCPExternalWorkRunNextActionObserveActiveGoalsV0) ||
		!codexStackStringInSetForTestV0(result.EvidenceRefs, "evidence-ref-external-work-goal-first-observer-required") {
		t.Fatalf("result goal-first inesperado=%+v", result)
	}
	if len(launcher.specs) != 1 ||
		launcher.specs[0].DirectorKind != orquestagoal.GoalDirectorKindCodexGoalV0 ||
		!launcher.specs[0].ClosurePolicy.RequireDomainReceipt {
		t.Fatalf("specs=%+v", launcher.specs)
	}
	run, err := stack.Stores.RunStore.LoadRunV0(context.Background(), result.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if string(run.CurrentPhase) != "programacion" ||
		len(run.Tasks) != 0 ||
		len(run.FunctionContracts) != 0 ||
		len(run.DirectorQuestions) != 0 ||
		len(run.StartedAgents) != 0 {
		t.Fatalf("run goal-first contiene loop legacy: %+v", run)
	}
	state, err := goalStates.LoadGoalWorkStateV0(context.Background(), result.RunRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	if state.GoalRef != result.GoalRef ||
		state.Spec.RunRef != result.RunRef ||
		state.Spec.DirectorKind != orquestagoal.GoalDirectorKindCodexGoalV0 {
		t.Fatalf("state=%+v result=%+v", state, result)
	}
	ranking := postRunQueuePriorityStackV0(t, stack, orquestamcp.MCPRunQueuePriorityToolInputV0{
		Action:   "rank",
		QueueRef: DefaultRunQueueRefV0,
	})
	if len(ranking.Ranked) != 0 {
		t.Fatalf("goal-first no debe encolar loop legacy: ranking=%+v", ranking)
	}
	supervisor := postRunSupervisorStackV0(t, stack, orquestamcp.MCPRunSupervisorToolInputV0{
		RequestID:     "req-external-work-goal-supervise-001",
		CorrelationID: "corr-external-work-goal-supervise-001",
		RunRef:        result.RunRef,
	})
	if supervisor.StopReason != "goal_first_observe_required" ||
		!codexStackStringInSetForTestV0(supervisor.NextActions, "observe_goal") ||
		!codexStackStringInSetForTestV0(supervisor.NextActions, "do_not_supervise_goal_first_with_legacy_loop") {
		t.Fatalf("supervisor goal-first inesperado=%+v", supervisor)
	}
	jobStats := postOPESDirectorJobStatsV0(t, stack, "job-ref-001")
	if jobStats.Estado != orquestamcp.MCPDirectorStatsEstadoOKV0 ||
		jobStats.ExternalJob == nil ||
		jobStats.ExternalJob.Status != "running" ||
		jobStats.ExternalJob.StatusReason != codexStackExternalJobStatusReasonGoalFirstRunningV0 ||
		jobStats.ExternalJob.DirectorExecutionMode != orquestamcp.MCPExternalWorkRunDirectorExecutionModeGoalFirstV0 ||
		jobStats.ExternalJob.GoalRef != result.GoalRef ||
		jobStats.ExternalJob.TaskRef != "" ||
		jobStats.ExternalJob.AgentRef != "" ||
		jobStats.Goal == nil ||
		jobStats.Goal.GoalRef != result.GoalRef {
		t.Fatalf("jobStats goal-first inesperado=%+v result=%+v", jobStats, result)
	}
}

func TestCodexStackV0ExternalWorkRunGoalFirstExplicitoArrancaGoalSinLegacyV0(t *testing.T) {
	launcher := &goalFirstQueueLauncherForTestV0{}
	goalStates := newGoalFirstQueueStateStoreForTestV0()
	stack := mustBuildCodexStackWithGoalBackendForTestV0(
		t,
		newFakeCodexStackRuntimeV0(),
		launcher,
		&goalFirstQueueObserverForTestV0{},
		goalStates,
	)

	result := postExternalWorkRunStackRawWithModeV0(
		t,
		stack,
		http.StatusOK,
		defaultExternalWorkRunChangeForTestV0(),
		orquestamcp.MCPExternalWorkRunDirectorExecutionModeGoalFirstV0,
	)

	if result.RoutePolicy != orquestamcp.MCPExternalWorkRunRoutePolicyGoalFirstV0 ||
		result.DirectorExecutionMode != orquestamcp.MCPExternalWorkRunDirectorExecutionModeGoalFirstV0 ||
		result.GoalRef == "" ||
		result.DirectorQuestionRef != "" ||
		len(launcher.specs) != 1 {
		t.Fatalf("result=%+v specs=%+v", result, launcher.specs)
	}
	ranking := postRunQueuePriorityStackV0(t, stack, orquestamcp.MCPRunQueuePriorityToolInputV0{
		Action:   "rank",
		QueueRef: DefaultRunQueueRefV0,
	})
	if len(ranking.Ranked) != 0 {
		t.Fatalf("goal_first explicito no debe encolar legacy: ranking=%+v", ranking)
	}
}

func TestCodexStackV0ExternalWorkRunGoalFirstConservaInputFieldsOPESV0(t *testing.T) {
	launcher := &goalFirstQueueLauncherForTestV0{}
	goalStates := newGoalFirstQueueStateStoreForTestV0()
	stack := mustBuildCodexStackWithGoalBackendForTestV0(
		t,
		newFakeCodexStackRuntimeV0(),
		launcher,
		&goalFirstQueueObserverForTestV0{},
		goalStates,
	)
	change := defaultExternalWorkRunChangeForTestV0()
	change.ExternalWork.InputFields = append(change.ExternalWork.InputFields,
		orquestadomainwork.DomainWorkFieldV0{
			Name:  "course_root_abs",
			Value: "/home/alberto/Trabajo/OPES/opes-salidas/curso",
		},
		orquestadomainwork.DomainWorkFieldV0{
			Name:  "topic_dir_abs",
			Value: "/home/alberto/Trabajo/OPES/opes-salidas/curso/tema_001",
		},
		orquestadomainwork.DomainWorkFieldV0{
			Name:  "topic_manifest_abs",
			Value: "/home/alberto/Trabajo/OPES/opes-salidas/curso/tema_001/manifest.json",
		},
		orquestadomainwork.DomainWorkFieldV0{
			Name:  "program_json_abs",
			Value: "/home/alberto/Trabajo/OPES/programas/programa.json",
		},
		orquestadomainwork.DomainWorkFieldV0{
			Name:   "required_read_refs",
			Values: []string{"temario/tema_001.md", "normativa/ley-ref-001"},
		},
		orquestadomainwork.DomainWorkFieldV0{
			Name:   "required_outputs",
			Values: []string{"04_markdown/tema_001.md", "paquete_final/tests.json"},
		},
		orquestadomainwork.DomainWorkFieldV0{
			Name:      "output_contract",
			ValueJSON: []byte(`{"artifact_type":"content_block","min_words":1200}`),
		},
	)

	result := postExternalWorkRunStackWithChangeV0(t, stack, change)

	if result.GoalRef == "" || len(launcher.specs) != 1 {
		t.Fatalf("goal-first no lanzo spec: result=%+v specs=%+v", result, launcher.specs)
	}
	launchedContext := codexStackGoalContextRefsTextForTestV0(launcher.specs[0].ContextRefs)
	for _, want := range []string{
		"input_fields.course_root_abs",
		"basename=curso",
		"input_fields.topic_dir_abs",
		"basename=tema_001",
		"input_fields.topic_manifest_abs",
		"basename=manifest.json",
		"input_fields.program_json_abs",
		"basename=programa.json",
		"input_fields.required_read_refs",
		"temario/tema_001.md",
		"normativa/ley-ref-001",
		"input_fields.required_outputs",
		"04_markdown/tema_001.md",
		"paquete_final/tests.json",
		"input_fields.output_contract",
		`"artifact_type":"content_block"`,
		"payload_ref",
		"input_fields_summary",
		"Contrato DomainWork trae 8 input_fields",
		"inlineados=8",
		"payload_refs=0",
		"omitidos=0",
	} {
		if !strings.Contains(launchedContext, want) {
			t.Fatalf("context_refs no conservan %q:\n%s", want, launchedContext)
		}
	}
	for _, forbidden := range []string{
		"/home/alberto/Trabajo/OPES",
		"/home-redacted/",
	} {
		if strings.Contains(launchedContext, forbidden) {
			t.Fatalf("context_refs exponen ruta local %q:\n%s", forbidden, launchedContext)
		}
	}
	criteria := strings.Join(launcher.specs[0].AcceptanceCriteria, "\n")
	if !strings.Contains(criteria, "Usar los input_fields inlineados") {
		t.Fatalf("acceptance no exige usar input_fields: %v", launcher.specs[0].AcceptanceCriteria)
	}
	state, err := goalStates.LoadGoalWorkStateV0(context.Background(), result.RunRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	persistedContext := codexStackGoalContextRefsTextForTestV0(state.Spec.ContextRefs)
	if persistedContext != launchedContext {
		t.Fatalf("context_refs persistidos difieren\nlaunched:\n%s\npersisted:\n%s", launchedContext, persistedContext)
	}
}

func TestCodexStackV0ExternalWorkRunGoalFirstPaqueteFinalIncluyeArtefactosAceptadosV0(t *testing.T) {
	stack, _, launcher := buildExternalWorkGoalFirstDomainDeliveryStackForTestV0(t)
	if err := stack.DomainDelivery.Ledger.RecordDomainWorkArtifactSubmissionV0(
		context.Background(),
		DomainWorkArtifactSubmissionRecordV0{
			IdempotencyKey: "idem-research-accepted",
			Status:         DomainWorkArtifactSubmissionStatusAcceptedV0,
			RunRef:         "run-research-001",
			CorrelationID:  "corr-external-work-stack-001",
			DomainRef:      "opes",
			JobRef:         "job-research-001",
			ArtifactRef:    "artifact-research-001",
			ArtifactType:   "exam_research_report",
			Summary:        "Investigacion de examenes aceptada por OPES temporal.",
			ReceiptRef:     "receipt-ref-research-001",
			PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
				{Name: "work_kind", Value: "research_exam_precedents"},
				{Name: "body", Value: "Resumen compacto de fuentes."},
			},
		},
	); err != nil {
		t.Fatalf("Record research: %v", err)
	}
	if err := stack.DomainDelivery.Ledger.RecordDomainWorkArtifactSubmissionV0(
		context.Background(),
		DomainWorkArtifactSubmissionRecordV0{
			IdempotencyKey: "idem-html-accepted",
			Status:         DomainWorkArtifactSubmissionStatusAcceptedV0,
			RunRef:         "run-html-001",
			CorrelationID:  "corr-external-work-stack-001",
			DomainRef:      "opes",
			JobRef:         "job-html-001",
			ArtifactRef:    "artifact-html-001",
			ArtifactType:   "local_html_site",
			Summary:        "HTML local aceptado por OPES temporal.",
			ReceiptRef:     "receipt-ref-html-001",
			PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
				{Name: "work_kind", Value: "generate_html_site"},
				{Name: "site_ref", Value: "site-ref-001"},
			},
		},
	); err != nil {
		t.Fatalf("Record html: %v", err)
	}
	if err := stack.DomainDelivery.Ledger.RecordDomainWorkArtifactSubmissionV0(
		context.Background(),
		DomainWorkArtifactSubmissionRecordV0{
			IdempotencyKey: "idem-other-correlation",
			Status:         DomainWorkArtifactSubmissionStatusAcceptedV0,
			RunRef:         "run-other-001",
			CorrelationID:  "corr-otra",
			DomainRef:      "opes",
			JobRef:         "job-other-001",
			ArtifactType:   "audio_asset",
			ReceiptRef:     "receipt-ref-other-001",
			PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
				{Name: "work_kind", Value: "generate_audio_asset"},
			},
		},
	); err != nil {
		t.Fatalf("Record other: %v", err)
	}
	change := defaultExternalWorkRunChangeForTestV0()
	change.ChangeRef = "opes-job-final-001"
	change.ExternalWork.JobRef = "job-final-001"
	change.ExternalWork.WorkKind = "finalize_temario_package"
	change.ExternalWork.InputFields = []orquestadomainwork.DomainWorkFieldV0{
		{Name: "topic_id", Value: "topic-ref-001"},
		{Name: "expected_artifact_type", Value: "completed_syllabus_package"},
	}

	result := postExternalWorkRunStackWithChangeV0(t, stack, change)

	if result.GoalRef == "" || len(launcher.specs) != 1 {
		t.Fatalf("goal-first no lanzo spec final: result=%+v specs=%+v", result, launcher.specs)
	}
	launchedContext := codexStackGoalContextRefsTextForTestV0(launcher.specs[0].ContextRefs)
	for _, want := range []string{
		"accepted_domain_artifact_manifest",
		"count=2",
		"accepted_domain_artifact",
		"artifact_type=exam_research_report",
		"artifact_dir=external/opes/research_exam_precedents/job-research-001",
		"path_candidates=external/opes/research_exam_precedents/job-research-001/exam_research_report.json",
		"receipt_ref=receipt-ref-research-001",
		"artifact_type=local_html_site",
		"artifact_dir=external/opes/generate_html_site/job-html-001",
		"receipt_ref=receipt-ref-html-001",
	} {
		if !strings.Contains(launchedContext, want) {
			t.Fatalf("context_refs final no contienen %q:\n%s", want, launchedContext)
		}
	}
	for _, forbidden := range []string{
		"receipt-ref-other-001",
	} {
		if strings.Contains(launchedContext, forbidden) {
			t.Fatalf("context_refs final incluyen %q:\n%s", forbidden, launchedContext)
		}
	}
	criteria := strings.Join(launcher.specs[0].AcceptanceCriteria, "\n")
	if !strings.Contains(criteria, "accepted_domain_artifact_manifest") ||
		!codexStackStringInSetForTestV0(
			launcher.specs[0].EvidenceRefs,
			externalWorkGoalFirstAcceptedArtifactContextEvidenceV0,
		) {
		t.Fatalf("spec final sin criterio/evidencia de artefactos aceptados: %+v", launcher.specs[0])
	}
}

func TestCodexStackV0ExternalWorkRunConObservadorResidenteNoMarcaObserverRequired(t *testing.T) {
	launcher := &goalFirstQueueLauncherForTestV0{}
	goalStates := newGoalFirstQueueStateStoreForTestV0()
	config := codexStackBaseConfigForTestV0(t, newFakeCodexStackRuntimeV0(), nil, nil)
	config.AppGoalLauncher = launcher
	config.AppGoalObserver = &goalFirstQueueObserverForTestV0{}
	config.AppGoalClosureValidator = orquestagoal.DefaultGoalWorkClosureValidatorV0{}
	config.Stores.AppGoalStateStore = goalStates
	config.GoalObserverResidentEnabled = true
	stack, err := BuildStackV0(config)
	if err != nil {
		t.Fatalf("BuildStackV0: %v", err)
	}

	result := postExternalWorkRunStackV0(t, stack)

	if result.Estado != orquestamcp.MCPExternalWorkRunEstadoOKV0 ||
		result.GoalRef == "" ||
		codexStackStringInSetForTestV0(result.NextActions, orquestamcp.MCPExternalWorkRunNextActionObserverRequiredV0) ||
		codexStackStringInSetForTestV0(result.NextActions, orquestamcp.MCPExternalWorkRunNextActionObserveGoalV0) ||
		!codexStackStringInSetForTestV0(result.NextActions, orquestamcp.MCPExternalWorkRunNextActionObserveActiveGoalsV0) ||
		codexStackStringInSetForTestV0(result.EvidenceRefs, "evidence-ref-external-work-goal-first-observer-required") ||
		!codexStackStringInSetForTestV0(result.EvidenceRefs, "evidence-ref-external-work-goal-first-resident-observer") {
		t.Fatalf("result goal-first residente inesperado=%+v", result)
	}
}

func TestCodexStackV0ExternalWorkRunResidenteSinListerMarcaObserverRequired(t *testing.T) {
	launcher := &goalFirstQueueLauncherForTestV0{}
	config := codexStackBaseConfigForTestV0(t, newFakeCodexStackRuntimeV0(), nil, nil)
	config.AppGoalLauncher = launcher
	config.AppGoalObserver = &goalFirstQueueObserverForTestV0{}
	config.AppGoalClosureValidator = orquestagoal.DefaultGoalWorkClosureValidatorV0{}
	config.Stores.AppGoalStateStore = codexStackGoalStateStoreForTestV0{}
	config.GoalObserverResidentEnabled = true
	stack, err := BuildStackV0(config)
	if err != nil {
		t.Fatalf("BuildStackV0: %v", err)
	}

	result := postExternalWorkRunStackV0(t, stack)

	if result.Estado != orquestamcp.MCPExternalWorkRunEstadoOKV0 ||
		result.GoalRef == "" ||
		!codexStackStringInSetForTestV0(result.NextActions, orquestamcp.MCPExternalWorkRunNextActionObserverRequiredV0) ||
		!codexStackStringInSetForTestV0(result.NextActions, orquestamcp.MCPExternalWorkRunNextActionObserveGoalV0) ||
		codexStackStringInSetForTestV0(result.NextActions, orquestamcp.MCPExternalWorkRunNextActionObserveActiveGoalsV0) ||
		!codexStackStringInSetForTestV0(result.EvidenceRefs, "evidence-ref-external-work-goal-first-observer-required") ||
		codexStackStringInSetForTestV0(result.EvidenceRefs, "evidence-ref-external-work-goal-first-resident-observer") {
		t.Fatalf("result goal-first sin lister inesperado=%+v", result)
	}
}

func TestCodexStackV0ExternalWorkRunGoalFirstLaunchFailedPublicaReasonCode(t *testing.T) {
	launcher := &goalFirstQueueLauncherForTestV0{
		receipt: orquestagoal.GoalLaunchReceiptV0{
			SchemaVersion: orquestagoal.GoalWorkLaunchReceiptSchemaV0,
			Status:        orquestagoal.GoalStatusInvalidV0,
			EvidenceRefs:  []string{"evidence-ref-goal-launch-socket-missing"},
			Issues: []orquestagoal.GoalWorkIssueV0{{
				Code: "codex_app_server_control_socket_missing",
			}},
		},
		err: errors.New("failed to connect to socket at /home/user/.codex/app-server-control/app-server-control.sock"),
	}
	config := codexStackBaseConfigForTestV0(t, newFakeCodexStackRuntimeV0(), nil, nil)
	config.AppGoalLauncher = launcher
	config.AppGoalObserver = &goalFirstQueueObserverForTestV0{}
	config.AppGoalClosureValidator = orquestagoal.DefaultGoalWorkClosureValidatorV0{}
	config.Stores.AppGoalStateStore = newGoalFirstQueueStateStoreForTestV0()
	config.GoalObserverResidentEnabled = true
	stack, err := BuildStackV0(config)
	if err != nil {
		t.Fatalf("BuildStackV0: %v", err)
	}

	result := postExternalWorkRunStackRawV0(t, stack, http.StatusBadRequest, defaultExternalWorkRunChangeForTestV0())

	if result.Estado != orquestamcp.MCPExternalWorkRunEstadoErrorV0 ||
		result.RoutePolicy != orquestamcp.MCPExternalWorkRunRoutePolicyGoalFirstV0 ||
		result.DirectorExecutionMode != orquestamcp.MCPExternalWorkRunDirectorExecutionModeGoalFirstV0 ||
		result.GoalRef == "" ||
		!codexStackStringInSetForTestV0(result.EvidenceRefs, "evidence-ref-external-work-goal-launch-failed") ||
		!codexStackStringInSetForTestV0(result.EvidenceRefs, "evidence-ref-goal-launch-socket-missing") ||
		!codexStackStringInSetForTestV0(result.NextActions, orquestamcp.MCPExternalWorkRunNextActionConfigureGoalBackendV0) ||
		!codexStackStringInSetForTestV0(result.NextActions, orquestamcp.MCPExternalWorkRunNextActionDoNotFallbackLegacyV0) ||
		!codexStackStringInSetForTestV0(result.NextActions, orquestamcp.MCPExternalWorkRunNextActionObserveGoalV0) ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != "external_work_goal_launch_failed" ||
		result.Errores[0].Field != "goal_launcher" ||
		result.Errores[0].Message != "codex_app_server_control_socket_missing" {
		t.Fatalf("result=%+v", result)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Marshal result: %v", err)
	}
	if strings.Contains(string(encoded), "/home/user") ||
		strings.Contains(string(encoded), "app-server-control.sock") {
		t.Fatalf("diagnostico publico filtro ruta local: %s", string(encoded))
	}
	state, err := config.Stores.AppGoalStateStore.LoadGoalWorkStateV0(context.Background(), result.RunRef)
	if err != nil {
		t.Fatalf("fallo de launch debe dejar GoalWorkStateV0 reparable: %v", err)
	}
	if state.Status != orquestagoal.GoalStatusInvalidV0 ||
		state.GoalRef != result.GoalRef ||
		len(state.LaunchReceipt.Issues) != 1 ||
		state.LaunchReceipt.Issues[0].Code != "codex_app_server_control_socket_missing" ||
		!codexStackStringInSetForTestV0(state.EvidenceRefs, "evidence-ref-external-work-goal-first-launch-failed-state-v0") {
		t.Fatalf("state reparable inesperado=%+v result=%+v", state, result)
	}
}

func TestCodexStackV0ExternalWorkGoalFirstBloqueaReceiptInventadoSinLedger(t *testing.T) {
	stack, observer, launcher := buildExternalWorkGoalFirstDomainDeliveryStackForTestV0(t)
	started := postExternalWorkRunStackV0(t, stack)
	spec := launcher.specs[0]
	observer.result = externalWorkGoalFirstCompleteResultForTestV0(
		spec,
		started.ExternalGoalRef,
		"receipt-ref-goal-first-inventado-001",
	)

	result, err := stack.ObserveAppDirectorGoalV0(
		context.Background(),
		orquestaappdirectorservice.ObserveAppDirectorGoalRequestV0{
			RunRef:        started.RunRef,
			CorrelationID: "corr-external-work-goal-first-fake-receipt-001",
			RequestedBy:   "orquesta-app-codex-stack-test",
		},
	)
	if err != nil {
		t.Fatalf("ObserveAppDirectorGoalV0: %v", err)
	}
	if result.Run.Status != orquestacoreworkflow.OrchestrationRunStatusBlockedV0 ||
		result.Closure.Accepted ||
		!result.Closure.NeedsRework ||
		!goalClosureHasIssueForTestV0(result.Closure, goalDomainReceiptLedgerAcceptedMissingIssueV0) {
		t.Fatalf("receipt inventado no bloqueo cierre: result=%+v", result)
	}
}

func TestCodexStackV0ExternalWorkGoalFirstBloqueaReceiptSiDomainDeliveryDesactivadoV0(t *testing.T) {
	launcher := &goalFirstQueueLauncherForTestV0{}
	observer := &goalFirstQueueObserverForTestV0{}
	config := codexStackBaseConfigForTestV0(
		t,
		newFakeCodexStackRuntimeV0(),
		nil,
		nil,
	)
	config.AppGoalLauncher = launcher
	config.AppGoalObserver = observer
	config.AppGoalClosureValidator = orquestagoal.DefaultGoalWorkClosureValidatorV0{}
	config.Stores.AppGoalStateStore = newGoalFirstQueueStateStoreForTestV0()
	stack, err := BuildStackV0(config)
	if err != nil {
		t.Fatalf("BuildStackV0: %v", err)
	}
	started := postExternalWorkRunStackV0(t, stack)
	spec := launcher.specs[0]
	observer.result = externalWorkGoalFirstCompleteResultForTestV0(
		spec,
		started.ExternalGoalRef,
		"receipt-ref-goal-first-sin-domain-delivery-001",
	)

	result, err := stack.ObserveAppDirectorGoalV0(
		context.Background(),
		orquestaappdirectorservice.ObserveAppDirectorGoalRequestV0{
			RunRef:        started.RunRef,
			CorrelationID: "corr-external-work-goal-first-no-domain-delivery-001",
			RequestedBy:   "orquesta-app-codex-stack-test",
		},
	)
	if err != nil {
		t.Fatalf("ObserveAppDirectorGoalV0: %v", err)
	}
	if result.Run.Status != orquestacoreworkflow.OrchestrationRunStatusBlockedV0 ||
		result.Closure.Accepted ||
		!result.Closure.NeedsRework ||
		!goalClosureHasIssueForTestV0(result.Closure, goalDomainReceiptLedgerAcceptedMissingIssueV0) {
		t.Fatalf("receipt sin DomainDelivery no bloqueo cierre: result=%+v", result)
	}
}

func TestCodexStackV0ExternalWorkGoalFirstSubeArtifactDomainWorkYCierraV0(t *testing.T) {
	stack, observer, launcher, domainWork := buildExternalWorkGoalFirstDomainDeliveryStackWithExecutorForTestV0(t)
	started := postExternalWorkRunStackV0(t, stack)
	spec := launcher.specs[0]
	contract := spec.ArtifactContracts[0]
	writeGoalFirstDomainWorkArtifactForTestV0(t, stack.Codex.ProjectWorkDir, spec, contract.ArtifactType)
	receiptRef := "receipt-ref-goal-first-local-domain-001"
	observer.result = externalWorkGoalFirstCompleteResultForTestV0(
		spec,
		started.ExternalGoalRef,
		receiptRef,
	)

	result, err := stack.ObserveAppDirectorGoalV0(
		context.Background(),
		orquestaappdirectorservice.ObserveAppDirectorGoalRequestV0{
			RunRef:        started.RunRef,
			CorrelationID: "corr-external-work-goal-first-submit-domain-001",
			RequestedBy:   "orquesta-app-codex-stack-test",
		},
	)
	if err != nil {
		t.Fatalf("ObserveAppDirectorGoalV0: %v", err)
	}
	if result.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		!result.Closure.Accepted ||
		domainWork.called != 1 ||
		domainWork.input.Action != orquestamcp.MCPDomainWorkActionSubmitArtifactV0 {
		t.Fatalf("goal-first no subio artefacto y cerro: result=%+v domainWork=%+v", result, domainWork)
	}
	submission := domainWork.input.ArtifactSubmission
	if submission.JobRef != "job-ref-001" ||
		submission.ArtifactRef != contract.ArtifactRef ||
		submission.ArtifactType != contract.ArtifactType ||
		!submission.CompleteJob ||
		domainWorkFieldStringValueV0(submission.PayloadFields, "body") == "" {
		t.Fatalf("submission inesperada=%+v", submission)
	}
	records, err := stack.DomainDelivery.Ledger.(DomainWorkArtifactSubmissionRecordReaderPortV0).ListDomainWorkArtifactSubmissionsV0(
		context.Background(),
		DomainWorkArtifactSubmissionRecordFilterV0{
			RunRef: started.RunRef,
			Status: DomainWorkArtifactSubmissionStatusAcceptedV0,
		},
	)
	if err != nil || len(records) != 1 {
		t.Fatalf("records=%+v err=%v", records, err)
	}
	if records[0].ReceiptRef != receiptRef ||
		!codexStackStringInSetForTestV0(records[0].EvidenceRefs, "domain-work-goal-receipt-alias-"+safeDomainWorkEvidenceRefV0(receiptRef)) ||
		!codexStackStringInSetForTestV0(result.Closure.EvidenceRefs, goalDomainReceiptLedgerAcceptedEvidenceRefV0) {
		t.Fatalf("accepted record no enlaza receipt del goal: record=%+v closure=%+v", records[0], result.Closure)
	}
}

func TestCodexStackV0ExternalWorkGoalFirstNoCierraArtifactBloqueadoAunqueOPESAcepteReceiptV0(t *testing.T) {
	stack, observer, launcher, domainWork := buildExternalWorkGoalFirstDomainDeliveryStackWithExecutorForTestV0(t)
	started := postExternalWorkRunStackV0(t, stack)
	spec := launcher.specs[0]
	contract := spec.ArtifactContracts[0]
	writeGoalFirstDomainWorkBlockedArtifactForTestV0(t, stack.Codex.ProjectWorkDir, spec, contract.ArtifactType)
	receiptRef := "receipt-ref-goal-first-structured-blocker-001"
	observer.result = externalWorkGoalFirstCompleteResultForTestV0(
		spec,
		started.ExternalGoalRef,
		receiptRef,
	)

	result, err := stack.ObserveAppDirectorGoalV0(
		context.Background(),
		orquestaappdirectorservice.ObserveAppDirectorGoalRequestV0{
			RunRef:        started.RunRef,
			CorrelationID: "corr-external-work-goal-first-structured-blocker-001",
			RequestedBy:   "orquesta-app-codex-stack-test",
		},
	)
	if err != nil {
		t.Fatalf("ObserveAppDirectorGoalV0: %v", err)
	}
	if result.Run.Status != orquestacoreworkflow.OrchestrationRunStatusBlockedV0 ||
		result.Closure.Accepted ||
		!result.Closure.NeedsRework ||
		!goalClosureHasIssueForTestV0(result.Closure, goalDomainReceiptStructuredArtifactIssueCodeV0) {
		t.Fatalf("artifact con bloqueo estructurado cerro goal-first: result=%+v domainWork=%+v", result, domainWork)
	}
	if domainWork.called != 1 ||
		domainWork.input.Action != orquestamcp.MCPDomainWorkActionSubmitArtifactV0 ||
		domainWork.input.ArtifactSubmission.CompleteJob {
		t.Fatalf("submission bloqueada debe subirse como incompleta: domainWork=%+v", domainWork)
	}
	records, err := stack.DomainDelivery.Ledger.(DomainWorkArtifactSubmissionRecordReaderPortV0).ListDomainWorkArtifactSubmissionsV0(
		context.Background(),
		DomainWorkArtifactSubmissionRecordFilterV0{
			RunRef: started.RunRef,
			Status: DomainWorkArtifactSubmissionStatusAcceptedV0,
		},
	)
	if err != nil || len(records) != 1 {
		t.Fatalf("records=%+v err=%v", records, err)
	}
	if records[0].CompleteJob ||
		!domainWorkFieldValuesForTestV0(records[0].PayloadFields, "validation_issue_refs", []string{domainWorkStructuredNonTerminalArtifactIssueCodeV0}) {
		t.Fatalf("record bloqueado debe conservar evidencia estructurada: record=%+v", records[0])
	}
}

func TestCodexStackV0ExternalWorkGoalFirstDerivaReceiptDesdeLedgerSiGoalNoLoDeclaraV0(t *testing.T) {
	stack, observer, launcher, domainWork := buildExternalWorkGoalFirstDomainDeliveryStackWithExecutorForTestV0(t)
	started := postExternalWorkRunStackV0(t, stack)
	spec := launcher.specs[0]
	contract := spec.ArtifactContracts[0]
	writeGoalFirstDomainWorkArtifactForTestV0(t, stack.Codex.ProjectWorkDir, spec, contract.ArtifactType)
	observer.result = externalWorkGoalFirstCompleteResultForTestV0(
		spec,
		started.ExternalGoalRef,
	)

	result, err := stack.ObserveAppDirectorGoalV0(
		context.Background(),
		orquestaappdirectorservice.ObserveAppDirectorGoalRequestV0{
			RunRef:        started.RunRef,
			CorrelationID: "corr-external-work-goal-first-derived-receipt-001",
			RequestedBy:   "orquesta-app-codex-stack-test",
		},
	)
	if err != nil {
		t.Fatalf("ObserveAppDirectorGoalV0: %v", err)
	}
	if result.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		!result.Closure.Accepted ||
		domainWork.called != 1 ||
		domainWork.input.Action != orquestamcp.MCPDomainWorkActionSubmitArtifactV0 {
		t.Fatalf("goal-first sin DomainReceiptRefs no subio artefacto y cerro: result=%+v domainWork=%+v", result, domainWork)
	}
	derivedReceipt := "receipt-ref-" + contract.ArtifactRef
	records, err := stack.DomainDelivery.Ledger.(DomainWorkArtifactSubmissionRecordReaderPortV0).ListDomainWorkArtifactSubmissionsV0(
		context.Background(),
		DomainWorkArtifactSubmissionRecordFilterV0{
			RunRef: started.RunRef,
			Status: DomainWorkArtifactSubmissionStatusAcceptedV0,
		},
	)
	if err != nil || len(records) != 1 {
		t.Fatalf("records=%+v err=%v", records, err)
	}
	if records[0].ReceiptRef != derivedReceipt ||
		!codexStackStringInSetForTestV0(result.Closure.EvidenceRefs, "domain-work-goal-receipt-derived-"+safeDomainWorkEvidenceRefV0(derivedReceipt)) ||
		!codexStackStringInSetForTestV0(result.Closure.EvidenceRefs, goalDomainReceiptLedgerAcceptedEvidenceRefV0) {
		t.Fatalf("receipt derivado no queda evidenciado: record=%+v closure=%+v", records[0], result.Closure)
	}
}

func TestCodexStackV0ExternalWorkGoalFirstReintentaArtifactTrasRechazoHTTP400V0(t *testing.T) {
	stack, observer, launcher, domainWork := buildExternalWorkGoalFirstDomainDeliveryStackWithExecutorForTestV0(t)
	started := postExternalWorkRunStackV0(t, stack)
	spec := launcher.specs[0]
	contract := spec.ArtifactContracts[0]
	writeGoalFirstDomainWorkArtifactForTestV0(t, stack.Codex.ProjectWorkDir, spec, contract.ArtifactType)
	rejected := externalWorkGoalFirstRejectedReceiptRecordForTestV0(
		started.RunRef,
		spec,
		[]string{"domain-work-submit-artifact-rejected", "opes_http_status_400"},
	)
	if err := stack.DomainDelivery.Ledger.RecordDomainWorkArtifactSubmissionV0(
		context.Background(),
		rejected,
	); err != nil {
		t.Fatalf("RecordDomainWorkArtifactSubmissionV0 rejected: %v", err)
	}
	observer.result = externalWorkGoalFirstCompleteResultForTestV0(
		spec,
		started.ExternalGoalRef,
	)

	result, err := stack.ObserveAppDirectorGoalV0(
		context.Background(),
		orquestaappdirectorservice.ObserveAppDirectorGoalRequestV0{
			RunRef:        started.RunRef,
			CorrelationID: "corr-external-work-goal-first-retry-http-400-001",
			RequestedBy:   "orquesta-app-codex-stack-test",
		},
	)
	if err != nil {
		t.Fatalf("ObserveAppDirectorGoalV0: %v", err)
	}
	if result.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		!result.Closure.Accepted ||
		domainWork.called != 1 ||
		domainWork.input.Action != orquestamcp.MCPDomainWorkActionSubmitArtifactV0 ||
		!codexStackStringInSetForTestV0(result.Closure.EvidenceRefs, goalDomainReceiptLedgerAcceptedEvidenceRefV0) {
		t.Fatalf("goal-first no reintento tras 400 y cerro: result=%+v domainWork=%+v", result, domainWork)
	}
	records, err := stack.DomainDelivery.Ledger.(DomainWorkArtifactSubmissionRecordReaderPortV0).ListDomainWorkArtifactSubmissionsV0(
		context.Background(),
		DomainWorkArtifactSubmissionRecordFilterV0{
			RunRef: started.RunRef,
			Status: DomainWorkArtifactSubmissionStatusAcceptedV0,
		},
	)
	if err != nil || len(records) != 1 {
		t.Fatalf("accepted records=%+v err=%v", records, err)
	}
	if records[0].IdempotencyKey != rejected.IdempotencyKey ||
		records[0].ReceiptRef == "" ||
		records[0].ArtifactRef != contract.ArtifactRef {
		t.Fatalf("record aceptado inesperado=%+v rejected=%+v", records[0], rejected)
	}
}

func TestCodexStackV0ExternalWorkGoalFirstReconciliaArtifactRefRecuperableV0(t *testing.T) {
	stack, observer, launcher, domainWork := buildExternalWorkGoalFirstDomainDeliveryStackWithExecutorForTestV0(t)
	started := postExternalWorkRunStackV0(t, stack)
	spec := launcher.specs[0]
	contract := spec.ArtifactContracts[0]
	writeGoalFirstDomainWorkArtifactForTestV0(t, stack.Codex.ProjectWorkDir, spec, contract.ArtifactType)
	result := externalWorkGoalFirstCompleteResultForTestV0(
		spec,
		started.ExternalGoalRef,
	)
	result.ArtifactRefs = []string{contract.ArtifactRef + "-alias-recuperable"}
	observer.result = result

	observed, err := stack.ObserveAppDirectorGoalV0(
		context.Background(),
		orquestaappdirectorservice.ObserveAppDirectorGoalRequestV0{
			RunRef:        started.RunRef,
			CorrelationID: "corr-external-work-goal-first-artifact-ref-recover-001",
			RequestedBy:   "orquesta-app-codex-stack-test",
		},
	)
	if err != nil {
		t.Fatalf("ObserveAppDirectorGoalV0: %v", err)
	}
	if observed.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		!observed.Closure.Accepted ||
		domainWork.called != 1 ||
		domainWork.input.Action != orquestamcp.MCPDomainWorkActionSubmitArtifactV0 ||
		domainWork.input.ArtifactSubmission.ArtifactRef != contract.ArtifactRef ||
		!codexStackStringInSetForTestV0(observed.Closure.EvidenceRefs, goalDomainReceiptLedgerAcceptedEvidenceRefV0) {
		t.Fatalf("goal-first no reconcilio artifact_ref recuperable: result=%+v domainWork=%+v", observed, domainWork)
	}
	records, err := stack.DomainDelivery.Ledger.(DomainWorkArtifactSubmissionRecordReaderPortV0).ListDomainWorkArtifactSubmissionsV0(
		context.Background(),
		DomainWorkArtifactSubmissionRecordFilterV0{
			RunRef: started.RunRef,
			Status: DomainWorkArtifactSubmissionStatusAcceptedV0,
		},
	)
	if err != nil || len(records) != 1 {
		t.Fatalf("records=%+v err=%v", records, err)
	}
	if records[0].ArtifactRef != contract.ArtifactRef ||
		records[0].ArtifactType != contract.ArtifactType {
		t.Fatalf("record aceptado inesperado=%+v contract=%+v", records[0], contract)
	}
}

func TestCodexStackV0ExternalWorkGoalFirstSubeArtifactSiRequiredTestDominioBloqueadoV0(t *testing.T) {
	stack, observer, launcher, domainWork := buildExternalWorkGoalFirstDomainDeliveryStackWithExecutorForTestV0(t)
	change := defaultExternalWorkRunChangeForTestV0()
	change.ExternalWork.RequiredTests = []orquestadomainwork.DomainWorkRequiredTestV0{{
		TestRef: "opes-domain-test-draft-content-block-job-ref-001",
		AcceptanceCriteria: []string{
			"OPES acepta el artefacto por contrato publico submit_artifact.",
			"El receipt OPES conserva job_ref, delivery_ref y trazabilidad causal.",
		},
		EvidenceRefs: []string{"opes-job-job-ref-001"},
	}}
	started := postExternalWorkRunStackWithChangeV0(t, stack, change)
	spec := launcher.specs[0]
	contract := spec.ArtifactContracts[0]
	if len(spec.RequiredTests) != 1 ||
		spec.RequiredTests[0].Command != "" ||
		spec.RequiredTests[0].CommandRef != "" {
		t.Fatalf("required test de dominio inesperado=%+v", spec.RequiredTests)
	}
	writeGoalFirstDomainWorkArtifactForTestV0(t, stack.Codex.ProjectWorkDir, spec, contract.ArtifactType)
	receiptRef := "domain-receipt-blocked-public-v0"
	result := externalWorkGoalFirstCompleteResultForTestV0(
		spec,
		started.ExternalGoalRef,
		receiptRef,
	)
	result.RequiredTestResults = []orquestagoal.GoalRequiredTestResultV0{{
		TestRef:      spec.RequiredTests[0].TestRef,
		Status:       orquestagoal.GoalStatusBlockedV0,
		EvidenceRefs: []string{"evidence-ref-external-work-goal-spec-v0"},
	}}
	observer.result = result

	observed, err := stack.ObserveAppDirectorGoalV0(
		context.Background(),
		orquestaappdirectorservice.ObserveAppDirectorGoalRequestV0{
			RunRef:        started.RunRef,
			CorrelationID: "corr-external-work-goal-first-domain-test-blocked-001",
			RequestedBy:   "orquesta-app-codex-stack-test",
		},
	)
	if err != nil {
		t.Fatalf("ObserveAppDirectorGoalV0: %v", err)
	}
	if observed.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		!observed.Closure.Accepted ||
		domainWork.called != 1 ||
		domainWork.input.Action != orquestamcp.MCPDomainWorkActionSubmitArtifactV0 ||
		!codexStackStringInSetForTestV0(observed.Closure.EvidenceRefs, goalDomainReceiptLedgerAcceptedEvidenceRefV0) {
		t.Fatalf("goal-first no reconcilio required_test dominio por ledger: result=%+v domainWork=%+v", observed, domainWork)
	}
	records, err := stack.DomainDelivery.Ledger.(DomainWorkArtifactSubmissionRecordReaderPortV0).ListDomainWorkArtifactSubmissionsV0(
		context.Background(),
		DomainWorkArtifactSubmissionRecordFilterV0{
			RunRef: started.RunRef,
			Status: DomainWorkArtifactSubmissionStatusAcceptedV0,
		},
	)
	if err != nil || len(records) != 1 {
		t.Fatalf("records=%+v err=%v", records, err)
	}
	if records[0].ReceiptRef != receiptRef ||
		records[0].ArtifactRef != contract.ArtifactRef ||
		!records[0].CompleteJob {
		t.Fatalf("record ledger inesperado=%+v contract=%+v", records[0], contract)
	}
}

func TestCodexStackV0ExternalWorkGoalFirstCierraConReceiptAceptadoEnLedger(t *testing.T) {
	stack, observer, launcher := buildExternalWorkGoalFirstDomainDeliveryStackForTestV0(t)
	started := postExternalWorkRunStackV0(t, stack)
	spec := launcher.specs[0]
	receiptRef := "receipt-ref-goal-first-domain-accepted-001"
	if err := stack.DomainDelivery.Ledger.RecordDomainWorkArtifactSubmissionV0(
		context.Background(),
		externalWorkGoalFirstAcceptedReceiptRecordForTestV0(started.RunRef, spec, receiptRef),
	); err != nil {
		t.Fatalf("RecordDomainWorkArtifactSubmissionV0: %v", err)
	}
	observer.result = externalWorkGoalFirstCompleteResultForTestV0(
		spec,
		started.ExternalGoalRef,
		receiptRef,
	)

	result, err := stack.ObserveAppDirectorGoalV0(
		context.Background(),
		orquestaappdirectorservice.ObserveAppDirectorGoalRequestV0{
			RunRef:        started.RunRef,
			CorrelationID: "corr-external-work-goal-first-ledger-receipt-001",
			RequestedBy:   "orquesta-app-codex-stack-test",
		},
	)
	if err != nil {
		t.Fatalf("ObserveAppDirectorGoalV0: %v", err)
	}
	if result.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		!result.Closure.Accepted ||
		!codexStackStringInSetForTestV0(result.Closure.EvidenceRefs, goalDomainReceiptLedgerAcceptedEvidenceRefV0) {
		t.Fatalf("receipt aceptado en ledger no cerro goal-first: result=%+v", result)
	}
}

func TestCodexStackV0ExternalWorkGoalFirstNoCierraReceiptDomainWorkIncompleteV0(t *testing.T) {
	stack, observer, launcher := buildExternalWorkGoalFirstDomainDeliveryStackForTestV0(t)
	started := postExternalWorkRunStackV0(t, stack)
	spec := launcher.specs[0]
	receiptRef := "receipt-ref-goal-first-domain-incomplete-001"
	record := externalWorkGoalFirstAcceptedReceiptRecordForTestV0(started.RunRef, spec, receiptRef)
	record.CompleteJob = false
	if err := stack.DomainDelivery.Ledger.RecordDomainWorkArtifactSubmissionV0(
		context.Background(),
		record,
	); err != nil {
		t.Fatalf("RecordDomainWorkArtifactSubmissionV0: %v", err)
	}
	observer.result = externalWorkGoalFirstCompleteResultForTestV0(
		spec,
		started.ExternalGoalRef,
		receiptRef,
	)

	result, err := stack.ObserveAppDirectorGoalV0(
		context.Background(),
		orquestaappdirectorservice.ObserveAppDirectorGoalRequestV0{
			RunRef:        started.RunRef,
			CorrelationID: "corr-external-work-goal-first-incomplete-receipt-001",
			RequestedBy:   "orquesta-app-codex-stack-test",
		},
	)
	if err != nil {
		t.Fatalf("ObserveAppDirectorGoalV0: %v", err)
	}
	if result.Run.Status != orquestacoreworkflow.OrchestrationRunStatusBlockedV0 ||
		result.Closure.Accepted ||
		!result.Closure.NeedsRework ||
		!goalClosureHasIssueForTestV0(result.Closure, goalDomainReceiptLedgerIncompleteArtifactV0) {
		t.Fatalf("receipt incompleto cerro goal-first: result=%+v", result)
	}
}

func TestCodexStackV0ExternalWorkGoalFirstNoCierraOPESFinalSinManifestV0(t *testing.T) {
	stack, observer, launcher := buildExternalWorkGoalFirstDomainDeliveryStackForTestV0(t)
	change := defaultExternalWorkRunChangeForTestV0()
	change.ChangeRef = "opes-job-final-goal-001"
	change.ExternalWork.ProjectRef = "opes"
	change.ExternalWork.JobRef = "job-final-goal-001"
	change.ExternalWork.WorkKind = "finalize_temario_package"
	change.ExternalWork.InputFields = []orquestadomainwork.DomainWorkFieldV0{
		{Name: "expected_artifact_type", Value: "completed_syllabus_package"},
	}
	started := postExternalWorkRunStackWithChangeV0(t, stack, change)
	spec := launcher.specs[0]
	receiptRef := "receipt-ref-goal-first-opes-final-incomplete-001"
	record := externalWorkGoalFirstAcceptedReceiptRecordForTestV0(started.RunRef, spec, receiptRef)
	record.DomainRef = "opes"
	record.JobRef = "job-final-goal-001"
	record.ArtifactType = orquestadomainwork.DomainWorkArtifactTypeFinalDomainPackageV0
	record.PayloadFields = append(record.PayloadFields,
		orquestadomainwork.DomainWorkFieldV0{Name: "source_work_kind", Value: "finalize_temario_package"},
		orquestadomainwork.DomainWorkFieldV0{Name: "package_ref", Value: "package-ref-final-incomplete-001"},
	)
	if err := stack.DomainDelivery.Ledger.RecordDomainWorkArtifactSubmissionV0(
		context.Background(),
		record,
	); err != nil {
		t.Fatalf("RecordDomainWorkArtifactSubmissionV0: %v", err)
	}
	observer.result = externalWorkGoalFirstCompleteResultForTestV0(
		spec,
		started.ExternalGoalRef,
		receiptRef,
	)

	result, err := stack.ObserveAppDirectorGoalV0(
		context.Background(),
		orquestaappdirectorservice.ObserveAppDirectorGoalRequestV0{
			RunRef:        started.RunRef,
			CorrelationID: "corr-external-work-goal-first-opes-final-incomplete-001",
			RequestedBy:   "orquesta-app-codex-stack-test",
		},
	)
	if err != nil {
		t.Fatalf("ObserveAppDirectorGoalV0: %v", err)
	}
	if result.Run.Status != orquestacoreworkflow.OrchestrationRunStatusBlockedV0 ||
		result.Closure.Accepted ||
		!result.Closure.NeedsRework ||
		!goalClosureHasIssueForTestV0(result.Closure, goalDomainReceiptOPESFinalPackageEvidenceIssueCodeV0) {
		t.Fatalf("paquete final OPES sin manifest cerro goal-first: result=%+v", result)
	}
}

func TestCodexStackV0ExternalWorkGoalFirstNoCierraReceiptConPayloadNoTerminalEnLedgerV0(t *testing.T) {
	stack, observer, launcher := buildExternalWorkGoalFirstDomainDeliveryStackForTestV0(t)
	started := postExternalWorkRunStackV0(t, stack)
	spec := launcher.specs[0]
	receiptRef := "receipt-ref-goal-first-domain-structured-blocked-001"
	record := externalWorkGoalFirstAcceptedReceiptRecordForTestV0(started.RunRef, spec, receiptRef)
	record.PayloadFields = append(record.PayloadFields,
		orquestadomainwork.DomainWorkFieldV0{Name: "ready", ValueJSON: []byte(`false`)},
		orquestadomainwork.DomainWorkFieldV0{Name: "public_blocker", ValueJSON: []byte(`{"blocked_step":"apply_topic_registry_update"}`)},
	)
	if err := stack.DomainDelivery.Ledger.RecordDomainWorkArtifactSubmissionV0(
		context.Background(),
		record,
	); err != nil {
		t.Fatalf("RecordDomainWorkArtifactSubmissionV0: %v", err)
	}
	observer.result = externalWorkGoalFirstCompleteResultForTestV0(
		spec,
		started.ExternalGoalRef,
		receiptRef,
	)

	result, err := stack.ObserveAppDirectorGoalV0(
		context.Background(),
		orquestaappdirectorservice.ObserveAppDirectorGoalRequestV0{
			RunRef:        started.RunRef,
			CorrelationID: "corr-external-work-goal-first-structured-ledger-001",
			RequestedBy:   "orquesta-app-codex-stack-test",
		},
	)
	if err != nil {
		t.Fatalf("ObserveAppDirectorGoalV0: %v", err)
	}
	if result.Run.Status != orquestacoreworkflow.OrchestrationRunStatusBlockedV0 ||
		result.Closure.Accepted ||
		!result.Closure.NeedsRework ||
		!goalClosureHasIssueForTestV0(result.Closure, goalDomainReceiptStructuredArtifactIssueCodeV0) {
		t.Fatalf("receipt con payload no terminal cerro goal-first: result=%+v", result)
	}
}

func TestCodexStackV0ExternalWorkGoalFirstNoCierraOPESSubrolesSinSeisEvidenciasV0(t *testing.T) {
	stack, observer, launcher := buildExternalWorkGoalFirstDomainDeliveryStackForTestV0(t)
	started := postExternalWorkRunStackWithChangeV0(t, stack, opesSubrolesExternalWorkChangeForTestV0())
	spec := launcher.specs[0]
	receiptRef := "receipt-ref-goal-first-opes-subroles-incomplete-001"
	record := externalWorkGoalFirstAcceptedReceiptRecordForTestV0(started.RunRef, spec, receiptRef)
	if err := stack.DomainDelivery.Ledger.RecordDomainWorkArtifactSubmissionV0(
		context.Background(),
		record,
	); err != nil {
		t.Fatalf("RecordDomainWorkArtifactSubmissionV0: %v", err)
	}
	observer.result = externalWorkGoalFirstCompleteResultForTestV0(
		spec,
		started.ExternalGoalRef,
		receiptRef,
	)

	result, err := stack.ObserveAppDirectorGoalV0(
		context.Background(),
		orquestaappdirectorservice.ObserveAppDirectorGoalRequestV0{
			RunRef:        started.RunRef,
			CorrelationID: "corr-external-work-goal-first-opes-subroles-missing-001",
			RequestedBy:   "orquesta-app-codex-stack-test",
		},
	)
	if err != nil {
		t.Fatalf("ObserveAppDirectorGoalV0: %v", err)
	}
	if result.Run.Status != orquestacoreworkflow.OrchestrationRunStatusBlockedV0 ||
		result.Closure.Accepted ||
		!result.Closure.NeedsRework ||
		!goalClosureHasIssueForTestV0(result.Closure, goalDomainReceiptOPESSubrolesEvidenceMissingIssueV0) {
		t.Fatalf("receipt OPES 1+6 sin seis evidencias cerro goal-first: result=%+v", result)
	}
}

func TestCodexStackV0ExternalWorkGoalFirstNoCierraOPESSubrolesJSONSinSeisEvidenciasV0(t *testing.T) {
	stack, observer, launcher := buildExternalWorkGoalFirstDomainDeliveryStackForTestV0(t)
	change := opesSubrolesExternalWorkChangeForTestV0()
	change.ChangeRef = "opes-job-goal-first-subroles-json-001"
	change.ExternalWork.InterfaceRefs = []string{"opes-rest-v0"}
	change.ExternalWork.InputFields = []orquestadomainwork.DomainWorkFieldV0{
		{Name: "topic_id", Value: "tema-022"},
		{Name: "subroles_required", ValueJSON: []byte(`6`)},
	}
	started := postExternalWorkRunStackWithChangeV0(t, stack, change)
	spec := launcher.specs[0]
	if strings.Contains(codexStackGoalContextRefsTextForTestV0(spec.ContextRefs), "opes.padre-tema-6-subroles.v1") {
		t.Fatalf("test debe depender de input_field subroles_required, no de domain_interface: refs=%v", spec.ContextRefs)
	}
	receiptRef := "receipt-ref-goal-first-opes-subroles-json-incomplete-001"
	record := externalWorkGoalFirstAcceptedReceiptRecordForTestV0(started.RunRef, spec, receiptRef)
	if err := stack.DomainDelivery.Ledger.RecordDomainWorkArtifactSubmissionV0(
		context.Background(),
		record,
	); err != nil {
		t.Fatalf("RecordDomainWorkArtifactSubmissionV0: %v", err)
	}
	observer.result = externalWorkGoalFirstCompleteResultForTestV0(
		spec,
		started.ExternalGoalRef,
		receiptRef,
	)

	result, err := stack.ObserveAppDirectorGoalV0(
		context.Background(),
		orquestaappdirectorservice.ObserveAppDirectorGoalRequestV0{
			RunRef:        started.RunRef,
			CorrelationID: "corr-external-work-goal-first-opes-subroles-json-missing-001",
			RequestedBy:   "orquesta-app-codex-stack-test",
		},
	)
	if err != nil {
		t.Fatalf("ObserveAppDirectorGoalV0: %v", err)
	}
	if result.Run.Status != orquestacoreworkflow.OrchestrationRunStatusBlockedV0 ||
		result.Closure.Accepted ||
		!result.Closure.NeedsRework ||
		!goalClosureHasIssueForTestV0(result.Closure, goalDomainReceiptOPESSubrolesEvidenceMissingIssueV0) {
		t.Fatalf("receipt OPES subroles_required JSON sin seis evidencias cerro goal-first: result=%+v", result)
	}
}

func TestCodexStackV0ExternalWorkGoalFirstNoCierraOPESSubrolesConSeisEvidenciasNominalesSiHayTaskRefsV0(t *testing.T) {
	stack, observer, launcher := buildExternalWorkGoalFirstDomainDeliveryStackForTestV0(t)
	change := opesSubrolesExternalWorkChangeForTestV0()
	change.ExternalWork.InputFields = append(change.ExternalWork.InputFields, goalDomainReceiptOPESSubroleTaskRefsFieldForTestV0())
	started := postExternalWorkRunStackWithChangeV0(t, stack, change)
	spec := launcher.specs[0]
	if got := goalDomainReceiptOPESSubroleExpectedTaskRefsV0(spec); len(got) != goalDomainReceiptOPESSubrolesRequiredCountV0 {
		t.Fatalf("spec no conserva task refs OPES esperadas: got=%v refs=%s", got, codexStackGoalContextRefsTextForTestV0(spec.ContextRefs))
	}
	receiptRef := "receipt-ref-goal-first-opes-subroles-nominal-001"
	record := externalWorkGoalFirstAcceptedReceiptRecordForTestV0(started.RunRef, spec, receiptRef)
	record.EvidenceRefs = append(record.EvidenceRefs, goalDomainReceiptOPESSubroleEvidenceRefsForTestV0()...)
	if err := stack.DomainDelivery.Ledger.RecordDomainWorkArtifactSubmissionV0(
		context.Background(),
		record,
	); err != nil {
		t.Fatalf("RecordDomainWorkArtifactSubmissionV0: %v", err)
	}
	observer.result = externalWorkGoalFirstCompleteResultForTestV0(
		spec,
		started.ExternalGoalRef,
		receiptRef,
	)

	result, err := stack.ObserveAppDirectorGoalV0(
		context.Background(),
		orquestaappdirectorservice.ObserveAppDirectorGoalRequestV0{
			RunRef:        started.RunRef,
			CorrelationID: "corr-external-work-goal-first-opes-subroles-nominal-001",
			RequestedBy:   "orquesta-app-codex-stack-test",
		},
	)
	if err != nil {
		t.Fatalf("ObserveAppDirectorGoalV0: %v", err)
	}
	if result.Run.Status != orquestacoreworkflow.OrchestrationRunStatusBlockedV0 ||
		result.Closure.Accepted ||
		!result.Closure.NeedsRework ||
		!goalClosureHasIssueForTestV0(result.Closure, goalDomainReceiptOPESSubrolesEvidenceMissingIssueV0) {
		t.Fatalf("receipt OPES 1+6 con seis evidencias nominales cerro goal-first sin task refs causales: result=%+v", result)
	}
}

func TestCodexStackV0ExternalWorkGoalFirstNoCierraOPESSubrolesSinTaskRefsCausalesV0(t *testing.T) {
	stack, observer, launcher := buildExternalWorkGoalFirstDomainDeliveryStackForTestV0(t)
	started := postExternalWorkRunStackWithChangeV0(t, stack, opesSubrolesExternalWorkChangeForTestV0())
	spec := launcher.specs[0]
	receiptRef := "receipt-ref-goal-first-opes-subroles-without-task-refs-001"
	record := externalWorkGoalFirstAcceptedReceiptRecordForTestV0(started.RunRef, spec, receiptRef)
	record.EvidenceRefs = append(record.EvidenceRefs, goalDomainReceiptOPESSubroleEvidenceRefsForTestV0()...)
	if err := stack.DomainDelivery.Ledger.RecordDomainWorkArtifactSubmissionV0(
		context.Background(),
		record,
	); err != nil {
		t.Fatalf("RecordDomainWorkArtifactSubmissionV0: %v", err)
	}
	observer.result = externalWorkGoalFirstCompleteResultForTestV0(
		spec,
		started.ExternalGoalRef,
		receiptRef,
	)

	result, err := stack.ObserveAppDirectorGoalV0(
		context.Background(),
		orquestaappdirectorservice.ObserveAppDirectorGoalRequestV0{
			RunRef:        started.RunRef,
			CorrelationID: "corr-external-work-goal-first-opes-subroles-without-task-refs-001",
			RequestedBy:   "orquesta-app-codex-stack-test",
		},
	)
	if err != nil {
		t.Fatalf("ObserveAppDirectorGoalV0: %v", err)
	}
	if result.Run.Status != orquestacoreworkflow.OrchestrationRunStatusBlockedV0 ||
		result.Closure.Accepted ||
		!result.Closure.NeedsRework ||
		!goalClosureHasIssueForTestV0(result.Closure, goalDomainReceiptOPESSubrolesEvidenceMissingIssueV0) {
		t.Fatalf("receipt OPES 1+6 con seis evidencias nominales cerro sin refs causales: result=%+v", result)
	}
}

func TestCodexStackV0ExternalWorkGoalFirstCierraOPESSubrolesConEvidenciasDeTaskRefsV0(t *testing.T) {
	stack, observer, launcher := buildExternalWorkGoalFirstDomainDeliveryStackForTestV0(t)
	change := opesSubrolesExternalWorkChangeForTestV0()
	change.ExternalWork.InputFields = append(change.ExternalWork.InputFields, goalDomainReceiptOPESSubroleTaskRefsFieldForTestV0())
	started := postExternalWorkRunStackWithChangeV0(t, stack, change)
	spec := launcher.specs[0]
	receiptRef := "receipt-ref-goal-first-opes-subroles-taskrefs-complete-001"
	record := externalWorkGoalFirstAcceptedReceiptRecordForTestV0(started.RunRef, spec, receiptRef)
	record.EvidenceRefs = append(record.EvidenceRefs, goalDomainReceiptOPESSubroleTaskRefEvidenceRefsForTestV0()...)
	if err := stack.DomainDelivery.Ledger.RecordDomainWorkArtifactSubmissionV0(
		context.Background(),
		record,
	); err != nil {
		t.Fatalf("RecordDomainWorkArtifactSubmissionV0: %v", err)
	}
	observer.result = externalWorkGoalFirstCompleteResultForTestV0(
		spec,
		started.ExternalGoalRef,
		receiptRef,
	)

	result, err := stack.ObserveAppDirectorGoalV0(
		context.Background(),
		orquestaappdirectorservice.ObserveAppDirectorGoalRequestV0{
			RunRef:        started.RunRef,
			CorrelationID: "corr-external-work-goal-first-opes-subroles-taskrefs-complete-001",
			RequestedBy:   "orquesta-app-codex-stack-test",
		},
	)
	if err != nil {
		t.Fatalf("ObserveAppDirectorGoalV0: %v", err)
	}
	if result.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		!result.Closure.Accepted ||
		!codexStackStringInSetForTestV0(result.Closure.EvidenceRefs, goalDomainReceiptOPESSubrolesAcceptedEvidenceRefV0) {
		t.Fatalf("receipt OPES 1+6 con evidencias de task refs no cerro goal-first: result=%+v", result)
	}
}

func TestCodexStackV0ExternalWorkGoalFirstSinStateNoDrenaLegacy(t *testing.T) {
	stack := mustBuildCodexStackWithGoalBackendForTestV0(
		t,
		newFakeCodexStackRuntimeV0(),
		&goalFirstQueueLauncherForTestV0{},
		&goalFirstQueueObserverForTestV0{},
		newGoalFirstQueueStateStoreForTestV0(),
	)
	run := orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:         "run-external-work-goal-missing-state-001",
		ProjectRef:    "opes",
		AppSpecRef:    "app-spec-external-work-opes",
		Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Phases:        orquestacoreworkflow.OrchestrationPhaseCatalogV0(),
	}
	if err := stack.Stores.RunStore.SaveRunV0(context.Background(), run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}

	supervisor, err := NewCodexStackRunSupervisorExecutorV0(&stack).Execute(context.Background(), orquestamcp.MCPRunSupervisorToolInputV0{
		RequestID:     "req-external-work-goal-missing-state-001",
		CorrelationID: "corr-external-work-goal-missing-state-001",
		RunRef:        run.RunID,
	})
	if err != nil {
		t.Fatalf("RunSupervisor Execute: %v", err)
	}
	if supervisor.Estado != orquestamcp.MCPRunSupervisorEstadoErrorV0 ||
		supervisor.StopReason != "goal_first_state_missing" ||
		!codexStackStringInSetForTestV0(supervisor.NextActions, "inspect_goal_state_store") {
		t.Fatalf("supervisor=%+v", supervisor)
	}
}

func TestCodexStackV0ExternalWorkGoalFirstMarcadoSinStateNoDrenaLegacyAunqueAppSpecNeutral(t *testing.T) {
	goalStates := newGoalFirstQueueStateStoreForTestV0()
	stack := mustBuildCodexStackWithGoalBackendForTestV0(
		t,
		newFakeCodexStackRuntimeV0(),
		&goalFirstQueueLauncherForTestV0{},
		&goalFirstQueueObserverForTestV0{},
		goalStates,
	)
	run := orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:         "run-external-work-goal-marker-missing-state-001",
		ProjectRef:    "opes",
		AppSpecRef:    "app-spec-neutral-domain-work-opes",
		Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Phases:        orquestacoreworkflow.OrchestrationPhaseCatalogV0(),
	}
	if err := stack.Stores.RunStore.SaveRunV0(context.Background(), run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	saveCodexStackGoalFirstRunMarkerForTestV0(t, context.Background(), goalStates, run.RunID)

	supervisor, err := NewCodexStackRunSupervisorExecutorV0(&stack).Execute(context.Background(), orquestamcp.MCPRunSupervisorToolInputV0{
		RequestID:     "req-external-work-goal-marker-missing-state-001",
		CorrelationID: "corr-external-work-goal-marker-missing-state-001",
		RunRef:        run.RunID,
	})
	if err != nil {
		t.Fatalf("RunSupervisor Execute: %v", err)
	}
	if supervisor.Estado != orquestamcp.MCPRunSupervisorEstadoErrorV0 ||
		supervisor.StopReason != "goal_first_state_missing" ||
		!codexStackStringInSetForTestV0(supervisor.EvidenceRefs, "evidence-ref-run-supervisor-goal-first-marker") ||
		!codexStackStringInSetForTestV0(supervisor.NextActions, "inspect_goal_state_store") {
		t.Fatalf("supervisor=%+v", supervisor)
	}
}

func TestCodexStackV0ExternalWorkRunAceptaContratoAmplioV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	body := bytes.NewBuffer(nil)
	err := json.NewEncoder(body).Encode(orquestamcp.MCPExternalWorkRunToolInputV0{
		RequestID:             "req-external-work-wide-001",
		CorrelationID:         "corr-external-work-wide-001",
		DirectorExecutionMode: orquestamcp.MCPExternalWorkRunDirectorExecutionModeLegacyLoopV0,
		AppChangeRequest: orquestaappchange.AppChangeRequestV0{
			ChangeRef:  "opes-job-job-ref-wide-001",
			AppRef:     "opes",
			UserIntent: "Resolver trabajo externo amplio de OPES.",
			AcceptanceCriteria: []string{
				"devolver artifact_type=topic_expansion_package",
				"payload_json valido y trazable",
				"sin placeholders",
				"sin leer internals de OPES",
				"entrega en fichero unico bajo allowed_write_set",
				"paquete apto para tema_grande con capitulos y bloques trazables",
				"conservar base para tema_mediano sin perder autores normativa ni procedimientos",
				"incluir resumen/memoria de repaso derivado del tema desarrollado",
				"incluir esquema de examen y plan de visuales cuando aporten valor",
			},
			ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
				ProjectRef: "opes",
				JobRef:     "job-ref-wide-001",
				WorkKind:   "expand_topic_from_summary",
				InputFields: []orquestadomainwork.DomainWorkFieldV0{{
					Name:  "expected_artifact",
					Value: "topic_expansion_package",
				}},
			},
		},
	})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/external-work/run", body)
	req.Header.Set("Content-Type", "application/json")
	stack.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("external work run status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPExternalWorkRunToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode external work run: %v", err)
	}
	if result.Estado != orquestamcp.MCPExternalWorkRunEstadoOKV0 {
		t.Fatalf("result=%+v", result)
	}
	run, err := stack.Stores.RunStore.LoadRunV0(context.Background(), result.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if len(run.DirectorQuestions) != 1 {
		t.Fatalf("director_questions=%v", run.DirectorQuestions)
	}
}

func TestCodexStackV0ExternalWorkRunSaneaPreguntaDirectorSinPerderContextoV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	body := bytes.NewBuffer(nil)
	err := json.NewEncoder(body).Encode(orquestamcp.MCPExternalWorkRunToolInputV0{
		RequestID:             "req-external-work-safe-director-001",
		CorrelationID:         "corr-external-work-safe-director-001",
		DirectorExecutionMode: orquestamcp.MCPExternalWorkRunDirectorExecutionModeLegacyLoopV0,
		AppChangeRequest: orquestaappchange.AppChangeRequestV0{
			ChangeRef:  "self-review-rework-001",
			AppRef:     "orquesta",
			UserIntent: "Corregir flujo Codex sin exponer HOME token runtime ni provider en outbox.",
			AcceptanceCriteria: []string{
				"El conector Codex queda probado sin filtrar token ni HOME.",
				"El adapter conserva hexagonalidad.",
			},
			AllowedWriteSet: []string{"modulos/orquesta-app-codex-stack"},
			RequiredTests:   []string{"go test -count=1 ./modulos/orquesta-app-codex-stack"},
			MetadataRefs:    []string{"rules-token-economy-high"},
			ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
				ProjectRef: "orquesta",
				JobRef:     "job-safe-director-001",
				WorkKind:   "self_programming",
				WorkRefs:   []string{"runtime-codex-review"},
				InputFields: []orquestadomainwork.DomainWorkFieldV0{{
					Name:  "context_profile",
					Value: "large",
				}},
			},
		},
	})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/external-work/run", body)
	req.Header.Set("Content-Type", "application/json")
	stack.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("external work run status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPExternalWorkRunToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode external work run: %v", err)
	}
	if result.Estado != orquestamcp.MCPExternalWorkRunEstadoOKV0 {
		t.Fatalf("result=%+v", result)
	}
	run, err := stack.Stores.RunStore.LoadRunV0(context.Background(), result.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if len(run.DirectorQuestions) != 1 {
		t.Fatalf("director_questions=%v", run.DirectorQuestions)
	}
}

func TestCodexStackV0ExternalWorkRunSelfProgrammingSupervisaSinBloqueoGoBootstrapV0(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	body := bytes.NewBuffer(nil)
	err := json.NewEncoder(body).Encode(orquestamcp.MCPExternalWorkRunToolInputV0{
		RequestID:             "req-self-programming-policy-001",
		CorrelationID:         "corr-self-programming-policy-001",
		DirectorExecutionMode: orquestamcp.MCPExternalWorkRunDirectorExecutionModeLegacyLoopV0,
		AppChangeRequest: orquestaappchange.AppChangeRequestV0{
			ChangeRef:  "self-programming-web-001",
			AppRef:     "orquesta",
			UserIntent: "Completar web de Orquesta reutilizando modulos existentes.",
			AcceptanceCriteria: []string{
				"UI operativa sin logica de orquestacion en web.",
				"Archivos manejables y tests focales.",
			},
			AllowedWriteSet: []string{
				"modulos/orquesta-web",
				"docs/runbooks/autoprogramacion_web_2026-05-23.md",
			},
			RequiredTests: []string{"go test -count=1 ./modulos/orquesta-web"},
			MetadataRefs: []string{
				"rules-hexagonal-pure",
				"rules-i18n",
				"rules-manageable-files",
			},
			ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
				ProjectRef: "orquesta",
				JobRef:     "job-self-programming-web-001",
				WorkKind:   "self_programming",
				WorkRefs:   []string{"web-cockpit"},
				InputFields: []orquestadomainwork.DomainWorkFieldV0{{
					Name:  "context_profile",
					Value: "large",
				}},
			},
		},
	})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/external-work/run", body)
	req.Header.Set("Content-Type", "application/json")
	stack.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("external work run status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPExternalWorkRunToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode external work run: %v", err)
	}
	if result.Estado != orquestamcp.MCPExternalWorkRunEstadoOKV0 {
		t.Fatalf("result=%+v", result)
	}

	drain, err := stack.DrainRunV0(context.Background(), DrainRunRequestV0{
		RunRef:               result.RunRef,
		CorrelationID:        "corr-self-programming-policy-drain-001",
		OccurredAt:           "2026-05-23T08:00:00Z",
		MaxBursts:            16,
		MaxStepsPerBurst:     8,
		MaxDispatchesPerWait: 8,
		MaxCommands:          32,
		MaxOutboxPerCycle:    8,
		MaxDecisionCycles:    4,
		MaxExternalWaits:     0,
	})
	if err != nil {
		t.Fatalf("DrainRunV0: %v drain=%+v", err, drain)
	}
	run, err := stack.Stores.RunStore.LoadRunV0(context.Background(), result.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if stringInSetV0(run.Blockers, "app-director-decision-director-decision-source-error-director-decision-source") {
		t.Fatalf("run bloqueado por source_error: %+v", run)
	}
	if len(run.Tasks) != 1 || len(run.StartedAgents) != 1 || runtime.launchCountV0() != 1 {
		t.Fatalf("run no lanzo agente: tasks=%v started=%v launches=%d blockers=%v", run.Tasks, run.StartedAgents, runtime.launchCountV0(), run.Blockers)
	}
}

func TestCodexStackV0ExternalWorkRunOPESSubrolesMaterializaPadreYSeisHijosV0(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	result := postExternalWorkRunStackWithChangeLegacyV0(t, stack, orquestaappchange.AppChangeRequestV0{
		ChangeRef:  "opes-job-tema-subroles-001",
		AppRef:     "opes",
		UserIntent: "Resolver tema OPES con padre y seis subroles reales.",
		AcceptanceCriteria: []string{
			"materializar padre y seis subagentes OPES",
			"conservar ACK causal por subrol",
		},
		ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
			ProjectRef:    "opes",
			JobRef:        "job-ref-opes-subroles-001",
			InterfaceRefs: []string{"opes-rest-v0", "opes.padre-tema-6-subroles.v1"},
			WorkKind:      "draft_content_block",
			WorkRefs:      []string{"curso-integrador-social-b", "tema-015"},
			InputFields: []orquestadomainwork.DomainWorkFieldV0{
				{Name: "topic_id", Value: "tema-015"},
				{Name: "subroles_required", Value: "6"},
			},
		},
	})

	drain, err := stack.DrainRunV0(context.Background(), DrainRunRequestV0{
		RunRef:               result.RunRef,
		CorrelationID:        "corr-opes-subroles-drain-001",
		OccurredAt:           "2026-06-26T00:10:00Z",
		MaxBursts:            32,
		MaxStepsPerBurst:     12,
		MaxDispatchesPerWait: 16,
		MaxCommands:          64,
		MaxOutboxPerCycle:    16,
		MaxDecisionCycles:    6,
		MaxExternalWaits:     0,
	})
	if err != nil {
		t.Fatalf("DrainRunV0: %v drain=%+v", err, drain)
	}
	run, err := stack.Stores.RunStore.LoadRunV0(context.Background(), result.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	tasks, err := stack.Stores.TaskStore.LoadWorkflowTasksV0(context.Background(), result.RunRef, run.Tasks)
	if err != nil {
		t.Fatalf("LoadWorkflowTasksV0: %v", err)
	}
	if len(run.Tasks) != 7 || len(tasks) != 7 {
		t.Fatalf("tasks run=%v loaded=%+v", run.Tasks, tasks)
	}
	parent, children := codexStackOPESSubroleTasksForTestV0(tasks)
	if parent.TaskID == "" ||
		parent.MaxChildAgents != 6 ||
		len(parent.ChildTaskRefs) != 6 ||
		len(parent.DependsOn) != 6 ||
		parent.CohortRef == "" ||
		parent.WaveRef == "" ||
		!codexStackStringInSetForTestV0(parent.WriteSet, "external/opes/draft_content_block") ||
		!codexStackStringInSetForTestV0(parent.WriteSet, "external/opes/job-ref-opes-subroles-001") ||
		!codexStackStringInSetForTestV0(parent.WriteSet, "external/opes/draft_content_block/coordinacion") ||
		!codexStackStringInSetForTestV0(parent.WriteSet, "external/opes/job-ref-opes-subroles-001/coordinacion") ||
		containsOPESSubroleWriteSetForTestV0(parent.WriteSet) {
		t.Fatalf("parent=%+v", parent)
	}
	if len(children) != 6 {
		t.Fatalf("children=%+v parent=%+v", children, parent)
	}
	seenChildren := map[string]bool{}
	for _, child := range children {
		seenChildren[child.TaskID] = true
		if child.ParentTaskRef != parent.TaskID ||
			child.DelegationDepth != 1 ||
			child.CohortRef != parent.CohortRef ||
			child.WaveRef != parent.WaveRef ||
			len(child.DependsOn) != 0 ||
			len(child.ChildTaskRefs) != 0 ||
			!containsOPESSubroleWriteSetForTestV0(child.WriteSet) {
			t.Fatalf("child=%+v parent=%+v", child, parent)
		}
	}
	for _, childRef := range parent.ChildTaskRefs {
		if !seenChildren[childRef] {
			t.Fatalf("child_ref %s no cargado: seen=%v", childRef, seenChildren)
		}
		if !codexStackStringInSetForTestV0(parent.DependsOn, childRef) {
			t.Fatalf("parent.depends_on no contiene child_ref %s: parent=%+v", childRef, parent)
		}
	}
	if len(run.StartedAgents) != 7 || runtime.launchCountV0() != 7 {
		t.Fatalf("subagentes no lanzados: started=%v launches=%d tasks=%v blockers=%v",
			run.StartedAgents,
			runtime.launchCountV0(),
			run.Tasks,
			run.Blockers,
		)
	}
}

func TestCodexStackV0ExternalWorkRunOPESSubrolesPadreConservaWriteSetProductoAutorizadoV0(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	result := postExternalWorkRunStackWithChangeLegacyV0(t, stack, orquestaappchange.AppChangeRequestV0{
		ChangeRef:       "opes-job-tema-subroles-producto-032",
		AppRef:          "opes",
		UserIntent:      "Resolver tema OPES con padre integrador y seis subroles reales.",
		AllowedWriteSet: []string{"temas/tema_032"},
		AcceptanceCriteria: []string{
			"materializar padre y seis subagentes OPES",
			"el padre conserva write-set de producto autorizado",
		},
		ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
			ProjectRef:    "opes",
			JobRef:        "job-ref-opes-subroles-producto-032",
			InterfaceRefs: []string{"opes-rest-v0", "opes.padre-tema-6-subroles.v1"},
			WorkKind:      "draft_content_block",
			WorkRefs:      []string{"curso-integrador-social-b", "tema-032"},
			InputFields: []orquestadomainwork.DomainWorkFieldV0{
				{Name: "topic_id", Value: "tema-032"},
				{Name: "subroles_required", Value: "6"},
			},
		},
	})

	drain, err := stack.DrainRunV0(context.Background(), DrainRunRequestV0{
		RunRef:               result.RunRef,
		CorrelationID:        "corr-opes-subroles-producto-drain-032",
		OccurredAt:           "2026-06-26T00:15:00Z",
		MaxBursts:            32,
		MaxStepsPerBurst:     12,
		MaxDispatchesPerWait: 16,
		MaxCommands:          64,
		MaxOutboxPerCycle:    16,
		MaxDecisionCycles:    6,
		MaxExternalWaits:     0,
	})
	if err != nil {
		t.Fatalf("DrainRunV0: %v drain=%+v", err, drain)
	}
	run, err := stack.Stores.RunStore.LoadRunV0(context.Background(), result.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	tasks, err := stack.Stores.TaskStore.LoadWorkflowTasksV0(context.Background(), result.RunRef, run.Tasks)
	if err != nil {
		t.Fatalf("LoadWorkflowTasksV0: %v", err)
	}
	parent, children := codexStackOPESSubroleTasksForTestV0(tasks)
	if parent.TaskID == "" ||
		!codexStackStringInSetForTestV0(parent.WriteSet, "temas/tema_032") ||
		!codexStackStringInSetForTestV0(parent.WriteSet, "temas/tema_032/coordinacion") ||
		containsOPESSubroleWriteSetForTestV0(parent.WriteSet) {
		t.Fatalf("parent=%+v", parent)
	}
	parentDescriptor := mustCodexStackDescriptorByTaskRefV0(t, stack, parent.TaskID)
	if !codexStackStringInSetForTestV0(parentDescriptor.Spec.AgentPacket.Task.WriteSet, "temas/tema_032") ||
		!codexStackStringInSetForTestV0(parentDescriptor.Spec.AgentPacket.Task.WriteSet, "temas/tema_032/coordinacion") ||
		containsOPESSubroleWriteSetForTestV0(parentDescriptor.Spec.AgentPacket.Task.WriteSet) {
		t.Fatalf("parent packet write_set=%+v parent=%+v", parentDescriptor.Spec.AgentPacket.Task.WriteSet, parent)
	}
	for _, child := range children {
		if child.ParentTaskRef != parent.TaskID ||
			codexStackStringInSetForTestV0(child.WriteSet, "temas/tema_032") ||
			!containsOPESSubroleWriteSetForTestV0(child.WriteSet) {
			t.Fatalf("child=%+v parent=%+v", child, parent)
		}
	}
	if len(children) != 6 || len(run.StartedAgents) != 7 || runtime.launchCountV0() != 7 {
		t.Fatalf("children=%d started=%v launches=%d", len(children), run.StartedAgents, runtime.launchCountV0())
	}
}

func TestCodexStackV0ExternalWorkRunOPESSubrolesSupervisorDirectoSinLimitesLanzaSeisYDejaColaRunningV0(t *testing.T) {
	runtime := newPendingAckCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	result := postExternalWorkRunStackWithChangeLegacyV0(t, stack, orquestaappchange.AppChangeRequestV0{
		ChangeRef:  "opes-job-tema-subroles-supervisor-001",
		AppRef:     "opes",
		UserIntent: "Resolver tema OPES con padre y seis subroles desde supervisor directo.",
		AcceptanceCriteria: []string{
			"materializar padre y seis subagentes OPES",
			"arrancar seis subroles sin limites expertos obligatorios",
		},
		ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
			ProjectRef:    "opes",
			JobRef:        "job-ref-opes-subroles-supervisor-001",
			InterfaceRefs: []string{"opes-rest-v0", "opes.padre-tema-6-subroles.v1"},
			WorkKind:      "draft_content_block",
			WorkRefs:      []string{"curso-integrador-social-b", "tema-021"},
			InputFields: []orquestadomainwork.DomainWorkFieldV0{
				{Name: "topic_id", Value: "tema-021"},
				{Name: "subroles_required", Value: "6"},
			},
		},
	})

	supervisor := postRunSupervisorStackV0(t, stack, legacyRunSupervisorInputForStackTestV0(orquestamcp.MCPRunSupervisorToolInputV0{
		RequestID:     "req-opes-subroles-supervisor-defaults-001",
		CorrelationID: "corr-opes-subroles-supervisor-defaults-001",
		RunRef:        result.RunRef,
	}))
	if supervisor.Estado != orquestamcp.MCPRunSupervisorEstadoOKV0 {
		t.Fatalf("supervisor=%+v", supervisor)
	}
	run, err := stack.Stores.RunStore.LoadRunV0(context.Background(), result.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if len(run.Tasks) != 7 || len(run.StartedAgents) != 6 || runtime.launchCountV0() != 6 {
		t.Fatalf("supervisor directo no lanzo seis subroles OPES: tasks=%v started=%v launches=%d supervisor=%+v",
			run.Tasks,
			run.StartedAgents,
			runtime.launchCountV0(),
			supervisor,
		)
	}
	candidates, err := stack.Stores.RunQueue.ListRunSchedulingCandidatesV0(
		context.Background(),
		orquestarunqueue.RunQueueReadRequestV0{
			QueueRef:             DefaultRunQueueRefV0,
			IncludeNonExecutable: true,
		},
	)
	if err != nil {
		t.Fatalf("ListRunSchedulingCandidatesV0: %v", err)
	}
	found := false
	for _, candidate := range candidates {
		if candidate.RunRef != result.RunRef {
			continue
		}
		found = true
		if candidate.Status != orquestarunqueue.RunStatusRunningV0 {
			t.Fatalf("supervisor directo dejo cola sin running: candidate=%+v started=%v delivered_agents=%v deliveries=%v delivered_tasks=%v closed_tasks=%v phase_artifacts=%v",
				candidate,
				run.StartedAgents,
				run.DeliveredAgents,
				run.Deliveries,
				run.DeliveredTasks,
				run.ClosedTasks,
				run.PhaseArtifacts,
			)
		}
	}
	if !found {
		t.Fatalf("supervisor directo no encontro candidato en cola: candidates=%+v", candidates)
	}
}

func TestCodexStackV0ExternalWorkRunSupervisorConsumeDeliverySinExpirarWaitV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	planStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0()
	stack.Ports.OperationalPlanStateStore = planStore
	stack.Ports.OperationalPlanStateWriter = planStore
	result := postExternalWorkRunStackLegacyV0(t, stack)

	drain, err := stack.DrainRunV0(context.Background(), DrainRunRequestV0{
		RunRef:               result.RunRef,
		CorrelationID:        "corr-external-work-supervisor-delivery-001",
		OccurredAt:           "2026-05-22T18:10:00Z",
		MaxBursts:            16,
		MaxStepsPerBurst:     8,
		MaxDispatchesPerWait: 8,
		MaxCommands:          32,
		MaxOutboxPerCycle:    8,
		MaxDecisionCycles:    4,
		MaxExternalWaits:     0,
	})
	if err != nil {
		t.Fatalf("DrainRunV0: %v drain=%+v", err, drain)
	}
	run, err := stack.Stores.RunStore.LoadRunV0(context.Background(), result.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if len(run.DeliveredAgents) != 1 || len(run.Deliveries) != 1 {
		t.Fatalf("run sin entrega: delivered=%v deliveries=%v", run.DeliveredAgents, run.Deliveries)
	}
	state, err := planStore.LoadOperationalDirectorPlanStateV0(
		context.Background(),
		result.RunRef,
		"operational-director-plan-director-decisions-"+appChangeSafeRefPartForTestV0(result.RunRef),
	)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	waitStep := codexStackExternalWorkPlanStepForTestV0(t, state, "step-wait-subagents")
	reviewStep := codexStackExternalWorkPlanStepForTestV0(t, state, "step-review-deliveries")
	if state.Status == orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		waitStep.Status == orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		waitStep.Reason == "external-wait-exhausted" {
		t.Fatalf("wait expiro tras delivery: state=%+v wait=%+v", state, waitStep)
	}
	if state.ActiveStepID == "step-wait-subagents" ||
		(reviewStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 &&
			reviewStep.Status != orquestadirectoroperativo.OperationalDirectorStepAcceptedV0) {
		t.Fatalf("state no avanzo a review: state=%+v review=%+v", state, reviewStep)
	}
}

func postExternalWorkRunStackV0(
	t *testing.T,
	stack StackV0,
) orquestamcp.MCPExternalWorkRunToolResultV0 {
	t.Helper()
	return postExternalWorkRunStackWithChangeV0(t, stack, defaultExternalWorkRunChangeForTestV0())
}

func postExternalWorkRunStackLegacyV0(
	t *testing.T,
	stack StackV0,
) orquestamcp.MCPExternalWorkRunToolResultV0 {
	t.Helper()
	return postExternalWorkRunStackWithChangeLegacyV0(t, stack, defaultExternalWorkRunChangeForTestV0())
}

func defaultExternalWorkRunChangeForTestV0() orquestaappchange.AppChangeRequestV0 {
	return orquestaappchange.AppChangeRequestV0{
		ChangeRef:  "opes-job-job-ref-001",
		AppRef:     "opes",
		UserIntent: "Resolver trabajo externo de OPES.",
		AcceptanceCriteria: []string{
			"markdown valido",
		},
		ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
			ProjectRef: "opes",
			JobRef:     "job-ref-001",
			WorkKind:   "draft_content_block",
			InputFields: []orquestadomainwork.DomainWorkFieldV0{{
				Name:  "topic_id",
				Value: "topic-ref-001",
			}},
		},
	}
}

func opesSubrolesExternalWorkChangeForTestV0() orquestaappchange.AppChangeRequestV0 {
	return orquestaappchange.AppChangeRequestV0{
		ChangeRef:  "opes-job-goal-first-subroles-001",
		AppRef:     "opes",
		UserIntent: "Resolver tema OPES con padre integrador y seis subroles reales.",
		AcceptanceCriteria: []string{
			"materializar seis subroles reales o bloquear con causa operativa",
			"entregar integracion canonica con evidencia de cada subrol",
		},
		ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
			ProjectRef:    "opes",
			JobRef:        "job-ref-goal-first-opes-subroles-001",
			InterfaceRefs: []string{"opes-rest-v0", "opes.padre-tema-6-subroles.v1"},
			WorkKind:      "draft_content_block",
			WorkRefs:      []string{"curso-integrador-social-b", "tema-022"},
			InputFields: []orquestadomainwork.DomainWorkFieldV0{
				{Name: "topic_id", Value: "tema-022"},
				{Name: "subroles_required", Value: "6"},
			},
		},
	}
}

func buildExternalWorkGoalFirstDomainDeliveryStackForTestV0(
	t *testing.T,
) (StackV0, *goalFirstQueueObserverForTestV0, *goalFirstQueueLauncherForTestV0) {
	t.Helper()
	stack, observer, launcher, _ := buildExternalWorkGoalFirstDomainDeliveryStackWithExecutorForTestV0(t)
	return stack, observer, launcher
}

func buildExternalWorkGoalFirstDomainDeliveryStackWithExecutorForTestV0(
	t *testing.T,
) (StackV0, *goalFirstQueueObserverForTestV0, *goalFirstQueueLauncherForTestV0, *fakeCodexStackDomainWorkExecutorV0) {
	t.Helper()
	launcher := &goalFirstQueueLauncherForTestV0{}
	observer := &goalFirstQueueObserverForTestV0{}
	domainWork := &fakeCodexStackDomainWorkExecutorV0{}
	config := codexStackBaseConfigForTestV0(
		t,
		newFakeCodexStackRuntimeV0(),
		domainWork,
		nil,
	)
	config.AppGoalLauncher = launcher
	config.AppGoalObserver = observer
	config.AppGoalClosureValidator = orquestagoal.DefaultGoalWorkClosureValidatorV0{}
	config.Stores.AppGoalStateStore = newGoalFirstQueueStateStoreForTestV0()
	stack, err := BuildStackV0(config)
	if err != nil {
		t.Fatalf("BuildStackV0: %v", err)
	}
	return stack, observer, launcher, domainWork
}

func writeGoalFirstDomainWorkArtifactForTestV0(
	t *testing.T,
	projectDir string,
	spec orquestagoal.GoalWorkSpecV0,
	artifactType string,
) {
	t.Helper()
	if len(spec.WriteSet) == 0 {
		t.Fatalf("spec sin write-set=%+v", spec)
	}
	dir := filepath.Join(projectDir, filepath.FromSlash(spec.WriteSet[0].Path))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	body := `{"artifact_type":"` + artifactType + `","payload_json":{"body":"Contenido OPES aceptable para cierre goal-first.","topic_id":"tema-001"}}`
	if codexStackDomainWorkFinalPackageArtifactTypeV0(artifactType) {
		body = `{
			"artifact_type":"` + artifactType + `",
			"payload_json":{
				"package_ref":"package-ref-goal-first-final-001",
				"manifest_cierre":{
					"schema_version":"opes_final_package_evidence_manifest.v0",
					"package_ref":"package-ref-goal-first-final-001",
					"manifest_ref":"manifest-cierre-ref-goal-first-final-001",
					"checksum_refs":["checksum-ref-goal-first-final-001"],
					"validation_report_ref":"validation-report-ref-goal-first-final-001",
					"review_matrix_ref":"review-matrix-ref-goal-first-final-001",
					"required_evidence_refs":{
						"html":"opes-final-evidence:html:goal-first",
						"rag":"opes-final-evidence:rag:goal-first",
						"audio":"opes-final-evidence:audio:goal-first",
						"tests":"opes-final-evidence:tests:goal-first",
						"visual":"opes-final-evidence:visual:goal-first",
						"qa":"opes-final-evidence:qa:goal-first"
					}
				}
			}
		}`
	}
	if err := os.WriteFile(filepath.Join(dir, artifactType+".json"), []byte(body), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func writeGoalFirstDomainWorkBlockedArtifactForTestV0(
	t *testing.T,
	projectDir string,
	spec orquestagoal.GoalWorkSpecV0,
	artifactType string,
) {
	t.Helper()
	if len(spec.WriteSet) == 0 {
		t.Fatalf("spec sin write-set=%+v", spec)
	}
	dir := filepath.Join(projectDir, filepath.FromSlash(spec.WriteSet[0].Path))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	body := `{"artifact_type":"` + artifactType + `","payload_json":{"ready":false,"status":"blocked","public_blocker":{"blocked_step":"apply_topic_registry_update"},"pending":["submit_artifact"]}}`
	if err := os.WriteFile(filepath.Join(dir, artifactType+".json"), []byte(body), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func externalWorkGoalFirstCompleteResultForTestV0(
	spec orquestagoal.GoalWorkSpecV0,
	externalGoalRef string,
	receiptRefs ...string,
) orquestagoal.GoalWorkResultV0 {
	return orquestagoal.GoalWorkResultV0{
		SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
		Status:          orquestagoal.GoalStatusCompleteV0,
		GoalRef:         spec.GoalRef,
		ExternalGoalRef: externalGoalRef,
		ArtifactRefs:    goalFirstQueueRequiredArtifactRefsV0(spec),
		RequiredTestResults: goalFirstQueueRequiredTestResultsV0(
			spec,
			"evidence-ref-external-work-goal-first-required-test",
		),
		DomainReceiptRefs: compactStringsV0(receiptRefs),
		EvidenceRefs:      append([]string(nil), spec.ClosurePolicy.RequiredEvidenceRefs...),
	}
}

func externalWorkGoalFirstAcceptedReceiptRecordForTestV0(
	runRef string,
	spec orquestagoal.GoalWorkSpecV0,
	receiptRef string,
) DomainWorkArtifactSubmissionRecordV0 {
	contract := orquestagoal.GoalArtifactContractV0{}
	if len(spec.ArtifactContracts) > 0 {
		contract = spec.ArtifactContracts[0]
	}
	return DomainWorkArtifactSubmissionRecordV0{
		IdempotencyKey: "idem-" + receiptRef,
		Status:         DomainWorkArtifactSubmissionStatusAcceptedV0,
		RunRef:         runRef,
		DeliveryRef:    contract.ArtifactRef,
		CorrelationID:  "corr-external-work-stack-001",
		DomainRef:      spec.DomainRef,
		JobRef:         "job-ref-001",
		ArtifactRef:    contract.ArtifactRef,
		ArtifactType:   contract.ArtifactType,
		Summary:        "Artefacto external-work aceptado por conector DomainWork.",
		CompleteJob:    true,
		ReceiptRef:     receiptRef,
		EvidenceRefs:   []string{"evidence-ref-" + receiptRef},
	}
}

func externalWorkGoalFirstRejectedReceiptRecordForTestV0(
	runRef string,
	spec orquestagoal.GoalWorkSpecV0,
	issueRefs []string,
) DomainWorkArtifactSubmissionRecordV0 {
	contract := orquestagoal.GoalArtifactContractV0{}
	if len(spec.ArtifactContracts) > 0 {
		contract = spec.ArtifactContracts[0]
	}
	work := defaultExternalWorkRunChangeForTestV0().ExternalWork
	task := goalFirstDomainWorkTaskV0(spec, contract)
	return DomainWorkArtifactSubmissionRecordV0{
		IdempotencyKey: domainWorkArtifactIdempotencyKeyV0(work, contract.ArtifactType),
		Status:         DomainWorkArtifactSubmissionStatusRejectedV0,
		RunRef:         runRef,
		TaskRef:        task.TaskID,
		DeliveryRef:    contract.ArtifactRef,
		CorrelationID:  "corr-external-work-stack-001",
		DomainRef:      spec.DomainRef,
		JobRef:         "job-ref-001",
		ArtifactRef:    contract.ArtifactRef,
		ArtifactType:   contract.ArtifactType,
		Summary:        "Artefacto external-work rechazado por conector DomainWork.",
		CompleteJob:    true,
		IssueRefs:      compactStringsV0(issueRefs),
		EvidenceRefs:   []string{"evidence-ref-domain-work-rejected-for-test"},
	}
}

func goalClosureHasIssueForTestV0(
	closure orquestagoal.GoalClosureValidationV0,
	code string,
) bool {
	for _, issue := range closure.Issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

func goalDomainReceiptOPESSubroleEvidenceRefsForTestV0() []string {
	return []string{
		"domain-work-opes-subrole-fuentes",
		"domain-work-opes-subrole-reutilizacion",
		"domain-work-opes-subrole-redaccion",
		"domain-work-opes-subrole-visuales",
		"domain-work-opes-subrole-tests-tutor",
		"domain-work-opes-subrole-html-rag-audio-qa",
	}
}

func goalDomainReceiptOPESSubroleTaskRefsForTestV0() []string {
	return []string{
		"task-opes-subrole-job-ref-goal-first-opes-subroles-001-redaccion",
		"task-opes-subrole-job-ref-goal-first-opes-subroles-001-tests",
		"task-opes-subrole-job-ref-goal-first-opes-subroles-001-visuales",
		"task-opes-subrole-job-ref-goal-first-opes-subroles-001-audio",
		"task-opes-subrole-job-ref-goal-first-opes-subroles-001-tutor_rag",
		"task-opes-subrole-job-ref-goal-first-opes-subroles-001-qa",
	}
}

func goalDomainReceiptOPESSubroleTaskRefsFieldForTestV0() orquestadomainwork.DomainWorkFieldV0 {
	return orquestadomainwork.DomainWorkFieldV0{
		Name:   "opes_subrole_task_refs",
		Values: goalDomainReceiptOPESSubroleTaskRefsForTestV0(),
	}
}

func goalDomainReceiptOPESSubroleTaskRefEvidenceRefsForTestV0() []string {
	taskRefs := goalDomainReceiptOPESSubroleTaskRefsForTestV0()
	refs := make([]string, 0, len(taskRefs))
	for _, taskRef := range taskRefs {
		refs = append(refs, goalDomainReceiptOPESSubrolesEvidencePrefixV0+taskRef)
	}
	return refs
}

func codexStackGoalContextRefsTextForTestV0(refs []orquestagoal.GoalContextRefV0) string {
	parts := make([]string, 0, len(refs))
	for _, ref := range refs {
		parts = append(parts, strings.TrimSpace(ref.Kind+" "+ref.Ref+" "+ref.Purpose))
	}
	return strings.Join(parts, "\n")
}

func postExternalWorkRunStackWithChangeV0(
	t *testing.T,
	stack StackV0,
	change orquestaappchange.AppChangeRequestV0,
) orquestamcp.MCPExternalWorkRunToolResultV0 {
	t.Helper()
	return postExternalWorkRunStackRawV0(t, stack, http.StatusOK, change)
}

func postExternalWorkRunStackWithChangeLegacyV0(
	t *testing.T,
	stack StackV0,
	change orquestaappchange.AppChangeRequestV0,
) orquestamcp.MCPExternalWorkRunToolResultV0 {
	t.Helper()
	return postExternalWorkRunStackRawWithModeV0(
		t,
		stack,
		http.StatusOK,
		change,
		orquestamcp.MCPExternalWorkRunDirectorExecutionModeLegacyLoopV0,
	)
}

func postExternalWorkRunStackRawV0(
	t *testing.T,
	stack StackV0,
	wantStatus int,
	change orquestaappchange.AppChangeRequestV0,
) orquestamcp.MCPExternalWorkRunToolResultV0 {
	t.Helper()
	return postExternalWorkRunStackRawWithModeV0(t, stack, wantStatus, change, "")
}

func postExternalWorkRunStackRawWithModeV0(
	t *testing.T,
	stack StackV0,
	wantStatus int,
	change orquestaappchange.AppChangeRequestV0,
	directorExecutionMode string,
) orquestamcp.MCPExternalWorkRunToolResultV0 {
	t.Helper()
	body := bytes.NewBuffer(nil)
	err := json.NewEncoder(body).Encode(orquestamcp.MCPExternalWorkRunToolInputV0{
		RequestID:             "req-external-work-stack-001",
		CorrelationID:         "corr-external-work-stack-001",
		DirectorExecutionMode: directorExecutionMode,
		AppChangeRequest:      change,
	})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/external-work/run", body)
	req.Header.Set("Content-Type", "application/json")
	stack.Handler.ServeHTTP(rec, req)
	if rec.Code != wantStatus {
		t.Fatalf("external work run status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPExternalWorkRunToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode external work run: %v", err)
	}
	if wantStatus == http.StatusOK && result.Estado != orquestamcp.MCPExternalWorkRunEstadoOKV0 {
		t.Fatalf("result=%+v", result)
	}
	return result
}

func codexStackOPESSubroleTasksForTestV0(
	tasks []orquestacoreworkflow.WorkflowTaskV0,
) (orquestacoreworkflow.WorkflowTaskV0, []orquestacoreworkflow.WorkflowTaskV0) {
	children := make([]orquestacoreworkflow.WorkflowTaskV0, 0, 6)
	var parent orquestacoreworkflow.WorkflowTaskV0
	for _, task := range tasks {
		if strings.TrimSpace(task.ParentTaskRef) != "" {
			children = append(children, task)
			continue
		}
		if len(task.ChildTaskRefs) > 0 {
			parent = task
		}
	}
	return parent, children
}

func containsOPESSubroleWriteSetForTestV0(writeSet []string) bool {
	for _, entry := range writeSet {
		if strings.Contains(entry, "/subroles/") {
			return true
		}
	}
	return false
}

func codexStackExternalWorkPlanStepForTestV0(
	t *testing.T,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
	stepID string,
) orquestacionnucleoapp.OperationalDirectorPlanStepStateV0 {
	t.Helper()
	for _, step := range state.Steps {
		if step.StepID == stepID {
			return step
		}
	}
	t.Fatalf("step %s no existe en %+v", stepID, state)
	return orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{}
}

func appChangeSafeRefPartForTestV0(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\\", "-")
	value = strings.ReplaceAll(value, "/", "-")
	value = strings.ReplaceAll(value, " ", "-")
	return value
}
