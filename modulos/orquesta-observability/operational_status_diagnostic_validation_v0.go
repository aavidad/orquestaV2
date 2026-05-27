package orquestaobservability

import (
	"fmt"
	"math"
	"strings"

	orquestarails "orquesta/modulos/orquesta-rails"
)

func validateDiagnosticoCompactoV0(diagnostic DiagnosticoCompactoV0, prefix string, add func(string, string)) {
	if strings.TrimSpace(diagnostic.SchemaVersion) != DiagnosticoCompactoSchemaVersionV0 {
		add(ErrOperationalStatusQueryInvalidaV0, fieldV0(prefix, "schema_version"))
	}
	validateRequiredOperationalRefV0(diagnostic.DiagnosticID, fieldV0(prefix, "diagnostic_id"), add)
	if !isOccurredAtV0(diagnostic.GeneratedAt) {
		add(ErrOperationalStatusQueryInvalidaV0, fieldV0(prefix, "generated_at"))
	}
	validateRequiredOperationalRefV0(diagnostic.CorrelationID, fieldV0(prefix, "correlation_id"), add)
	scope := strings.TrimSpace(diagnostic.Scope)
	if !allowedV0(allowedOperationalScopesV0, scope) {
		add(ErrScopeNoSoportadoV0, fieldV0(prefix, "scope"))
	}
	validateOptionalOperationalRefV0(diagnostic.SubjectRef, fieldV0(prefix, "subject_ref"), add)
	validateRequiredOperationalRefV0(diagnostic.ProjectionRef, fieldV0(prefix, "projection_ref"), add)
	validateDiagnosticoFreshnessV0(diagnostic.Freshness, fieldV0(prefix, "freshness"), add)
	if !allowedV0(allowedDiagnosticoEstadosV0, strings.TrimSpace(diagnostic.Estado)) {
		add(ErrOperationalStatusQueryInvalidaV0, fieldV0(prefix, "estado"))
	}
	validateDiagnosticoProgresoV0(diagnostic.Progreso, fieldV0(prefix, "progreso"), add)
	validateDiagnosticoSaludV0(diagnostic.Salud, fieldV0(prefix, "salud"), add)
	validateDiagnosticoBloqueosV0(diagnostic.Bloqueos, fieldV0(prefix, "bloqueos"), add)
	validateDiagnosticoActividadV0(diagnostic.ActividadReciente, fieldV0(prefix, "actividad_reciente"), add)
	validateDiagnosticoContadoresV0(diagnostic.Contadores, fieldV0(prefix, "contadores"), add)
	validateDiagnosticoReferenciasV0(diagnostic.Referencias, fieldV0(prefix, "referencias"), add)
	validateDiagnosticoWarningsV0(diagnostic.Warnings, fieldV0(prefix, "warnings"), add)
	validateDiagnosticoPrivacyV0(diagnostic.Privacy, fieldV0(prefix, "privacy"), add)
	validateOperationalJSONSizeV0(diagnostic, maxDiagnosticoJSONBytesV0, fieldV0(prefix, ""), add)
}

func validateDiagnosticoFreshnessV0(freshness DiagnosticoFreshnessV0, prefix string, add func(string, string)) {
	validateRequiredOperationalRefV0(freshness.WatermarkRef, fieldV0(prefix, "watermark_ref"), add)
	if freshness.MaxAgeSeconds < 0 || freshness.MaxAgeSeconds > maxOperationalFreshnessAgeSecondsV0 {
		add(ErrFrescuraNoGarantizadaV0, fieldV0(prefix, "max_age_seconds"))
	}
}

