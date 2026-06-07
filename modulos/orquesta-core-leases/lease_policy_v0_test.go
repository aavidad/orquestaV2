package orquestacoreleases

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestAgentLeasePolicyV0ValidaPoliticaCompacta(t *testing.T) {
	policy := validAgentLeasePolicyV0()

	if issues := ValidateAgentLeasePolicyV0(policy); len(issues) > 0 {
		t.Fatalf("politica valida rechazada: %#v", issues)
	}
	if !policy.Valid() {
		t.Fatalf("Valid()=false para politica valida")
	}

	raw, err := json.Marshal(policy)
	if err != nil {
		t.Fatalf("marshal policy: %v", err)
	}
	decoded, err := DecodeAgentLeasePolicyV0(raw)
	if err != nil {
		t.Fatalf("decode policy valida: %v", err)
	}
	if decoded.LeasePolicyRef != policy.LeasePolicyRef || decoded.TimeoutAction != policy.TimeoutAction {
		t.Fatalf("round-trip inesperado: %#v", decoded)
	}
}

func TestAgentLeasePolicyV0RechazaTiemposYAccionesInvalidas(t *testing.T) {
	tests := []struct {
		name   string
		policy AgentLeasePolicyV0
		code   AgentLeaseIssueCodeV0
	}{
		{
			name: "launch cero",
			policy: func() AgentLeasePolicyV0 {
				policy := validAgentLeasePolicyV0()
				policy.LaunchTimeoutSeconds = 0
				return policy
			}(),
			code: ErrAgentLeaseTiempoInvalidoV0,
		},
		{
			name: "total menor que heartbeat",
			policy: func() AgentLeasePolicyV0 {
				policy := validAgentLeasePolicyV0()
				policy.TotalTimeoutSeconds = 30
				policy.HeartbeatTimeoutSeconds = 60
				return policy
			}(),
			code: ErrAgentLeaseTiempoInvalidoV0,
		},
		{
			name: "accion no contratada",
			policy: func() AgentLeasePolicyV0 {
				policy := validAgentLeasePolicyV0()
				policy.TimeoutAction = "kill_process"
				return policy
			}(),
			code: ErrAgentLeaseAccionInvalidaV0,
		},
		{
			name: "reintentos negativos",
			policy: func() AgentLeasePolicyV0 {
				policy := validAgentLeasePolicyV0()
				policy.MaxRetries = -1
				return policy
			}(),
			code: ErrAgentLeaseInvalidoV0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requireAgentLeaseIssueCodeV0(t, ValidateAgentLeasePolicyV0(test.policy), test.code)
		})
	}
}

func TestAgentHeartbeatReportV0ValidaHeartbeatCompacto(t *testing.T) {
	report := validAgentHeartbeatReportV0()

	if issues := report.Validate(); len(issues) > 0 {
		t.Fatalf("heartbeat valido rechazado: %#v", issues)
	}

	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal heartbeat: %v", err)
	}
	decoded, err := DecodeAgentHeartbeatReportV0(raw)
	if err != nil {
		t.Fatalf("decode heartbeat valido: %v", err)
	}
	if decoded.ObservedAt != report.ObservedAt || decoded.Status != report.Status {
		t.Fatalf("round-trip inesperado: %#v", decoded)
	}
}

func TestAgentHeartbeatReportV0RechazaObservedAtYStatusInvalidos(t *testing.T) {
	report := validAgentHeartbeatReportV0()
	report.ObservedAt = "2026-02-30T10:15:00Z"
	requireAgentLeaseIssueCodeV0(t, report.Validate(), ErrAgentHeartbeatObservedAtV0)

	report = validAgentHeartbeatReportV0()
	report.ObservedAt = "2026-05-06T25:15:00Z"
	requireAgentLeaseIssueCodeV0(t, report.Validate(), ErrAgentHeartbeatObservedAtV0)

	report = validAgentHeartbeatReportV0()
	report.Status = "running"
	requireAgentLeaseIssueCodeV0(t, report.Validate(), ErrAgentHeartbeatStatusInvalidoV0)
}

func TestAgentLeaseDTOsPermitenVocabularioOperativoOpaco(t *testing.T) {
	policy := validAgentLeasePolicyV0()
	policy.EvidenceRefs = []string{
		"runtime-provider-model-db-sql-home-ref-001",
		"progress-report-ref-runtime-provider-001",
	}

	if issues := ValidateAgentLeasePolicyV0(policy); len(issues) != 0 {
		t.Fatalf("refs operativas opacas rechazadas: %#v", issues)
	}
}

func TestAgentLeaseDTOsRechazanDetallesSensiblesEfectivos(t *testing.T) {
	policy := validAgentLeasePolicyV0()
	forbiddenRefs := []string{
		"api_key=valor",
		"client_secret=valor",
		"prompt=raw",
		"postgres://user:pass@host/db",
	}

	for _, ref := range forbiddenRefs {
		t.Run(ref, func(t *testing.T) {
			policy.EvidenceRefs = []string{ref}
			requireAgentLeaseIssueCodeV0(t, ValidateAgentLeasePolicyV0(policy), ErrAgentLeaseDetalleProhibidoV0)
		})
	}
}

