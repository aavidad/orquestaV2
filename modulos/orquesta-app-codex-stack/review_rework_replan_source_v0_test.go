package orquestaappcodexstack

import (
	"context"
	"os"
	"strings"
	"testing"

	orquestacorereplanner "orquesta/modulos/orquesta-core-replanner"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func TestReviewReworkReplanSourceV0UsaTaskRefDelReceiptDelRework(t *testing.T) {
	store := orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(
		reviewReworkDescriptorForTestV0("delivery-ref-first", "task-ref-first"),
		reviewReworkDescriptorForTestV0("delivery-ref-target", "task-ref-target"),
	)
	source := ReviewReworkReplanSourceV0{
		Store: store,
		Capacity: CapacityConfigV0{
			Tier: orquestacoreworkflow.OrchestrationCapacityXHighV0,
		},
	}

	plans, err := source.BuildReviewReworkReplanPlansV0(
		context.Background(),
		reviewReworkPlanRequestForTestV0(false),
	)
	if err != nil {
		t.Fatalf("BuildReviewReworkReplanPlansV0: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("plans=%d %+v", len(plans), plans)
	}
	plan := plans[0]
	if plan.TaskRef != "task-ref-target" {
		t.Fatalf("task_ref=%q, want task-ref-target", plan.TaskRef)
	}
	if plan.TaskRef == "task-ref-first" {
		t.Fatalf("uso la primera tarea como preferente: %+v", plan)
	}
	if plan.MinimumRecommendedCapacity != orquestacoreworkflow.OrchestrationCapacityXHighV0 {
		t.Fatalf("capacity=%q", plan.MinimumRecommendedCapacity)
	}
	if plan.ReplanRef == "" || plan.CapacityRequestRef == "" ||
		plan.AgentRequestID == "" || plan.ReasonRef == "" {
		t.Fatalf("refs incompletas: %+v", plan)
	}
}

func TestReviewReworkReplanSourceV0FallbackPrimeraTareaSoloSinReceipt(t *testing.T) {
	source := ReviewReworkReplanSourceV0{
		Store: orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(),
	}

	plans, err := source.BuildReviewReworkReplanPlansV0(
		context.Background(),
		reviewReworkPlanRequestForTestV0(false),
	)
	if err != nil {
		t.Fatalf("BuildReviewReworkReplanPlansV0: %v", err)
	}
	if len(plans) != 1 || plans[0].TaskRef != "task-ref-first" {
		t.Fatalf("plans=%+v", plans)
	}
	if plans[0].MinimumRecommendedCapacity != orquestacoreworkflow.OrchestrationCapacityHighV0 {
		t.Fatalf("capacity fallback=%q", plans[0].MinimumRecommendedCapacity)
	}
	if !reviewReworkPlanHasEvidenceForTestV0(plans[0].EvidenceRefs, "evidence-ref-review-rework-task-fallback") {
		t.Fatalf("fallback sin evidencia: %v", plans[0].EvidenceRefs)
	}
}

func TestReviewReworkReplanSourceV0MantieneEvidenciaOperativaOpaca(t *testing.T) {
	source := ReviewReworkReplanSourceV0{
		Store: orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(
			reviewReworkDescriptorForTestV0("delivery-ref-target", "task-ref-target"),
		),
	}
	request := reviewReworkPlanRequestForTestV0(false)
	request.EvidenceRefs = []string{
		"evidence-ref-neutral-review-rework",
		"evidence-ref-codex-supervisor-stack-drain",
		"evidence-ref-runtime-drain",
	}

	plans, err := source.BuildReviewReworkReplanPlansV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildReviewReworkReplanPlansV0: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("plans=%d %+v", len(plans), plans)
	}
	evidence := plans[0].EvidenceRefs
	for _, want := range request.EvidenceRefs {
		if !reviewReworkPlanHasEvidenceForTestV0(evidence, want) {
			t.Fatalf("evidence_refs perdio evidencia operativa %q: %v", want, evidence)
		}
	}
}

func TestReviewReworkReplanSourceV0MantienePlanTrasReplanHastaAgente(t *testing.T) {
	store := orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(
		reviewReworkDescriptorForTestV0("delivery-ref-target", "task-ref-target"),
	)
	source := ReviewReworkReplanSourceV0{Store: store}

	plans, err := source.BuildReviewReworkReplanPlansV0(
		context.Background(),
		reviewReworkPlanRequestForTestV0(true),
	)
	if err != nil {
		t.Fatalf("BuildReviewReworkReplanPlansV0: %v", err)
	}
	if len(plans) != 1 || plans[0].TaskRef != "task-ref-target" {
		t.Fatalf("plan tras replan debe seguir disponible hasta agente: %+v", plans)
	}

	request := reviewReworkPlanRequestForTestV0(true)
	request.Run.Agents = []string{plans[0].AgentRequestID}
	plans, err = source.BuildReviewReworkReplanPlansV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildReviewReworkReplanPlansV0 con agente: %v", err)
	}
	if len(plans) != 0 {
		t.Fatalf("plan debe parar tras agente solicitado: %+v", plans)
	}
}

