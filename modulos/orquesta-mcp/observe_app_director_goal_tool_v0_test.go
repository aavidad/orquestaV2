package orquestamcp

import (
	"strings"
	"testing"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestagoal "orquesta/modulos/orquesta-goal"
)

func TestObserveGoalDescriptorsDeclaranEvidenciaYAccionRecomendadaV0(t *testing.T) {
	observe := MCPObserveAppDirectorGoalDescriptorV0()
	autoprogramming := MCPAutoprogrammingObserveGoalDescriptorV0()
	for _, descriptor := range []struct {
		name   string
		output string
	}{
		{name: observe.Name, output: observe.Output},
		{name: autoprogramming.Name, output: autoprogramming.Output},
	} {
		if !strings.Contains(descriptor.output, "evidence_refs?") ||
			!strings.Contains(descriptor.output, "recommended_action?") ||
			!strings.Contains(descriptor.output, "error:{errores_publicos,recommended_action?,evidence_refs?}") {
			t.Fatalf("%s debe declarar evidencia y accion en observe_goal: %s", descriptor.name, descriptor.output)
		}
	}
}

func TestEnrichMCPObserveAppDirectorGoalWithMaterializedRefsV0QAFailedPublicTextPideRework(t *testing.T) {
	result := EnrichMCPObserveAppDirectorGoalWithMaterializedRefsV0(
		MCPObserveAppDirectorGoalToolResultV0{
			GoalRef:    "goal-ref-observe-qa-failed-public-text-001",
			GoalStatus: "blocked",
		},
		MCPDirectorGoalMaterializedRefsV0{
			ArtifactRefs: []string{"artifact-ref-materialized-qa-failed-public-text-001"},
			EvidenceRefs: []string{"evidence-ref-goal-materialized-qa-failed-public-text-001"},
			IssueCodes:   []string{MCPGoalFirstQAFailedPublicTextV0},
		},
	)

	if result.RecommendedAction != MCPGoalFirstReworkPublicTextActionV0 ||
		len(result.ClosureIssues) != 1 ||
		result.ClosureIssues[0].Code != MCPGoalFirstQAFailedPublicTextV0 ||
		result.ClosureIssues[0].Field != "goal_first.qa_public_text" ||
		!containsStringMCPV0(result.ArtifactRefs, "artifact-ref-materialized-qa-failed-public-text-001") {
		t.Fatalf("result=%+v", result)
	}
}

func TestEnrichMCPObserveAppDirectorGoalWithMaterializedRefsV0QAFailedPublicTextRunningNoEspera(t *testing.T) {
	cases := []struct {
		name              string
		issueCode         string
		recommendedAction string
	}{
		{
			name:              "qa_failed_public_text",
			issueCode:         MCPGoalFirstQAFailedPublicTextV0,
			recommendedAction: MCPGoalFirstReworkPublicTextActionV0,
		},
		{
			name:              "artifact_paths_omitted_materialized",
			issueCode:         MCPGoalFirstArtifactPathsOmittedMaterializedV0,
			recommendedAction: MCPGoalFirstRepairReceiptActionV0,
		},
		{
			name:              "out_of_scope_materialized_artifacts",
			issueCode:         MCPGoalFirstOutOfScopeMaterializedArtifactsV0,
			recommendedAction: MCPGoalFirstReworkWriteSetViolationActionV0,
		},
		{
			name:              "partial_artifacts_written",
			issueCode:         MCPGoalFirstPartialArtifactsWrittenV0,
			recommendedAction: MCPGoalFirstReviewPartialArtifactsActionV0,
		},
		{
			name:              "phase0_complete_non_publishable",
			issueCode:         MCPGoalFirstPhase0CompleteNonPublishableV0,
			recommendedAction: MCPGoalFirstContinueFromPhase0ActionV0,
		},
		{
			name:              "required_test_evidence_missing",
			issueCode:         MCPGoalFirstRequiredTestEvidenceMissingV0,
			recommendedAction: MCPGoalFirstRepairReceiptActionV0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := EnrichMCPObserveAppDirectorGoalWithMaterializedRefsV0(
				MCPObserveAppDirectorGoalToolResultV0{
					GoalRef:    "goal-ref-observe-" + tc.name + "-running-001",
					GoalStatus: orquestagoal.GoalStatusRunningV0,
				},
				MCPDirectorGoalMaterializedRefsV0{
					ArtifactRefs: []string{"artifact-ref-materialized-" + tc.name + "-running-001"},
					EvidenceRefs: []string{"evidence-ref-goal-materialized-" + tc.name + "-running-001"},
					IssueCodes:   []string{tc.issueCode},
				},
			)

			if result.RecommendedAction != tc.recommendedAction ||
				!containsStringMCPV0(result.ArtifactRefs, "artifact-ref-materialized-"+tc.name+"-running-001") ||
				!containsStringMCPV0(result.EvidenceRefs, "evidence-ref-goal-materialized-"+tc.name+"-running-001") {
				t.Fatalf("result=%+v", result)
			}
		})
	}
}

