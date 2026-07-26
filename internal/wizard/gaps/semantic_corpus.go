package gaps

import (
	"fmt"
	"strings"

	"orquesta/internal/intake"
	"orquesta/internal/wizard/catalog"
)

type semanticCorpusCase struct {
	Ref    string          `json:"ref"`
	Input  semanticInput   `json:"input"`
	Result *semanticResult `json:"result,omitempty"`
	Error  *semanticError  `json:"error,omitempty"`
}

type semanticInput struct {
	Facts              semanticFacts               `json:"facts"`
	Selections         []semanticSelection         `json:"selections"`
	QuestionSelections []semanticQuestionSelection `json:"question_selections"`
	PackRefs           []string                    `json:"pack_refs"`
}

type semanticFacts struct {
	Surface                string `json:"surface"`
	SharingIntent          string `json:"sharing_intent"`
	CorporateIdentity      string `json:"corporate_identity"`
	TargetUsers            string `json:"target_users"`
	IntegrationAuth        string `json:"integration_auth"`
	IntegrationCriticality string `json:"integration_criticality"`
}

type semanticSelection struct {
	Dimension string `json:"dimension"`
	Option    string `json:"option"`
	FreeText  string `json:"free_text"`
}

type semanticQuestionSelection struct {
	Question string `json:"question"`
	Option   string `json:"option"`
	FreeText string `json:"free_text"`
}

type semanticResult struct {
	SchemaVersion   string                    `json:"schema_version"`
	Issues          []semanticIssue           `json:"issues"`
	Questions       []semanticQuestion        `json:"questions"`
	Defaults        []semanticDefaultProposal `json:"defaults"`
	PackRefs        []string                  `json:"pack_refs"`
	IntakeIssues    []intake.Issue            `json:"intake_issues"`
	IntakeQuestions []intake.Question         `json:"intake_questions"`
}

type semanticIssue struct {
	Ref       string   `json:"ref"`
	Kind      string   `json:"kind"`
	RuleRef   string   `json:"rule_ref"`
	Dimension string   `json:"dimension"`
	Field     string   `json:"field"`
	DetailKey string   `json:"detail_key"`
	DependsOn []string `json:"depends_on"`
}

type semanticQuestion struct {
	Ref          string                 `json:"ref"`
	Dimension    string                 `json:"dimension"`
	PackRef      string                 `json:"pack_ref"`
	Slot         string                 `json:"slot"`
	DecisionKind string                 `json:"decision_kind"`
	DerivedFrom  []string               `json:"derived_from"`
	DependsOn    []string               `json:"depends_on"`
	PromptKey    string                 `json:"prompt_key"`
	WhyKey       string                 `json:"why_key"`
	HelpKey      string                 `json:"help_key"`
	ExampleKey   string                 `json:"example_key"`
	Options      []semanticResultOption `json:"options"`
}

type semanticResultOption struct {
	Ref          string `json:"ref"`
	Kind         string `json:"kind"`
	LabelKey     string `json:"label_key"`
	HelpKey      string `json:"help_key"`
	ExampleKey   string `json:"example_key"`
	RationaleKey string `json:"rationale_key"`
	Recommended  bool   `json:"recommended"`
}

type semanticDefaultProposal struct {
	Dimension    string `json:"dimension"`
	DecisionKind string `json:"decision_kind"`
	Option       string `json:"option"`
	LabelKey     string `json:"label_key"`
	HelpKey      string `json:"help_key"`
	ExampleKey   string `json:"example_key"`
	RationaleKey string `json:"rationale_key"`
	Applied      bool   `json:"implicitly_applied"`
}

type semanticError struct {
	Code  string `json:"code"`
	Field string `json:"field"`
}

type evaluatorCorpusSeed struct {
	ref   string
	input Input
}

