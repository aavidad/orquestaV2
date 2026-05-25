package orquestadirector

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestBuildAgentProgressSupervisionV0LoopDetectedDetieneAgente(t *testing.T) {
	const (
		taskRef       = "task-ref-supervisor-loop-001"
		agentRef      = "agent-request-ref-supervisor-loop-001"
		assessmentRef = "assessment-ref-supervisor-loop-001"
	)
	h := newSupervisionHarnessWithAgentV0(t, taskRef, agentRef, "loop")
	report := validSupervisionProgressReportV0(h.run.RunID, agentRef, orquestaruntime.AgentLoopDetectedV0)
	report.ReportID = "agent-progress-report-ref-supervisor-loop-001"
	report.NoProgressTicks = 4
	report.RepeatedActionCount = 3

	result, err := BuildAgentProgressSupervisionV0(AgentProgressSupervisionInputV0{
		CommandMeta:   h.meta("agent-supervision-loop"),
		Report:        report,
		PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		TaskRef:       taskRef,
		AssessmentRef: assessmentRef,
	})
	if err != nil {
		t.Fatalf("BuildAgentProgressSupervisionV0: %v", err)
	}
	if result.AskDirectorCommand != nil {
		t.Fatalf("AskDirector no esperado: %+v", result.AskDirectorCommand)
	}
	payload := decodeAssessAgentWorkPayloadV0(t, result.AssessCommand)
	if payload.Verdict != orquestacoreworkflow.AgentAssessmentVerdictLoopDetectedV0 ||
		payload.Action != orquestacoreworkflow.AgentAssessmentActionStopAgentV0 ||
		payload.Severity != orquestacoreworkflow.AgentAssessmentSeverityCriticalV0 {
		t.Fatalf("payload assessment inesperado: %+v", payload)
	}

	handled := h.handle(result.AssessCommand)
	if len(handled.Events) != 2 ||
		handled.Events[0].EventType != orquestacoreworkflow.OrchestrationEventAgentWorkAssessedV0 ||
		handled.Events[1].EventType != orquestacoreworkflow.OrchestrationEventAgentStopRequestedV0 {
		t.Fatalf("eventos inesperados: %+v", handled.Events)
	}
	if len(handled.Outbox) != 1 ||
		handled.Outbox[0].MessageType != orquestacoreworkflow.OutboxMessageStopRuntimeAgentV0 {
		t.Fatalf("outbox inesperado: %+v", handled.Outbox)
	}
}

func TestBuildAgentProgressSupervisionV0StalledPreguntaNoBloqueanteAlDirector(t *testing.T) {
	const (
		taskRef       = "task-ref-supervisor-stalled-001"
		agentRef      = "agent-request-ref-supervisor-stalled-001"
		assessmentRef = "assessment-ref-supervisor-stalled-001"
		questionID    = "question-ref-supervisor-stalled-001"
	)
	h := newSupervisionHarnessWithAgentV0(t, taskRef, agentRef, "stalled")
	report := validSupervisionProgressReportV0(h.run.RunID, agentRef, orquestaruntime.AgentStalledV0)
	report.ReportID = "agent-progress-report-ref-supervisor-stalled-001"
	report.NoProgressTicks = 6
	report.RepeatedActionCount = 1

	result, err := BuildAgentProgressSupervisionV0(AgentProgressSupervisionInputV0{
		CommandMeta:   h.meta("agent-supervision-stalled"),
		Report:        report,
		PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		TaskRef:       taskRef,
		AssessmentRef: assessmentRef,
		QuestionID:    questionID,
	})
	if err != nil {
		t.Fatalf("BuildAgentProgressSupervisionV0: %v", err)
	}
	payload := decodeAssessAgentWorkPayloadV0(t, result.AssessCommand)
	if payload.Verdict != orquestacoreworkflow.AgentAssessmentVerdictNeedsRevisionV0 ||
		payload.Action != orquestacoreworkflow.AgentAssessmentActionAskDirectorV0 ||
		payload.Severity != orquestacoreworkflow.AgentAssessmentSeverityHighV0 {
		t.Fatalf("payload assessment inesperado: %+v", payload)
	}
	if result.AskDirectorCommand == nil {
		t.Fatalf("AskDirector command requerido")
	}
	askPayload := decodeAskDirectorPayloadV0(t, *result.AskDirectorCommand)
	if askPayload.Blocking {
		t.Fatalf("stalled no debe bloquear el run: %+v", askPayload)
	}

	h.handle(result.AssessCommand)
	asked := h.handle(*result.AskDirectorCommand)
	if len(asked.Outbox) != 1 ||
		asked.Outbox[0].MessageType != orquestacoreworkflow.OutboxMessageSendDirectorQuestionV0 {
		t.Fatalf("outbox AskDirector inesperado: %+v", asked.Outbox)
	}
	if supervisionEventTypeCountV0(asked.Events, orquestacoreworkflow.OrchestrationEventRunBlockedV0) != 0 {
		t.Fatalf("stalled no debe emitir RunBlocked: %+v", asked.Events)
	}
}