func TestEnrichMCPObserveAppDirectorGoalWithMaterializedRefsV0ArtifactPathsOmitidosPideRepairReceipt(t *testing.T) {
	result := EnrichMCPObserveAppDirectorGoalWithMaterializedRefsV0(
		MCPObserveAppDirectorGoalToolResultV0{
			GoalRef:    "goal-ref-observe-artifact-paths-omitted-001",
			GoalStatus: "blocked",
		},
		MCPDirectorGoalMaterializedRefsV0{
			ArtifactRefs: []string{"artifact-ref-materialized-omitted-path-001"},
			EvidenceRefs: []string{"evidence-ref-goal-materialized-artifact-paths-omitted"},
			IssueCodes:   []string{MCPGoalFirstArtifactPathsOmittedMaterializedV0},
		},
	)

	if result.RecommendedAction != MCPGoalFirstRepairReceiptActionV0 ||
		len(result.ClosureIssues) != 1 ||
		result.ClosureIssues[0].Code != MCPGoalFirstArtifactPathsOmittedMaterializedV0 ||
		result.ClosureIssues[0].Field != "goal_first.artifact_paths" ||
		!containsStringMCPV0(result.ArtifactRefs, "artifact-ref-materialized-omitted-path-001") {
		t.Fatalf("result=%+v", result)
	}
}

func TestEnrichMCPObserveAppDirectorGoalWithMaterializedRefsV0OutOfScopePideRework(t *testing.T) {
	result := EnrichMCPObserveAppDirectorGoalWithMaterializedRefsV0(
		MCPObserveAppDirectorGoalToolResultV0{
			GoalRef:    "goal-ref-observe-out-of-scope-001",
			GoalStatus: "blocked",
		},
		MCPDirectorGoalMaterializedRefsV0{
			ArtifactRefs: []string{"artifact-ref-materialized-out-of-scope-001"},
			EvidenceRefs: []string{"evidence-ref-goal-materialized-out-of-scope-artifacts"},
			IssueCodes:   []string{MCPGoalFirstOutOfScopeMaterializedArtifactsV0},
		},
	)

	if result.RecommendedAction != MCPGoalFirstReworkWriteSetViolationActionV0 ||
		len(result.ClosureIssues) != 1 ||
		result.ClosureIssues[0].Code != MCPGoalFirstOutOfScopeMaterializedArtifactsV0 ||
		result.ClosureIssues[0].Field != "goal_first.write_set" ||
		!containsStringMCPV0(result.ArtifactRefs, "artifact-ref-materialized-out-of-scope-001") {
		t.Fatalf("result=%+v", result)
	}
}

