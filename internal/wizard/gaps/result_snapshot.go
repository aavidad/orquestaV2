package gaps

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"io"
	"strconv"
	"strings"

	"orquesta/internal/wizard/catalog"
)

const (
	// ResultSnapshotSchema identifies the complete canonical representation of
	// Result. Any wire change requires a new schema instead of a permissive
	// decoder fallback.
	ResultSnapshotSchema = "orquesta.wizard.gaps.result-snapshot.v1"

	maxResultSnapshotBytes        = 1 << 20
	maxResultSnapshotIssues       = 64
	maxResultSnapshotQuestions    = 64
	maxResultSnapshotDefaults     = 8
	maxResultSnapshotPackRefs     = 32
	maxResultSnapshotOptions      = 16
	maxResultSnapshotCausalRefs   = 64
	maxResultSnapshotDependencies = 32
)

const ErrorInvalidResultSnapshot ErrorCode = "wizard_gaps.invalid_result_snapshot"

type resultSnapshotDocument struct {
	Schema           string                          `json:"schema"`
	ResultSchema     string                          `json:"result_schema"`
	Issues           []resultSnapshotIssue           `json:"issues"`
	Questions        []resultSnapshotQuestion        `json:"questions"`
	DefaultProposals []resultSnapshotDefaultProposal `json:"default_proposals"`
	PackRefs         []string                        `json:"pack_refs"`
	Digest           string                          `json:"digest,omitempty"`
}

type resultSnapshotIssue struct {
	Ref       IssueRef       `json:"ref"`
	Kind      IssueKind      `json:"kind"`
	RuleRef   RuleRef        `json:"rule_ref"`
	Dimension DimensionRef   `json:"dimension"`
	Field     SlotKey        `json:"field"`
	DetailKey MessageKey     `json:"detail_key"`
	DependsOn []DimensionRef `json:"depends_on"`
}

type resultSnapshotQuestion struct {
	Ref          QuestionRef            `json:"ref"`
	Dimension    DimensionRef           `json:"dimension"`
	PackRef      string                 `json:"pack_ref"`
	Slot         SlotKey                `json:"slot"`
	DecisionKind DecisionKind           `json:"decision_kind"`
	DerivedFrom  []IssueRef             `json:"derived_from"`
	DependsOn    []QuestionRef          `json:"depends_on"`
	PromptKey    MessageKey             `json:"prompt_key"`
	WhyKey       MessageKey             `json:"why_key"`
	HelpKey      MessageKey             `json:"help_key"`
	ExampleKey   MessageKey             `json:"example_key"`
	Options      []resultSnapshotOption `json:"options"`
}

type resultSnapshotOption struct {
	Ref          OptionRef  `json:"ref"`
	Kind         OptionKind `json:"kind"`
	LabelKey     MessageKey `json:"label_key"`
	HelpKey      MessageKey `json:"help_key"`
	ExampleKey   MessageKey `json:"example_key"`
	RationaleKey MessageKey `json:"rationale_key"`
	Recommended  bool       `json:"recommended"`
}

type resultSnapshotDefaultProposal struct {
	Dimension    DimensionRef `json:"dimension"`
	DecisionKind DecisionKind `json:"decision_kind"`
	Option       OptionRef    `json:"option"`
	LabelKey     MessageKey   `json:"label_key"`
	HelpKey      MessageKey   `json:"help_key"`
	ExampleKey   MessageKey   `json:"example_key"`
	RationaleKey MessageKey   `json:"rationale_key"`
}

// MarshalResultSnapshot serializes every semantic Result field and returns the
// lower-case SHA-256 of the complete canonical document out of band. The
// document's v1 digest member remains only a wire self-check.
func MarshalResultSnapshot(value Result) ([]byte, string, error) {
	document := snapshotDocumentFromResult(value)
	if err := validateResultSnapshotDocument(document); err != nil {
		return nil, "", err
	}
	encoded, err := marshalResultSnapshotDocument(document)
	if err != nil {
		return nil, "", err
	}
	return encoded, resultSnapshotExternalDigest(encoded), nil
}

