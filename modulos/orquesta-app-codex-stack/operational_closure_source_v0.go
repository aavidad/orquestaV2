package orquestaappcodexstack

import (
	"context"
	"strings"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

type codexStackOperationalClosureSourceV0 struct {
	TaskStore                  orquestacionnucleoapp.WorkflowTaskStorePortV0
	EventReader                orquestacionnucleoapp.RunEventReaderPortV0
	RequiredTestEvidenceStore  orquestacionnucleoapp.RequiredTestEvidenceReaderPortV0
	RequiredTestEvidenceWriter orquestacionnucleoapp.RequiredTestEvidenceWriterPortV0
	ReceiptStore               orquestaruntimecodexdelivery.CodexReceiptDescriptorStorePortV0
	RuntimeWorkDir             string
	AppChangeStore             orquestaappchange.AppChangeRecordSourcePortV0
	DomainSubmissionLedger     DomainWorkArtifactSubmissionRecordReaderPortV0
}

var _ orquestaappdirectorservice.AppDirectorOperationalClosureSourcePortV0 = codexStackOperationalClosureSourceV0{}

func operationalClosureSourceV0(
	config ConfigV0,
) orquestaappdirectorservice.AppDirectorOperationalClosureSourcePortV0 {
	reader, _ := config.Stores.EventSink.(orquestacionnucleoapp.RunEventReaderPortV0)
	if config.Stores.TaskStore == nil || reader == nil {
		return nil
	}
	requiredTestEvidenceStore := requiredTestEvidenceStoreV0(config)
	return codexStackOperationalClosureSourceV0{
		TaskStore:                  config.Stores.TaskStore,
		EventReader:                reader,
		RequiredTestEvidenceStore:  requiredTestEvidenceStore,
		RequiredTestEvidenceWriter: requiredTestEvidenceStore,
		ReceiptStore:               config.Stores.ReceiptStore,
		RuntimeWorkDir:             config.Codex.RuntimeWorkDir,
		AppChangeStore:             config.Stores.AppChangeStore,
		DomainSubmissionLedger:     domainWorkSubmissionRecordReaderV0(config.DomainDelivery.Ledger),
	}
}

func eventReaderV0(config ConfigV0) orquestacionnucleoapp.RunEventReaderPortV0 {
	reader, _ := config.Stores.EventSink.(orquestacionnucleoapp.RunEventReaderPortV0)
	return reader
}

func (source codexStackOperationalClosureSourceV0) BuildOperationalDirectorClosureRequestV0(
	ctx context.Context,
	request orquestaappdirectorservice.AppDirectorOperationalClosureRequestV0,
) (orquestacionnucleoapp.OperationalDirectorClosureRequestV0, bool, error) {
	if source.TaskStore == nil || source.EventReader == nil ||
		strings.TrimSpace(request.Run.RunID) == "" ||
		request.Run.Status != orquestacoreworkflow.OrchestrationRunStatusActiveV0 ||
		len(request.Run.Tasks) == 0 {
		return orquestacionnucleoapp.OperationalDirectorClosureRequestV0{}, false, nil
	}
	tasks, err := source.TaskStore.LoadWorkflowTasksV0(ctx, request.Run.RunID, request.Run.Tasks)
	if err != nil {
		return orquestacionnucleoapp.OperationalDirectorClosureRequestV0{}, false, err
	}
	events, err := source.EventReader.LoadRunEventsV0(ctx, request.Run.RunID)
	if err != nil {
		return orquestacionnucleoapp.OperationalDirectorClosureRequestV0{}, false, err
	}
	trace := codexStackOperationalClosureTraceFromEventsAndRunV0(events, request.Run)
	candidates, err := source.codexStackOperationalClosureCandidateTasksV0(ctx, request, trace, tasks)
	if err != nil {
		return orquestacionnucleoapp.OperationalDirectorClosureRequestV0{}, false, err
	}
	for _, task := range candidates {
		closureRequest, ok, err := source.codexStackOperationalClosureRequestForTaskV0(ctx, request, trace, task)
		if err != nil {
			return orquestacionnucleoapp.OperationalDirectorClosureRequestV0{}, false, err
		}
		if ok {
			return closureRequest, true, nil
		}
	}
	return orquestacionnucleoapp.OperationalDirectorClosureRequestV0{}, false, nil
}

type codexStackOperationalClosureTraceV0 struct {
	Deliveries       map[string]orquestacoreworkflow.DeliveryRegisteredPayloadV0
	DeliveryRefs     []string
	ReviewRequests   map[string]orquestacoreworkflow.ReviewRequestedPayloadV0
	ReviewResults    map[string]orquestacoreworkflow.ReviewResultV0
	ReviewResultRefs []string
	AcceptedReviews  map[string]orquestacoreworkflow.ReviewAcceptedPayloadV0
}

func (source codexStackOperationalClosureSourceV0) codexStackOperationalClosureCandidateTasksV0(
	ctx context.Context,
	request orquestaappdirectorservice.AppDirectorOperationalClosureRequestV0,
	trace codexStackOperationalClosureTraceV0,
	tasks []orquestacoreworkflow.WorkflowTaskV0,
) ([]orquestacoreworkflow.WorkflowTaskV0, error) {
	scopeAgents := codexStackOperationalClosureSetV0(request.WaitAgentRefs)
	if request.WaitScopeApplied && len(scopeAgents) == 0 {
		return nil, nil
	}
	open := make([]orquestacoreworkflow.WorkflowTaskV0, 0, len(tasks))
	closed := make([]orquestacoreworkflow.WorkflowTaskV0, 0, len(tasks))
	for _, task := range tasks {
		closable, err := source.codexStackOperationalClosureTaskIsClosableV0(ctx, request, task, tasks)
		if err != nil {
			return nil, err
		}
		if !closable {
			continue
		}
		if !codexStackOperationalClosureChildrenClosedV0(request, task, tasks) {
			continue
		}
		if len(scopeAgents) > 0 && !codexStackOperationalClosureTaskMatchesScopeV0(request.Run, trace, task, tasks, scopeAgents) {
			continue
		}
		if codexStackOperationalClosureContainsV0(request.Run.ClosedTasks, task.TaskID) {
			closed = append(closed, task)
			continue
		}
		open = append(open, task)
	}
	if len(open) > 0 {
		return open, nil
	}
	return closed, nil
}

func codexStackOperationalClosureTaskMatchesScopeV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	trace codexStackOperationalClosureTraceV0,
	task orquestacoreworkflow.WorkflowTaskV0,
	tasks []orquestacoreworkflow.WorkflowTaskV0,
	scopeAgents map[string]bool,
) bool {
	if len(scopeAgents) == 0 {
		return true
	}
	if scopeAgents[orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(task.TaskID)] {
		return true
	}
	if codexStackOperationalClosureTaskDeliveryMatchesScopeV0(run, trace, task.TaskID, scopeAgents) {
		return true
	}
	tasksByRef := make(map[string]orquestacoreworkflow.WorkflowTaskV0, len(tasks))
	for _, item := range tasks {
		tasksByRef[strings.TrimSpace(item.TaskID)] = item
	}
	if codexStackOperationalClosureAncestorMatchesScopeV0(run, trace, task, tasksByRef, scopeAgents, map[string]bool{}) {
		return true
	}
	return codexStackOperationalClosureDescendantMatchesScopeV0(run, task, tasksByRef, scopeAgents, map[string]bool{})
}

