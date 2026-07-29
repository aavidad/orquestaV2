package i18n

import (
	"reflect"
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

func TestWizardBaseHelpExampleKeyRatchet(t *testing.T) {
	questions := 0
	options := 0
	actual := make([]string, 0, 46)
	for _, question := range catalog.BuiltIn().ComposeAll().Questions() {
		if strings.HasPrefix(question.Slot().String(), "domains.") {
			continue
		}
		questions++
		actual = append(actual, question.HelpKey().String(), question.ExampleKey().String())
		for _, option := range question.Options() {
			options++
			actual = append(actual, option.HelpKey().String(), option.ExampleKey().String())
		}
	}
	sort.Strings(actual)
	expected := wizardBaseHelpExampleKeys()
	sort.Strings(expected)
	if questions != 7 || options != 16 || len(actual) != 46 {
		t.Fatalf("base Wizard presentation shape questions=%d options=%d keys=%d want=7/16/46", questions, options, len(actual))
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("base Wizard help/example keys:\ngot  %q\nwant %q", actual, expected)
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
	add(wizardBaseHelpExampleKeys()...)
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

func wizardBaseHelpExampleKeys() []string {
	return []string{
		"wizard.question.app_template.example",
		"wizard.question.app_template.help",
		"wizard.question.app_template.option.api_service.example",
		"wizard.question.app_template.option.api_service.help",
		"wizard.question.app_template.option.automation.example",
		"wizard.question.app_template.option.automation.help",
		"wizard.question.app_template.option.web_application.example",
		"wizard.question.app_template.option.web_application.help",
		"wizard.question.assistance_level.example",
		"wizard.question.assistance_level.help",
		"wizard.question.assistance_level.option.contextual.example",
		"wizard.question.assistance_level.option.contextual.help",
		"wizard.question.assistance_level.option.explain_all.example",
		"wizard.question.assistance_level.option.explain_all.help",
		"wizard.question.data_sensitivity.example",
		"wizard.question.data_sensitivity.help",
		"wizard.question.data_sensitivity.option.internal.example",
		"wizard.question.data_sensitivity.option.internal.help",
		"wizard.question.data_sensitivity.option.personal.example",
		"wizard.question.data_sensitivity.option.personal.help",
		"wizard.question.data_sensitivity.option.public.example",
		"wizard.question.data_sensitivity.option.public.help",
		"wizard.question.data_sensitivity.option.regulated.example",
		"wizard.question.data_sensitivity.option.regulated.help",
		"wizard.question.inference_mode.example",
		"wizard.question.inference_mode.help",
		"wizard.question.inference_mode.option.assistant_optional.example",
		"wizard.question.inference_mode.option.assistant_optional.help",
		"wizard.question.inference_mode.option.deterministic.example",
		"wizard.question.inference_mode.option.deterministic.help",
		"wizard.question.objective.example",
		"wizard.question.objective.help",
		"wizard.question.quality_profile.example",
		"wizard.question.quality_profile.help",
		"wizard.question.quality_profile.option.high_assurance.example",
		"wizard.question.quality_profile.option.high_assurance.help",
		"wizard.question.quality_profile.option.regulated.example",
		"wizard.question.quality_profile.option.regulated.help",
		"wizard.question.quality_profile.option.standard.example",
		"wizard.question.quality_profile.option.standard.help",
		"wizard.question.work_mode.example",
		"wizard.question.work_mode.help",
		"wizard.question.work_mode.option.create_new.example",
		"wizard.question.work_mode.option.create_new.help",
		"wizard.question.work_mode.option.work_existing.example",
		"wizard.question.work_mode.option.work_existing.help",
	}
}
