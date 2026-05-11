package orquestacapacity

import "fmt"

type modelEscalationPolicyValidatorV0 struct {
	issues []ModelEscalationPolicyIssueV0
}

func (v *modelEscalationPolicyValidatorV0) validate(policy ModelEscalationPolicyV0) {
	v.requireConst("schema_version", policy.SchemaVersion, ModelEscalationPolicySchemaVersionV0, ErrModelEscalationPolicySchemaNoSoportadoV0)
	v.requireVersionedPolicyRef("policy_ref", policy.PolicyRef)
	v.validatePhaseRules(policy.PhaseRules)
	v.validateRiskQualityRules(policy.RiskQualityRules)
	v.validateEvidenceRules(policy.EvidenceRules)
	v.validateQuotaRules(policy.QuotaRules)
	v.validateLocalityRules(policy.LocalityRules)
	v.validateXHighGate(policy.XHighGate)
	v.validateDegradationOrder(policy.DegradationOrder)
	v.validateCrossInvariants(policy)
}

func (v *modelEscalationPolicyValidatorV0) validatePhaseRules(rules []ModelEscalationPhaseRuleV0) {
	v.requireCount("phase_rules", len(rules), 1, 32)
	for i, rule := range rules {
		field := fmt.Sprintf("phase_rules[%d]", i)
		v.requireOpaque(field+".rule_ref", rule.RuleRef)
		if !isOneOfV0(rule.PerfilTarea, "analisis", "implementacion", "revision", "documentacion", "script", "handoff", "orquestacion", "otro") {
			v.add(ErrModelEscalationPolicyInvalidaV0, field+".perfil_tarea")
		}
		v.requirePhaseRef(field+".fase", rule.Fase)
		v.requireCapacityLevel(field+".base_level", rule.BaseLevel)
		if !isOneOfV0(rule.MaxWithoutEvidence, "low", "medium", "high") {
			v.add(ErrModelEscalationPolicyInvalidaV0, field+".max_without_evidence")
		}
		v.validateLevelSet(field+".allowed_levels", rule.AllowedLevels)
		if rule.BaseLevel != "" && !containsV0(rule.AllowedLevels, rule.BaseLevel) {
			v.add(ErrModelEscalationPolicyInvalidaV0, field+".base_level")
		}
		if rule.MaxWithoutEvidence != "" && !containsV0(rule.AllowedLevels, rule.MaxWithoutEvidence) {
			v.add(ErrModelEscalationPolicyInvalidaV0, field+".max_without_evidence")
		}
		v.validateTriggers(field+".triggers", rule.Triggers)
		v.requireSafeText(field+".rationale", rule.Rationale)
	}
}

func (v *modelEscalationPolicyValidatorV0) validateRiskQualityRules(rules []ModelEscalationRiskQualityRuleV0) {
	v.requireCount("risk_quality_rules", len(rules), 1, 32)
	for i, rule := range rules {
		field := fmt.Sprintf("risk_quality_rules[%d]", i)
		v.requireOpaque(field+".rule_ref", rule.RuleRef)
		if !isOneOfV0(rule.Riesgo, "low", "medium", "high", "critical") {
			v.add(ErrModelEscalationPolicyInvalidaV0, field+".riesgo")
		}
		if !isOneOfV0(rule.CalidadRequerida, "normal", "alta", "critica") {
			v.add(ErrModelEscalationPolicyInvalidaV0, field+".calidad_requerida")
		}
		if !isOneOfV0(rule.Adjustment, "keep", "raise_one", "raise_to_high", "allow_xhigh_with_gate", "degrade") {
			v.add(ErrModelEscalationPolicyInvalidaV0, field+".adjustment")
		}
		v.validateLevelSet(field+".allowed_levels", rule.AllowedLevels)
		if containsV0(rule.AllowedLevels, "xhigh") || rule.Adjustment == "allow_xhigh_with_gate" {
			if !rule.RequiresEvidence || len(rule.AllowedLevels) == 1 {
				v.add(ErrModelEscalationPolicyXHighGateV0, field+".allowed_levels")
			}
		}
	}
}

func (v *modelEscalationPolicyValidatorV0) validateEvidenceRules(rules []ModelEscalationEvidenceRuleV0) {
	v.requireCount("evidence_rules", len(rules), 1, 32)
	for i, rule := range rules {
		field := fmt.Sprintf("evidence_rules[%d]", i)
		v.requireOpaque(field+".rule_ref", rule.RuleRef)
		if !isOneOfV0(rule.SignalKind, "quality", "benchmark", "telemetry", "quota", "failure", "human") {
			v.add(ErrModelEscalationPolicyInvalidaV0, field+".signal_kind")
		}
		if !isOneOfV0(rule.Source, "real", "estimated", "manual", "adapter", "benchmark", "observed_score", "human", "unavailable") {
			v.add(ErrModelEscalationPolicyInvalidaV0, field+".source")
		}
		v.requireFreshness(field+".freshness", rule.Freshness)
		if !isOneOfV0(rule.Weight, "low", "medium", "high", "critical") {
			v.add(ErrModelEscalationPolicyInvalidaV0, field+".weight")
		}
		if !isOneOfV0(rule.Effect, "raise", "keep", "degrade", "block", "require_handoff") {
			v.add(ErrModelEscalationPolicyInvalidaV0, field+".effect")
		}
		if rule.Source == "unavailable" && rule.Effect == "raise" {
			v.add(ErrModelEscalationPolicyInvalidaV0, field+".effect")
		}
		v.optionalSafeText(field+".rationale", rule.Rationale)
	}
}