func TestEnrichMCPObserveAppDirectorGoalWithMaterializedRefsV0ArtefactosParcialesPideRevision(t *testing.T) {
	result := EnrichMCPObserveAppDirectorGoalWithMaterializedRefsV0(
		MCPObserveAppDirectorGoalToolResultV0{
			GoalRef:    "goal-ref-observe-partial-artifacts-001",
			GoalStatus: "blocked",
		},
		MCPDirectorGoalMaterializedRefsV0{
			ArtifactRefs: []string{"artifact-ref-materialized-partial-001"},
			EvidenceRefs: []string{"evidence-ref-goal-materialized-partial-artifacts-written"},
			IssueCodes:   []string{MCPGoalFirstPartialArtifactsWrittenV0},
		},
	)

	if result.RecommendedAction != MCPGoalFirstReviewPartialArtifactsActionV0 ||
		len(result.ClosureIssues) != 1 ||
		result.ClosureIssues[0].Code != MCPGoalFirstPartialArtifactsWrittenV0 ||
		result.ClosureIssues[0].Field != "goal_first.partial_artifacts" ||
		!containsStringMCPV0(result.ArtifactRefs, "artifact-ref-materialized-partial-001") ||
		!containsStringMCPV0(result.EvidenceRefs, "evidence-ref-goal-materialized-partial-artifacts-written") {
		t.Fatalf("result=%+v", result)
	}
}

func TestEnrichMCPObserveAppDirectorGoalWithMaterializedRefsV0Phase0NoPublicablePideContinuar(t *testing.T) {
	result := EnrichMCPObserveAppDirectorGoalWithMaterializedRefsV0(
		MCPObserveAppDirectorGoalToolResultV0{
			GoalRef:    "goal-ref-observe-phase0-001",
			GoalStatus: "blocked",
		},
		MCPDirectorGoalMaterializedRefsV0{
			ArtifactRefs: []string{"artifact-ref-materialized-phase0-001"},
			EvidenceRefs: []string{"evidence-ref-goal-materialized-phase0-complete-non-publishable"},
			IssueCodes:   []string{MCPGoalFirstPhase0CompleteNonPublishableV0},
		},
	)

	if result.RecommendedAction != MCPGoalFirstContinueFromPhase0ActionV0 ||
		len(result.ClosureIssues) != 1 ||
		result.ClosureIssues[0].Code != MCPGoalFirstPhase0CompleteNonPublishableV0 ||
		result.ClosureIssues[0].Field != "goal_first.phase0" ||
		!containsStringMCPV0(result.ArtifactRefs, "artifact-ref-materialized-phase0-001") ||
		!containsStringMCPV0(result.EvidenceRefs, "evidence-ref-goal-materialized-phase0-complete-non-publishable") {
		t.Fatalf("result=%+v", result)
	}
}

func TestObserveAppDirectorGoalRecommendedActionV0RepairReceiptRequiresReworkGanaAParciales(t *testing.T) {
	result := EnrichMCPObserveAppDirectorGoalWithMaterializedRefsV0(
		MCPObserveAppDirectorGoalToolResultV0{
			GoalRef:            "goal-ref-observe-repair-receipt-rework-001",
			GoalStatus:         orquestagoal.GoalStatusCompleteV0,
			ClosureStatus:      orquestagoal.GoalStatusBlockedV0,
			ClosureNeedsRework: true,
			ClosureIssues: []MCPValidationIssueV0{{
				Code:    MCPGoalFirstRepairReceiptRequiresReworkV0,
				Field:   "goal_first.receipt",
				Message: MCPGoalFirstRepairReceiptRequiresReworkV0,
			}},
		},
		MCPDirectorGoalMaterializedRefsV0{
			ArtifactRefs: []string{"artifact-ref-materialized-partial-after-repair-rework-001"},
			EvidenceRefs: []string{"evidence-ref-goal-materialized-partial-after-repair-rework-001"},
			IssueCodes:   []string{MCPGoalFirstPartialArtifactsWrittenV0},
		},
	)

	if result.RecommendedAction != "replan" ||
		!containsStringMCPV0(result.ArtifactRefs, "artifact-ref-materialized-partial-after-repair-rework-001") ||
		!mcpObserveAppDirectorGoalHasClosureIssueCodeV0(result.ClosureIssues, MCPGoalFirstRepairReceiptRequiresReworkV0) ||
		!mcpObserveAppDirectorGoalHasClosureIssueCodeV0(result.ClosureIssues, MCPGoalFirstPartialArtifactsWrittenV0) {
		t.Fatalf("result=%+v", result)
	}
}

