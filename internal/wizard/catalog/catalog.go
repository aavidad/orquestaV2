package catalog

type CatalogInput struct {
	Version       CatalogVersion
	FoundationRef PackRef
	Packs         []Pack
}

// Catalog is an immutable registry. It validates global reference consistency
// once, before any pack selection can be composed.
type Catalog struct {
	version       CatalogVersion
	foundationRef PackRef
	packs         []Pack
}

func NewCatalog(input CatalogInput) (Catalog, error) {
	if input.Version.value == "" {
		return Catalog{}, domainError(ErrorInvalidArgument, "catalog.version")
	}
	if input.FoundationRef.value == "" {
		return Catalog{}, domainError(ErrorInvalidRef, "catalog.foundation_ref")
	}
	packs := clonePacks(input.Packs)
	if len(packs) == 0 {
		return Catalog{}, domainError(ErrorInvalidArgument, "catalog.packs")
	}
	sortPacks(packs)
	seenPacks := make(map[string]struct{}, len(packs))
	foundFoundation := false
	taxonomies := make(map[string]Taxonomy)
	questions := make(map[string]Question)
	defaults := make(map[string]TechnicalDefault)
	for _, pack := range packs {
		if _, ok := seenPacks[pack.ref.value]; ok {
			return Catalog{}, domainError(ErrorDuplicateRef, "catalog.packs")
		}
		seenPacks[pack.ref.value] = struct{}{}
		if pack.ref == input.FoundationRef {
			foundFoundation = true
		}
		if err := mergeDefinitions(taxonomies, pack.taxonomies, func(value Taxonomy) string {
			return value.ref.value
		}); err != nil {
			return Catalog{}, err
		}
		if err := mergeDefinitions(questions, pack.questions, func(value Question) string {
			return value.ref.value
		}); err != nil {
			return Catalog{}, err
		}
		if err := mergeDefinitions(defaults, pack.defaults, func(value TechnicalDefault) string {
			return value.ref.value
		}); err != nil {
			return Catalog{}, err
		}
	}
	if !foundFoundation {
		return Catalog{}, domainError(ErrorPackNotFound, "catalog.foundation_ref")
	}
	if err := validateQuestionLinks(questions, taxonomies); err != nil {
		return Catalog{}, err
	}
	return Catalog{
		version: input.Version, foundationRef: input.FoundationRef, packs: packs,
	}, nil
}

func (value Catalog) Version() CatalogVersion { return value.version }
func (value Catalog) FoundationRef() PackRef  { return value.foundationRef }
func (value Catalog) Packs() []Pack           { return clonePacks(value.packs) }

// Compose selects packs only by explicit opaque refs. Input order and repeated
// refs do not affect output. Foundation is always included exactly once.
func (value Catalog) Compose(refs ...PackRef) (Selection, error) {
	byRef := make(map[string]Pack, len(value.packs))
	for _, pack := range value.packs {
		byRef[pack.ref.value] = pack
	}
	selectedRefs := map[string]struct{}{value.foundationRef.value: {}}
	for _, ref := range refs {
		if _, ok := byRef[ref.value]; !ok {
			return Selection{}, domainError(ErrorPackNotFound, "compose.pack_ref")
		}
		selectedRefs[ref.value] = struct{}{}
	}
	selectedPacks := make([]Pack, 0, len(selectedRefs))
	for ref := range selectedRefs {
		selectedPacks = append(selectedPacks, byRef[ref])
	}
	sortPacks(selectedPacks)

	taxonomies := make(map[string]Taxonomy)
	questions := make(map[string]Question)
	defaults := make(map[string]TechnicalDefault)
	roadmapCapabilityRefs := make(map[RoadmapCapabilityRef]struct{})
	for _, pack := range selectedPacks {
		for _, ref := range pack.roadmapCapabilityRefs {
			roadmapCapabilityRefs[ref] = struct{}{}
		}
		mergeKnownDefinitions(taxonomies, pack.taxonomies, func(value Taxonomy) string {
			return value.ref.value
		})
		mergeKnownDefinitions(questions, pack.questions, func(value Question) string {
			return value.ref.value
		})
		mergeKnownDefinitions(defaults, pack.defaults, func(value TechnicalDefault) string {
			return value.ref.value
		})
	}
	return newSelection(
		value.version, selectedPacks, mapValues(taxonomies),
		mapValues(questions), mapValues(defaults),
		mapRoadmapCapabilityRefs(roadmapCapabilityRefs),
	), nil
}

