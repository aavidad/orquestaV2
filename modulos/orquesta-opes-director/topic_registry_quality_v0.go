package orquestaopesdirector

import (
	"encoding/json"
	"strings"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

const topicRegistryQualityNeedsReworkRefV0 = "topic-quality-needs-rework"

func topicRegistryQualityResultForRecordV0(
	record OPESCausalArtifactRecordV0,
) (OPESTopicQualityContractResultV0, bool) {
	request, ok := topicRegistryQualityRequestForRecordV0(record)
	if !ok {
		return OPESTopicQualityContractResultV0{}, false
	}
	return ValidateOPESTopicQualityContractV0(request), true
}

func topicRegistryQualityRequestForRecordV0(
	record OPESCausalArtifactRecordV0,
) (OPESTopicQualityContractRequestV0, bool) {
	fields := record.PayloadFields
	request := OPESTopicQualityContractRequestV0{
		TopicRef: fieldStringV0(fields, "topic_id", "topic_ref"),
		Level: firstNonEmptyV0(
			fieldStringV0(fields, "level"),
			fieldStringV0(fields, "topic_level"),
			fieldStringV0(fields, "quality_level"),
			OPESTopicQualityLevelBV0,
		),
		Text: firstNonEmptyV0(
			fieldStringV0(fields, "topic_text"),
			fieldStringV0(fields, "public_text"),
			fieldStringV0(fields, "markdown"),
			fieldStringV0(fields, "text"),
			fieldStringV0(fields, "content"),
			fieldStringV0(fields, "tema_ampliado"),
		),
		CanonicalWordCount: fieldIntV0(
			fields,
			"canonical_word_count",
			"canonical_words",
			"strict_word_count",
			"strict_public_word_count",
		),
		CanonicalWordCountText: fieldStringV0(fields, "canonical_word_count_text", "strict_word_count_text"),
		WordCount:              fieldIntV0(fields, "word_count", "words", "declared_word_count"),
		WordCountText:          fieldStringV0(fields, "word_count_text", "wc_output"),
		WordCountSource:        fieldStringV0(fields, "word_count_source", "word_counter"),
		ReportStatus: firstNonEmptyV0(
			fieldStringV0(fields, "topic_quality_status"),
			fieldStringV0(fields, "quality_status"),
			fieldStringV0(fields, "qa_status"),
			fieldStringV0(fields, "strict_editorial_qa_status"),
		),
		ReportText: firstNonEmptyV0(
			fieldStringV0(fields, "topic_quality_report"),
			fieldStringV0(fields, "quality_report"),
			fieldStringV0(fields, "qa_report"),
			fieldStringV0(fields, "strict_editorial_qa_report"),
		),
		RequireDidacticVisual: topicRegistryQualityBoolFieldV0(fields, "require_didactic_visual", "visual_required"),
		Visuals:               topicRegistryQualityVisualsForFieldsV0(fields),
		EvidenceRefs: compactStringsV0(append(
			append([]string(nil), record.EvidenceRefs...),
			fieldStringsV0(fields, "topic_quality_evidence_refs", "qa_report_refs", "validation_refs")...,
		)),
	}
	if !topicRegistryRecordDeclaresTopicQualityV0(record, request) {
		return OPESTopicQualityContractRequestV0{}, false
	}
	return request, true
}

func topicRegistryRecordDeclaresTopicQualityV0(
	record OPESCausalArtifactRecordV0,
	request OPESTopicQualityContractRequestV0,
) bool {
	if strings.TrimSpace(request.Text) != "" ||
		request.CanonicalWordCount > 0 ||
		strings.TrimSpace(request.CanonicalWordCountText) != "" ||
		request.WordCount > 0 ||
		strings.TrimSpace(request.WordCountText) != "" ||
		strings.TrimSpace(request.ReportStatus) != "" ||
		strings.TrimSpace(request.ReportText) != "" {
		return true
	}
	if opesDirectorIsFinalPackageArtifactTypeV0(record.ArtifactType) {
		return false
	}
	values := append([]string(nil), record.EvidenceRefs...)
	values = append(values, record.PayloadRefs...)
	values = append(values, fieldStringsV0(record.PayloadFields, "topic_quality_evidence_refs", "qa_report_refs", "validation_refs")...)
	for _, value := range values {
		if topicRegistryQualityRefLooksExplicitV0(value) {
			return true
		}
	}
	return false
}

func topicRegistryQualityRefLooksExplicitV0(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return false
	}
	for _, token := range []string{
		"topic_quality",
		"mojibake",
		"strict_editorial",
		"official_text_qa",
		"extension_pass",
		"informe_texto_publico",
		"andamiaje_interno",
		"contaminacion_estructural",
		"structural_contamination",
		"metacomentarios_examen",
	} {
		if strings.Contains(normalized, token) {
			return true
		}
	}
	return false
}

