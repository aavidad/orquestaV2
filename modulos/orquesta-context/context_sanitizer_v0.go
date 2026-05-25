package orquestacontext

import "strings"

const contextSanitizationReviewHintV0 = "CONSULTA_AL_DIRECTOR: context_sanitization_review_required"

func SanitizeMaterializedContextBundleV0(
	bundle ContextMaterializedBundleV0,
	sanitizer ContextSanitizerPortV0,
) ContextMaterializedBundleV0 {
	if sanitizer == nil || len(bundle.Entries) == 0 {
		return bundle
	}
	out := bundle
	out.Entries = make([]ContextMaterializedEntryV0, 0, len(bundle.Entries))
	out.SanitizationEvidence = append([]ContextSanitizationEvidenceV0(nil), bundle.SanitizationEvidence...)
	out.TotalBytes = 0
	for _, entry := range bundle.Entries {
		sanitized, evidence, issues := sanitizeContextMaterializedEntryV0(bundle, entry, sanitizer)
		if len(issues) > 0 {
			out.Issues = append(out.Issues, issues...)
			continue
		}
		if evidence.SchemaVersion != "" {
			out.SanitizationEvidence = append(out.SanitizationEvidence, evidence)
		}
		if sanitized.Required && evidence.ReviewRequired {
			out.DirectorQuestionHint = appendContextSanitizationHintV0(out.DirectorQuestionHint)
		}
		out.TotalBytes += sanitized.Bytes
		out.Entries = append(out.Entries, sanitized)
	}
	return out
}

func ContextBundleRequiresSanitizationReviewV0(bundle ContextMaterializedBundleV0) bool {
	for _, evidence := range bundle.SanitizationEvidence {
		if evidence.ReviewRequired || evidence.Status == ContextSanitizationStatusReviewRequiredV0 {
			return true
		}
	}
	return false
}

func sanitizeContextMaterializedEntryV0(
	bundle ContextMaterializedBundleV0,
	entry ContextMaterializedEntryV0,
	sanitizer ContextSanitizerPortV0,
) (ContextMaterializedEntryV0, ContextSanitizationEvidenceV0, []ContextMaterializationIssueV0) {
	if entry.Mode != ContextMaterializationModeContentV0 || entry.Content == "" {
		return entry, ContextSanitizationEvidenceV0{}, nil
	}
	result := sanitizer.SanitizeContextEntryV0(ContextSanitizationRequestV0{
		BundleRef:    bundle.BundleRef,
		WorkOrderRef: bundle.WorkOrderRef,
		TargetModule: bundle.TargetModule,
		EntryRef:     entry.EntryRef,
		SourceRef:    entry.SourceRef,
		Content:      entry.Content,
		Bytes:        entry.Bytes,
	})
	if len(result.Issues) > 0 {
		return ContextMaterializedEntryV0{}, ContextSanitizationEvidenceV0{}, result.Issues
	}
	evidence := normalizeContextSanitizationEvidenceV0(bundle, entry, result)
	sanitized := entry
	sanitized.EntryRef = contextSanitizationSafeRefValueV0(sanitized.EntryRef, "entry-ref", entry.SourceRef)
	sanitized.SourceRef = contextSanitizationSafeRefValueV0(sanitized.SourceRef, "source-ref", entry.EntryRef)
	switch evidence.Status {
	case ContextSanitizationStatusReviewRequiredV0:
		sanitized = contextMaterializedRefOnlyV0(contextEntryFromMaterializedV0(sanitized))
		sanitized.Truncated = true
		sanitized.RefOnlyReason = ContextRefOnlyReasonSanitizationReviewV0
		sanitized.RequiredRefAction = ContextRequiredRefActionAskDirectorV0
	case ContextSanitizationStatusSanitizedV0, ContextSanitizationStatusCleanV0:
		sanitized.Content = firstContextSanitizedContentV0(result.Content, entry.Content)
		sanitized.Bytes = len(sanitized.Content)
		sanitized.Truncated = entry.Truncated
	default:
		return ContextMaterializedEntryV0{}, ContextSanitizationEvidenceV0{}, []ContextMaterializationIssueV0{
			contextMaterializationIssueV0(ErrContextMaterializationDetalleProhibidoV0, entry.SourceRef, "sanitizer status invalido"),
		}
	}
	if sanitized.Mode == ContextMaterializationModeContentV0 &&
		contextMaterializedContentHasForbiddenDetailV0(sanitized.Content) {
		evidence.Status = ContextSanitizationStatusReviewRequiredV0
		evidence.ReviewRequired = true
		sanitized = contextMaterializedRefOnlyV0(contextEntryFromMaterializedV0(sanitized))
		sanitized.Truncated = true
	}
	return sanitized, evidence, nil
}

