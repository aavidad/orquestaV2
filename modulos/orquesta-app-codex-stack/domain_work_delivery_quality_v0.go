package orquestaappcodexstack

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func validateDomainWorkDeliveryQualityV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
	artifactType string,
	body string,
	payloadBody string,
) error {
	switch strings.TrimSpace(artifactType) {
	case orquestadomainwork.DomainDocumentPlanArtifactTypeV0:
		return validateDomainWorkDocumentPlanDeliveryV0(payloadBody)
	case "topic_expansion_package":
	default:
		return nil
	}
	if domainWorkExpansionPayloadHasForbiddenPlaceholderV0(payloadBody) {
		return newDomainWorkQualityIssueErrorV0(
			"placeholder_visible",
			"payload.visible_text",
			"placeholder_or_pending_marker",
		)
	}
	minWords := domainWorkExpansionMinWordsV0(fields)
	if minWords <= 0 {
		return nil
	}
	words := domainWorkExpansionPayloadWordsV0(payloadBody)
	if words < minWords {
		return domainWorkQualityIssueErrorV0{
			Issue: orquestadomainwork.DomainWorkIssueV0{
				Code:  "min_words_not_met",
				Field: "payload.chapters.blocks",
			},
			Reason:      "visible_content_below_target",
			MinWords:    minWords,
			ActualWords: words,
		}
	}
	return nil
}

func validateDomainWorkDocumentPlanDeliveryV0(payloadBody string) error {
	payloadBody = strings.TrimSpace(payloadBody)
	if payloadBody == "" {
		return fmt.Errorf("domain_work_artifact_quality_gate_failed")
	}
	var plan orquestadomainwork.DomainDocumentPlanV0
	if err := json.Unmarshal([]byte(payloadBody), &plan); err != nil {
		return newDomainWorkQualityIssueErrorV0(
			"invalid_json",
			"payload",
			"document_plan_json_unmarshal_failed",
		)
	}
	if issues := orquestadomainwork.ValidateDomainDocumentPlanV0(plan); len(issues) > 0 {
		return domainWorkQualityIssueErrorV0{
			Issue:  issues[0],
			Reason: "document_plan_validation_failed",
		}
	}
	return nil
}

type domainWorkQualityIssueErrorV0 struct {
	Issue       orquestadomainwork.DomainWorkIssueV0
	Reason      string
	MinWords    int
	ActualWords int
}

func newDomainWorkQualityIssueErrorV0(code, field, reason string) domainWorkQualityIssueErrorV0 {
	return domainWorkQualityIssueErrorV0{
		Issue: orquestadomainwork.DomainWorkIssueV0{
			Code:  code,
			Field: field,
		},
		Reason: reason,
	}
}

func (err domainWorkQualityIssueErrorV0) Error() string {
	parts := []string{"domain_work_artifact_quality_gate_failed"}
	if code := strings.TrimSpace(err.Issue.Code); code != "" {
		parts = append(parts, "code="+code)
	}
	if field := strings.TrimSpace(err.Issue.Field); field != "" {
		parts = append(parts, "field="+field)
	}
	if reason := strings.TrimSpace(err.Reason); reason != "" {
		parts = append(parts, "reason="+reason)
	}
	if err.MinWords > 0 {
		parts = append(parts, fmt.Sprintf("min_words=%d", err.MinWords))
	}
	if err.ActualWords > 0 {
		parts = append(parts, fmt.Sprintf("actual_words=%d", err.ActualWords))
	}
	return strings.Join(parts, " ")
}

func domainWorkExpansionMinWordsV0(fields []orquestadomainwork.DomainWorkFieldV0) int {
	if value := domainWorkFieldIntV0(fields, "target_words_min"); value > 0 {
		return value
	}
	if pages := domainWorkFieldIntV0(fields, "target_pages_min"); pages > 0 {
		return pages * 350
	}
	return 0
}

func domainWorkFieldIntV0(fields []orquestadomainwork.DomainWorkFieldV0, name string) int {
	for _, field := range fields {
		if strings.TrimSpace(field.Name) != name {
			continue
		}
		if value, err := strconv.Atoi(strings.TrimSpace(field.Value)); err == nil && value > 0 {
			return value
		}
		var decoded int
		if len(field.ValueJSON) > 0 && json.Unmarshal(field.ValueJSON, &decoded) == nil && decoded > 0 {
			return decoded
		}
	}
	return 0
}

type domainWorkExpansionPayloadTextV0 struct {
	Markdown string `json:"markdown"`
	Body     string `json:"body"`
	Content  string `json:"content"`
}