func topicRegistryQualityPendingRefsForRecordV0(record OPESCausalArtifactRecordV0) []string {
	result, ok := topicRegistryQualityResultForRecordV0(record)
	if !ok || result.Status != OPESTopicQualityStatusNeedsReworkV0 {
		return nil
	}
	refs := []string{topicRegistryQualityNeedsReworkRefV0}
	for _, issue := range result.Issues {
		if code := strings.TrimSpace(issue.Code); code != "" {
			refs = append(refs, "topic-quality-"+safeRefV0(code))
		}
	}
	return compactStringsV0(refs)
}

func topicRegistryQualityFieldsForRecordV0(record OPESCausalArtifactRecordV0) []orquestadomainwork.DomainWorkFieldV0 {
	result, ok := topicRegistryQualityResultForRecordV0(record)
	if !ok {
		return nil
	}
	issueRefs := make([]string, 0, len(result.Issues))
	for _, issue := range result.Issues {
		if code := strings.TrimSpace(issue.Code); code != "" {
			issueRefs = append(issueRefs, code)
		}
	}
	return []orquestadomainwork.DomainWorkFieldV0{
		{Name: "topic_quality_status", Value: result.Status},
		{Name: "topic_quality_issue_refs", Values: compactStringsV0(issueRefs)},
		{Name: "topic_quality_evidence_refs", Values: compactStringsV0(result.EvidenceRefs)},
	}
}

func topicRegistryQualityBoolFieldV0(fields []orquestadomainwork.DomainWorkFieldV0, names ...string) bool {
	value := strings.ToLower(fieldStringV0(fields, names...))
	switch value {
	case "true", "1", "yes", "si", "required", "obligatorio":
		return true
	default:
		return false
	}
}

type topicRegistryQualityVisualPayloadV0 struct {
	VisualRef        string   `json:"visual_ref"`
	Ref              string   `json:"ref"`
	AssetRef         string   `json:"asset_ref"`
	ArtifactRef      string   `json:"artifact_ref"`
	Raster           bool     `json:"raster"`
	VisualType       string   `json:"visual_type"`
	Format           string   `json:"format"`
	ContentType      string   `json:"content_type"`
	AnchorRef        string   `json:"anchor_ref"`
	PlacementRef     string   `json:"placement_ref"`
	TopicAnchorRef   string   `json:"topic_anchor_ref"`
	SectionRef       string   `json:"section_ref"`
	DidacticFunction string   `json:"didactic_function"`
	Objective        string   `json:"objective"`
	Purpose          string   `json:"purpose"`
	EvidenceRef      string   `json:"evidence_ref"`
	EvidenceRefs     []string `json:"evidence_refs"`
}

func topicRegistryQualityVisualsForFieldsV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
) []OPESTopicQualityVisualEvidenceV0 {
	var visuals []OPESTopicQualityVisualEvidenceV0
	visuals = append(visuals, topicRegistryQualityJSONVisualsForFieldsV0(fields)...)
	if visual, ok := topicRegistryQualityScalarVisualForFieldsV0(fields); ok {
		visuals = append(visuals, visual)
	}
	return topicRegistryQualityCompactVisualsV0(visuals)
}

func topicRegistryQualityJSONVisualsForFieldsV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
) []OPESTopicQualityVisualEvidenceV0 {
	var visuals []OPESTopicQualityVisualEvidenceV0
	for _, field := range fields {
		if !topicRegistryQualityVisualJSONFieldNameV0(field.Name) {
			continue
		}
		if len(field.ValueJSON) > 0 {
			visuals = append(visuals, topicRegistryQualityVisualsFromRawJSONV0(field.ValueJSON)...)
		}
		if value := strings.TrimSpace(field.Value); value != "" {
			visuals = append(visuals, topicRegistryQualityVisualsFromRawJSONV0([]byte(value))...)
		}
		for _, value := range field.Values {
			if value = strings.TrimSpace(value); value != "" {
				visuals = append(visuals, topicRegistryQualityVisualsFromRawJSONV0([]byte(value))...)
			}
		}
	}
	return visuals
}

func topicRegistryQualityVisualJSONFieldNameV0(name string) bool {
	switch strings.TrimSpace(name) {
	case "visuals",
		"topic_visuals",
		"didactic_visuals",
		"visual_evidence",
		"visual_evidences",
		"visual_assets":
		return true
	default:
		return false
	}
}

