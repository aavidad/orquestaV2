package gaps

import (
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"orquesta/internal/wizard/catalog"
)

func validateFacts(facts Facts) error {
	if !oneOfSurface(facts.Surface) {
		return domainError(ErrorInvalidArgument, "facts.surface")
	}
	if !oneOfSharing(facts.SharingIntent) {
		return domainError(ErrorInvalidArgument, "facts.sharing_intent")
	}
	declarations := []struct {
		field string
		value Declaration
	}{
		{"facts.corporate_identity", facts.CorporateIdentity},
		{"facts.target_users", facts.TargetUsers},
		{"facts.integration_auth", facts.IntegrationAuth},
		{"facts.integration_criticality", facts.IntegrationCriticality},
	}
	for _, item := range declarations {
		if item.value != DeclarationUnspecified &&
			item.value != DeclarationMissing &&
			item.value != DeclarationDeclared {
			return domainError(ErrorInvalidArgument, item.field)
		}
	}
	return nil
}

func oneOfSurface(value Surface) bool {
	switch value {
	case SurfaceUnspecified, SurfaceHumanUI, SurfaceNativeMobile,
		SurfaceNativeDesktop, SurfaceServerService, SurfaceKernelModule:
		return true
	default:
		return false
	}
}

func oneOfSharing(value SharingIntent) bool {
	switch value {
	case SharingUnspecified, SharingAmbiguous, SharingPersonal, SharingShared:
		return true
	default:
		return false
	}
}

func indexSelections(
	values []Selection,
	dimensions map[DimensionRef]DimensionDescriptor,
) (selectionIndex, error) {
	unique := make(map[DimensionRef]map[string]Selection)
	for index, value := range values {
		dimension, found := dimensions[value.Dimension]
		if !found {
			return selectionIndex{}, domainError(
				ErrorUnknownDimension,
				"selections["+strconv.Itoa(index)+"].dimension",
			)
		}
		option, found := findOption(dimension.options, value.Option)
		if !found {
			return selectionIndex{}, domainError(
				ErrorUnknownOption,
				"selections["+strconv.Itoa(index)+"].option",
			)
		}
		if err := validateSelectedOption(
			option,
			value.FreeText,
			"selections["+strconv.Itoa(index)+"].free_text",
		); err != nil {
			return selectionIndex{}, err
		}
		if unique[value.Dimension] == nil {
			unique[value.Dimension] = make(map[string]Selection)
		}
		key := string(value.Option) + "\x00" + value.FreeText
		unique[value.Dimension][key] = value
	}
	result := selectionIndex{
		resolved:  make(map[DimensionRef]Selection),
		conflicts: make(map[DimensionRef]bool),
	}
	for ref, choices := range unique {
		if len(choices) != 1 {
			result.conflicts[ref] = true
			continue
		}
		for _, choice := range choices {
			result.resolved[ref] = choice
		}
	}
	return result, nil
}

func indexQuestionSelections(
	values []QuestionSelection,
	questions map[QuestionRef]Question,
) (questionSelectionIndex, error) {
	unique := make(map[QuestionRef]map[string]QuestionSelection)
	for index, value := range values {
		question, found := questions[value.Question]
		if !found {
			return questionSelectionIndex{}, domainError(
				ErrorUnknownQuestion,
				"question_selections["+strconv.Itoa(index)+"].question",
			)
		}
		option, found := findOption(question.options, value.Option)
		if !found {
			return questionSelectionIndex{}, domainError(
				ErrorUnknownOption,
				"question_selections["+strconv.Itoa(index)+"].option",
			)
		}
		if err := validateSelectedOption(
			option,
			value.FreeText,
			"question_selections["+strconv.Itoa(index)+"].free_text",
		); err != nil {
			return questionSelectionIndex{}, err
		}
		if unique[value.Question] == nil {
			unique[value.Question] = make(map[string]QuestionSelection)
		}
		key := string(value.Option) + "\x00" + value.FreeText
		unique[value.Question][key] = value
	}
	result := questionSelectionIndex{
		resolved:  make(map[QuestionRef]QuestionSelection),
		conflicts: make(map[QuestionRef]bool),
	}
	for ref, choices := range unique {
		if len(choices) != 1 {
			result.conflicts[ref] = true
			continue
		}
		for _, choice := range choices {
			result.resolved[ref] = choice
		}
	}
	return result, nil
}

func validateSelectedOption(option Option, freeText, field string) error {
	switch option.kind {
	case OptionFreeText:
		if strings.TrimSpace(freeText) == "" ||
			utf8.RuneCountInString(freeText) > 4096 {
			return domainError(ErrorInvalidFreeText, field)
		}
	case OptionPreset:
		if freeText != "" {
			return domainError(ErrorInvalidFreeText, field)
		}
	default:
		return domainError(ErrorInvalidArgument, field)
	}
	return nil
}

func supplementalQuestionCatalog(
	packRefs []catalog.PackRef,
) map[QuestionRef]Question {
	result := map[QuestionRef]Question{
		QuestionRef("intake-question:wizard.r5"): ruleQuestion(
			RuleR5,
			"product.integration_governance",
			DecisionProduct,
			"service_auth_medium",
			"public_low", "service_auth_medium", "oauth_high",
		),
		QuestionRef("intake-question:wizard.r8"): ruleQuestion(
			RuleR8,
			"product.target_users",
			DecisionProduct,
			"internal_team",
			"internal_team", "invited_customers", "public_users",
		),
	}
	selectedRefs := make(map[string]struct{}, len(packRefs))
	for _, ref := range packRefs {
		selectedRefs[ref.String()] = struct{}{}
	}
	for _, pack := range catalog.BuiltIn().Packs() {
		if _, selected := selectedRefs[pack.Ref().String()]; !selected {
			continue
		}
		for _, source := range pack.Questions() {
			if !strings.HasPrefix(source.Slot().String(), "domains.") {
				continue
			}
			question := packQuestion(pack.Ref(), source)
			result[question.ref] = question
		}
	}
	return result
}

func validatePackRefs(values []catalog.PackRef) ([]catalog.PackRef, error) {
	builtIn := catalog.BuiltIn()
	if _, err := builtIn.Compose(values...); err != nil {
		return nil, domainError(ErrorUnknownPack, "pack_refs")
	}
	byRef := make(map[string]catalog.PackRef, len(values))
	for _, ref := range values {
		if ref == builtIn.FoundationRef() {
			continue
		}
		byRef[ref.String()] = ref
	}
	keys := make([]string, 0, len(byRef))
	for key := range byRef {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]catalog.PackRef, len(keys))
	for index, key := range keys {
		result[index] = byRef[key]
	}
	return result, nil
}
