package orquestaserver

import (
	"encoding/json"
	"strings"
	"testing"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

func TestServerPublicStatusV0ExponeGoalFirstCompactoSinSpecCompletaV0(t *testing.T) {
	state := StateV0{
		Status: "running",
		IdleSelfImprovementOperationalMessage: &ServerOperationalMessageV0{
			Status:     orquestagoal.GoalStatusCompleteV0,
			ReasonCode: "goal_complete_pending_closure_validation",
			GoalRefs:   []string{"goal-ref-public-001", "external-goal-ref-public-001"},
		},
		IdleSelfImprovementGoalSpec: &orquestagoal.GoalWorkSpecV0{
			GoalRef:         "goal-ref-public-001",
			RequestRef:      "request-ref-public-001",
			RunRef:          "run-ref-public-001",
			ProjectRef:      "project-ref-public-001",
			WorkKind:        "idle_self_improvement",
			WorkProfileKind: "autoprogramming",
			Objective:       "objetivo interno que no debe salir en status publico",
			DirectorKind:    orquestagoal.GoalDirectorKindCodexGoalV0,
			ContextRefs:     []orquestagoal.GoalContextRefV0{{Ref: "doc-ref-public-001"}},
			RuleRefs:        []orquestagoal.GoalRuleRefV0{{Ref: "rule-ref-public-001"}},
			SkillRefs:       []string{"skill-ref-public-001"},
			WriteSet:        []orquestagoal.GoalWriteScopeV0{{Path: "modulos/orquesta-server/private_file.go"}},
			RequiredTests:   []orquestagoal.GoalRequiredTestV0{{TestRef: "test-ref-public-001"}},
			ArtifactContracts: []orquestagoal.GoalArtifactContractV0{{
				ArtifactRef:  "artifact-contract-ref-public-001",
				ArtifactType: "patch",
				Required:     true,
			}},
			EvidenceRefs: []string{"evidence-ref-spec-public-001"},
		},
		IdleSelfImprovementGoalReceipt: &orquestagoal.GoalLaunchReceiptV0{
			Status:          orquestagoal.GoalStatusAcceptedV0,
			GoalRef:         "goal-ref-public-001",
			ExternalGoalRef: "external-goal-ref-public-001",
			EvidenceRefs:    []string{"evidence-ref-receipt-public-001"},
		},
		IdleSelfImprovementGoalResult: &orquestagoal.GoalWorkResultV0{
			Status:          orquestagoal.GoalStatusCompleteV0,
			GoalRef:         "goal-ref-public-001",
			ExternalGoalRef: "external-goal-ref-public-001",
			Summary:         "resumen interno que no debe salir en status publico",
			ArtifactRefs:    []string{"artifact-ref-public-001"},
			RequiredTestResults: []orquestagoal.GoalRequiredTestResultV0{{
				TestRef: "test-ref-public-001",
				Status:  "passed",
			}},
			DomainReceiptRefs: []string{"receipt-ref-public-001"},
			EvidenceRefs:      []string{"evidence-ref-result-public-001"},
		},
		IdleSelfImprovementGoalClosure: &orquestagoal.GoalClosureValidationV0{
			Status:       orquestagoal.GoalStatusBlockedV0,
			NeedsRework:  true,
			EvidenceRefs: []string{"evidence-ref-closure-public-001"},
			Issues:       []orquestagoal.GoalWorkIssueV0{{Code: orquestagoal.ErrGoalClosureInvalidV0, Field: "required_tests"}},
		},
	}

	status := NewServerPublicStatusV0(state)
	if status.IdleSelfImprovementGoal == nil {
		t.Fatalf("goal publico nil")
	}
	goal := status.IdleSelfImprovementGoal
	if goal.Active ||
		goal.GoalRef != "goal-ref-public-001" ||
		goal.ExternalGoalRef != "external-goal-ref-public-001" ||
		goal.RequestRef != "request-ref-public-001" ||
		goal.RunRef != "run-ref-public-001" ||
		goal.ProjectRef != "project-ref-public-001" ||
		goal.WorkKind != "idle_self_improvement" ||
		goal.DirectorKind != orquestagoal.GoalDirectorKindCodexGoalV0 ||
		!goal.SpecPresent ||
		!goal.ReceiptPresent ||
		!goal.ResultPresent ||
		!goal.ClosurePresent ||
		goal.ReceiptStatus != orquestagoal.GoalStatusAcceptedV0 ||
		goal.ReceiptEvidenceRefCount != 1 ||
		goal.ResultStatus != orquestagoal.GoalStatusCompleteV0 ||
		goal.ResultRequiredTestPassed != 1 ||
		goal.ClosureStatus != orquestagoal.GoalStatusBlockedV0 ||
		!goal.ClosureNeedsRework ||
		goal.ClosureIssueCount != 1 ||
		goal.SpecWriteSetCount != 1 ||
		goal.SpecArtifactContractCount != 1 ||
		goal.ResultArtifactRefCount != 1 ||
		goal.ResultDomainReceiptRefCount != 1 {
		t.Fatalf("goal publico inesperado: %+v", goal)
	}
	payload, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	body := string(payload)
	for _, forbidden := range []string{
		"objetivo interno",
		"resumen interno",
		"private_file.go",
		"artifact-contract-ref-public-001",
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("status publico filtra %q en %s", forbidden, body)
		}
	}
	if !strings.Contains(body, "idle_self_improvement_goal") ||
		!strings.Contains(body, "goal-ref-public-001") {
		t.Fatalf("status publico sin goal compacto: %s", body)
	}
}

