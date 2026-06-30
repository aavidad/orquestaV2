package orquestamcp

import (
	"encoding/json"
	"strconv"
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

type MCPGoalDomainOperationalMetadataV0 struct {
	CurrentPhase                    string
	RetryFromPhase                  string
	OperationalReason               string
	DomainCounters                  map[string]int
	CloseSupersededByLocalEvidence  bool
	SupersededByLocalEvidenceFlag   bool
	CloseSupersededEvidenceObserved bool
}

type mcpGoalDomainOperationalMetadataV0 = MCPGoalDomainOperationalMetadataV0

func GoalWorkStateDomainOperationalMetadataV0(
	state orquestagoal.GoalWorkStateV0,
) MCPGoalDomainOperationalMetadataV0 {
	return mcpGoalWorkStateDomainOperationalMetadataV0(state)
}

func mcpGoalWorkStateDomainOperationalMetadataV0(
	state orquestagoal.GoalWorkStateV0,
) mcpGoalDomainOperationalMetadataV0 {
	spec := orquestagoal.NormalizeGoalWorkSpecV0(state.Spec)
	metadata := mcpGoalDomainOperationalMetadataV0{}
	for _, value := range append(
		append([]string(nil), spec.EvidenceRefs...),
		append(state.EvidenceRefs, mcpGoalResultAndClosureEvidenceRefsV0(state)...)...,
	) {
		metadata = mcpGoalDomainOperationalMetadataFromTextV0(metadata, value)
	}
	for _, ref := range spec.ContextRefs {
		metadata = mcpGoalDomainOperationalMetadataFromTextV0(metadata, ref.Ref)
		metadata = mcpGoalDomainOperationalMetadataFromTextV0(metadata, ref.Purpose)
		metadata = mcpGoalDomainOperationalMetadataFromInputFieldPurposeV0(metadata, ref.Purpose)
	}
	for _, criterion := range spec.AcceptanceCriteria {
		metadata = mcpGoalDomainOperationalMetadataFromTextV0(metadata, criterion)
	}
	return finalizeMCPGoalDomainOperationalMetadataV0(metadata)
}

func mcpGoalResultAndClosureEvidenceRefsV0(state orquestagoal.GoalWorkStateV0) []string {
	var refs []string
	if state.LastResult != nil {
		refs = append(refs, state.LastResult.EvidenceRefs...)
		refs = append(refs, state.LastResult.DomainReceiptRefs...)
		refs = append(refs, state.LastResult.ArtifactRefs...)
	}
	if state.LastClosure != nil {
		refs = append(refs, state.LastClosure.EvidenceRefs...)
	}
	return compactStringsMCPV0(refs)
}

func mcpGoalDomainOperationalMetadataFromTextV0(
	metadata mcpGoalDomainOperationalMetadataV0,
	value string,
) mcpGoalDomainOperationalMetadataV0 {
	for _, token := range strings.Fields(strings.ReplaceAll(strings.TrimSpace(value), ",", " ")) {
		key, raw, ok := strings.Cut(token, "=")
		if !ok {
			key, raw, ok = strings.Cut(token, ":")
		}
		if !ok {
			if strings.TrimSpace(token) == "superseded_by_local_evidence" ||
				strings.Contains(strings.TrimSpace(token), "superseded-by-local-evidence") {
				metadata.SupersededByLocalEvidenceFlag = true
			}
			metadata = mcpGoalDomainOperationalEvidenceFromTextV0(metadata, token)
			continue
		}
		metadata = mcpGoalDomainOperationalMetadataApplyPairV0(metadata, key, raw)
	}
	if strings.Contains(value, "superseded_by_local_evidence") ||
		strings.Contains(value, "superseded-by-local-evidence") {
		metadata.SupersededByLocalEvidenceFlag = true
	}
	metadata = mcpGoalDomainOperationalEvidenceFromTextV0(metadata, value)
	return metadata
}

func mcpGoalDomainOperationalMetadataFromInputFieldPurposeV0(
	metadata mcpGoalDomainOperationalMetadataV0,
	purpose string,
) mcpGoalDomainOperationalMetadataV0 {
	const marker = " inlineado de forma acotada: "
	_, raw, ok := strings.Cut(strings.TrimSpace(purpose), marker)
	if !ok {
		return metadata
	}
	var summary map[string]any
	if err := json.Unmarshal([]byte(raw), &summary); err != nil {
		return metadata
	}
	name, _ := summary["name"].(string)
	if value, ok := summary["value"].(string); ok {
		metadata = mcpGoalDomainOperationalMetadataApplyPairV0(metadata, name, value)
	}
	if values, ok := summary["values"].([]any); ok {
		for _, value := range values {
			if text, ok := value.(string); ok {
				metadata = mcpGoalDomainOperationalMetadataApplyPairV0(metadata, name, text)
			}
		}
	}
	if valueJSON, ok := summary["value_json"]; ok {
		metadata = mcpGoalDomainOperationalMetadataApplyJSONValueV0(metadata, name, valueJSON)
	}
	return metadata
}

func mcpGoalDomainOperationalMetadataApplyPairV0(
	metadata mcpGoalDomainOperationalMetadataV0,
	key string,
	raw string,
) mcpGoalDomainOperationalMetadataV0 {
	key = strings.ToLower(strings.TrimSpace(key))
	raw = strings.Trim(strings.TrimSpace(raw), `"'`)
	switch key {
	case "current_phase", "opes_current_phase", "audio_current_phase":
		metadata.CurrentPhase = firstNonEmptyMCPV0(metadata.CurrentPhase, raw)
	case "retry_from_phase", "recommended_retry_phase":
		metadata.RetryFromPhase = firstNonEmptyMCPV0(metadata.RetryFromPhase, raw)
	case "operational_reason", "provider_reason", "provider_status", "stop_reason":
		metadata.OperationalReason = firstNonEmptyMCPV0(metadata.OperationalReason, raw)
	case "provider_timeout", "running_no_recent_progress":
		if mcpGoalDomainOperationalTruthyV0(raw) {
			metadata.CurrentPhase = firstNonEmptyMCPV0(metadata.CurrentPhase, "tts")
			metadata.RetryFromPhase = firstNonEmptyMCPV0(metadata.RetryFromPhase, "tts")
			metadata.OperationalReason = firstNonEmptyMCPV0(metadata.OperationalReason, key)
		}
	case "superseded_by_local_evidence":
		metadata.SupersededByLocalEvidenceFlag = mcpGoalDomainOperationalTruthyV0(raw)
	case "superseded_by_local_evidence_ref", "local_evidence_ref", "local_artifact_ref":
		if strings.TrimSpace(raw) != "" {
			metadata.CloseSupersededEvidenceObserved = true
		}
	case "audio_counters", "domain_counters":
		metadata.DomainCounters = mcpGoalDomainCountersFromStringV0(metadata.DomainCounters, raw)
	default:
		if strings.HasPrefix(key, "audio_counter_") {
			metadata.DomainCounters = mcpGoalDomainCounterSetV0(metadata.DomainCounters, strings.TrimPrefix(key, "audio_counter_"), raw)
		}
	}
	return metadata
}

func mcpGoalDomainOperationalMetadataApplyJSONValueV0(
	metadata mcpGoalDomainOperationalMetadataV0,
	name string,
	value any,
) mcpGoalDomainOperationalMetadataV0 {
	switch typed := value.(type) {
	case map[string]any:
		if strings.EqualFold(strings.TrimSpace(name), "audio_counters") ||
			strings.EqualFold(strings.TrimSpace(name), "domain_counters") {
			for key, raw := range typed {
				metadata.DomainCounters = mcpGoalDomainCounterSetV0(metadata.DomainCounters, key, raw)
			}
			return metadata
		}
		for key, raw := range typed {
			metadata = mcpGoalDomainOperationalMetadataApplyPairV0(metadata, key, mcpGoalDomainOperationalScalarV0(raw))
		}
	case string:
		metadata = mcpGoalDomainOperationalMetadataApplyPairV0(metadata, name, typed)
	case float64:
		metadata.DomainCounters = mcpGoalDomainCounterSetV0(metadata.DomainCounters, name, typed)
	case bool:
		metadata = mcpGoalDomainOperationalMetadataApplyPairV0(metadata, name, strconv.FormatBool(typed))
	}
	return metadata
}

func mcpGoalDomainOperationalEvidenceFromTextV0(
	metadata mcpGoalDomainOperationalMetadataV0,
	value string,
) mcpGoalDomainOperationalMetadataV0 {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return metadata
	}
	if (strings.Contains(value, "evidence-ref-") ||
		strings.Contains(value, "domain-receipt-ref-") ||
		strings.Contains(value, "artifact-ref-")) &&
		(strings.Contains(value, "current") || strings.Contains(value, "local-evidence") || strings.Contains(value, "local_evidence")) {
		metadata.CloseSupersededEvidenceObserved = true
	}
	return metadata
}

