package orquestacapacity

func (v *capacityDecisionValidatorV0) validateCrossInvariants(req CapacityDecisionRequestV0, res CapacityDecisionResponseV0) {
	if res.Modelo.ModelRef != "" && !poolContainsModelV0(res.Pool, res.Modelo.ModelRef) {
		v.add(ErrModeloNoHabilitadoV0, "response.modelo.model_ref")
	}
	if !res.Modelo.Enabled {
		v.add(ErrModeloNoHabilitadoV0, "response.modelo.enabled")
	}
	if req.Restricciones.PaidAllowed == false && res.Pool.Paid {
		v.add(ErrProveedorRestringidoV0, "response.pool.paid")
	}
	if req.Restricciones.LocalOnly && (res.Pool.ProviderKind != "local" || res.Modelo.Locality != "local") {
		v.add(ErrProveedorRestringidoV0, "response.pool.provider_kind")
	}
	if !req.Restricciones.RemoteAllowed && (res.Pool.ProviderKind == "remote" || res.Modelo.Locality == "remote") {
		v.add(ErrProveedorRestringidoV0, "response.pool.provider_kind")
	}
	if req.Restricciones.MaxCosteRelativo != nil && res.Modelo.RelativeCost > *req.Restricciones.MaxCosteRelativo && !res.Pool.AllowsOvercost {
		v.add(ErrProveedorRestringidoV0, "response.modelo.relative_cost")
	}
	if req.Restricciones.RequiereHandoffPreventivo && res.Handoff == "none" {
		v.add(ErrHandoffRequeridoV0, "response.handoff")
	}
	v.validateEffortSupported(res)
	v.validateXHighGate(req, res)
	v.validateQuotaDecision(req, res)
}

func (v *capacityDecisionValidatorV0) validateEffortSupported(res CapacityDecisionResponseV0) {
	if len(res.Modelo.SupportedEfforts) == 0 && isOneOfV0(res.ReasoningEffort, "high", "xhigh") {
		v.add(ErrModeloNoHabilitadoV0, "response.modelo.supported_efforts")
		return
	}
	if len(res.Modelo.SupportedEfforts) > 0 && !containsV0(res.Modelo.SupportedEfforts, res.ReasoningEffort) {
		v.add(ErrModeloNoHabilitadoV0, "response.reasoning_effort")
	}
}

func (v *capacityDecisionValidatorV0) validateXHighGate(req CapacityDecisionRequestV0, res CapacityDecisionResponseV0) {
	if res.NivelCapacidad != "xhigh" && res.ReasoningEffort != "xhigh" {
		return
	}
	if !isOneOfV0(req.Riesgo, "high", "critical") || !hasFreshEscalationGateV0(req.Evidencia) {
		v.add(ErrEvidenciaInsuficienteParaXHighV0, "request.evidencia")
	}
	if res.Cuota.Freshness == "obsolete" || res.Cuota.Source == "unavailable" {
		v.add(ErrCuotaObsoletaV0, "response.cuota")
	}
}

func (v *capacityDecisionValidatorV0) validateQuotaDecision(req CapacityDecisionRequestV0, res CapacityDecisionResponseV0) {
	forced := isForcedDegradationOrHandoffV0(res)
	if res.Cuota.Source == "unavailable" && !forced {
		v.add(ErrCuotaNoDisponibleV0, "response.cuota.source")
	}
	if res.Cuota.Freshness == "obsolete" {
		if !forced || isOneOfV0(res.NivelCapacidad, "high", "xhigh") {
			v.add(ErrCuotaObsoletaV0, "response.cuota.freshness")
		}
	}
	if windowInsufficientV0(req.ContextoEstimado, res.Cuota) && !forced {
		v.add(ErrHandoffRequeridoV0, "response.handoff")
	}
}
