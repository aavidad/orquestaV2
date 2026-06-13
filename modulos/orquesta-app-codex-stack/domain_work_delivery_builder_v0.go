package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestarails "orquesta/modulos/orquesta-rails"
)

type defaultDomainWorkArtifactSubmissionBuilderV0 struct{}

func (defaultDomainWorkArtifactSubmissionBuilderV0) BuildDomainWorkArtifactSubmissionV0(
	_ context.Context,
	input DomainWorkArtifactSubmissionBuildInputV0,
) (orquestadomainwork.DomainWorkArtifactSubmissionV0, bool, error) {
	work := input.Record.Request.ExternalWork
	if work == nil || strings.TrimSpace(work.JobRef) == "" {
		return orquestadomainwork.DomainWorkArtifactSubmissionV0{}, false, nil
	}
	fields := copyDomainWorkFieldsForContextV0(work.InputFields)
	fields = appendDomainWorkFieldIfMissingV0(fields, "source_work_kind", work.WorkKind)
	artifactType := domainWorkExpectedArtifactTypeV0(fields, work.WorkKind)
	intake, err := readDomainWorkDeliveryArtifactV0(input.Descriptor, input.Ack, artifactType)
	if err != nil {
		return orquestadomainwork.DomainWorkArtifactSubmissionV0{}, false, err
	}
	body := intake.Body
	payloadBody := domainWorkDeliveryPayloadBodyV0(body, artifactType)
	payloadBody = canonicalDomainWorkDeliveryPayloadBodyV0(artifactType, payloadBody, input)
	fields = domainWorkDeliveryPayloadFieldsV0(fields, artifactType, input.Task.Title, payloadBody)
	fields = appendDomainWorkFieldIfMissingV0(fields, "file_ref", intake.FileRef)
	fields = redactDomainWorkDeliveryPayloadFieldsV0(fields)
	return orquestadomainwork.NormalizeDomainWorkArtifactSubmissionV0(
		orquestadomainwork.DomainWorkArtifactSubmissionV0{
			RequestID:      domainWorkArtifactRequestIDV0(work, artifactType),
			CorrelationID:  input.Record.Request.CorrelationID,
			IdempotencyKey: domainWorkArtifactIdempotencyKeyV0(work, artifactType),
			RequestedBy:    "orquesta",
			DomainRef:      work.ProjectRef,
			JobRef:         work.JobRef,
			ArtifactRef:    input.Observation.DeliveryRef,
			ArtifactType:   artifactType,
			Summary:        input.Observation.Summary,
			PayloadFields:  fields,
			ExternalRefs:   domainWorkArtifactExternalRefsV0(input),
			EvidenceRefs:   append([]string{input.Descriptor.DescriptorRef}, input.Observation.EvidenceRefs...),
			CompleteJob:    true,
		},
	), true, nil
}

func domainWorkArtifactTypeForWorkKindV0(workKind string) string {
	return orquestadomainwork.ExpectedDomainWorkArtifactTypeForWorkKindV0(
		domainWorkCanonicalWorkKindForArtifactV0(workKind),
	)
}

func domainWorkExpectedArtifactTypeV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
	workKind string,
) string {
	if expected := domainWorkFieldStringValueV0(fields, "expected_artifact_type"); expected != "" {
		return expected
	}
	return domainWorkArtifactTypeForWorkKindV0(workKind)
}

func domainWorkCanonicalWorkKindForArtifactV0(workKind string) string {
	key := normalizeDomainWorkDeliveryAliasV0(workKind)
	switch key {
	case "redaccion_tema", "redaccion_documental", "investigacion_y_redaccion",
		"sintesis_pedagogica", "control_editorial", "desarrollo_contenido",
		"tema_grande":
		return "draft_content_block"
	case "visual_asset_plan", "plan_visual", "visual_assets", "revision_visual_assets",
		"revision_visual", "esquema_estudio", "diagrama_flujo":
		return "generate_visual_asset"
	case "revision_legal", "revision_legal_deontologica":
		return "review_legal"
	case "revision_pedagogica", "revision_pedagogical":
		return "review_pedagogical"
	case "revision_psicologia", "revision_contenido", "revision_contenido_tecnico",
		"revision_editorial", "revision_calidad":
		return "review_quality"
	case "review_codex", "review_gemini", "review_claude":
		return "review_agent_independent"
	case "review_pair_codex_gemini", "review_pair_codex_claude", "review_pair_gemini_claude":
		return "review_agent_pair"
	case "validacion_contrato", "validacion_documental", "validacion_tema", "validar_tema":
		return "validate_topic"
	case "ensamblado_y_exportacion", "ensamblado", "exportacion":
		return "assemble_topic"
	case "generacion_audio", "generar_audio", "crear_audio_tema", "audio_tema",
		"narracion_tema", "sintesis_voz_tema", "topic_audio":
		return "generate_audio_asset"
	default:
		return key
	}
}

