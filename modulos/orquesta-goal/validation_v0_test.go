package orquestagoal

import (
	"context"
	"testing"
)

func TestValidateGoalWorkSpecV0AceptaContratoGoalFirst(t *testing.T) {
	spec := GoalWorkSpecV0{
		GoalRef:       "goal-ref-autoprogramming-001",
		RequestRef:    "request-ref-001",
		Objective:     "Implementar una mejora acotada con pruebas.",
		DirectorKind:  GoalDirectorKindCodexGoalV0,
		WriteSet:      []GoalWriteScopeV0{{Path: "modulos/orquesta-goal"}},
		ContextRefs:   []GoalContextRefV0{{Kind: "doc", Ref: "docs/estado_actual_2026-05-17.md", Required: true}},
		RuleRefs:      []GoalRuleRefV0{{Ref: "AGENTS.md", Enforcement: GoalRuleEnforcementAdvisoryV0}},
		SkillRefs:     []string{"orquesta-programacion-tests"},
		RequiredTests: []GoalRequiredTestV0{{TestRef: "test-ref-goal", Command: "go test -count=1 ./modulos/orquesta-goal"}},
		ClosurePolicy: GoalClosurePolicyV0{RequireRequiredTests: true, RequiredEvidenceRefs: []string{"evidence-ref-goal-tests"}},
	}
	if issues := ValidateGoalWorkSpecV0(spec); len(issues) != 0 {
		t.Fatalf("issues=%v", issues)
	}
}

func TestNormalizeGoalWorkSpecV0UsaRuntimeGoalPorDefecto(t *testing.T) {
	spec := NormalizeGoalWorkSpecV0(GoalWorkSpecV0{
		GoalRef:   "goal-ref-001",
		Objective: "Objetivo",
		WriteSet:  []GoalWriteScopeV0{{Path: "docs"}},
	})
	if spec.SchemaVersion != GoalWorkSpecSchemaV0 {
		t.Fatalf("schema=%q", spec.SchemaVersion)
	}
	if spec.DirectorKind != GoalDirectorKindRuntimeGoalV0 {
		t.Fatalf("director=%q", spec.DirectorKind)
	}
}

func TestValidateGoalWorkSpecV0RechazaObjetivoVacio(t *testing.T) {
	issues := ValidateGoalWorkSpecV0(GoalWorkSpecV0{
		GoalRef:      "goal-ref-001",
		DirectorKind: GoalDirectorKindRuntimeGoalV0,
		WriteSet:     []GoalWriteScopeV0{{Path: "docs"}},
	})
	if !hasGoalIssueV0(issues, ErrGoalObjectiveRequiredV0) {
		t.Fatalf("issues=%v", issues)
	}
}

func TestValidateGoalWorkSpecV0RechazaWriteSetAbsoluto(t *testing.T) {
	issues := ValidateGoalWorkSpecV0(GoalWorkSpecV0{
		GoalRef:      "goal-ref-001",
		Objective:    "Objetivo",
		DirectorKind: GoalDirectorKindRuntimeGoalV0,
		WriteSet:     []GoalWriteScopeV0{{Path: "/home/alberto/Trabajo/orquesta"}},
	})
	if !hasGoalIssueV0(issues, ErrGoalWriteSetInvalidV0) {
		t.Fatalf("issues=%v", issues)
	}
}

func TestValidateGoalWorkSpecV0RechazaTraversalEnWriteSet(t *testing.T) {
	for _, path := range []string{"../docs", "docs/..", "docs/../cmd"} {
		issues := ValidateGoalWorkSpecV0(GoalWorkSpecV0{
			GoalRef:      "goal-ref-001",
			Objective:    "Objetivo",
			DirectorKind: GoalDirectorKindRuntimeGoalV0,
			WriteSet:     []GoalWriteScopeV0{{Path: path}},
		})
		if !hasGoalIssueV0(issues, ErrGoalWriteSetInvalidV0) {
			t.Fatalf("path=%q issues=%v", path, issues)
		}
	}
}

