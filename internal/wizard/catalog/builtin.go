package catalog

const currentVersionValue = "orquesta.wizard.catalog.v1"

var domainKeys = []string{
	"billing",
	"booking",
	"calendar",
	"commerce",
	"community",
	"crm",
	"documents",
	"education",
	"finance",
	"health",
	"inventory",
	"iot",
	"maps",
	"media",
	"project_tasks",
}

func CurrentVersion() CatalogVersion {
	return mustCatalogVersion(currentVersionValue)
}

func FoundationPackRef() PackRef {
	return mustPackRef("pack:foundation")
}

// DomainPackRefs lists every built-in domain pack in stable reference order.
// Selection remains explicit: no prose or keyword matching occurs here.
func DomainPackRefs() []PackRef {
	refs := make([]PackRef, len(domainKeys))
	for index, key := range domainKeys {
		refs[index] = mustPackRef("pack:" + key)
	}
	return refs
}

// BuiltIn is the current catalog alias. Versioned evaluators must call the
// frozen constructor they were accredited with instead.
func BuiltIn() Catalog {
	return BuiltInV1()
}

// BuiltInV1 returns the catalog frozen for Wizard gaps evaluator V1. New
// catalog semantics require a new constructor; mutating this one breaks its
// semantic-digest golden test and therefore cannot silently rewrite replay.
func BuiltInV1() Catalog {
	packs := []Pack{foundationPack()}
	for _, key := range domainKeys {
		packs = append(packs, domainPack(key))
	}
	value, err := NewCatalog(CatalogInput{
		Version: CurrentVersion(), FoundationRef: FoundationPackRef(), Packs: packs,
	})
	if err != nil {
		panic(err)
	}
	return value
}

func foundationPack() Pack {
	taxonomies := []Taxonomy{
		builtinTaxonomy("work_mode", []string{"create_new", "work_existing"}),
		builtinTaxonomy("decision_kind", []string{"product", "technical_default"}),
		builtinTaxonomy("app_template", []string{"web_application", "api_service", "automation"}),
		builtinTaxonomy("quality_profile", []string{"standard", "regulated", "high_assurance"}),
		builtinTaxonomy("answer_kind", []string{"choice", "free_text"}),
		builtinTaxonomy("assistance_level", []string{"contextual", "explain_all"}),
		builtinTaxonomy("inference_mode", []string{"deterministic", "assistant_optional"}),
		builtinTaxonomy("domain_pack", domainKeys),
	}
	questions := []Question{
		builtinChoiceQuestion(
			"work_mode", "app.work_mode", "work_mode",
			[]string{"create_new", "work_existing"}, "create_new",
		),
		builtinFreeTextQuestion("objective", "app.objective"),
		builtinChoiceQuestion(
			"app_template", "app.template", "app_template",
			[]string{"web_application", "api_service", "automation"}, "web_application",
		),
		builtinChoiceQuestion(
			"quality_profile", "quality.profile", "quality_profile",
			[]string{"standard", "regulated", "high_assurance"}, "standard",
		),
		builtinChoiceQuestion(
			"assistance_level", "wizard.assistance_level", "assistance_level",
			[]string{"contextual", "explain_all"}, "contextual",
		),
		builtinChoiceQuestion(
			"inference_mode", "wizard.inference_mode", "inference_mode",
			[]string{"deterministic", "assistant_optional"}, "deterministic",
		),
	}
	defaults := []TechnicalDefault{
		builtinDefault("architecture", "technical.architecture", "hexagonal"),
		builtinDefault("i18n", "technical.i18n", "catalog"),
		builtinDefault("tests", "technical.tests", "unit_and_architecture"),
		builtinDefault("typed_errors", "technical.errors", "stable_codes"),
		builtinDefault("resilience", "technical.resilience", "bounded_io"),
	}
	return mustPack(PackInput{
		Ref:        FoundationPackRef(),
		LabelKey:   key("wizard.pack.foundation.label"),
		HelpKey:    key("wizard.pack.foundation.help"),
		ExampleKey: key("wizard.pack.foundation.example"),
		RoadmapCapabilityRefs: []RoadmapCapabilityRef{
			mustRoadmapCapabilityRef("WIZ-01"),
			mustRoadmapCapabilityRef("WIZ-06"),
			mustRoadmapCapabilityRef("WIZ-11"),
			mustRoadmapCapabilityRef("WIZ-16"),
			mustRoadmapCapabilityRef("WIZ-17"),
			mustRoadmapCapabilityRef("WIZ-19"),
			mustRoadmapCapabilityRef("WIZ-25"),
		},
		Taxonomies: taxonomies, Questions: questions, Defaults: defaults,
	})
}

