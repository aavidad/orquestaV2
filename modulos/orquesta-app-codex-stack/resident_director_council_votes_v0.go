package orquestaappcodexstack

import (
	"context"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadecisioncouncil "orquesta/modulos/orquesta-decision-council"
	orquestadirectorcandidates "orquesta/modulos/orquesta-director-candidates"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

type DecisionCouncilConfigV0 struct {
	VoteSource DecisionCouncilVoteSourcePortV0
}

type DecisionCouncilVoteSourcePortV0 interface {
	BuildDecisionCouncilVotesV0(
		ctx context.Context,
		request DecisionCouncilVoteBuildRequestV0,
	) (DecisionCouncilVoteBuildResultV0, error)
}

type DecisionCouncilVoteBuildRequestV0 struct {
	Run          orquestacoreworkflow.OrchestrationRunV0
	Tasks        []orquestacoreworkflow.WorkflowTaskV0
	VoteTasks    []orquestacoreworkflow.WorkflowTaskV0
	TopicRef     string
	VoteRef      string
	DecisionRef  string
	OptionRefs   []string
	EvidenceRefs []string
}

type DecisionCouncilVoteBuildResultV0 struct {
	Votes               []orquestadecisioncouncil.CouncilVoteV0
	EvidenceRefs        []string
	PendingEvidenceRefs []string
}

func (source codexStackResidentBriefingSourceV0) shouldAcceptDecisionCouncilVotesV0(
	ctx context.Context,
	request orquestacionnucleoapp.ResidentDirectorBriefingBuildRequestV0,
) bool {
	if source.RunStore == nil || source.TaskStore == nil || source.DecisionCouncil.VoteSource == nil {
		return false
	}
	run, err := source.RunStore.LoadRunV0(ctx, request.RunRef)
	if err != nil {
		return false
	}
	if run.Status != orquestacoreworkflow.OrchestrationRunStatusActiveV0 ||
		run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0 ||
		stringInSetV0(run.Decisions, codexStackResidentCouncilDecisionRefV0(run.RunID)) {
		return false
	}
	if codexStackResidentCouncilProgrammingHoldV0(ctx, run, source.TaskStore) {
		return false
	}
	tasks, ok := codexStackResidentCouncilTasksV0(ctx, run, source.TaskStore)
	if !ok || !codexStackResidentCouncilReadyForVotePhaseV0(run, tasks) {
		return false
	}
	return codexStackResidentCouncilVotesDeliveredV0(run, tasks)
}

func (handler codexStackResidentExternalActionHandlerV0) acceptDecisionCouncilResultV0(
	ctx context.Context,
	request orquestacionnucleoapp.DirectorBriefingExternalActionRequestV0,
) (orquestacionnucleoapp.DirectorBriefingExternalActionResultV0, error) {
	runRef := firstNonEmptyQueuedSourceV0(request.Briefing.RunRef, request.Action.RunRef)
	result := orquestacionnucleoapp.DirectorBriefingExternalActionResultV0{
		RunRef:       runRef,
		ActionRef:    request.Action.ActionRef,
		EvidenceRefs: compactStringsV0(request.EvidenceRefs),
	}
	if handler.Ports.RunStore == nil || handler.Ports.EventSink == nil || handler.Ports.DirectorTaskStore == nil {
		result.Status = orquestacionnucleoapp.DirectorBriefingExecutionStatusExternalPendingV0
		result.EvidenceRefs = compactStringsV0(append(result.EvidenceRefs, "evidence-ref-codex-stack-resident-council-ports-missing"))
		return result, nil
	}
	if handler.DecisionCouncil.VoteSource == nil {
		result.Status = orquestacionnucleoapp.DirectorBriefingExecutionStatusExternalPendingV0
		result.EvidenceRefs = compactStringsV0(append(result.EvidenceRefs, "evidence-ref-codex-stack-resident-council-vote-source-missing"))
		return result, nil
	}
	run, err := handler.Ports.RunStore.LoadRunV0(ctx, runRef)
	if err != nil {
		return result, err
	}
	if codexStackResidentCouncilProgrammingHoldV0(ctx, run, handler.Ports.DirectorTaskStore) {
		result.Status = orquestacionnucleoapp.DirectorBriefingExecutionStatusExternalPendingV0
		result.EvidenceRefs = compactStringsV0(append(result.EvidenceRefs, "evidence-ref-codex-stack-resident-council-open-programming-hold"))
		return result, nil
	}
	decisionRef := codexStackResidentCouncilDecisionRefV0(run.RunID)
	if stringInSetV0(run.Decisions, decisionRef) {
		result.Status = orquestacionnucleoapp.DirectorBriefingExecutionStatusExternalAppliedV0
		result.EvidenceRefs = compactStringsV0(append(result.EvidenceRefs, "evidence-ref-codex-stack-resident-council-decision-already-accepted"))
		return result, nil
	}
	tasks, ok := codexStackResidentCouncilTasksV0(ctx, run, handler.Ports.DirectorTaskStore)
	if !ok || !codexStackResidentCouncilReadyForVotePhaseV0(run, tasks) || !codexStackResidentCouncilVotesDeliveredV0(run, tasks) {
		result.Status = orquestacionnucleoapp.DirectorBriefingExecutionStatusExternalPendingV0
		result.EvidenceRefs = compactStringsV0(append(result.EvidenceRefs, "evidence-ref-codex-stack-resident-council-votes-not-ready"))
		return result, nil
	}
	plan, err := codexStackResidentCouncilPlanFromRunV0(run, request.EvidenceRefs)
	if err != nil {
		result.Status = orquestacionnucleoapp.DirectorBriefingExecutionStatusExternalPendingV0
		result.EvidenceRefs = compactStringsV0(append(result.EvidenceRefs, "evidence-ref-codex-stack-resident-council-plan-pending"))
		return result, nil
	}
	optionRefs := codexStackResidentCouncilOptionRefsV0(tasks)
	voteTasks := codexStackResidentCouncilTasksByRoleV0(tasks, "v")
	voteBuild, err := handler.DecisionCouncil.VoteSource.BuildDecisionCouncilVotesV0(ctx, DecisionCouncilVoteBuildRequestV0{
		Run:          run,
		Tasks:        tasks,
		VoteTasks:    voteTasks,
		TopicRef:     codexStackResidentCouncilTopicRefV0(run.RunID),
		VoteRef:      codexStackResidentCouncilVoteRefV0(run),
		DecisionRef:  decisionRef,
		OptionRefs:   optionRefs,
		EvidenceRefs: compactStringsV0(request.EvidenceRefs),
	})
	if err != nil {
		return result, err
	}
	result.EvidenceRefs = compactStringsV0(append(result.EvidenceRefs, voteBuild.EvidenceRefs...))
	if len(voteBuild.PendingEvidenceRefs) > 0 {
		result.Status = orquestacionnucleoapp.DirectorBriefingExecutionStatusExternalPendingV0
		result.EvidenceRefs = compactStringsV0(append(result.EvidenceRefs, voteBuild.PendingEvidenceRefs...))
		return result, nil
	}
	votes := codexStackResidentCouncilHydrateVotesV0(plan, voteTasks, voteBuild.Votes)
	if len(votes) < len(voteTasks) {
		result.Status = orquestacionnucleoapp.DirectorBriefingExecutionStatusExternalPendingV0
		result.EvidenceRefs = compactStringsV0(append(result.EvidenceRefs, "evidence-ref-codex-stack-resident-council-votes-incomplete"))
		return result, nil
	}
	voteResult, err := orquestadecisioncouncil.EvaluateDecisionCouncilVotesV0(
		orquestadecisioncouncil.DecisionCouncilVoteInputV0{
			DecisionTopicRef:        codexStackResidentCouncilTopicRefV0(run.RunID),
			OptionRefs:              optionRefs,
			Votes:                   votes,
			MinimumVotes:            len(voteTasks),
			MinimumNonAuthorVotes:   len(voteTasks),
			MinimumDistinctFamilies: codexStackResidentCouncilVoteGateMinimumFamiliesV0(plan),
			ApprovalThresholdPct:    67,
		},
	)
	if err != nil {
		result.Status = orquestacionnucleoapp.DirectorBriefingExecutionStatusExternalPendingV0
		result.EvidenceRefs = compactStringsV0(append(result.EvidenceRefs, "evidence-ref-codex-stack-resident-council-votes-invalid"))
		return result, nil
	}
	if !voteResult.Accepted {
		result.Status = orquestacionnucleoapp.DirectorBriefingExecutionStatusExternalPendingV0
		result.EvidenceRefs = compactStringsV0(append(result.EvidenceRefs, "evidence-ref-codex-stack-resident-council-vote-result-"+codexStackOperationalClosureSafeRefV0(voteResult.ReasonCode)))
		return result, nil
	}
	command, ok, err := orquestacionnucleoapp.BuildDecisionCouncilAcceptDecisionCommandV0(
		orquestacionnucleoapp.DecisionCouncilAcceptDecisionCommandRequestV0{
			Run:           run,
			VoteRef:       codexStackResidentCouncilVoteRefV0(run),
			DecisionRef:   decisionRef,
			Summary:       "Decision aceptada por consejo residente multiagente.",
			VoteResult:    voteResult,
			OccurredAt:    request.OccurredAt,
			CorrelationID: request.CorrelationID,
			RequestedBy:   "orquesta-codex-stack-resident-council",
		},
	)
	if err != nil {
		return result, err
	}
	if !ok {
		result.Status = orquestacionnucleoapp.DirectorBriefingExecutionStatusExternalAppliedV0
		result.EvidenceRefs = compactStringsV0(append(result.EvidenceRefs, "evidence-ref-codex-stack-resident-council-decision-already-reflected"))
		return result, nil
	}
	if _, err := orquestacionnucleoapp.HandleStoredWorkflowCommandV0(ctx, handler.Ports.RunStore, handler.Ports.EventSink, command); err != nil {
		return result, err
	}
	result.Status = orquestacionnucleoapp.DirectorBriefingExecutionStatusExternalAppliedV0
	result.EvidenceRefs = compactStringsV0(append(result.EvidenceRefs, "evidence-ref-codex-stack-resident-council-decision-accepted", decisionRef))
	return result, nil
}

func codexStackResidentCouncilDecisionRefV0(runRef string) string {
	return "decision-ref-resident-council-" + codexStackOperationalClosureSafeRefV0(runRef)
}

func codexStackResidentCouncilPlanFromRunV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	evidenceRefs []string,
) (orquestadecisioncouncil.DecisionCouncilPlanV0, error) {
	return orquestadirectorcandidates.BuildDecisionCouncilTeamPlanFromComplexityV0(
		orquestadirectorcandidates.TeamPlanFromComplexityInputV0{
			RunRef:               run.RunID,
			DecisionTopicRef:     codexStackResidentCouncilTopicRefV0(run.RunID),
			BrainstormRequestRef: codexStackResidentCouncilBrainstormRefV0(run),
			VoteRequestRef:       codexStackResidentCouncilVoteRefV0(run),
			Complexity:           codexStackResidentCouncilComplexityV0(run),
			EvidenceRefs: compactStringsV0(append(
				[]string{"evidence-ref-codex-stack-resident-council-plan"},
				evidenceRefs...,
			)),
		},
	)
}