func codexStackOperationalClosureAncestorMatchesScopeV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	trace codexStackOperationalClosureTraceV0,
	task orquestacoreworkflow.WorkflowTaskV0,
	tasksByRef map[string]orquestacoreworkflow.WorkflowTaskV0,
	scopeAgents map[string]bool,
	seen map[string]bool,
) bool {
	for _, parentRef := range codexStackOperationalClosureParentTaskRefsV0(run, task, tasksByRef) {
		if seen[parentRef] {
			continue
		}
		seen[parentRef] = true
		parent, ok := tasksByRef[parentRef]
		if !ok {
			continue
		}
		if scopeAgents[orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(parent.TaskID)] ||
			codexStackOperationalClosureTaskDeliveryMatchesScopeV0(run, trace, parent.TaskID, scopeAgents) {
			return true
		}
		if codexStackOperationalClosureAncestorMatchesScopeV0(run, trace, parent, tasksByRef, scopeAgents, seen) {
			return true
		}
	}
	return false
}

func codexStackOperationalClosureParentTaskRefsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	task orquestacoreworkflow.WorkflowTaskV0,
	tasksByRef map[string]orquestacoreworkflow.WorkflowTaskV0,
) []string {
	taskRef := strings.TrimSpace(task.TaskID)
	refs := []string{}
	for _, candidate := range tasksByRef {
		if !codexStackOperationalClosureContainsV0(codexStackOperationalClosureReplanFollowupRefsV0(run, candidate.TaskID), taskRef) {
			continue
		}
		refs = append(refs, candidate.TaskID)
	}
	return codexStackOperationalClosureCompactRefsV0(refs)
}

