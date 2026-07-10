package orquestaestadovivo

import "strings"

const (
	RazonVeredictoRuntimeConfirmadoV0            = "runtime_liveness_confirmed"
	RazonVeredictoResultadoDurableTerminalV0     = "durable_terminal_result"
	RazonVeredictoProcesoMuertoEstadoStaleV0     = "runtime_observed_dead_state_running"
	RazonVeredictoProcesoVivoTrasTerminalV0      = "process_live_after_durable_terminal"
	RazonVeredictoLivenessNoConfirmadoV0         = "runtime_liveness_unconfirmed"
	RazonVeredictoObservacionRuntimeIncompletaV0 = "runtime_observation_indeterminate"
	RazonVeredictoIdentidadRuntimeFaltanteV0     = "goal_runtime_identity_missing"
	RazonVeredictoIdentidadNoCoincidenteV0       = "goal_execution_identity_mismatch"
	RazonVeredictoEvidenciaCausalInsuficienteV0  = "causal_evidence_insufficient"
)

// DerivarVeredictoCausalV0 es la unica autoridad pura para reconciliar state,
// resultado durable y liveness runtime. Las fuentes observan y traducen; no
// deciden si un trabajo sigue vivo o es terminal.
func DerivarVeredictoCausalV0(evidencias []EvidenciaEstadoV0) VeredictoCausalV0 {
	veredicto := VeredictoCausalV0{SchemaVersion: VeredictoCausalSchemaV0}
	var fuentesProceso, fuentesTerminal []string
	expectedIdentities := map[string]struct{}{}
	expectedGenerations := map[string]struct{}{}
	observedIdentities := map[string]struct{}{}
	observedGenerations := map[string]struct{}{}
	identidadRuntimeFaltante := false
	identidadNoCoincidente := false
	observacionIncompleta := false
	procesoVivoObservado := false

	for _, evidencia := range evidencias {
		veredicto.RunRef, identidadNoCoincidente = acumularRefCoincidenteV0(veredicto.RunRef, evidencia.RunRef, identidadNoCoincidente)
		veredicto.GoalRef, identidadNoCoincidente = acumularRefCoincidenteV0(veredicto.GoalRef, evidencia.GoalRef, identidadNoCoincidente)
		veredicto.ExternalGoalRef, identidadNoCoincidente = acumularRefCoincidenteV0(veredicto.ExternalGoalRef, evidencia.ExternalGoalRef, identidadNoCoincidente)
		if estado := tokenEstadoV0(evidencia.Estado); evidencia.Scope != ScopeBackendServiceV0 && esEstadoRunningV0(estado) {
			veredicto.EstadoPersistido = estado
		}
		if evidencia.Scope != ScopeGoalExecutionV0 && evidencia.Scope != ScopeBackendServiceV0 &&
			(evidencia.RuntimeObservationAttempted || evidencia.RuntimeObservado || evidencia.ProcesoVivo) {
			if strings.TrimSpace(evidencia.RuntimeIdentityRef) == "" && strings.TrimSpace(evidencia.RuntimeGenerationRef) == "" {
				identidadRuntimeFaltante = true
			} else {
				observacionIncompleta = true
			}
		}
		if evidencia.Scope == ScopeGoalExecutionV0 {
			if evidencia.RuntimeIdentityMismatch {
				identidadNoCoincidente = true
			}
			identityRef := strings.TrimSpace(evidencia.RuntimeIdentityRef)
			generationRef := strings.TrimSpace(evidencia.RuntimeGenerationRef)
			esObservacion := evidencia.RuntimeObservationAttempted || evidencia.RuntimeObservado || evidencia.ProcesoVivo
			if !esObservacion {
				agregarRefSetV0(expectedIdentities, identityRef)
				agregarRefSetV0(expectedGenerations, generationRef)
			}
			if esObservacion && identityRef == "" && generationRef == "" {
				identidadRuntimeFaltante = true
			}
			if evidencia.RuntimeObservationAttempted && !evidencia.RuntimeObservado {
				observacionIncompleta = true
			}
			if evidencia.ProcesoVivo && !evidencia.RuntimeObservado {
				observacionIncompleta = true
			}
			if evidencia.RuntimeObservado {
				veredicto.RuntimeObservado = true
				agregarRefSetV0(observedIdentities, identityRef)
				agregarRefSetV0(observedGenerations, generationRef)
				fuentesProceso = append(fuentesProceso, evidencia.Fuente)
				if evidencia.ProcesoVivo {
					procesoVivoObservado = true
				}
			}
		}
		if evidencia.Scope != ScopeBackendServiceV0 && evidencia.Terminal {
			veredicto.ResultadoTerminal = true
			fuentesTerminal = append(fuentesTerminal, evidencia.Fuente)
			if evidencia.Aceptado {
				veredicto.ResultadoAceptado = true
			}
		}
		veredicto.EvidenceRefs = append(veredicto.EvidenceRefs, evidencia.EvidenceRefs...)
	}

	if refsNoCoincidenV0(expectedIdentities, observedIdentities) || refsNoCoincidenV0(expectedGenerations, observedGenerations) {
		identidadNoCoincidente = true
	}
	veredicto.RuntimeIdentityRefs = refsOrdenadasV0(expectedIdentities, observedIdentities)
	veredicto.RuntimeGenerationRefs = refsOrdenadasV0(expectedGenerations, observedGenerations)
	if len(veredicto.RuntimeIdentityRefs) == 1 {
		veredicto.RuntimeIdentityRef = veredicto.RuntimeIdentityRefs[0]
	}
	if len(veredicto.RuntimeGenerationRefs) == 1 {
		veredicto.RuntimeGenerationRef = veredicto.RuntimeGenerationRefs[0]
	}
	veredicto.ProcesoVivo = procesoVivoObservado && !identidadNoCoincidente && !identidadRuntimeFaltante
	veredicto.EvidenceRefs = ordenarStringsUnicosV0(veredicto.EvidenceRefs)

	switch {
	case identidadNoCoincidente:
		marcarDivergenciaV0(&veredicto, RazonVeredictoIdentidadNoCoincidenteV0)
	case identidadRuntimeFaltante:
		marcarDivergenciaV0(&veredicto, RazonVeredictoIdentidadRuntimeFaltanteV0)
	case veredicto.ProcesoVivo && veredicto.ResultadoTerminal:
		marcarDivergenciaV0(&veredicto, RazonVeredictoProcesoVivoTrasTerminalV0)
		veredicto.Conflictos = []ConflictoEstadoV0{{
			RunRef:  veredicto.RunRef,
			Codigo:  CodigoConflictoProcesoVivoTrasTerminalV0,
			Fuentes: ordenarStringsUnicosV0(append(fuentesProceso, fuentesTerminal...)),
		}}
	case observacionIncompleta:
		veredicto.Clase = VeredictoIndeterminateV0
		veredicto.ReasonCode = RazonVeredictoObservacionRuntimeIncompletaV0
	case veredicto.ResultadoTerminal:
		veredicto.Clase = VeredictoTerminalByArtifactV0
		veredicto.ReasonCode = RazonVeredictoResultadoDurableTerminalV0
	case veredicto.ProcesoVivo:
		veredicto.Clase = VeredictoRunningConfirmedV0
		veredicto.ReasonCode = RazonVeredictoRuntimeConfirmadoV0
		veredicto.PublicarRunning = true
	case veredicto.EstadoPersistido != "" && veredicto.RuntimeObservado:
		veredicto.Clase = VeredictoProcessDeadStateStaleV0
		veredicto.ReasonCode = RazonVeredictoProcesoMuertoEstadoStaleV0
		veredicto.RequiereReparacion = true
	case veredicto.EstadoPersistido != "":
		veredicto.Clase = VeredictoIndeterminateV0
		veredicto.ReasonCode = RazonVeredictoLivenessNoConfirmadoV0
	default:
		veredicto.Clase = VeredictoIndeterminateV0
		veredicto.ReasonCode = RazonVeredictoEvidenciaCausalInsuficienteV0
	}

	return veredicto
}

