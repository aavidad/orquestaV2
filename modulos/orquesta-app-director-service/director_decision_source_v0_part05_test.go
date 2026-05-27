package orquestaappdirectorservice

import (
	"encoding/json"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	"strings"
	"testing"
)

func requireServiceDecisionRecoveryBlockV0(
	t *testing.T,
	run orquestacoreworkflow.OrchestrationRunV0,
	events []orquestacoreworkflow.OrchestrationEventV0,
	field string,
) {
	t.Helper()
	if run.Status != orquestacoreworkflow.OrchestrationRunStatusBlockedV0 {
		t.Fatalf("status=%s, want bloqueada", run.Status)
	}
	if len(run.Blockers) != 1 || !strings.HasPrefix(run.Blockers[0], "app-director-decision-") {
		t.Fatalf("blockers=%v", run.Blockers)
	}
	for _, event := range events {
		if event.EventType != orquestacoreworkflow.OrchestrationEventRunBlockedV0 {
			continue
		}
		var payload orquestacoreworkflow.RunBlockedPayloadV0
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			t.Fatalf("RunBlocked payload: %v", err)
		}
		if strings.Contains(payload.Summary, "field="+field) {
			return
		}
	}
	t.Fatalf("sin RunBlocked con field=%s: %+v", field, events)
}