func TestBuildAgentProgressSupervisionV0CapacidadLimitadaCierraParaRelevo(t *testing.T) {
	const (
		taskRef       = "task-ref-supervisor-capacity-001"
		agentRef      = "agent-request-ref-supervisor-capacity-001"
		assessmentRef = "assessment-ref-supervisor-capacity-001"
	)
	h := newSupervisionHarnessWithAgentV0(t, taskRef, agentRef, "capacity")
	report := validSupervisionProgressReportV0(h.run.RunID, agentRef, orquestaruntime.AgentStoppedV0)
	report.ReportID = "agent-progress-report-ref-supervisor-capacity-001"
	report.BudgetStatus = orquestaruntime.AgentProgressBudgetCapacityLimitedV0
	report.BudgetReason = "Capacidad externa limitada antes de ACK."
	report.DecisionRequired = true

	result, err := BuildAgentProgressSupervisionV0(AgentProgressSupervisionInputV0{
		CommandMeta:   h.meta("agent-supervision-capacity"),
		Report:        report,
		PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		TaskRef:       taskRef,
		AssessmentRef: assessmentRef,
	})
	if err != nil {
		t.Fatalf("BuildAgentProgressSupervisionV0: %v", err)
	}
	if result.AskDirectorCommand != nil {
		t.Fatalf("AskDirector no esperado para relevo automatico: %+v", result.AskDirectorCommand)
	}
	payload := decodeAssessAgentWorkPayloadV0(t, result.AssessCommand)
	if payload.Verdict != orquestacoreworkflow.AgentAssessmentVerdictCapacityLimitedV0 ||
		payload.Action != orquestacoreworkflow.AgentAssessmentActionStopAgentV0 ||
		payload.Severity != orquestacoreworkflow.AgentAssessmentSeverityHighV0 {
		t.Fatalf("payload capacity inesperado: %+v", payload)
	}
}

