package orquestaopesbridge

import (
	"encoding/json"
	"strings"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

const (
	OPESAudioWorkflowPreconditionMissingErrorV0 = "opes_audio_workflow_precondition_missing"

	OPESAudioWorkflowTextPublicPassRequiredReasonV0 = "phase_precondition_missing:opes_audio:text_public_pass"
	OPESAudioWorkflowTextEncodingCorruptReasonV0    = "phase_precondition_missing:opes_audio:text_encoding_corrupted"
	OPESAudioWorkflowPrepareRefsRequiredReasonV0    = "phase_precondition_missing:opes_audio:prepare_refs_required"
	OPESAudioWorkflowPrepareStaleRequiredReasonV0   = "phase_precondition_missing:opes_audio:prepare_stale"
	OPESAudioWorkflowSelectiveRegenRequiredReasonV0 = "phase_precondition_missing:opes_audio:selective_regeneration_required"
)

type OPESAudioWorkflowPreconditionEvaluationV0 struct {
	Applies               bool           `json:"applies"`
	Ready                 bool           `json:"ready"`
	CurrentPhase          string         `json:"current_phase,omitempty"`
	OperationalReason     string         `json:"operational_reason,omitempty"`
	MissingPreconditions  []string       `json:"missing_preconditions,omitempty"`
	NextActions           []string       `json:"next_actions,omitempty"`
	RecommendedRetryPhase string         `json:"recommended_retry_phase,omitempty"`
	AudioCounters         map[string]int `json:"audio_counters,omitempty"`
}

func EvaluateOPESAudioWorkflowPreconditionsV0(
	request orquestadomainwork.DomainWorkJobRequestV0,
) OPESAudioWorkflowPreconditionEvaluationV0 {
	counters := opesAudioWorkflowCountersV0(request.InputFields)
	if !opesAudioWorkflowPreconditionAppliesV0(request) {
		return OPESAudioWorkflowPreconditionEvaluationV0{Ready: true}
	}
	if !opesAudioWorkflowTextPublicPassV0(request.InputFields) {
		return OPESAudioWorkflowPreconditionEvaluationV0{
			Applies:               true,
			Ready:                 false,
			CurrentPhase:          "text_qa",
			OperationalReason:     OPESAudioWorkflowTextPublicPassRequiredReasonV0,
			MissingPreconditions:  []string{"text_public_pass"},
			NextActions:           []string{"run_text_qa", "declare_text_public_pass", "retry_from_phase=text_qa"},
			RecommendedRetryPhase: "text_qa",
			AudioCounters:         counters,
		}
	}
	if opesAudioWorkflowHasMojibakeV0(request.InputFields) {
		return OPESAudioWorkflowPreconditionEvaluationV0{
			Applies:               true,
			Ready:                 false,
			CurrentPhase:          "text_qa",
			OperationalReason:     OPESAudioWorkflowTextEncodingCorruptReasonV0,
			MissingPreconditions:  []string{"text_encoding_clean"},
			NextActions:           []string{"run_text_encoding_qa", "repair_public_text_encoding", "regenerate_audio_prepare_artifacts", "retry_from_phase=text_qa"},
			RecommendedRetryPhase: "text_qa",
			AudioCounters:         counters,
		}
	}
	if !opesAudioWorkflowPrepareRefsReadyV0(request.InputFields) {
		return OPESAudioWorkflowPreconditionEvaluationV0{
			Applies:               true,
			Ready:                 false,
			CurrentPhase:          "prepare",
			OperationalReason:     OPESAudioWorkflowPrepareRefsRequiredReasonV0,
			MissingPreconditions:  []string{"audio_prepare_refs"},
			NextActions:           []string{"prepare_audio_sidecars", "declare_audio_manifest_ref", "declare_source_content_or_text_hash_ref", "retry_from_phase=prepare"},
			RecommendedRetryPhase: "prepare",
			AudioCounters:         counters,
		}
	}
	if !opesAudioWorkflowPrepareCurrentV0(request.InputFields) {
		return OPESAudioWorkflowPreconditionEvaluationV0{
			Applies:               true,
			Ready:                 false,
			CurrentPhase:          "prepare",
			OperationalReason:     OPESAudioWorkflowPrepareStaleRequiredReasonV0,
			MissingPreconditions:  []string{"audio_prepare_current"},
			NextActions:           []string{"invalidate_audio_sidecars", "prepare_audio_sidecars", "retry_from_phase=prepare"},
			RecommendedRetryPhase: "prepare",
			AudioCounters:         counters,
		}
	}
	if !opesAudioWorkflowSelectiveRegenerationV0(request.InputFields) {
		return OPESAudioWorkflowPreconditionEvaluationV0{
			Applies:               true,
			Ready:                 false,
			CurrentPhase:          "prepare",
			OperationalReason:     OPESAudioWorkflowSelectiveRegenRequiredReasonV0,
			MissingPreconditions:  []string{"selective_audio_regeneration"},
			NextActions:           []string{"prepare_audio_sidecars", "declare_audio_regeneration_mode_selective", "retry_from_phase=prepare"},
			RecommendedRetryPhase: "prepare",
			AudioCounters:         counters,
		}
	}
	return OPESAudioWorkflowPreconditionEvaluationV0{
		Applies:       true,
		Ready:         true,
		CurrentPhase:  "tts",
		AudioCounters: counters,
	}
}

func opesAudioWorkflowPreconditionAppliesV0(
	request orquestadomainwork.DomainWorkJobRequestV0,
) bool {
	if orquestadomainwork.ExpectedDomainWorkArtifactTypeForWorkKindV0(request.WorkKind) == orquestadomainwork.DomainWorkArtifactTypeAudioAssetV0 {
		return true
	}
	for _, value := range opesAudioWorkflowFieldValuesByNameV0(request.InputFields, "expected_artifact_type") {
		if strings.TrimSpace(value) == orquestadomainwork.DomainWorkArtifactTypeAudioAssetV0 {
			return true
		}
	}
	return false
}

func opesAudioWorkflowTextPublicPassV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
) bool {
	for _, value := range opesAudioWorkflowFieldValuesByNamesV0(fields,
		"text_public_pass",
		"text_public_status",
		"public_text_status",
		"text_qa_status",
		"textual_gate_status",
		"validated_text_status",
		"editorial_status",
	) {
		if opesAudioWorkflowPassValueV0(value) {
			return true
		}
	}
	return false
}