type domainWorkExpansionPayloadBlockV0 struct {
	Markdown string `json:"markdown"`
	Body     string `json:"body"`
}

type domainWorkExpansionPayloadChapterV0 struct {
	Blocks []domainWorkExpansionPayloadBlockV0 `json:"blocks"`
}

type domainWorkExpansionPayloadVisualV0 struct {
	Markdown string `json:"markdown"`
	Body     string `json:"body"`
}

type domainWorkExpansionPayloadDocV0 struct {
	Chapters      []domainWorkExpansionPayloadChapterV0 `json:"chapters"`
	TemaGrande    domainWorkExpansionPayloadTextV0      `json:"tema_grande"`
	TemaMediano   domainWorkExpansionPayloadTextV0      `json:"tema_mediano"`
	Resumen       domainWorkExpansionPayloadTextV0      `json:"resumen"`
	EsquemaRepaso []string                              `json:"esquema_repaso"`
	PlanVisuales  []domainWorkExpansionPayloadVisualV0  `json:"plan_visuales"`
}

func domainWorkExpansionPayloadWordsV0(body string) int {
	payload, ok := parseDomainWorkExpansionPayloadDocV0(body)
	if !ok {
		return 0
	}
	words := 0
	for _, chapter := range payload.Chapters {
		for _, block := range chapter.Blocks {
			words += countDomainWorkWordsV0(firstNonEmptyDomainWorkTextV0(block.Markdown, block.Body))
		}
	}
	return words
}

func domainWorkExpansionPayloadHasForbiddenPlaceholderV0(body string) bool {
	texts, ok := domainWorkExpansionPayloadVisibleTextsV0(body)
	if !ok {
		return domainWorkExpansionHasForbiddenPlaceholderV0(body)
	}
	for _, text := range texts {
		if domainWorkExpansionHasForbiddenPlaceholderV0(text) {
			return true
		}
	}
	return false
}

func domainWorkExpansionPayloadVisibleTextsV0(body string) ([]string, bool) {
	payload, ok := parseDomainWorkExpansionPayloadDocV0(body)
	if !ok {
		return nil, false
	}
	texts := make([]string, 0, len(payload.Chapters)+len(payload.EsquemaRepaso)+len(payload.PlanVisuales)+3)
	for _, chapter := range payload.Chapters {
		for _, block := range chapter.Blocks {
			texts = append(texts, firstNonEmptyDomainWorkTextV0(block.Markdown, block.Body))
		}
	}
	texts = append(texts,
		firstNonEmptyDomainWorkTextV0(payload.TemaGrande.Markdown, payload.TemaGrande.Body, payload.TemaGrande.Content),
		firstNonEmptyDomainWorkTextV0(payload.TemaMediano.Markdown, payload.TemaMediano.Body, payload.TemaMediano.Content),
		firstNonEmptyDomainWorkTextV0(payload.Resumen.Markdown, payload.Resumen.Body, payload.Resumen.Content),
	)
	texts = append(texts, payload.EsquemaRepaso...)
	for _, visual := range payload.PlanVisuales {
		texts = append(texts, firstNonEmptyDomainWorkTextV0(visual.Markdown, visual.Body))
	}
	return compactCodexStackStringsV0(texts), true
}

func parseDomainWorkExpansionPayloadDocV0(body string) (domainWorkExpansionPayloadDocV0, bool) {
	var payload domainWorkExpansionPayloadDocV0
	if err := json.Unmarshal([]byte(strings.TrimSpace(body)), &payload); err != nil {
		return domainWorkExpansionPayloadDocV0{}, false
	}
	return payload, true
}

func firstNonEmptyDomainWorkTextV0(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func domainWorkExpansionHasForbiddenPlaceholderV0(body string) bool {
	normalized := normalizeDomainWorkMarkerTextV0(body)
	return strings.Contains(normalized, "pendiente de ampliacion") ||
		strings.Contains(normalized, "[pendiente]") ||
		strings.Contains(normalized, "pendiente_revision") ||
		strings.Contains(normalized, "placeholder") ||
		strings.Contains(body, "TODO")
}

func normalizeDomainWorkMarkerTextV0(body string) string {
	normalized := strings.ToLower(strings.TrimSpace(body))
	replacer := strings.NewReplacer(
		"á", "a",
		"é", "e",
		"í", "i",
		"ó", "o",
		"ú", "u",
		"ü", "u",
	)
	return replacer.Replace(normalized)
}

func countDomainWorkWordsV0(value string) int {
	inWord := false
	count := 0
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if !inWord {
				count++
				inWord = true
			}
			continue
		}
		inWord = false
	}
	return count
}
