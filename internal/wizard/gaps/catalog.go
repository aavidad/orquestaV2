package gaps

type DimensionDescriptor struct {
	ref          DimensionRef
	layer        Layer
	ordinal      int
	slot         SlotKey
	decisionKind DecisionKind
	dependsOn    []DimensionRef
	promptKey    MessageKey
	whyKey       MessageKey
	helpKey      MessageKey
	exampleKey   MessageKey
	options      []Option
	defaultRef   OptionRef
}

func (value DimensionDescriptor) Ref() DimensionRef { return value.ref }
func (value DimensionDescriptor) Layer() Layer      { return value.layer }
func (value DimensionDescriptor) Ordinal() int      { return value.ordinal }
func (value DimensionDescriptor) Slot() SlotKey     { return value.slot }
func (value DimensionDescriptor) DecisionKind() DecisionKind {
	return value.decisionKind
}
func (value DimensionDescriptor) DependsOn() []DimensionRef {
	return append([]DimensionRef(nil), value.dependsOn...)
}
func (value DimensionDescriptor) PromptKey() MessageKey  { return value.promptKey }
func (value DimensionDescriptor) WhyKey() MessageKey     { return value.whyKey }
func (value DimensionDescriptor) HelpKey() MessageKey    { return value.helpKey }
func (value DimensionDescriptor) ExampleKey() MessageKey { return value.exampleKey }
func (value DimensionDescriptor) Options() []Option {
	return append([]Option(nil), value.options...)
}
func (value DimensionDescriptor) DefaultOption() OptionRef { return value.defaultRef }

type RuleDescriptor struct {
	ref       RuleRef
	ordinal   int
	detailKey MessageKey
	dependsOn []DimensionRef
	targets   []DimensionRef
}

func (value RuleDescriptor) Ref() RuleRef          { return value.ref }
func (value RuleDescriptor) Ordinal() int          { return value.ordinal }
func (value RuleDescriptor) DetailKey() MessageKey { return value.detailKey }
func (value RuleDescriptor) DependsOn() []DimensionRef {
	return append([]DimensionRef(nil), value.dependsOn...)
}
func (value RuleDescriptor) Targets() []DimensionRef {
	return append([]DimensionRef(nil), value.targets...)
}

// Inventory is immutable and versioned. It proves catalog coverage, not V23
// wiring, exercise or accreditation.
type Inventory struct {
	schemaVersion string
	dimensions    []DimensionDescriptor
	rules         []RuleDescriptor
}

func BuiltIn() Inventory {
	return Inventory{
		schemaVersion: SchemaVersion,
		dimensions:    builtInDimensions(),
		rules:         builtInRules(),
	}
}

func (value Inventory) SchemaVersion() string { return value.schemaVersion }
func (value Inventory) Dimensions() []DimensionDescriptor {
	return cloneDimensionDescriptors(value.dimensions)
}
func (value Inventory) Rules() []RuleDescriptor {
	out := append([]RuleDescriptor(nil), value.rules...)
	for index := range out {
		out[index].dependsOn = append([]DimensionRef(nil), out[index].dependsOn...)
		out[index].targets = append([]DimensionRef(nil), out[index].targets...)
	}
	return out
}