func opesAudioWorkflowHasMojibakeV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
) bool {
	for _, field := range fields {
		if !opesAudioWorkflowPotentialTextFieldV0(field.Name) {
			continue
		}
		for _, value := range opesAudioWorkflowFieldValuesV0(field) {
			if opesAudioWorkflowTextContainsMojibakeV0(value) {
				return true
			}
		}
	}
	return false
}

func opesAudioWorkflowPotentialTextFieldV0(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return false
	}
	for _, marker := range []string{
		"text",
		"markdown",
		"md",
		"html",
		"rag",
		"manifest",
		"content",
		"body",
		"title",
		"summary",
		"resumen",
		"ampliado",
		"public",
		"source",
	} {
		if name == marker || strings.Contains(name, marker+"_") || strings.Contains(name, "_"+marker) {
			return true
		}
	}
	return false
}

func opesAudioWorkflowTextContainsMojibakeV0(value string) bool {
	for _, marker := range []string{"mÃ", "Ã", "Â", "�", "â€"} {
		if strings.Contains(value, marker) {
			return true
		}
	}
	return false
}

func opesAudioWorkflowPrepareRefsReadyV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
) bool {
	hasManifestOrSidecar := len(opesAudioWorkflowFieldValuesByNamesV0(fields,
		"audio_manifest_ref",
		"audio_manifest_refs",
		"audio_sidecar_ref",
		"audio_sidecar_refs",
	)) > 0
	hasSourceOrHash := len(opesAudioWorkflowFieldValuesByNamesV0(fields,
		"source_content_ref",
		"text_hash_ref",
		"source_text_hash_ref",
		"html_text_hash_ref",
	)) > 0
	return hasManifestOrSidecar && hasSourceOrHash
}

func opesAudioWorkflowSelectiveRegenerationV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
) bool {
	for _, value := range opesAudioWorkflowFieldValuesByNamesV0(fields,
		"audio_regeneration_mode",
		"audio_generation_mode",
		"tts_regeneration_mode",
		"tts_mode",
		"regeneration_mode",
		"generate_audio_mode",
	) {
		if opesAudioWorkflowSelectiveRegenValueV0(value) {
			return true
		}
	}
	return false
}

func opesAudioWorkflowPrepareCurrentV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
) bool {
	for _, value := range opesAudioWorkflowFieldValuesByNamesV0(fields,
		"audio_prepare_status",
		"audio_prepare_phase_status",
		"audio_manifest_status",
		"audio_sidecar_status",
		"prepare_status",
	) {
		if opesAudioWorkflowStaleValueV0(value) {
			return false
		}
	}
	if opesAudioWorkflowAnyPairMismatchV0(
		opesAudioWorkflowFieldValuesByNamesV0(fields,
			"current_text_hash_ref",
			"current_text_hash",
			"current_html_text_hash_ref",
			"current_html_text_hash",
			"current_source_content_hash_ref",
			"current_source_content_hash",
			"html_text_hash_current",
		),
		opesAudioWorkflowFieldValuesByNamesV0(fields,
			"prepared_text_hash_ref",
			"prepared_text_hash",
			"audio_prepared_text_hash_ref",
			"audio_prepared_text_hash",
			"audio_sidecar_text_hash_ref",
			"audio_manifest_text_hash_ref",
			"prepared_source_content_hash_ref",
			"prepared_source_content_hash",
		),
	) {
		return false
	}
	if opesAudioWorkflowAnyPairMismatchV0(
		opesAudioWorkflowFieldValuesByNamesV0(fields,
			"current_source_content_ref",
			"current_html_content_ref",
		),
		opesAudioWorkflowFieldValuesByNamesV0(fields,
			"prepared_source_content_ref",
			"audio_prepared_source_content_ref",
			"audio_sidecar_source_content_ref",
		),
	) {
		return false
	}
	return true
}

