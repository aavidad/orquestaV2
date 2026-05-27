package orquestaserver

import (
	"encoding/json"
	"strings"
	"testing"

	orquestarails "orquesta/modulos/orquesta-rails"
)

func TestAuditEventV0ResumePayloadOperativoSinVolcarRequestResultadoV0(t *testing.T) {
	t.Setenv(orquestarails.SecurityModeEnvV0, orquestarails.SecurityModeProductionV0)
	event := normalizeAuditEventV0(AuditEventV0{
		Event:  "idle_self_improvement_prepare_result",
		Status: "ok",
		Error:  "fallo con token=abc en /tmp/runtime",
		Payload: map[string]interface{}{
			"request": IdleSelfImprovementRequestV0{
				RequestRef:     "request-ref-idle-001",
				FailureSummary: strings.Repeat("x", 400),
			},
			"result": IdleSelfImprovementResultV0{
				Accepted: true,
				RunRef:   "run-ref-idle-001",
				Message:  "token en /tmp/runtime",
			},
			"message": "token en /tmp/runtime",
		},
	})
	encoded, _ := json.Marshal(event)
	body := string(encoded)
	if event.Error != serverOperationalMessageRedactedV0 ||
		event.Payload["request"] != nil ||
		event.Payload["result"] != nil ||
		event.Payload["request_summary"] == nil ||
		event.Payload["result_summary"] == nil ||
		event.Payload["message"] != serverOperationalMessageRedactedV0 ||
		strings.Contains(body, "/tmp/runtime") ||
		strings.Contains(body, "token=abc") ||
		strings.Contains(body, strings.Repeat("x", 80)) {
		t.Fatalf("audit event no acotado/redactado: %s", body)
	}
}

func TestAuditEventV0ModoProgramacionConservaErrorOperativoV0(t *testing.T) {
	t.Setenv(orquestarails.SecurityModeEnvV0, orquestarails.SecurityModeProgrammingV0)
	message := "supervisor failed: open /home/alberto/Trabajo/.orquesta-control/orquesta/state/run-state/x.json: no such file or directory"

	event := normalizeAuditEventV0(AuditEventV0{
		Event:  "supervisor_tick_error",
		Status: "error",
		Error:  message,
		Payload: map[string]interface{}{
			"blocker_message": message,
		},
	})
	if event.Error != message ||
		event.Payload["blocker_message"] != message {
		t.Fatalf("audit en modo programacion debe conservar diagnostico: %+v", event)
	}
}
