package catalog

import (
	"reflect"
	"sort"
	"strings"
)

func validRef(prefix, value string) (string, error) {
	if !strings.HasPrefix(value, prefix+":") || len(value) == len(prefix)+1 ||
		!validMachineKey(strings.TrimPrefix(value, prefix+":")) {
		return "", domainError(ErrorInvalidRef, prefix+"_ref")
	}
	return value, nil
}

func validMachineKey(value string) bool {
	if value == "" || len(value) > 256 || strings.TrimSpace(value) != value ||
		strings.HasPrefix(value, ".") || strings.HasSuffix(value, ".") ||
		strings.Contains(value, "..") {
		return false
	}
	for _, char := range value {
		if (char < 'a' || char > 'z') && (char < '0' || char > '9') &&
			char != '.' && char != '_' && char != '-' {
			return false
		}
	}
	return true
}

func containsDot(value string) bool {
	return strings.Contains(value, ".")
}

func validatePresentationKeys(field string, keys ...MessageKey) error {
	for index, key := range keys {
		if key.value == "" {
			return domainError(
				ErrorInvalidMessageKey,
				field+".presentation_key["+indexString(index)+"]",
			)
		}
	}
	return nil
}

func normalizeRoadmapCapabilityRefs(
	values []RoadmapCapabilityRef,
) ([]RoadmapCapabilityRef, error) {
	seen := make(map[RoadmapCapabilityRef]struct{}, len(values))
	out := make([]RoadmapCapabilityRef, 0, len(values))
	for _, value := range values {
		if !validRoadmapCapabilityRef(value.value) {
			return nil, domainError(ErrorInvalidRef, "pack.roadmap_capability_refs")
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	if len(out) == 0 {
		return nil, domainError(ErrorInvalidArgument, "pack.roadmap_capability_refs")
	}
	sort.Slice(out, func(i, j int) bool { return out[i].value < out[j].value })
	return out, nil
}

func validRoadmapCapabilityRef(value string) bool {
	switch value {
	case "WIZ-01", "WIZ-06", "WIZ-11", "WIZ-16", "WIZ-17", "WIZ-19", "WIZ-25":
		return true
	default:
		return false
	}
}

func uniqueTerms(values []Term, field string) error {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value.ref.value == "" {
			return domainError(ErrorInvalidRef, field)
		}
		if _, ok := seen[value.ref.value]; ok {
			return domainError(ErrorDuplicateRef, field)
		}
		seen[value.ref.value] = struct{}{}
	}
	return nil
}

func uniqueOptions(values []Option, field string) error {
	seenRefs := make(map[string]struct{}, len(values))
	seenTerms := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value.ref.value == "" || value.termRef.value == "" {
			return domainError(ErrorInvalidRef, field)
		}
		if _, ok := seenRefs[value.ref.value]; ok {
			return domainError(ErrorDuplicateRef, field)
		}
		if _, ok := seenTerms[value.termRef.value]; ok {
			return domainError(ErrorDuplicateRef, field)
		}
		seenRefs[value.ref.value] = struct{}{}
		seenTerms[value.termRef.value] = struct{}{}
	}
	return nil
}

func uniqueTaxonomies(values []Taxonomy, field string) error {
	return uniqueByRef(values, field, func(value Taxonomy) string { return value.ref.value })
}

func uniqueQuestions(values []Question, field string) error {
	return uniqueByRef(values, field, func(value Question) string { return value.ref.value })
}

func uniqueDefaults(values []TechnicalDefault, field string) error {
	return uniqueByRef(values, field, func(value TechnicalDefault) string { return value.ref.value })
}

func uniqueByRef[T any](values []T, field string, ref func(T) string) error {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		key := ref(value)
		if key == "" {
			return domainError(ErrorInvalidRef, field)
		}
		if _, ok := seen[key]; ok {
			return domainError(ErrorDuplicateRef, field)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func sortTerms(values []Term) {
	sort.Slice(values, func(i, j int) bool { return values[i].ref.value < values[j].ref.value })
}

func sortOptions(values []Option) {
	sort.Slice(values, func(i, j int) bool { return values[i].ref.value < values[j].ref.value })
}

func sortTaxonomies(values []Taxonomy) {
	sort.Slice(values, func(i, j int) bool { return values[i].ref.value < values[j].ref.value })
}

func sortQuestions(values []Question) {
	sort.Slice(values, func(i, j int) bool { return values[i].ref.value < values[j].ref.value })
}

func sortDefaults(values []TechnicalDefault) {
	sort.Slice(values, func(i, j int) bool { return values[i].ref.value < values[j].ref.value })
}

func sortPacks(values []Pack) {
	sort.Slice(values, func(i, j int) bool { return values[i].ref.value < values[j].ref.value })
}

func cloneTerms(values []Term) []Term {
	return append([]Term(nil), values...)
}

func cloneOptions(values []Option) []Option {
	return append([]Option(nil), values...)
}

func cloneTaxonomies(values []Taxonomy) []Taxonomy {
	out := append([]Taxonomy(nil), values...)
	for index := range out {
		out[index].terms = cloneTerms(out[index].terms)
	}
	return out
}

func cloneQuestions(values []Question) []Question {
	out := append([]Question(nil), values...)
	for index := range out {
		out[index].options = cloneOptions(out[index].options)
	}
	return out
}

func cloneDefaults(values []TechnicalDefault) []TechnicalDefault {
	return append([]TechnicalDefault(nil), values...)
}

func cloneRoadmapCapabilityRefs(
	values []RoadmapCapabilityRef,
) []RoadmapCapabilityRef {
	return append([]RoadmapCapabilityRef(nil), values...)
}

func clonePacks(values []Pack) []Pack {
	out := append([]Pack(nil), values...)
	for index := range out {
		out[index].roadmapCapabilityRefs = cloneRoadmapCapabilityRefs(
			out[index].roadmapCapabilityRefs,
		)
		out[index].taxonomies = cloneTaxonomies(out[index].taxonomies)
		out[index].questions = cloneQuestions(out[index].questions)
		out[index].defaults = cloneDefaults(out[index].defaults)
	}
	return out
}

func sameDefinition(left, right any) bool {
	return reflect.DeepEqual(left, right)
}

func indexString(value int) string {
	const digits = "0123456789"
	if value < 10 {
		return string(digits[value])
	}
	return "n"
}