func domainWorkDeliveryPayloadFieldsV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
	artifactType string,
	title string,
	body string,
) []orquestadomainwork.DomainWorkFieldV0 {
	if parsed, ok := domainWorkDeliveryJSONPayloadFieldsV0(body, artifactType); ok {
		fields = appendDomainWorkFieldIfMissingV0(
			fields,
			"content_type",
			domainWorkDeliveryContentTypeV0(fields, artifactType),
		)
		fields = appendDomainWorkFieldIfMissingV0(fields, "title", title)
		return append(fields, parsed...)
	}
	fields = appendDomainWorkFieldIfMissingV0(
		fields,
		"content_type",
		domainWorkDeliveryContentTypeV0(fields, artifactType),
	)
	fields = appendDomainWorkFieldIfMissingV0(fields, "title", title)
	if strings.TrimSpace(body) != "" {
		bodyField := "body"
		if artifactType == "topic_summary" {
			bodyField = "markdown"
		}
		fields = append(fields, orquestadomainwork.DomainWorkFieldV0{Name: bodyField, Value: strings.TrimSpace(body)})
	}
	return fields
}

func domainWorkDeliveryPayloadBodyV0(body string, artifactType string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}
	var envelope struct {
		ArtifactType string          `json:"artifact_type"`
		PayloadJSON  json.RawMessage `json:"payload_json"`
	}
	if err := json.Unmarshal([]byte(body), &envelope); err != nil || len(envelope.PayloadJSON) == 0 {
		return body
	}
	payload := strings.TrimSpace(string(envelope.PayloadJSON))
	if payload == "" || payload == "null" || !json.Valid(envelope.PayloadJSON) {
		return body
	}
	return payload
}

func domainWorkDeliveryJSONPayloadFieldsV0(
	body string,
	artifactType string,
) ([]orquestadomainwork.DomainWorkFieldV0, bool) {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, false
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &payload); err != nil || len(payload) == 0 {
		return nil, false
	}
	keys := make([]string, 0, len(payload))
	for name := range payload {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	fields := make([]orquestadomainwork.DomainWorkFieldV0, 0, len(payload))
	for _, name := range keys {
		field, ok := domainWorkDeliveryJSONFieldV0(artifactType, name, payload[name])
		if !ok {
			continue
		}
		fields = append(fields, field)
	}
	return fields, len(fields) > 0
}

func domainWorkDeliveryJSONFieldV0(
	artifactType string,
	name string,
	raw json.RawMessage,
) (orquestadomainwork.DomainWorkFieldV0, bool) {
	name = domainWorkDeliveryCanonicalPayloadFieldNameV0(artifactType, name)
	if name == "" || len(raw) == 0 || !json.Valid(raw) {
		return orquestadomainwork.DomainWorkFieldV0{}, false
	}
	if name == "source_refs" && !sourceRefsAlreadyCompactV0(raw) {
		if refs, ok := compactSourceRefsFromRichPayloadV0(raw); ok {
			return orquestadomainwork.DomainWorkFieldV0{Name: name, Values: refs}, true
		}
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return orquestadomainwork.DomainWorkFieldV0{Name: name, Value: strings.TrimSpace(text)}, true
	}
	var texts []string
	if err := json.Unmarshal(raw, &texts); err == nil {
		return orquestadomainwork.DomainWorkFieldV0{Name: name, Values: compactCodexStackStringsV0(texts)}, true
	}
	return orquestadomainwork.DomainWorkFieldV0{Name: name, ValueJSON: append([]byte(nil), raw...)}, true
}

func appendDomainWorkFieldIfMissingV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
	name string,
	value string,
) []orquestadomainwork.DomainWorkFieldV0 {
	name = strings.TrimSpace(name)
	value = strings.TrimSpace(value)
	if name == "" || value == "" || domainWorkFieldHasNameV0(fields, name) {
		return fields
	}
	return append(fields, orquestadomainwork.DomainWorkFieldV0{Name: name, Value: value})
}