func (v *modelEscalationPolicyValidatorV0) validateQuotaRules(rules []ModelEscalationQuotaRuleV0) {
	v.requireCount("quota_rules", len(rules), 1, 32)
	for i, rule := range rules {
		field := fmt.Sprintf("quota_rules[%d]", i)
		v.requireOpaque(field+".rule_ref", rule.RuleRef)
		if !isOneOfV0(rule.Source, "real", "estimated", "manual", "unavailable") {
			v.add(ErrModelEscalationPolicyInvalidaV0, field+".source")
		}
		v.requireFreshness(field+".freshness", rule.Freshness)
		v.requireQuotaScope(field+".scope", rule.Scope)
		if !isOneOfV0(rule.Effect, "allow", "degrade", "block_escalation", "require_handoff") {
			v.add(ErrModelEscalationPolicyInvalidaV0, field+".effect")
		}
		if (rule.Source == "unavailable" || rule.Freshness == "obsolete") && rule.Effect == "allow" {
			v.add(ErrModelEscalationPolicyInvalidaV0, field+".effect")
		}
		v.validateActionSet(field+".preferred_actions", rule.PreferredActions, 1, 6)
		v.optionalSafeText(field+".rationale", rule.Rationale)
	}
}

func (v *modelEscalationPolicyValidatorV0) validateLocalityRules(rules []ModelEscalationLocalityRuleV0) {
	v.requireCount("locality_rules", len(rules), 1, 32)
	for i, rule := range rules {
		field := fmt.Sprintf("locality_rules[%d]", i)
		v.requireOpaque(field+".rule_ref", rule.RuleRef)
		if !isOneOfV0(rule.Locality, "remote", "local", "hybrid") {
			v.add(ErrModelEscalationPolicyInvalidaV0, field+".locality")
		}
		if !isOneOfV0(rule.Effect, "eligible", "preferred", "degrade", "blocked") {
			v.add(ErrModelEscalationPolicyInvalidaV0, field+".effect")
		}
		v.optionalConfidence(field+".minimum_confidence", rule.MinimumConfidence)
		v.optionalNonNegativeInt(field+".minimum_samples", rule.MinimumSamples)
		v.validateLocalityInvariant(field, rule)
		if !isOneOfV0(rule.RequiresQuotaFreshness, "fresh", "fresh_or_stale", "any", "not_applicable") {
			v.add(ErrModelEscalationPolicyInvalidaV0, field+".requires_quota_freshness")
		}
		v.requireSafeText(field+".rationale", rule.Rationale)
	}
}

func (v *modelEscalationPolicyValidatorV0) validateXHighGate(gate ModelEscalationXHighGateV0) {
	if !gate.Allowed {
		v.add(ErrModelEscalationPolicyXHighGateV0, "xhigh_gate.allowed")
	}
	v.validateStringSet("xhigh_gate.required_risk", gate.RequiredRisk, 1, 2, "high", "critical")
	if !isOneOfV0(gate.RequiredEvidenceFreshness, "fresh", "fresh_or_stale") {
		v.add(ErrModelEscalationPolicyXHighGateV0, "xhigh_gate.required_evidence_freshness")
	}
	v.validateStringSet(
		"xhigh_gate.accepted_justifications",
		gate.AcceptedJustifications,
		1,
		8,
		"fallo_fresco",
		"riesgo_critico",
		"arquitectura_critica",
		"revision_bloqueada",
		"ambiguedad_material",
		"solicitud_humana_justificada",
	)
	if gate.MinimumEvidenceCount < 1 || gate.MinimumEvidenceCount > 8 {
		v.add(ErrModelEscalationPolicyXHighGateV0, "xhigh_gate.minimum_evidence_count")
	}
	if !isOneOfV0(gate.OutcomeWithoutGate, "degrade_to_high", "degrade_to_medium", "require_handoff", "block") {
		v.add(ErrModelEscalationPolicyXHighGateV0, "xhigh_gate.outcome_without_gate")
	}
}

func (v *modelEscalationPolicyValidatorV0) validateDegradationOrder(actions []string) {
	v.validateActionSet("degradation_order", actions, 3, 6)
}

func (v *modelEscalationPolicyValidatorV0) validateCrossInvariants(policy ModelEscalationPolicyV0) {
	if !policyAllowsXHighV0(policy) {
		return
	}
	if !policy.XHighGate.Allowed || policy.XHighGate.MinimumEvidenceCount < 1 || len(policy.XHighGate.AcceptedJustifications) == 0 {
		v.add(ErrModelEscalationPolicyXHighGateV0, "xhigh_gate")
	}
	if !policyHasFreshRaiseEvidenceV0(policy.EvidenceRules) {
		v.add(ErrModelEscalationPolicyXHighGateV0, "evidence_rules")
	}
}