func TestEnrichMCPObserveAppDirectorGoalWithMaterializedRefsV0NoPisaCierreAceptado(t *testing.T) {
	result := EnrichMCPObserveAppDirectorGoalWithMaterializedRefsV0(
		MCPObserveAppDirectorGoalToolResultV0{
			GoalRef:         "goal-ref-observe-partial-after-close-001",
			GoalStatus:      orquestagoal.GoalStatusCompleteV0,
			ClosureStatus:   orquestagoal.GoalStatusAcceptedV0,
			ClosureAccepted: true,
		},
		MCPDirectorGoalMaterializedRefsV0{
			ArtifactRefs: []string{"artifact-ref-materialized-partial-after-close-001"},
			EvidenceRefs: []string{"evidence-ref-goal-materialized-partial-after-close"},
			IssueCodes:   []string{MCPGoalFirstPartialArtifactsWrittenV0},
		},
	)

	if result.RecommendedAction != "no_action_closed" ||
		!result.ClosureAccepted ||
		!containsStringMCPV0(result.ArtifactRefs, "artifact-ref-materialized-partial-after-close-001") {
		t.Fatalf("result=%+v", result)
	}
}

func TestEnrichMCPObserveAppDirectorGoalWithMaterializedRefsV0RequiredTestEvidenceAusentePideRepairReceipt(t *testing.T) {
	result := EnrichMCPObserveAppDirectorGoalWithMaterializedRefsV0(
		MCPObserveAppDirectorGoalToolResultV0{
			GoalRef:    "goal-ref-observe-required-test-evidence-001",
			GoalStatus: "blocked",
		},
		MCPDirectorGoalMaterializedRefsV0{
			ArtifactRefs: []string{"artifact-ref-materialized-required-test-evidence-001"},
			EvidenceRefs: []string{"evidence-ref-goal-materialized-required-test-evidence-missing"},
			IssueCodes:   []string{MCPGoalFirstRequiredTestEvidenceMissingV0},
		},
	)

	if result.RecommendedAction != MCPGoalFirstRepairReceiptActionV0 ||
		len(result.ClosureIssues) != 1 ||
		result.ClosureIssues[0].Code != MCPGoalFirstRequiredTestEvidenceMissingV0 ||
		result.ClosureIssues[0].Field != "goal_first.required_tests" ||
		!containsStringMCPV0(result.ArtifactRefs, "artifact-ref-materialized-required-test-evidence-001") ||
		!containsStringMCPV0(result.EvidenceRefs, "evidence-ref-goal-materialized-required-test-evidence-missing") {
		t.Fatalf("result=%+v", result)
	}
}

