package orquestacapacity

import "fmt"

func (v *capacityDecisionValidatorV0) validatePool(pool PoolCapacidadV0) {
	v.requireOpaque("response.pool.pool_id", pool.PoolID)
	if !isOneOfV0(pool.ProviderKind, "remote", "local", "hybrid") {
		v.add(ErrPoolNoDisponibleV0, "response.pool.provider_kind")
	}
	if !isOneOfV0(pool.RuntimeKind, "cli", "api", "local_runtime", "browser", "mobile_runtime", "otro") {
		v.add(ErrPoolNoDisponibleV0, "response.pool.runtime_kind")
	}
	if !isOneOfV0(pool.PlanKind, "free", "paid", "enterprise", "local", "unknown") {
		v.add(ErrPoolNoDisponibleV0, "response.pool.plan_kind")
	}
	if !pool.Active {
		v.add(ErrPoolNoDisponibleV0, "response.pool.active")
	}
	if pool.TotalSlots < 1 || pool.TotalSlots > 64 {
		v.add(ErrPoolNoDisponibleV0, "response.pool.total_slots")
	}
	if pool.ReservedSlots < 0 || pool.ReservedSlots > pool.TotalSlots {
		v.add(ErrPoolNoDisponibleV0, "response.pool.reserved_slots")
	}
	if pool.TotalSlots > 0 && pool.ReservedSlots >= pool.TotalSlots {
		v.add(ErrPoolNoDisponibleV0, "response.pool.reserved_slots")
	}
	if !isOneOfV0(pool.HandoffPolicy, "preventivo", "estricto", "manual", "none") {
		v.add(ErrPoolNoDisponibleV0, "response.pool.handoff_policy")
	}
	if !isOneOfV0(pool.TelemetrySource, "real", "estimated", "manual", "unavailable") {
		v.add(ErrPoolNoDisponibleV0, "response.pool.telemetry_source")
	}
	v.validateStringSet("response.pool.credential_modes", pool.CredentialModes, "oauth", "api_key", "browser_session", "local_runtime", "none", "unknown")
	v.requireQuotaScope("response.pool.quota_scope", pool.QuotaScope)
	if len(pool.Modelos) == 0 || len(pool.Modelos) > 16 {
		v.add(ErrModeloNoHabilitadoV0, "response.pool.modelos")
	}
	for i, model := range pool.Modelos {
		v.validateModel(fmt.Sprintf("response.pool.modelos[%d]", i), model)
	}
}

func (v *capacityDecisionValidatorV0) validateModel(field string, model ModelCapacityRefV0) {
	v.requireOpaque(field+".model_ref", model.ModelRef)
	if !model.Enabled {
		v.add(ErrModeloNoHabilitadoV0, field+".enabled")
	}
	if model.Priority < 0 || model.Priority > 1000 {
		v.add(ErrModeloNoHabilitadoV0, field+".priority")
	}
	if model.RelativeCost <= 0 || model.RelativeCost > 100 {
		v.add(ErrModeloNoHabilitadoV0, field+".relative_cost")
	}
	if !isOneOfV0(model.Locality, "remote", "local", "unknown") {
		v.add(ErrModeloNoHabilitadoV0, field+".locality")
	}
	v.validateKnownLimits(field+".known_limits", model.KnownLimits)
	if !isOneOfV0(model.ExecutionMode, "remote_api", "remote_cli", "local_process", "local_service", "browser", "unknown") {
		v.add(ErrModeloNoHabilitadoV0, field+".execution_mode")
	}
	if !isOneOfV0(model.EnablementSource, "benchmark", "observed_score", "manual_candidate", "adapter_declared", "unknown") {
		v.add(ErrModeloNoHabilitadoV0, field+".enablement_source")
	}
	v.validateStringSet(field+".supported_efforts", model.SupportedEfforts, "none", "low", "medium", "high", "xhigh")
	if model.Locality == "local" {
		v.validateLocalModelScore(field, model)
	}
}

func (v *capacityDecisionValidatorV0) validateKnownLimits(field string, limits *KnownLimitsV0) {
	if limits == nil {
		return
	}
	v.optionalNonNegativeInt(field+".messages", limits.Messages)
	v.optionalNonNegativeInt(field+".tokens", limits.Tokens)
	v.optionalNonNegativeInt(field+".seconds", limits.Seconds)
	v.optionalNonNegativeFloat(field+".credits", limits.Credits)
	if limits.Window != "" && !isOneOfV0(limits.Window, "primary", "short", "daily", "weekly", "unknown") {
		v.add(ErrModeloNoHabilitadoV0, field+".window")
	}
}

func (v *capacityDecisionValidatorV0) validateLocalModelScore(field string, model ModelCapacityRefV0) {
	if model.Score == nil {
		v.add(ErrLocalSinScoreSuficienteV0, field+".score")
		return
	}
	score := model.Score
	if !isOneOfV0(model.EnablementSource, "benchmark", "observed_score") {
		v.add(ErrLocalSinScoreSuficienteV0, field+".enablement_source")
	}
	if !isOneOfV0(score.Materia, "codigo", "documentacion", "analisis", "revision", "otro") {
		v.add(ErrLocalSinScoreSuficienteV0, field+".score.materia")
	}
	if score.Confianza < 0.6 || score.Confianza > 1 {
		v.add(ErrLocalSinScoreSuficienteV0, field+".score.confianza")
	}
	if score.Muestras < 0 || score.Benchmarks < 0 || score.Muestras+score.Benchmarks == 0 {
		v.add(ErrLocalSinScoreSuficienteV0, field+".score")
	}
	v.optionalUTCTime(field+".score.last_observed_at", score.LastObservedAt)
}

