package i18n

import (
	"sort"
	"strings"
	"testing"

	"orquesta/internal/application"
	"orquesta/internal/wizard/catalog"
	"orquesta/internal/wizard/gaps"
)

// TestWizardCatalogCoversEveryBuiltInPresentationKey protects the catalog
// boundary: built-in Wizard structures may add machine refs, but every key
// they can present must be translated in every bundled locale.
func TestWizardCatalogCoversEveryBuiltInPresentationKey(t *testing.T) {
	catalogue, err := LoadBundled()
	if err != nil {
		t.Fatal(err)
	}
	available := make(map[string]struct{}, len(catalogue.Keys()))
	for _, key := range catalogue.Keys() {
		available[key] = struct{}{}
	}
	missing := make([]string, 0)
	for _, key := range wizardPresentationKeys() {
		if _, found := available[key]; !found {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("Wizard presentation keys missing from bundled catalog: %s", strings.Join(missing, ", "))
	}
}

func TestWizardSpanishLabelsAreTranslated(t *testing.T) {
	catalogue, err := LoadBundled()
	if err != nil {
		t.Fatal(err)
	}
	allowedBrands := map[string]string{
		"wizard.gaps.dimension.t2.option.ldap.label": "LDAP",
		"wizard.gaps.dimension.t2.option.oidc.label": "OpenID Connect",
		"wizard.gaps.dimension.t5.option.grpc.label": "gRPC",
	}
	seenAllowed := make(map[string]struct{}, len(allowedBrands))
	for _, key := range catalogue.Keys() {
		if !strings.HasPrefix(key, "wizard.") || !strings.HasSuffix(key, ".label") {
			continue
		}
		spanish, spanishErr := catalogue.Text("es", key)
		english, englishErr := catalogue.Text("en", key)
		if spanishErr != nil || englishErr != nil {
			t.Fatalf("key=%s es_err=%v en_err=%v", key, spanishErr, englishErr)
		}
		if allowed, found := allowedBrands[key]; found {
			if spanish != allowed || english != allowed {
				t.Fatalf("brand key=%s es=%q en=%q want=%q", key, spanish, english, allowed)
			}
			seenAllowed[key] = struct{}{}
			continue
		}
		if spanish == english {
			t.Errorf("untranslated Spanish Wizard label key=%s value=%q", key, spanish)
		}
	}
	if len(seenAllowed) != len(allowedBrands) {
		t.Fatalf("brand allowlist coverage=%d want=%d", len(seenAllowed), len(allowedBrands))
	}
}

func wizardPresentationKeys() []string {
	keys := make(map[string]struct{})
	add := func(values ...string) {
		for _, key := range values {
			if key != "" {
				keys[key] = struct{}{}
			}
		}
	}
	for _, dimension := range gaps.BuiltIn().Dimensions() {
		add(string(dimension.PromptKey()), string(dimension.WhyKey()), string(dimension.HelpKey()), string(dimension.ExampleKey()))
		prefix := "wizard.gaps.dimension." + strings.ToLower(string(dimension.Ref())) + ".issue"
		add(prefix+".gap", prefix+".contradiction")
		for _, option := range dimension.Options() {
			add(string(option.LabelKey()), string(option.HelpKey()), string(option.ExampleKey()), string(option.RationaleKey()))
		}
		if dimension.Layer() == gaps.LayerTechnical {
			prefix := "wizard.gaps.default." + strings.ToLower(string(dimension.Ref()))
			add(prefix+".label", prefix+".help", prefix+".example")
		}
	}
	for _, rule := range gaps.BuiltIn().Rules() {
		prefix := "wizard.gaps.rule." + strings.ToLower(string(rule.Ref()))
		add(string(rule.DetailKey()), prefix+".gap", prefix+".contradiction")
	}
	for _, rule := range []struct {
		ref     string
		options []string
	}{
		{"r5", []string{"public_low", "service_auth_medium", "oauth_high", "custom"}},
		{"r8", []string{"internal_team", "invited_customers", "public_users", "custom"}},
	} {
		prefix := "wizard.gaps.rule." + rule.ref
		add(prefix+".prompt", prefix+".why", prefix+".help", prefix+".example")
		for _, option := range rule.options {
			optionPrefix := prefix + ".option." + option
			add(optionPrefix+".label", optionPrefix+".help", optionPrefix+".example", optionPrefix+".rationale")
		}
	}
	add(
		"wizard.gaps.pack.option.custom.label",
		"wizard.gaps.pack.option.custom.help",
		"wizard.gaps.pack.option.custom.example",
		"wizard.gaps.pack.option.custom.rationale",
	)
	for _, pack := range catalog.BuiltIn().Packs() {
		for _, question := range pack.Questions() {
			if !strings.HasPrefix(question.Slot().String(), "domains.") {
				continue
			}
			add(question.PromptKey().String(), question.WhyKey().String(), question.HelpKey().String(), question.ExampleKey().String())
			for _, option := range question.Options() {
				add(option.LabelKey().String(), option.HelpKey().String(), option.ExampleKey().String(), option.RationaleKey().String())
			}
		}
	}
	for _, section := range []application.IntakeDossierSectionKind{
		application.IntakeDossierSectionProductScope,
		application.IntakeDossierSectionUsersRoles,
		application.IntakeDossierSectionArchitecture,
		application.IntakeDossierSectionData,
		application.IntakeDossierSectionIntegrations,
		application.IntakeDossierSectionSecurityPrivacy,
		application.IntakeDossierSectionUIUX,
		application.IntakeDossierSectionI18NL10N,
		application.IntakeDossierSectionDeployOperations,
		application.IntakeDossierSectionOrchestrationPlan,
		application.IntakeDossierSectionRisksOpenIssues,
	} {
		add("intake.dossier." + string(section) + ".title")
	}
	for _, diagram := range []application.IntakeDossierDiagramPurpose{
		application.IntakeDossierDiagramArchitecture,
		application.IntakeDossierDiagramUserFlow,
		application.IntakeDossierDiagramDataIntegrations,
		application.IntakeDossierDiagramI18N,
		application.IntakeDossierDiagramDeployment,
		application.IntakeDossierDiagramDecisions,
	} {
		add("intake.dossier.diagram." + string(diagram) + ".alt")
	}
	values := make([]string, 0, len(keys))
	for key := range keys {
		values = append(values, key)
	}
	sort.Strings(values)
	return values
}
