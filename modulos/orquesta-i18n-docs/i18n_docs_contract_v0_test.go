package orquestai18ndocs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateAppI18nDocsPlanV0FixtureMinimoValido(t *testing.T) {
	plan := readPlanFixtureV0(t, "plan_minimo_valido.json")
	if issues := ValidateAppI18nDocsPlanV0(plan); len(issues) != 0 {
		t.Fatalf("fixture valido produjo issues: %+v", issues)
	}
}

func TestValidateAppI18nDocsPlanV0RejectsDefaultLocaleOutsideLocales(t *testing.T) {
	tests := map[string]func(AppI18nDocsPlanV0) AppI18nDocsPlanV0{
		"web_bundle": func(plan AppI18nDocsPlanV0) AppI18nDocsPlanV0 {
			plan.WebBundle.DefaultLocale = "en-US"
			return plan
		},
		"factory_bundle": func(plan AppI18nDocsPlanV0) AppI18nDocsPlanV0 {
			plan.FactoryBundle.DefaultLocale = "en-US"
			return plan
		},
		"docs_bundle": func(plan AppI18nDocsPlanV0) AppI18nDocsPlanV0 {
			plan.DocsBundle.DefaultLocale = "en-US"
			return plan
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			issues := ValidateAppI18nDocsPlanV0(mutate(validPlanV0(t)))
			if !hasI18nDocsIssueCodeV0(issues, ErrDefaultLocaleFueraDeCatalogo) {
				t.Fatalf("expected %s, got %+v", ErrDefaultLocaleFueraDeCatalogo, issues)
			}
		})
	}
}

func TestValidateAppI18nDocsPlanV0RejectsCatalogParityMismatch(t *testing.T) {
	plan := validPlanV0(t)
	plan.WebBundle.Locales = append(plan.WebBundle.Locales, "en-US")

	issues := ValidateAppI18nDocsPlanV0(plan)
	if !hasI18nDocsIssueCodeV0(issues, ErrLocaleSinCatalogo) {
		t.Fatalf("expected missing catalog issue, got %+v", issues)
	}

	plan = validPlanV0(t)
	plan.WebBundle.Catalogs["en-US"] = plan.WebBundle.Catalogs["es-ES"]
	issues = ValidateAppI18nDocsPlanV0(plan)
	if !hasI18nDocsIssueCodeV0(issues, ErrLocaleSinCatalogo) {
		t.Fatalf("expected undeclared catalog issue, got %+v", issues)
	}
}

func TestValidateAppI18nDocsPlanV0RejectsMissingRequiredKeyInCatalog(t *testing.T) {
	plan := validPlanV0(t)
	delete(plan.WebBundle.Catalogs["es-ES"].Messages, "app.title")

	issues := ValidateAppI18nDocsPlanV0(plan)
	if !hasI18nDocsIssueCodeV0(issues, ErrClaveRequeridaFaltante) {
		t.Fatalf("expected %s, got %+v", ErrClaveRequeridaFaltante, issues)
	}
}

func TestValidateAppI18nDocsPlanV0RejectsMissingRequiredDocTypeByLocale(t *testing.T) {
	plan := validPlanV0(t)
	plan.DocsBundle.Locales = append(plan.DocsBundle.Locales, "en-US")

	issues := ValidateAppI18nDocsPlanV0(plan)
	if !hasI18nDocsIssueCodeV0(issues, ErrTipoDocumentoFaltante) {
		t.Fatalf("expected %s, got %+v", ErrTipoDocumentoFaltante, issues)
	}
}

func TestValidateAppI18nDocsPlanV0RejectsFallbackLocaleOutsideBundles(t *testing.T) {
	plan := validPlanV0(t)
	plan.SkeletonLoaderShape.FallbackLocale = "en-US"

	issues := ValidateAppI18nDocsPlanV0(plan)
	if !hasI18nDocsIssueCodeV0(issues, ErrFallbackLocaleInvalido) {
		t.Fatalf("expected %s, got %+v", ErrFallbackLocaleInvalido, issues)
	}
}

func TestValidateAppI18nDocsPlanV0RejectsSkeletonLoaderShapeMismatch(t *testing.T) {
	plan := validPlanV0(t)
	plan.SkeletonLoaderShape.RequiredKeysHash = "sha256:desalineado"

	issues := ValidateAppI18nDocsPlanV0(plan)
	if !hasI18nDocsIssueCodeV0(issues, ErrSkeletonLoaderDesalineado) {
		t.Fatalf("expected %s, got %+v", ErrSkeletonLoaderDesalineado, issues)
	}
}

func TestValidateGeneratedDocV0RejectsMissingLocaleTitleSectionAndDuplicateOrder(t *testing.T) {
	doc := GeneratedDocV0{
		DocID:      "doc-1",
		DocType:    "user_manual",
		ContentKey: "docs.user.content",
		Format:     "markdown",
		Sections: []DocSectionV0{
			{SectionID: "overview", TitleKey: "docs.user.overview.title", ContentKey: "docs.user.overview.content", Order: 1},
			{SectionID: "details", ContentKey: "docs.user.details.content", Order: 1},
		},
	}

	issues := ValidateGeneratedDocV0("docs_bundle.docs[0]", doc)
	for _, code := range []string{ErrGeneratedDocInvalido, ErrGeneratedDocSeccionInvalida, ErrGeneratedDocOrdenDuplicado} {
		if !hasI18nDocsIssueCodeV0(issues, code) {
			t.Fatalf("expected %s, got %+v", code, issues)
		}
	}
}

func validPlanV0(t *testing.T) AppI18nDocsPlanV0 {
	t.Helper()
	raw := readFixtureBytesV0(t, "plan_minimo_valido.json")
	var plan AppI18nDocsPlanV0
	if err := json.Unmarshal(raw, &plan); err != nil {
		t.Fatalf("decode valid plan fixture: %v", err)
	}
	return plan
}

func readPlanFixtureV0(t *testing.T, name string) AppI18nDocsPlanV0 {
	t.Helper()
	raw := readFixtureBytesV0(t, name)
	plan, issues := DecodeAppI18nDocsPlanV0(raw)
	if len(issues) != 0 {
		t.Fatalf("decode fixture %s: %+v", name, issues)
	}
	return plan
}

func readFixtureBytesV0(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("docs", "fixtures", "i18n_docs_v0", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

func hasI18nDocsIssueCodeV0(issues []I18nDocsValidationIssueV0, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}
