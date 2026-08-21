package application

import (
	"errors"
	"strconv"

	"orquesta/internal/goal"
	"orquesta/internal/review"
)

var ErrProviderPipelineInvalid = errors.New("application.provider_pipeline_invalid")

const providerPipelineReceiptPrefix = "provider-pipeline-receipt:"

type ProviderPipelineRejectionReason string

const (
	ProviderPipelineSubjectInvalid  ProviderPipelineRejectionReason = "subject_invalid"
	ProviderPipelineOrderInvalid    ProviderPipelineRejectionReason = "order_invalid"
	ProviderPipelineRoleInvalid     ProviderPipelineRejectionReason = "role_invalid"
	ProviderPipelineRouteInvalid    ProviderPipelineRejectionReason = "route_invalid"
	ProviderPipelineFallbackDenied  ProviderPipelineRejectionReason = "fallback_denied"
	ProviderPipelineReceiptMismatch ProviderPipelineRejectionReason = "receipt_mismatch"
	ProviderPipelineChainInvalid    ProviderPipelineRejectionReason = "chain_invalid"
	ProviderPipelineReplay          ProviderPipelineRejectionReason = "replay"
)

type ProviderPipelineError struct {
	Reason ProviderPipelineRejectionReason
}

func (err *ProviderPipelineError) Error() string {
	if err == nil {
		return ""
	}
	return "application.provider_pipeline_" + string(err.Reason)
}

func (err *ProviderPipelineError) Unwrap() error { return ErrProviderPipelineInvalid }

// ProviderPipelineArtifactDigests identifies the exact source, diff and test
// set that every stage must consume. Values use the canonical SHA-256 wire
// representation already used by Goal and review contracts.
type ProviderPipelineArtifactDigests struct {
	SourceDigest        string
	DiffDigest          string
	RequiredTestsDigest string
}

// ProviderPipelineSubject is immutable decision input. It is not Goal state
// and grants no lifecycle or provider-launch authority.
type ProviderPipelineSubject struct {
	ProjectRef         goal.ProjectRef
	GoalRef            goal.GoalRef
	GoalRevision       goal.Revision
	PlanGeneration     goal.PlanGeneration
	WorkItemRef        goal.WorkItemRef
	WorkItemGeneration goal.Revision
	AppSpecGeneration  goal.AppSpecGeneration
	SpecHash           string
	Artifacts          ProviderPipelineArtifactDigests
}

func (subject ProviderPipelineSubject) Digest() string {
	return fingerprintFields(
		"orquesta.provider-pipeline-subject.v1",
		subject.ProjectRef.String(), subject.GoalRef.String(),
		strconv.FormatUint(uint64(subject.GoalRevision), 10),
		strconv.FormatUint(uint64(subject.PlanGeneration), 10),
		subject.WorkItemRef.String(), strconv.FormatUint(uint64(subject.WorkItemGeneration), 10),
		strconv.FormatUint(uint64(subject.AppSpecGeneration), 10), subject.SpecHash,
		subject.Artifacts.SourceDigest, subject.Artifacts.DiffDigest,
		subject.Artifacts.RequiredTestsDigest,
	)
}

func providerPipelineDigest(subject ProviderPipelineSubject, stages []ProviderPipelineStage) string {
	fields := []string{subject.Digest()}
	for _, stage := range stages {
		fields = append(fields, stage.Digest())
	}
	return fingerprintFields("orquesta.provider-pipeline.v1", fields...)
}

// ProviderPipelineStage declares exact order, role, execution and route. A
// candidate after the first is a fallback and is unreachable unless the
// caller explicitly sets AllowFallback.
type ProviderPipelineStage struct {
	Sequence         uint64
	Role             review.Role
	ExecutionRef     goal.ExecutionRef
	ExecutionAttempt uint64
	Candidates       []ProviderRouteCandidate
	AllowFallback    bool
}

