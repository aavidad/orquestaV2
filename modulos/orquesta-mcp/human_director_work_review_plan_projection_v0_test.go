package orquestamcp

import (
	"testing"

	orquestaappdirectorintake "orquesta/modulos/orquesta-app-director-intake"
)

func TestHumanDirectorTaskCandidateMCPV0ProjectsAcceptanceChecks(t *testing.T) {
	candidate := humanDirectorTaskCandidateMCPV0(
		orquestaappdirectorintake.HumanDirectorReviewablePlanV0{},
		orquestaappdirectorintake.HumanDirectorPlanStepV0{
			StepRef: "step-ref-human-acceptance-001",
			Area:    "mcp",
			AcceptanceChecks: []orquestaappdirectorintake.HumanDirectorAcceptanceCheckV0{{
				CriterionRef: "criterion-ref-human-acceptance-001",
				Description:  "la comprobacion llega al candidato",
				Command:      "go test -count=1 ./modulos/orquesta-mcp",
			}},
		},
	)
	if len(candidate.AcceptanceChecks) != 1 ||
		candidate.AcceptanceChecks[0].CriterionRef != "criterion-ref-human-acceptance-001" ||
		candidate.AcceptanceChecks[0].Description != "la comprobacion llega al candidato" ||
		candidate.AcceptanceChecks[0].Command != "go test -count=1 ./modulos/orquesta-mcp" {
		t.Fatalf("acceptance_checks=%+v", candidate.AcceptanceChecks)
	}
}

func TestHumanDirectorTaskCandidateMCPV0LegacySinAcceptanceChecks(t *testing.T) {
	candidate := humanDirectorTaskCandidateMCPV0(
		orquestaappdirectorintake.HumanDirectorReviewablePlanV0{},
		orquestaappdirectorintake.HumanDirectorPlanStepV0{StepRef: "step-ref-human-legacy-001", Area: "mcp"},
	)
	if candidate.AcceptanceChecks != nil {
		t.Fatalf("acceptance_checks=%+v", candidate.AcceptanceChecks)
	}
}