func TestBuildAgentProgressSupervisionV0AuthInvalidaPreguntaDirectorSinStop(t *testing.T) {
	const (
		taskRef       = "task-ref-supervisor-auth-001"
		agentRef      = "agent-request-ref-supervisor-auth-001"
		assessmentRef = "assessment-ref-supervisor-auth-001"
		questionID    = "question-ref-supervisor-auth-001"
	)
	h := newSupervisionHarnessWithAgentV0(t, taskRef, agentRef, "auth")
	report := validSupervisionProgressReportV0(h.run.RunID, agentRef, orquestaruntime.AgentStoppedV0)
	report.ReportID = "agent-progress-report-ref-supervisor-auth-001"
	report.DecisionRequired = true
	report.Summary = "Autenticacion externa invalida; requiere reautorizacion del operador."
	report.EvidenceRefs = append(report.EvidenceRefs, "evidence-ref-auth-config-blocker")

	result, err := BuildAgentProgressSupervisionV0(AgentProgressSupervisionInputV0{
		CommandMeta:   h.meta("agent-supervision-auth"),
		Report:        report,
		PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		TaskRef:       taskRef,
		AssessmentRef: assessmentRef,
		QuestionID:    questionID,
	})
	if err != nil {
		t.Fatalf("BuildAgentProgressSupervisionV0: %v", err)
	}
	payload := decodeAssessAgentWorkPayloadV0(t, result.AssessCommand)
	if payload.Verdict != orquestacoreworkflow.AgentAssessmentVerdictNeedsRevisionV0 ||
		payload.Action != orquestacoreworkflow.AgentAssessmentActionAskDirectorV0 ||
		payload.Severity != orquestacoreworkflow.AgentAssessmentSeverityCriticalV0 {
		t.Fatalf("payload auth inesperado: %+v", payload)
	}
	if result.AskDirectorCommand == nil {
		t.Fatalf("AskDirector command requerido")
	}
	if result.RegisterLostCommand == nil {
		t.Fatalf("RegisterAgentLost command requerido")
	}
	handled := h.handle(result.AssessCommand)
	if len(handled.Outbox) != 0 {
		t.Fatalf("auth invalida no debe pedir stop automatico: %+v", handled.Outbox)
	}
	lost := h.handle(*result.RegisterLostCommand)
	if len(lost.Outbox) != 0 ||
		supervisionEventTypeCountV0(lost.Events, orquestacoreworkflow.OrchestrationEventAgentLostV0) != 1 {
		t.Fatalf("auth invalida debe registrar perdida sin outbox: %+v", lost)
	}
}

func TestBuildAgentProgressSupervisionV0NoACKPreguntaDirectorSinGarbage(t *testing.T) {
	const (
		taskRef       = "task-ref-supervisor-no-ack-001"
		agentRef      = "agent-request-ref-supervisor-no-ack-001"
		assessmentRef = "assessment-ref-supervisor-no-ack-001"
		questionID    = "question-ref-supervisor-no-ack-001"
	)
	h := newSupervisionHarnessWithAgentV0(t, taskRef, agentRef, "no-ack")
	report := validSupervisionProgressReportV0(h.run.RunID, agentRef, orquestaruntime.AgentStoppedV0)
	report.ReportID = "agent-progress-report-ref-supervisor-no-ack-001"
	report.DecisionRequired = true
	report.Summary = "Proceso detenido sin ACK."
	report.EvidenceRefs = append(report.EvidenceRefs, "evidence-ref-no-ack")

	result, err := BuildAgentProgressSupervisionV0(AgentProgressSupervisionInputV0{
		CommandMeta:   h.meta("agent-supervision-no-ack"),
		Report:        report,
		PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		TaskRef:       taskRef,
		AssessmentRef: assessmentRef,
		QuestionID:    questionID,
	})
	if err != nil {
		t.Fatalf("BuildAgentProgressSupervisionV0: %v", err)
	}
	payload := decodeAssessAgentWorkPayloadV0(t, result.AssessCommand)
	if payload.Verdict != orquestacoreworkflow.AgentAssessmentVerdictNeedsRevisionV0 ||
		payload.Action != orquestacoreworkflow.AgentAssessmentActionAskDirectorV0 ||
		payload.Severity != orquestacoreworkflow.AgentAssessmentSeverityHighV0 ||
		payload.Summary != "Proceso detenido sin ACK; revisar evidencias antes de replanificar." {
		t.Fatalf("payload no_ack inesperado: %+v", payload)
	}
	if result.AskDirectorCommand == nil {
		t.Fatalf("AskDirector command requerido")
	}
	if result.RegisterLostCommand == nil {
		t.Fatalf("RegisterAgentLost command requerido")
	}
	handled := h.handle(result.AssessCommand)
	if len(handled.Outbox) != 0 {
		t.Fatalf("no_ack no debe pedir stop automatico: %+v", handled.Outbox)
	}
	lost := h.handle(*result.RegisterLostCommand)
	if len(lost.Outbox) != 0 ||
		supervisionEventTypeCountV0(lost.Events, orquestacoreworkflow.OrchestrationEventAgentLostV0) != 1 {
		t.Fatalf("no_ack debe registrar perdida sin outbox: %+v", lost)
	}
}