func (stage ProviderPipelineStage) Digest() string {
	fields := []string{
		strconv.FormatUint(stage.Sequence, 10), string(stage.Role), stage.ExecutionRef.String(),
		strconv.FormatUint(stage.ExecutionAttempt, 10), strconv.FormatBool(stage.AllowFallback),
	}
	for _, candidate := range stage.Candidates {
		fields = append(fields, candidate.ProviderRef, candidate.ModelRef)
	}
	return fingerprintFields("orquesta.provider-pipeline-stage.v1", fields...)
}

// ProviderPipelineReceipt is immutable, content-addressed evidence supplied
// to this pure evaluator. It proves no provider effect or durable existence.
type ProviderPipelineReceipt struct {
	Ref                    string
	Subject                ProviderPipelineSubject
	Sequence               uint64
	Role                   review.Role
	ExecutionRef           goal.ExecutionRef
	ExecutionAttempt       uint64
	Candidate              ProviderRouteCandidate
	PipelineDigest         string
	StageDigest            string
	LaunchReceiptRef       string
	ReviewRef              string
	ReviewSubjectDigest    string
	ReviewAssessmentDigest string
	ReviewVerdict          review.Verdict
	ConsumedReceiptRef     string
}

func (receipt ProviderPipelineReceipt) Digest() string {
	return fingerprintFields(
		"orquesta.provider-pipeline-receipt.v1", receipt.Subject.Digest(),
		strconv.FormatUint(receipt.Sequence, 10), string(receipt.Role), receipt.ExecutionRef.String(),
		strconv.FormatUint(receipt.ExecutionAttempt, 10), receipt.Candidate.ProviderRef,
		receipt.Candidate.ModelRef, receipt.PipelineDigest, receipt.StageDigest,
		receipt.LaunchReceiptRef, receipt.ReviewRef, receipt.ReviewSubjectDigest,
		receipt.ReviewAssessmentDigest, string(receipt.ReviewVerdict), receipt.ConsumedReceiptRef,
	)
}

func (receipt ProviderPipelineReceipt) CanonicalRef() string {
	return providerPipelineReceiptPrefix + receipt.Digest()
}

type ProviderPipelineRequest struct {
	Subject  ProviderPipelineSubject
	Stages   []ProviderPipelineStage
	Receipts []ProviderPipelineReceipt
}

func (request ProviderPipelineRequest) Digest() string {
	return providerPipelineDigest(request.Subject, request.Stages)
}

// CanonicalReviewSubjectDigest binds both reviewers to the exact declared
// pipeline, author stage and canonical author receipt.
func (request ProviderPipelineRequest) CanonicalReviewSubjectDigest() string {
	if len(request.Stages) == 0 || len(request.Receipts) == 0 {
		return ""
	}
	return "sha256:" + fingerprintFields(
		"orquesta.provider-pipeline-review-subject.v1",
		request.Digest(), request.Stages[0].Digest(), request.Receipts[0].CanonicalRef(),
	)
}

// ProviderPipelineDecision projects only the next declared stage. Complete is
// true after an exact receipt prefix covers every stage.
type ProviderPipelineDecision struct {
	Complete           bool
	NextStage          ProviderPipelineStage
	ConsumedReceiptRef string
	AcceptedReceipts   []ProviderPipelineReceipt
}

