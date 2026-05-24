package orquestacoreworkflow

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestStartRunFromAppSpecV0MapsDraftDeterministically(t *testing.T) {
	draft := validAppSpecRunDraftV0()

	first := mustStartRunFromAppSpecV0(t, draft)
	second := mustStartRunFromAppSpecV0(t, draft)

	if first.CommandID != second.CommandID || first.CommandID != "cmd:start_run_from_appspec:v0:run-appspec-001" {
		t.Fatalf("command_id no determinista: first=%q second=%q", first.CommandID, second.CommandID)
	}
	if first.CommandType != OrchestrationCommandStartRunV0 {
		t.Fatalf("command_type=%q, want %q", first.CommandType, OrchestrationCommandStartRunV0)
	}
	if first.RunID != draft.RunID || first.IdempotencyKey != draft.IdempotencyKey {
		t.Fatalf("meta no refleja draft: %+v", first)
	}
	payload, err := decodeStartRunCommandPayloadV0(first.Payload)
	if err != nil {
		t.Fatalf("decode StartRun payload: %v", err)
	}
	if payload.ProjectRef != draft.ProjectRef || payload.AppSpecRef != draft.AppSpecRef {
		t.Fatalf("payload=%+v, want refs from draft %+v", payload, draft)
	}
}

func TestStartRunFromAppSpecV0RejectsEmptyRefs(t *testing.T) {
	cases := map[string]func(*AppSpecRunDraftV0){
		"run_id":      func(draft *AppSpecRunDraftV0) { draft.RunID = " " },
		"project_ref": func(draft *AppSpecRunDraftV0) { draft.ProjectRef = " " },
		"app_spec_ref": func(draft *AppSpecRunDraftV0) {
			draft.AppSpecRef = " "
		},
	}
	for field, mutate := range cases {
		t.Run(field, func(t *testing.T) {
			draft := validAppSpecRunDraftV0()
			mutate(&draft)

			_, err := StartRunFromAppSpecV0(draft)
			var publicErr OrchestrationCommandErrorV0
			if !errors.As(err, &publicErr) {
				t.Fatalf("expected public command error, got %T %v", err, err)
			}
			if publicErr.Code != ErrAppSpecRunDraftInvalidoV0 || publicErr.Field != field {
				t.Fatalf("error=%+v, want code=%s field=%s", publicErr, ErrAppSpecRunDraftInvalidoV0, field)
			}
		})
	}
}

func TestStartRunFromAppSpecV0RejectsSensitiveValueDetails(t *testing.T) {
	cases := map[string]func(*AppSpecRunDraftV0){
		"api_key":         func(draft *AppSpecRunDraftV0) { draft.ProjectRef = "project:api_key=valor" },
		"authorization":   func(draft *AppSpecRunDraftV0) { draft.RequestedBy = "authorization: bearer valor" },
		"client_secret":   func(draft *AppSpecRunDraftV0) { draft.CorrelationID = "corr-client_secret=valor" },
		"private_key_pem": func(draft *AppSpecRunDraftV0) { draft.AppSpecRef = "-----BEGIN PRIVATE KEY-----" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			draft := validAppSpecRunDraftV0()
			mutate(&draft)

			_, err := StartRunFromAppSpecV0(draft)
			var publicErr OrchestrationCommandErrorV0
			if !errors.As(err, &publicErr) {
				t.Fatalf("expected public command error, got %T %v", err, err)
			}
			if publicErr.Code != ErrDetalleProhibidoV0 {
				t.Fatalf("code=%q, want %q", publicErr.Code, ErrDetalleProhibidoV0)
			}
		})
	}
}

func TestStartRunFromAppSpecV0ProducesValidStartRunCommand(t *testing.T) {
	command := mustStartRunFromAppSpecV0(t, validAppSpecRunDraftV0())

	if err := ValidateOrchestrationCommandV0(command); err != nil {
		t.Fatalf("invalid StartRun command: %v", err)
	}
	result, err := HandleCommandV0(OrchestrationRunV0{}, command)
	if err != nil {
		t.Fatalf("handle StartRun command: %v", err)
	}
	assertSingleEventTypeV0(t, result, OrchestrationEventRunStartedV0)
}

func TestStartRunFromAppSpecV0SerializationStaysLocalAndCompact(t *testing.T) {
	command := mustStartRunFromAppSpecV0(t, validAppSpecRunDraftV0())
	data, err := json.Marshal(command)
	if err != nil {
		t.Fatalf("marshal command: %v", err)
	}

	serialized := strings.ToLower(string(data))
	for _, forbidden := range operationalSensitiveFragmentsForTestV0() {
		if containsForbiddenFragmentV0(serialized, forbidden) {
			t.Fatalf("serialized command contains forbidden detail %q: %s", forbidden, serialized)
		}
	}
}

func validAppSpecRunDraftV0() AppSpecRunDraftV0 {
	return AppSpecRunDraftV0{
		RunID:          "run-appspec-001",
		ProjectRef:     "project:ventas",
		AppSpecRef:     "appspec:req-001",
		RequestedBy:    "director",
		CorrelationID:  "corr-appspec-001",
		IdempotencyKey: "idem-appspec-001",
		OccurredAt:     "2026-05-04T11:00:00Z",
	}
}

func mustStartRunFromAppSpecV0(t *testing.T, draft AppSpecRunDraftV0) OrchestrationCommandV0 {
	t.Helper()
	command, err := StartRunFromAppSpecV0(draft)
	if err != nil {
		t.Fatalf("StartRunFromAppSpecV0: %v", err)
	}
	return command
}