func codexStackOperationalClosureReplanFollowupRefsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	taskRef string,
) []string {
	taskRef = strings.TrimSpace(taskRef)
	refs := []string{}
	for _, rawReplan := range run.ReplanDecisions {
		replan, ok := codexStackReviewGateParseReplanProjectionV0(rawReplan)
		if !ok ||
			strings.TrimSpace(replan.TaskRef) != taskRef ||
			replan.AcceptedAction != string(orquestacoreworkflow.ReplanDecisionActionSplitTaskV0) {
			continue
		}
		refs = append(refs, replan.FollowupRefs...)
	}
	return codexStackOperationalClosureCompactRefsV0(refs)
}

func codexStackOperationalClosureTaskDeliveryMatchesScopeV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	trace codexStackOperationalClosureTraceV0,
	taskRef string,
	scopeAgents map[string]bool,
) bool {
	taskRef = strings.TrimSpace(taskRef)
	if taskRef == "" || !codexStackOperationalClosureContainsV0(run.DeliveredTasks, taskRef) {
		return false
	}
	for _, deliveryRef := range trace.DeliveryRefs {
		delivery := trace.Deliveries[deliveryRef]
		if strings.TrimSpace(delivery.TaskID) != taskRef ||
			!codexStackOperationalClosureContainsV0(run.Deliveries, delivery.DeliveryRef) {
			continue
		}
		agentRef := strings.TrimSpace(delivery.AgentRef)
		if agentRef == "" || !scopeAgents[agentRef] {
			continue
		}
		if !codexStackOperationalClosureContainsV0(run.DeliveredAgents, agentRef) {
			continue
		}
		return true
	}
	return false
}

func codexStackOperationalClosureDescendantMatchesScopeV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	task orquestacoreworkflow.WorkflowTaskV0,
	tasksByRef map[string]orquestacoreworkflow.WorkflowTaskV0,
	scopeAgents map[string]bool,
	seen map[string]bool,
) bool {
	for _, childRef := range codexStackOperationalClosureChildTaskRefsV0(run, task) {
		if seen[childRef] {
			continue
		}
		seen[childRef] = true
		if scopeAgents[orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(childRef)] {
			return true
		}
		child, ok := tasksByRef[childRef]
		if ok && codexStackOperationalClosureDescendantMatchesScopeV0(run, child, tasksByRef, scopeAgents, seen) {
			return true
		}
	}
	return false
}

func (source codexStackOperationalClosureSourceV0) codexStackOperationalClosureTaskIsClosableV0(
	ctx context.Context,
	request orquestaappdirectorservice.AppDirectorOperationalClosureRequestV0,
	task orquestacoreworkflow.WorkflowTaskV0,
	tasks []orquestacoreworkflow.WorkflowTaskV0,
) (bool, error) {
	if codexStackOperationalClosureTaskIsOperationalDirectorV0(task) ||
		codexStackWorkflowTaskLooksAutoprogrammingV0(task) ||
		codexStackOperationalClosureTaskIsOperationalReplanFollowupV0(request.Run, task, tasks) {
		return true, nil
	}
	if appChangeExternal, err := source.codexStackOperationalClosureTaskIsAppChangeExternalWorkV0(ctx, request.Run.RunID, task); err != nil || appChangeExternal {
		if err != nil {
			return false, err
		}
		materialized, err := source.codexStackOperationalClosureOPESSubrolesMaterializedV0(ctx, request.Run, task, tasks)
		if err != nil {
			return false, err
		}
		return materialized, nil
	}
	return false, nil
}

