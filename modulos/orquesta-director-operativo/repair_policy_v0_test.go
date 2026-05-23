package orquestadirectoroperativo

import "testing"

func TestDecideOperationalDirectorRepairPolicyV0PrefersRepairForUsefulOutput(t *testing.T) {
	decision := DecideOperationalDirectorRepairPolicyV0(OperationalDirectorRepairRequestV0{
		OutputRef:           "delivery-ref-001",
		Reasonable:          true,
		NormalizableAliases: []string{"web_application:web_app", "modulos/orquesta-director-operativo/**"},
		Cause:               "aliases seguros detectados",
	})

	if decision.Action != OperationalDirectorRepairNormalizeV0 ||
		decision.StepStatus != OperationalDirectorStepChangesRequestedV0 ||
		!decision.PreserveOutput ||
		!decision.RequiresFollowup ||
		decision.Cause != "aliases seguros detectados" {
		t.Fatalf("decision=%+v", decision)
	}
}

func TestDecideOperationalDirectorRepairPolicyV0MapsAmbiguousOutputToReview(t *testing.T) {
	decision := DecideOperationalDirectorRepairPolicyV0(OperationalDirectorRepairRequestV0{
		OutputRef:  "delivery-ref-ambiguous",
		Reasonable: true,
		Ambiguous:  true,
	})

	if decision.Action != OperationalDirectorRepairDelegateReviewV0 ||
		decision.StepStatus != OperationalDirectorStepChangesRequestedV0 ||
		!decision.PreserveOutput {
		t.Fatalf("decision=%+v", decision)
	}
}

func TestDecideOperationalDirectorRepairPolicyV0HardRejectsOnlyUnsafeOutput(t *testing.T) {
	decision := DecideOperationalDirectorRepairPolicyV0(OperationalDirectorRepairRequestV0{
		OutputRef:                  "delivery-ref-unsafe",
		Reasonable:                 true,
		NormalizableAliases:        []string{"alias"},
		UnauthorizedExternalEffect: true,
	})

	if decision.Action != OperationalDirectorRepairHardRejectV0 ||
		decision.StepStatus != OperationalDirectorStepBlockedV0 ||
		decision.PreserveOutput ||
		len(decision.Issues) != 1 ||
		decision.Issues[0].Code != "unauthorized_external_effect" {
		t.Fatalf("decision=%+v", decision)
	}
}

func TestBuildOperationalDirectorPlanV0CarriesGeneralRepairPolicy(t *testing.T) {
	result := BuildOperationalDirectorPlanV0(validProgrammingRequestV0(nil))

	if !result.Accepted || !result.Plan.RepairPolicy.PreferRepair ||
		!repairActionInSetV0(result.Plan.RepairPolicy.RepairActions, OperationalDirectorRepairNormalizeV0) ||
		!repairActionInSetV0(result.Plan.RepairPolicy.RepairActions, OperationalDirectorRepairPostponeV0) {
		t.Fatalf("plan repair policy=%+v result=%+v", result.Plan.RepairPolicy, result)
	}
	reviewStep := planStepByKindV0(result.Plan, OperationalDirectorStepReviewDeliveriesV0)
	if !containsFragmentInSetV0(reviewStep.AcceptanceCriteria, "Reparar antes que rechazar") ||
		!containsFragmentInSetV0(reviewStep.AcceptanceCriteria, "Cortar fuerte solo por seguridad") {
		t.Fatalf("review step=%+v", reviewStep)
	}
}

func repairActionInSetV0(
	values []OperationalDirectorRepairActionV0,
	want OperationalDirectorRepairActionV0,
) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
