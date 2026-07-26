package catalog

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

type semanticCatalog struct {
	Version       string         `json:"version"`
	FoundationRef string         `json:"foundation_ref"`
	Packs         []semanticPack `json:"packs"`
}

type semanticPack struct {
	Ref                   string             `json:"ref"`
	LabelKey              string             `json:"label_key"`
	HelpKey               string             `json:"help_key"`
	ExampleKey            string             `json:"example_key"`
	RoadmapCapabilityRefs []string           `json:"roadmap_capability_refs"`
	Taxonomies            []semanticTaxonomy `json:"taxonomies"`
	Questions             []semanticQuestion `json:"questions"`
	Defaults              []semanticDefault  `json:"defaults"`
}

type semanticTaxonomy struct {
	Ref        string         `json:"ref"`
	LabelKey   string         `json:"label_key"`
	HelpKey    string         `json:"help_key"`
	ExampleKey string         `json:"example_key"`
	Terms      []semanticTerm `json:"terms"`
}

type semanticTerm struct {
	Ref        string `json:"ref"`
	LabelKey   string `json:"label_key"`
	HelpKey    string `json:"help_key"`
	ExampleKey string `json:"example_key"`
}

type semanticQuestion struct {
	Ref          string           `json:"ref"`
	Slot         string           `json:"slot"`
	AnswerKind   AnswerKind       `json:"answer_kind"`
	DecisionKind DecisionKind     `json:"decision_kind"`
	TaxonomyRef  string           `json:"taxonomy_ref"`
	PromptKey    string           `json:"prompt_key"`
	WhyKey       string           `json:"why_key"`
	HelpKey      string           `json:"help_key"`
	ExampleKey   string           `json:"example_key"`
	Options      []semanticOption `json:"options"`
}

type semanticOption struct {
	Ref          string `json:"ref"`
	TermRef      string `json:"term_ref"`
	LabelKey     string `json:"label_key"`
	HelpKey      string `json:"help_key"`
	ExampleKey   string `json:"example_key"`
	RationaleKey string `json:"rationale_key"`
	Recommended  bool   `json:"recommended"`
}

type semanticDefault struct {
	Ref          string       `json:"ref"`
	Slot         string       `json:"slot"`
	Value        string       `json:"value"`
	DecisionKind DecisionKind `json:"decision_kind"`
	LabelKey     string       `json:"label_key"`
	HelpKey      string       `json:"help_key"`
	ExampleKey   string       `json:"example_key"`
	RationaleKey string       `json:"rationale_key"`
}

// SemanticDigest binds every machine-visible catalog decision in canonical
// order. Catalog constructors already normalize all collections.
func SemanticDigest(value Catalog) string {
	snapshot := semanticCatalog{
		Version: value.Version().String(), FoundationRef: value.FoundationRef().String(),
		Packs: make([]semanticPack, 0, len(value.Packs())),
	}
	for _, pack := range value.Packs() {
		item := semanticPack{
			Ref: pack.Ref().String(), LabelKey: pack.LabelKey().String(),
			HelpKey: pack.HelpKey().String(), ExampleKey: pack.ExampleKey().String(),
		}
		for _, ref := range pack.RoadmapCapabilityRefs() {
			item.RoadmapCapabilityRefs = append(item.RoadmapCapabilityRefs, ref.String())
		}
		for _, taxonomy := range pack.Taxonomies() {
			projected := semanticTaxonomy{
				Ref: taxonomy.Ref().String(), LabelKey: taxonomy.LabelKey().String(),
				HelpKey: taxonomy.HelpKey().String(), ExampleKey: taxonomy.ExampleKey().String(),
			}
			for _, term := range taxonomy.Terms() {
				projected.Terms = append(projected.Terms, semanticTerm{
					Ref: term.Ref().String(), LabelKey: term.LabelKey().String(),
					HelpKey: term.HelpKey().String(), ExampleKey: term.ExampleKey().String(),
				})
			}
			item.Taxonomies = append(item.Taxonomies, projected)
		}
		for _, question := range pack.Questions() {
			projected := semanticQuestion{
				Ref: question.Ref().String(), Slot: question.Slot().String(),
				AnswerKind: question.AnswerKind(), DecisionKind: question.DecisionKind(),
				TaxonomyRef: question.TaxonomyRef().String(),
				PromptKey:   question.PromptKey().String(), WhyKey: question.WhyKey().String(),
				HelpKey: question.HelpKey().String(), ExampleKey: question.ExampleKey().String(),
			}
			for _, option := range question.Options() {
				projected.Options = append(projected.Options, semanticOption{
					Ref: option.Ref().String(), TermRef: option.TermRef().String(),
					LabelKey: option.LabelKey().String(), HelpKey: option.HelpKey().String(),
					ExampleKey:   option.ExampleKey().String(),
					RationaleKey: option.RationaleKey().String(),
					Recommended:  option.Recommended(),
				})
			}
			item.Questions = append(item.Questions, projected)
		}
		for _, value := range pack.Defaults() {
			item.Defaults = append(item.Defaults, semanticDefault{
				Ref: value.Ref().String(), Slot: value.Slot().String(),
				Value: value.Value().String(), DecisionKind: value.DecisionKind(),
				LabelKey: value.LabelKey().String(), HelpKey: value.HelpKey().String(),
				ExampleKey:   value.ExampleKey().String(),
				RationaleKey: value.RationaleKey().String(),
			})
		}
		snapshot.Packs = append(snapshot.Packs, item)
	}
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		panic(err)
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}