// EvaluateProviderPipeline is a pure causal decision. It never launches a
// provider, writes Goal lifecycle, persists receipts or chooses an undeclared
// fallback.
func EvaluateProviderPipeline(request ProviderPipelineRequest) (ProviderPipelineDecision, error) {
	if !validProviderPipelineSubject(request.Subject) {
		return rejectProviderPipeline(ProviderPipelineSubjectInvalid)
	}
	if len(request.Stages) != 3 {
		return rejectProviderPipeline(ProviderPipelineOrderInvalid)
	}
	if len(request.Receipts) > len(request.Stages) {
		return rejectProviderPipeline(ProviderPipelineOrderInvalid)
	}
	if err := validateProviderPipelineStages(request.Stages); err != nil {
		return ProviderPipelineDecision{}, err
	}
	accepted, previousRef, err := validateProviderPipelineReceipts(request)
	if err != nil {
		return ProviderPipelineDecision{}, err
	}
	decision := ProviderPipelineDecision{
		Complete:           len(accepted) == len(request.Stages),
		ConsumedReceiptRef: previousRef,
		AcceptedReceipts:   accepted,
	}
	if !decision.Complete {
		decision.NextStage = cloneProviderPipelineStage(request.Stages[len(accepted)])
	}
	return decision, nil
}

func validProviderPipelineSubject(subject ProviderPipelineSubject) bool {
	return subject.ProjectRef.String() != "" && subject.GoalRef.String() != "" &&
		subject.GoalRevision != 0 && subject.PlanGeneration != 0 &&
		subject.WorkItemRef.String() != "" && subject.WorkItemGeneration != 0 &&
		subject.AppSpecGeneration != 0 && goal.IsCanonicalAppSpecHash(subject.SpecHash) &&
		goal.IsCanonicalAppSpecHash(subject.Artifacts.SourceDigest) &&
		goal.IsCanonicalAppSpecHash(subject.Artifacts.DiffDigest) &&
		goal.IsCanonicalAppSpecHash(subject.Artifacts.RequiredTestsDigest)
}

func validateProviderPipelineStages(stages []ProviderPipelineStage) error {
	wantRoles := [...]review.Role{review.RoleAuthor, review.RolePrimary, review.RoleAdversarial}
	seenExecutions := make(map[goal.ExecutionRef]struct{}, len(stages))
	for index, stage := range stages {
		if stage.Sequence != uint64(index+1) || stage.ExecutionRef.String() == "" || stage.ExecutionAttempt == 0 {
			return &ProviderPipelineError{Reason: ProviderPipelineOrderInvalid}
		}
		if stage.Role != wantRoles[index] {
			return &ProviderPipelineError{Reason: ProviderPipelineRoleInvalid}
		}
		if _, duplicate := seenExecutions[stage.ExecutionRef]; duplicate {
			return &ProviderPipelineError{Reason: ProviderPipelineReplay}
		}
		seenExecutions[stage.ExecutionRef] = struct{}{}
		if !validProviderPipelineCandidates(stage.Candidates) {
			return &ProviderPipelineError{Reason: ProviderPipelineRouteInvalid}
		}
	}
	return nil
}

func validProviderPipelineCandidates(candidates []ProviderRouteCandidate) bool {
	if len(candidates) == 0 {
		return false
	}
	seen := make(map[ProviderRouteCandidate]struct{}, len(candidates))
	for _, candidate := range candidates {
		if !validProviderRouteRef(candidate.ProviderRef) || !validProviderRouteRef(candidate.ModelRef) {
			return false
		}
		if _, duplicate := seen[candidate]; duplicate {
			return false
		}
		seen[candidate] = struct{}{}
	}
	return true
}