func validateDiagnosticoProgresoV0(progreso DiagnosticoProgresoV0, prefix string, add func(string, string)) {
	if progreso.Completed < 0 {
		add(ErrOperationalStatusQueryInvalidaV0, fieldV0(prefix, "completed"))
	}
	if progreso.Total < 0 {
		add(ErrOperationalStatusQueryInvalidaV0, fieldV0(prefix, "total"))
	}
	if progreso.Total > 0 && progreso.Completed > progreso.Total {
		add(ErrOperationalStatusQueryInvalidaV0, prefix)
	}
	if progreso.Percent != nil {
		if math.IsNaN(*progreso.Percent) || math.IsInf(*progreso.Percent, 0) || *progreso.Percent < 0 || *progreso.Percent > 100 {
			add(ErrOperationalStatusQueryInvalidaV0, fieldV0(prefix, "percent"))
		}
	}
	validateOperationalTextV0(progreso.Phase, fieldV0(prefix, "phase"), maxOperationalTokenRunesV0, false, add)
	validateOperationalTextV0(progreso.Summary, fieldV0(prefix, "summary"), maxOperationalTextRunesV0, false, add)
}

func validateDiagnosticoSaludV0(items []DiagnosticoSaludCheckV0, prefix string, add func(string, string)) {
	if len(items) > maxOperationalListItemsV0 {
		add(ErrConsultaDemasiadoAmpliaV0, prefix)
	}
	for index, item := range items {
		itemPrefix := fmt.Sprintf("%s[%d]", prefix, index)
		validateOperationalAreaV0(item.Area, fieldV0(itemPrefix, "area"), add)
		if !allowedV0(allowedSeveritiesV0, strings.TrimSpace(item.Severity)) {
			add(ErrOperationalStatusQueryInvalidaV0, fieldV0(itemPrefix, "severity"))
		}
		if !allowedV0(allowedDiagnosticoEstadosV0, strings.TrimSpace(item.Estado)) {
			add(ErrOperationalStatusQueryInvalidaV0, fieldV0(itemPrefix, "estado"))
		}
		validateOperationalI18nKeyV0(item.I18nKey, fieldV0(itemPrefix, "i18n_key"), add)
		validateOperationalRefListV0(item.EvidenceRefs, maxOperationalEvidenceRefsV0, fieldV0(itemPrefix, "evidence_refs"), add)
	}
}

func validateDiagnosticoBloqueosV0(items []DiagnosticoBloqueoV0, prefix string, add func(string, string)) {
	if len(items) > maxOperationalListItemsV0 {
		add(ErrConsultaDemasiadoAmpliaV0, prefix)
	}
	for index, item := range items {
		itemPrefix := fmt.Sprintf("%s[%d]", prefix, index)
		validateRequiredOperationalRefV0(item.BlockerRef, fieldV0(itemPrefix, "blocker_ref"), add)
		if !allowedV0(allowedSeveritiesV0, strings.TrimSpace(item.Severity)) {
			add(ErrOperationalStatusQueryInvalidaV0, fieldV0(itemPrefix, "severity"))
		}
		validateOperationalAreaV0(item.OwnerArea, fieldV0(itemPrefix, "owner_area"), add)
		validateOperationalTextV0(item.Summary, fieldV0(itemPrefix, "summary"), maxOperationalTextRunesV0, true, add)
		validateOperationalRefListV0(item.EvidenceRefs, maxOperationalEvidenceRefsV0, fieldV0(itemPrefix, "evidence_refs"), add)
	}
}

func validateDiagnosticoActividadV0(items []DiagnosticoActividadV0, prefix string, add func(string, string)) {
	if len(items) > maxOperationalListItemsV0 {
		add(ErrConsultaDemasiadoAmpliaV0, prefix)
	}
	for index, item := range items {
		itemPrefix := fmt.Sprintf("%s[%d]", prefix, index)
		validateRequiredOperationalRefV0(item.ActivityRef, fieldV0(itemPrefix, "activity_ref"), add)
		if !isOccurredAtV0(item.OccurredAt) {
			add(ErrOperationalStatusQueryInvalidaV0, fieldV0(itemPrefix, "occurred_at"))
		}
		validateOperationalAreaV0(item.Area, fieldV0(itemPrefix, "area"), add)
		validateOperationalTextV0(item.Summary, fieldV0(itemPrefix, "summary"), maxOperationalTextRunesV0, true, add)
		validateOptionalOperationalRefV0(item.EventRef, fieldV0(itemPrefix, "event_ref"), add)
		validateOptionalOperationalRefV0(item.ArtifactRef, fieldV0(itemPrefix, "artifact_ref"), add)
	}
}