func TestBuildAgentProgressSupervisionV0TiempoSinActividadDetieneAunqueHayaStalledPrevio(t *testing.T) {
	const (
		taskRef       = "task-ref-supervisor-timeout-001"
		agentRef      = "agent-request-ref-supervisor-timeout-001"
		assessmentRef = "assessment-ref-supervisor-timeout-001"
	)
	h := newSupervisionHarnessWithAgentV0(t, taskRef, agentRef, "timeout")
	report := validSupervisionProgressReportV0(h.run.RunID, agentRef, orquestaruntime.AgentStalledV0)
	report.ReportID = "agent-progress-report-ref-supervisor-timeout-001"
	report.BudgetStatus = orquestaruntime.AgentProgressBudgetOverBudgetNoActivityV0
	report.BudgetReason = "Tiempo excedido sin actividad reciente."
	report.DecisionRequired = true
	report.EvidenceRefs = append(report.EvidenceRefs, "evidence-ref-artifact-without-ack")

	result, err := BuildAgentProgressSupervisionV0(AgentProgressSupervisionInputV0{
		CommandMeta:   h.meta("agent-supervision-timeout"),
		Report:        report,
		PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		TaskRef:       taskRef,
		AssessmentRef: assessmentRef,
	})
	if err != nil {
		t.Fatalf("BuildAgentProgressSupervisionV0: %v", err)
	}
	if result.AskDirectorCommand != nil {
		t.Fatalf("AskDirector no esperado para timeout sin actividad: %+v", result.AskDirectorCommand)
	}
	payload := decodeAssessAgentWorkPayloadV0(t, result.AssessCommand)
	if payload.Verdict != orquestacoreworkflow.AgentAssessmentVerdictTimeoutV0 ||
		payload.Action != orquestacoreworkflow.AgentAssessmentActionStopAgentV0 ||
		payload.Severity != orquestacoreworkflow.AgentAssessmentSeverityHighV0 ||
		!strings.Contains(payload.Summary, "artefacto sin ACK") {
		t.Fatalf("payload timeout inesperado: %+v", payload)
	}
	handled := h.handle(result.AssessCommand)
	if len(handled.Outbox) != 1 ||
		handled.Outbox[0].MessageType != orquestacoreworkflow.OutboxMessageStopRuntimeAgentV0 {
		t.Fatalf("outbox timeout inesperado: %+v", handled.Outbox)
	}
}

func TestBuildAgentProgressSupervisionV0TiempoExcedidoConActividadPreguntaDirector(t *testing.T) {
	const (
		taskRef       = "task-ref-supervisor-over-budget-active-001"
		agentRef      = "agent-request-ref-supervisor-over-budget-active-001"
		assessmentRef = "assessment-ref-supervisor-over-budget-active-001"
		questionID    = "question-ref-supervisor-over-budget-active-001"
	)
	h := newSupervisionHarnessWithAgentV0(t, taskRef, agentRef, "over-budget-active")
	report := validSupervisionProgressReportV0(h.run.RunID, agentRef, orquestaruntime.AgentProgressingV0)
	report.ReportID = "agent-progress-report-ref-supervisor-over-budget-active-001"
	report.BudgetStatus = orquestaruntime.AgentProgressBudgetOverBudgetButActiveV0
	report.BudgetReason = "Tiempo excedido con actividad reciente."
	report.DecisionRequired = true

	result, err := BuildAgentProgressSupervisionV0(AgentProgressSupervisionInputV0{
		CommandMeta:   h.meta("agent-supervision-over-budget-active"),
		Report:        report,
		PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		TaskRef:       taskRef,
		AssessmentRef: assessmentRef,
		QuestionID:    questionID,
	})
	if err != nil {
		t.Fatalf("BuildAgentProgressSupervisionV0: %v", err)
	}
	payload := decodeAssessAgentWorkPayloadV0(t, result.AssessCommand)
	if payload.Verdict != orquestacoreworkflow.AgentAssessmentVerdictNeedsRevisionV0 ||
		payload.Action != orquestacoreworkflow.AgentAssessmentActionAskDirectorV0 ||
		payload.Severity != orquestacoreworkflow.AgentAssessmentSeverityHighV0 {
		t.Fatalf("payload over budget activo inesperado: %+v", payload)
	}
	if result.AskDirectorCommand == nil {
		t.Fatalf("AskDirector command requerido")
	}
	handled := h.handle(result.AssessCommand)
	if len(handled.Outbox) != 0 {
		t.Fatalf("over budget activo no debe parar directamente: %+v", handled.Outbox)
	}
}