func validateProviderPipelineReceipts(request ProviderPipelineRequest) (
	[]ProviderPipelineReceipt,
	string,
	error,
) {
	accepted := make([]ProviderPipelineReceipt, 0, len(request.Receipts))
	seenRefs := make(map[string]struct{}, len(request.Receipts))
	seenLaunchRefs := make(map[string]struct{}, len(request.Receipts))
	seenReviewRefs := make(map[string]struct{}, len(request.Receipts))
	pipelineDigest := providerPipelineDigest(request.Subject, request.Stages)
	previousRef := ""
	reviewSubjectDigest := request.CanonicalReviewSubjectDigest()
	for index, receipt := range request.Receipts {
		stage := request.Stages[index]
		if receipt.Subject != request.Subject || receipt.Sequence != stage.Sequence || receipt.Role != stage.Role ||
			receipt.ExecutionRef != stage.ExecutionRef || receipt.ExecutionAttempt != stage.ExecutionAttempt ||
			receipt.PipelineDigest != pipelineDigest || receipt.StageDigest != stage.Digest() ||
			receipt.Ref != receipt.CanonicalRef() || !validApplicationRef(receipt.LaunchReceiptRef) ||
			!validProviderPipelineReviewBinding(receipt, reviewSubjectDigest) {
			return nil, "", &ProviderPipelineError{Reason: ProviderPipelineReceiptMismatch}
		}
		if _, replay := seenLaunchRefs[receipt.LaunchReceiptRef]; replay {
			return nil, "", &ProviderPipelineError{Reason: ProviderPipelineReplay}
		}
		seenLaunchRefs[receipt.LaunchReceiptRef] = struct{}{}
		if receipt.Role != review.RoleAuthor {
			if _, replay := seenReviewRefs[receipt.ReviewRef]; replay {
				return nil, "", &ProviderPipelineError{Reason: ProviderPipelineReplay}
			}
			seenReviewRefs[receipt.ReviewRef] = struct{}{}
		}
		candidateIndex := providerPipelineCandidateIndex(stage.Candidates, receipt.Candidate)
		if candidateIndex < 0 {
			return nil, "", &ProviderPipelineError{Reason: ProviderPipelineRouteInvalid}
		}
		if candidateIndex > 0 && !stage.AllowFallback {
			return nil, "", &ProviderPipelineError{Reason: ProviderPipelineFallbackDenied}
		}
		if receipt.ConsumedReceiptRef != previousRef {
			if _, replay := seenRefs[receipt.ConsumedReceiptRef]; replay {
				return nil, "", &ProviderPipelineError{Reason: ProviderPipelineReplay}
			}
			return nil, "", &ProviderPipelineError{Reason: ProviderPipelineChainInvalid}
		}
		if receipt.ConsumedReceiptRef == receipt.Ref {
			return nil, "", &ProviderPipelineError{Reason: ProviderPipelineChainInvalid}
		}
		if _, replay := seenRefs[receipt.Ref]; replay {
			return nil, "", &ProviderPipelineError{Reason: ProviderPipelineReplay}
		}
		seenRefs[receipt.Ref] = struct{}{}
		accepted = append(accepted, receipt)
		previousRef = receipt.Ref
	}
	return accepted, previousRef, nil
}

func validProviderPipelineReviewBinding(receipt ProviderPipelineReceipt, subjectDigest string) bool {
	if receipt.Role == review.RoleAuthor {
		return receipt.ReviewRef == "" && receipt.ReviewSubjectDigest == "" &&
			receipt.ReviewAssessmentDigest == "" && receipt.ReviewVerdict == ""
	}
	if !validApplicationRef(receipt.ReviewRef) ||
		receipt.ReviewSubjectDigest != subjectDigest ||
		!goal.IsCanonicalAppSpecHash(receipt.ReviewAssessmentDigest) ||
		(receipt.ReviewVerdict != review.VerdictApprove &&
			receipt.ReviewVerdict != review.VerdictChangesRequested) {
		return false
	}
	return subjectDigest != ""
}

func providerPipelineCandidateIndex(candidates []ProviderRouteCandidate, selected ProviderRouteCandidate) int {
	for index, candidate := range candidates {
		if candidate == selected {
			return index
		}
	}
	return -1
}

func cloneProviderPipelineStage(stage ProviderPipelineStage) ProviderPipelineStage {
	clone := stage
	clone.Candidates = append([]ProviderRouteCandidate(nil), stage.Candidates...)
	return clone
}

func rejectProviderPipeline(reason ProviderPipelineRejectionReason) (ProviderPipelineDecision, error) {
	return ProviderPipelineDecision{}, &ProviderPipelineError{Reason: reason}
}
