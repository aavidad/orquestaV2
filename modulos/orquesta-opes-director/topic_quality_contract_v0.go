package orquestaopesdirector

import (
	"strconv"
	"strings"
	"unicode"
)

const (
	OPESTopicQualityContractSchemaV0 = "opes_topic_quality_contract.v0"

	OPESTopicQualityStatusCompleteV0    = "complete"
	OPESTopicQualityStatusNeedsReworkV0 = "needs_rework"

	OPESTopicQualityLevelBV0         = "B"
	OPESTopicQualityMinWordsLevelBV0 = 10800

	ErrOPESTopicQualityWordsRequiredV0          = "opes_topic_words_required"
	ErrOPESTopicQualityMinWordsNotMetV0         = "needs_expansion_min_words_B"
	ErrOPESTopicQualityPublicMetacommentV0      = "opes_public_text_metacomment"
	ErrOPESTopicQualityDidacticVisualRequiredV0 = "opes_visual_didactic_function_required"
)

type OPESTopicQualityContractRequestV0 struct {
	TopicRef              string
	Level                 string
	Text                  string
	WordCount             int
	WordCountText         string
	RequireDidacticVisual bool
	Visuals               []OPESTopicQualityVisualEvidenceV0
	EvidenceRefs          []string
}

type OPESTopicQualityVisualEvidenceV0 struct {
	VisualRef        string
	Raster           bool
	AnchorRef        string
	DidacticFunction string
	EvidenceRefs     []string
}

type OPESTopicQualityContractResultV0 struct {
	SchemaVersion string                           `json:"schema_version"`
	Status        string                           `json:"status"`
	TopicRef      string                           `json:"topic_ref,omitempty"`
	Level         string                           `json:"level,omitempty"`
	WordCount     int                              `json:"word_count,omitempty"`
	MinWords      int                              `json:"min_words,omitempty"`
	EvidenceRefs  []string                         `json:"evidence_refs,omitempty"`
	Issues        []OPESTopicQualityIssueV0        `json:"issues,omitempty"`
	Visuals       []OPESTopicQualityVisualResultV0 `json:"visuals,omitempty"`
}

type OPESTopicQualityVisualResultV0 struct {
	VisualRef        string   `json:"visual_ref,omitempty"`
	Raster           bool     `json:"raster,omitempty"`
	AnchorRef        string   `json:"anchor_ref,omitempty"`
	DidacticFunction string   `json:"didactic_function,omitempty"`
	EvidenceRefs     []string `json:"evidence_refs,omitempty"`
}

