package orquestacoreleases

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestAgentLeaseExpiredFromAssessmentV0ContinueNoProduceExpiracion(t *testing.T) {
	assessment := validAgentTimeoutAssessmentV0()
	assessment.Decision = AgentTimeoutDecisionContinueV0
	assessment.ReasonCode = AgentTimeoutReasonHeartbeatCurrentV0

	expired, ok, err := AgentLeaseExpiredFromAssessmentV0(assessment)
	if err != nil {
		t.Fatalf("AgentLeaseExpiredFromAssessmentV0: %v", err)
	}
	if ok {
		t.Fatalf("continue produjo expiracion candidata: %#v", expired)
	}
}

func TestAgentLeaseExpiredFromAssessmentV0NoContinueConservaContratoCompacto(t *testing.T) {
	tests := []AgentTimeoutDecisionV0{
		AgentTimeoutDecisionRetryV0,
		AgentTimeoutDecisionAskDirectorV0,
		AgentTimeoutDecisionStopAgentV0,
		AgentTimeoutDecisionMarkFailedV0,
		AgentTimeoutDecisionMarkStoppedV0,
		AgentTimeoutDecisionReplanTaskV0,
		AgentTimeoutDecisionAlertOnlyV0,
	}

	for _, decision := range tests {
		t.Run(string(decision), func(t *testing.T) {
			assessment := validAgentTimeoutAssessmentV0()
			assessment.Decision = decision

			expired, ok, err := AgentLeaseExpiredFromAssessmentV0(assessment)
			if err != nil {
				t.Fatalf("AgentLeaseExpiredFromAssessmentV0: %v", err)
			}
			if !ok {
				t.Fatalf("decision %q no produjo expiracion candidata", decision)
			}
			if expired.RunRef != assessment.RunRef ||
				expired.AgentRequestID != assessment.AgentRequestID ||
				expired.LeaseRef != assessment.LeaseRef {
				t.Fatalf("refs no conservadas: %#v desde %#v", expired, assessment)
			}
			if expired.ObservedAt != assessment.NowObservedAt {
				t.Fatalf("observed_at=%q, want %q", expired.ObservedAt, assessment.NowObservedAt)
			}
			if expired.RecommendedAction != decision {
				t.Fatalf("recommended_action=%q, want %q", expired.RecommendedAction, decision)
			}
			if len(expired.EvidenceRefs) != len(assessment.EvidenceRefs) {
				t.Fatalf("evidence_refs=%#v, want %#v", expired.EvidenceRefs, assessment.EvidenceRefs)
			}
			if !expired.Valid() {
				t.Fatalf("expiracion invalida: %#v", expired.Validate())
			}
		})
	}
}

func TestAgentLeaseExpiredFromAssessmentV0EsIdempotenteYClonaEvidencia(t *testing.T) {
	assessment := validAgentTimeoutAssessmentV0()

	first, ok, err := AgentLeaseExpiredFromAssessmentV0(assessment)
	if err != nil {
		t.Fatalf("AgentLeaseExpiredFromAssessmentV0 first: %v", err)
	}
	if !ok {
		t.Fatal("assessment no produjo expiracion candidata")
	}
	second, ok, err := AgentLeaseExpiredFromAssessmentV0(assessment)
	if err != nil {
		t.Fatalf("AgentLeaseExpiredFromAssessmentV0 second: %v", err)
	}
	if !ok {
		t.Fatal("assessment no produjo expiracion candidata en segunda llamada")
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("traduccion no idempotente:\nfirst=%#v\nsecond=%#v", first, second)
	}

	first.EvidenceRefs[0] = "evidence-mutated-lse-003"
	if assessment.EvidenceRefs[0] != "evidence-lse-003" {
		t.Fatalf("evento comparte evidence_refs con assessment: %#v", assessment.EvidenceRefs)
	}

	assessment.EvidenceRefs[1] = "evidence-assessment-mutated-lse-003"
	if second.EvidenceRefs[1] != "heartbeat-lse-001" {
		t.Fatalf("assessment comparte evidence_refs con evento: %#v", second.EvidenceRefs)
	}
}

func TestAgentLeaseExpiredV0JSONTieneSoloCamposCompactos(t *testing.T) {
	expired := validAgentLeaseExpiredV0()

	raw, err := json.Marshal(expired)
	if err != nil {
		t.Fatalf("marshal expired: %v", err)
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	wantKeys := map[string]bool{
		"run_ref":            true,
		"agent_request_id":   true,
		"lease_ref":          true,
		"reason_code":        true,
		"observed_at":        true,
		"recommended_action": true,
		"evidence_refs":      true,
	}
	if len(payload) != len(wantKeys) {
		t.Fatalf("campos=%v, want solo %v", mapKeysAgentLeaseTestV0(payload), mapBoolKeysAgentLeaseTestV0(wantKeys))
	}
	for key := range payload {
		if !wantKeys[key] {
			t.Fatalf("campo no compacto en AgentLeaseExpiredV0: %q", key)
		}
	}
}

func TestAgentLeaseExpiredFromAssessmentV0StoppedFailedSonSenalesTerminalesSinProceso(t *testing.T) {
	tests := []struct {
		name     string
		status   AgentHeartbeatStatusV0
		decision AgentTimeoutDecisionV0
	}{
		{name: "stopped", status: AgentHeartbeatStoppedV0, decision: AgentTimeoutDecisionMarkStoppedV0},
		{name: "failed", status: AgentHeartbeatFailedV0, decision: AgentTimeoutDecisionMarkFailedV0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := validAgentLeaseEvaluationInputV0()
			input.LastHeartbeat.Status = test.status

			assessment, err := EvaluateAgentLeaseV0(input)
			if err != nil {
				t.Fatalf("EvaluateAgentLeaseV0: %v", err)
			}
			expired, ok, err := AgentLeaseExpiredFromAssessmentV0(assessment)
			if err != nil {
				t.Fatalf("AgentLeaseExpiredFromAssessmentV0: %v", err)
			}
			if !ok {
				t.Fatalf("senal terminal %q no produjo expiracion candidata", test.name)
			}
			if expired.RecommendedAction != test.decision {
				t.Fatalf("recommended_action=%q, want %q", expired.RecommendedAction, test.decision)
			}
		})
	}
}