func topicRegistryQualityVisualsFromRawJSONV0(raw []byte) []OPESTopicQualityVisualEvidenceV0 {
	if len(raw) == 0 || !json.Valid(raw) {
		return nil
	}
	var payloads []topicRegistryQualityVisualPayloadV0
	if err := json.Unmarshal(raw, &payloads); err != nil {
		var payload topicRegistryQualityVisualPayloadV0
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil
		}
		payloads = []topicRegistryQualityVisualPayloadV0{payload}
	}
	visuals := make([]OPESTopicQualityVisualEvidenceV0, 0, len(payloads))
	for _, payload := range payloads {
		if visual, ok := topicRegistryQualityVisualFromPayloadV0(payload); ok {
			visuals = append(visuals, visual)
		}
	}
	return visuals
}

func topicRegistryQualityVisualFromPayloadV0(
	payload topicRegistryQualityVisualPayloadV0,
) (OPESTopicQualityVisualEvidenceV0, bool) {
	visual := OPESTopicQualityVisualEvidenceV0{
		VisualRef: firstNonEmptyV0(
			payload.VisualRef,
			payload.Ref,
			payload.AssetRef,
			payload.ArtifactRef,
		),
		Raster: payload.Raster || topicRegistryQualityRasterFormatV0(
			payload.VisualType,
			payload.Format,
			payload.ContentType,
		),
		AnchorRef: firstNonEmptyV0(
			payload.AnchorRef,
			payload.PlacementRef,
			payload.TopicAnchorRef,
			payload.SectionRef,
		),
		DidacticFunction: firstNonEmptyV0(
			payload.DidacticFunction,
			payload.Objective,
			payload.Purpose,
		),
		EvidenceRefs: compactStringsV0(append(append([]string(nil), payload.EvidenceRefs...), payload.EvidenceRef)),
	}
	return visual, topicRegistryQualityVisualHasDataV0(visual)
}

func topicRegistryQualityScalarVisualForFieldsV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
) (OPESTopicQualityVisualEvidenceV0, bool) {
	visual := OPESTopicQualityVisualEvidenceV0{
		VisualRef: fieldStringV0(
			fields,
			"visual_ref",
			"visual_asset_ref",
			"didactic_visual_ref",
			"topic_visual_ref",
		),
		Raster: topicRegistryQualityBoolFieldV0(fields, "visual_raster", "is_raster", "raster") ||
			topicRegistryQualityRasterFormatV0(
				fieldStringV0(fields, "visual_type"),
				fieldStringV0(fields, "visual_format"),
				fieldStringV0(fields, "visual_content_type"),
			),
		AnchorRef: fieldStringV0(
			fields,
			"visual_anchor_ref",
			"anchor_ref",
			"placement_ref",
			"visual_placement_ref",
			"topic_anchor_ref",
			"section_ref",
		),
		DidacticFunction: fieldStringV0(
			fields,
			"visual_didactic_function",
			"didactic_function",
			"visual_objective",
			"visual_purpose",
		),
		EvidenceRefs: fieldStringsV0(fields, "visual_evidence_refs", "visual_validation_refs"),
	}
	return visual, topicRegistryQualityVisualHasDataV0(visual)
}

func topicRegistryQualityRasterFormatV0(values ...string) bool {
	for _, value := range values {
		normalized := strings.ToLower(strings.TrimSpace(value))
		switch normalized {
		case "raster", "bitmap", "png", "jpg", "jpeg", "webp", "image/png", "image/jpeg", "image/webp":
			return true
		}
	}
	return false
}

func topicRegistryQualityVisualHasDataV0(visual OPESTopicQualityVisualEvidenceV0) bool {
	return strings.TrimSpace(visual.VisualRef) != "" ||
		visual.Raster ||
		strings.TrimSpace(visual.AnchorRef) != "" ||
		strings.TrimSpace(visual.DidacticFunction) != "" ||
		len(compactStringsV0(visual.EvidenceRefs)) > 0
}

func topicRegistryQualityCompactVisualsV0(
	visuals []OPESTopicQualityVisualEvidenceV0,
) []OPESTopicQualityVisualEvidenceV0 {
	var out []OPESTopicQualityVisualEvidenceV0
	seen := map[string]bool{}
	for _, visual := range visuals {
		visual.VisualRef = strings.TrimSpace(visual.VisualRef)
		visual.AnchorRef = strings.TrimSpace(visual.AnchorRef)
		visual.DidacticFunction = strings.TrimSpace(visual.DidacticFunction)
		visual.EvidenceRefs = compactStringsV0(visual.EvidenceRefs)
		if !topicRegistryQualityVisualHasDataV0(visual) {
			continue
		}
		key := strings.Join([]string{
			visual.VisualRef,
			visual.AnchorRef,
			visual.DidacticFunction,
		}, "\x00")
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, visual)
	}
	return out
}
