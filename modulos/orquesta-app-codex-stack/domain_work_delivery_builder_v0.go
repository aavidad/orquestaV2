package orquestaappcodexstack

import (
	"context"
	"os"
	"path/filepath"
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
	fields = domainWorkDeliveryPayloadFieldsV0(fields, artifactType, input.Task.Title, body)
	return orquestadomainwork.NormalizeDomainWorkArtifactSubmissionV0(
		orquestadomainwork.DomainWorkArtifactSubmissionV0{
			RequestID:      "req-domain-work-artifact-" + input.Observation.DeliveryRef,
			CorrelationID:  input.Record.Request.CorrelationID,
			IdempotencyKey: "idem-domain-work-artifact-" + input.Observation.DeliveryRef,
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
	switch strings.TrimSpace(workKind) {
	case "draft_content_block", "generate_block", "generate_program_topic_draft":
		return "content_block"
	case "generate_visual_asset":
		return "visual_asset"
	case "review_legal", "review_pedagogical", "review_quality", "validate_topic":
		return "block_revision"
	case "research_sources", "download_source", "verify_sources":
		return "source"
	default:
		return "work_delivery"
	}
}

func domainWorkDeliveryPayloadFieldsV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
	artifactType string,
	title string,
	body string,
) []orquestadomainwork.DomainWorkFieldV0 {
	fields = appendDomainWorkFieldIfMissingV0(
		fields,
		"content_type",
		domainWorkDeliveryContentTypeV0(fields, artifactType),
	)
	fields = appendDomainWorkFieldIfMissingV0(fields, "title", title)
	if strings.TrimSpace(body) != "" {
		fields = append(fields, orquestadomainwork.DomainWorkFieldV0{Name: "body", Value: strings.TrimSpace(body)})
	}
	return fields
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