func evaluatorV1BehaviorCorpus() []semanticCorpusCase {
	seeds := evaluatorV1SuccessfulCorpusSeeds()
	for _, pack := range evaluatorV1DomainPackRefs() {
		seeds = append(seeds, evaluatorCorpusSeed{
			ref:   "pack/" + strings.TrimPrefix(pack.String(), "pack:"),
			input: Input{PackRefs: []catalog.PackRef{pack}},
		})
	}
	result := make([]semanticCorpusCase, 0, len(seeds)+4)
	for _, seed := range seeds {
		evaluated, err := EvaluateV1(seed.input)
		if err != nil {
			panic(fmt.Sprintf("wizard gaps V1 corpus %s failed: %v", seed.ref, err))
		}
		projected := projectSemanticResult(evaluated)
		result = append(result, semanticCorpusCase{
			Ref: seed.ref, Input: projectSemanticInput(seed.input), Result: &projected,
		})
	}
	for _, seed := range evaluatorV1ErrorCorpusSeeds() {
		_, err := EvaluateV1(seed.input)
		domainErr, ok := err.(*DomainError)
		if !ok {
			panic(fmt.Sprintf("wizard gaps V1 error corpus %s returned %v", seed.ref, err))
		}
		result = append(result, semanticCorpusCase{
			Ref: seed.ref, Input: projectSemanticInput(seed.input),
			Error: &semanticError{Code: string(domainErr.Code), Field: domainErr.Field},
		})
	}
	validateEvaluatorV1CorpusCoverage(result)
	return result
}