func codexStackOperationalClosureTaskIsOperationalReplanFollowupV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	task orquestacoreworkflow.WorkflowTaskV0,
	tasks []orquestacoreworkflow.WorkflowTaskV0,
) bool {
	taskRef := strings.TrimSpace(task.TaskID)
	if taskRef == "" {
		return false
	}
	tasksByRef := make(map[string]orquestacoreworkflow.WorkflowTaskV0, len(tasks))
	for _, item := range tasks {
		tasksByRef[strings.TrimSpace(item.TaskID)] = item
	}
	for _, rawReplan := range run.ReplanDecisions {
		replan, ok := codexStackReviewGateParseReplanProjectionV0(rawReplan)
		if !ok ||
			replan.AcceptedAction != string(orquestacoreworkflow.ReplanDecisionActionSplitTaskV0) ||
			!codexStackOperationalClosureContainsV0(replan.FollowupRefs, taskRef) {
			continue
		}
		parent, ok := tasksByRef[strings.TrimSpace(replan.TaskRef)]
		if !ok {
			continue
		}
		if codexStackWorkflowTaskLooksOperationalDirectorV0(parent) ||
			codexStackWorkflowTaskLooksAutoprogrammingV0(parent) {
			return true
		}
	}
	return false
}

func codexStackOperationalClosureTaskIsOperationalDirectorV0(
	task orquestacoreworkflow.WorkflowTaskV0,
) bool {
	return codexStackWorkflowTaskLooksOperationalDirectorV0(task)
}

func codexStackOperationalClosureChildrenClosedV0(
	request orquestaappdirectorservice.AppDirectorOperationalClosureRequestV0,
	task orquestacoreworkflow.WorkflowTaskV0,
	tasks []orquestacoreworkflow.WorkflowTaskV0,
) bool {
	childRefs := codexStackOperationalClosureChildTaskRefsV0(request.Run, task)
	if len(childRefs) == 0 {
		return true
	}
	tasksByRef := make(map[string]bool, len(tasks))
	for _, item := range tasks {
		tasksByRef[strings.TrimSpace(item.TaskID)] = true
	}
	for _, childRef := range childRefs {
		if !tasksByRef[childRef] || !codexStackOperationalClosureContainsV0(request.Run.ClosedTasks, childRef) {
			return false
		}
	}
	return true
}

func codexStackOperationalClosureChildTaskRefsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	task orquestacoreworkflow.WorkflowTaskV0,
) []string {
	refs := append([]string(nil), task.ChildTaskRefs...)
	taskRef := strings.TrimSpace(task.TaskID)
	for _, rawReplan := range run.ReplanDecisions {
		replan, ok := codexStackReviewGateParseReplanProjectionV0(rawReplan)
		if !ok ||
			strings.TrimSpace(replan.TaskRef) != taskRef ||
			replan.AcceptedAction != string(orquestacoreworkflow.ReplanDecisionActionSplitTaskV0) {
			continue
		}
		refs = append(refs, replan.FollowupRefs...)
	}
	return codexStackOperationalClosureCompactRefsV0(refs)
}