func TestValidateGoalWorkSpecV0RechazaContextRefAbsoluta(t *testing.T) {
	issues := ValidateGoalWorkSpecV0(GoalWorkSpecV0{
		GoalRef:      "goal-ref-001",
		Objective:    "Objetivo",
		DirectorKind: GoalDirectorKindRuntimeGoalV0,
		WriteSet:     []GoalWriteScopeV0{{Path: "docs"}},
		ContextRefs:  []GoalContextRefV0{{Ref: "/tmp/orquesta-context.md"}},
	})
	if !hasGoalIssueV0(issues, ErrGoalRefFieldInvalidV0) {
		t.Fatalf("issues=%v", issues)
	}
}

func TestValidateGoalWorkClosureV0ExigeEvidenciasTestsYArtefactos(t *testing.T) {
	spec := GoalWorkSpecV0{
		GoalRef:           "goal-ref-001",
		Objective:         "Objetivo",
		WriteSet:          []GoalWriteScopeV0{{Path: "docs"}},
		RequiredTests:     []GoalRequiredTestV0{{TestRef: "test-ref-001"}},
		ArtifactContracts: []GoalArtifactContractV0{{ArtifactRef: "artifact-ref-001", ArtifactType: "doc", Required: true}},
		ClosurePolicy: GoalClosurePolicyV0{
			RequireRequiredTests: true,
			RequireArtifacts:     true,
			RequiredEvidenceRefs: []string{"evidence-ref-001"},
		},
	}
	result := GoalWorkResultV0{
		GoalRef:             "goal-ref-001",
		Status:              GoalStatusCompleteV0,
		EvidenceRefs:        []string{"evidence-ref-001"},
		ArtifactRefs:        []string{"artifact-ref-001"},
		RequiredTestResults: []GoalRequiredTestResultV0{{TestRef: "test-ref-001", Status: "passed"}},
	}
	validation := ValidateGoalWorkClosureV0(spec, result)
	if !validation.Accepted {
		t.Fatalf("validation=%+v", validation)
	}
}

func TestValidateGoalWorkClosureV0RechazaGoalRefDistinto(t *testing.T) {
	spec := GoalWorkSpecV0{
		GoalRef:   "goal-ref-001",
		Objective: "Objetivo",
		WriteSet:  []GoalWriteScopeV0{{Path: "docs"}},
	}
	validation := ValidateGoalWorkClosureV0(spec, GoalWorkResultV0{Status: GoalStatusCompleteV0, GoalRef: "goal-ref-otro"})
	if validation.Accepted || !validation.NeedsRework {
		t.Fatalf("validation=%+v", validation)
	}
}

func TestValidateGoalWorkClosureV0RechazaSpecInvalida(t *testing.T) {
	validation := ValidateGoalWorkClosureV0(
		GoalWorkSpecV0{GoalRef: "goal-ref-001"},
		GoalWorkResultV0{Status: GoalStatusCompleteV0, GoalRef: "goal-ref-001"},
	)

	if validation.Accepted ||
		!validation.NeedsRework ||
		!hasGoalIssueV0(validation.Issues, ErrGoalObjectiveRequiredV0) ||
		!hasGoalIssueV0(validation.Issues, ErrGoalWriteSetRequiredV0) {
		t.Fatalf("validation=%+v", validation)
	}
}

func TestValidateGoalWorkClosureV0RechazaGoalRefVacio(t *testing.T) {
	spec := GoalWorkSpecV0{
		GoalRef:   "goal-ref-001",
		Objective: "Objetivo",
		WriteSet:  []GoalWriteScopeV0{{Path: "docs"}},
	}
	validation := ValidateGoalWorkClosureV0(spec, GoalWorkResultV0{Status: GoalStatusCompleteV0})
	if validation.Accepted || !validation.NeedsRework || !hasGoalIssueFieldV0(validation.Issues, "goal_ref") {
		t.Fatalf("validation=%+v", validation)
	}
}