func opesAudioWorkflowPassValueV0(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "yes", "y", "si", "sí", "1",
		"pass", "passed", "ok", "ready", "listo",
		"approved", "aprobado", "validated", "validado",
		"closed", "cerrado", "final", "publicable", "final_publicable":
		return true
	default:
		return false
	}
}

func opesAudioWorkflowStaleValueV0(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "stale", "obsolete", "obsoleto", "outdated", "dirty",
		"invalid", "invalidated", "invalidado", "superseded",
		"desactualizado", "changed", "text_changed", "html_changed":
		return true
	default:
		return false
	}
}

func opesAudioWorkflowSelectiveRegenValueV0(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "selective", "selectivo",
		"selective_by_sidecar", "selective_by_hash",
		"missing_or_stale_only", "missing_or_obsolete_only",
		"missing_only", "stale_only", "obsolete_only", "changed_only",
		"incremental", "reuse_existing", "reuse_and_regenerate_stale":
		return true
	default:
		return false
	}
}

func opesAudioWorkflowAnyPairMismatchV0(current []string, prepared []string) bool {
	current = compactStringsV0(current)
	prepared = compactStringsV0(prepared)
	if len(current) == 0 || len(prepared) == 0 {
		return false
	}
	for _, left := range current {
		for _, right := range prepared {
			if strings.TrimSpace(left) == strings.TrimSpace(right) {
				return false
			}
		}
	}
	return true
}

func opesAudioWorkflowCountersV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
) map[string]int {
	counters := map[string]int{
		"audio_manifest_refs": len(opesAudioWorkflowFieldValuesByNamesV0(fields,
			"audio_manifest_ref",
			"audio_manifest_refs",
		)),
		"audio_sidecar_refs": len(opesAudioWorkflowFieldValuesByNamesV0(fields,
			"audio_sidecar_ref",
			"audio_sidecar_refs",
		)),
		"source_content_refs": len(opesAudioWorkflowFieldValuesByNamesV0(fields,
			"source_content_ref",
			"current_source_content_ref",
			"prepared_source_content_ref",
		)),
		"text_hash_refs": len(opesAudioWorkflowFieldValuesByNamesV0(fields,
			"text_hash_ref",
			"source_text_hash_ref",
			"html_text_hash_ref",
			"current_text_hash_ref",
			"prepared_text_hash_ref",
			"audio_sidecar_text_hash_ref",
		)),
	}
	for key, value := range counters {
		if value == 0 {
			delete(counters, key)
		}
	}
	if len(counters) == 0 {
		return nil
	}
	return counters
}

func opesAudioWorkflowFieldValuesByNamesV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
	names ...string,
) []string {
	out := []string{}
	for _, name := range names {
		out = append(out, opesAudioWorkflowFieldValuesByNameV0(fields, name)...)
	}
	return compactStringsV0(out)
}

func opesAudioWorkflowFieldValuesByNameV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
	name string,
) []string {
	name = strings.TrimSpace(name)
	if name == "" {
		return []string{}
	}
	out := []string{}
	for _, field := range fields {
		if strings.TrimSpace(field.Name) != name {
			continue
		}
		out = append(out, opesAudioWorkflowFieldValuesV0(field)...)
	}
	return compactStringsV0(out)
}

func opesAudioWorkflowFieldValuesV0(field orquestadomainwork.DomainWorkFieldV0) []string {
	out := []string{field.Value}
	out = append(out, field.Values...)
	out = append(out, opesAudioWorkflowJSONValuesV0(field.ValueJSON)...)
	return compactStringsV0(out)
}

func opesAudioWorkflowJSONValuesV0(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return []string{}
	}
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return []string{text}
	}
	var texts []string
	if json.Unmarshal(raw, &texts) == nil {
		return texts
	}
	var flag bool
	if json.Unmarshal(raw, &flag) == nil {
		if flag {
			return []string{"true"}
		}
		return []string{"false"}
	}
	var value any
	if json.Unmarshal(raw, &value) == nil {
		return opesAudioWorkflowJSONScalarValuesV0(value)
	}
	return []string{}
}

func opesAudioWorkflowJSONScalarValuesV0(value any) []string {
	switch typed := value.(type) {
	case string:
		return []string{typed}
	case []any:
		out := []string{}
		for _, item := range typed {
			out = append(out, opesAudioWorkflowJSONScalarValuesV0(item)...)
		}
		return out
	case map[string]any:
		out := []string{}
		for _, item := range typed {
			out = append(out, opesAudioWorkflowJSONScalarValuesV0(item)...)
		}
		return out
	default:
		return []string{}
	}
}
