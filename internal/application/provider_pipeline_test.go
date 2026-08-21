package application

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"orquesta/internal/goal"
	"orquesta/internal/review"
)

func TestProviderPipelineAdvancesExactReceiptChainAndCopiesDecision(t *testing.T) {
	request := providerPipelineTestRequest(t)
	receipts := append([]ProviderPipelineReceipt(nil), request.Receipts...)
	tests := []struct {
		count    int
		complete bool
		nextRole review.Role
	}{
		{0, false, review.RoleAuthor},
		{1, false, review.RolePrimary},
		{2, false, review.RoleAdversarial},
		{3, true, ""},
	}
	for _, test := range tests {
		request.Receipts = receipts[:test.count]
		decision, err := EvaluateProviderPipeline(request)
		if err != nil {
			t.Fatalf("prefix %d: %v", test.count, err)
		}
		wantConsumed := ""
		if test.count > 0 {
			wantConsumed = receipts[test.count-1].Ref
		}
		if decision.Complete != test.complete || decision.ConsumedReceiptRef != wantConsumed ||
			len(decision.AcceptedReceipts) != test.count {
			t.Fatalf("prefix %d decision: %+v", test.count, decision)
		}
		if !test.complete && (decision.NextStage.Sequence != uint64(test.count+1) || decision.NextStage.Role != test.nextRole) {
			t.Fatalf("prefix %d next stage: %+v", test.count, decision.NextStage)
		}
	}

	request.Receipts = receipts
	first, err := EvaluateProviderPipeline(request)
	second, repeatedErr := EvaluateProviderPipeline(request)
	if err != nil || repeatedErr != nil || !reflect.DeepEqual(first, second) {
		t.Fatalf("replay changed decision: first=%+v second=%+v errors=%v/%v", first, second, err, repeatedErr)
	}
	for index, receipt := range first.AcceptedReceipts {
		stage := request.Stages[index]
		if receipt.Role != stage.Role || receipt.Candidate != request.Receipts[index].Candidate ||
			receipt.PipelineDigest != request.Digest() || receipt.StageDigest != stage.Digest() {
			t.Fatalf("receipt %d lost role, provider/model or digest: %+v", index, receipt)
		}
	}

	request.Receipts = receipts[:1]
	copied, err := EvaluateProviderPipeline(request)
	if err != nil {
		t.Fatal(err)
	}
	copied.NextStage.Candidates[0].ProviderRef = "provider:mutated-output"
	if request.Stages[1].Candidates[0].ProviderRef == "provider:mutated-output" {
		t.Fatal("decision aliases request candidates")
	}
}

func TestProviderPipelineRejectsInvalidDeclaration(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*ProviderPipelineRequest)
		want   ProviderPipelineRejectionReason
	}{
		{"too few stages", func(r *ProviderPipelineRequest) { r.Stages = r.Stages[:2] }, ProviderPipelineOrderInvalid},
		{"order gap", func(r *ProviderPipelineRequest) { r.Stages[1].Sequence = 3 }, ProviderPipelineOrderInvalid},
		{"reviewer first", func(r *ProviderPipelineRequest) { r.Stages[0].Role = review.RolePrimary }, ProviderPipelineRoleInvalid},
		{"second author", func(r *ProviderPipelineRequest) { r.Stages[1].Role = review.RoleAuthor }, ProviderPipelineRoleInvalid},
		{"roles swapped", func(r *ProviderPipelineRequest) {
			r.Stages[1].Role, r.Stages[2].Role = review.RoleAdversarial, review.RolePrimary
		}, ProviderPipelineRoleInvalid},
		{"execution replay", func(r *ProviderPipelineRequest) { r.Stages[2].ExecutionRef = r.Stages[0].ExecutionRef }, ProviderPipelineReplay},
		{"missing route", func(r *ProviderPipelineRequest) { r.Stages[0].Candidates = nil }, ProviderPipelineRouteInvalid},
		{"duplicate route", func(r *ProviderPipelineRequest) {
			r.Stages[1].Candidates = append(r.Stages[1].Candidates, r.Stages[1].Candidates[0])
		}, ProviderPipelineRouteInvalid},
		{"undeclared selection", func(r *ProviderPipelineRequest) {
			r.Receipts[1].Candidate = ProviderRouteCandidate{ProviderRef: "provider:other", ModelRef: "model:other"}
			rechainProviderPipelineReceipts(r, 1)
		}, ProviderPipelineRouteInvalid},
		{"implicit fallback", func(r *ProviderPipelineRequest) {
			r.Stages[1].AllowFallback = false
			rebuildProviderPipelineTestReceipts(r)
		}, ProviderPipelineFallbackDenied},
		{"extra receipt", func(r *ProviderPipelineRequest) { r.Receipts = append(r.Receipts, r.Receipts[2]) }, ProviderPipelineOrderInvalid},
		{"empty goal", func(r *ProviderPipelineRequest) { r.Subject.GoalRef = goal.GoalRef{} }, ProviderPipelineSubjectInvalid},
		{"invalid artifact", func(r *ProviderPipelineRequest) { r.Subject.Artifacts.DiffDigest = "invalid" }, ProviderPipelineSubjectInvalid},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := providerPipelineTestRequest(t)
			test.mutate(&request)
			assertProviderPipelineRejection(t, request, test.want)
		})
	}
}