func redactDomainWorkDeliveryPayloadFieldsV0(fields []orquestadomainwork.DomainWorkFieldV0) []orquestadomainwork.DomainWorkFieldV0 {
	out := make([]orquestadomainwork.DomainWorkFieldV0, 0, len(fields))
	for _, field := range fields {
		out = append(out, redactDomainWorkDeliveryPayloadFieldV0(field))
	}
	return out
}

func redactDomainWorkDeliveryPayloadFieldV0(
	field orquestadomainwork.DomainWorkFieldV0,
) orquestadomainwork.DomainWorkFieldV0 {
	if domainWorkDeliveryFieldNameRequiresRedactionV0(field.Name) {
		field.Value = "<redacted-sensitive-field>"
		field.Values = nil
		field.ValueJSON = nil
		return field
	}
	if domainWorkDeliveryReferenceFieldV0(field.Name) {
		return field
	}
	field.Value, _ = orquestarails.RedactOperationalTextForFieldV0(
		"codex_stack_domain_work_delivery",
		field.Name,
		field.Value,
	)
	for i, value := range field.Values {
		field.Values[i], _ = orquestarails.RedactOperationalTextForFieldV0(
			"codex_stack_domain_work_delivery",
			field.Name,
			value,
		)
	}
	if len(field.ValueJSON) > 0 {
		redacted, _ := orquestarails.RedactOperationalTextForFieldV0(
			"codex_stack_domain_work_delivery",
			field.Name,
			string(field.ValueJSON),
		)
		field.ValueJSON = []byte(redacted)
	}
	return field
}

func domainWorkDeliveryReferenceFieldV0(name string) bool {
	key := normalizeDomainWorkDeliveryAliasV0(name)
	return strings.HasSuffix(key, "_ref") ||
		strings.HasSuffix(key, "_refs") ||
		strings.Contains(key, "artifact_ref") ||
		strings.Contains(key, "package_ref")
}

func domainWorkDeliveryFieldNameRequiresRedactionV0(name string) bool {
	switch normalizeDomainWorkDeliveryAliasV0(name) {
	case "api_key", "access_token", "refresh_token", "client_secret", "password",
		"passwd", "pwd", "secret", "secreto", "credential", "credencial",
		"authorization", "raw_http", "http_request", "http_response",
		"provider_payload", "provider_response", "model_payload":
		return true
	default:
		return false
	}
}

func domainWorkFieldHasNameV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
	name string,
) bool {
	name = strings.TrimSpace(name)
	for _, field := range fields {
		if strings.TrimSpace(field.Name) == name {
			return true
		}
	}
	return false
}

func domainWorkDeliveryContentTypeV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
	artifactType string,
) string {
	if artifactType == orquestadomainwork.DomainDocumentPlanArtifactTypeV0 {
		return "application/json"
	}
	if artifactType == "audio_asset" {
		return "application/json"
	}
	if artifactType != "visual_asset" {
		return "text/markdown"
	}
	switch strings.ToLower(strings.TrimSpace(domainWorkFieldStringValueV0(fields, "format"))) {
	case "svg":
		return "image/svg+xml"
	case "mermaid":
		return "text/mermaid"
	case "html_panel":
		return "text/html"
	default:
		return "text/markdown"
	}
}

func domainWorkFieldStringValueV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
	name string,
) string {
	name = strings.TrimSpace(name)
	for _, field := range fields {
		if strings.TrimSpace(field.Name) == name {
			return strings.TrimSpace(field.Value)
		}
	}
	return ""
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

func domainWorkArtifactExternalRefsV0(
	input DomainWorkArtifactSubmissionBuildInputV0,
) []orquestadomainwork.DomainWorkExternalRefV0 {
	refs := []orquestadomainwork.DomainWorkExternalRefV0{
		{Kind: "run_ref", Ref: input.Run.RunID},
		{Kind: "task_ref", Ref: input.Task.TaskID},
		{Kind: "delivery_ref", Ref: input.Observation.DeliveryRef},
		{Kind: "agent_ref", Ref: input.Observation.AgentRef},
	}
	if input.Record.Request.ChangeRef != "" {
		refs = append(refs, orquestadomainwork.DomainWorkExternalRefV0{
			Kind: "change_ref",
			Ref:  input.Record.Request.ChangeRef,
		})
	}
	return refs
}
