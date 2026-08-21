package acceptance_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/review"
)

func TestAcceptanceV25AGT10ProviderPipeline(t *testing.T) {
	request := v25AGT10PipelineRequest(t)
	complete, err := application.EvaluateProviderPipeline(request)
	if err != nil {
		t.Fatal(err)
	}
	if !complete.Complete || complete.ConsumedReceiptRef != request.Receipts[2].Ref ||
		len(complete.AcceptedReceipts) != 3 {
		t.Fatalf("pipeline did not complete exact author-review chain: %+v", complete)
	}
	repeated, err := application.EvaluateProviderPipeline(request)
	if err != nil || !reflect.DeepEqual(complete, repeated) {
		t.Fatalf("pipeline replay changed decision: decision=%+v err=%v", repeated, err)
	}
	wantRoles := []review.Role{review.RoleAuthor, review.RolePrimary, review.RoleAdversarial}
	for index, receipt := range complete.AcceptedReceipts {
		if receipt.Role != wantRoles[index] || receipt.Candidate != request.Receipts[index].Candidate ||
			receipt.ConsumedReceiptRef != request.Receipts[index].ConsumedReceiptRef {
			t.Fatalf("receipt %d lost role, provider/model or chain: %+v", index, receipt)
		}
	}

	withoutReceipts := request
	withoutReceipts.Receipts = nil
	next, err := application.EvaluateProviderPipeline(withoutReceipts)
	if err != nil {
		t.Fatal(err)
	}
	if next.Complete || next.NextStage.Role != review.RoleAuthor || next.NextStage.Sequence != 1 ||
		next.ConsumedReceiptRef != "" {
		t.Fatalf("pipeline did not select exact author first: %+v", next)
	}

	tests := []struct {
		name   string
		mutate func(*testing.T, *application.ProviderPipelineRequest)
		want   application.ProviderPipelineRejectionReason
	}{
		{"fallback not allowed", func(_ *testing.T, r *application.ProviderPipelineRequest) {
			r.Stages[1].AllowFallback = false
			v25AGT10RebuildReceipts(r)
		}, application.ProviderPipelineFallbackDenied},
		{"receipt transplanted from another goal", func(t *testing.T, r *application.ProviderPipelineRequest) {
			donor := v25AGT10PipelineRequest(t)
			donor.Subject.GoalRef = v25AGT10MustRef(t, goal.NewGoalRef, "goal:acceptance:agt-10:donor")
			v25AGT10RebuildReceipts(&donor)
			r.Receipts = append([]application.ProviderPipelineReceipt(nil), donor.Receipts[0])
		}, application.ProviderPipelineReceiptMismatch},
		{"receipt transplanted from another stage", func(_ *testing.T, r *application.ProviderPipelineRequest) {
			r.Receipts[1] = r.Receipts[2]
			r.Receipts = r.Receipts[:2]
			v25AGT10RechainReceipts(r, 1)
		}, application.ProviderPipelineReceiptMismatch},
		{"launch receipt duplicated", func(_ *testing.T, r *application.ProviderPipelineRequest) {
			r.Receipts[1].LaunchReceiptRef = r.Receipts[0].LaunchReceiptRef
			v25AGT10RechainReceipts(r, 1)
		}, application.ProviderPipelineReplay},
		{"review receipt duplicated", func(_ *testing.T, r *application.ProviderPipelineRequest) {
			r.Receipts[2].ReviewRef = r.Receipts[1].ReviewRef
			v25AGT10RechainReceipts(r, 2)
		}, application.ProviderPipelineReplay},
		{"review subject transplanted from another goal", func(t *testing.T, r *application.ProviderPipelineRequest) {
			donor := v25AGT10PipelineRequest(t)
			donor.Subject.GoalRef = v25AGT10MustRef(t, goal.NewGoalRef, "goal:acceptance:agt-10:subject-donor")
			v25AGT10RebuildReceipts(&donor)
			for index := 1; index < 3; index++ {
				r.Receipts[index].ReviewSubjectDigest = donor.CanonicalReviewSubjectDigest()
			}
			v25AGT10RechainReceipts(r, 1)
		}, application.ProviderPipelineReceiptMismatch},
		{"known receipt skipped", func(_ *testing.T, r *application.ProviderPipelineRequest) {
			r.Receipts[2].ConsumedReceiptRef = r.Receipts[0].Ref
			r.Receipts[2].Ref = r.Receipts[2].CanonicalRef()
		}, application.ProviderPipelineReplay},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mutated := v25AGT10PipelineRequest(t)
			test.mutate(t, &mutated)
			assertV25AGT10PipelineRejection(t, mutated, test.want)
		})
	}
}