func TestProviderPipelineRejectsReceiptTransplantsAndDuplicates(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*testing.T, *ProviderPipelineRequest)
		want   ProviderPipelineRejectionReason
	}{
		{"receipt from another goal", func(t *testing.T, r *ProviderPipelineRequest) {
			donor := providerPipelineTestRequest(t)
			donor.Subject.GoalRef = mustProviderPipelineRef(t, goal.NewGoalRef, "goal:provider-pipeline:donor")
			rebuildProviderPipelineTestReceipts(&donor)
			r.Receipts = append([]ProviderPipelineReceipt(nil), donor.Receipts[0])
		}, ProviderPipelineReceiptMismatch},
		{"receipt from another stage", func(_ *testing.T, r *ProviderPipelineRequest) {
			r.Receipts[1] = r.Receipts[2]
			r.Receipts = r.Receipts[:2]
			rechainProviderPipelineReceipts(r, 1)
		}, ProviderPipelineReceiptMismatch},
		{"author launch duplicated by primary", func(_ *testing.T, r *ProviderPipelineRequest) {
			r.Receipts[1].LaunchReceiptRef = r.Receipts[0].LaunchReceiptRef
			rechainProviderPipelineReceipts(r, 1)
		}, ProviderPipelineReplay},
		{"primary launch duplicated by adversarial", func(_ *testing.T, r *ProviderPipelineRequest) {
			r.Receipts[2].LaunchReceiptRef = r.Receipts[1].LaunchReceiptRef
			rechainProviderPipelineReceipts(r, 2)
		}, ProviderPipelineReplay},
		{"review duplicated", func(_ *testing.T, r *ProviderPipelineRequest) {
			r.Receipts[2].ReviewRef = r.Receipts[1].ReviewRef
			rechainProviderPipelineReceipts(r, 2)
		}, ProviderPipelineReplay},
		{"arbitrary shared review subject", func(_ *testing.T, r *ProviderPipelineRequest) {
			for index := 1; index < 3; index++ {
				r.Receipts[index].ReviewSubjectDigest = "sha256:" + providerPipelineHash('9')
			}
			rechainProviderPipelineReceipts(r, 1)
		}, ProviderPipelineReceiptMismatch},
		{"review subject from another goal", func(t *testing.T, r *ProviderPipelineRequest) {
			donor := providerPipelineTestRequest(t)
			donor.Subject.GoalRef = mustProviderPipelineRef(t, goal.NewGoalRef, "goal:provider-pipeline:subject-donor")
			rebuildProviderPipelineTestReceipts(&donor)
			for index := 1; index < 3; index++ {
				r.Receipts[index].ReviewSubjectDigest = donor.CanonicalReviewSubjectDigest()
			}
			rechainProviderPipelineReceipts(r, 1)
		}, ProviderPipelineReceiptMismatch},
		{"review subject from wrong stage", func(_ *testing.T, r *ProviderPipelineRequest) {
			wrong := "sha256:" + fingerprintFields("orquesta.provider-pipeline-review-subject.v1",
				r.Digest(), r.Stages[1].Digest(), r.Receipts[0].CanonicalRef())
			for index := 1; index < 3; index++ {
				r.Receipts[index].ReviewSubjectDigest = wrong
			}
			rechainProviderPipelineReceipts(r, 1)
		}, ProviderPipelineReceiptMismatch},
		{"future stage changed after author receipt", func(_ *testing.T, r *ProviderPipelineRequest) {
			r.Receipts = r.Receipts[:1]
			r.Stages[2].Candidates[0].ModelRef = "model:future-mutated"
		}, ProviderPipelineReceiptMismatch},
		{"known receipt skipped", func(_ *testing.T, r *ProviderPipelineRequest) {
			r.Receipts[2].ConsumedReceiptRef = r.Receipts[0].Ref
			r.Receipts[2].Ref = r.Receipts[2].CanonicalRef()
		}, ProviderPipelineReplay},
		{"unknown predecessor", func(_ *testing.T, r *ProviderPipelineRequest) {
			r.Receipts[1].ConsumedReceiptRef = "provider-pipeline-receipt:unknown"
			r.Receipts[1].Ref = r.Receipts[1].CanonicalRef()
		}, ProviderPipelineChainInvalid},
		{"author carries review", func(_ *testing.T, r *ProviderPipelineRequest) {
			r.Receipts[0].ReviewRef = "review:forged-author"
			r.Receipts[0].Ref = r.Receipts[0].CanonicalRef()
		}, ProviderPipelineReceiptMismatch},
		{"review missing ref", func(_ *testing.T, r *ProviderPipelineRequest) {
			r.Receipts[1].ReviewRef = ""
			rechainProviderPipelineReceipts(r, 1)
		}, ProviderPipelineReceiptMismatch},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := providerPipelineTestRequest(t)
			test.mutate(t, &request)
			assertProviderPipelineRejection(t, request, test.want)
		})
	}
}