func builtInDimensions() []DimensionDescriptor {
	values := []DimensionDescriptor{
		dimension(
			DimensionU1, LayerUniversal, 1, "product.audience", nil,
			"team", "personal", "team", "public",
		),
		dimension(
			DimensionU2, LayerUniversal, 2, "product.platform", nil,
			"responsive_web", "responsive_web", "native_mobile", "native_desktop",
			"no_ui",
		),
		dimension(
			DimensionU3, LayerUniversal, 3, "product.identity",
			[]DimensionRef{DimensionU1},
			"external_oidc", "no_login", "local_login", "external_oidc",
		),
		dimension(
			DimensionU4, LayerUniversal, 4, "product.data_lifecycle", nil,
			"persisted_user_data", "persisted_user_data", "external_source", "ephemeral",
		),
		dimension(
			DimensionU5, LayerUniversal, 5, "product.privacy", nil,
			"minimized_personal", "no_personal", "minimized_personal",
			"sensitive", "regulated",
		),
		dimension(
			DimensionU6, LayerUniversal, 6, "product.collaboration",
			[]DimensionRef{DimensionU1},
			"in_app", "none", "in_app", "realtime", "external_notifications",
		),
		dimension(
			DimensionU7, LayerUniversal, 7, "product.integrations", nil,
			"external_services", "none", "external_services", "public_api",
		),
		dimension(
			DimensionU8, LayerUniversal, 8, "product.search", nil,
			"simple_search_tags", "structured_filters", "simple_search_tags", "full_text",
		),
		dimension(
			DimensionU9, LayerUniversal, 9, "product.language_accessibility", nil,
			"system_preferences", "system_preferences", "enhanced_accessibility",
			"additional_locales",
		),
		dimension(
			DimensionU10, LayerUniversal, 10, "product.deployment", nil,
			"cloud_container", "local_device", "own_server", "cloud_container",
			"distribution_store",
		),
		dimension(
			DimensionU11, LayerUniversal, 11, "product.scale",
			[]DimensionRef{DimensionU1},
			"normal_10x", "normal_10x", "realtime", "high_concurrency",
		),
		dimension(
			DimensionU12, LayerUniversal, 12, "product.history",
			[]DimensionRef{DimensionU4},
			"versioned_restore", "versioned_restore", "audit_only", "no_history",
		),
		dimension(
			DimensionT1, LayerTechnical, 13, "technical.access_control",
			[]DimensionRef{DimensionU1},
			"rbac_basic", "rbac_basic", "fine_acl", "rbac_acl_multitenant",
		),
		dimension(
			DimensionT2, LayerTechnical, 14, "technical.corporate_identity",
			[]DimensionRef{DimensionU1},
			"oidc", "oidc", "ldap", "saml_scim",
		),
		dimension(
			DimensionT3, LayerTechnical, 15, "technical.observability",
			[]DimensionRef{DimensionU10},
			"structured_rotated_health", "structured_rotated_health",
			"journald_metrics", "central_telemetry", "kernel_trace",
		),
		dimension(
			DimensionT4, LayerTechnical, 16, "technical.persistence",
			[]DimensionRef{DimensionU4},
			"sqlite_restore", "sqlite_restore", "postgres_restore", "mysql_or_kv",
		),
		dimension(
			DimensionT5, LayerTechnical, 17, "technical.api_contracts",
			[]DimensionRef{DimensionU7},
			"rest_openapi", "rest_openapi", "grpc", "graphql_events",
		),
		dimension(
			DimensionT6, LayerTechnical, 18, "technical.advanced_deployment",
			[]DimensionRef{DimensionU10},
			"container_systemd_ci", "container_systemd_ci", "managed_platform",
			"ha_dr", "kernel_build_ci",
		),
		dimension(
			DimensionT7, LayerTechnical, 19, "technical.resilience",
			[]DimensionRef{DimensionU7, DimensionU11},
			"bounded_io_pagination", "bounded_io_pagination", "latency_budget",
			"high_resilience",
		),
		dimension(
			DimensionT8, LayerTechnical, 20, "technical.compliance",
			[]DimensionRef{DimensionU5},
			"immutable_audit_rgpd", "immutable_audit_rgpd", "anonymization",
			"deletion_retention",
		),
	}
	return values
}

func builtInRules() []RuleDescriptor {
	return []RuleDescriptor{
		rule(RuleR1, 1, []DimensionRef{DimensionU1}, []DimensionRef{DimensionU1}),
		rule(RuleR2, 2, nil, []DimensionRef{DimensionU2}),
		rule(RuleR3, 3, []DimensionRef{DimensionU7}, nil),
		rule(RuleR4, 4, []DimensionRef{DimensionU4}, []DimensionRef{DimensionT4}),
		rule(RuleR5, 5, []DimensionRef{DimensionU7}, nil),
		rule(RuleR6, 6, []DimensionRef{DimensionU2}, []DimensionRef{DimensionU2}),
		rule(RuleR7, 7, []DimensionRef{DimensionU2}, []DimensionRef{DimensionU10}),
		rule(RuleR8, 8, []DimensionRef{DimensionU1}, nil),
	}
}