func evaluatorV1SuccessfulCorpusSeeds() []evaluatorCorpusSeed {
	return []evaluatorCorpusSeed{
		{ref: "G00/empty", input: Input{}},
		{
			ref: "G01/all-technical-active",
			input: Input{
				Facts: Facts{
					CorporateIdentity:      DeclarationDeclared,
					TargetUsers:            DeclarationDeclared,
					IntegrationAuth:        DeclarationDeclared,
					IntegrationCriticality: DeclarationDeclared,
				},
				Selections: []Selection{
					corpusPreset(DimensionU1, "team"),
					corpusPreset(DimensionU4, "persisted_user_data"),
					corpusPreset(DimensionU5, "sensitive"),
					corpusPreset(DimensionU7, "public_api"),
					corpusPreset(DimensionU10, "cloud_container"),
					corpusPreset(DimensionU11, "realtime"),
				},
			},
		},
		{
			ref: "G02/r1-sharing-contradiction",
			input: Input{
				Facts:      Facts{SharingIntent: SharingShared},
				Selections: []Selection{corpusPreset(DimensionU1, "personal")},
			},
		},
		{
			ref: "G03/r1-personal-contradiction",
			input: Input{
				Facts:      Facts{SharingIntent: SharingPersonal},
				Selections: []Selection{corpusPreset(DimensionU1, "team")},
			},
		},
		{
			ref: "G04/ambiguous-human-surface",
			input: Input{Facts: Facts{
				Surface: SurfaceHumanUI, SharingIntent: SharingAmbiguous,
				CorporateIdentity: DeclarationMissing,
			}},
		},
		{
			ref: "G05/r2-server-surface",
			input: Input{
				Facts:      Facts{Surface: SurfaceServerService},
				Selections: []Selection{corpusPreset(DimensionU2, "responsive_web")},
			},
		},
		{
			ref: "G06/native-desktop-surface",
			input: Input{
				Facts:      Facts{Surface: SurfaceNativeDesktop},
				Selections: []Selection{corpusPreset(DimensionU2, "responsive_web")},
			},
		},
		{
			ref:   "G07/kernel-surface",
			input: Input{Facts: Facts{Surface: SurfaceKernelModule}},
		},
		{
			ref: "G08/r4-persisted-data",
			input: Input{
				Selections: []Selection{corpusPreset(DimensionU4, "persisted_user_data")},
			},
		},
		{
			ref: "G09/r5-integration-governance",
			input: Input{
				Facts: Facts{
					IntegrationAuth:        DeclarationMissing,
					IntegrationCriticality: DeclarationMissing,
				},
				Selections: []Selection{corpusPreset(DimensionU7, "external_services")},
			},
		},
		{
			ref: "G10/r6-mobile-platform",
			input: Input{
				Facts:      Facts{Surface: SurfaceNativeMobile},
				Selections: []Selection{corpusPreset(DimensionU2, "native_desktop")},
			},
		},
		{
			ref: "G11/r7-mobile-deployment",
			input: Input{
				Facts:      Facts{Surface: SurfaceNativeMobile},
				Selections: []Selection{corpusPreset(DimensionU10, "cloud_container")},
			},
		},
		{
			ref: "G12/r8-target-users",
			input: Input{
				Facts:      Facts{TargetUsers: DeclarationMissing},
				Selections: []Selection{corpusPreset(DimensionU1, "team")},
			},
		},
		{
			ref: "G13/pack-order-deduplication",
			input: Input{PackRefs: []catalog.PackRef{
				corpusPackRef("pack:commerce"),
				corpusPackRef("pack:calendar"),
				corpusPackRef("pack:commerce"),
			}},
		},
		{
			ref: "G14/supplemental-free-text",
			input: Input{
				Selections: []Selection{corpusPreset(DimensionU7, "external_services")},
				QuestionSelections: []QuestionSelection{{
					Question: "intake-question:wizard.r5",
					Option:   "intake-option:wizard.r5.custom",
					FreeText: "mTLS y credenciales rotatorias",
				}},
			},
		},
		{
			ref: "G15/selection-conflict",
			input: Input{Selections: []Selection{
				corpusPreset(DimensionU1, "personal"),
				corpusPreset(DimensionU1, "team"),
			}},
		},
		{
			ref: "G16/free-text",
			input: Input{Selections: []Selection{{
				Dimension: DimensionU1,
				Option:    optionRef(DimensionU1, "custom"),
				FreeText:  "equipo clínico",
			}}},
		},
		{
			ref: "G17/free-text-4096-runes",
			input: Input{Selections: []Selection{{
				Dimension: DimensionU1,
				Option:    optionRef(DimensionU1, "custom"),
				FreeText:  strings.Repeat("á", 4096),
			}}},
		},
		{
			ref:   "G18/r1-personal-gap",
			input: Input{Facts: Facts{SharingIntent: SharingPersonal}},
		},
		{
			ref:   "G19/r1-shared-gap",
			input: Input{Facts: Facts{SharingIntent: SharingShared}},
		},
		{
			ref:   "G20/r2-native-mobile-gap",
			input: Input{Facts: Facts{Surface: SurfaceNativeMobile}},
		},
		{
			ref: "G21/compatible-ambiguous-mobile",
			input: Input{
				Facts: Facts{
					Surface: SurfaceNativeMobile, SharingIntent: SharingAmbiguous,
					TargetUsers: DeclarationDeclared, IntegrationAuth: DeclarationDeclared,
					IntegrationCriticality: DeclarationDeclared,
				},
				Selections: []Selection{
					corpusPreset(DimensionU1, "personal"),
					corpusPreset(DimensionU2, "native_mobile"),
					corpusPreset(DimensionU4, "ephemeral"),
					corpusPreset(DimensionU7, "none"),
					corpusPreset(DimensionU10, "distribution_store"),
				},
			},
		},
		{
			ref: "G22/r1-personal-compatible",
			input: Input{
				Facts:      Facts{SharingIntent: SharingPersonal},
				Selections: []Selection{corpusPreset(DimensionU1, "personal")},
			},
		},
		{
			ref: "G23/r1-shared-compatible",
			input: Input{
				Facts:      Facts{SharingIntent: SharingShared},
				Selections: []Selection{corpusPreset(DimensionU1, "team")},
			},
		},
		{
			ref: "G24/mobile-custom-compatible",
			input: Input{
				Facts: Facts{Surface: SurfaceNativeMobile},
				Selections: []Selection{
					{
						Dimension: DimensionU2,
						Option:    optionRef(DimensionU2, "custom"),
						FreeText:  "shell nativa especializada",
					},
					{
						Dimension: DimensionU10,
						Option:    optionRef(DimensionU10, "custom"),
						FreeText:  "distribución administrada",
					},
				},
			},
		},
	}
}

