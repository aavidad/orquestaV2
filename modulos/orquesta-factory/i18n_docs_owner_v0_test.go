package orquestafactory

import (
	i18ndocs "orquesta/modulos/orquesta-i18n-docs"
	"testing"
	"time"
)

func TestFactoryBuildI18nDocsPlanFromAppSpecV0ConsumesActiveOwner(t *testing.T) {
	spec, issues := SolicitarNuevaAppV0(validMinimalRequestV0(), time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC))
	if len(issues) != 0 {
		t.Fatalf("spec issues: %+v", issues)
	}

	plan, planIssues := BuildI18nDocsPlanFromAppSpecV0(spec)
	if len(planIssues) != 0 {
		t.Fatalf("i18n docs plan issues: %+v", planIssues)
	}
	if plan.SourceContract != appSpecSourceContractTestV0 {
		t.Fatalf("source contract=%q", plan.SourceContract)
	}
	if plan.WebBundle == nil || plan.FactoryBundle == nil || plan.DocsBundle == nil {
		t.Fatalf("plan missing bundles: %+v", plan)
	}

	owner, ownerIssues := I18nDocsActiveOwnerForAppSpecV0(spec)
	if len(ownerIssues) != 0 {
		t.Fatalf("owner issues: %+v", ownerIssues)
	}
	if owner.OwnerModule != i18ndocs.ActiveI18nDocsOwnerModuleV0 || owner.RequiredKeysHash != plan.SkeletonLoaderShape.RequiredKeysHash {
		t.Fatalf("owner projection mismatch: %+v plan=%+v", owner, plan.SkeletonLoaderShape)
	}
}

const appSpecSourceContractTestV0 = "AppSpecV0"