func TestBuildAgentProgressSupervisionV0ArtefactoSinACKPreguntaDirector(t *testing.T) {
	const (
		taskRef       = "task-ref-supervisor-artifact-001"
		agentRef      = "agent-request-ref-supervisor-artifact-001"
		assessmentRef = "assessment-ref-supervisor-artifact-001"
		questionID    = "question-ref-supervisor-artifact-001"
	)
	h := newSupervisionHarnessWithAgentV0(t, taskRef, agentRef, "artifact")
	report := validSupervisionProgressReportV0(h.run.RunID, agentRef, orquestaruntime.AgentStoppedV0)
	report.ReportID = "agent-progress-report-ref-supervisor-artifact-001"
	report.DecisionRequired = true
	report.EvidenceRefs = append(report.EvidenceRefs, "evidence-ref-no-ack", "evidence-ref-artifact-without-ack")

	result, err := BuildAgentProgressSupervisionV0(AgentProgressSupervisionInputV0{
		CommandMeta:   h.meta("agent-supervision-artifact"),
		Report:        report,
		PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		TaskRef:       taskRef,
		AssessmentRef: assessmentRef,
		QuestionID:    questionID,
	})
	if err != nil {
		t.Fatalf("BuildAgentProgressSupervisionV0: %v", err)
	}
	payload := decodeAssessAgentWorkPayloadV0(t, result.AssessCommand)
	if payload.Verdict != orquestacoreworkflow.AgentAssessmentVerdictNeedsRevisionV0 ||
		payload.Action != orquestacoreworkflow.AgentAssessmentActionAskDirectorV0 ||
		payload.Severity != orquestacoreworkflow.AgentAssessmentSeverityHighV0 {
		t.Fatalf("payload artifact inesperado: %+v", payload)
	}
	if result.AskDirectorCommand == nil {
		t.Fatalf("AskDirector command requerido")
	}
	handled := h.handle(result.AssessCommand)
	if len(handled.Outbox) != 0 {
		t.Fatalf("artefacto sin ACK no debe pedir stop inmediato: %+v", handled.Outbox)
	}
	askPayload := decodeAskDirectorPayloadV0(t, *result.AskDirectorCommand)
	if askPayload.Blocking {
		t.Fatalf("revision de artefacto no debe bloquear el run: %+v", askPayload)
	}
}

