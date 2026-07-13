package orquestaautoprogramming

import (
	"errors"
	"testing"
)

func TestAutoprogrammingPrepareRunIdempotencyClaimV0CreateOnceAndConflictV0(t *testing.T) {
	envelope := AutoprogrammingPrepareRunEnvelopeV0{
		RequestID:              "request-claim-001",
		CorrelationID:          "corr-claim-001",
		IdempotencyKey:         "idem-claim-001",
		OccurredAt:             "2026-07-13T10:00:00Z",
		RequestedBy:            "operator-ref-claim-001",
		AutoprogrammingRequest: validAutoprogrammingRequestV0(nil),
	}
	command, issues := BuildAutoprogrammingIntentManifestFromPrepareRunEnvelopeV0(envelope)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	claim, issues := BuildAutoprogrammingPrepareRunIdempotencyClaimV0(envelope.IdempotencyKey, command, command)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	store := NewInMemoryAutoprogrammingIntentManifestStoreV0()
	first, err := store.CreateAutoprogrammingPrepareRunIdempotencyClaimIfAbsentV0(nil, claim)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := store.CreateAutoprogrammingPrepareRunIdempotencyClaimIfAbsentV0(nil, claim)
	if err != nil || !EqualAutoprogrammingPrepareRunIdempotencyClaimV0(first, replay) {
		t.Fatalf("replay=%+v err=%v", replay, err)
	}
	claim.CommandSHA256 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if _, err := store.CreateAutoprogrammingPrepareRunIdempotencyClaimIfAbsentV0(nil, claim); !errors.Is(err, ErrAutoprogrammingPrepareRunIdempotencyClaimConflictV0) {
		t.Fatalf("conflict err=%v", err)
	}
}

func TestBuildAutoprogrammingPrepareRunIdempotencyClaimV0LinksLegacyManifestWithoutChangingCommandHashV0(t *testing.T) {
	request := validAutoprogrammingRequestV0(nil)
	legacy, issues := BuildAutoprogrammingIntentManifestV0(request)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	envelope := AutoprogrammingPrepareRunEnvelopeV0{
		RequestID:              "request-claim-legacy-001",
		CorrelationID:          "corr-claim-legacy-001",
		IdempotencyKey:         "idem-claim-legacy-001",
		OccurredAt:             "2026-07-13T10:00:00Z",
		RequestedBy:            "operator-ref-claim-legacy-001",
		AutoprogrammingRequest: request,
	}
	command, issues := BuildAutoprogrammingIntentManifestFromPrepareRunEnvelopeV0(envelope)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	claim, issues := BuildAutoprogrammingPrepareRunIdempotencyClaimV0(envelope.IdempotencyKey, command, legacy)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	if claim.CommandSHA256 != command.RequestSHA256 || claim.ManifestSHA256 != legacy.RequestSHA256 || claim.ManifestSHA256 == claim.CommandSHA256 {
		t.Fatalf("claim=%+v command=%+v legacy=%+v", claim, command, legacy)
	}
}

func TestCanonicalAutoprogrammingPrepareRunEnvelopeV0RejectsMaterialIdentityAndLimitMutationsV0(t *testing.T) {
	base := AutoprogrammingPrepareRunEnvelopeV0{
		RequestID:              "request-envelope-validation-001",
		CorrelationID:          "corr-envelope-validation-001",
		IdempotencyKey:         "idem-envelope-validation-001",
		OccurredAt:             "2026-07-13T12:00:00+02:00",
		RequestedBy:            "operator-ref-envelope-validation-001",
		AutoprogrammingRequest: validAutoprogrammingRequestV0(nil),
	}
	canonical, issues := CanonicalAutoprogrammingPrepareRunEnvelopeV0(base)
	if len(issues) != 0 || canonical.DirectorExecutionMode != AutoprogrammingPrepareRunExecutionModeGoalFirstV0 || canonical.OccurredAt != "2026-07-13T10:00:00Z" {
		t.Fatalf("canonical=%+v issues=%+v", canonical, issues)
	}
	mutations := map[string]struct {
		mutate func(*AutoprogrammingPrepareRunEnvelopeV0)
		code   string
	}{
		"request id":  {func(value *AutoprogrammingPrepareRunEnvelopeV0) { value.RequestID = "bad\nrequest" }, "prepare_run_envelope_request_id_invalid"},
		"correlation": {func(value *AutoprogrammingPrepareRunEnvelopeV0) { value.CorrelationID = "" }, "prepare_run_envelope_correlation_id_invalid"},
		"idempotency": {func(value *AutoprogrammingPrepareRunEnvelopeV0) { value.IdempotencyKey = "" }, "prepare_run_envelope_idempotency_key_invalid"},
		"timestamp":   {func(value *AutoprogrammingPrepareRunEnvelopeV0) { value.OccurredAt = "13/07/2026" }, "prepare_run_envelope_occurred_at_invalid"},
		"actor":       {func(value *AutoprogrammingPrepareRunEnvelopeV0) { value.RequestedBy = "" }, "prepare_run_envelope_requested_by_invalid"},
		"mode":        {func(value *AutoprogrammingPrepareRunEnvelopeV0) { value.DirectorExecutionMode = "goal-first-v2" }, "prepare_run_envelope_execution_mode_invalid"},
		"limit":       {func(value *AutoprogrammingPrepareRunEnvelopeV0) { value.MaxCommands = -1 }, "prepare_run_envelope_limit_invalid"},
		"setting key": {func(value *AutoprogrammingPrepareRunEnvelopeV0) {
			value.RequiredSettings = []AutoprogrammingPrepareRunRequiredSettingV0{{Key: "bad\nkey"}}
		}, "prepare_run_required_setting_key_invalid"},
	}
	for name, mutation := range mutations {
		t.Run(name, func(t *testing.T) {
			value := base
			mutation.mutate(&value)
			_, issues := CanonicalAutoprogrammingPrepareRunEnvelopeV0(value)
			if !hasAutoprogrammingIssueCodeV0(issues, mutation.code) {
				t.Fatalf("issues=%+v want=%s", issues, mutation.code)
			}
		})
	}
}
