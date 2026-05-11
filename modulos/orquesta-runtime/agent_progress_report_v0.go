package orquestaruntime

import (
	"fmt"
	"strings"
)

type AgentProgressStatusV0 string

const (
	AgentProgressingV0  AgentProgressStatusV0 = "progressing"
	AgentStalledV0      AgentProgressStatusV0 = "stalled"
	AgentLoopDetectedV0 AgentProgressStatusV0 = "loop_detected"
	AgentStoppedV0      AgentProgressStatusV0 = "stopped"
)

type AgentProgressBudgetStatusV0 string

const (
	AgentProgressBudgetWorkingV0              AgentProgressBudgetStatusV0 = "working"
	AgentProgressBudgetStalledV0              AgentProgressBudgetStatusV0 = "stalled"
	AgentProgressBudgetOverBudgetButActiveV0  AgentProgressBudgetStatusV0 = "over_budget_but_active"
	AgentProgressBudgetOverBudgetNoActivityV0 AgentProgressBudgetStatusV0 = "over_budget_no_activity"
	AgentProgressBudgetAckCleanupV0           AgentProgressBudgetStatusV0 = "ack_registered_cleanup"
)

type AgentProgressReportErrorCodeV0 string

const (
	AgentProgressReportInvalidoV0       AgentProgressReportErrorCodeV0 = "agent_progress_report_invalido"
	AgentProgressStatusInvalidoV0       AgentProgressReportErrorCodeV0 = "agent_progress_status_invalido"
	AgentProgressReferenciaNoOpacaV0    AgentProgressReportErrorCodeV0 = "referencia_no_opaca"
	AgentProgressSecretoDetectadoV0     AgentProgressReportErrorCodeV0 = "secreto_detectado"
	AgentProgressDetalleProveedorV0     AgentProgressReportErrorCodeV0 = "detalle_proveedor_detectado"
	AgentProgressLoopCounterRequeridoV0 AgentProgressReportErrorCodeV0 = "loop_counter_requerido"
)

type AgentProgressReportV0 struct {
	ReportID               string                      `json:"report_id"`
	RunID                  string                      `json:"run_id"`
	AgentRequestID         string                      `json:"agent_request_id"`
	Status                 AgentProgressStatusV0       `json:"status"`
	BudgetStatus           AgentProgressBudgetStatusV0 `json:"budget_status,omitempty"`
	BudgetReason           string                      `json:"budget_reason,omitempty"`
	NoProgressTicks        int                         `json:"no_progress_ticks"`
	RepeatedActionCount    int                         `json:"repeated_action_count"`
	AgeSeconds             int64                       `json:"age_seconds,omitempty"`
	SecondsSinceActivity   int64                       `json:"seconds_since_activity,omitempty"`
	SecondsSinceAck        int64                       `json:"seconds_since_ack,omitempty"`
	MaxExpectedSeconds     int64                       `json:"max_expected_seconds,omitempty"`
	NoActivityLimitSeconds int64                       `json:"no_activity_limit_seconds,omitempty"`
	StartedAt              string                      `json:"started_at,omitempty"`
	LastActivityAt         string                      `json:"last_activity_at,omitempty"`
	LastAckAt              string                      `json:"last_ack_at,omitempty"`
	DecisionRequired       bool                        `json:"decision_required,omitempty"`
	Summary                string                      `json:"summary"`
	EvidenceRefs           []string                    `json:"evidence_refs,omitempty"`
}

type AgentProgressReportErrorV0 struct {
	Code       AgentProgressReportErrorCodeV0 `json:"code"`
	MessageKey string                         `json:"message_key"`
	Field      string                         `json:"field,omitempty"`
	Retryable  bool                           `json:"retryable"`
	Evidence   []string                       `json:"evidence,omitempty"`
}

func (e AgentProgressReportErrorV0) Error() string {
	if e.Field == "" {
		return string(e.Code)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Field)
}

func ValidateAgentProgressReportV0(report AgentProgressReportV0) []AgentProgressReportErrorV0 {
	v := agentProgressReportValidatorV0{}
	v.validate(report)
	return v.errors
}

func (report AgentProgressReportV0) Validate() []AgentProgressReportErrorV0 {
	return ValidateAgentProgressReportV0(report)
}

func (report AgentProgressReportV0) Valid() bool {
	return len(ValidateAgentProgressReportV0(report)) == 0
}

type agentProgressReportValidatorV0 struct {
	errors []AgentProgressReportErrorV0
}