// RestoreResultSnapshot accepts only the complete canonical bytes bound to the
// lower-case expectedDigest supplied by the caller from an external durable
// record. Unknown fields, alternate encodings and rehashed wire changes fail
// closed.
func RestoreResultSnapshot(encoded []byte, expectedDigest string) (Result, error) {
	if len(encoded) == 0 || len(encoded) > maxResultSnapshotBytes {
		return Result{}, invalidResultSnapshot("snapshot")
	}
	if !validResultSnapshotDigest(expectedDigest) {
		return Result{}, invalidResultSnapshot("expected_digest")
	}
	actualDigest := resultSnapshotExternalDigest(encoded)
	if subtle.ConstantTimeCompare(
		[]byte(actualDigest),
		[]byte(expectedDigest),
	) != 1 {
		return Result{}, invalidResultSnapshot("expected_digest")
	}

	var document resultSnapshotDocument
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil {
		return Result{}, invalidResultSnapshot("snapshot_json")
	}
	if err := requireResultSnapshotEOF(decoder); err != nil {
		return Result{}, err
	}

	suppliedWireChecksum := document.Digest
	document.Digest = ""
	payload, err := json.Marshal(document)
	if err != nil {
		return Result{}, invalidResultSnapshot("snapshot_json")
	}
	if !validResultSnapshotDigest(suppliedWireChecksum) ||
		resultSnapshotWireChecksum(payload) != suppliedWireChecksum {
		return Result{}, invalidResultSnapshot("digest")
	}
	if err := validateResultSnapshotDocument(document); err != nil {
		return Result{}, err
	}

	canonical, err := marshalResultSnapshotDocument(document)
	if err != nil || !bytes.Equal(encoded, canonical) {
		return Result{}, invalidResultSnapshot("snapshot_json")
	}
	return resultFromSnapshotDocument(document), nil
}

func snapshotDocumentFromResult(value Result) resultSnapshotDocument {
	document := resultSnapshotDocument{
		Schema:           ResultSnapshotSchema,
		ResultSchema:     value.schemaVersion,
		Issues:           make([]resultSnapshotIssue, len(value.issues)),
		Questions:        make([]resultSnapshotQuestion, len(value.questions)),
		DefaultProposals: make([]resultSnapshotDefaultProposal, len(value.defaults)),
		PackRefs:         make([]string, len(value.packRefs)),
	}
	for index, issue := range value.issues {
		dependsOn := append([]DimensionRef(nil), issue.dependsOn...)
		if dependsOn == nil {
			dependsOn = []DimensionRef{}
		}
		document.Issues[index] = resultSnapshotIssue{
			Ref: issue.ref, Kind: issue.kind, RuleRef: issue.ruleRef,
			Dimension: issue.dimension, Field: issue.field,
			DetailKey: issue.detailKey, DependsOn: dependsOn,
		}
	}
	for index, question := range value.questions {
		derivedFrom := append([]IssueRef(nil), question.derivedFrom...)
		if derivedFrom == nil {
			derivedFrom = []IssueRef{}
		}
		dependsOn := append([]QuestionRef(nil), question.dependsOn...)
		if dependsOn == nil {
			dependsOn = []QuestionRef{}
		}
		projected := resultSnapshotQuestion{
			Ref: question.ref, Dimension: question.dimension,
			PackRef: question.packRef.String(), Slot: question.slot,
			DecisionKind: question.decisionKind,
			DerivedFrom:  derivedFrom, DependsOn: dependsOn,
			PromptKey: question.promptKey, WhyKey: question.whyKey,
			HelpKey: question.helpKey, ExampleKey: question.exampleKey,
			Options: make([]resultSnapshotOption, len(question.options)),
		}
		for optionIndex, option := range question.options {
			projected.Options[optionIndex] = resultSnapshotOption{
				Ref: option.ref, Kind: option.kind,
				LabelKey: option.labelKey, HelpKey: option.helpKey,
				ExampleKey: option.exampleKey, RationaleKey: option.rationaleKey,
				Recommended: option.recommended,
			}
		}
		document.Questions[index] = projected
	}
	for index, proposal := range value.defaults {
		document.DefaultProposals[index] = resultSnapshotDefaultProposal{
			Dimension: proposal.dimension, DecisionKind: proposal.decisionKind,
			Option: proposal.option, LabelKey: proposal.labelKey,
			HelpKey: proposal.helpKey, ExampleKey: proposal.exampleKey,
			RationaleKey: proposal.rationaleKey,
		}
	}
	for index, ref := range value.packRefs {
		document.PackRefs[index] = ref.String()
	}
	return document
}