func TestBuildAgentProgressSupervisionV0LoopConArtefactoNoDetieneSinRevision(t *testing.T) {
	const (
		taskRef       = "task-ref-supervisor-loop-artifact-001"
		agentRef      = "agent-request-ref-supervisor-loop-artifact-001"
		assessmentRef = "assessment-ref-supervisor-loop-artifact-001"
		questionID    = "question-ref-supervisor-loop-artifact-001"
	)
	h := newSupervisionHarnessWithAgentV0(t, taskRef, agentRef, "loop-artifact")
	report := validSupervisionProgressReportV0(h.run.RunID, agentRef, orquestaruntime.AgentLoopDetectedV0)
	report.ReportID = "agent-progress-report-ref-supervisor-loop-artifact-001"
	report.DecisionRequired = true
	report.RepeatedActionCount = 4
	report.EvidenceRefs = append(report.EvidenceRefs, "evidence-ref-no-ack", "evidence-ref-artifact-without-ack")

	result, err := BuildAgentProgressSupervisionV0(AgentProgressSupervisionInputV0{
		CommandMeta:   h.meta("agent-supervision-loop-artifact"),
		Report:        report,
		PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		TaskRef:       taskRef,
		AssessmentRef: assessmentRef,
		QuestionID:    questionID,
	})
	if err != nil {
		t.Fatalf("BuildAgentProgressSupervisionV0: %v", err)
	}
	payload := decodeAssessAgentWorkPayloadV0(t, result.AssessCommand)
	if payload.Verdict != orquestacoreworkflow.AgentAssessmentVerdictNeedsRevisionV0 ||
		payload.Action != orquestacoreworkflow.AgentAssessmentActionAskDirectorV0 ||
		payload.Summary != "Entrega sin ACK con artefacto materializado; validar antes de replanificar." {
		t.Fatalf("payload artifact inesperado: %+v", payload)
	}
	handled := h.handle(result.AssessCommand)
	if len(handled.Outbox) != 0 {
		t.Fatalf("artefacto en loop no debe pedir stop inmediato: %+v", handled.Outbox)
	}
	if result.AskDirectorCommand == nil {
		t.Fatalf("AskDirector command requerido")
	}
}

func TestBuildAgentProgressSupervisionV0RechazaReporteInvalido(t *testing.T) {
	h := newProgressiveHarnessV0(t)
	report := validSupervisionProgressReportV0(h.run.RunID, "agent-request-ref-invalid-001", orquestaruntime.AgentStalledV0)
	report.Status = "waiting"

	_, err := BuildAgentProgressSupervisionV0(AgentProgressSupervisionInputV0{
		CommandMeta:   h.meta("agent-supervision-invalid"),
		Report:        report,
		PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		AssessmentRef: "assessment-ref-supervisor-invalid-001",
		QuestionID:    "question-ref-supervisor-invalid-001",
	})

	var supervisionErr AgentProgressSupervisionErrorV0
	if !errors.As(err, &supervisionErr) {
		t.Fatalf("error type=%T, want AgentProgressSupervisionErrorV0", err)
	}
	if supervisionErr.Code != ErrDirectorAgentProgressSupervisionInvalidaV0 {
		t.Fatalf("code=%q", supervisionErr.Code)
	}
}

func TestBuildAgentProgressSupervisionV0SalidaSinDetallesProhibidos(t *testing.T) {
	h := newSupervisionHarnessWithAgentV0(t, "task-ref-supervisor-safe-001", "agent-request-ref-supervisor-safe-001", "safe")
	report := validSupervisionProgressReportV0(h.run.RunID, "agent-request-ref-supervisor-safe-001", orquestaruntime.AgentStoppedV0)
	report.ReportID = "agent-progress-report-ref-supervisor-safe-001"
	report.Summary = "Ver transcript completo en HOME no debe propagarse."
	report.EvidenceRefs = []string{"transcript-ref-supervisor-001", "supervision-evidence-ref-safe-001"}

	result, err := BuildAgentProgressSupervisionV0(AgentProgressSupervisionInputV0{
		CommandMeta:   h.meta("agent-supervision-safe"),
		Report:        report,
		PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		TaskRef:       "task-ref-supervisor-safe-001",
		AssessmentRef: "assessment-ref-supervisor-safe-001",
	})
	if err != nil {
		t.Fatalf("BuildAgentProgressSupervisionV0: %v", err)
	}
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	serialized := strings.ToLower(string(raw))
	for _, forbidden := range []string{"provider", "proveedor", "mysql", "postgres", "sqlite", "oauth", "home", "transcript", "prompt"} {
		if strings.Contains(serialized, forbidden) {
			t.Fatalf("serialized result contains forbidden fragment %q: %s", forbidden, serialized)
		}
	}
	payload := decodeAssessAgentWorkPayloadV0(t, result.AssessCommand)
	if payload.Verdict != orquestacoreworkflow.AgentAssessmentVerdictGarbageV0 ||
		payload.Action != orquestacoreworkflow.AgentAssessmentActionStopAgentV0 ||
		payload.Severity != orquestacoreworkflow.AgentAssessmentSeverityHighV0 {
		t.Fatalf("payload stopped inesperado: %+v", payload)
	}
}