func TestReviewReworkReplanSourceV0NoCreaSegundoPadreParaMismaTarea(t *testing.T) {
	descriptor := reviewReworkDescriptorForTestV0("delivery-ref-target", "task-ref-target")
	source := ReviewReworkReplanSourceV0{
		Store: orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(descriptor),
	}
	request := reviewReworkPlanRequestForTestV0(false)
	request.Run.FunctionContracts = []string{"BuildAutoprogrammingProgrammableWorkV0"}
	request.Run.Agents = []string{descriptor.AgentRef}
	request.Run.StartedAgents = []string{descriptor.AgentRef}

	plans, err := source.BuildReviewReworkReplanPlansV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildReviewReworkReplanPlansV0: %v", err)
	}
	if len(plans) != 0 {
		t.Fatalf("no debe crear otro agente padre para la misma tarea: %+v", plans)
	}
}

func TestReviewReworkReplanSourceV0CreaTareaCorreccionSiYaHayPadreYWriteSet(t *testing.T) {
	descriptor := reviewReworkDescriptorForTestV0("delivery-ref-target", "task-ref-target")
	descriptor.Spec.AgentPacket.Task.WriteSet = []string{"modulos/orquesta-app-codex-stack"}
	descriptor.Spec.AgentPacket.Task.RequiredTests = []string{"go test -count=1 ./modulos/orquesta-app-codex-stack"}
	source := ReviewReworkReplanSourceV0{
		Store: orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(descriptor),
	}
	request := reviewReworkPlanRequestForTestV0(false)
	request.Run.FunctionContracts = []string{"BuildAutoprogrammingProgrammableWorkV0"}
	request.Run.Agents = []string{descriptor.AgentRef}
	request.Run.StartedAgents = []string{descriptor.AgentRef}

	plans, err := source.BuildReviewReworkReplanPlansV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildReviewReworkReplanPlansV0: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("plans=%+v", plans)
	}
	plan := plans[0]
	if plan.RequestedAction != orquestacorereplanner.ReplanActionSplitTaskV0 ||
		plan.AgentRequestID != "" ||
		plan.CapacityRequestRef != "" ||
		len(plan.SplitTasks) != 1 {
		t.Fatalf("plan debe crear split_task sin segundo padre: %+v", plan)
	}
	task := plan.SplitTasks[0]
	if task.TaskID == "" ||
		!reviewReworkReplanStringInSetV0(task.DependsOn, "task-ref-target") ||
		!reviewReworkReplanStringInSetV0(task.RequiredTests, "go test -count=1 ./modulos/orquesta-app-codex-stack") ||
		!reviewReworkPlanHasEvidenceForTestV0(plan.EvidenceRefs, "evidence-ref-review-rework-task-boundary") ||
		len(task.FunctionContractRefs) != 1 ||
		task.FunctionContractRefs[0].ContractRef != "BuildAutoprogrammingProgrammableWorkV0" {
		t.Fatalf("tarea de correccion sin causalidad/evidencia: plan=%+v task=%+v", plan, task)
	}
}

func TestReviewReworkReplanSourceV0UsaAgentRefCortoYEstableParaReworkReal(t *testing.T) {
	longDeliveryRef := "ack-ref-app-stack-" + strings.Repeat("delivery-ref-real-review-rework-", 8)
	longTaskRef := "task-ref-app-change-" + strings.Repeat("review-rework-task-", 6)
	descriptor := reviewReworkDescriptorForTestV0(longDeliveryRef, longTaskRef)
	source := ReviewReworkReplanSourceV0{
		Store: orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(descriptor),
	}
	request := reviewReworkPlanRequestForTestV0(false)
	request.Run.Tasks = []string{longTaskRef}
	request.Run.ReworkRequests = []string{
		"rework-request-ref-" + strings.Repeat("review-result-real-damaged-delivery-", 6) +
			"#review_result:review-result-ref-long#review_request:review-request-ref-long#delivery:" +
			longDeliveryRef,
	}
	request.Run.ReviewResults = []string{
		"review-result-ref-long#review_result:changes_requested" +
			"#review_request:review-request-ref-long#delivery:" + longDeliveryRef,
	}

	plans, err := source.BuildReviewReworkReplanPlansV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildReviewReworkReplanPlansV0: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("plans=%+v", plans)
	}
	if plans[0].TaskRef != longTaskRef {
		t.Fatalf("task_ref=%q", plans[0].TaskRef)
	}
	if len(plans[0].AgentRequestID) > 80 {
		t.Fatalf("agent_request_id demasiado largo: %s", plans[0].AgentRequestID)
	}
	if !strings.HasPrefix(plans[0].AgentRequestID, "agent-ref-task-ref-app-change-") {
		t.Fatalf("agent_request_id no conserva tarea: %s", plans[0].AgentRequestID)
	}

	request.Run.Agents = []string{plans[0].AgentRequestID}
	again, err := source.BuildReviewReworkReplanPlansV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildReviewReworkReplanPlansV0 replay: %v", err)
	}
	if len(again) != 0 {
		t.Fatalf("replay no debe duplicar agente de rework: %+v", again)
	}
}

