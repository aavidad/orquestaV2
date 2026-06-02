package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestaappchangedirectorsource "orquesta/modulos/orquesta-app-change-director-source"
	orquestacontext "orquesta/modulos/orquesta-context"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func (resolver CodexLaunchSpecResolverV0) agentContextV0(
	ctx context.Context,
	payload orquestaruntime.LaunchRuntimeAgentRequestV0,
	area string,
	task orquestaruntime.AgentStartTaskV0,
) (orquestacontext.ContextMaterializedBundleV0, error) {
	bundle := contextBundleV0(area, task.TaskRef)
	if !isProgrammingPhaseV0(payload.PhaseID) || resolver.AppChangeStore == nil {
		return bundle, nil
	}
	work, ok, err := resolver.externalWorkForTaskV0(ctx, payload.RunID, task.TaskRef)
	if err != nil || !ok {
		return bundle, err
	}
	entries := externalWorkContextEntriesV0(area, task.TaskRef, work)
	bundle.Entries = append(bundle.Entries, entries...)
	for _, entry := range entries {
		bundle.TotalBytes += entry.Bytes
	}
	return bundle, nil
}

func (resolver CodexLaunchSpecResolverV0) externalWorkForTaskV0(
	ctx context.Context,
	runRef string,
	taskRef string,
) (*orquestaappchange.AppChangeExternalWorkV0, bool, error) {
	records, err := resolver.AppChangeStore.ListAppChangeRecordsV0(
		ctx,
		orquestaappchange.AppChangeRecordFilterV0{RunRef: strings.TrimSpace(runRef)},
	)
	if err != nil {
		return nil, false, err
	}
	for _, record := range records {
		if orquestaappchangedirectorsource.AppChangeTaskRefV0(record.Request.ChangeRef) != strings.TrimSpace(taskRef) ||
			record.Request.ExternalWork == nil {
			continue
		}
		work := cloneExternalWorkForContextV0(record.Request.ExternalWork)
		return &work, true, nil
	}
	return nil, false, nil
}

func cloneExternalWorkForContextV0(
	work *orquestaappchange.AppChangeExternalWorkV0,
) orquestaappchange.AppChangeExternalWorkV0 {
	if work == nil {
		return orquestaappchange.AppChangeExternalWorkV0{}
	}
	out := *work
	out.InterfaceRefs = append([]string(nil), work.InterfaceRefs...)
	out.WorkRefs = append([]string(nil), work.WorkRefs...)
	out.InputFields = copyDomainWorkFieldsForContextV0(work.InputFields)
	return out
}

func copyDomainWorkFieldsForContextV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
) []orquestadomainwork.DomainWorkFieldV0 {
	out := make([]orquestadomainwork.DomainWorkFieldV0, 0, len(fields))
	for _, field := range fields {
		out = append(out, orquestadomainwork.DomainWorkFieldV0{
			Name:      strings.TrimSpace(field.Name),
			Value:     strings.TrimSpace(field.Value),
			Values:    compactCodexStackStringsV0(field.Values),
			ValueJSON: append([]byte(nil), field.ValueJSON...),
		})
	}
	return out
}

func externalWorkContextEntriesV0(
	area string,
	taskRef string,
	work *orquestaappchange.AppChangeExternalWorkV0,
) []orquestacontext.ContextMaterializedEntryV0 {
	if work == nil || len(work.InputFields) == 0 {
		return nil
	}
	policy := externalWorkContextPolicyForWorkV0(work)
	fields := externalWorkPrioritizedContextFieldsV0(work.InputFields)
	limit := len(fields)
	if limit > policy.MaxFields {
		limit = policy.MaxFields
	}
	entries := make([]orquestacontext.ContextMaterializedEntryV0, 0, limit+1)
	remaining := policy.TotalMaxBytes
	omitted := len(fields) - limit
	for index, field := range fields[:limit] {
		if remaining <= 0 {
			omitted += limit - index
			break
		}
		content := externalWorkFieldContentV0(field)
		if content == "" {
			continue
		}
		content, truncated := boundedExternalWorkContextContentV0(content, minIntV0(policy.FieldMaxBytes, remaining))
		name := externalContextRefPartV0(field.Name)
		refSuffix := packetRefSuffixV0(taskRef, area, name, fmt.Sprintf("%02d", index+1))
		entries = append(entries, orquestacontext.ContextMaterializedEntryV0{
			EntryRef:  "entry-ref-app-stack-external-" + refSuffix,
			Layer:     orquestacontext.ContextLayerTaskContextV0,
			Kind:      orquestacontext.ContextEntryDocRefV0,
			SourceRef: "source-ref-app-stack-external-" + refSuffix,
			Mode:      orquestacontext.ContextMaterializationModeContentV0,
			Content:   content,
			Bytes:     len(content),
			Truncated: truncated,
			Required:  true,
		})
		remaining -= len(content)
	}
	if omitted > 0 {
		entries = append(entries, externalWorkContextBudgetEntryV0(area, taskRef, omitted))
	}
	return entries
}

func externalWorkPrioritizedContextFieldsV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
) []orquestadomainwork.DomainWorkFieldV0 {
	if len(fields) == 0 {
		return nil
	}
	priority := make([]orquestadomainwork.DomainWorkFieldV0, 0, len(fields))
	rest := make([]orquestadomainwork.DomainWorkFieldV0, 0, len(fields))
	for _, field := range fields {
		if externalWorkFieldIsCriticalContextV0(field.Name) {
			priority = append(priority, field)
			continue
		}
		rest = append(rest, field)
	}
	return append(priority, rest...)
}

func externalWorkFieldIsCriticalContextV0(name string) bool {
	switch normalizedExternalWorkFieldNameV0(name) {
	case "job_id",
		"job_ref",
		"job_type",
		"work_kind",
		"expected_artifact_type",
		"artifact_type",
		"title",
		"topic_title",
		"planned_title",
		"objective",
		"language_code",
		"level",
		"source_refs",
		"official_topic_ref",
		"topic_ref",
		"official_topic_text",
		"official_text",
		"topic_official_text",
		"topic_text",
		"source_text",
		"document_plan",
		"target_words_min",
		"target_words_max",
		"audio_roman_numeral_policy":
		return true
	default:
		return false
	}
}

func normalizedExternalWorkFieldNameV0(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.NewReplacer("-", "_", " ", "_", ".", "_").Replace(name)
	for strings.Contains(name, "__") {
		name = strings.ReplaceAll(name, "__", "_")
	}
	return strings.Trim(name, "_")
}

func externalWorkFieldContentV0(
	field orquestadomainwork.DomainWorkFieldV0,
) string {
	field.Name = strings.TrimSpace(field.Name)
	field.Value = strings.TrimSpace(field.Value)
	field.Values = compactCodexStackStringsV0(field.Values)
	if field.Name == "" ||
		(field.Value == "" && len(field.Values) == 0 && len(field.ValueJSON) == 0) {
		return ""
	}
	field = redactExternalWorkSensitiveFieldV0(field)
	raw, err := json.Marshal(field)
	if err != nil {
		return ""
	}
	return sanitizeExternalWorkContextContentV0(string(raw))
}