func validateDiagnosticoContadoresV0(counters map[string]float64, prefix string, add func(string, string)) {
	if len(counters) > maxOperationalCountersV0 {
		add(ErrConsultaDemasiadoAmpliaV0, prefix)
	}
	for key, value := range counters {
		field := fieldV0(prefix, key)
		if !validSizedPatternV0(key, 1, maxOperationalTokenRunesV0, operationalCounterKeyPatternV0) {
			add(ErrOperationalStatusQueryInvalidaV0, field)
			continue
		}
		if code := forbiddenOperationalTextCodeV0(key); code != "" {
			add(code, field)
		}
		if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
			add(ErrOperationalStatusQueryInvalidaV0, field)
		}
	}
}

func validateDiagnosticoReferenciasV0(items []DiagnosticoReferenciaV0, prefix string, add func(string, string)) {
	if len(items) > maxOperationalReferenceItemsV0 {
		add(ErrConsultaDemasiadoAmpliaV0, prefix)
	}
	for index, item := range items {
		itemPrefix := fmt.Sprintf("%s[%d]", prefix, index)
		if !allowedV0(allowedDiagnosticoReferenceRelsV0, strings.TrimSpace(item.Rel)) {
			add(ErrOperationalStatusQueryInvalidaV0, fieldV0(itemPrefix, "rel"))
		}
		if !allowedV0(allowedDiagnosticoReferenceTypesV0, strings.TrimSpace(item.TargetType)) {
			add(ErrOperationalStatusQueryInvalidaV0, fieldV0(itemPrefix, "target_type"))
		}
		validateRequiredOperationalRefV0(item.TargetRef, fieldV0(itemPrefix, "target_ref"), add)
	}
}

func validateDiagnosticoWarningsV0(items []DiagnosticoWarningV0, prefix string, add func(string, string)) {
	if len(items) > maxOperationalWarningItemsV0 {
		add(ErrConsultaDemasiadoAmpliaV0, prefix)
	}
	for index, item := range items {
		itemPrefix := fmt.Sprintf("%s[%d]", prefix, index)
		validateOperationalTokenTextV0(item.Code, fieldV0(itemPrefix, "code"), add)
		if strings.TrimSpace(item.Section) != "" && !allowedV0(allowedOperationalSectionsV0, strings.TrimSpace(item.Section)) {
			add(ErrOperationalStatusQueryInvalidaV0, fieldV0(itemPrefix, "section"))
		}
		validateOperationalTextV0(item.Summary, fieldV0(itemPrefix, "summary"), maxOperationalTextRunesV0, false, add)
	}
}

func validateDiagnosticoPrivacyV0(privacy DiagnosticoPrivacyV0, prefix string, add func(string, string)) {
	if privacy.ContainsSecret {
		add(ErrSecretoDetectadoV0, fieldV0(prefix, "contains_secret"))
	}
	if privacy.ContainsTranscript {
		add(ErrTranscriptNoPermitidoV0, fieldV0(prefix, "contains_transcript"))
	}
	if privacy.ContainsPrompt {
		add(ErrOperationalStatusQueryInvalidaV0, fieldV0(prefix, "contains_prompt"))
	}
	if privacy.ContainsCompletion {
		add(ErrOperationalStatusQueryInvalidaV0, fieldV0(prefix, "contains_completion"))
	}
	if privacy.ContainsConnectionDetail {
		add(ErrOperationalStatusQueryInvalidaV0, fieldV0(prefix, "contains_connection_detail"))
	}
	if strings.TrimSpace(privacy.RedactionLevel) != "" &&
		!orquestarails.IsOperationalPrivacyRedactionLevelV0(privacy.RedactionLevel) {
		add(ErrOperationalStatusQueryInvalidaV0, fieldV0(prefix, "redaction_level"))
	}
}
