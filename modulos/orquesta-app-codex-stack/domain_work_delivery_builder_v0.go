package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
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
	body := readDomainWorkDeliveryBodyV0(input.Descriptor, input.Ack)
	fields := copyDomainWorkFieldsForContextV0(work.InputFields)
	artifactType := domainWorkArtifactTypeForWorkKindV0(work.WorkKind)
	payloadBody := domainWorkDeliveryPayloadBodyV0(body, artifactType)
	payloadBody = canonicalDomainWorkDeliveryPayloadBodyV0(artifactType, payloadBody, input)
	if err := validateDomainWorkDeliveryQualityV0(fields, artifactType, body, payloadBody); err != nil {
		return orquestadomainwork.DomainWorkArtifactSubmissionV0{}, false, err
	}
	fields = domainWorkDeliveryPayloadFieldsV0(fields, artifactType, input.Task.Title, payloadBody)
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
	return orquestadomainwork.ExpectedDomainWorkArtifactTypeForWorkKindV0(workKind)
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

func readDomainWorkDeliveryBodyV0(
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
	ack orquestaruntimecodex.CodexAgentAckV0,
) string {
	for _, file := range ack.Files {
		path, ok := safeDomainWorkDeliveryFilePathV0(descriptor.ProjectWorkDir, string(file))
		if !ok {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		return strings.TrimSpace(string(data))
	}
	return ""
}

func safeDomainWorkDeliveryFilePathV0(baseDir string, rel string) (string, bool) {
	baseDir = strings.TrimSpace(baseDir)
	rel = filepath.ToSlash(filepath.Clean(strings.TrimSpace(rel)))
	if baseDir == "" || rel == "" || rel == "." || rel == ".." || strings.HasPrefix(rel, "../") ||
		filepath.IsAbs(rel) {
		return "", false
	}
	path := filepath.Join(baseDir, filepath.FromSlash(rel))
	cleanBase, err := filepath.Abs(baseDir)
	if err != nil {
		return "", false
	}
	cleanPath, err := filepath.Abs(path)
	if err != nil {
		return "", false
	}
	if cleanPath != cleanBase && !strings.HasPrefix(cleanPath, cleanBase+string(filepath.Separator)) {
		return "", false
	}
	return cleanPath, true
}
