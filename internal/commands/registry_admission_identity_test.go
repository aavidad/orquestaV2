package commands

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"orquesta/internal/adapters/state/sqlite"
	"orquesta/internal/application"
)

type registryUpgradeApplication struct {
	*fakeApplication
	seenSubmitRequests map[string]struct{}
	submitEffects      int
}

func (api *registryUpgradeApplication) Submit(
	_ context.Context,
	_ application.Access,
	request application.SubmitRequest,
) (application.SubmitResult, error) {
	api.called("Submit")
	_, replay := api.seenSubmitRequests[request.RequestRef]
	if !replay {
		api.seenSubmitRequests[request.RequestRef] = struct{}{}
		api.submitEffects++
	}
	return application.SubmitResult{Created: !replay}, nil
}

func TestHistoricalGlobalRegistryDigestReplaysOnlyUnchangedDefinitionAfterAdditiveUpgrade(t *testing.T) {
	ctx := context.Background()
	repository, err := sqlite.Open(ctx, sqlite.Options{
		Path: filepath.Join(t.TempDir(), "private", "orquesta.sqlite"), BusyTimeout: time.Second, MaxOpenConnections: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = repository.Close() })

	api := &registryUpgradeApplication{
		fakeApplication:    newFakeApplication(),
		seenSubmitRequests: make(map[string]struct{}),
	}
	dispatcher, err := newDispatcher(
		api, repository, APILimits{MaxRequestBytes: 1 << 20, MaxListLimit: 100},
		exactTestExecutionAuthority(t),
	)
	if err != nil {
		t.Fatal(err)
	}
	definition := dispatcher.byID["orquesta.goals.create"]
	invocation := Invocation{
		CommandID: definition.ID, CommandVersion: definition.Version,
		RequestRef: "request:historical-registry-upgrade", ProjectRef: "project:test",
		Principal: testPrincipal(t), Payload: json.RawMessage(`{"statement":"build","confirm":true}`),
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
	if _, err := repository.Begin(ctx, historical); err != nil {
		t.Fatal(err)
	}

	first := dispatcher.Dispatch(ctx, invocation)
	replayed := dispatcher.Dispatch(ctx, invocation)
	if first.Failure != nil || replayed.Failure != nil ||
		first.AuditRef != historical.Ref || replayed.AuditRef != historical.Ref ||
		string(first.Data) != string(replayed.Data) ||
		api.calls["Submit"] != 2 || api.submitEffects != 1 {
		t.Fatalf("first=%+v replay=%+v calls=%v effects=%d",
			first, replayed, api.calls, api.submitEffects)
	}

	mutated := definition
	mutated.DescriptionKey = "command.goals.create.semantic-change"
	semanticChange := admissionRecord(mutated, invocation, bound, payload)
	if semanticChange.RegistryDigest == historical.RegistryDigest {
		t.Fatalf("semantic mutation retained historical digest %q", semanticChange.RegistryDigest)
	}
	if _, err := repository.Begin(ctx, semanticChange); !errors.Is(err, ErrAuditConflict) {
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
