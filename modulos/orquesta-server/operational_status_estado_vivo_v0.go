package orquestaserver

import (
	"strings"

	orquestaestadovivo "orquesta/modulos/orquesta-estado-vivo"
	orquestaobservability "orquesta/modulos/orquesta-observability"
)

func residentOperationalCountersWithEstadoVivoV0(
	counters map[string]float64,
	evidencias []orquestaestadovivo.EvidenciaEstadoV0,
	projection orquestaestadovivo.ProyeccionCicloVidaV0,
	warnings *[]orquestaobservability.DiagnosticoWarningV0,
) map[string]float64 {
	if counters == nil {
		counters = map[string]float64{}
	}
	for _, item := range []struct {
		key   string
		value float64
	}{
		{key: "estado_vivo_evidencias", value: float64(nonNegativeServerIntV0(len(evidencias)))},
		{key: "estado_vivo_nodos", value: float64(nonNegativeServerIntV0(len(projection.Nodos)))},
		{key: "estado_vivo_conflictos", value: float64(nonNegativeServerIntV0(residentOperationalEstadoVivoConflictsV0(projection)))},
		{key: "estado_vivo_procesos_vivos", value: float64(nonNegativeServerIntV0(residentOperationalEstadoVivoLiveProcessesV0(evidencias)))},
		{key: "estado_vivo_terminales", value: float64(nonNegativeServerIntV0(residentOperationalEstadoVivoTerminalNodesV0(projection)))},
		{key: "estado_vivo_entregas_parciales", value: float64(nonNegativeServerIntV0(residentOperationalEstadoVivoPartialDeliveryNodesV0(projection)))},
	} {
		if _, exists := counters[item.key]; !exists && len(counters) >= residentOperationalDiagnosticoCounterBudgetV0 {
			*warnings = residentOperationalAppendWarningOnceV0(
				*warnings,
				residentOperationalEstadoVivoCounterBudgetCodeV0,
				orquestaobservability.OperationalStatusSectionContadoresV0,
				"contadores de estado vivo compactados por presupuesto",
			)
			continue
		}
		counters[item.key] = item.value
	}
	return counters
}

func residentOperationalReferencesWithEstadoVivoV0(
	refs []orquestaobservability.DiagnosticoReferenciaV0,
	evidencias []orquestaestadovivo.EvidenciaEstadoV0,
	warnings *[]orquestaobservability.DiagnosticoWarningV0,
) []orquestaobservability.DiagnosticoReferenciaV0 {
	seen := map[string]struct{}{}
	for _, ref := range refs {
		seen[strings.TrimSpace(ref.Rel)+"\x00"+strings.TrimSpace(ref.TargetType)+"\x00"+strings.TrimSpace(ref.TargetRef)] = struct{}{}
	}
	appendRef := func(rel string, targetType string, targetRef string) {
		targetRef = strings.TrimSpace(targetRef)
		if targetRef == "" || !serverEvidenceRefPatternV0.MatchString(targetRef) {
			return
		}
		key := rel + "\x00" + targetType + "\x00" + targetRef
		if _, ok := seen[key]; ok {
			return
		}
		if len(refs) >= residentOperationalDiagnosticoReferenceBudgetV0 {
			*warnings = residentOperationalAppendWarningOnceV0(
				*warnings,
				residentOperationalEstadoVivoReferenceBudgetCodeV0,
				orquestaobservability.OperationalStatusSectionReferenciasV0,
				"referencias de estado vivo compactadas por presupuesto",
			)
			return
		}
		seen[key] = struct{}{}
		refs = append(refs, orquestaobservability.DiagnosticoReferenciaV0{
			Rel:        rel,
			TargetType: targetType,
			TargetRef:  targetRef,
		})
	}
	for _, evidencia := range evidencias {
		appendRef("related", "flow", evidencia.RunRef)
		appendRef("runtime", "runtime", evidencia.GoalRef)
		appendRef("runtime", "runtime", evidencia.ExternalGoalRef)
		for _, evidenceRef := range sanitizeServerEvidenceRefsV0(evidencia.EvidenceRefs) {
			appendRef("artifact", "artifact", evidenceRef)
		}
	}
	return refs
}

func residentOperationalEstadoVivoConflictsV0(
	projection orquestaestadovivo.ProyeccionCicloVidaV0,
) int {
	total := 0
	for _, node := range projection.Nodos {
		total += len(node.Conflictos)
	}
	return total
}

func residentOperationalEstadoVivoLiveProcessesV0(
	evidencias []orquestaestadovivo.EvidenciaEstadoV0,
) int {
	total := 0
	for _, evidencia := range evidencias {
		if evidencia.ProcesoVivo {
			total++
		}
	}
	return total
}

func residentOperationalEstadoVivoTerminalNodesV0(
	projection orquestaestadovivo.ProyeccionCicloVidaV0,
) int {
	total := 0
	for _, node := range projection.Nodos {
		switch node.Fase {
		case orquestaestadovivo.FaseTerminalAceptadoV0, orquestaestadovivo.FaseTerminalReworkV0:
			total++
		}
	}
	return total
}

func residentOperationalEstadoVivoPartialDeliveryNodesV0(
	projection orquestaestadovivo.ProyeccionCicloVidaV0,
) int {
	total := 0
	for _, node := range projection.Nodos {
		if node.Fase == orquestaestadovivo.FaseEntregadoParcialV0 {
			total++
		}
	}
	return total
}

func residentOperationalAppendWarningOnceV0(
	warnings []orquestaobservability.DiagnosticoWarningV0,
	code string,
	section string,
	summary string,
) []orquestaobservability.DiagnosticoWarningV0 {
	code = strings.TrimSpace(code)
	for _, warning := range warnings {
		if strings.TrimSpace(warning.Code) == code {
			return warnings
		}
	}
	return append(warnings, orquestaobservability.DiagnosticoWarningV0{
		Code:    code,
		Section: strings.TrimSpace(section),
		Summary: strings.TrimSpace(summary),
	})
}