func TestServerPublicStatusV0NoMarcaActivoUnIntentoBloqueadoConReceiptRunningV0(t *testing.T) {
	state := StateV0{
		Status:       "running",
		StartupReady: true,
		IdleSelfImprovementOperationalMessage: &ServerOperationalMessageV0{
			SchemaVersion: serverOperationalMessageSchemaV0,
			Scope:         "idle_self_improvement",
			ReasonCode:    "attempt_blocked",
			Status:        "checked",
			GoalRefs:      []string{"goal-ref-public-blocked-001", "external-goal-ref-public-blocked-001"},
		},
		IdleSelfImprovementGoalSpec: &orquestagoal.GoalWorkSpecV0{
			SchemaVersion: orquestagoal.GoalWorkSpecSchemaV0,
			GoalRef:       "goal-ref-public-blocked-001",
			Objective:     "NO_DEBE_FILTRARSE blocked",
			DirectorKind:  orquestagoal.GoalDirectorKindCodexGoalV0,
		},
		IdleSelfImprovementGoalReceipt: &orquestagoal.GoalLaunchReceiptV0{
			SchemaVersion:   orquestagoal.GoalWorkLaunchReceiptSchemaV0,
			Status:          orquestagoal.GoalStatusRunningV0,
			GoalRef:         "goal-ref-public-blocked-001",
			ExternalGoalRef: "external-goal-ref-public-blocked-001",
		},
	}

	status := NewServerPublicStatusV0(state)
	readiness := NewServerReadinessV0(state)

	if status.IdleSelfImprovementGoal == nil {
		t.Fatalf("goal publico nil")
	}
	if status.IdleSelfImprovementGoal.Active ||
		status.IdleSelfImprovementGoal.OperationalReasonCode != "attempt_blocked" ||
		status.IdleSelfImprovementGoal.ReceiptStatus != orquestagoal.GoalStatusRunningV0 {
		t.Fatalf("goal publico inesperado: %+v", status.IdleSelfImprovementGoal)
	}
	if readiness.IdleSelfImprovementGoalActive ||
		readiness.IdleSelfImprovementGoalReasonCode != "attempt_blocked" ||
		readiness.IdleSelfImprovementGoalStatus != "checked" {
		t.Fatalf("readiness=%+v", readiness)
	}
	payload, _ := json.Marshal(status)
	if strings.Contains(string(payload), "NO_DEBE_FILTRARSE") {
		t.Fatalf("status filtra objetivo: %s", string(payload))
	}
}

func TestServerPublicStatusV0OmiteGoalFirstSiNoHayGoalV0(t *testing.T) {
	status := NewServerPublicStatusV0(StateV0{Status: "running"})
	if status.IdleSelfImprovementGoal != nil {
		t.Fatalf("goal publico debe omitirse: %+v", status.IdleSelfImprovementGoal)
	}
}

func TestServerPublicStatusV0OmiteGoalFirstSiSoloHayReasonSinGoalV0(t *testing.T) {
	state := StateV0{
		Status: "running",
		IdleSelfImprovementOperationalMessage: &ServerOperationalMessageV0{
			SchemaVersion: serverOperationalMessageSchemaV0,
			Scope:         "idle_self_improvement",
			ReasonCode:    idleSelfImprovementGoalLauncherUnavailableReasonV0,
			Status:        "checked",
		},
	}

	status := NewServerPublicStatusV0(state)
	readiness := NewServerReadinessV0(StateV0{
		Status:                                "running",
		StartupReady:                          true,
		IdleSelfImprovementOperationalMessage: state.IdleSelfImprovementOperationalMessage,
	})

	if status.IdleSelfImprovementGoal != nil {
		t.Fatalf("goal publico debe omitirse sin refs/estado durable: %+v", status.IdleSelfImprovementGoal)
	}
	if !readiness.Ready ||
		readiness.IdleSelfImprovementGoalActive ||
		readiness.IdleSelfImprovementGoalRef != "" ||
		readiness.IdleSelfImprovementGoalStatus != "" ||
		readiness.IdleSelfImprovementGoalReasonCode != "" {
		t.Fatalf("readiness=%+v", readiness)
	}
}