func resultFromSnapshotDocument(document resultSnapshotDocument) Result {
	result := Result{
		schemaVersion: document.ResultSchema,
		issues:        make([]Issue, len(document.Issues)),
		questions:     make([]Question, len(document.Questions)),
		defaults:      make([]DefaultProposal, len(document.DefaultProposals)),
		packRefs:      make([]catalog.PackRef, len(document.PackRefs)),
	}
	for index, issue := range document.Issues {
		result.issues[index] = Issue{
			ref: issue.Ref, kind: issue.Kind, ruleRef: issue.RuleRef,
			dimension: issue.Dimension, field: issue.Field,
			detailKey: issue.DetailKey,
			dependsOn: append([]DimensionRef(nil), issue.DependsOn...),
		}
	}
	for index, question := range document.Questions {
		projected := Question{
			ref: question.Ref, dimension: question.Dimension,
			slot: question.Slot, decisionKind: question.DecisionKind,
			derivedFrom: append([]IssueRef(nil), question.DerivedFrom...),
			dependsOn:   append([]QuestionRef(nil), question.DependsOn...),
			promptKey:   question.PromptKey, whyKey: question.WhyKey,
			helpKey: question.HelpKey, exampleKey: question.ExampleKey,
			options: make([]Option, len(question.Options)),
		}
		if question.PackRef != "" {
			projected.packRef, _ = catalog.NewPackRef(question.PackRef)
		}
		for optionIndex, option := range question.Options {
			projected.options[optionIndex] = Option{
				ref: option.Ref, kind: option.Kind,
				labelKey: option.LabelKey, helpKey: option.HelpKey,
				exampleKey:   option.ExampleKey,
				rationaleKey: option.RationaleKey,
				recommended:  option.Recommended,
			}
		}
		result.questions[index] = projected
	}
	for index, proposal := range document.DefaultProposals {
		result.defaults[index] = DefaultProposal{
			dimension: proposal.Dimension, decisionKind: proposal.DecisionKind,
			option: proposal.Option, labelKey: proposal.LabelKey,
			helpKey: proposal.HelpKey, exampleKey: proposal.ExampleKey,
			rationaleKey: proposal.RationaleKey,
		}
	}
	for index, value := range document.PackRefs {
		result.packRefs[index], _ = catalog.NewPackRef(value)
	}
	result.issues = nilIfEmpty(result.issues)
	result.questions = nilIfEmpty(result.questions)
	result.defaults = nilIfEmpty(result.defaults)
	result.packRefs = nilIfEmpty(result.packRefs)
	return result
}

func nilIfEmpty[T any](values []T) []T {
	if len(values) == 0 {
		return nil
	}
	return values
}

type resultSnapshotInventory struct {
	dimensions     map[DimensionRef]DimensionDescriptor
	questions      map[QuestionRef]Question
	r3IssueFields  map[IssueRef]SlotKey
	issueOrder     map[IssueRef]int
	questionOrder  map[QuestionRef]int
	dimensionOrder map[DimensionRef]int
}