func TestAgentLeaseDTOsNoConviertenDiagnosticoLocalEnDetalleProhibido(t *testing.T) {
	policy := validAgentLeasePolicyV0()
	refs := []string{
		"home-$HOME",
		"pid=1234",
	}

	for _, ref := range refs {
		t.Run(ref, func(t *testing.T) {
			policy.EvidenceRefs = []string{ref}
			issues := ValidateAgentLeasePolicyV0(policy)
			requireAgentLeaseIssueCodeV0(t, issues, ErrAgentLeaseReferenciaNoOpacaV0)
			rejectAgentLeaseIssueCodeV0(t, issues, ErrAgentLeaseDetalleProhibidoV0)
		})
	}
}

func TestDecodeAgentHeartbeatReportV0DistingueJSONInvalidoYDetalleSensible(t *testing.T) {
	_, err := DecodeAgentHeartbeatReportV0([]byte(`{
		"heartbeat_ref":"heartbeat-lse-001",
		"run_ref":"run-lse-001",
		"agent_request_id":"agent-request-lse-001",
		"lease_ref":"lease-lse-001",
		"observed_at":"2026-05-06T10:15:00Z",
		"status":"alive",
		"campo_no_contratado":true
	}`))
	requireAgentLeaseErrorCodeV0(t, err, ErrAgentLeaseJSONInvalidoV0)

	_, err = DecodeAgentHeartbeatReportV0([]byte(`{
		"heartbeat_ref":"heartbeat-lse-001",
		"run_ref":"run-lse-001",
		"agent_request_id":"agent-request-lse-001",
		"lease_ref":"lease-lse-001",
		"observed_at":"2026-05-06T10:15:00Z",
		"status":"alive",
		"pid":1234
	}`))
	requireAgentLeaseErrorCodeV0(t, err, ErrAgentLeaseJSONInvalidoV0)

	_, err = DecodeAgentHeartbeatReportV0([]byte(`{
		"heartbeat_ref":"heartbeat-lse-001",
		"run_ref":"run-lse-001",
		"agent_request_id":"agent-request-lse-001",
		"lease_ref":"lease-lse-001",
		"observed_at":"2026-05-06T10:15:00Z",
		"status":"alive",
		"api_key":"valor"
	}`))
	requireAgentLeaseErrorCodeV0(t, err, ErrAgentLeaseDetalleProhibidoV0)
}

func validAgentLeasePolicyV0() AgentLeasePolicyV0 {
	return AgentLeasePolicyV0{
		LeasePolicyRef:          "lease-policy-lse-001",
		LaunchTimeoutSeconds:    120,
		HeartbeatTimeoutSeconds: 60,
		TotalTimeoutSeconds:     900,
		TimeoutAction:           AgentLeaseTimeoutAskDirectorV0,
		MaxRetries:              2,
		EvidenceRefs:            []string{"evidence-lse-001"},
	}
}

func validAgentHeartbeatReportV0() AgentHeartbeatReportV0 {
	return AgentHeartbeatReportV0{
		HeartbeatRef:      "heartbeat-lse-001",
		RunRef:            "run-lse-001",
		AgentRequestID:    "agent-request-lse-001",
		LeaseRef:          "lease-lse-001",
		ObservedAt:        "2026-05-06T10:15:00Z",
		Status:            AgentHeartbeatProgressingV0,
		ProgressReportRef: "progress-report-lse-001",
		EvidenceRefs:      []string{"evidence-lse-002"},
	}
}

func requireAgentLeaseErrorCodeV0(t *testing.T, err error, code AgentLeaseIssueCodeV0) {
	t.Helper()
	if err == nil {
		t.Fatalf("se esperaba error %q", code)
	}
	var validationErr AgentLeaseValidationErrorV0
	if !errors.As(err, &validationErr) {
		t.Fatalf("error no es AgentLeaseValidationErrorV0: %T %v", err, err)
	}
	requireAgentLeaseIssueCodeV0(t, validationErr.Issues, code)
}

func requireAgentLeaseIssueCodeV0(t *testing.T, issues []AgentLeaseIssueV0, code AgentLeaseIssueCodeV0) {
	t.Helper()
	for _, issue := range issues {
		if issue.Code == code {
			return
		}
	}
	t.Fatalf("no se encontro codigo %q en %#v", code, issues)
}

func rejectAgentLeaseIssueCodeV0(t *testing.T, issues []AgentLeaseIssueV0, code AgentLeaseIssueCodeV0) {
	t.Helper()
	for _, issue := range issues {
		if issue.Code == code {
			t.Fatalf("no se esperaba codigo %q en %#v", code, issues)
		}
	}
}
