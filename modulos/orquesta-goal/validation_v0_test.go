package orquestagoal

import (
	"context"
	"strings"
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

func TestNormalizeGoalWorkSpecV0NoMutaSlicesDeEntrada(t *testing.T) {
	spec := GoalWorkSpecV0{
		GoalRef:           " goal-ref-001 ",
		Objective:         " Objetivo ",
		ContextRefs:       []GoalContextRefV0{{Kind: " doc ", Ref: " context-ref-001 ", Purpose: " soporte "}},
		RuleRefs:          []GoalRuleRefV0{{Kind: " rule ", Ref: " rule-ref-001 ", Enforcement: " "}},
		SkillRefs:         []string{" skill-ref-001 "},
		WriteSet:          []GoalWriteScopeV0{{Path: " docs/foo.md ", Purpose: " editar "}},
		RequiredTests:     []GoalRequiredTestV0{{TestRef: " test-ref-001 ", CommandRef: " command-ref-001 ", Command: " go test ./... ", AcceptanceCriteria: []string{" criterio "}, AcceptanceCriteriaRefs: []string{" criteria-ref-001 "}, EvidenceRefs: []string{" evidence-ref-test-001 "}}},
		ArtifactContracts: []GoalArtifactContractV0{{ArtifactRef: " artifact-ref-001 ", ArtifactType: " markdown ", EvidenceRefs: []string{" evidence-ref-artifact-001 "}}},
		EvidenceRefs:      []string{" evidence-ref-spec-001 "},
		ClosurePolicy:     GoalClosurePolicyV0{RequiredEvidenceRefs: []string{" evidence-ref-closure-001 "}},
	}

	normalized := NormalizeGoalWorkSpecV0(spec)
	normalized.ContextRefs[0].Kind = "changed"
	normalized.RuleRefs[0].Ref = "changed"
	normalized.SkillRefs[0] = "changed"
	normalized.WriteSet[0].Path = "changed"
	normalized.RequiredTests[0].Command = "changed"
	normalized.RequiredTests[0].AcceptanceCriteria[0] = "changed"
	normalized.RequiredTests[0].AcceptanceCriteriaRefs[0] = "changed"
	normalized.RequiredTests[0].EvidenceRefs[0] = "changed"
	normalized.ArtifactContracts[0].ArtifactRef = "changed"
	normalized.ArtifactContracts[0].EvidenceRefs[0] = "changed"
	normalized.EvidenceRefs[0] = "changed"
	normalized.ClosurePolicy.RequiredEvidenceRefs[0] = "changed"

	if spec.ContextRefs[0].Kind != " doc " ||
		spec.RuleRefs[0].Ref != " rule-ref-001 " ||
		spec.SkillRefs[0] != " skill-ref-001 " ||
		spec.WriteSet[0].Path != " docs/foo.md " ||
		spec.RequiredTests[0].Command != " go test ./... " ||
		spec.RequiredTests[0].AcceptanceCriteria[0] != " criterio " ||
		spec.RequiredTests[0].AcceptanceCriteriaRefs[0] != " criteria-ref-001 " ||
		spec.RequiredTests[0].EvidenceRefs[0] != " evidence-ref-test-001 " ||
		spec.ArtifactContracts[0].ArtifactRef != " artifact-ref-001 " ||
		spec.ArtifactContracts[0].EvidenceRefs[0] != " evidence-ref-artifact-001 " ||
		spec.EvidenceRefs[0] != " evidence-ref-spec-001 " ||
		spec.ClosurePolicy.RequiredEvidenceRefs[0] != " evidence-ref-closure-001 " {
		t.Fatalf("NormalizeGoalWorkSpecV0 muto el spec de entrada: %+v", spec)
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

func TestValidateGoalWorkSpecV0RechazaObjetivoEnorme(t *testing.T) {
	issues := ValidateGoalWorkSpecV0(GoalWorkSpecV0{
		GoalRef:      "goal-ref-001",
		Objective:    strings.Repeat("x", GoalWorkSpecMaxStringBytesV0+1),
		DirectorKind: GoalDirectorKindRuntimeGoalV0,
		WriteSet:     []GoalWriteScopeV0{{Path: "docs"}},
	})
	if !hasGoalIssueFieldCodeV0(issues, "objective", ErrGoalSpecLimitExceededV0) {
		t.Fatalf("issues=%v", issues)
	}
}

func TestValidateGoalWorkSpecV0RechazaListaEnorme(t *testing.T) {
	skillRefs := make([]string, GoalWorkSpecMaxListItemsV0+1)
	for i := range skillRefs {
		skillRefs[i] = "skill-ref-goal"
	}
	issues := ValidateGoalWorkSpecV0(GoalWorkSpecV0{
		GoalRef:      "goal-ref-001",
		Objective:    "Objetivo",
		DirectorKind: GoalDirectorKindRuntimeGoalV0,
		WriteSet:     []GoalWriteScopeV0{{Path: "docs"}},
		SkillRefs:    skillRefs,
	})
	if !hasGoalIssueFieldCodeV0(issues, "skill_refs", ErrGoalSpecLimitExceededV0) {
		t.Fatalf("issues=%v", issues)
	}
}

func TestValidateGoalWorkSpecV0RechazaComandoDeTestEnorme(t *testing.T) {
	issues := ValidateGoalWorkSpecV0(GoalWorkSpecV0{
		GoalRef:      "goal-ref-001",
		Objective:    "Objetivo",
		DirectorKind: GoalDirectorKindRuntimeGoalV0,
		WriteSet:     []GoalWriteScopeV0{{Path: "docs"}},
		RequiredTests: []GoalRequiredTestV0{{
			TestRef: "test-ref-001",
			Command: strings.Repeat("x", GoalWorkSpecMaxCommandBytesV0+1),
		}},
	})
	if !hasGoalIssueFieldCodeV0(issues, "required_tests.command", ErrGoalSpecLimitExceededV0) {
		t.Fatalf("issues=%v", issues)
	}
}

func TestValidateGoalWorkSpecV0RechazaTamanoProyectadoEnorme(t *testing.T) {
	requiredTests := make([]GoalRequiredTestV0, GoalWorkSpecMaxListItemsV0)
	for i := range requiredTests {
		requiredTests[i] = GoalRequiredTestV0{
			TestRef: "test-ref-projected",
			Command: strings.Repeat("x", 600),
		}
	}
	issues := ValidateGoalWorkSpecV0(GoalWorkSpecV0{
		GoalRef:       "goal-ref-001",
		Objective:     "Objetivo",
		DirectorKind:  GoalDirectorKindRuntimeGoalV0,
		WriteSet:      []GoalWriteScopeV0{{Path: "docs"}},
		RequiredTests: requiredTests,
	})
	if !hasGoalIssueFieldCodeV0(issues, "goal_work_spec", ErrGoalSpecLimitExceededV0) {
		t.Fatalf("issues=%v", issues)
	}
}

func TestValidateGoalWorkSpecV0AceptaVocabularioOperativoRecuperable(t *testing.T) {
	spec := GoalWorkSpecV0{
		GoalRef:      "goal-ref-001",
		Objective:    "Revisar entrega marcada como capacity_limited, garbage o failed y conservar lo recuperable sin rail de detalle_prohibido.",
		DirectorKind: GoalDirectorKindRuntimeGoalV0,
		WorkKind:     "web_application",
		WriteSet:     []GoalWriteScopeV0{{Path: "modulos/orquesta-goal", Purpose: "normalizar alias y reparar formato recuperable"}},
		RequiredTests: []GoalRequiredTestV0{{
			TestRef:            "test-ref-goal",
			Command:            "go test -count=1 ./modulos/orquesta-goal",
			AcceptanceCriteria: []string{"No bloquear por vocabulario operativo recuperable ni por alias web_application."},
		}},
		AcceptanceCriteria: []string{"Los rails blandos quedan como evidencia o rework, no como veto automatico."},
	}
	if issues := ValidateGoalWorkSpecV0(spec); len(issues) != 0 {
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

func TestValidateGoalWorkClosureV0BloqueaInvalidConRework(t *testing.T) {
	spec := GoalWorkSpecV0{
		GoalRef:   "goal-ref-001",
		Objective: "Objetivo",
		WriteSet:  []GoalWriteScopeV0{{Path: "docs"}},
	}
	validation := ValidateGoalWorkClosureV0(spec, GoalWorkResultV0{
		Status:  GoalStatusInvalidV0,
		GoalRef: "goal-ref-001",
	})

	if validation.Accepted ||
		validation.Status != GoalStatusBlockedV0 ||
		!validation.NeedsRework ||
		!hasGoalIssueFieldV0(validation.Issues, "status") {
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

func hasGoalIssueFieldCodeV0(issues []GoalWorkIssueV0, field, code string) bool {
	for _, issue := range issues {
		if issue.Field == field && issue.Code == code {
			return true
		}
	}
	return false
}