func validateResultSnapshotDocument(document resultSnapshotDocument) error {
	if document.Schema != ResultSnapshotSchema {
		return invalidResultSnapshot("schema")
	}
	if document.ResultSchema != SchemaVersion {
		return invalidResultSnapshot("result_schema")
	}
	if document.Issues == nil ||
		document.Questions == nil ||
		document.DefaultProposals == nil ||
		document.PackRefs == nil {
		return invalidResultSnapshot("snapshot")
	}
	if len(document.Issues) > maxResultSnapshotIssues {
		return invalidResultSnapshot("issues")
	}
	if len(document.Questions) > maxResultSnapshotQuestions {
		return invalidResultSnapshot("questions")
	}
	if len(document.DefaultProposals) > maxResultSnapshotDefaults {
		return invalidResultSnapshot("default_proposals")
	}
	if len(document.PackRefs) > maxResultSnapshotPackRefs {
		return invalidResultSnapshot("pack_refs")
	}

	packRefs, err := validateSnapshotPackRefs(document.PackRefs)
	if err != nil {
		return err
	}
	inventory := buildResultSnapshotInventory(packRefs)

	issues := make(map[IssueRef]resultSnapshotIssue, len(document.Issues))
	previousIssueOrder := -1
	for index, issue := range document.Issues {
		field := snapshotIndex("issues", index)
		if (issue.Kind != IssueGap && issue.Kind != IssueContradiction) ||
			issue.DependsOn == nil ||
			len(issue.DependsOn) > maxResultSnapshotDependencies ||
			!validSnapshotIssue(issue, inventory) {
			return invalidResultSnapshot(field)
		}
		order, ordered := inventory.issueOrder[issue.Ref]
		if !ordered || order <= previousIssueOrder {
			return invalidResultSnapshot(field + ".ref")
		}
		if _, duplicate := issues[issue.Ref]; duplicate {
			return invalidResultSnapshot(field + ".ref")
		}
		previousIssueOrder = order
		issues[issue.Ref] = issue
	}

	questionRefs := make(map[QuestionRef]struct{}, len(document.Questions))
	questionDimensions := make(
		map[DimensionRef]struct{},
		len(document.Questions),
	)
	optionRefs := make(map[OptionRef]struct{})
	referencedIssues := make(map[IssueRef]struct{}, len(issues))
	previousQuestionOrder := -1
	for index, question := range document.Questions {
		field := snapshotIndex("questions", index)
		expected, known := inventory.questions[question.Ref]
		if !known {
			return invalidResultSnapshot(field + ".ref")
		}
		order, ordered := inventory.questionOrder[question.Ref]
		if !ordered || order <= previousQuestionOrder {
			return invalidResultSnapshot(field + ".ref")
		}
		if question.Dimension != expected.dimension ||
			question.PackRef != expected.packRef.String() ||
			question.Slot != expected.slot ||
			question.DecisionKind != expected.decisionKind ||
			question.PromptKey != expected.promptKey ||
			question.WhyKey != expected.whyKey ||
			question.HelpKey != expected.helpKey ||
			question.ExampleKey != expected.exampleKey {
			return invalidResultSnapshot(field)
		}
		if _, duplicate := questionRefs[question.Ref]; duplicate {
			return invalidResultSnapshot(field + ".ref")
		}
		previousQuestionOrder = order
		if len(question.DerivedFrom) == 0 ||
			len(question.DerivedFrom) > maxResultSnapshotCausalRefs {
			return invalidResultSnapshot(field + ".derived_from")
		}
		expectedDerivedFrom := make(
			[]IssueRef,
			0,
			len(question.DerivedFrom),
		)
		for _, issue := range document.Issues {
			if snapshotIssueCausesQuestion(issue, question) {
				expectedDerivedFrom = append(expectedDerivedFrom, issue.Ref)
			}
		}
		if !equalIssueRefs(question.DerivedFrom, expectedDerivedFrom) {
			return invalidResultSnapshot(field + ".derived_from")
		}
		for _, ref := range expectedDerivedFrom {
			if _, duplicate := referencedIssues[ref]; duplicate {
				return invalidResultSnapshot(field + ".derived_from")
			}
			referencedIssues[ref] = struct{}{}
		}
		if question.DependsOn == nil ||
			len(question.DependsOn) > maxResultSnapshotDependencies ||
			!equalQuestionRefs(
				question.DependsOn,
				snapshotQuestionDependencies(question.DerivedFrom, issues),
			) {
			return invalidResultSnapshot(field + ".depends_on")
		}
		if err := validateSnapshotOptions(
			question.Options,
			expected,
			optionRefs,
			field+".options",
		); err != nil {
			return err
		}
		if question.Dimension != "" {
			questionDimensions[question.Dimension] = struct{}{}
		}
		questionRefs[question.Ref] = struct{}{}
	}
	if len(referencedIssues) != len(issues) {
		return invalidResultSnapshot("issues")
	}

	defaultDimensions := make(map[DimensionRef]struct{}, len(document.DefaultProposals))
	previousDefaultOrder := -1
	for index, proposal := range document.DefaultProposals {
		field := snapshotIndex("default_proposals", index)
		dimension, known := inventory.dimensions[proposal.Dimension]
		if !known || dimension.Layer() != LayerTechnical ||
			proposal.DecisionKind != DecisionTechnicalDefault {
			return invalidResultSnapshot(field)
		}
		order, ordered := inventory.dimensionOrder[proposal.Dimension]
		if !ordered || order <= previousDefaultOrder {
			return invalidResultSnapshot(field + ".dimension")
		}
		if _, duplicate := defaultDimensions[proposal.Dimension]; duplicate {
			return invalidResultSnapshot(field + ".dimension")
		}
		if _, questioned := questionDimensions[proposal.Dimension]; questioned {
			return invalidResultSnapshot(field + ".dimension")
		}
		option, found := findOption(dimension.options, proposal.Option)
		prefix := MessageKey("wizard.gaps.default." + lowerRef(proposal.Dimension))
		if !found ||
			!validSnapshotDefaultOption(dimension, proposal.Option) ||
			proposal.LabelKey != prefix+".label" ||
			proposal.HelpKey != prefix+".help" ||
			proposal.ExampleKey != prefix+".example" ||
			proposal.RationaleKey != option.rationaleKey {
			return invalidResultSnapshot(field)
		}
		previousDefaultOrder = order
		defaultDimensions[proposal.Dimension] = struct{}{}
	}
	return nil
}