func TestReviewReworkReplanSourceV0CortaBucleTrasRetriesAmpliosPorTarea(t *testing.T) {
	taskRef := "task-ref-target"
	source := ReviewReworkReplanSourceV0{
		Store: orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(
			reviewReworkDescriptorForTestV0("delivery-ref-target", taskRef),
		),
	}
	request := reviewReworkPlanRequestForTestV0(false)
	request.Run.StartedAgents = make([]string, 0, reviewReworkReplanMaxRetryAgentsPerTaskV0)
	for idx := 0; idx < reviewReworkReplanMaxRetryAgentsPerTaskV0; idx++ {
		rework := reviewReworkProjectionV0{
			ReworkRequestRef: "rework-request-ref-target-" + string(rune('a'+idx)),
			ReviewResultRef:  "review-result-ref-target-" + string(rune('a'+idx)),
			ReviewRequestID:  "review-request-ref-target-" + string(rune('a'+idx)),
			DeliveryRef:      "delivery-ref-target-" + string(rune('a'+idx)),
		}
		request.Run.StartedAgents = append(
			request.Run.StartedAgents,
			"agent-ref-"+reviewReworkReplanAgentSuffixV0(rework, taskRef),
		)
	}

	plans, err := source.BuildReviewReworkReplanPlansV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildReviewReworkReplanPlansV0: %v", err)
	}
	if len(plans) != 0 {
		t.Fatalf("no debe encadenar rework indefinido tras limite amplio: %+v", plans)
	}
}

func TestReviewReworkReplanSourceV0ReplanificaAunqueLaEntregaOriginalTengaAckCompletado(t *testing.T) {
	descriptor := reviewReworkDescriptorForTestV0("delivery-ref-target", "task-ref-target")
	descriptor.AckPath = writeReviewReworkAckForTestV0(t, descriptor.AgentRef, "task-ref-target", "completed")
	source := ReviewReworkReplanSourceV0{
		Store: orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(descriptor),
	}

	plans, err := source.BuildReviewReworkReplanPlansV0(
		context.Background(),
		reviewReworkPlanRequestForTestV0(false),
	)
	if err != nil {
		t.Fatalf("BuildReviewReworkReplanPlansV0: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("debe replanificar entrega original rechazada aunque el ACK inicial este completado: %+v", plans)
	}
	if plans[0].TaskRef != "task-ref-target" || plans[0].AgentRequestID == descriptor.AgentRef {
		t.Fatalf("plan de rework debe apuntar a la tarea original con agente followup nuevo: %+v", plans[0])
	}
}

func TestReviewReworkReplanSourceV0NoRelanzaSplitTaskAceptado(t *testing.T) {
	original := reviewReworkDescriptorForTestV0("delivery-ref-target", "task-ref-target")
	followup := reviewReworkDescriptorForTestV0("delivery-ref-followup", "task-ref-followup")
	source := ReviewReworkReplanSourceV0{
		Store: orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(original, followup),
	}
	request := reviewReworkPlanRequestForTestV0(false)
	request.Run.ReplanDecisions = []string{
		"replan-ref-target-split#source:rework-request-ref-target#task:task-ref-target" +
			"#action:split_task#followups:task-ref-followup",
	}
	request.Run.ClosedTasks = []string{"task-ref-followup"}
	request.Run.Deliveries = []string{"delivery-ref-followup"}
	request.Run.AcceptedReviews = []string{"accepted-review-ref-delivery-ref-followup"}

	plans, err := source.BuildReviewReworkReplanPlansV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildReviewReworkReplanPlansV0: %v", err)
	}
	if len(plans) != 0 {
		t.Fatalf("no debe relanzar rework split_task ya aceptado: %+v", plans)
	}
}

