package orquestaappdirectorservice

import (
	"context"
	"strings"

	orquestaappdirectorintake "orquesta/modulos/orquesta-app-director-intake"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func startAppDirectorRequestWithOperationalPlanRunRefV0(
	request StartAppDirectorRequestV0,
) StartAppDirectorRequestV0 {
	if !continueHasOperationalDirectorPlanV0(request.OperationalDirectorPlan) {
		return request
	}
	plan := request.OperationalDirectorPlan
	plan.PlanRef = strings.TrimSpace(plan.PlanRef)
	plan.RequestRef = strings.TrimSpace(plan.RequestRef)
	plan.RunRef = strings.TrimSpace(plan.RunRef)
	plan.ProjectRef = strings.TrimSpace(plan.ProjectRef)
	request.OperationalDirectorPlan = plan
	if strings.TrimSpace(request.RunRef) == "" && plan.RunRef != "" {
		request.RunRef = plan.RunRef
	}
	return request
}

func startAppDirectorWithOperationalDirectorPlanV0(
	ctx context.Context,
	request StartAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	prepared orquestaappdirectorintake.AppDirectorIntakePreparedV0,
) (StartAppDirectorRequestV0, error) {
	if !continueHasOperationalDirectorPlanV0(request.OperationalDirectorPlan) {
		return request, nil
	}
	runRef := strings.TrimSpace(prepared.Run.RunID)
	if runRef == "" {
		return StartAppDirectorRequestV0{}, AppDirectorServiceIssueV0{Field: "run_ref"}
	}
	request.RunRef = runRef
	plan := request.OperationalDirectorPlan
	if strings.TrimSpace(plan.RunRef) == "" {
		plan.RunRef = runRef
	}
	if strings.TrimSpace(plan.RunRef) != runRef {
		return StartAppDirectorRequestV0{}, AppDirectorServiceIssueV0{Field: "operational_director_plan.run_ref"}
	}
	request.OperationalDirectorPlan = plan
	request.OperationalDirectorPlanRef = strings.TrimSpace(plan.PlanRef)
	functionContractRefs := startAppDirectorOperationalFunctionContractRefsV0(request)
	if len(functionContractRefs) == 0 {
		return StartAppDirectorRequestV0{}, AppDirectorServiceIssueV0{Field: "operational_director_function_contract_refs"}
	}
	request.OperationalDirectorFunctionContractRefs = functionContractRefs
	targetPhaseID := startAppDirectorOperationalTargetPhaseIDV0(request)
	if err := startAppDirectorPrepareOperationalDirectorWorkflowV0(ctx, request, ports, prepared, targetPhaseID); err != nil {
		return StartAppDirectorRequestV0{}, err
	}
	materializeRequest := startOperationalDirectorContinueRequestV0(request, targetPhaseID)
	materialized, err := materializeContinueOperationalDirectorPlanV0(ctx, materializeRequest, ports)
	if err != nil {
		return StartAppDirectorRequestV0{}, err
	}
	continued := continueRequestWithOperationalDirectorWaitV0(materializeRequest, materialized)
	request.WaitWaveRef = continued.WaitWaveRef
	request.WaitCohortRef = continued.WaitCohortRef
	request.WaitParentTaskRef = continued.WaitParentTaskRef
	request.WaitAgentRefs = append([]string(nil), continued.WaitAgentRefs...)
	return startRequestWithOperationalDirectorPlanStateV0(ctx, request, ports)
}

func startAppDirectorPrepareOperationalDirectorWorkflowV0(
	ctx context.Context,
	request StartAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	prepared orquestaappdirectorintake.AppDirectorIntakePreparedV0,
	targetPhaseID orquestacoreworkflow.OrchestrationPhaseIDV0,
) error {
	voteRef := startAppDirectorOperationalVoteRefV0(request)
	decisionRef := startAppDirectorOperationalDecisionRefV0(request)
	commands, err := startAppDirectorOperationalBootstrapCommandsV0(request, prepared, targetPhaseID, voteRef, decisionRef)
	if err != nil {
		return err
	}
	for _, command := range commands {
		if _, err := orquestacionnucleoapp.HandleStoredWorkflowCommandV0(ctx, ports.RunStore, ports.EventSink, command); err != nil {
			return err
		}
	}
	return nil
}

func startAppDirectorOperationalBootstrapCommandsV0(
	request StartAppDirectorRequestV0,
	prepared orquestaappdirectorintake.AppDirectorIntakePreparedV0,
	targetPhaseID orquestacoreworkflow.OrchestrationPhaseIDV0,
	voteRef string,
	decisionRef string,
) ([]orquestacoreworkflow.OrchestrationCommandV0, error) {
	commands := make([]orquestacoreworkflow.OrchestrationCommandV0, 0, 4+len(request.OperationalDirectorFunctionContractRefs))
	openVote, err := orquestacoreworkflow.NewOpenPhaseCommandV0(
		startAppDirectorOperationalCommandMetaV0(request, "open-vote"),
		orquestacoreworkflow.OpenPhaseCommandPayloadV0{
			PhaseID: string(orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0),
			Reason:  "Arranque directo con plan operativo recibido.",
		},
	)
	if err != nil {
		return nil, err
	}
	commands = append(commands, openVote)
	vote, err := orquestacoreworkflow.NewRequestVoteCommandV0(
		startAppDirectorOperationalCommandMetaV0(request, "request-vote"),
		orquestacoreworkflow.RequestVoteCommandPayloadV0{
			VoteRequestID:              voteRef,
			PhaseID:                    string(orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0),
			DecisionTopicRef:           request.OperationalDirectorPlan.PlanRef,
			BrainstormRef:              prepared.DirectorTask.BrainstormRef,
			Summary:                    "Validar plan operativo recibido para arrancar subagentes.",
			MinimumRecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
			EvidenceRefs:               []string{"evidence-ref-app-director-operational-plan-start-v0"},
		},
	)
	if err != nil {
		return nil, err
	}
	commands = append(commands, vote)
	decision, err := orquestacoreworkflow.NewAcceptDecisionCommandV0(
		startAppDirectorOperationalCommandMetaV0(request, "accept-decision"),
		orquestacoreworkflow.AcceptDecisionCommandPayloadV0{
			DecisionRef:       decisionRef,
			PhaseID:           string(orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0),
			VoteRef:           voteRef,
			AcceptedOptionRef: "operational-director-plan-ready",
			Summary:           "Plan operativo recibido y aceptado para materializacion causal.",
			EvidenceRefs:      []string{"evidence-ref-app-director-operational-plan-start-v0"},
		},
	)
	if err != nil {
		return nil, err
	}
	commands = append(commands, decision)
	openTarget, err := orquestacoreworkflow.NewOpenPhaseCommandV0(
		startAppDirectorOperationalCommandMetaV0(request, "open-"+string(targetPhaseID)),
		orquestacoreworkflow.OpenPhaseCommandPayloadV0{
			PhaseID: string(targetPhaseID),
			Reason:  "Preparar materializacion causal del plan operativo.",
		},
	)
	if err != nil {
		return nil, err
	}
	commands = append(commands, openTarget)
	for _, ref := range startAppDirectorOperationalFunctionContractRefsV0(request) {
		publish, err := orquestacoreworkflow.NewPublishFunctionContractCommandV0(
			startAppDirectorOperationalCommandMetaV0(request, "publish-"+ref.ContractRef),
			orquestacoreworkflow.PublishFunctionContractCommandPayloadV0{
				ContractRef:   ref.ContractRef,
				PhaseID:       string(targetPhaseID),
				DecisionRef:   decisionRef,
				Summary:       "Contrato funcional para plan operativo directo.",
				FunctionNames: startAppDirectorOperationalFunctionNamesV0(ref),
				EvidenceRefs:  []string{"evidence-ref-app-director-operational-plan-start-v0"},
			},
		)
		if err != nil {
			return nil, err
		}
		commands = append(commands, publish)
	}
	return commands, nil
}

func startOperationalDirectorContinueRequestV0(
	request StartAppDirectorRequestV0,
	targetPhaseID orquestacoreworkflow.OrchestrationPhaseIDV0,
) ContinueAppDirectorRequestV0 {
	return ContinueAppDirectorRequestV0{
		RunRef:                                  request.RunRef,
		OccurredAt:                              request.OccurredAt,
		CorrelationID:                           request.CorrelationID,
		RequestedBy:                             request.RequestedBy,
		MaxBursts:                               request.MaxBursts,
		MaxStepsPerBurst:                        request.MaxStepsPerBurst,
		MaxDispatchesPerWait:                    request.MaxDispatchesPerWait,
		WaitAgentRefs:                           append([]string(nil), request.WaitAgentRefs...),
		WaitCohortRef:                           request.WaitCohortRef,
		WaitWaveRef:                             request.WaitWaveRef,
		WaitParentTaskRef:                       request.WaitParentTaskRef,
		MaxCommands:                             request.MaxCommands,
		MaxOutboxPerCycle:                       request.MaxOutboxPerCycle,
		MaxDecisionCycles:                       request.MaxDecisionCycles,
		MaxExternalWaits:                        request.MaxExternalWaits,
		OperationalDirectorPlanRef:              request.OperationalDirectorPlanRef,
		OperationalDirectorPlan:                 request.OperationalDirectorPlan,
		OperationalDirectorFunctionContractRefs: append([]orquestacoreworkflow.WorkflowFunctionContractRefV0(nil), request.OperationalDirectorFunctionContractRefs...),
		OperationalDirectorTargetPhaseID:        targetPhaseID,
		OperationalDirectorMaxItems:             request.OperationalDirectorMaxItems,
	}
}

func startAppDirectorOperationalTargetPhaseIDV0(
	request StartAppDirectorRequestV0,
) orquestacoreworkflow.OrchestrationPhaseIDV0 {
	if strings.TrimSpace(string(request.OperationalDirectorTargetPhaseID)) != "" {
		return request.OperationalDirectorTargetPhaseID
	}
	return orquestacoreworkflow.OrchestrationPhaseProgramacionV0
}

func startAppDirectorOperationalTargetPhaseAllowedV0(
	phaseID orquestacoreworkflow.OrchestrationPhaseIDV0,
) bool {
	switch phaseID {
	case orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0:
		return true
	default:
		return false
	}
}

func startAppDirectorOperationalFunctionContractRefsV0(
	request StartAppDirectorRequestV0,
) []orquestacoreworkflow.WorkflowFunctionContractRefV0 {
	refs := make([]orquestacoreworkflow.WorkflowFunctionContractRefV0, 0, len(request.OperationalDirectorFunctionContractRefs))
	seen := map[string]bool{}
	for _, ref := range request.OperationalDirectorFunctionContractRefs {
		ref.ContractRef = strings.TrimSpace(ref.ContractRef)
		ref.FunctionName = strings.TrimSpace(ref.FunctionName)
		if ref.ContractRef == "" || seen[ref.ContractRef] {
			continue
		}
		seen[ref.ContractRef] = true
		refs = append(refs, ref)
	}
	return refs
}

func startAppDirectorOperationalFunctionNamesV0(
	ref orquestacoreworkflow.WorkflowFunctionContractRefV0,
) []string {
	if strings.TrimSpace(ref.FunctionName) == "" {
		return nil
	}
	return []string{strings.TrimSpace(ref.FunctionName)}
}

func startAppDirectorOperationalCommandMetaV0(
	request StartAppDirectorRequestV0,
	kind string,
) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	suffix := appDirectorSafeRefPartV0(strings.Join(compactStartAppDirectorStringsV0([]string{
		"operational-start",
		request.RunRef,
		request.OperationalDirectorPlan.PlanRef,
		kind,
	}), "-"))
	return orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      "cmd-" + suffix,
		RunID:          request.RunRef,
		IdempotencyKey: "idem-" + suffix,
		CorrelationID:  request.CorrelationID,
		RequestedBy:    request.RequestedBy,
		OccurredAt:     request.OccurredAt,
	}
}

func startAppDirectorOperationalVoteRefV0(request StartAppDirectorRequestV0) string {
	return "vote-ref-" + appDirectorSafeRefPartV0(strings.Join(compactStartAppDirectorStringsV0([]string{
		request.RunRef,
		request.OperationalDirectorPlan.PlanRef,
		"operational-plan",
	}), "-"))
}

func startAppDirectorOperationalDecisionRefV0(request StartAppDirectorRequestV0) string {
	return "decision-ref-" + appDirectorSafeRefPartV0(strings.Join(compactStartAppDirectorStringsV0([]string{
		request.RunRef,
		request.OperationalDirectorPlan.PlanRef,
		"operational-plan",
	}), "-"))
}
