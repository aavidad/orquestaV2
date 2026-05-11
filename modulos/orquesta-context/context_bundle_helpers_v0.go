package orquestacontext

import "strings"

const (
	defaultContextBundleMaxEntriesV0    = 18
	defaultContextBundleMaxTotalBytesV0 = 24000
	defaultContextBundleEntryBytesV0    = 3000
)

func trimContextV0(value string) string {
	return strings.TrimSpace(value)
}

func compactContextStringsV0(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = trimContextV0(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func normalizeContextBundleRequestV0(request ContextBundleRequestV0) ContextBundleRequestV0 {
	request.SchemaVersion = trimContextV0(request.SchemaVersion)
	request.BundleRef = trimContextV0(request.BundleRef)
	request.WorkOrderRef = trimContextV0(request.WorkOrderRef)
	request.TargetModule = trimContextV0(request.TargetModule)
	request.Phase = trimContextV0(request.Phase)
	request.TaskKind = trimContextV0(request.TaskKind)
	request.Objective = trimContextV0(request.Objective)
	request.CapacityLevel = trimContextV0(request.CapacityLevel)
	request.ReadSet = compactContextStringsV0(request.ReadSet)
	request.WriteSet = compactContextStringsV0(request.WriteSet)
	request.ContractRefs = compactContextStringsV0(request.ContractRefs)
	request.CrossModuleRefs = compactContextStringsV0(request.CrossModuleRefs)
	request.EvidenceRefs = compactContextStringsV0(request.EvidenceRefs)
	if request.MaxEntries == 0 {
		request.MaxEntries = defaultContextBundleMaxEntriesV0
	}
	if request.MaxTotalBytes == 0 {
		request.MaxTotalBytes = defaultContextBundleMaxTotalBytesV0
	}
	return request
}

func contextBundleIssueV0(code ContextBundleIssueCodeV0, field string, message string) ContextBundleIssueV0 {
	return ContextBundleIssueV0{
		Code:    code,
		Field:   field,
		Message: message,
	}
}

func contextBundleEntryV0(
	index int,
	layer string,
	kind string,
	sourceRef string,
	reason string,
	required bool,
) ContextBundleEntryV0 {
	return ContextBundleEntryV0{
		EntryRef:  "context-entry-" + contextIntV0(index),
		Layer:     layer,
		Kind:      kind,
		SourceRef: sourceRef,
		Reason:    reason,
		Required:  required,
		MaxBytes:  defaultContextBundleEntryBytesV0,
	}
}

func contextIntV0(value int) string {
	if value < 10 {
		return "00" + string(rune('0'+value))
	}
	if value < 100 {
		return "0" + string(rune('0'+value/10)) + string(rune('0'+value%10))
	}
	return "999"
}