func validateSnapshotPackRefs(values []string) ([]catalog.PackRef, error) {
	refs := make([]catalog.PackRef, len(values))
	for index, value := range values {
		ref, err := catalog.NewPackRef(value)
		if err != nil {
			return nil, invalidResultSnapshot(snapshotIndex("pack_refs", index))
		}
		refs[index] = ref
	}
	canonical, err := validatePackRefs(refs)
	if err != nil || len(canonical) != len(refs) {
		return nil, invalidResultSnapshot("pack_refs")
	}
	for index := range refs {
		if refs[index] != canonical[index] {
			return nil, invalidResultSnapshot(snapshotIndex("pack_refs", index))
		}
	}
	return refs, nil
}

func buildResultSnapshotInventory(packRefs []catalog.PackRef) resultSnapshotInventory {
	dimensions := dimensionsByRef(BuiltInV1().Dimensions())
	questions := make(map[QuestionRef]Question, len(dimensions))
	dimensionOrder := make(map[DimensionRef]int, len(dimensions))
	questionOrder := make(map[QuestionRef]int, len(dimensions))
	issueOrder := make(map[IssueRef]int, len(dimensions))
	for index, dimension := range BuiltInV1().Dimensions() {
		ref := questionRef(dimension.ref)
		questions[ref] = Question{
			ref: ref, dimension: dimension.ref, slot: dimension.slot,
			decisionKind: dimension.decisionKind,
			promptKey:    dimension.promptKey, whyKey: dimension.whyKey,
			helpKey: dimension.helpKey, exampleKey: dimension.exampleKey,
			options: append([]Option(nil), dimension.options...),
		}
		dimensionOrder[dimension.ref] = index
		questionOrder[ref] = len(questionOrder)
		issueOrder[dimensionIssueRef(dimension.ref)] = len(issueOrder)
	}

	supplemental := supplementalQuestionCatalog(packRefs)
	for _, ref := range []QuestionRef{
		"intake-question:wizard.r5",
		"intake-question:wizard.r8",
	} {
		questions[ref] = supplemental[ref]
	}

	selectedPacks := make(map[string]struct{}, len(packRefs))
	for _, ref := range packRefs {
		selectedPacks[ref.String()] = struct{}{}
	}
	r3IssueFields := make(map[IssueRef]SlotKey)
	seenSlots := make(map[SlotKey]struct{})
	var r3IssueRefs []IssueRef
	var packQuestionRefs []QuestionRef
	for _, pack := range catalog.BuiltInV1().Packs() {
		if _, selected := selectedPacks[pack.Ref().String()]; !selected {
			continue
		}
		for _, source := range pack.Questions() {
			question := packQuestion(pack.Ref(), source)
			if _, applicable := supplemental[question.ref]; !applicable {
				continue
			}
			if _, duplicate := seenSlots[question.slot]; duplicate {
				continue
			}
			questions[question.ref] = question
			issueRef := IssueRef(
				"intake-issue:wizard.rule.r3." +
					strings.TrimPrefix(source.Ref().String(), "question:"),
			)
			r3IssueFields[issueRef] = question.slot
			r3IssueRefs = append(r3IssueRefs, issueRef)
			packQuestionRefs = append(packQuestionRefs, question.ref)
			seenSlots[question.slot] = struct{}{}
		}
	}
	for _, ref := range []RuleRef{
		RuleR1, RuleR2, RuleR3, RuleR4,
		RuleR5, RuleR6, RuleR7, RuleR8,
	} {
		if ref == RuleR3 {
			for _, issueRef := range r3IssueRefs {
				issueOrder[issueRef] = len(issueOrder)
			}
			continue
		}
		issueOrder[ruleIssueRef(ref)] = len(issueOrder)
	}
	for _, ref := range packQuestionRefs {
		questionOrder[ref] = len(questionOrder)
	}
	for _, ref := range []QuestionRef{
		"intake-question:wizard.r5",
		"intake-question:wizard.r8",
	} {
		questionOrder[ref] = len(questionOrder)
	}
	return resultSnapshotInventory{
		dimensions: dimensions, questions: questions,
		r3IssueFields: r3IssueFields, issueOrder: issueOrder,
		questionOrder: questionOrder, dimensionOrder: dimensionOrder,
	}
}

