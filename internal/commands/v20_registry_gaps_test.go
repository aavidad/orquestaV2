package commands

import (
	"context"
	"encoding/json"
	"testing"

	"orquesta/internal/application"
)

func TestCommandAuditDenialAndFailureNeverInventSuccess(t *testing.T) {
	dispatcher, api, audit := testDispatcher(t)
	cases := []struct {
		name       string
		err        error
		code       string
		wantStatus string
	}{
		{name: "denial", err: application.ErrForbidden, code: CodeForbidden, wantStatus: "rejected"},
		{name: "failure", err: context.DeadlineExceeded, code: CodeUnavailable, wantStatus: "failed"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			requestRef := "request:v20-" + test.name
			before := api.calls["Status"]
			api.statusErr = test.err
			result := invoke(t, dispatcher, "orquesta.system.status", requestRef, map[string]any{}, false)
			if result.Failure == nil || result.Failure.Code != test.code || len(result.Data) != 0 {
				t.Fatalf("result=%+v", result)
			}
			entry := audit.entries[result.AuditRef]
			terminal, found := entry.terminals[result.AuditRef+":outcome"]
			if !found || terminal.Status != test.wantStatus || terminal.ErrorCode != test.code ||
				terminal.OutputDigest != resultEnvelopeDigest(result) || !validTerminal(terminal, true) {
				t.Fatalf("terminal=%+v result=%+v", terminal, result)
			}

			api.statusErr = nil
			replay := invoke(t, dispatcher, "orquesta.system.status", requestRef, map[string]any{}, false)
			if replay.Failure == nil || replay.Failure.Code != test.code || len(replay.Data) != 0 ||
				replay.AuditRef != result.AuditRef || api.calls["Status"] != before+1 ||
				resultEnvelopeDigest(replay) != terminal.OutputDigest {
				t.Fatalf("replay=%+v calls=%d", replay, api.calls["Status"])
			}
		})
	}
}

func TestRegistrySchemasMatchSealedApplicationUseCaseContracts(t *testing.T) {
	dispatcher, api, _ := testDispatcher(t)
	payloads := canonicalPayloads()
	definitions := dispatcher.Definitions()
	if len(definitions) != len(expectedHandlerPermissions) || len(payloads) != len(definitions) {
		t.Fatalf("definitions=%d handlers=%d payloads=%d", len(definitions), len(expectedHandlerPermissions), len(payloads))
	}
	for index, definition := range definitions {
		payload, found := payloads[definition.ID]
		if !found {
			t.Errorf("%s lacks canonical application payload", definition.ID)
			continue
		}
		encoded, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := validatePayload(definition.InputSchema, encoded); err != nil {
			t.Errorf("%s input contract: %v", definition.ID, err)
			continue
		}
		result := invoke(t, dispatcher, definition.ID, "request:v20-schema:"+definition.ID, payload, definition.ExecutionBound)
		if result.Failure != nil {
			t.Errorf("%s use case failed: %+v", definition.ID, result.Failure)
			continue
		}
		if _, err := validatePayload(definition.OutputSchema, result.Data); err != nil {
			t.Errorf("%s output contract: %v data=%s", definition.ID, err, result.Data)
		}

		var object map[string]any
		if err := json.Unmarshal(encoded, &object); err == nil {
			object["v20_unknown_field"] = index
			unknown, _ := json.Marshal(object)
			if _, err := validatePayload(definition.InputSchema, unknown); err == nil {
				t.Errorf("%s input schema accepted an undeclared field", definition.ID)
			}
		}
		if api.calls[definition.Handler] != 1 {
			t.Errorf("%s handler calls=%d", definition.Handler, api.calls[definition.Handler])
		}
	}

}
