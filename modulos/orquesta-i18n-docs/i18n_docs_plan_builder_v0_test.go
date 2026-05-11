package orquestai18ndocs

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestBuildAppI18nDocsPlanV0DefaultSeedProducesValidPlan(t *testing.T) {
	plan := BuildAppI18nDocsPlanV0(PlanSeedV0{})

	if issues := ValidateAppI18nDocsPlanV0(plan); len(issues) != 0 {
		t.Fatalf("generated plan produced issues: %+v", issues)
	}
	if plan.WebBundle == nil || plan.FactoryBundle == nil || plan.DocsBundle == nil {
		t.Fatalf("expected web, factory and docs bundles by default: %+v", plan)
	}
	if plan.WebBundle.Catalogs[defaultLocaleV0].Messages["app.title"] == "" {
		t.Fatalf("expected default app.title message")
	}
	if strings.HasPrefix(plan.SkeletonLoaderShape.CatalogPathPattern, "/") {
		t.Fatalf("catalog path pattern must be relative: %s", plan.SkeletonLoaderShape.CatalogPathPattern)
	}
	if plan.SkeletonLoaderShape.StructureVersion != plan.WebBundle.StructureVersion || plan.SkeletonLoaderShape.StructureVersion != plan.FactoryBundle.StructureVersion {
		t.Fatalf("skeleton structure version must match i18n bundles")
	}
	requiredKeys := append([]string{}, plan.WebBundle.RequiredKeys...)
	requiredKeys = append(requiredKeys, plan.FactoryBundle.RequiredKeys...)
	if plan.SkeletonLoaderShape.RequiredKeysHash != requiredKeysHashV0(requiredKeys) {
		t.Fatalf("skeleton required_keys_hash is not derived from generated keys")
	}
}

func TestBuildAppI18nDocsPlanV0IsDeterministic(t *testing.T) {
	seed := PlanSeedV0{
		AppIDHint:          "panel-reservas",
		AppTitle:           "Panel de reservas",
		PrimaryActionLabel: "Crear reserva",
		UILocales:          []string{"es-ES", "en-US", "es-ES"},
		DocsLocales:        []string{"es-ES", "en-US"},
	}

	first := BuildAppI18nDocsPlanV0(seed)
	second := BuildAppI18nDocsPlanV0(seed)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("expected deterministic plans")
	}

	firstJSON, err := json.Marshal(first)
	if err != nil {
		t.Fatalf("marshal first plan: %v", err)
	}
	secondJSON, err := json.Marshal(second)
	if err != nil {
		t.Fatalf("marshal second plan: %v", err)
	}
	if string(firstJSON) != string(secondJSON) {
		t.Fatalf("expected deterministic JSON")
	}
}

func TestBuildAppI18nDocsPlanV0MatchesValidFixtureShape(t *testing.T) {
	fixture := validPlanV0(t)
	plan := BuildAppI18nDocsPlanV0(PlanSeedV0{
		AppIDHint:          fixture.AppIDHint,
		AppTitle:           fixture.WebBundle.Catalogs["es-ES"].Messages["app.title"],
		PrimaryActionLabel: fixture.WebBundle.Catalogs["es-ES"].Messages["app.action.primary"],
		UILocales:          fixture.WebBundle.Locales,
		DocsLocales:        fixture.DocsBundle.Locales,
		UIDefaultLocale:    fixture.UIDefaultLocale,
		DocsDefaultLocale:  fixture.DocsDefaultLocale,
	})

	if issues := ValidateAppI18nDocsPlanV0(plan); len(issues) != 0 {
		t.Fatalf("fixture-equivalent seed produced issues: %+v", issues)
	}
	if !reflect.DeepEqual(plan.WebBundle.RequiredKeys, fixture.WebBundle.RequiredKeys) {
		t.Fatalf("web required keys mismatch\nwant: %+v\ngot:  %+v", fixture.WebBundle.RequiredKeys, plan.WebBundle.RequiredKeys)
	}
	if !reflect.DeepEqual(plan.FactoryBundle.RequiredKeys, fixture.FactoryBundle.RequiredKeys) {
		t.Fatalf("factory required keys mismatch\nwant: %+v\ngot:  %+v", fixture.FactoryBundle.RequiredKeys, plan.FactoryBundle.RequiredKeys)
	}
	if !reflect.DeepEqual(plan.DocsBundle.RequiredDocTypes, fixture.DocsBundle.RequiredDocTypes) {
		t.Fatalf("required doc types mismatch\nwant: %+v\ngot:  %+v", fixture.DocsBundle.RequiredDocTypes, plan.DocsBundle.RequiredDocTypes)
	}
	if !reflect.DeepEqual(plan.SkeletonLoaderShape.RequiredNamespaces, fixture.SkeletonLoaderShape.RequiredNamespaces) {
		t.Fatalf("required namespaces mismatch\nwant: %+v\ngot:  %+v", fixture.SkeletonLoaderShape.RequiredNamespaces, plan.SkeletonLoaderShape.RequiredNamespaces)
	}
}

func TestBuildAppI18nDocsPlanV0NormalizesDefaultLocalesIntoCatalogs(t *testing.T) {
	plan := BuildAppI18nDocsPlanV0(PlanSeedV0{
		UILocales:         []string{"es-ES"},
		DocsLocales:       []string{"es-ES"},
		UIDefaultLocale:   "en-US",
		DocsDefaultLocale: "en-US",
	})

	if issues := ValidateAppI18nDocsPlanV0(plan); len(issues) != 0 {
		t.Fatalf("normalized plan produced issues: %+v", issues)
	}
	if _, ok := plan.WebBundle.Catalogs["en-US"]; !ok {
		t.Fatalf("expected ui default locale catalog")
	}
	if len(plan.DocsBundle.Docs) != len(plan.DocsBundle.Locales)*len(plan.DocsBundle.RequiredDocTypes) {
		t.Fatalf("expected all docs for all locales")
	}
}
