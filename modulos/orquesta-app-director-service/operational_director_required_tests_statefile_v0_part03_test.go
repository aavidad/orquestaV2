package orquestaappdirectorservice

import (
	"context"
	"encoding/json"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestastatefile "orquesta/modulos/orquesta-state-file"
	"testing"
)

func serviceAssertRequiredTestsGateCountsV0(
	t *testing.T,
	store *orquestastatefile.StoreV0,
	runRef string,
	wantBlockedGateEvents int,
	wantAcceptedGateEvents int,
	wantReplanEvents int,
) {
	t.Helper()
	events, err := store.LoadRunEventsV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("LoadRunEventsV0: %v", err)
	}
	var blockedGateEvents, acceptedGateEvents, replanEvents int
	for _, event := range events {
		switch event.EventType {
		case orquestacoreworkflow.OrchestrationEventQualityGateRecordedV0:
			var payload orquestacoreworkflow.QualityGateRecordedPayloadV0
			if err := json.Unmarshal(event.Payload, &payload); err != nil {
				t.Fatalf("QualityGateRecorded payload: %v", err)
			}
			switch payload.Decision {
			case orquestacoreworkflow.QualityGateDecisionBlockedV0:
				blockedGateEvents++
				if !serviceStringInSetV0(payload.IssueRefs, "required-tests-evidence-missing") {
					t.Fatalf("quality gate blocked payload=%+v", payload)
				}
			case orquestacoreworkflow.QualityGateDecisionAcceptedV0:
				acceptedGateEvents++
			default:
				t.Fatalf("quality gate payload=%+v", payload)
			}
		case orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0:
			replanEvents++
		}
	}
	if blockedGateEvents != wantBlockedGateEvents ||
		acceptedGateEvents != wantAcceptedGateEvents ||
		replanEvents != wantReplanEvents {
		t.Fatalf("blockedGateEvents=%d acceptedGateEvents=%d replanEvents=%d want=%d/%d/%d events=%+v",
			blockedGateEvents,
			acceptedGateEvents,
			replanEvents,
			wantBlockedGateEvents,
			wantAcceptedGateEvents,
			wantReplanEvents,
			events,
		)
	}
	run, err := store.LoadRunV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if len(run.QualityGates) != wantBlockedGateEvents+wantAcceptedGateEvents || len(run.ReplanDecisions) != wantReplanEvents {
		t.Fatalf("run quality_gates=%v replan_decisions=%v", run.QualityGates, run.ReplanDecisions)
	}
}
