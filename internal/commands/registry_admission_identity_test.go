package commands

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"orquesta/internal/application"
)

type registryUpgradeApplication struct {
	*fakeApplication
	seenAmendRequests map[string]struct{}
	amendEffects      int
}

func (api *registryUpgradeApplication) Amend(
	_ context.Context,
	_ application.Access,
	request application.AmendRequest,
) (application.AmendResult, error) {
	api.called("Amend")
	_, replay := api.seenAmendRequests[request.RequestRef]
	if !replay {
		api.seenAmendRequests[request.RequestRef] = struct{}{}
		api.amendEffects++
	}
	return application.AmendResult{Created: !replay}, nil
}

func TestHistoricalGlobalRegistryDigestReplaysOnlyUnchangedDefinitionAfterAdditiveUpgrade(t *testing.T) {
	ctx := context.Background()
	api := &registryUpgradeApplication{
		fakeApplication:   newFakeApplication(),
		seenAmendRequests: make(map[string]struct{}),
	}
	audit := newMemoryAudit()
	dispatcher, err := newDispatcher(
		api, audit, APILimits{
			MaxRequestBytes: 1 << 20, MaxListLimit: 100, IntakeMaxQuestionRounds: 6,
		},
		exactTestExecutionAuthority(t),
	)
	if err != nil {
		t.Fatal(err)
	}
	definition := dispatcher.byID["orquesta.goals.amend"]
	invocation := Invocation{
		CommandID: definition.ID, CommandVersion: definition.Version,
		RequestRef: "request:historical-registry-upgrade", ProjectRef: "project:test",
		Principal: testPrincipal(t), Payload: json.RawMessage(
			`{"source_goal_ref":"goal:source","expected_source_revision":1,` +
				`"expected_source_spec_hash":"sha256:source","statement":"amend",` +
				`"reason":"historical replay","confirm":true}`,
		),
	}
	bound, failure := dispatcher.bindAuthority(ctx, definition, invocation)
	if failure != nil {
		t.Fatalf("bind failure=%+v", failure)
	}
	payload, err := validatePayload(definition.InputSchema, invocation.Payload)
	if err != nil {
		t.Fatal(err)
	}
	historical := admissionRecord(definition, invocation, bound, payload)
	if historical.RegistryDigest != historicalRegistrySourceSHA256 ||
		historical.RegistryDigest == RegistrySourceSHA256 {
		t.Fatalf("historical digest=%q current global=%q", historical.RegistryDigest, RegistrySourceSHA256)
	}
	if _, err := audit.Begin(ctx, historical); err != nil {
		t.Fatal(err)
	}

	first := dispatcher.Dispatch(ctx, invocation)
	replayed := dispatcher.Dispatch(ctx, invocation)
	if first.Failure != nil || replayed.Failure != nil ||
		first.AuditRef != historical.Ref || replayed.AuditRef != historical.Ref ||
		string(first.Data) != string(replayed.Data) ||
		api.calls["Amend"] != 2 || api.amendEffects != 1 {
		t.Fatalf("first=%+v replay=%+v calls=%v effects=%d",
			first, replayed, api.calls, api.amendEffects)
	}

	mutated := definition
	mutated.DescriptionKey = "command.goals.amend.semantic-change"
	semanticChange := admissionRecord(mutated, invocation, bound, payload)
	if semanticChange.RegistryDigest == historical.RegistryDigest {
		t.Fatalf("semantic mutation retained historical digest %q", semanticChange.RegistryDigest)
	}
	if _, err := audit.Begin(ctx, semanticChange); !errors.Is(err, ErrAuditConflict) {
		t.Fatalf("semantic mutation replay err=%v", err)
	}
}

func TestHistoricalRegistryCompatibilityLedgerMatchesEveryAccreditedDefinition(t *testing.T) {
	if len(historicalRegistryDefinitionDigests) != historicalRegistryDefinitionCount {
		t.Fatalf("historical ledger definitions=%d want=%d",
			len(historicalRegistryDefinitionDigests), historicalRegistryDefinitionCount)
	}
	matched := 0
	for _, definition := range compiledDefinitions {
		key := definition.ID + "@" + definition.Version
		want, historical := historicalRegistryDefinitionDigests[key]
		if !historical {
			if got := registryAdmissionDigest(definition); got != definitionIdentityDigest(definition) {
				t.Fatalf("%s new definition digest=%q want=%q", key, got, definitionIdentityDigest(definition))
			}
			continue
		}
		matched++
		if got := definitionIdentityDigest(definition); got != want {
			t.Fatalf("%s definition digest=%q want=%q", key, got, want)
		}
		if got := registryAdmissionDigest(definition); got != historicalRegistrySourceSHA256 {
			t.Fatalf("%s admission digest=%q", key, got)
		}
	}
	if matched != len(historicalRegistryDefinitionDigests) {
		t.Fatalf("matched historical definitions=%d ledger=%d", matched, len(historicalRegistryDefinitionDigests))
	}
}