func TestReviewReworkReplanSourceV0NoRelanzaRetryAceptado(t *testing.T) {
	original := reviewReworkDescriptorForTestV0("delivery-ref-target", "task-ref-target")
	followup := reviewReworkDescriptorForTestV0("delivery-ref-followup", "task-ref-followup")
	source := ReviewReworkReplanSourceV0{
		Store: orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(original, followup),
	}
	request := reviewReworkPlanRequestForTestV0(false)
	request.Run.ReplanDecisions = []string{
		"replan-ref-target-retry#source:rework-request-ref-target#task:task-ref-target" +
			"#action:retry_task#followups:capacity-ref-followup+" + followup.AgentRef,
	}
	request.Run.Deliveries = []string{"delivery-ref-followup"}
	request.Run.AcceptedReviews = []string{"accepted-review-ref-delivery-ref-followup"}

	plans, err := source.BuildReviewReworkReplanPlansV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildReviewReworkReplanPlansV0: %v", err)
	}
	if len(plans) != 0 {
		t.Fatalf("no debe relanzar rework retry_task ya aceptado: %+v", plans)
	}
}

func TestReviewReworkReplanSourceV0NoRelanzaBucleDocumentalPorGoTestGlobalNoEjecutable(t *testing.T) {
	descriptor := reviewReworkDescriptorForTestV0("delivery-ref-target", "task-ref-target")
	descriptor.Spec.AgentPacket.Task.WriteSet = []string{
		"docs/autoprogramacion_orquesta_pendientes_2026-05-23.md",
		"docs/runbooks",
	}
	descriptor.Spec.AgentPacket.Task.RequiredTests = []string{"go test -count=1 ./..."}
	descriptor.AckPath = writeReviewReworkExternalRailAckForTestV0(t, descriptor.AgentRef, "task-ref-target")
	source := ReviewReworkReplanSourceV0{
		Store: orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(descriptor),
	}

	plans, err := source.BuildReviewReworkReplanPlansV0(
		context.Background(),
		reviewReworkPlanRequestForTestV0(false),
	)
	if err != nil {
		t.Fatalf("BuildReviewReworkReplanPlansV0: %v", err)
	}
	if len(plans) != 0 {
		t.Fatalf("no debe relanzar rework documental por rail externo no ejecutable: %+v", plans)
	}
}

func TestReviewReworkReplanSourceV0DescribeWriteSetFaltanteParaElAgente(t *testing.T) {
	projectDir := t.TempDir()
	writeStackReviewGateFileForTestV0(t, projectDir, "internal/api/handler.go", "package api\n")
	descriptor := reviewReworkDescriptorForTestV0("delivery-ref-target", "task-ref-target")
	descriptor.ProjectWorkDir = projectDir
	descriptor.Spec.AgentPacket.Task.WriteSet = []string{"internal/api", "web"}
	source := ReviewReworkReplanSourceV0{
		Store: orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(descriptor),
	}

	plans, err := source.BuildReviewReworkReplanPlansV0(
		context.Background(),
		reviewReworkPlanRequestForTestV0(false),
	)
	if err != nil {
		t.Fatalf("BuildReviewReworkReplanPlansV0: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("plans=%+v", plans)
	}
	if !strings.Contains(plans[0].Summary, "completar faltantes: web") {
		t.Fatalf("summary=%q", plans[0].Summary)
	}
	if !reviewReworkPlanHasEvidenceForTestV0(plans[0].EvidenceRefs, "review-rework-missing-web") {
		t.Fatalf("evidence_refs=%v", plans[0].EvidenceRefs)
	}
}

func TestBuildStackV0CableaReviewReworkReplanSource(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	if stack.Ports.ReviewReworkReplanSource == nil {
		t.Fatalf("ReviewReworkReplanSource no cableado")
	}
}

func writeReviewReworkExternalRailAckForTestV0(t interface {
	Helper()
	Fatalf(string, ...any)
	TempDir() string
}, agentRef string, taskRef string) string {
	t.Helper()
	path := t.TempDir() + "/" + orquestaruntimecodex.CodexAgentAckFileNameV0
	data := []byte(`{
		"schema_version":"codex_agent_ack.v0",
		"request_id":"` + agentRef + `",
		"correlation_id":"corr-review-rework-ack-test",
		"ack_ref":"delivery-ref-target",
		"target_module":"orquesta-app-codex-stack",
		"task_ref":"` + taskRef + `",
		"status":"completed",
		"files":["docs/autoprogramacion_orquesta_pendientes_2026-05-23.md"],
		"test_receipts":[{
			"schema_version":"codex_required_test_receipt.v0",
			"command":"go test -count=1 ./...",
			"status":"failed",
			"exit_code":75,
			"evidence_refs":["GOCACHE read-only; httptest socket denied; codex wrapper exit 75"],
			"occurred_at":"2026-06-11T10:00:00Z",
			"sequence":1,
			"output_redacted":true
		}],
		"notes":["solo fallo de entorno/sandbox, entrega documental completada"]
	}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write ack: %v", err)
	}
	return path
}