func TestAgentLeaseExpiredV0RechazaContinueYCamposNoContratados(t *testing.T) {
	expired := validAgentLeaseExpiredV0()
	expired.RecommendedAction = AgentTimeoutDecisionContinueV0
	requireAgentLeaseIssueCodeV0(t, expired.Validate(), ErrAgentLeaseDecisionInvalidaV0)

	raw := []byte(`{
		"run_ref":"run-lse-001",
		"agent_request_id":"agent-request-lse-001",
		"lease_ref":"lease-lse-001",
		"reason_code":"heartbeat_timeout",
		"observed_at":"2026-05-06T10:17:01Z",
		"recommended_action":"stop_agent",
		"process_ref":"process-real-001"
	}`)
	_, err := DecodeAgentLeaseExpiredV0(raw)
	requireAgentLeaseErrorCodeV0(t, err, ErrAgentLeaseJSONInvalidoV0)
}

func TestAgentLeaseExpiredV0DistingueCamposOperativosYSecretosReales(t *testing.T) {
	tests := []string{
		"DB",
		"runtime_provider",
		"provider",
		"HOME",
		"oauth",
		"modelo",
		"secrets",
	}

	for _, field := range tests {
		t.Run(field, func(t *testing.T) {
			raw := []byte(`{
				"run_ref":"run-lse-001",
				"agent_request_id":"agent-request-lse-001",
				"lease_ref":"lease-lse-001",
				"reason_code":"heartbeat_timeout",
				"observed_at":"2026-05-06T10:17:01Z",
				"recommended_action":"stop_agent",
				"` + field + `":"detalle-real-prohibido"
			}`)
			_, err := DecodeAgentLeaseExpiredV0(raw)
			requireAgentLeaseErrorCodeV0(t, err, ErrAgentLeaseJSONInvalidoV0)
		})
	}

	raw := []byte(`{
		"run_ref":"run-lse-001",
		"agent_request_id":"agent-request-lse-001",
		"lease_ref":"lease-lse-001",
		"reason_code":"heartbeat_timeout",
		"observed_at":"2026-05-06T10:17:01Z",
		"recommended_action":"stop_agent",
		"client_secret":"valor"
	}`)
	_, err := DecodeAgentLeaseExpiredV0(raw)
	requireAgentLeaseErrorCodeV0(t, err, ErrAgentLeaseDetalleProhibidoV0)

	expired := validAgentLeaseExpiredV0()
	expired.EvidenceRefs = []string{"secret-token-ref-lse-003"}
	if issues := expired.Validate(); len(issues) > 0 {
		t.Fatalf("ref opaca con vocabulario token/secret rechazada: %#v", issues)
	}

	expired.EvidenceRefs = []string{"api_key=valor"}
	requireAgentLeaseIssueCodeV0(t, expired.Validate(), ErrAgentLeaseDetalleProhibidoV0)
}

func TestAgentLeaseExpiredV0RoundTripJSONEstricto(t *testing.T) {
	expired := validAgentLeaseExpiredV0()

	raw, err := json.Marshal(expired)
	if err != nil {
		t.Fatalf("marshal expired: %v", err)
	}
	decoded, err := DecodeAgentLeaseExpiredV0(raw)
	if err != nil {
		t.Fatalf("DecodeAgentLeaseExpiredV0: %v", err)
	}
	if decoded.ObservedAt != expired.ObservedAt || decoded.RecommendedAction != expired.RecommendedAction {
		t.Fatalf("round-trip inesperado: %#v", decoded)
	}
}

func mapKeysAgentLeaseTestV0(payload map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(payload))
	for key := range payload {
		keys = append(keys, key)
	}
	return keys
}

func mapBoolKeysAgentLeaseTestV0(payload map[string]bool) []string {
	keys := make([]string, 0, len(payload))
	for key := range payload {
		keys = append(keys, key)
	}
	return keys
}

func validAgentTimeoutAssessmentV0() AgentTimeoutAssessmentV0 {
	return AgentTimeoutAssessmentV0{
		AssessmentRef:  "assessment-lse-003",
		RunRef:         "run-lse-001",
		AgentRequestID: "agent-request-lse-001",
		LeaseRef:       "lease-lse-001",
		NowObservedAt:  "2026-05-06T10:17:01Z",
		Decision:       AgentTimeoutDecisionStopAgentV0,
		ReasonCode:     AgentTimeoutReasonHeartbeatTimeoutV0,
		EvidenceRefs:   []string{"evidence-lse-003", "heartbeat-lse-001"},
	}
}

func validAgentLeaseExpiredV0() AgentLeaseExpiredV0 {
	expired, ok, err := AgentLeaseExpiredFromAssessmentV0(validAgentTimeoutAssessmentV0())
	if err != nil {
		panic(err)
	}
	if !ok {
		panic("assessment valido no produjo expiracion")
	}
	return expired
}