func TestNewMCPObserveAppDirectorGoalPartialResultFromStateV0ConservaLaunchIssueWriteSetReadOnly(t *testing.T) {
	result, err := NewMCPObserveAppDirectorGoalPartialResultFromStateV0(
		MCPObserveAppDirectorGoalToolInputV0{RunRef: "run-ref-observe-write-set-read-only-001"},
		orquestagoal.GoalWorkStateV0{
			RunRef:  "run-ref-observe-write-set-read-only-001",
			GoalRef: "goal-ref-observe-write-set-read-only-001",
			Status:  orquestagoal.GoalStatusInvalidV0,
			Spec: orquestagoal.GoalWorkSpecV0{
				RunRef:       "run-ref-observe-write-set-read-only-001",
				GoalRef:      "goal-ref-observe-write-set-read-only-001",
				Objective:    "No perder causa del launch invalid.",
				DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
				WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "docs"}},
			},
			LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
				GoalRef: "goal-ref-observe-write-set-read-only-001",
				Status:  orquestagoal.GoalStatusInvalidV0,
				Issues: []orquestagoal.GoalWorkIssueV0{{
					Code: mcpAutoprogrammingActionWriteSetRequiresWorkspaceWriteV0,
				}},
				EvidenceRefs: []string{"evidence-ref-codex-app-server-write-set-requires-workspace-write"},
			},
			EvidenceRefs: []string{"evidence-ref-observe-write-set-read-only-state"},
		},
	)

	if err != nil {
		t.Fatalf("partial: %v", err)
	}
	if result.RecommendedAction != mcpQueueGlobalStatusActionConfigureWorkspaceWriteSandboxV0 ||
		len(result.ClosureIssues) != 1 ||
		result.ClosureIssues[0].Code != mcpAutoprogrammingActionWriteSetRequiresWorkspaceWriteV0 ||
		!containsStringMCPV0(result.EvidenceRefs, "evidence-ref-codex-app-server-write-set-requires-workspace-write") ||
		!containsStringMCPV0(result.EvidenceRefs, "evidence-ref-observe-write-set-read-only-state") {
		t.Fatalf("result=%+v", result)
	}
}

func TestNewMCPObserveAppDirectorGoalPartialResultFromStateV0ConservaLaunchIssueWriteSetGuardContract(t *testing.T) {
	result, err := NewMCPObserveAppDirectorGoalPartialResultFromStateV0(
		MCPObserveAppDirectorGoalToolInputV0{RunRef: "run-ref-observe-write-set-guard-mismatch-001"},
		orquestagoal.GoalWorkStateV0{
			RunRef:  "run-ref-observe-write-set-guard-mismatch-001",
			GoalRef: "goal-ref-observe-write-set-guard-mismatch-001",
			Status:  orquestagoal.GoalStatusInvalidV0,
			Spec: orquestagoal.GoalWorkSpecV0{
				RunRef:       "run-ref-observe-write-set-guard-mismatch-001",
				GoalRef:      "goal-ref-observe-write-set-guard-mismatch-001",
				Objective:    "No perder causa de contrato write-set invalido.",
				DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
				WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "docs"}},
			},
			LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
				GoalRef: "goal-ref-observe-write-set-guard-mismatch-001",
				Status:  orquestagoal.GoalStatusInvalidV0,
				Issues: []orquestagoal.GoalWorkIssueV0{{
					Code: mcpAutoprogrammingActionWriteSetGuardAllowedWriteSetMismatchV0,
				}},
				EvidenceRefs: []string{"evidence-ref-codex-app-server-write-set-guard-allowed-write-set-mismatch"},
			},
			EvidenceRefs: []string{"evidence-ref-observe-write-set-guard-mismatch-state"},
		},
	)

	if err != nil {
		t.Fatalf("partial: %v", err)
	}
	if result.RecommendedAction != mcpQueueGlobalStatusActionRepairGoalWriteSetContractV0 ||
		len(result.ClosureIssues) != 1 ||
		result.ClosureIssues[0].Code != mcpAutoprogrammingActionWriteSetGuardAllowedWriteSetMismatchV0 ||
		!containsStringMCPV0(result.EvidenceRefs, "evidence-ref-codex-app-server-write-set-guard-allowed-write-set-mismatch") ||
		!containsStringMCPV0(result.EvidenceRefs, "evidence-ref-observe-write-set-guard-mismatch-state") {
		t.Fatalf("result=%+v", result)
	}
}

