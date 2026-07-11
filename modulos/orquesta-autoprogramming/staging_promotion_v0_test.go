package orquestaautoprogramming

import "testing"

func TestEvaluateAutoprogrammingStagingPromotionV0ReadyV0(t *testing.T) {
	decision := EvaluateAutoprogrammingStagingPromotionV0(AutoprogrammingStagingPromotionRequestV0{
		RequestRef:         "request-ref-autoprogramming-promotion-001",
		RunRef:             "run-ref-autoprogramming-promotion-001",
		GoalRef:            "goal-ref-autoprogramming-promotion-001",
		ProjectRef:         "project-ref-autoprogramming-promotion-001",
		WorktreeRef:        "worktree-ref-autoprogramming-promotion-001",
		BranchRef:          "branch-ref-autoprogramming-promotion-001",
		RunClosed:          true,
		WriteSet:           []string{"modulos/orquesta-autoprogramming"},
		RequiredTests:      []string{"go test ./modulos/orquesta-autoprogramming"},
		ClosedTaskRefs:     []string{"task-ref-autoprogramming-promotion-001"},
		AcceptedReviewRefs: []string{"accepted-review-ref-autoprogramming-promotion-001"},
		RequiredTestEvidence: []AutoprogrammingRequiredTestEvidenceV0{{
			EvidenceRef: "test-evidence-ref-autoprogramming-promotion-001",
			TaskRef:     "task-ref-autoprogramming-promotion-001",
			TestCommand: "go test ./modulos/orquesta-autoprogramming",
			Status:      "passed",
		}},
	})

	if !decision.Ready ||
		decision.Status != AutoprogrammingStagingPromotionStatusReadyV0 ||
		decision.PromotionCommand.WorktreeRef != "worktree-ref-autoprogramming-promotion-001" ||
		decision.PromotionCommand.BranchRef != "branch-ref-autoprogramming-promotion-001" ||
		decision.PromotionCommand.GoalRef != "goal-ref-autoprogramming-promotion-001" ||
		decision.CleanupCommand.GoalRef != "goal-ref-autoprogramming-promotion-001" ||
		decision.CleanupCommand.ArchiveRef == "" {
		t.Fatalf("decision=%+v", decision)
	}
}

func TestEvaluateAutoprogrammingStagingPromotionV0BloqueaSinTestsV0(t *testing.T) {
	decision := EvaluateAutoprogrammingStagingPromotionV0(AutoprogrammingStagingPromotionRequestV0{
		RunRef:             "run-ref-autoprogramming-promotion-002",
		ProjectRef:         "project-ref-autoprogramming-promotion-002",
		WorktreeRef:        "worktree-ref-autoprogramming-promotion-002",
		BranchRef:          "branch-ref-autoprogramming-promotion-002",
		RunClosed:          true,
		WriteSet:           []string{"cmd/orquesta-server"},
		RequiredTests:      []string{"go test ./cmd/orquesta-server"},
		ClosedTaskRefs:     []string{"task-ref-autoprogramming-promotion-002"},
		AcceptedReviewRefs: []string{"accepted-review-ref-autoprogramming-promotion-002"},
	})

	if decision.Ready ||
		decision.Status != AutoprogrammingStagingPromotionStatusBlockedV0 ||
		len(decision.Issues) != 1 ||
		decision.Issues[0].Code != "required_test_not_passed" {
		t.Fatalf("decision=%+v", decision)
	}
}

func TestEvaluateAutoprogrammingStagingPromotionV0PendientePorSolapeVivoV0(t *testing.T) {
	decision := EvaluateAutoprogrammingStagingPromotionV0(AutoprogrammingStagingPromotionRequestV0{
		RunRef:             "run-ref-autoprogramming-promotion-003",
		ProjectRef:         "project-ref-autoprogramming-promotion-003",
		WorktreeRef:        "worktree-ref-autoprogramming-promotion-003",
		BranchRef:          "branch-ref-autoprogramming-promotion-003",
		RunClosed:          true,
		WriteSet:           []string{"modulos/orquesta-app-codex-stack"},
		RequiredTests:      []string{"go test ./modulos/orquesta-app-codex-stack"},
		ClosedTaskRefs:     []string{"task-ref-autoprogramming-promotion-003"},
		AcceptedReviewRefs: []string{"accepted-review-ref-autoprogramming-promotion-003"},
		RequiredTestEvidence: []AutoprogrammingRequiredTestEvidenceV0{{
			EvidenceRef: "test-evidence-ref-autoprogramming-promotion-003",
			TestCommand: "go test ./modulos/orquesta-app-codex-stack",
			Status:      "passed",
		}},
		LiveWorks: []AutoprogrammingLiveWorkV0{{
			WorkRef:  "run-ref-live-overlap-001",
			Status:   "ready",
			WriteSet: []string{"modulos/orquesta-app-codex-stack/drain_v0.go"},
		}},
	})

	if decision.Ready ||
		decision.Status != AutoprogrammingStagingPromotionStatusPendingV0 ||
		len(decision.PendingRefs) != 1 ||
		decision.PendingRefs[0] != "run-ref-live-overlap-001" {
		t.Fatalf("decision=%+v", decision)
	}
}

func TestAutoprogrammingStagingPromotionV0DeclaraReciboIntegracionV0(t *testing.T) {
	result := AutoprogrammingStagingEffectResultV0{
		SchemaVersion:     AutoprogrammingStagingPromotionSchemaVersionV0,
		Status:            AutoprogrammingStagingEffectPendingPushV0,
		IntegrationStatus: AutoprogrammingStagingIntegrationStatusPendingIntegrationV0,
		CommitRef:         "commit-ref-autoprogramming-pending-integration-001",
		EvidenceRefs:      []string{"evidence-ref-autoprogramming-pending-integration"},
	}

	if result.IntegrationStatus != AutoprogrammingStagingIntegrationStatusPendingIntegrationV0 ||
		result.Status != AutoprogrammingStagingEffectPendingPushV0 ||
		result.IntegrationReceiptRef != "" {
		t.Fatalf("result=%+v", result)
	}
}
