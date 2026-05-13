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

const (
	externalWorkContextMaxFieldBytesV0 = 1800
	externalWorkContextMaxFieldsV0     = 24
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
	limit := len(work.InputFields)
	if limit > externalWorkContextMaxFieldsV0 {
		limit = externalWorkContextMaxFieldsV0
	}
	entries := make([]orquestacontext.ContextMaterializedEntryV0, 0, limit)
	for index, field := range work.InputFields[:limit] {
		content := externalWorkFieldContentV0(field)
		if content == "" {
			continue
		}
		content, truncated := boundedExternalWorkContextContentV0(content)
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
	}
	return entries
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
	raw, err := json.Marshal(field)
	if err != nil {
		return ""
	}
	return sanitizeExternalWorkContextContentV0(string(raw))
}

func boundedExternalWorkContextContentV0(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if len(value) <= externalWorkContextMaxFieldBytesV0 {
		return value, false
	}
	return value[:externalWorkContextMaxFieldBytesV0], true
}

func sanitizeExternalWorkContextContentV0(value string) string {
	replacer := strings.NewReplacer(
		"://", "_url_",
		"/home/", "/home-redacted/",
		"/Users/", "/users-redacted/",
		"/users/", "/users-redacted/",
		"$HOME", "HOME_REF",
		"~/", "HOME_REF/",
	)
	return replacer.Replace(strings.TrimSpace(value))
}

func externalContextRefPartV0(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.NewReplacer(
		" ", "-",
		"\t", "-",
		"\n", "-",
		"\r", "-",
		"/", "-",
		"\\", "-",
	).Replace(value)
	value = strings.Trim(value, "-")
	if value == "" {
		return "field"
	}
	return value
}