func validSnapshotIssue(
	issue resultSnapshotIssue,
	inventory resultSnapshotInventory,
) bool {
	if issue.RuleRef == "" {
		dimension, found := inventory.dimensions[issue.Dimension]
		if !found {
			return false
		}
		return issue.Ref == dimensionIssueRef(issue.Dimension) &&
			issue.Field == dimension.slot &&
			issue.DetailKey == MessageKey(
				"wizard.gaps.dimension."+
					lowerRef(issue.Dimension)+
					".issue."+
					string(issue.Kind),
			) &&
			equalDimensionRefs(issue.DependsOn, dimension.dependsOn)
	}

	var dimension DimensionRef
	var field SlotKey
	var dependencies []DimensionRef
	switch issue.RuleRef {
	case RuleR1:
		dimension = DimensionU1
	case RuleR2:
		dimension = DimensionU2
	case RuleR3:
		var found bool
		field, found = inventory.r3IssueFields[issue.Ref]
		if !found {
			return false
		}
		dependencies = []DimensionRef{DimensionU7}
	case RuleR4:
		dimension = DimensionT4
		dependencies = []DimensionRef{DimensionU4}
	case RuleR5:
		field = "product.integration_governance"
		dependencies = []DimensionRef{DimensionU7}
	case RuleR6:
		dimension = DimensionU2
	case RuleR7:
		dimension = DimensionU10
		dependencies = []DimensionRef{DimensionU2}
	case RuleR8:
		field = "product.target_users"
		dependencies = []DimensionRef{DimensionU1}
	default:
		return false
	}
	if dimension != "" {
		field = inventory.dimensions[dimension].slot
	}
	validRef := issue.Ref == ruleIssueRef(issue.RuleRef)
	if issue.RuleRef == RuleR3 {
		_, validRef = inventory.r3IssueFields[issue.Ref]
	}
	return validRef &&
		issue.Dimension == dimension &&
		issue.Field == field &&
		issue.DetailKey == MessageKey(
			"wizard.gaps.rule."+
				lowerRuleRef(issue.RuleRef)+"."+
				string(issue.Kind),
		) &&
		equalDimensionRefs(issue.DependsOn, dependencies)
}

func snapshotIssueCausesQuestion(
	issue resultSnapshotIssue,
	question resultSnapshotQuestion,
) bool {
	if question.Dimension != "" {
		return issue.Dimension == question.Dimension
	}
	switch question.Ref {
	case "intake-question:wizard.r5":
		return issue.RuleRef == RuleR5
	case "intake-question:wizard.r8":
		return issue.RuleRef == RuleR8
	default:
		return question.PackRef != "" &&
			issue.RuleRef == RuleR3 &&
			issue.Field == question.Slot
	}
}

func snapshotQuestionDependencies(
	causes []IssueRef,
	issues map[IssueRef]resultSnapshotIssue,
) []QuestionRef {
	byDimension := make(map[DimensionRef]struct{})
	for _, cause := range causes {
		for _, dependency := range issues[cause].DependsOn {
			byDimension[dependency] = struct{}{}
		}
	}
	result := make([]QuestionRef, 0, len(byDimension))
	for _, dimension := range BuiltInV1().Dimensions() {
		if _, found := byDimension[dimension.ref]; found {
			result = append(result, questionRef(dimension.ref))
		}
	}
	return result
}

