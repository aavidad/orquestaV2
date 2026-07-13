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

func TestValidateGoalWorkSpecV0RejectsUppercaseIntentManifestSHA256V0(t *testing.T) {
	spec := GoalWorkSpecV0{GoalRef: "goal-ref-intent-001", RequestRef: "request-001", Objective: "objetivo", DirectorKind: GoalDirectorKindCodexGoalV0, WriteSet: []GoalWriteScopeV0{{Path: "docs"}}, IntentManifestRef: "intent-manifest-ref-request-001", IntentManifestSHA256: strings.Repeat("A", 64)}
	if !hasGoalIssueV0(ValidateGoalWorkSpecV0(spec), ErrGoalRefFieldInvalidV0) {
		t.Fatal("uppercase manifest hash accepted")
	}
}

func TestValidateGoalWorkSpecV0VinculaManifestConRequestRefV0(t *testing.T) {
	base := GoalWorkSpecV0{
		GoalRef: "goal-ref-intent-link-001", RequestRef: "request-ref-intent-link-001",
		Objective: "objetivo", DirectorKind: GoalDirectorKindCodexGoalV0,
		WriteSet:             []GoalWriteScopeV0{{Path: "docs"}},
		IntentManifestRef:    "intent-manifest-ref-request-ref-intent-link-001",
		IntentManifestSHA256: strings.Repeat("a", 64),
	}
	if issues := ValidateGoalWorkSpecV0(base); len(issues) != 0 {
		t.Fatalf("valid manifest contract rejected: %v", issues)
	}
	cases := []GoalWorkSpecV0{base, base, base}
	cases[0].RequestRef = ""
	cases[1].IntentManifestRef = "intent-manifest-ref-request-ref-other"
	cases[2].IntentManifestSHA256 = ""
	for i, spec := range cases {
		if !hasGoalIssueV0(ValidateGoalWorkSpecV0(spec), ErrGoalRefFieldInvalidV0) {
			t.Fatalf("case %d accepted: %+v", i, spec)
		}
	}
}