func evaluatorV1ErrorCorpusSeeds() []evaluatorCorpusSeed {
	unknownPack, err := catalog.NewPackRef("pack:unknown")
	if err != nil {
		panic(err)
	}
	return []evaluatorCorpusSeed{
		{
			ref:   "E01/invalid-fact",
			input: Input{Facts: Facts{Surface: "keyword_web"}},
		},
		{
			ref: "E02/unknown-dimension",
			input: Input{Selections: []Selection{{
				Dimension: "U99", Option: "intake-option:wizard.u99.a",
			}}},
		},
		{
			ref: "E03/invalid-free-text",
			input: Input{Selections: []Selection{{
				Dimension: DimensionU1, Option: optionRef(DimensionU1, "custom"),
			}}},
		},
		{
			ref:   "E04/unknown-pack",
			input: Input{PackRefs: []catalog.PackRef{unknownPack}},
		},
		{
			ref: "E05/unknown-option",
			input: Input{Selections: []Selection{{
				Dimension: DimensionU1, Option: "intake-option:wizard.u1.unknown",
			}}},
		},
		{
			ref: "E06/unknown-question",
			input: Input{QuestionSelections: []QuestionSelection{{
				Question: "intake-question:wizard.unknown",
				Option:   "intake-option:wizard.unknown.a",
			}}},
		},
		{
			ref: "E07/preset-with-free-text",
			input: Input{Selections: []Selection{{
				Dimension: DimensionU1, Option: optionRef(DimensionU1, "personal"),
				FreeText: "texto no permitido",
			}}},
		},
		{
			ref: "E08/free-text-4097-runes",
			input: Input{Selections: []Selection{{
				Dimension: DimensionU1,
				Option:    optionRef(DimensionU1, "custom"),
				FreeText:  strings.Repeat("á", 4097),
			}}},
		},
	}
}

func evaluatorV1DomainPackRefs() []catalog.PackRef {
	builtIn := catalog.BuiltInV1()
	result := make([]catalog.PackRef, 0, len(builtIn.Packs())-1)
	for _, pack := range builtIn.Packs() {
		if pack.Ref() != builtIn.FoundationRef() {
			result = append(result, pack.Ref())
		}
	}
	return result
}

func corpusPreset(dimension DimensionRef, option string) Selection {
	return Selection{Dimension: dimension, Option: optionRef(dimension, option)}
}

func corpusPackRef(want string) catalog.PackRef {
	for _, ref := range evaluatorV1DomainPackRefs() {
		if ref.String() == want {
			return ref
		}
	}
	panic("wizard gaps V1 corpus pack missing " + want)
}

func projectSemanticInput(value Input) semanticInput {
	result := semanticInput{
		Facts: semanticFacts{
			Surface:                string(value.Facts.Surface),
			SharingIntent:          string(value.Facts.SharingIntent),
			CorporateIdentity:      string(value.Facts.CorporateIdentity),
			TargetUsers:            string(value.Facts.TargetUsers),
			IntegrationAuth:        string(value.Facts.IntegrationAuth),
			IntegrationCriticality: string(value.Facts.IntegrationCriticality),
		},
	}
	for _, selection := range value.Selections {
		result.Selections = append(result.Selections, semanticSelection{
			Dimension: string(selection.Dimension), Option: string(selection.Option),
			FreeText: selection.FreeText,
		})
	}
	for _, selection := range value.QuestionSelections {
		result.QuestionSelections = append(
			result.QuestionSelections,
			semanticQuestionSelection{
				Question: string(selection.Question), Option: string(selection.Option),
				FreeText: selection.FreeText,
			},
		)
	}
	for _, ref := range value.PackRefs {
		result.PackRefs = append(result.PackRefs, ref.String())
	}
	return result
}