func codexStackResidentCouncilVotesDeliveredV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	tasks []orquestacoreworkflow.WorkflowTaskV0,
) bool {
	votes := codexStackResidentCouncilTasksByRoleV0(tasks, "v")
	if len(votes) == 0 {
		return false
	}
	for _, task := range votes {
		if !codexStackResidentCouncilTaskDoneV0(run, task.TaskID) {
			return false
		}
	}
	return true
}

func codexStackResidentCouncilTasksByRoleV0(
	tasks []orquestacoreworkflow.WorkflowTaskV0,
	role string,
) []orquestacoreworkflow.WorkflowTaskV0 {
	out := []orquestacoreworkflow.WorkflowTaskV0{}
	for _, task := range tasks {
		if codexStackResidentCouncilTaskRoleV0(task) == role {
			out = append(out, task)
		}
	}
	return out
}

func codexStackResidentCouncilOptionRefsV0(
	tasks []orquestacoreworkflow.WorkflowTaskV0,
) []string {
	refs := []string{}
	for _, task := range tasks {
		if codexStackResidentCouncilTaskRoleV0(task) == "p" {
			refs = append(refs, task.TaskID)
		}
	}
	return compactStringsV0(refs)
}

func codexStackResidentCouncilHydrateVotesV0(
	plan orquestadecisioncouncil.DecisionCouncilPlanV0,
	voteTasks []orquestacoreworkflow.WorkflowTaskV0,
	votes []orquestadecisioncouncil.CouncilVoteV0,
) []orquestadecisioncouncil.CouncilVoteV0 {
	taskRefs := map[string]bool{}
	for _, task := range voteTasks {
		taskRefs[strings.TrimSpace(task.TaskID)] = true
	}
	byTaskRef := map[string]orquestadecisioncouncil.CouncilVoteV0{}
	for _, vote := range votes {
		taskRef := strings.TrimSpace(vote.TaskRef)
		if taskRef == "" && taskRefs[strings.TrimSpace(vote.VoteRef)] {
			taskRef = strings.TrimSpace(vote.VoteRef)
		}
		if taskRef != "" {
			byTaskRef[taskRef] = vote
		}
	}
	out := []orquestadecisioncouncil.CouncilVoteV0{}
	for _, task := range voteTasks {
		vote, ok := byTaskRef[strings.TrimSpace(task.TaskID)]
		if !ok {
			continue
		}
		assignment, ok := codexStackResidentCouncilAssignmentForTaskV0(plan, task)
		if !ok {
			continue
		}
		vote.TaskRef = task.TaskID
		vote.VoteRef = firstNonEmptyQueuedSourceV0(vote.VoteRef, "vote-ref-"+task.TaskID)
		vote.VoterRef = assignment.AgentRef
		vote.FamilyRef = assignment.FamilyRef
		vote.EvidenceRefs = compactStringsV0(vote.EvidenceRefs)
		out = append(out, vote)
	}
	return out
}

