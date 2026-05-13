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
	fields = append(fields,
		orquestadomainwork.DomainWorkFieldV0{Name: "content_type", Value: "text/markdown"},
		orquestadomainwork.DomainWorkFieldV0{Name: "title", Value: input.Task.Title},
	)
	if body != "" {
		fields = append(fields, orquestadomainwork.DomainWorkFieldV0{Name: "body", Value: body})
	}
	return orquestadomainwork.NormalizeDomainWorkArtifactSubmissionV0(
		orquestadomainwork.DomainWorkArtifactSubmissionV0{
			RequestID:      "req-domain-work-artifact-" + input.Observation.DeliveryRef,
			CorrelationID:  input.Record.Request.CorrelationID,
			IdempotencyKey: "idem-domain-work-artifact-" + input.Observation.DeliveryRef,
			RequestedBy:    "orquesta",
			DomainRef:      work.ProjectRef,
			JobRef:         work.JobRef,
			ArtifactRef:    input.Observation.DeliveryRef,
			ArtifactType:   domainWorkArtifactTypeForWorkKindV0(work.WorkKind),
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
	case "review_legal", "review_pedagogical", "review_quality", "validate_topic":
		return "block_revision"
	case "research_sources", "download_source", "verify_sources":
		return "source"
	default:
		return "work_delivery"
	}
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
