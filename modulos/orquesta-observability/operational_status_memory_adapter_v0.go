package orquestaobservability

import "strings"

type OperationalStatusMemoryAdapterV0 struct {
	diagnostics map[operationalStatusMemoryKeyV0]DiagnosticoCompactoV0
}

type operationalStatusMemoryKeyV0 struct {
	scope         string
	subjectRef    string
	correlationID string
}

func NewOperationalStatusMemoryAdapterV0(diagnostics []DiagnosticoCompactoV0) (*OperationalStatusMemoryAdapterV0, error) {
	adapter := &OperationalStatusMemoryAdapterV0{
		diagnostics: make(map[operationalStatusMemoryKeyV0]DiagnosticoCompactoV0, len(diagnostics)),
	}
	for _, diagnostic := range diagnostics {
		if err := ValidateDiagnosticoCompactoV0(diagnostic); err != nil {
			return nil, err
		}
		key := operationalStatusMemoryDiagnosticKeyV0(diagnostic)
		if _, exists := adapter.diagnostics[key]; exists {
			return nil, operationalStatusValidationErrorV0(ErrOperationalStatusQueryInvalidaV0, "diagnostics")
		}
		adapter.diagnostics[key] = cloneDiagnosticoCompactoForMemoryV0(diagnostic)
	}
	return adapter, nil
}

func (adapter *OperationalStatusMemoryAdapterV0) QueryOperationalStatusV0(query OperationalStatusQueryV0) (DiagnosticoCompactoV0, error) {
	if err := ValidateOperationalStatusQueryV0(query); err != nil {
		return DiagnosticoCompactoV0{}, err
	}
	if adapter == nil || len(adapter.diagnostics) == 0 {
		return DiagnosticoCompactoV0{}, operationalStatusValidationErrorV0(ErrProyeccionNoDisponibleV0, "projection")
	}

	diagnostic, exists := adapter.diagnostics[operationalStatusMemoryQueryKeyV0(query)]
	if !exists {
		return DiagnosticoCompactoV0{}, operationalStatusValidationErrorV0(ErrDiagnosticoNoDisponibleV0, "diagnostic")
	}
	if err := validateOperationalStatusMemoryFreshnessV0(query, diagnostic); err != nil {
		return DiagnosticoCompactoV0{}, err
	}

	return filterDiagnosticoCompactoForQueryV0(diagnostic, query), nil
}

func operationalStatusMemoryDiagnosticKeyV0(diagnostic DiagnosticoCompactoV0) operationalStatusMemoryKeyV0 {
	return operationalStatusMemoryKeyV0{
		scope:         strings.TrimSpace(diagnostic.Scope),
		subjectRef:    strings.TrimSpace(diagnostic.SubjectRef),
		correlationID: strings.TrimSpace(diagnostic.CorrelationID),
	}
}

func operationalStatusMemoryQueryKeyV0(query OperationalStatusQueryV0) operationalStatusMemoryKeyV0 {
	return operationalStatusMemoryKeyV0{
		scope:         strings.TrimSpace(query.Scope),
		subjectRef:    strings.TrimSpace(query.SubjectRef),
		correlationID: strings.TrimSpace(query.CorrelationID),
	}
}

func validateOperationalStatusMemoryFreshnessV0(query OperationalStatusQueryV0, diagnostic DiagnosticoCompactoV0) error {
	if query.Freshness == nil {
		return nil
	}
	if diagnostic.Freshness.Stale {
		return operationalStatusValidationErrorV0(ErrFrescuraNoGarantizadaV0, "freshness.stale")
	}
	if query.Freshness.MaxAgeSeconds > 0 && diagnostic.Freshness.MaxAgeSeconds > query.Freshness.MaxAgeSeconds {
		return operationalStatusValidationErrorV0(ErrFrescuraNoGarantizadaV0, "freshness.max_age_seconds")
	}
	queryWatermarkRef := strings.TrimSpace(query.Freshness.WatermarkRef)
	if queryWatermarkRef != "" && strings.TrimSpace(diagnostic.Freshness.WatermarkRef) != queryWatermarkRef {
		return operationalStatusValidationErrorV0(ErrFrescuraNoGarantizadaV0, "freshness.watermark_ref")
	}
	return nil
}