func projectSemanticResult(value Result) semanticResult {
	result := semanticResult{SchemaVersion: value.SchemaVersion()}
	for _, issue := range value.Issues() {
		projected := semanticIssue{
			Ref: string(issue.Ref()), Kind: string(issue.Kind()),
			RuleRef: string(issue.RuleRef()), Dimension: string(issue.Dimension()),
			Field: string(issue.Field()), DetailKey: string(issue.DetailKey()),
		}
		for _, dependency := range issue.DependsOn() {
			projected.DependsOn = append(projected.DependsOn, string(dependency))
		}
		result.Issues = append(result.Issues, projected)
		result.IntakeIssues = append(result.IntakeIssues, issue.IntakeIssue())
	}
	for _, question := range value.Questions() {
		projected := semanticQuestion{
			Ref: string(question.Ref()), Dimension: string(question.Dimension()),
			PackRef: question.PackRef().String(), Slot: string(question.Slot()),
			DecisionKind: string(question.DecisionKind()),
			PromptKey:    string(question.PromptKey()), WhyKey: string(question.WhyKey()),
			HelpKey: string(question.HelpKey()), ExampleKey: string(question.ExampleKey()),
		}
		for _, ref := range question.DerivedFrom() {
			projected.DerivedFrom = append(projected.DerivedFrom, string(ref))
		}
		for _, ref := range question.DependsOn() {
			projected.DependsOn = append(projected.DependsOn, string(ref))
		}
		for _, option := range question.Options() {
			projected.Options = append(projected.Options, semanticResultOption{
				Ref: string(option.Ref()), Kind: string(option.Kind()),
				LabelKey: string(option.LabelKey()), HelpKey: string(option.HelpKey()),
				ExampleKey:   string(option.ExampleKey()),
				RationaleKey: string(option.RationaleKey()),
				Recommended:  option.Recommended(),
			})
		}
		result.Questions = append(result.Questions, projected)
		result.IntakeQuestions = append(
			result.IntakeQuestions,
			question.IntakeQuestion(),
		)
	}
	for _, value := range value.DefaultProposals() {
		result.Defaults = append(result.Defaults, semanticDefaultProposal{
			Dimension: string(value.Dimension()), DecisionKind: string(value.DecisionKind()),
			Option: string(value.Option()), LabelKey: string(value.LabelKey()),
			HelpKey: string(value.HelpKey()), ExampleKey: string(value.ExampleKey()),
			RationaleKey: string(value.RationaleKey()),
			Applied:      value.ImplicitlyApplied(),
		})
	}
	for _, ref := range value.PackRefs() {
		result.PackRefs = append(result.PackRefs, ref.String())
	}
	return result
}