func v25AGT10PipelineRequest(t *testing.T) application.ProviderPipelineRequest {
	t.Helper()
	request := application.ProviderPipelineRequest{
		Subject: application.ProviderPipelineSubject{
			ProjectRef: v25AGT10MustRef(t, goal.NewProjectRef, "project:acceptance:agt-10"),
			GoalRef:    v25AGT10MustRef(t, goal.NewGoalRef, "goal:acceptance:agt-10"), GoalRevision: 12,
			PlanGeneration: 5, WorkItemRef: v25AGT10MustRef(t, goal.NewWorkItemRef, "work-item:acceptance:agt-10"),
			WorkItemGeneration: 8, AppSpecGeneration: 3, SpecHash: strings.Repeat("a", 64),
			Artifacts: application.ProviderPipelineArtifactDigests{
				SourceDigest: strings.Repeat("b", 64), DiffDigest: strings.Repeat("c", 64),
				RequiredTestsDigest: strings.Repeat("d", 64),
			},
		},
		Stages: []application.ProviderPipelineStage{
			{
				Sequence: 1, Role: review.RoleAuthor,
				ExecutionRef: v25AGT10MustRef(t, goal.NewExecutionRef, "execution:acceptance:agt-10:author"), ExecutionAttempt: 1,
				Candidates: []application.ProviderRouteCandidate{{ProviderRef: "provider:author", ModelRef: "model:author"}},
			},
			{
				Sequence: 2, Role: review.RolePrimary,
				ExecutionRef: v25AGT10MustRef(t, goal.NewExecutionRef, "execution:acceptance:agt-10:primary"), ExecutionAttempt: 1,
				Candidates: []application.ProviderRouteCandidate{
					{ProviderRef: "provider:primary", ModelRef: "model:primary"},
					{ProviderRef: "provider:primary-fallback", ModelRef: "model:primary-fallback"},
				},
				AllowFallback: true,
			},
			{
				Sequence: 3, Role: review.RoleAdversarial,
				ExecutionRef: v25AGT10MustRef(t, goal.NewExecutionRef, "execution:acceptance:agt-10:adversarial"), ExecutionAttempt: 1,
				Candidates: []application.ProviderRouteCandidate{{ProviderRef: "provider:adversarial", ModelRef: "model:adversarial"}},
			},
		},
	}
	v25AGT10RebuildReceipts(&request)
	return request
}

func v25AGT10RebuildReceipts(request *application.ProviderPipelineRequest) {
	request.Receipts = request.Receipts[:0]
	pipelineDigest := request.Digest()
	for index, stage := range request.Stages {
		candidate := stage.Candidates[0]
		if index == 1 {
			candidate = stage.Candidates[1]
		}
		receipt := application.ProviderPipelineReceipt{
			Subject: request.Subject, Sequence: stage.Sequence, Role: stage.Role,
			ExecutionRef: stage.ExecutionRef, ExecutionAttempt: stage.ExecutionAttempt,
			Candidate: candidate, PipelineDigest: pipelineDigest, StageDigest: stage.Digest(),
			LaunchReceiptRef: "launch-receipt:" + stage.ExecutionRef.String(),
		}
		if index > 0 {
			receipt.ConsumedReceiptRef = request.Receipts[index-1].Ref
			receipt.ReviewRef = "review:" + stage.ExecutionRef.String()
			receipt.ReviewSubjectDigest = request.CanonicalReviewSubjectDigest()
			receipt.ReviewAssessmentDigest = strings.Repeat(string(rune('1'+index)), 64)
			receipt.ReviewVerdict = review.VerdictApprove
		}
		receipt.Ref = receipt.CanonicalRef()
		request.Receipts = append(request.Receipts, receipt)
	}
}

func v25AGT10RechainReceipts(request *application.ProviderPipelineRequest, start int) {
	for index := start; index < len(request.Receipts); index++ {
		request.Receipts[index].ConsumedReceiptRef = request.Receipts[index-1].Ref
		request.Receipts[index].Ref = request.Receipts[index].CanonicalRef()
	}
}

func assertV25AGT10PipelineRejection(
	t *testing.T,
	request application.ProviderPipelineRequest,
	want application.ProviderPipelineRejectionReason,
) {
	t.Helper()
	_, err := application.EvaluateProviderPipeline(request)
	if !errors.Is(err, application.ErrProviderPipelineInvalid) {
		t.Fatalf("pipeline error %v does not expose stable machine code", err)
	}
	var rejected *application.ProviderPipelineError
	if !errors.As(err, &rejected) || rejected.Reason != want {
		t.Fatalf("pipeline rejection=%v want=%s", err, want)
	}
}

func v25AGT10MustRef[T any](t *testing.T, parse func(string) (T, error), value string) T {
	t.Helper()
	ref, err := parse(value)
	if err != nil {
		t.Fatal(err)
	}
	return ref
}
