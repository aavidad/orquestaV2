package application

import (
	"encoding/json"
	"errors"
	"fmt"

	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

type councilLaunchEvidence struct {
	SubjectDigest string       `json:"subject_digest"`
	ReviewSubject string       `json:"review_subject_digest"`
	ReviewGate    string       `json:"review_gate_digest"`
	ChangeSetRef  string       `json:"change_set_ref"`
	Role          council.Role `json:"role"`
}

func councilAgentLaunchRequest(record GoalRecord, item goal.WorkItem, execution ExecutionRecord,
	phase goal.PhaseInstance,
) (ports.AgentLaunchRequest, error) {
	round, err := councilAttachment(record, item, execution)
	if err != nil {
		return ports.AgentLaunchRequest{}, err
	}
	return councilAgentLaunchRequestForSubject(record, item, execution, phase, round.Subject)
}

func councilAgentLaunchRequestForSubject(record GoalRecord, item goal.WorkItem, execution ExecutionRecord,
	phase goal.PhaseInstance, subject council.Subject,
) (ports.AgentLaunchRequest, error) {
	role, ok := councilRole(execution)
	if !ok || execution.ReviewSubjectDigest != "" || execution.CouncilSubjectDigest != mustCouncilSubjectDigest(subject) {
		return ports.AgentLaunchRequest{}, errors.New("council.attachment_invalid")
	}
	evidence, err := json.Marshal(councilLaunchEvidence{SubjectDigest: subject.Digest(), ReviewSubject: subject.ReviewSubjectDigest,
		ReviewGate: subject.ReviewGateDigest, ChangeSetRef: subject.ChangeSetRef, Role: role})
	if err != nil {
		return ports.AgentLaunchRequest{}, errors.New("council.launch_evidence_invalid")
	}
	request := launchEffectTargetRequest(record.Goal, item, execution)
	request.Objective = fmt.Sprintf("Assess exact approved V18 evidence %s. Return exactly one JSON envelope: {\"schema\":\"orquesta.council.contribution.v1\",\"subject_digest\":\"%s\",\"role\":\"%s\",\"body\":\"...\",\"ballot\":\"accept|reject|abstain|security_veto\",\"evidence\":[{\"kind\":\"...\",\"ref\":\"...\"}]}.", string(evidence), subject.Digest(), role)
	request.PhaseRef, request.PhaseKey, request.PhaseTemplateRef = phase.Ref().String(), item.Phase().String(), phase.TemplateRef().String()
	request.PhaseInputRefs, request.PhaseCriterionRefs = workItemRefs(phase.InputRefs()), workItemRefs(phase.CriterionRefs())
	request.RoleKey, request.OutputContract = "role:council-"+string(role), council.ContributionSchema
	request.ArtifactMediaType, request.WriteSet = council.ContributionMediaType, nil
	request.MaxOutputBytes, request.BudgetDemand = execution.MaxOutputBytes, item.BudgetDemand()
	request.SecurityCriticality, request.ReasoningEffort = item.SecurityCriticality(), item.ReasoningEffort()
	return request, nil
}

func councilAttachment(record GoalRecord, item goal.WorkItem, execution ExecutionRecord) (CouncilRoundRecord, error) {
	if _, ok := councilRole(execution); !ok || execution.GoalRef != record.Goal.Ref() || execution.WorkItemRef != item.Ref() ||
		execution.ReviewSubjectDigest != "" || !validCouncilDigest(string(execution.CouncilSubjectDigest)) {
		return CouncilRoundRecord{}, errors.New("council.attachment_invalid")
	}
	round, found := councilRoundFor(record, execution.CouncilSubjectDigest)
	if !found || round.Subject.WorkItemRef != item.Ref().String() || round.Subject.GoalRef != record.Goal.Ref().String() ||
		round.SubjectDigest != execution.CouncilSubjectDigest {
		return CouncilRoundRecord{}, errors.New("council.attachment_invalid")
	}
	return round, nil
}

func validateCouncilLaunch(record GoalRecord, item goal.WorkItem, execution ExecutionRecord) error {
	if item.State() != goal.WorkItemStateRunning {
		return errors.New("council.attachment_invalid")
	}
	_, err := councilAttachment(record, item, execution)
	return err
}

func councilExternalRefAvailable(record GoalRecord, execution ExecutionRecord, externalRef string) bool {
	if externalRef == "" {
		return false
	}
	for _, participant := range record.Executions {
		if participant.Ref == execution.Ref || participant.GoalRef != execution.GoalRef || participant.WorkItemRef != execution.WorkItemRef || participant.ExternalRef == "" {
			continue
		}
		if participant.ExternalRef == externalRef {
			return false
		}
	}
	return true
}

func isCouncilExecution(execution ExecutionRecord) bool { _, ok := councilRole(execution); return ok }

func mustCouncilSubjectDigest(subject council.Subject) CouncilSubjectDigest {
	digest, _ := councilDigest(subject.Digest())
	return digest
}