func validateEvaluatorV1CorpusCoverage(values []semanticCorpusCase) {
	dimensions := make(map[string]struct{})
	rules := make(map[string]struct{})
	packs := make(map[string]struct{})
	cases := make(map[string]struct{}, len(values))
	surfaces := make(map[string]struct{})
	sharing := make(map[string]struct{})
	declarations := make(map[string]struct{})
	errorCodes := make(map[string]struct{})
	for _, value := range values {
		cases[value.Ref] = struct{}{}
		surfaces[value.Input.Facts.Surface] = struct{}{}
		sharing[value.Input.Facts.SharingIntent] = struct{}{}
		for _, declaration := range []string{
			value.Input.Facts.CorporateIdentity,
			value.Input.Facts.TargetUsers,
			value.Input.Facts.IntegrationAuth,
			value.Input.Facts.IntegrationCriticality,
		} {
			declarations[declaration] = struct{}{}
		}
		if value.Error != nil {
			errorCodes[value.Error.Code] = struct{}{}
		}
		if value.Result == nil {
			continue
		}
		for _, issue := range value.Result.Issues {
			if issue.Dimension != "" {
				dimensions[issue.Dimension] = struct{}{}
			}
			if issue.RuleRef != "" {
				rules[issue.RuleRef] = struct{}{}
			}
		}
		for _, question := range value.Result.Questions {
			if question.Dimension != "" {
				dimensions[question.Dimension] = struct{}{}
			}
			if question.PackRef != "" {
				packs[question.PackRef] = struct{}{}
			}
		}
		for _, proposal := range value.Result.Defaults {
			dimensions[proposal.Dimension] = struct{}{}
		}
	}
	for _, dimension := range BuiltInV1().Dimensions() {
		if _, found := dimensions[string(dimension.Ref())]; !found {
			panic("wizard gaps V1 corpus misses dimension " + string(dimension.Ref()))
		}
	}
	for _, rule := range BuiltInV1().Rules() {
		if _, found := rules[string(rule.Ref())]; !found {
			panic("wizard gaps V1 corpus misses rule " + string(rule.Ref()))
		}
	}
	for _, pack := range evaluatorV1DomainPackRefs() {
		if _, found := packs[pack.String()]; !found {
			panic("wizard gaps V1 corpus misses pack " + pack.String())
		}
	}
	for _, ref := range []string{
		"G00/empty", "G01/all-technical-active",
		"G02/r1-sharing-contradiction", "G03/r1-personal-contradiction",
		"G04/ambiguous-human-surface", "G05/r2-server-surface",
		"G06/native-desktop-surface", "G07/kernel-surface",
		"G08/r4-persisted-data", "G09/r5-integration-governance",
		"G10/r6-mobile-platform", "G11/r7-mobile-deployment",
		"G12/r8-target-users", "G13/pack-order-deduplication",
		"G14/supplemental-free-text", "G15/selection-conflict",
		"G16/free-text", "G17/free-text-4096-runes",
		"G18/r1-personal-gap", "G19/r1-shared-gap",
		"G20/r2-native-mobile-gap",
		"G21/compatible-ambiguous-mobile",
		"G22/r1-personal-compatible", "G23/r1-shared-compatible",
		"G24/mobile-custom-compatible",
		"E01/invalid-fact", "E02/unknown-dimension", "E03/invalid-free-text",
		"E04/unknown-pack", "E05/unknown-option", "E06/unknown-question",
		"E07/preset-with-free-text", "E08/free-text-4097-runes",
	} {
		if _, found := cases[ref]; !found {
			panic("wizard gaps V1 corpus misses case " + ref)
		}
	}
	for _, value := range []Surface{
		SurfaceUnspecified, SurfaceHumanUI, SurfaceNativeMobile,
		SurfaceNativeDesktop, SurfaceServerService, SurfaceKernelModule,
	} {
		if _, found := surfaces[string(value)]; !found {
			panic("wizard gaps V1 corpus misses surface " + string(value))
		}
	}
	for _, value := range []SharingIntent{
		SharingUnspecified, SharingAmbiguous, SharingPersonal, SharingShared,
	} {
		if _, found := sharing[string(value)]; !found {
			panic("wizard gaps V1 corpus misses sharing " + string(value))
		}
	}
	for _, value := range []Declaration{
		DeclarationUnspecified, DeclarationMissing, DeclarationDeclared,
	} {
		if _, found := declarations[string(value)]; !found {
			panic("wizard gaps V1 corpus misses declaration " + string(value))
		}
	}
	for _, code := range []ErrorCode{
		ErrorInvalidArgument, ErrorUnknownDimension, ErrorUnknownQuestion,
		ErrorUnknownOption, ErrorInvalidFreeText, ErrorUnknownPack,
	} {
		if _, found := errorCodes[string(code)]; !found {
			panic("wizard gaps V1 corpus misses error " + string(code))
		}
	}
}