func (v *agentProgressReportValidatorV0) validate(report AgentProgressReportV0) {
	v.requireOpaque("report_id", report.ReportID)
	v.requireOpaque("run_id", report.RunID)
	v.requireOpaque("agent_request_id", report.AgentRequestID)
	v.requireStatus("status", report.Status)
	v.requireBudgetStatus("budget_status", report.BudgetStatus)
	v.requireNonNegative("no_progress_ticks", report.NoProgressTicks)
	v.requireNonNegative("repeated_action_count", report.RepeatedActionCount)
	v.requireNonNegativeInt64("age_seconds", report.AgeSeconds)
	v.requireNonNegativeInt64("seconds_since_activity", report.SecondsSinceActivity)
	v.requireNonNegativeInt64("seconds_since_ack", report.SecondsSinceAck)
	v.requireNonNegativeInt64("max_expected_seconds", report.MaxExpectedSeconds)
	v.requireNonNegativeInt64("no_activity_limit_seconds", report.NoActivityLimitSeconds)
	if strings.TrimSpace(report.BudgetReason) != "" {
		v.requireSummary("budget_reason", report.BudgetReason)
	}
	v.requireSummary("summary", report.Summary)
	for i, ref := range report.EvidenceRefs {
		v.requireOpaque(fmt.Sprintf("evidence_refs[%d]", i), ref)
	}
	if report.Status == AgentLoopDetectedV0 &&
		report.NoProgressTicks == 0 &&
		report.RepeatedActionCount == 0 {
		v.add(AgentProgressLoopCounterRequeridoV0, "status")
	}
}

func (v *agentProgressReportValidatorV0) requireStatus(field string, status AgentProgressStatusV0) {
	if !isOneOf(string(status),
		string(AgentProgressingV0),
		string(AgentStalledV0),
		string(AgentLoopDetectedV0),
		string(AgentStoppedV0),
	) {
		v.add(AgentProgressStatusInvalidoV0, field)
	}
}

func (v *agentProgressReportValidatorV0) requireBudgetStatus(field string, status AgentProgressBudgetStatusV0) {
	if status == "" {
		return
	}
	if !isOneOf(string(status),
		string(AgentProgressBudgetWorkingV0),
		string(AgentProgressBudgetStalledV0),
		string(AgentProgressBudgetOverBudgetButActiveV0),
		string(AgentProgressBudgetOverBudgetNoActivityV0),
		string(AgentProgressBudgetAckCleanupV0),
	) {
		v.add(AgentProgressStatusInvalidoV0, field)
	}
}

func (v *agentProgressReportValidatorV0) requireOpaque(field, value string) {
	if value == "" {
		v.add(AgentProgressReportInvalidoV0, field)
		return
	}
	switch {
	case looksLikeSecret(value):
		v.add(AgentProgressSecretoDetectadoV0, field)
	case looksLikeProviderDetailV0(value):
		v.add(AgentProgressDetalleProveedorV0, field)
	case !opaqueRefPatternV0.MatchString(value):
		v.add(AgentProgressReferenciaNoOpacaV0, field)
	}
}

func (v *agentProgressReportValidatorV0) requireNonNegative(field string, value int) {
	if value < 0 {
		v.add(AgentProgressReportInvalidoV0, field)
	}
}

func (v *agentProgressReportValidatorV0) requireNonNegativeInt64(field string, value int64) {
	if value < 0 {
		v.add(AgentProgressReportInvalidoV0, field)
	}
}

func (v *agentProgressReportValidatorV0) requireSummary(field, value string) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || len(trimmed) > 1000 {
		v.add(AgentProgressReportInvalidoV0, field)
		return
	}
	switch {
	case looksLikeSecret(trimmed):
		v.add(AgentProgressSecretoDetectadoV0, field)
	case looksLikeProviderDetailV0(trimmed):
		v.add(AgentProgressDetalleProveedorV0, field)
	}
}

func (v *agentProgressReportValidatorV0) add(code AgentProgressReportErrorCodeV0, field string) {
	v.errors = append(v.errors, AgentProgressReportErrorV0{
		Code:       code,
		MessageKey: "orquesta.runtime.agent_progress_report." + string(code),
		Field:      field,
		Retryable:  false,
	})
}

func looksLikeProviderDetailV0(value string) bool {
	low := strings.ToLower(strings.TrimSpace(value))
	return strings.Contains(low, "openai") ||
		strings.Contains(low, "anthropic") ||
		strings.Contains(low, "claude") ||
		strings.Contains(low, "gpt-") ||
		strings.Contains(low, "gemini") ||
		strings.Contains(low, "oauth") ||
		strings.Contains(low, "provider") ||
		strings.Contains(low, "proveedor") ||
		strings.Contains(low, "modelo") ||
		strings.Contains(low, "home=") ||
		strings.Contains(low, "provider=") ||
		strings.Contains(low, "model=") ||
		strings.Contains(low, "://")
}