// ComposeAll returns one deterministic view of foundation plus every pack.
func (value Catalog) ComposeAll() Selection {
	refs := make([]PackRef, 0, len(value.packs))
	for _, pack := range value.packs {
		refs = append(refs, pack.ref)
	}
	selection, err := value.Compose(refs...)
	if err != nil {
		panic(err)
	}
	return selection
}

type Selection struct {
	version               CatalogVersion
	packRefs              []PackRef
	roadmapCapabilityRefs []RoadmapCapabilityRef
	taxonomies            []Taxonomy
	questions             []Question
	defaults              []TechnicalDefault
}

func newSelection(
	version CatalogVersion,
	packs []Pack,
	taxonomies []Taxonomy,
	questions []Question,
	defaults []TechnicalDefault,
	roadmapCapabilityRefs []RoadmapCapabilityRef,
) Selection {
	packRefs := make([]PackRef, len(packs))
	for index, pack := range packs {
		packRefs[index] = pack.ref
	}
	sortTaxonomies(taxonomies)
	sortQuestions(questions)
	sortDefaults(defaults)
	sortRoadmapCapabilityRefs(roadmapCapabilityRefs)
	return Selection{
		version: version, packRefs: packRefs,
		roadmapCapabilityRefs: roadmapCapabilityRefs,
		taxonomies:            taxonomies, questions: questions, defaults: defaults,
	}
}

func (value Selection) Version() CatalogVersion { return value.version }
func (value Selection) PackRefs() []PackRef {
	return append([]PackRef(nil), value.packRefs...)
}

// RoadmapCapabilityRefs aggregates audit/navigation links from selected
// packs. Presence does not assert implementation, coverage or accreditation.
func (value Selection) RoadmapCapabilityRefs() []RoadmapCapabilityRef {
	return cloneRoadmapCapabilityRefs(value.roadmapCapabilityRefs)
}

func (value Selection) Taxonomies() []Taxonomy       { return cloneTaxonomies(value.taxonomies) }
func (value Selection) Questions() []Question        { return cloneQuestions(value.questions) }
func (value Selection) Defaults() []TechnicalDefault { return cloneDefaults(value.defaults) }

func mergeDefinitions[T any](
	destination map[string]T,
	values []T,
	ref func(T) string,
) error {
	for _, value := range values {
		key := ref(value)
		if existing, ok := destination[key]; ok {
			if !sameDefinition(existing, value) {
				return domainError(ErrorConflictingDefinition, key)
			}
			continue
		}
		destination[key] = value
	}
	return nil
}

func mergeKnownDefinitions[T any](
	destination map[string]T,
	values []T,
	ref func(T) string,
) {
	for _, value := range values {
		if _, ok := destination[ref(value)]; !ok {
			destination[ref(value)] = value
		}
	}
}

func validateQuestionLinks(
	questions map[string]Question,
	taxonomies map[string]Taxonomy,
) error {
	for _, question := range questions {
		if question.answerKind != AnswerChoice {
			continue
		}
		taxonomy, ok := taxonomies[question.taxonomyRef.value]
		if !ok {
			return domainError(ErrorTaxonomyNotFound, question.ref.value)
		}
		terms := make(map[string]struct{}, len(taxonomy.terms))
		for _, term := range taxonomy.terms {
			terms[term.ref.value] = struct{}{}
		}
		for _, option := range question.options {
			if _, ok := terms[option.termRef.value]; !ok {
				return domainError(ErrorTermNotFound, option.ref.value)
			}
		}
	}
	return nil
}

func mapValues[T any](values map[string]T) []T {
	out := make([]T, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	return out
}

func mapRoadmapCapabilityRefs(
	values map[RoadmapCapabilityRef]struct{},
) []RoadmapCapabilityRef {
	out := make([]RoadmapCapabilityRef, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	return out
}

func sortRoadmapCapabilityRefs(values []RoadmapCapabilityRef) {
	for current := 1; current < len(values); current++ {
		for previous := current; previous > 0 &&
			values[previous].value < values[previous-1].value; previous-- {
			values[previous], values[previous-1] = values[previous-1], values[previous]
		}
	}
}