func TestBuildAgentProgressSupervisionV0AceptaCorrelationIDOpaquo(t *testing.T) {
	h := newSupervisionHarnessWithAgentV0(t, "task-ref-supervisor-opaque-001", "agent-request-ref-supervisor-opaque-001", "opaque")
	report := validSupervisionProgressReportV0(h.run.RunID, "agent-request-ref-supervisor-opaque-001", orquestaruntime.AgentStoppedV0)
	report.ReportID = "agent-progress-report-ref-supervisor-opaque-001"
	meta := h.meta("agent-supervision-opaque")
	meta.CorrelationID = "corr-opaco-db-home-model-001"

	_, err := BuildAgentProgressSupervisionV0(AgentProgressSupervisionInputV0{
		CommandMeta:   meta,
		Report:        report,
		PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		TaskRef:       "task-ref-supervisor-opaque-001",
		AssessmentRef: "assessment-ref-supervisor-opaque-001",
	})
	if err != nil {
		t.Fatalf("BuildAgentProgressSupervisionV0: %v", err)
	}
}

func newSupervisionHarnessWithAgentV0(
	t *testing.T,
	taskRef string,
	agentRef string,
	suffix string,
) *progressiveHarnessV0 {
	t.Helper()
	h := newProgressiveHarnessV0(t)
	runtimeFake := newProgressiveRuntimeFakeOutboxDispatcherV0(t)
	launched := h.prepareFullFlowTask(
		taskRef,
		"contract-ref-supervisor-"+suffix+"-001",
		"capacity-request-ref-supervisor-"+suffix+"-001",
		agentRef,
		runtimeFake,
	)
	h.handle(h.registerAgentStarted(agentRef, launched.LaunchRef))
	return h
}

func validSupervisionProgressReportV0(
	runID string,
	agentRef string,
	status orquestaruntime.AgentProgressStatusV0,
) orquestaruntime.AgentProgressReportV0 {
	return orquestaruntime.AgentProgressReportV0{
		ReportID:            "agent-progress-report-ref-supervisor-001",
		RunID:               runID,
		AgentRequestID:      agentRef,
		Status:              status,
		NoProgressTicks:     2,
		RepeatedActionCount: 1,
		Summary:             "Evidencia compacta de supervision.",
		EvidenceRefs:        []string{"supervision-evidence-ref-001"},
	}
}

func decodeAssessAgentWorkPayloadV0(
	t *testing.T,
	command orquestacoreworkflow.OrchestrationCommandV0,
) orquestacoreworkflow.AssessAgentWorkCommandPayloadV0 {
	t.Helper()
	var payload orquestacoreworkflow.AssessAgentWorkCommandPayloadV0
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		t.Fatalf("decode AssessAgentWork payload: %v", err)
	}
	return payload
}

func decodeAskDirectorPayloadV0(
	t *testing.T,
	command orquestacoreworkflow.OrchestrationCommandV0,
) orquestacoreworkflow.AskDirectorCommandPayloadV0 {
	t.Helper()
	var payload orquestacoreworkflow.AskDirectorCommandPayloadV0
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		t.Fatalf("decode AskDirector payload: %v", err)
	}
	return payload
}

func supervisionEventTypeCountV0(
	events []orquestacoreworkflow.OrchestrationEventV0,
	eventType string,
) int {
	count := 0
	for _, event := range events {
		if event.EventType == eventType {
			count++
		}
	}
	return count
}
