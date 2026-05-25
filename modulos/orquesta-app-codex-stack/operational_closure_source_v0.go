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
	trace := codexStackOperationalClosureTraceFromEventsV0(events)
	candidates, err := source.codexStackOperationalClosureCandidateTasksV0(ctx, request, tasks)
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
	Deliveries      map[string]orquestacoreworkflow.DeliveryRegisteredPayloadV0
	ReviewRequests  map[string]orquestacoreworkflow.ReviewRequestedPayloadV0
	ReviewResults   map[string]orquestacoreworkflow.ReviewResultV0
	AcceptedReviews map[string]orquestacoreworkflow.ReviewAcceptedPayloadV0
}

func (source codexStackOperationalClosureSourceV0) codexStackOperationalClosureCandidateTasksV0(
	ctx context.Context,
	request orquestaappdirectorservice.AppDirectorOperationalClosureRequestV0,
	tasks []orquestacoreworkflow.WorkflowTaskV0,
) ([]orquestacoreworkflow.WorkflowTaskV0, error) {
	scopeAgents := codexStackOperationalClosureSetV0(request.WaitAgentRefs)
	if request.WaitScopeApplied && len(scopeAgents) == 0 {
		return nil, nil
	}
	open := make([]orquestacoreworkflow.WorkflowTaskV0, 0, len(tasks))
	closed := make([]orquestacoreworkflow.WorkflowTaskV0, 0, len(tasks))
	for _, task := range tasks {
		closable, err := source.codexStackOperationalClosureTaskIsClosableV0(ctx, request, task)
		if err != nil {
			return nil, err
		}
		if !closable {
			continue
		}
		if !codexStackOperationalClosureChildrenClosedV0(request, task, tasks) {
			continue
		}
		if len(scopeAgents) > 0 && !codexStackOperationalClosureTaskMatchesScopeV0(request.Run, task, tasks, scopeAgents) {
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
	tasksByRef := make(map[string]orquestacoreworkflow.WorkflowTaskV0, len(tasks))
	for _, item := range tasks {
		tasksByRef[strings.TrimSpace(item.TaskID)] = item
	}
	return codexStackOperationalClosureDescendantMatchesScopeV0(run, task, tasksByRef, scopeAgents, map[string]bool{})
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
) (bool, error) {
	if codexStackOperationalClosureTaskIsOperationalDirectorV0(task) ||
		codexStackWorkflowTaskLooksAutoprogrammingV0(task) {
		return true, nil
	}
	return source.codexStackOperationalClosureTaskIsOPESDomainWorkV0(ctx, request.Run.RunID, task)
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