func filterDiagnosticoCompactoForQueryV0(diagnostic DiagnosticoCompactoV0, query OperationalStatusQueryV0) DiagnosticoCompactoV0 {
	filtered := cloneDiagnosticoCompactoForMemoryV0(diagnostic)
	sections := map[string]bool{}
	for _, section := range query.IncludeSections {
		sections[strings.TrimSpace(section)] = true
	}

	if !sections[OperationalStatusSectionProgresoV0] {
		filtered.Progreso = DiagnosticoProgresoV0{}
	}
	if !sections[OperationalStatusSectionSaludV0] {
		filtered.Salud = nil
	}
	if !sections[OperationalStatusSectionBloqueosV0] {
		filtered.Bloqueos = nil
	}
	if !sections[OperationalStatusSectionActividadRecienteV0] {
		filtered.ActividadReciente = nil
	}
	if !sections[OperationalStatusSectionContadoresV0] {
		filtered.Contadores = nil
	}
	if !sections[OperationalStatusSectionReferenciasV0] {
		filtered.Referencias = nil
	}

	filtered.Salud = limitDiagnosticoSaludV0(filtered.Salud, query.Limit)
	filtered.Bloqueos = limitDiagnosticoBloqueosV0(filtered.Bloqueos, query.Limit)
	filtered.ActividadReciente = limitDiagnosticoActividadV0(filtered.ActividadReciente, query.Limit)
	filtered.Referencias = limitDiagnosticoReferenciasV0(filtered.Referencias, query.Limit)
	filtered.Warnings = limitDiagnosticoWarningsV0(filtered.Warnings, query.Limit)
	return filtered
}

func cloneDiagnosticoCompactoForMemoryV0(diagnostic DiagnosticoCompactoV0) DiagnosticoCompactoV0 {
	clone := diagnostic
	clone.Salud = append([]DiagnosticoSaludCheckV0(nil), diagnostic.Salud...)
	for index := range clone.Salud {
		clone.Salud[index].EvidenceRefs = append([]string(nil), diagnostic.Salud[index].EvidenceRefs...)
	}
	clone.Bloqueos = append([]DiagnosticoBloqueoV0(nil), diagnostic.Bloqueos...)
	for index := range clone.Bloqueos {
		clone.Bloqueos[index].EvidenceRefs = append([]string(nil), diagnostic.Bloqueos[index].EvidenceRefs...)
	}
	clone.ActividadReciente = append([]DiagnosticoActividadV0(nil), diagnostic.ActividadReciente...)
	if diagnostic.Contadores != nil {
		clone.Contadores = make(map[string]float64, len(diagnostic.Contadores))
		for key, value := range diagnostic.Contadores {
			clone.Contadores[key] = value
		}
	}
	clone.Referencias = append([]DiagnosticoReferenciaV0(nil), diagnostic.Referencias...)
	clone.Warnings = append([]DiagnosticoWarningV0(nil), diagnostic.Warnings...)
	return clone
}

func limitDiagnosticoSaludV0(items []DiagnosticoSaludCheckV0, limit int) []DiagnosticoSaludCheckV0 {
	if limit >= len(items) {
		return items
	}
	return items[:limit]
}

func limitDiagnosticoBloqueosV0(items []DiagnosticoBloqueoV0, limit int) []DiagnosticoBloqueoV0 {
	if limit >= len(items) {
		return items
	}
	return items[:limit]
}

func limitDiagnosticoActividadV0(items []DiagnosticoActividadV0, limit int) []DiagnosticoActividadV0 {
	if limit >= len(items) {
		return items
	}
	return items[:limit]
}

func limitDiagnosticoReferenciasV0(items []DiagnosticoReferenciaV0, limit int) []DiagnosticoReferenciaV0 {
	if limit >= len(items) {
		return items
	}
	return items[:limit]
}

func limitDiagnosticoWarningsV0(items []DiagnosticoWarningV0, limit int) []DiagnosticoWarningV0 {
	if limit >= len(items) {
		return items
	}
	return items[:limit]
}