func TestNewMCPObserveAppDirectorGoalResultV0OutputSaneadoPideContextoAcotado(t *testing.T) {
	result := NewMCPObserveAppDirectorGoalResultV0(
		MCPObserveAppDirectorGoalToolInputV0{RunRef: "run-ref-observe-thread-output-sanitized-001"},
		orquestaappdirectorservice.ObserveAppDirectorGoalResultV0{
			Status:                orquestagoal.GoalStatusRunningV0,
			DirectorExecutionMode: orquestaappdirectorservice.AppDirectorExecutionModeGoalFirstV0,
			RunRef:                "run-ref-observe-thread-output-sanitized-001",
			GoalRef:               "goal-ref-observe-thread-output-sanitized-001",
			Run: orquestacoreworkflow.OrchestrationRunV0{
				RunID:  "run-ref-observe-thread-output-sanitized-001",
				Status: orquestacoreworkflow.OrchestrationRunStatusActiveV0,
			},
			GoalResult: orquestagoal.GoalWorkResultV0{
				SchemaVersion: orquestagoal.GoalWorkResultSchemaV0,
				Status:        orquestagoal.GoalStatusRunningV0,
				GoalRef:       "goal-ref-observe-thread-output-sanitized-001",
			},
			EvidenceRefs: []string{mcpAutoprogrammingEvidenceCodexAppServerThreadOutputSanitizedV0},
		},
	)

	if result.GoalStatus != orquestagoal.GoalStatusRunningV0 ||
		result.RecommendedAction != mcpQueueGlobalStatusActionReplanNarrowContextV0 ||
		!containsStringMCPV0(result.EvidenceRefs, mcpAutoprogrammingEvidenceCodexAppServerThreadOutputSanitizedV0) {
		t.Fatalf("result=%+v", result)
	}
}

func TestNewMCPObserveAppDirectorGoalResultV0ReworkRunningNoPublicaClosureBloqueada(t *testing.T) {
	result := NewMCPObserveAppDirectorGoalResultV0(
		MCPObserveAppDirectorGoalToolInputV0{RunRef: "run-ref-observe-rework-running-001"},
		orquestaappdirectorservice.ObserveAppDirectorGoalResultV0{
			Status:                orquestagoal.GoalStatusRunningV0,
			DirectorExecutionMode: orquestaappdirectorservice.AppDirectorExecutionModeGoalFirstV0,
			RunRef:                "run-ref-observe-rework-running-001",
			GoalRef:               "goal-ref-observe-rework-running-001-rework-1",
			ExternalGoalRef:       "thread-ref-observe-rework-running-001",
			Run: orquestacoreworkflow.OrchestrationRunV0{
				RunID:  "run-ref-observe-rework-running-001",
				Status: orquestacoreworkflow.OrchestrationRunStatusActiveV0,
			},
			GoalResult: orquestagoal.GoalWorkResultV0{
				SchemaVersion: orquestagoal.GoalWorkResultSchemaV0,
				Status:        orquestagoal.GoalStatusBlockedV0,
				GoalRef:       "goal-ref-observe-rework-running-001",
				Summary:       "codex_app_server_goal_active_timeout",
				Issues: []orquestagoal.GoalWorkIssueV0{{
					Code:  "codex_app_server_goal_active_timeout",
					Field: "codex_goal_backend",
				}},
			},
			Closure: orquestagoal.GoalClosureValidationV0{
				Status:      orquestagoal.GoalStatusBlockedV0,
				NeedsRework: true,
				Issues: []orquestagoal.GoalWorkIssueV0{{
					Code:  "goal_closure_invalid",
					Field: "status",
				}},
			},
		},
	)

	if result.GoalStatus != orquestagoal.GoalStatusRunningV0 ||
		result.RunStatus != string(orquestacoreworkflow.OrchestrationRunStatusActiveV0) ||
		result.RecommendedAction != "observe_later" ||
		result.ClosureStatus != "" ||
		result.ClosureNeedsRework {
		t.Fatalf("result=%+v", result)
	}
}