func codexStackResidentCouncilAssignmentForTaskV0(
	plan orquestadecisioncouncil.DecisionCouncilPlanV0,
	task orquestacoreworkflow.WorkflowTaskV0,
) (orquestadecisioncouncil.CouncilAssignmentV0, bool) {
	assignmentRef := codexStackResidentCouncilTaskCriteriaValueV0(task, "assignment_ref:")
	if assignmentRef == "" {
		return orquestadecisioncouncil.CouncilAssignmentV0{}, false
	}
	for _, assignment := range plan.Assignments {
		if strings.TrimSpace(assignment.AssignmentRef) == assignmentRef {
			return assignment, true
		}
	}
	return orquestadecisioncouncil.CouncilAssignmentV0{}, false
}

func codexStackResidentCouncilTaskCriteriaValueV0(
	task orquestacoreworkflow.WorkflowTaskV0,
	prefix string,
) string {
	for _, criterion := range task.AcceptanceCriteria {
		if value, ok := strings.CutPrefix(strings.TrimSpace(criterion), prefix); ok {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func codexStackResidentCouncilVoteGateMinimumFamiliesV0(
	plan orquestadecisioncouncil.DecisionCouncilPlanV0,
) int {
	for _, gate := range plan.Gates {
		if gate.WaitForRole == orquestadecisioncouncil.CouncilRoleVoteV0 && gate.MinimumDistinctFamilies > 0 {
			return gate.MinimumDistinctFamilies
		}
	}
	return 2
}