func marcarDivergenciaV0(veredicto *VeredictoCausalV0, reason string) {
	veredicto.Clase = VeredictoDivergentNeedsRepairV0
	veredicto.ReasonCode = reason
	veredicto.RequiereReparacion = true
}

func agregarRefSetV0(refs map[string]struct{}, ref string) {
	ref = strings.TrimSpace(ref)
	if ref != "" {
		refs[ref] = struct{}{}
	}
}

func refsNoCoincidenV0(expected, observed map[string]struct{}) bool {
	if len(expected) == 0 || len(observed) == 0 {
		return false
	}
	for ref := range observed {
		if _, ok := expected[ref]; !ok {
			return true
		}
	}
	return false
}

func refsOrdenadasV0(sets ...map[string]struct{}) []string {
	refs := make([]string, 0)
	for _, set := range sets {
		for ref := range set {
			refs = append(refs, ref)
		}
	}
	return ordenarStringsUnicosV0(refs)
}

func acumularRefCoincidenteV0(actual, candidata string, noCoincidente bool) (string, bool) {
	actual = strings.TrimSpace(actual)
	candidata = strings.TrimSpace(candidata)
	if candidata == "" {
		return actual, noCoincidente
	}
	if actual == "" {
		return candidata, noCoincidente
	}
	if candidata != actual {
		noCoincidente = true
	}
	if candidata < actual {
		actual = candidata
	}
	return actual, noCoincidente
}

func esEstadoRunningV0(estado string) bool {
	switch estado {
	case "running", "active", "in_progress", "proceso_vivo":
		return true
	default:
		return false
	}
}
