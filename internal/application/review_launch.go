package application

import (
	"encoding/json"
	"errors"
	"fmt"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
	"orquesta/internal/review"
)

func reviewerAgentLaunchRequest(record GoalRecord, item goal.WorkItem, execution ExecutionRecord,
	phase goal.PhaseInstance, policy TestAttestationPolicy,
) (ports.AgentLaunchRequest, error) {
	role, reviewer := reviewerRole(execution)
	attachment, err := reviewAttached(record, item, execution, policy)
	if !reviewer {
		return ports.AgentLaunchRequest{}, errors.New("review.attachment_invalid")
	}
	if err != nil {
		return ports.AgentLaunchRequest{}, err
	}
	return reviewerAgentLaunchRequestFromAttachment(record, item, execution, phase, role, attachment)
}

func reviewerAgentLaunchRequestFromAttachment(record GoalRecord, item goal.WorkItem, execution ExecutionRecord,
	phase goal.PhaseInstance, role review.Role, attachment ReviewAttachment,
) (ports.AgentLaunchRequest, error) {
	change, subject := attachment.Change, attachment.Subject
	tests := make([]reviewTestEvidence, 0, len(item.RequiredTests()))
	for _, spec := range item.RequiredTests() {
		tests = append(tests, reviewTestEvidence{Ref: spec.Ref().String(), Digest: spec.Digest()})
	}
	evidence, err := json.Marshal(reviewLaunchEvidence{
		SubjectDigest: subject.Digest(), Role: role, BaseOID: change.BaseOID, ParentOID: change.ParentOID,
		HeadOID: change.HeadOID, TreeOID: change.TreeOID, ChangeRef: change.Ref.String(),
		ChangeDigest: change.Digest(), DiffDigest: change.DiffDigest, WriteSetDigest: change.WriteSetDigest,
		ChangedPaths: append([]string(nil), change.ChangedPaths...), RequiredTestsDigest: subject.RequiredTestsDigest,
		RequiredTests: tests, TestAttestationRef: subject.TestAttestationRef,
		TestSubjectDigest: subject.TestSubjectDigest, TestPolicyDigest: subject.TestPolicyDigest,
		InputRefs: workItemRefs(phase.InputRefs()), CriterionRefs: workItemRefs(phase.CriterionRefs()),
	})
	if err != nil {
		return ports.AgentLaunchRequest{}, errors.New("review.launch_evidence_invalid")
	}
	request := launchEffectTargetRequest(record.Goal, item, execution)
	request.Objective = fmt.Sprintf(
		"Review exact immutable evidence %s. Inspect with `git diff %s..%s`. Return only JSON: {\"schema_version\":1,\"subject_digest\":\"%s\",\"role\":\"%s\",\"verdict\":\"approve|changes_requested\",\"summary\":\"...\",\"findings\":[{\"code\":\"...\",\"severity\":\"low|medium|high\",\"evidence_ref\":\"...\"}]}",
		string(evidence), change.BaseOID, change.HeadOID, subject.Digest(), role,
	)
	request.PhaseRef, request.PhaseKey, request.PhaseTemplateRef = phase.Ref().String(), item.Phase().String(), phase.TemplateRef().String()
	request.PhaseInputRefs, request.PhaseCriterionRefs = workItemRefs(phase.InputRefs()), workItemRefs(phase.CriterionRefs())
	request.RoleKey, request.OutputContract = "role:reviewer", string(goal.OutputContractAttestation)
	request.ArtifactMediaType, request.WriteSet = review.AssessmentMediaType, nil
	request.MaxOutputBytes, request.BudgetDemand = execution.MaxOutputBytes, item.BudgetDemand()
	request.SecurityCriticality, request.ReasoningEffort = item.SecurityCriticality(), item.ReasoningEffort()
	if err := bindDurableAgentLaunchEgressAuthority(record, &request); err != nil {
		return ports.AgentLaunchRequest{}, err
	}
	return request, nil
}

type reviewTestEvidence struct {
	Ref    string `json:"ref"`
	Digest string `json:"digest"`
}

type reviewLaunchEvidence struct {
	SubjectDigest       string               `json:"subject_digest"`
	Role                review.Role          `json:"role"`
	BaseOID             string               `json:"base_oid"`
	ParentOID           string               `json:"parent_oid"`
	HeadOID             string               `json:"head_oid"`
	TreeOID             string               `json:"tree_oid"`
	ChangeRef           string               `json:"change_ref"`
	ChangeDigest        string               `json:"change_digest"`
	DiffDigest          string               `json:"diff_digest"`
	WriteSetDigest      string               `json:"write_set_digest"`
	ChangedPaths        []string             `json:"changed_paths"`
	RequiredTestsDigest string               `json:"required_tests_digest"`
	RequiredTests       []reviewTestEvidence `json:"required_tests"`
	TestAttestationRef  string               `json:"test_attestation_ref"`
	TestSubjectDigest   string               `json:"test_subject_digest"`
	TestPolicyDigest    string               `json:"test_policy_digest"`
	InputRefs           []string             `json:"input_refs"`
	CriterionRefs       []string             `json:"criterion_refs"`
}

func reviewExternalRefAvailable(record GoalRecord, execution ExecutionRecord, externalRef string) bool {
	if externalRef == "" {
		return false
	}
	for _, participant := range record.Executions {
		if participant.Ref == execution.Ref || participant.WorkItemRef != execution.WorkItemRef ||
			participant.GoalRef != execution.GoalRef || participant.ExternalRef == "" {
			continue
		}
		if (participant.Purpose == ExecutionPurposeAuthor || isReviewerExecution(participant)) &&
			participant.ExternalRef == externalRef {
			return false
		}
	}
	return true
}
