package orquestacapacity

import "fmt"

type capacityDecisionValidatorV0 struct {
	issues []CapacityDecisionIssueV0
}

func (v *capacityDecisionValidatorV0) validate(decision CapacityDecisionV0) {
	v.requireConst("schema_version", decision.SchemaVersion, CapacityDecisionSchemaVersionV0, ErrCapacityDecisionSchemaNoSoportadoV0)
	v.validateRequest(decision.Request)
	v.validateResponse(decision.Request, decision.Response)
}

func (v *capacityDecisionValidatorV0) validateRequest(req CapacityDecisionRequestV0) {
	v.requireOpaque("request.task_ref", req.TaskRef)
	if !isOneOfV0(req.PerfilTarea, "analisis", "implementacion", "revision", "documentacion", "script", "handoff", "orquestacion", "otro") {
		v.add(ErrPerfilTareaNoSoportadoV0, "request.perfil_tarea")
	}
	v.optionalPhaseRef("request.fase", req.Fase)
	if !isOneOfV0(req.Riesgo, "low", "medium", "high", "critical") {
		v.add(ErrCapacityDecisionInvalidaV0, "request.riesgo")
	}
	if !isOneOfV0(req.CalidadRequerida, "normal", "alta", "critica") {
		v.add(ErrCapacityDecisionInvalidaV0, "request.calidad_requerida")
	}
	v.validateContexto(req.ContextoEstimado)
	v.validateRestricciones(req.Restricciones)
	v.validateEvidence(req.Evidencia)
}

func (v *capacityDecisionValidatorV0) validateResponse(req CapacityDecisionRequestV0, res CapacityDecisionResponseV0) {
	v.requireOpaque("response.decision_id", res.DecisionID)
	v.requireCapacityLevel("response.nivel_capacidad", res.NivelCapacidad)
	v.requireReasoningEffort("response.reasoning_effort", res.ReasoningEffort)
	v.validatePool(res.Pool)
	v.validateModel("response.modelo", res.Modelo)
	v.validateHome(res.Home, res.Pool.PoolID)
	v.validateQuota(res.Cuota)
	v.requireOpaque("response.politica_escalado", res.PoliticaEscalado)
	if !isOneOfV0(res.Handoff, "none", "preventivo", "requerido", "bloqueado") {
		v.add(ErrCapacityDecisionInvalidaV0, "response.handoff")
	}
	v.validateMotivos(res.Motivos)
	v.validateAlternativas(res.Alternativas)
	v.validateDegradacion(res.Degradacion)
	v.validateCrossInvariants(req, res)
}

func (v *capacityDecisionValidatorV0) validateContexto(ctx ContextoEstimadoV0) {
	if !isOneOfV0(ctx.Status, "known", "unknown") {
		v.add(ErrCapacityDecisionInvalidaV0, "request.contexto_estimado.status")
	}
	v.optionalNonNegativeInt("request.contexto_estimado.estimated_tokens", ctx.EstimatedTokens)
	v.optionalNonNegativeInt("request.contexto_estimado.estimated_messages", ctx.EstimatedMessages)
	v.optionalNonNegativeInt("request.contexto_estimado.estimated_seconds", ctx.EstimatedSeconds)
}

func (v *capacityDecisionValidatorV0) validateRestricciones(restricciones RestriccionesV0) {
	if restricciones.MaxCosteRelativo != nil && (*restricciones.MaxCosteRelativo <= 0 || *restricciones.MaxCosteRelativo > 100) {
		v.add(ErrCapacityDecisionInvalidaV0, "request.restricciones.max_coste_relativo")
	}
	if restricciones.LocalOnly && restricciones.RemoteAllowed {
		v.add(ErrProveedorRestringidoV0, "request.restricciones")
	}
}

func (v *capacityDecisionValidatorV0) validateEvidence(bundle EvidenceBundleV0) {
	v.validateSignals("request.evidencia.quality_signals", "quality", bundle.QualitySignals)
	v.validateSignals("request.evidencia.benchmark_signals", "benchmark", bundle.BenchmarkSignals)
	v.validateSignals("request.evidencia.telemetry_signals", "telemetry", bundle.TelemetrySignals)
	v.validateSignals("request.evidencia.quota_signals", "quota", bundle.QuotaSignals)
	v.validateSignals("request.evidencia.failure_signals", "failure", bundle.FailureSignals)
	v.validateSignals("request.evidencia.human_signals", "human", bundle.HumanSignals)
}

func (v *capacityDecisionValidatorV0) validateSignals(field, expectedKind string, signals []EvidenceSignalV0) {
	if len(signals) > 12 {
		v.add(ErrCapacityDecisionInvalidaV0, field)
	}
	for i, signal := range signals {
		prefix := fmt.Sprintf("%s[%d]", field, i)
		v.requireOpaque(prefix+".signal_ref", signal.SignalRef)
		if signal.Kind != expectedKind || !isOneOfV0(signal.Kind, "quality", "benchmark", "telemetry", "quota", "failure", "human") {
			v.add(ErrCapacityDecisionInvalidaV0, prefix+".kind")
		}
		if !isOneOfV0(signal.Source, "real", "estimated", "manual", "adapter", "benchmark", "observed_score", "human", "unavailable") {
			v.add(ErrCapacityDecisionInvalidaV0, prefix+".source")
		}
		v.requireFreshness(prefix+".freshness", signal.Freshness)
		if !isOneOfV0(signal.Weight, "low", "medium", "high", "critical") {
			v.add(ErrCapacityDecisionInvalidaV0, prefix+".weight")
		}
		v.optionalSafeText(prefix+".summary", signal.Summary)
	}
}