func domainPack(domain string) Pack {
	integrationTaxonomyKey := "domain_" + domain + "_integration_mode"
	return mustPack(PackInput{
		Ref:        mustPackRef("pack:" + domain),
		LabelKey:   key("wizard.pack." + domain + ".label"),
		HelpKey:    key("wizard.pack." + domain + ".help"),
		ExampleKey: key("wizard.pack." + domain + ".example"),
		RoadmapCapabilityRefs: []RoadmapCapabilityRef{
			mustRoadmapCapabilityRef("WIZ-25"),
		},
		Taxonomies: []Taxonomy{
			builtinTaxonomy("data_sensitivity", []string{"public", "internal", "personal", "regulated"}),
			builtinTaxonomy(integrationTaxonomyKey, []string{"native", "sync", "deferred"}),
		},
		Questions: []Question{
			builtinChoiceQuestion(
				"data_sensitivity", "data.sensitivity", "data_sensitivity",
				[]string{"public", "internal", "personal", "regulated"}, "internal",
			),
			builtinChoiceQuestion(
				"domain_"+domain+"_integration_mode",
				"domains."+domain+".integration_mode",
				integrationTaxonomyKey,
				[]string{"native", "sync", "deferred"},
				"native",
			),
		},
	})
}

func builtinTaxonomy(name string, terms []string) Taxonomy {
	items := make([]Term, len(terms))
	for index, term := range terms {
		prefix := "wizard.taxonomy." + name + ".term." + term
		items[index] = mustTerm(TermInput{
			Ref:        mustTermRef("term:" + name + "." + term),
			LabelKey:   key(prefix + ".label"),
			HelpKey:    key(prefix + ".help"),
			ExampleKey: key(prefix + ".example"),
		})
	}
	prefix := "wizard.taxonomy." + name
	return mustTaxonomy(TaxonomyInput{
		Ref:        mustTaxonomyRef("taxonomy:" + name),
		LabelKey:   key(prefix + ".label"),
		HelpKey:    key(prefix + ".help"),
		ExampleKey: key(prefix + ".example"),
		Terms:      items,
	})
}

func builtinChoiceQuestion(
	name string,
	slot string,
	taxonomy string,
	terms []string,
	recommended string,
) Question {
	options := make([]Option, len(terms))
	for index, term := range terms {
		prefix := "wizard.question." + name + ".option." + term
		options[index] = mustOption(OptionInput{
			Ref:          mustOptionRef("option:" + name + "." + term),
			TermRef:      mustTermRef("term:" + taxonomy + "." + term),
			LabelKey:     key(prefix + ".label"),
			HelpKey:      key(prefix + ".help"),
			ExampleKey:   key(prefix + ".example"),
			RationaleKey: key(prefix + ".rationale"),
			Recommended:  term == recommended,
		})
	}
	prefix := "wizard.question." + name
	return mustQuestion(QuestionInput{
		Ref:          mustQuestionRef("question:" + name),
		Slot:         mustSlotKey(slot),
		AnswerKind:   AnswerChoice,
		DecisionKind: DecisionProduct,
		TaxonomyRef:  mustTaxonomyRef("taxonomy:" + taxonomy),
		PromptKey:    key(prefix + ".prompt"),
		WhyKey:       key(prefix + ".why"),
		HelpKey:      key(prefix + ".help"),
		ExampleKey:   key(prefix + ".example"),
		Options:      options,
	})
}

