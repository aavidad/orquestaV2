package orquestacapacity

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

const ModelEscalationPolicySchemaVersionV0 = "model_escalation_policy.v0"

type ModelEscalationPolicyErrorCodeV0 string

const (
	ErrModelEscalationPolicyJSONInvalidoV0      ModelEscalationPolicyErrorCodeV0 = "model_escalation_policy_json_invalido"
	ErrModelEscalationPolicySchemaNoSoportadoV0 ModelEscalationPolicyErrorCodeV0 = "model_escalation_policy_schema_no_soportado"
	ErrModelEscalationPolicyInvalidaV0          ModelEscalationPolicyErrorCodeV0 = "model_escalation_policy_invalida"
	ErrModelEscalationPolicyReferenciaNoOpacaV0 ModelEscalationPolicyErrorCodeV0 = "referencia_no_opaca"
	ErrModelEscalationPolicySecretoDetectadoV0  ModelEscalationPolicyErrorCodeV0 = "secreto_detectado"
	ErrModelEscalationPolicyRutaHomeRealV0      ModelEscalationPolicyErrorCodeV0 = "ruta_home_real_detectada"
	ErrModelEscalationPolicyTablaRolModeloV0    ModelEscalationPolicyErrorCodeV0 = "tabla_rol_modelo_detectada"
	ErrModelEscalationPolicyXHighGateV0         ModelEscalationPolicyErrorCodeV0 = "evidencia_insuficiente_para_xhigh"
)

type ModelEscalationPolicyV0 struct {
	SchemaVersion    string                             `json:"schema_version"`
	PolicyRef        string                             `json:"policy_ref"`
	PhaseRules       []ModelEscalationPhaseRuleV0       `json:"phase_rules"`
	RiskQualityRules []ModelEscalationRiskQualityRuleV0 `json:"risk_quality_rules"`
	EvidenceRules    []ModelEscalationEvidenceRuleV0    `json:"evidence_rules"`
	QuotaRules       []ModelEscalationQuotaRuleV0       `json:"quota_rules"`
	LocalityRules    []ModelEscalationLocalityRuleV0    `json:"locality_rules"`
	XHighGate        ModelEscalationXHighGateV0         `json:"xhigh_gate"`
	DegradationOrder []string                           `json:"degradation_order"`
}

type ModelEscalationPhaseRuleV0 struct {
	RuleRef            string   `json:"rule_ref"`
	PerfilTarea        string   `json:"perfil_tarea"`
	Fase               string   `json:"fase"`
	BaseLevel          string   `json:"base_level"`
	MaxWithoutEvidence string   `json:"max_without_evidence"`
	AllowedLevels      []string `json:"allowed_levels"`
	Triggers           []string `json:"triggers,omitempty"`
	Rationale          string   `json:"rationale"`
}

type ModelEscalationRiskQualityRuleV0 struct {
	RuleRef          string   `json:"rule_ref"`
	Riesgo           string   `json:"riesgo"`
	CalidadRequerida string   `json:"calidad_requerida"`
	Adjustment       string   `json:"adjustment"`
	AllowedLevels    []string `json:"allowed_levels"`
	RequiresEvidence bool     `json:"requires_evidence"`
}

type ModelEscalationEvidenceRuleV0 struct {
	RuleRef         string `json:"rule_ref"`
	SignalKind      string `json:"signal_kind"`
	Source          string `json:"source"`
	Freshness       string `json:"freshness"`
	Weight          string `json:"weight"`
	Effect          string `json:"effect"`
	RequiresSummary bool   `json:"requires_summary,omitempty"`
	Rationale       string `json:"rationale,omitempty"`
}

type ModelEscalationQuotaRuleV0 struct {
	RuleRef          string   `json:"rule_ref"`
	Source           string   `json:"source"`
	Freshness        string   `json:"freshness"`
	Scope            string   `json:"scope"`
	Effect           string   `json:"effect"`
	PreferredActions []string `json:"preferred_actions"`
	Rationale        string   `json:"rationale,omitempty"`
}

type ModelEscalationLocalityRuleV0 struct {
	RuleRef                  string   `json:"rule_ref"`
	Locality                 string   `json:"locality"`
	Effect                   string   `json:"effect"`
	MinimumConfidence        *float64 `json:"minimum_confidence,omitempty"`
	MinimumSamples           *int     `json:"minimum_samples,omitempty"`
	RequiresRuntimeAvailable bool     `json:"requires_runtime_available"`
	RequiresQuotaFreshness   string   `json:"requires_quota_freshness"`
	Rationale                string   `json:"rationale"`
}

type ModelEscalationXHighGateV0 struct {
	Allowed                   bool     `json:"allowed"`
	RequiredRisk              []string `json:"required_risk"`
	RequiredEvidenceFreshness string   `json:"required_evidence_freshness"`
	AcceptedJustifications    []string `json:"accepted_justifications"`
	MinimumEvidenceCount      int      `json:"minimum_evidence_count"`
	HumanOverrideAllowed      bool     `json:"human_override_allowed"`
	OutcomeWithoutGate        string   `json:"outcome_without_gate"`
}

type ModelEscalationPolicyIssueV0 struct {
	Code  ModelEscalationPolicyErrorCodeV0 `json:"code"`
	Field string                           `json:"field,omitempty"`
}

type ModelEscalationPolicyValidationErrorV0 struct {
	Issues []ModelEscalationPolicyIssueV0 `json:"issues"`
}

func (err ModelEscalationPolicyValidationErrorV0) Error() string {
	if len(err.Issues) == 0 {
		return string(ErrModelEscalationPolicyInvalidaV0)
	}
	return string(err.Issues[0].Code)
}

func DecodeModelEscalationPolicyV0(data []byte) (ModelEscalationPolicyV0, error) {
	if issues := detectForbiddenModelEscalationShapeV0(data); len(issues) > 0 {
		return ModelEscalationPolicyV0{}, ModelEscalationPolicyValidationErrorV0{Issues: issues}
	}

	var policy ModelEscalationPolicyV0
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&policy); err != nil {
		return ModelEscalationPolicyV0{}, ModelEscalationPolicyValidationErrorV0{
			Issues: []ModelEscalationPolicyIssueV0{{Code: ErrModelEscalationPolicyJSONInvalidoV0}},
		}
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return ModelEscalationPolicyV0{}, ModelEscalationPolicyValidationErrorV0{
			Issues: []ModelEscalationPolicyIssueV0{{Code: ErrModelEscalationPolicyJSONInvalidoV0}},
		}
	}
	if issues := ValidateModelEscalationPolicyV0(policy); len(issues) > 0 {
		return ModelEscalationPolicyV0{}, ModelEscalationPolicyValidationErrorV0{Issues: issues}
	}
	return policy, nil
}

func ValidateModelEscalationPolicyV0(policy ModelEscalationPolicyV0) []ModelEscalationPolicyIssueV0 {
	v := modelEscalationPolicyValidatorV0{}
	v.validate(policy)
	return v.issues
}

func (policy ModelEscalationPolicyV0) Validate() []ModelEscalationPolicyIssueV0 {
	return ValidateModelEscalationPolicyV0(policy)
}

func (policy ModelEscalationPolicyV0) Valid() bool {
	return len(ValidateModelEscalationPolicyV0(policy)) == 0
}