func TestValidateGoalWorkSpecV0RechazaTraversalYRefsNoPortablesEnManifestV0(t *testing.T) {
	base := GoalWorkSpecV0{
		GoalRef: "goal-ref-intent-portable-001", RequestRef: "request-ref-intent-portable-001",
		Objective: "objetivo", DirectorKind: GoalDirectorKindCodexGoalV0,
		WriteSet:             []GoalWriteScopeV0{{Path: "docs"}},
		IntentManifestRef:    "intent-manifest-ref-request-ref-intent-portable-001",
		IntentManifestSHA256: strings.Repeat("a", 64),
	}
	for name, mutate := range map[string]func(*GoalWorkSpecV0){
		"request traversal": func(spec *GoalWorkSpecV0) {
			spec.RequestRef = "request-ref-a..b"
			spec.IntentManifestRef = "intent-manifest-ref-" + spec.RequestRef
		},
		"manifest traversal": func(spec *GoalWorkSpecV0) {
			spec.RequestRef = "../outside"
			spec.IntentManifestRef = "intent-manifest-ref-../outside"
		},
		"uppercase": func(spec *GoalWorkSpecV0) {
			spec.RequestRef = "REQUEST-REF-UPPER"
			spec.IntentManifestRef = "intent-manifest-ref-" + spec.RequestRef
		},
		"too long": func(spec *GoalWorkSpecV0) {
			spec.RequestRef = "request-ref-" + strings.Repeat("x", goalIntentManifestRequestRefMaxBytesV0)
			spec.IntentManifestRef = "intent-manifest-ref-" + spec.RequestRef
		},
	} {
		t.Run(name, func(t *testing.T) {
			spec := base
			mutate(&spec)
			if !hasGoalIssueV0(ValidateGoalWorkSpecV0(spec), ErrGoalRefFieldInvalidV0) {
				t.Fatalf("manifest path identity accepted: %+v", spec)
			}
		})
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
		ClosurePolicy:     GoalClosurePolicyV0{RequiredAcceptanceCriteriaRefs: []string{" criterion-ref-closure-001 "}, RequiredEvidenceRefs: []string{" evidence-ref-closure-001 "}},
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
	normalized.ClosurePolicy.RequiredAcceptanceCriteriaRefs[0] = "changed"
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
		spec.ClosurePolicy.RequiredAcceptanceCriteriaRefs[0] != " criterion-ref-closure-001 " ||
		spec.ClosurePolicy.RequiredEvidenceRefs[0] != " evidence-ref-closure-001 " {
		t.Fatalf("NormalizeGoalWorkSpecV0 muto el spec de entrada: %+v", spec)
	}
}

func TestValidateGoalWorkSpecV0RequiredAcceptanceCriteriaRefsRequireAttestation(t *testing.T) {
	spec := goalSpecWithRequiredAcceptanceCriterionForTestV0()
	spec.ClosurePolicy.RequireIndependentRequiredTestAttestation = false
	issues := ValidateGoalWorkSpecV0(spec)
	if !hasGoalIssueFieldCodeV0(issues, "closure_policy.require_independent_required_test_attestation", ErrGoalRequiredTestAttestationMissingV0) {
		t.Fatalf("issues=%v", issues)
	}
}

func TestValidateGoalWorkSpecV0RequiredAcceptanceCriterionWithoutFrozenTestCoverageIsInvalid(t *testing.T) {
	spec := goalSpecWithRequiredAcceptanceCriterionForTestV0()
	spec.ClosurePolicy.RequiredAcceptanceCriteriaRefs = []string{"criterion-ref-unmapped-001"}
	issues := ValidateGoalWorkSpecV0(spec)
	if !hasGoalIssueFieldCodeV0(issues, "closure_policy.required_acceptance_criteria_refs", ErrGoalRequiredAcceptanceCriterionAttestationMissingV0) {
		t.Fatalf("issues=%v", issues)
	}
}

func TestValidateGoalWorkSpecV0RequiredAcceptanceCriterionMappedToFrozenTestIsValid(t *testing.T) {
	if issues := ValidateGoalWorkSpecV0(goalSpecWithRequiredAcceptanceCriterionForTestV0()); len(issues) != 0 {
		t.Fatalf("issues=%v", issues)
	}
}

func TestNormalizeGoalWorkSpecV0DeduplicatesRequiredAcceptanceCriteriaRefs(t *testing.T) {
	spec := NormalizeGoalWorkSpecV0(GoalWorkSpecV0{
		ClosurePolicy: GoalClosurePolicyV0{RequiredAcceptanceCriteriaRefs: []string{" criterion-ref-001 ", "criterion-ref-001"}},
	})
	if len(spec.ClosurePolicy.RequiredAcceptanceCriteriaRefs) != 1 || spec.ClosurePolicy.RequiredAcceptanceCriteriaRefs[0] != "criterion-ref-001" {
		t.Fatalf("refs=%v", spec.ClosurePolicy.RequiredAcceptanceCriteriaRefs)
	}
}

func TestValidateGoalWorkSpecV0LegacyClosurePolicyWithoutRequiredAcceptanceCriteriaRefsIsValid(t *testing.T) {
	spec := GoalWorkSpecV0{
		GoalRef: "goal-ref-legacy-001", Objective: "Legacy contract remains valid.",
		DirectorKind: GoalDirectorKindRuntimeGoalV0, WriteSet: []GoalWriteScopeV0{{Path: "modulos/orquesta-goal"}},
	}
	if issues := ValidateGoalWorkSpecV0(spec); len(issues) != 0 {
		t.Fatalf("issues=%v", issues)
	}
}

func goalSpecWithRequiredAcceptanceCriterionForTestV0() GoalWorkSpecV0 {
	writeSet := []GoalWriteScopeV0{{Path: "modulos/orquesta-goal"}}
	requiredTest := FreezeGoalRequiredTestV0(GoalRequiredTestV0{
		TestRef: "test-ref-criterion-001", CommandRef: "command-ref-criterion-001", Command: "go test ./modulos/orquesta-goal",
		AcceptanceCriteriaRefs: []string{"criterion-ref-001"},
	})
	return GoalWorkSpecV0{
		GoalRef: "goal-ref-criterion-001", Objective: "Attest required acceptance criteria.",
		DirectorKind: GoalDirectorKindRuntimeGoalV0, WriteSet: writeSet, WriteSetSHA256: GoalWriteSetSHA256V0(writeSet),
		RequiredTests: []GoalRequiredTestV0{requiredTest},
		ClosurePolicy: GoalClosurePolicyV0{
			RequireIndependentRequiredTestAttestation: true,
			RequiredAcceptanceCriteriaRefs:            []string{"criterion-ref-001"},
		},
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

func TestValidateGoalWorkClosureV0BloqueaArtifactPathsFueraDeWriteSet(t *testing.T) {
	spec := GoalWorkSpecV0{
		GoalRef:   "goal-ref-001",
		Objective: "Objetivo",
		WriteSet:  []GoalWriteScopeV0{{Path: "docs"}},
	}
	validation := ValidateGoalWorkClosureV0(spec, GoalWorkResultV0{
		GoalRef:       "goal-ref-001",
		Status:        GoalStatusCompleteV0,
		ArtifactPaths: []string{"docs/orquesta_goal_result_v0.json", "html/index.html"},
	})

	if validation.Accepted ||
		!validation.NeedsRework ||
		!hasGoalIssueFieldCodeV0(validation.Issues, "artifact_paths", ErrGoalArtifactPathScopeV0) {
		t.Fatalf("validation=%+v", validation)
	}
}

func TestValidateGoalWorkClosureV0BloqueaMaterializedArtifactsFueraDeWriteSet(t *testing.T) {
	spec := GoalWorkSpecV0{
		GoalRef:   "goal-ref-001",
		Objective: "Objetivo",
		WriteSet:  []GoalWriteScopeV0{{Path: "docs"}},
	}
	validation := ValidateGoalWorkClosureV0(spec, GoalWorkResultV0{
		GoalRef: "goal-ref-001",
		Status:  GoalStatusCompleteV0,
		MaterializedArtifacts: []GoalMaterializedArtifactV0{{
			ArtifactRef:  "artifact-ref-html-extra",
			Path:         "html/index.html",
			ArtifactType: "html",
			Status:       GoalMaterializedArtifactStatusValidV0,
		}},
	})

	if validation.Accepted ||
		!validation.NeedsRework ||
		!hasGoalIssueFieldCodeV0(validation.Issues, "materialized_artifacts.path", ErrGoalArtifactPathScopeV0) {
		t.Fatalf("validation=%+v", validation)
	}
}

func TestValidateGoalWorkClosureV0ExigeArtifactPathsCuandoLaPoliticaLoPide(t *testing.T) {
	spec := GoalWorkSpecV0{
		GoalRef:       "goal-ref-001",
		Objective:     "Objetivo",
		WriteSet:      []GoalWriteScopeV0{{Path: "docs"}},
		ClosurePolicy: GoalClosurePolicyV0{RequireArtifactPaths: true},
	}
	validation := ValidateGoalWorkClosureV0(spec, GoalWorkResultV0{
		GoalRef: "goal-ref-001",
		Status:  GoalStatusCompleteV0,
	})

	if validation.Accepted ||
		!validation.NeedsRework ||
		!hasGoalIssueFieldCodeV0(validation.Issues, "artifact_paths", ErrGoalClosureInvalidV0) {
		t.Fatalf("validation=%+v", validation)
	}
}

func TestValidateGoalWorkClosureV0AceptaArtifactPathsDescendientesDeWriteSet(t *testing.T) {
	spec := GoalWorkSpecV0{
		GoalRef:       "goal-ref-001",
		Objective:     "Objetivo",
		ClosurePolicy: GoalClosurePolicyV0{RequireArtifactPaths: true},
		WriteSet: []GoalWriteScopeV0{
			{Path: "docs"},
			{Path: "trabajo/validacion"},
		},
	}
	validation := ValidateGoalWorkClosureV0(spec, GoalWorkResultV0{
		GoalRef:       "goal-ref-001",
		Status:        GoalStatusCompleteV0,
		ArtifactPaths: []string{"docs/orquesta_goal_result_v0.json", "trabajo/validacion/informe.json"},
	})

	if !validation.Accepted {
		t.Fatalf("validation=%+v", validation)
	}
}

func TestValidateGoalWorkClosureV0BloqueaCompleteConArtefactosParciales(t *testing.T) {
	spec := GoalWorkSpecV0{
		GoalRef:   "goal-ref-001",
		Objective: "Objetivo",
		WriteSet:  []GoalWriteScopeV0{{Path: "trabajo"}},
		ClosurePolicy: GoalClosurePolicyV0{
			RequireMaterializedArtifacts:         true,
			RequireChecklist:                     true,
			RequireReworkPlanForPartialArtifacts: true,
		},
	}
	result := GoalWorkResultV0{
		GoalRef: "goal-ref-001",
		Status:  GoalStatusCompleteV0,
		MaterializedArtifacts: []GoalMaterializedArtifactV0{{
			ArtifactRef:  "artifact-ref-tema-001-borrador",
			Path:         "trabajo/tema_001.md",
			ArtifactType: "markdown",
			Status:       GoalMaterializedArtifactStatusPartialV0,
			EvidenceRefs: []string{"evidence-ref-artifact-partial"},
		}},
		Checklist: GoalWorkChecklistV0{
			ExpectedRefs:  []string{"check-ref-texto-publicable", "check-ref-qa"},
			CompletedRefs: []string{"check-ref-texto-publicable"},
			MissingRefs:   []string{"check-ref-qa"},
			EvidenceRefs:  []string{"evidence-ref-checklist"},
		},
		EvidenceRefs: []string{"evidence-ref-goal"},
	}

	validation := ValidateGoalWorkClosureV0(spec, result)
	if validation.Accepted ||
		!validation.NeedsRework ||
		!hasGoalIssueFieldCodeV0(validation.Issues, "materialized_artifacts.status", ErrGoalMaterializedArtifactInvalidV0) ||
		!hasGoalIssueFieldCodeV0(validation.Issues, "checklist.missing_refs", ErrGoalChecklistIncompleteV0) ||
		!hasGoalIssueFieldCodeV0(validation.Issues, "rework_plan_refs", ErrGoalReworkPlanRequiredV0) ||
		!hasGoalStringV0(validation.EvidenceRefs, "evidence-ref-artifact-partial") ||
		!hasGoalStringV0(validation.EvidenceRefs, "evidence-ref-checklist") {
		t.Fatalf("validation=%+v", validation)
	}
}

func TestValidateGoalWorkClosureV0BlockedConArtefactosParcialesExponeCausa(t *testing.T) {
	spec := GoalWorkSpecV0{
		GoalRef:   "goal-ref-001",
		Objective: "Objetivo",
		WriteSet:  []GoalWriteScopeV0{{Path: "trabajo"}},
	}
	result := GoalWorkResultV0{
		GoalRef: "goal-ref-001",
		Status:  GoalStatusBlockedV0,
		MaterializedArtifacts: []GoalMaterializedArtifactV0{{
			Path:         "trabajo/tema_001.md",
			ArtifactType: "markdown",
			Status:       GoalMaterializedArtifactStatusNonPublishableV0,
			Issues:       []GoalWorkIssueV0{{Code: "qa_failed_public_text", Field: "public_text"}},
		}},
		ReworkPlanRefs: []string{"rework-plan-ref-tema-001"},
	}

	validation := ValidateGoalWorkClosureV0(spec, result)
	if validation.Accepted ||
		!validation.NeedsRework ||
		!hasGoalIssueFieldCodeV0(validation.Issues, "materialized_artifacts.status", ErrGoalMaterializedArtifactInvalidV0) ||
		!hasGoalIssueFieldV0(validation.Issues, "status") {
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
		GoalRef:              "",
		ExternalGoalRef:      "/tmp/external-goal",
		RuntimeGenerationRef: "/tmp/generation",
	})

	if !hasGoalIssueFieldV0(issues, "goal_ref") ||
		!hasGoalIssueFieldV0(issues, "external_goal_ref") ||
		!hasGoalIssueFieldV0(issues, "runtime_generation_ref") {
		t.Fatalf("issues=%v", issues)
	}
}

func TestValidateGoalLaunchReceiptV0RechazaGeneracionInvalidaV0(t *testing.T) {
	issues := ValidateGoalLaunchReceiptV0(GoalLaunchReceiptV0{
		Status: GoalStatusRunningV0, GoalRef: "goal-ref-generation-validation-001",
		ExternalGoalRef:      "thread-ref-generation-validation-001",
		RuntimeGenerationRef: "/tmp/generation",
	})
	if !hasGoalIssueFieldV0(issues, "runtime_generation_ref") {
		t.Fatalf("issues=%v", issues)
	}
}

func TestValidateGoalWorkResultV0RechazaRefsInvalidas(t *testing.T) {
	issues := ValidateGoalWorkResultV0(GoalWorkResultV0{
		Status:            GoalStatusCompleteV0,
		GoalRef:           "goal-ref-001",
		ExternalGoalRef:   "external-goal-ref-001",
		ArtifactRefs:      []string{"artifact-ref-001", "/tmp/artifact"},
		ArtifactPaths:     []string{"docs/ok.md", "/tmp/artifact.md"},
		DomainReceiptRefs: []string{"receipt-ref-001"},
		EvidenceRefs:      []string{"evidence-ref-001"},
		RequiredTestResults: []GoalRequiredTestResultV0{{
			TestRef:      "test-ref-001",
			Status:       "passed",
			EvidenceRefs: []string{"/tmp/test-evidence"},
		}},
	})

	if !hasGoalIssueFieldV0(issues, "artifact_refs") ||
		!hasGoalIssueFieldV0(issues, "artifact_paths") ||
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

func hasGoalStringV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
