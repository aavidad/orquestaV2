package orquestaautoprogramming

import "testing"

func TestMatchAutoprogrammingCuratedSkillRefsV0CasaPorClaseDeTarea(t *testing.T) {
	result := MatchAutoprogrammingCuratedSkillRefsV0(
		[]AutoprogrammingCuratedSkillV0{{
			Name:        "orquesta-programacion-tests",
			Description: "Pruebas focales, suite completa y evidencia de validacion.",
			Tags:        []string{"tests", "validacion"},
		}},
		AutoprogrammingCuratedSkillMatchRequestV0{
			Area:           "modulos-orquesta-autoprogramming",
			FailureSummary: "Anadir matcher de habilidades curadas.",
			RequiredTests:  []string{"go test -count=1 ./modulos/orquesta-autoprogramming"},
		},
	)

	want := "skill-ref-orquesta-programacion-tests-v0"
	if len(result.Issues) != 0 {
		t.Fatalf("issues=%+v", result.Issues)
	}
	if !containsAutoprogrammingStringForTestV0(result.SkillRefs, want) {
		t.Fatalf("skill_refs=%+v want %s", result.SkillRefs, want)
	}
	if !containsAutoprogrammingStringForTestV0(result.ContextRefs, AutoprogrammingCuratedSkillContextRefPrefixV0+want) {
		t.Fatalf("context_refs=%+v", result.ContextRefs)
	}
}

func TestMatchAutoprogrammingCuratedSkillRefsV0SinCatalogoNoCambia(t *testing.T) {
	result := MatchAutoprogrammingCuratedSkillRefsV0(
		nil,
		AutoprogrammingCuratedSkillMatchRequestV0{
			RequiredTests: []string{"go test -count=1 ./..."},
		},
	)

	if len(result.SkillRefs) != 0 || len(result.ContextRefs) != 0 || len(result.Issues) != 0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestValidateAutoprogrammingCuratedSkillCatalogV0RechazaSecretosYRutasAbsolutas(t *testing.T) {
	issues := ValidateAutoprogrammingCuratedSkillCatalogV0([]AutoprogrammingCuratedSkillV0{{
		Name:        "orquesta-programacion-tests",
		Description: "Plantilla local en /home/alberto/privado con token=secreto.",
		Tags:        []string{"tests"},
	}})

	if len(issues) == 0 {
		t.Fatalf("esperaba issue por ruta absoluta/secreto")
	}
	if issues[0].Code != "curated_skill_sensitive_detail" {
		t.Fatalf("issues=%+v", issues)
	}
}

func TestBuildAutoprogrammingSkillDistillationReviewProposalV0NoAutoCommit(t *testing.T) {
	proposal := BuildAutoprogrammingSkillDistillationReviewProposalV0(
		AutoprogrammingSkillDistillationCandidateV0{
			SourceResultRef: "result-ref-goal-001",
			SourceRunRef:    "run-ref-goal-001",
			SkillName:       "orquesta-programacion-tests",
			Summary:         "Patron reutilizable de test focal y suite completa.",
			Tags:            []string{"tests"},
			EvidenceRefs:    []string{"evidence-ref-goal-accepted"},
		},
	)

	if !proposal.Accepted || !proposal.ReviewRequired || proposal.AutoCommitSkills {
		t.Fatalf("proposal=%+v", proposal)
	}
	if !containsAutoprogrammingStringForTestV0(proposal.SuggestedWriteSet, "skills/docs") {
		t.Fatalf("write_set=%+v", proposal.SuggestedWriteSet)
	}
	if containsAutoprogrammingStringForTestV0(proposal.SuggestedWriteSet, "skills/orquesta-programacion-tests/SKILL.md") {
		t.Fatalf("auto-commit de skill no permitido: %+v", proposal.SuggestedWriteSet)
	}
}

func containsAutoprogrammingStringForTestV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