func finalizeMCPGoalDomainOperationalMetadataV0(
	metadata mcpGoalDomainOperationalMetadataV0,
) mcpGoalDomainOperationalMetadataV0 {
	metadata.CloseSupersededByLocalEvidence =
		metadata.SupersededByLocalEvidenceFlag &&
			metadata.CloseSupersededEvidenceObserved
	return metadata
}

func mcpGoalDomainCountersFromStringV0(counters map[string]int, raw string) map[string]int {
	var decoded map[string]int
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &decoded); err == nil {
		return mergeMCPDomainOperationalCountersV0(counters, decoded)
	}
	for _, token := range strings.Fields(strings.ReplaceAll(raw, ",", " ")) {
		key, value, ok := strings.Cut(token, "=")
		if !ok {
			key, value, ok = strings.Cut(token, ":")
		}
		if ok {
			counters = mcpGoalDomainCounterSetV0(counters, key, value)
		}
	}
	return counters
}

func mcpGoalDomainCounterSetV0(counters map[string]int, key string, raw any) map[string]int {
	key = strings.TrimSpace(key)
	if key == "" {
		return counters
	}
	value := 0
	switch typed := raw.(type) {
	case int:
		value = typed
	case float64:
		value = int(typed)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err == nil {
			value = parsed
		}
	}
	if value == 0 {
		return counters
	}
	if counters == nil {
		counters = map[string]int{}
	}
	counters[key] = value
	return counters
}

func mergeMCPDomainOperationalCountersV0(values ...map[string]int) map[string]int {
	out := map[string]int{}
	for _, counters := range values {
		for key, value := range counters {
			key = strings.TrimSpace(key)
			if key == "" || value == 0 {
				continue
			}
			out[key] = value
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func mcpGoalDomainOperationalTruthyV0(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "si", "sí", "provider_timeout", "running_no_recent_progress", "superseded_by_local_evidence":
		return true
	default:
		return false
	}
}

func mcpGoalDomainOperationalScalarV0(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case bool:
		return strconv.FormatBool(typed)
	case float64:
		return strconv.Itoa(int(typed))
	case int:
		return strconv.Itoa(typed)
	default:
		return ""
	}
}
