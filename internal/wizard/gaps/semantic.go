package gaps

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"orquesta/internal/wizard/catalog"
)

const evaluatorSchema = "orquesta.wizard.gaps.evaluator"
const evaluatorV1Version = "v1"

var evaluatorV1Contract = []string{
	"typed_facts_only_no_prose_classification.v1",
	"equal_selection_dedup_distinct_selection_conflict.v1",
	"explicit_pack_refs_foundation_implicit.v1",
	"universal_u1_u12_then_conditional_technical_t1_t8.v1",
	"rules_r1_r8_ordered_after_dimension_detection.v1",
	"question_dependencies_semantic_transitive_reopen.v1",
	"technical_defaults_disclosed_never_implicitly_applied.v1",
	"issues_questions_defaults_pack_refs_canonical_order.v1",
	"free_text_explicit_option_utf8_4096_runes.v1",
}

type semanticEvaluator struct {
	Schema         string               `json:"schema"`
	Version        string               `json:"version"`
	SourceDigest   string               `json:"source_digest"`
	ResultSchema   string               `json:"result_schema"`
	Contract       []string             `json:"contract"`
	Inventory      semanticInventory    `json:"inventory"`
	CatalogVersion string               `json:"catalog_version"`
	CatalogDigest  string               `json:"catalog_digest"`
	BehaviorCorpus []semanticCorpusCase `json:"behavior_corpus"`
}

type semanticInventory struct {
	SchemaVersion string              `json:"schema_version"`
	Dimensions    []semanticDimension `json:"dimensions"`
	Rules         []semanticRule      `json:"rules"`
}

type semanticDimension struct {
	Ref          DimensionRef     `json:"ref"`
	Layer        Layer            `json:"layer"`
	Ordinal      int              `json:"ordinal"`
	Slot         SlotKey          `json:"slot"`
	DecisionKind DecisionKind     `json:"decision_kind"`
	DependsOn    []DimensionRef   `json:"depends_on"`
	PromptKey    MessageKey       `json:"prompt_key"`
	WhyKey       MessageKey       `json:"why_key"`
	HelpKey      MessageKey       `json:"help_key"`
	ExampleKey   MessageKey       `json:"example_key"`
	Default      OptionRef        `json:"default"`
	Options      []semanticOption `json:"options"`
}

type semanticOption struct {
	Ref          OptionRef  `json:"ref"`
	Kind         OptionKind `json:"kind"`
	LabelKey     MessageKey `json:"label_key"`
	HelpKey      MessageKey `json:"help_key"`
	ExampleKey   MessageKey `json:"example_key"`
	RationaleKey MessageKey `json:"rationale_key"`
	Recommended  bool       `json:"recommended"`
}

type semanticRule struct {
	Ref       RuleRef        `json:"ref"`
	Ordinal   int            `json:"ordinal"`
	DetailKey MessageKey     `json:"detail_key"`
	DependsOn []DimensionRef `json:"depends_on"`
	Targets   []DimensionRef `json:"targets"`
}

func evaluatorV1SemanticDigest() string {
	inventory := BuiltInV1()
	catalogV1 := catalog.BuiltInV1()
	snapshot := semanticEvaluator{
		Schema: evaluatorSchema, Version: evaluatorV1Version,
		SourceDigest: evaluatorV1SourceDigestGolden,
		ResultSchema: SchemaVersion,
		Contract:     append([]string(nil), evaluatorV1Contract...),
		Inventory: semanticInventory{
			SchemaVersion: inventory.SchemaVersion(),
		},
		CatalogVersion: catalogV1.Version().String(),
		CatalogDigest:  catalog.SemanticDigest(catalogV1),
		BehaviorCorpus: evaluatorV1BehaviorCorpus(),
	}
	for _, dimension := range inventory.Dimensions() {
		projected := semanticDimension{
			Ref: dimension.Ref(), Layer: dimension.Layer(),
			Ordinal: dimension.Ordinal(), Slot: dimension.Slot(),
			DecisionKind: dimension.DecisionKind(),
			DependsOn:    append([]DimensionRef(nil), dimension.DependsOn()...),
			PromptKey:    dimension.PromptKey(), WhyKey: dimension.WhyKey(),
			HelpKey: dimension.HelpKey(), ExampleKey: dimension.ExampleKey(),
			Default: dimension.DefaultOption(),
		}
		for _, option := range dimension.Options() {
			projected.Options = append(projected.Options, semanticOption{
				Ref: option.Ref(), Kind: option.Kind(),
				LabelKey: option.LabelKey(), HelpKey: option.HelpKey(),
				ExampleKey: option.ExampleKey(), RationaleKey: option.RationaleKey(),
				Recommended: option.Recommended(),
			})
		}
		snapshot.Inventory.Dimensions = append(
			snapshot.Inventory.Dimensions,
			projected,
		)
	}
	for _, rule := range inventory.Rules() {
		snapshot.Inventory.Rules = append(snapshot.Inventory.Rules, semanticRule{
			Ref: rule.Ref(), Ordinal: rule.Ordinal(), DetailKey: rule.DetailKey(),
			DependsOn: append([]DimensionRef(nil), rule.DependsOn()...),
			Targets:   append([]DimensionRef(nil), rule.Targets()...),
		})
	}
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		panic(err)
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}