func (v *capacityDecisionValidatorV0) validateHome(home *AgentHomeSummaryV0, poolID string) {
	if home == nil {
		return
	}
	v.requireOpaque("response.home.logical_agent_ref", home.LogicalAgentRef)
	v.requireOpaque("response.home.home_ref", home.HomeRef)
	v.requireOpaque("response.home.pool_id", home.PoolID)
	v.requireOpaque("response.home.runtime_ref", home.RuntimeRef)
	v.requireOpaque("response.home.provider_ref", home.ProviderRef)
	v.requireOpaque("response.home.account_ref", home.AccountRef)
	v.optionalOpaque("response.home.credential_ref", home.CredentialRef)
	if home.PoolID != "" && poolID != "" && home.PoolID != poolID {
		v.add(ErrPoolNoDisponibleV0, "response.home.pool_id")
	}
}

func (v *capacityDecisionValidatorV0) validateQuota(quota QuotaSnapshotV0) {
	if !isOneOfV0(quota.Source, "real", "estimated", "manual", "unavailable") {
		v.add(ErrCuotaNoDisponibleV0, "response.cuota.source")
	}
	v.requireFreshness("response.cuota.freshness", quota.Freshness)
	v.optionalUTCTime("response.cuota.checked_at", quota.CheckedAt)
	v.requireQuotaScope("response.cuota.scope", quota.Scope)
	if !isOneOfV0(quota.WindowKind, "primary", "short", "daily", "weekly", "provider", "unknown") {
		v.add(ErrCuotaNoDisponibleV0, "response.cuota.window_kind")
	}
	v.optionalUTCTime("response.cuota.reset_at", quota.ResetAt)
	v.optionalNonNegativeInt("response.cuota.remaining_seconds", quota.RemainingSeconds)
	v.optionalNonNegativeInt("response.cuota.remaining_messages", quota.RemainingMessages)
	v.optionalNonNegativeInt("response.cuota.remaining_tokens", quota.RemainingTokens)
	v.optionalNonNegativeFloat("response.cuota.remaining_credits", quota.RemainingCredits)
	v.optionalConfidence("response.cuota.confidence", quota.Confidence)
	v.optionalOpaque("response.cuota.raw_ref", quota.RawRef)
}

func (v *capacityDecisionValidatorV0) validateMotivos(motivos []string) {
	if len(motivos) == 0 || len(motivos) > 10 {
		v.add(ErrCapacityDecisionInvalidaV0, "response.motivos")
	}
	seen := map[string]bool{}
	for i, motivo := range motivos {
		field := fmt.Sprintf("response.motivos[%d]", i)
		v.requireSafeText(field, motivo)
		if seen[motivo] {
			v.add(ErrCapacityDecisionInvalidaV0, field)
		}
		seen[motivo] = true
	}
}

func (v *capacityDecisionValidatorV0) validateAlternativas(alternativas []CapacityAlternativeV0) {
	if len(alternativas) > 8 {
		v.add(ErrCapacityDecisionInvalidaV0, "response.alternativas")
	}
	for i, alt := range alternativas {
		field := fmt.Sprintf("response.alternativas[%d]", i)
		if alt.Priority < 0 || alt.Priority > 1000 {
			v.add(ErrCapacityDecisionInvalidaV0, field+".priority")
		}
		v.requireOpaque(field+".pool_id", alt.PoolID)
		v.requireOpaque(field+".model_ref", alt.ModelRef)
		if alt.RelativeCost <= 0 || alt.RelativeCost > 100 {
			v.add(ErrCapacityDecisionInvalidaV0, field+".relative_cost")
		}
		if !isOneOfV0(alt.Availability, "available", "limited", "blocked", "unknown") {
			v.add(ErrCapacityDecisionInvalidaV0, field+".availability")
		}
		v.requireSafeText(field+".reason", alt.Reason)
	}
}

func (v *capacityDecisionValidatorV0) validateDegradacion(degradacion DegradacionV0) {
	if !isOneOfV0(degradacion.Action, "none", "bajar_nivel", "cambiar_modelo", "cambiar_home", "cambiar_pool", "handoff_preventivo", "bloqueado") {
		v.add(ErrCapacityDecisionInvalidaV0, "response.degradacion.action")
	}
	if degradacion.TargetLevel != "" {
		v.requireCapacityLevel("response.degradacion.target_level", degradacion.TargetLevel)
	}
	v.optionalOpaque("response.degradacion.target_pool_id", degradacion.TargetPoolID)
	v.optionalOpaque("response.degradacion.target_model_ref", degradacion.TargetModelRef)
	v.requireSafeText("response.degradacion.reason", degradacion.Reason)
}