func validateSnapshotOptions(
	values []resultSnapshotOption,
	expected Question,
	globalRefs map[OptionRef]struct{},
	field string,
) error {
	if len(values) < 2 || len(values) > maxResultSnapshotOptions ||
		len(values) != len(expected.options) {
		return invalidResultSnapshot(field)
	}
	recommended, freeText := 0, 0
	var recommendedRef OptionRef
	for index, option := range values {
		itemField := snapshotIndex(field, index)
		want := expected.options[index]
		if option.Ref != want.ref ||
			option.Kind != want.kind ||
			option.LabelKey != want.labelKey ||
			option.HelpKey != want.helpKey ||
			option.ExampleKey != want.exampleKey ||
			option.RationaleKey != want.rationaleKey {
			return invalidResultSnapshot(itemField)
		}
		if _, duplicate := globalRefs[option.Ref]; duplicate {
			return invalidResultSnapshot(itemField + ".ref")
		}
		if option.Recommended {
			if option.Kind == OptionFreeText {
				return invalidResultSnapshot(itemField + ".recommended")
			}
			recommended++
			recommendedRef = option.Ref
		}
		switch option.Kind {
		case OptionPreset:
		case OptionFreeText:
			freeText++
		default:
			return invalidResultSnapshot(itemField + ".kind")
		}
		globalRefs[option.Ref] = struct{}{}
	}
	if recommended != 1 ||
		freeText != 1 ||
		!validSnapshotRecommendation(expected, recommendedRef) {
		return invalidResultSnapshot(field)
	}
	return nil
}

func validSnapshotRecommendation(question Question, ref OptionRef) bool {
	for _, option := range question.options {
		if option.recommended && option.ref == ref {
			return true
		}
	}
	switch question.dimension {
	case DimensionU1:
		return ref == optionRef(DimensionU1, "personal")
	case DimensionU2:
		return ref == optionRef(DimensionU2, "native_mobile") ||
			ref == optionRef(DimensionU2, "native_desktop") ||
			ref == optionRef(DimensionU2, "no_ui")
	case DimensionU3:
		return ref == optionRef(DimensionU3, "no_login")
	case DimensionU6:
		return ref == optionRef(DimensionU6, "none")
	case DimensionU10:
		return ref == optionRef(DimensionU10, "distribution_store") ||
			ref == optionRef(DimensionU10, "local_device")
	case DimensionT3:
		return ref == optionRef(DimensionT3, "kernel_trace")
	case DimensionT4:
		return ref == optionRef(DimensionT4, "postgres_restore")
	case DimensionT6:
		return ref == optionRef(DimensionT6, "kernel_build_ci")
	default:
		return false
	}
}

func validSnapshotDefaultOption(
	dimension DimensionDescriptor,
	ref OptionRef,
) bool {
	if ref == dimension.defaultRef {
		return true
	}
	return dimension.ref == DimensionT4 &&
		ref == optionRef(DimensionT4, "postgres_restore")
}

func equalDimensionRefs(left, right []DimensionRef) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func equalQuestionRefs(left, right []QuestionRef) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func equalIssueRefs(left, right []IssueRef) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func marshalResultSnapshotDocument(document resultSnapshotDocument) ([]byte, error) {
	document.Digest = ""
	payload, err := json.Marshal(document)
	if err != nil {
		return nil, invalidResultSnapshot("snapshot_json")
	}
	document.Digest = resultSnapshotWireChecksum(payload)
	encoded, err := json.Marshal(document)
	if err != nil {
		return nil, invalidResultSnapshot("snapshot_json")
	}
	return encoded, nil
}

func requireResultSnapshotEOF(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return invalidResultSnapshot("snapshot_json")
	}
	return nil
}

func resultSnapshotExternalDigest(document []byte) string {
	return resultSnapshotSHA256(document)
}

func resultSnapshotWireChecksum(payload []byte) string {
	return resultSnapshotSHA256(payload)
}

func resultSnapshotSHA256(value []byte) string {
	digest := sha256.Sum256(value)
	return hex.EncodeToString(digest[:])
}

func validResultSnapshotDigest(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	for _, char := range value {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}

func snapshotIndex(field string, index int) string {
	return field + "[" + strconv.Itoa(index) + "]"
}

func invalidResultSnapshot(field string) error {
	return domainError(ErrorInvalidResultSnapshot, field)
}