func providerPipelineTestRequest(t *testing.T) ProviderPipelineRequest {
	t.Helper()
	request := ProviderPipelineRequest{
		Subject: ProviderPipelineSubject{
			ProjectRef: mustProviderPipelineRef(t, goal.NewProjectRef, "project:provider-pipeline"),
			GoalRef:    mustProviderPipelineRef(t, goal.NewGoalRef, "goal:provider-pipeline"), GoalRevision: 9,
			PlanGeneration: 4, WorkItemRef: mustProviderPipelineRef(t, goal.NewWorkItemRef, "work-item:provider-pipeline"),
			WorkItemGeneration: 7, AppSpecGeneration: 2, SpecHash: providerPipelineHash('a'),
			Artifacts: ProviderPipelineArtifactDigests{SourceDigest: providerPipelineHash('b'),
				DiffDigest: providerPipelineHash('c'), RequiredTestsDigest: providerPipelineHash('d')},
		},
		Stages: []ProviderPipelineStage{
			{Sequence: 1, Role: review.RoleAuthor,
				ExecutionRef: mustProviderPipelineRef(t, goal.NewExecutionRef, "execution:provider-pipeline:author"), ExecutionAttempt: 1,
				Candidates: []ProviderRouteCandidate{{ProviderRef: "provider:author", ModelRef: "model:author"}}},
			{Sequence: 2, Role: review.RolePrimary,
				ExecutionRef: mustProviderPipelineRef(t, goal.NewExecutionRef, "execution:provider-pipeline:primary"), ExecutionAttempt: 1,
				Candidates: []ProviderRouteCandidate{{ProviderRef: "provider:primary", ModelRef: "model:primary"},
					{ProviderRef: "provider:primary-fallback", ModelRef: "model:primary-fallback"}}, AllowFallback: true},
			{Sequence: 3, Role: review.RoleAdversarial,
				ExecutionRef: mustProviderPipelineRef(t, goal.NewExecutionRef, "execution:provider-pipeline:adversarial"), ExecutionAttempt: 1,
				Candidates: []ProviderRouteCandidate{{ProviderRef: "provider:adversarial", ModelRef: "model:adversarial"}}},
		},
	}
	rebuildProviderPipelineTestReceipts(&request)
	return request
}

func rebuildProviderPipelineTestReceipts(request *ProviderPipelineRequest) {
	request.Receipts = nil
	for index, stage := range request.Stages {
		candidate := stage.Candidates[0]
		if index == 1 && len(stage.Candidates) > 1 {
			candidate = stage.Candidates[1]
		}
		receipt := ProviderPipelineReceipt{Subject: request.Subject, Sequence: stage.Sequence, Role: stage.Role,
			ExecutionRef: stage.ExecutionRef, ExecutionAttempt: stage.ExecutionAttempt, Candidate: candidate,
			PipelineDigest: request.Digest(), StageDigest: stage.Digest(), LaunchReceiptRef: "launch:" + stage.ExecutionRef.String()}
		if index > 0 {
			receipt.ConsumedReceiptRef = request.Receipts[index-1].Ref
			receipt.ReviewRef = "review:" + stage.ExecutionRef.String()
			receipt.ReviewSubjectDigest = request.CanonicalReviewSubjectDigest()
			receipt.ReviewAssessmentDigest = providerPipelineHash(byte('1' + index))
			receipt.ReviewVerdict = review.VerdictApprove
		}
		receipt.Ref = receipt.CanonicalRef()
		request.Receipts = append(request.Receipts, receipt)
	}
}

func rechainProviderPipelineReceipts(request *ProviderPipelineRequest, start int) {
	for index := start; index < len(request.Receipts); index++ {
		if index > 0 {
			request.Receipts[index].ConsumedReceiptRef = request.Receipts[index-1].Ref
		}
		request.Receipts[index].Ref = request.Receipts[index].CanonicalRef()
	}
}

func assertProviderPipelineRejection(t *testing.T, request ProviderPipelineRequest, want ProviderPipelineRejectionReason) {
	t.Helper()
	_, err := EvaluateProviderPipeline(request)
	if !errors.Is(err, ErrProviderPipelineInvalid) {
		t.Fatalf("error %v does not wrap stable pipeline error", err)
	}
	var rejected *ProviderPipelineError
	if !errors.As(err, &rejected) || rejected.Reason != want {
		t.Fatalf("rejection=%v want=%s", err, want)
	}
}

func mustProviderPipelineRef[T any](t *testing.T, parse func(string) (T, error), value string) T {
	t.Helper()
	ref, err := parse(value)
	if err != nil {
		t.Fatal(err)
	}
	return ref
}

func providerPipelineHash(value byte) string { return strings.Repeat(string(value), 64) }