func normalizeContextSanitizationEvidenceV0(
	bundle ContextMaterializedBundleV0,
	entry ContextMaterializedEntryV0,
	result ContextSanitizationResultV0,
) ContextSanitizationEvidenceV0 {
	status := result.Status
	if status == "" {
		status = ContextSanitizationStatusCleanV0
	}
	evidence := result.Evidence
	evidence.SchemaVersion = firstContextValueV0(evidence.SchemaVersion, ContextSanitizationEvidenceSchemaVersionV0)
	evidence.EvidenceRef = firstContextValueV0(evidence.EvidenceRef, "context-sanitization-evidence-"+entry.EntryRef)
	evidence.BundleRef = firstContextValueV0(evidence.BundleRef, bundle.BundleRef)
	evidence.WorkOrderRef = firstContextValueV0(evidence.WorkOrderRef, bundle.WorkOrderRef)
	evidence.TargetModule = firstContextValueV0(evidence.TargetModule, bundle.TargetModule)
	evidence.EntryRef = firstContextValueV0(evidence.EntryRef, entry.EntryRef)
	evidence.SourceRef = firstContextValueV0(evidence.SourceRef, entry.SourceRef)
	evidence.SanitizerRef = firstContextValueV0(evidence.SanitizerRef, "context-sanitizer-ref-local")
	evidence.Status = status
	evidence.ReviewRequired = evidence.ReviewRequired || status == ContextSanitizationStatusReviewRequiredV0
	evidence.EvidenceRef = contextSanitizationSafeRefValueV0(
		evidence.EvidenceRef,
		"context-sanitization-evidence",
		entry.EntryRef,
	)
	evidence.EntryRef = contextSanitizationSafeRefValueV0(evidence.EntryRef, "entry-ref", entry.SourceRef)
	evidence.SourceRef = contextSanitizationSafeRefValueV0(evidence.SourceRef, "source-ref", entry.EntryRef)
	return evidence
}

func contextEntryFromMaterializedV0(entry ContextMaterializedEntryV0) ContextBundleEntryV0 {
	return ContextBundleEntryV0{
		EntryRef:  entry.EntryRef,
		Layer:     entry.Layer,
		Kind:      entry.Kind,
		SourceRef: entry.SourceRef,
		Required:  entry.Required,
	}
}

func appendContextSanitizationHintV0(current string) string {
	if strings.Contains(current, contextSanitizationReviewHintV0) {
		return current
	}
	if strings.TrimSpace(current) == "" {
		return contextSanitizationReviewHintV0
	}
	return strings.TrimSpace(current) + " " + contextSanitizationReviewHintV0
}

func firstContextSanitizedContentV0(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func firstContextValueV0(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func contextSanitizationSafeRefValueV0(value string, prefix string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" || !contextSanitizationRefHasForbiddenDetailV0(value) {
		return value
	}
	return strings.Trim(prefix, "-") + "-" + contextSanitizationSafeRefPartV0(fallback)
}

func contextSanitizationRefHasForbiddenDetailV0(value string) bool {
	return contextMaterializedRefHasForbiddenDetailV0(value)
}

func contextSanitizationSafeRefPartV0(value string) string {
	if contextSanitizationRefHasForbiddenDetailV0(value) {
		return "context-" + contextSanitizationStableHashV0(value)
	}
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		allowed := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if allowed {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(builder.String(), "-")
	if out == "" {
		return "context"
	}
	return out
}

func contextSanitizationStableHashV0(value string) string {
	var hash uint32 = 2166136261
	for _, b := range []byte(value) {
		hash ^= uint32(b)
		hash *= 16777619
	}
	const alphabet = "0123456789abcdef"
	out := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		out[i] = alphabet[hash&0xf]
		hash >>= 4
	}
	return string(out)
}