func TestDefaultGoalWorkClosureValidatorV0ImplementaPuerto(t *testing.T) {
	spec := GoalWorkSpecV0{
		GoalRef:   "goal-ref-001",
		Objective: "Objetivo",
		WriteSet:  []GoalWriteScopeV0{{Path: "docs"}},
	}
	var validator GoalWorkClosureValidatorPortV0 = DefaultGoalWorkClosureValidatorV0{}
	validation, err := validator.ValidateGoalWorkClosureV0(context.Background(), spec, GoalWorkResultV0{Status: GoalStatusCompleteV0, GoalRef: "goal-ref-001"})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if !validation.Accepted {
		t.Fatalf("validation=%+v", validation)
	}
}

func TestValidateGoalWorkClosureV0NoAceptaCompleteSinEvidencia(t *testing.T) {
	spec := GoalWorkSpecV0{
		GoalRef:       "goal-ref-001",
		Objective:     "Objetivo",
		WriteSet:      []GoalWriteScopeV0{{Path: "docs"}},
		ClosurePolicy: GoalClosurePolicyV0{RequiredEvidenceRefs: []string{"evidence-ref-001"}},
	}
	validation := ValidateGoalWorkClosureV0(spec, GoalWorkResultV0{Status: GoalStatusCompleteV0, GoalRef: "goal-ref-001"})
	if validation.Accepted || !validation.NeedsRework || !hasGoalIssueFieldV0(validation.Issues, "evidence_refs") {
		t.Fatalf("validation=%+v", validation)
	}
}

func TestValidateGoalObservationRequestV0RechazaRefsInvalidas(t *testing.T) {
	issues := ValidateGoalObservationRequestV0(GoalObservationRequestV0{
		GoalRef:         "",
		ExternalGoalRef: "/tmp/external-goal",
	})

	if !hasGoalIssueFieldV0(issues, "goal_ref") ||
		!hasGoalIssueFieldV0(issues, "external_goal_ref") {
		t.Fatalf("issues=%v", issues)
	}
}

func TestValidateGoalWorkResultV0RechazaRefsInvalidas(t *testing.T) {
	issues := ValidateGoalWorkResultV0(GoalWorkResultV0{
		Status:            GoalStatusCompleteV0,
		GoalRef:           "goal-ref-001",
		ExternalGoalRef:   "external-goal-ref-001",
		ArtifactRefs:      []string{"artifact-ref-001", "/tmp/artifact"},
		DomainReceiptRefs: []string{"receipt-ref-001"},
		EvidenceRefs:      []string{"evidence-ref-001"},
		RequiredTestResults: []GoalRequiredTestResultV0{{
			TestRef:      "test-ref-001",
			Status:       "passed",
			EvidenceRefs: []string{"/tmp/test-evidence"},
		}},
	})

	if !hasGoalIssueFieldV0(issues, "artifact_refs") ||
		!hasGoalIssueFieldV0(issues, "required_test_results.evidence_refs") {
		t.Fatalf("issues=%v", issues)
	}
}

func TestValidateGoalWorkResultV0RechazaAcceptedComoEstadoObservado(t *testing.T) {
	issues := ValidateGoalWorkResultV0(GoalWorkResultV0{
		Status:  GoalStatusAcceptedV0,
		GoalRef: "goal-ref-001",
	})

	if !hasGoalIssueFieldV0(issues, "status") {
		t.Fatalf("issues=%v", issues)
	}
}

func TestValidateGoalWorkClosureV0RechazaResultadoConRefsInvalidas(t *testing.T) {
	spec := GoalWorkSpecV0{
		GoalRef:   "goal-ref-001",
		Objective: "Objetivo",
		WriteSet:  []GoalWriteScopeV0{{Path: "docs"}},
	}
	validation := ValidateGoalWorkClosureV0(spec, GoalWorkResultV0{
		Status:       GoalStatusCompleteV0,
		GoalRef:      "goal-ref-001",
		EvidenceRefs: []string{"/tmp/raw-evidence"},
	})

	if validation.Accepted || !validation.NeedsRework || !hasGoalIssueFieldV0(validation.Issues, "evidence_refs") {
		t.Fatalf("validation=%+v", validation)
	}
}

func hasGoalIssueV0(issues []GoalWorkIssueV0, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

func hasGoalIssueFieldV0(issues []GoalWorkIssueV0, field string) bool {
	for _, issue := range issues {
		if issue.Field == field {
			return true
		}
	}
	return false
}