func dimension(
	ref DimensionRef,
	layer Layer,
	ordinal int,
	slot string,
	dependsOn []DimensionRef,
	recommended string,
	values ...string,
) DimensionDescriptor {
	prefix := "wizard.gaps.dimension." + lowerRef(ref)
	options := make([]Option, 0, len(values)+1)
	for _, value := range values {
		options = append(options, option(ref, value, value == recommended, OptionPreset))
	}
	options = append(options, option(ref, "custom", false, OptionFreeText))
	decisionKind := DecisionProduct
	defaultRef := OptionRef("")
	if layer == LayerTechnical {
		decisionKind = DecisionTechnical
		defaultRef = optionRef(ref, recommended)
	}
	return DimensionDescriptor{
		ref: ref, layer: layer, ordinal: ordinal, slot: SlotKey(slot),
		decisionKind: decisionKind,
		dependsOn:    append([]DimensionRef(nil), dependsOn...),
		promptKey:    MessageKey(prefix + ".prompt"),
		whyKey:       MessageKey(prefix + ".why"),
		helpKey:      MessageKey(prefix + ".help"),
		exampleKey:   MessageKey(prefix + ".example"),
		options:      options,
		defaultRef:   defaultRef,
	}
}

func option(
	dimension DimensionRef,
	value string,
	recommended bool,
	kind OptionKind,
) Option {
	prefix := "wizard.gaps.dimension." + lowerRef(dimension) + ".option." + value
	return Option{
		ref:          optionRef(dimension, value),
		kind:         kind,
		labelKey:     MessageKey(prefix + ".label"),
		helpKey:      MessageKey(prefix + ".help"),
		exampleKey:   MessageKey(prefix + ".example"),
		rationaleKey: MessageKey(prefix + ".rationale"),
		recommended:  recommended,
	}
}

func rule(
	ref RuleRef,
	ordinal int,
	dependsOn []DimensionRef,
	targets []DimensionRef,
) RuleDescriptor {
	return RuleDescriptor{
		ref: ref, ordinal: ordinal,
		detailKey: MessageKey("wizard.gaps.rule." + lowerRuleRef(ref) + ".detail"),
		dependsOn: append([]DimensionRef(nil), dependsOn...),
		targets:   append([]DimensionRef(nil), targets...),
	}
}

func optionRef(dimension DimensionRef, value string) OptionRef {
	return OptionRef("intake-option:wizard." + lowerRef(dimension) + "." + value)
}

func questionRef(dimension DimensionRef) QuestionRef {
	return QuestionRef("intake-question:wizard." + lowerRef(dimension))
}

func dimensionIssueRef(dimension DimensionRef) IssueRef {
	return IssueRef("intake-issue:wizard.dimension." + lowerRef(dimension))
}

func ruleIssueRef(ref RuleRef) IssueRef {
	return IssueRef("intake-issue:wizard.rule." + lowerRuleRef(ref))
}

func lowerRef(ref DimensionRef) string {
	if len(ref) < 2 {
		return ""
	}
	return string(ref[0]-'A'+'a') + string(ref[1:])
}

func lowerRuleRef(ref RuleRef) string {
	if len(ref) < 2 {
		return ""
	}
	return string(ref[0]-'A'+'a') + string(ref[1:])
}

func cloneDimensionDescriptors(values []DimensionDescriptor) []DimensionDescriptor {
	out := append([]DimensionDescriptor(nil), values...)
	for index := range out {
		out[index].dependsOn = append([]DimensionRef(nil), out[index].dependsOn...)
		out[index].options = append([]Option(nil), out[index].options...)
	}
	return out
}

func dimensionsByRef(values []DimensionDescriptor) map[DimensionRef]DimensionDescriptor {
	out := make(map[DimensionRef]DimensionDescriptor, len(values))
	for _, value := range values {
		out[value.ref] = value
	}
	return out
}