func (source codexStackOperationalClosureSourceV0) codexStackOperationalClosureRequestForTaskV0(
	ctx context.Context,
	request orquestaappdirectorservice.AppDirectorOperationalClosureRequestV0,
	trace codexStackOperationalClosureTraceV0,
	task orquestacoreworkflow.WorkflowTaskV0,
) (orquestacionnucleoapp.OperationalDirectorClosureRequestV0, bool, error) {
	for _, delivery := range codexStackOperationalClosureDeliveriesForTaskV0(request, trace, task) {
		reviewRequest, accepted, result, ok := codexStackOperationalClosureAcceptedReviewForDeliveryV0(request, trace, delivery.DeliveryRef)
		if !ok {
			continue
		}
		if codexStackOperationalClosureDeliveryHasAgentBlockedGateV0(delivery, reviewRequest, accepted, result) &&
			!codexStackOperationalClosureRunHasExplicitAgentBlockedResolutionV0(request.Run, task.TaskID, delivery.DeliveryRef) {
			continue
		}
		requiredTestEvidenceRefs := []string(nil)
		if len(task.RequiredTests) > 0 {
			if source.RequiredTestEvidenceStore == nil {
				continue
			}
			evidence, err := source.codexStackOperationalClosureLoadRequiredTestEvidenceForResultV0(ctx, request, result)
			if err != nil {
				return orquestacionnucleoapp.OperationalDirectorClosureRequestV0{}, false, err
			}
			requiredTestEvidenceRefs = codexStackOperationalClosurePassedTestEvidenceRefsV0(task, evidence, accepted, result)
			if len(requiredTestEvidenceRefs) == 0 {
				requiredTestEvidenceRefs, err = source.codexStackOperationalClosureEnsureAckRequiredTestEvidenceRefsV0(
					ctx,
					request,
					task,
					delivery,
					reviewRequest,
					accepted,
					result,
				)
				if err != nil {
					return orquestacionnucleoapp.OperationalDirectorClosureRequestV0{}, false, err
				}
			}
			if len(requiredTestEvidenceRefs) == 0 {
				continue
			}
		}
		domainRefs, requiredDomainReceipt, ok, err := source.codexStackOperationalClosureDomainWorkEvidenceRefsForDeliveryV0(
			ctx,
			request,
			task,
			delivery.DeliveryRef,
		)
		if err != nil {
			return orquestacionnucleoapp.OperationalDirectorClosureRequestV0{}, false, err
		}
		if requiredDomainReceipt && !ok {
			continue
		}
		evidenceRefs := codexStackOperationalClosureEvidenceRefsV0(request, delivery, reviewRequest, accepted, result)
		evidenceRefs = codexStackOperationalClosureCompactRefsV0(append(evidenceRefs, domainRefs...))
		return orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
			RunRef:                   request.Run.RunID,
			TaskID:                   task.TaskID,
			DeliveryRef:              delivery.DeliveryRef,
			AcceptedReviewRef:        accepted.AcceptedReviewRef,
			ValidationRef:            "validation-ref-operational-director-" + codexStackOperationalClosureSafeRefV0(task.TaskID),
			ClosureRef:               "closure-ref-operational-director-" + codexStackOperationalClosureSafeRefV0(request.Run.RunID),
			OccurredAt:               request.OccurredAt,
			CorrelationID:            request.CorrelationID,
			RequestedBy:              request.RequestedBy,
			Summary:                  "Cierre causal generado desde el stack de Orquesta.",
			RequiredTestEvidenceRefs: requiredTestEvidenceRefs,
			EvidenceRefs:             evidenceRefs,
		}, true, nil
	}
	return orquestacionnucleoapp.OperationalDirectorClosureRequestV0{}, false, nil
}

func codexStackOperationalClosureDeliveryHasAgentBlockedGateV0(
	delivery orquestacoreworkflow.DeliveryRegisteredPayloadV0,
	reviewRequest orquestacoreworkflow.ReviewRequestedPayloadV0,
	accepted orquestacoreworkflow.ReviewAcceptedPayloadV0,
	result orquestacoreworkflow.ReviewResultV0,
) bool {
	for _, refs := range [][]string{
		delivery.EvidenceRefs,
		reviewRequest.EvidenceRefs,
		accepted.EvidenceRefs,
		result.EvidenceRefs,
	} {
		if codexStackOperationalClosureContainsV0(refs, "gate-issue:agent_blocked") {
			return true
		}
	}
	return false
}

func codexStackOperationalClosureRunHasExplicitAgentBlockedResolutionV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	taskRef string,
	deliveryRef string,
) bool {
	taskRef = strings.TrimSpace(taskRef)
	for _, rawReplan := range run.ReplanDecisions {
		replan, ok := codexStackReviewGateParseReplanProjectionV0(rawReplan)
		if !ok || strings.TrimSpace(replan.TaskRef) != taskRef {
			continue
		}
		return true
	}
	return false
}