type OPESTopicQualityIssueV0 struct {
	Code         string   `json:"code"`
	Field        string   `json:"field,omitempty"`
	Message      string   `json:"message,omitempty"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

func ValidateOPESTopicQualityContractV0(
	request OPESTopicQualityContractRequestV0,
) OPESTopicQualityContractResultV0 {
	request = normalizeOPESTopicQualityContractRequestV0(request)
	result := OPESTopicQualityContractResultV0{
		SchemaVersion: OPESTopicQualityContractSchemaV0,
		Status:        OPESTopicQualityStatusCompleteV0,
		TopicRef:      request.TopicRef,
		Level:         request.Level,
		WordCount:     opesTopicQualityWordCountV0(request),
		MinWords:      opesTopicQualityMinWordsV0(request.Level),
		EvidenceRefs:  compactStringsV0(request.EvidenceRefs),
		Visuals:       opesTopicQualityVisualResultsV0(request.Visuals),
	}
	if result.WordCount <= 0 {
		result.Issues = append(result.Issues, opesTopicQualityIssueV0(
			ErrOPESTopicQualityWordsRequiredV0,
			"word_count",
			"word count is required for OPES public topic quality validation",
		))
	}
	if result.MinWords > 0 && result.WordCount > 0 && result.WordCount < result.MinWords {
		result.Issues = append(result.Issues, opesTopicQualityIssueV0(
			ErrOPESTopicQualityMinWordsNotMetV0,
			"word_count",
			"topic text is below OPES level B minimum words",
		))
	}
	for _, phrase := range opesTopicPublicMetacommentMatchesV0(request.Text) {
		result.Issues = append(result.Issues, OPESTopicQualityIssueV0{
			Code:    ErrOPESTopicQualityPublicMetacommentV0,
			Field:   "text",
			Message: phrase,
		})
	}
	if request.RequireDidacticVisual && !opesTopicHasDidacticVisualV0(request.Visuals) {
		result.Issues = append(result.Issues, opesTopicQualityIssueV0(
			ErrOPESTopicQualityDidacticVisualRequiredV0,
			"visuals",
			"raster visual evidence must include didactic function and topic anchor",
		))
	}
	if len(result.Issues) > 0 {
		result.Status = OPESTopicQualityStatusNeedsReworkV0
	}
	return result
}

func normalizeOPESTopicQualityContractRequestV0(
	request OPESTopicQualityContractRequestV0,
) OPESTopicQualityContractRequestV0 {
	request.TopicRef = strings.TrimSpace(request.TopicRef)
	request.Level = strings.ToUpper(strings.TrimSpace(request.Level))
	request.Text = strings.TrimSpace(request.Text)
	request.WordCountText = strings.TrimSpace(request.WordCountText)
	request.EvidenceRefs = compactStringsV0(request.EvidenceRefs)
	for index := range request.Visuals {
		request.Visuals[index].VisualRef = strings.TrimSpace(request.Visuals[index].VisualRef)
		request.Visuals[index].AnchorRef = strings.TrimSpace(request.Visuals[index].AnchorRef)
		request.Visuals[index].DidacticFunction = strings.TrimSpace(request.Visuals[index].DidacticFunction)
		request.Visuals[index].EvidenceRefs = compactStringsV0(request.Visuals[index].EvidenceRefs)
	}
	return request
}

func opesTopicQualityWordCountV0(request OPESTopicQualityContractRequestV0) int {
	if request.WordCount > 0 {
		return request.WordCount
	}
	if parsed := parseOPESTopicQualityWordCountV0(request.WordCountText); parsed > 0 {
		return parsed
	}
	return countOPESTopicWordsV0(request.Text)
}

func opesTopicQualityMinWordsV0(level string) int {
	switch strings.ToUpper(strings.TrimSpace(level)) {
	case OPESTopicQualityLevelBV0:
		return OPESTopicQualityMinWordsLevelBV0
	default:
		return 0
	}
}

func parseOPESTopicQualityWordCountV0(value string) int {
	token := firstOPESTopicQualityNumberTokenV0(value)
	if token == "" {
		return 0
	}
	normalized := strings.NewReplacer(".", "", ",", "", " ", "").Replace(token)
	parsed, err := strconv.Atoi(normalized)
	if err != nil || parsed <= 0 {
		return 0
	}
	return parsed
}

func firstOPESTopicQualityNumberTokenV0(value string) string {
	var b strings.Builder
	started := false
	for _, r := range value {
		if unicode.IsDigit(r) || (started && (r == '.' || r == ',' || unicode.IsSpace(r))) {
			b.WriteRune(r)
			started = true
			continue
		}
		if started {
			break
		}
	}
	return strings.TrimSpace(b.String())
}

func countOPESTopicWordsV0(value string) int {
	count := 0
	inWord := false
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if !inWord {
				count++
			}
			inWord = true
			continue
		}
		inWord = false
	}
	return count
}

func opesTopicPublicMetacommentMatchesV0(text string) []string {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" {
		return nil
	}
	patterns := []string{
		"en examen",
		"para examen",
		"nota de test",
		"para este tema basta",
		"para el nivel de este tema",
		"lo evaluable en este tema",
		"debe estudiarse",
	}
	var matches []string
	for _, pattern := range patterns {
		if strings.Contains(normalized, pattern) {
			matches = append(matches, pattern)
		}
	}
	return compactStringsV0(matches)
}

func opesTopicHasDidacticVisualV0(visuals []OPESTopicQualityVisualEvidenceV0) bool {
	for _, visual := range visuals {
		if visual.Raster &&
			strings.TrimSpace(visual.AnchorRef) != "" &&
			strings.TrimSpace(visual.DidacticFunction) != "" {
			return true
		}
	}
	return false
}

func opesTopicQualityVisualResultsV0(
	visuals []OPESTopicQualityVisualEvidenceV0,
) []OPESTopicQualityVisualResultV0 {
	out := make([]OPESTopicQualityVisualResultV0, 0, len(visuals))
	for _, visual := range visuals {
		visual.VisualRef = strings.TrimSpace(visual.VisualRef)
		visual.AnchorRef = strings.TrimSpace(visual.AnchorRef)
		visual.DidacticFunction = strings.TrimSpace(visual.DidacticFunction)
		out = append(out, OPESTopicQualityVisualResultV0{
			VisualRef:        visual.VisualRef,
			Raster:           visual.Raster,
			AnchorRef:        visual.AnchorRef,
			DidacticFunction: visual.DidacticFunction,
			EvidenceRefs:     compactStringsV0(visual.EvidenceRefs),
		})
	}
	return out
}

func opesTopicQualityIssueV0(code string, field string, message string) OPESTopicQualityIssueV0 {
	return OPESTopicQualityIssueV0{
		Code:    strings.TrimSpace(code),
		Field:   strings.TrimSpace(field),
		Message: strings.TrimSpace(message),
	}
}