func builtinFreeTextQuestion(name, slot string) Question {
	prefix := "wizard.question." + name
	return mustQuestion(QuestionInput{
		Ref:          mustQuestionRef("question:" + name),
		Slot:         mustSlotKey(slot),
		AnswerKind:   AnswerFreeText,
		DecisionKind: DecisionProduct,
		PromptKey:    key(prefix + ".prompt"),
		WhyKey:       key(prefix + ".why"),
		HelpKey:      key(prefix + ".help"),
		ExampleKey:   key(prefix + ".example"),
	})
}

func builtinDefault(name, slot, value string) TechnicalDefault {
	prefix := "wizard.default." + name
	return mustDefault(TechnicalDefaultInput{
		Ref:          mustDefaultRef("default:" + name),
		Slot:         mustSlotKey(slot),
		Value:        mustTermRef("term:default." + value),
		LabelKey:     key(prefix + ".label"),
		HelpKey:      key(prefix + ".help"),
		ExampleKey:   key(prefix + ".example"),
		RationaleKey: key(prefix + ".rationale"),
	})
}

func key(value string) MessageKey {
	result, err := NewMessageKey(value)
	if err != nil {
		panic(err)
	}
	return result
}

func mustCatalogVersion(value string) CatalogVersion {
	result, err := NewCatalogVersion(value)
	if err != nil {
		panic(err)
	}
	return result
}

func mustRoadmapCapabilityRef(value string) RoadmapCapabilityRef {
	result, err := NewRoadmapCapabilityRef(value)
	if err != nil {
		panic(err)
	}
	return result
}

func mustPackRef(value string) PackRef {
	result, err := NewPackRef(value)
	if err != nil {
		panic(err)
	}
	return result
}

func mustTaxonomyRef(value string) TaxonomyRef {
	result, err := NewTaxonomyRef(value)
	if err != nil {
		panic(err)
	}
	return result
}

func mustTermRef(value string) TermRef {
	result, err := NewTermRef(value)
	if err != nil {
		panic(err)
	}
	return result
}

func mustQuestionRef(value string) QuestionRef {
	result, err := NewQuestionRef(value)
	if err != nil {
		panic(err)
	}
	return result
}

func mustOptionRef(value string) OptionRef {
	result, err := NewOptionRef(value)
	if err != nil {
		panic(err)
	}
	return result
}

func mustDefaultRef(value string) DefaultRef {
	result, err := NewDefaultRef(value)
	if err != nil {
		panic(err)
	}
	return result
}

func mustSlotKey(value string) SlotKey {
	result, err := NewSlotKey(value)
	if err != nil {
		panic(err)
	}
	return result
}

func mustTerm(input TermInput) Term {
	result, err := NewTerm(input)
	if err != nil {
		panic(err)
	}
	return result
}

func mustTaxonomy(input TaxonomyInput) Taxonomy {
	result, err := NewTaxonomy(input)
	if err != nil {
		panic(err)
	}
	return result
}

func mustOption(input OptionInput) Option {
	result, err := NewOption(input)
	if err != nil {
		panic(err)
	}
	return result
}

func mustQuestion(input QuestionInput) Question {
	result, err := NewQuestion(input)
	if err != nil {
		panic(err)
	}
	return result
}

func mustDefault(input TechnicalDefaultInput) TechnicalDefault {
	result, err := NewTechnicalDefault(input)
	if err != nil {
		panic(err)
	}
	return result
}

func mustPack(input PackInput) Pack {
	result, err := NewPack(input)
	if err != nil {
		panic(err)
	}
	return result
}
