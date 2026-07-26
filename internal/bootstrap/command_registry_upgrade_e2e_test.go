package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"

	commandcore "orquesta/internal/commands"
	"orquesta/internal/ports"
)

func TestHistoricalRegistryAdmissionReplaysThroughCurrentDispatcherAndSQLite(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	runtime, err := Build(ctx, Options{
		ConfigPath: writeTestConfig(t, root), Version: "registry-upgrade-e2e",
		AgentFactory: countingFactory(new(atomic.Int64)),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { shutdownRuntime(t, runtime) })

	principal, hierarchy, err := localIdentityComposition(runtime.config)
	if err != nil {
		t.Fatal(err)
	}
	definition := bootstrapCommandDefinition(t, "orquesta.goals.create")
	payload, err := json.Marshal(map[string]any{
		"statement": "build registry upgrade evidence", "confirm": true,
	})
	if err != nil {
		t.Fatal(err)
	}
	invocation := commandcore.Invocation{
		CommandID: definition.ID, CommandVersion: definition.Version,
		RequestRef: "request:registry-upgrade-e2e", ProjectRef: hierarchy.ProjectRef().String(),
		Principal: principal, Payload: payload,
	}
	historical := bootstrapHistoricalCommandAdmission(definition, invocation, payload)
	admitted, err := runtime.repository.Begin(ctx, historical)
	if err != nil || !admitted.AdmissionCreated {
		t.Fatalf("historical admission=%+v err=%v", admitted, err)
	}

	first := runtime.dispatcher.Dispatch(ctx, invocation)
	replayed := runtime.dispatcher.Dispatch(ctx, invocation)
	if first.Failure != nil || replayed.Failure != nil ||
		first.AuditRef != historical.Ref || replayed.AuditRef != historical.Ref ||
		!bytes.Equal(first.Data, replayed.Data) {
		t.Fatalf("first=%+v replay=%+v", first, replayed)
	}
	status, err := runtime.orchestrator.Status(ctx, testRuntimeAccess(t, runtime))
	if err != nil || status.Goals != 1 {
		t.Fatalf("status=%+v err=%v", status, err)
	}

	mutated := definition
	mutated.DescriptionKey = "command.goals.create.semantic-change"
	semanticChange := historical
	semanticChange.RegistryDigest = bootstrapDefinitionIdentityDigest(t, mutated)
	if semanticChange.RegistryDigest == historical.RegistryDigest {
		t.Fatalf("semantic mutation retained historical digest %q", semanticChange.RegistryDigest)
	}
	if _, err := runtime.repository.Begin(ctx, semanticChange); !errors.Is(err, ports.ErrCommandAuditConflict) {
		t.Fatalf("semantic definition replay err=%v", err)
	}
}

func bootstrapCommandDefinition(t *testing.T, commandID string) commandcore.Definition {
	t.Helper()
	for _, definition := range commandcore.CanonicalDefinitions() {
		if definition.ID == commandID {
			return definition
		}
	}
	t.Fatalf("command definition %q not found", commandID)
	return commandcore.Definition{}
}

func bootstrapHistoricalCommandAdmission(
	definition commandcore.Definition,
	invocation commandcore.Invocation,
	payload json.RawMessage,
) commandcore.CommandAuditRecord {
	return commandcore.CommandAuditRecord{
		Ref: "command-audit:" + bootstrapCommandDigestFields(
			invocation.Principal.Ref.String(), invocation.ProjectRef,
			definition.ID, definition.Version, invocation.RequestRef,
		),
		CommandID: definition.ID, CommandVersion: definition.Version,
		RegistryDigest: "sha256:56e48ffa327e9b593628ca07cc025ed073ad28cb529d25d11dde4a7440b94033",
		SchemaDigest: bootstrapCommandDigestFields(
			string(definition.InputSchema), string(definition.OutputSchema),
		),
		RequestRef: invocation.RequestRef, InputDigest: bootstrapCommandDigestFields(string(payload)),
		PrincipalRef: invocation.Principal.Ref.String(), ProjectRef: invocation.ProjectRef,
		ReplayMode: definition.ReplayMode, Status: "admitted",
	}
}

func bootstrapCommandDigestFields(fields ...string) string {
	digest := sha256.New()
	var size [8]byte
	for _, field := range fields {
		binary.BigEndian.PutUint64(size[:], uint64(len(field)))
		_, _ = digest.Write(size[:])
		_, _ = digest.Write([]byte(field))
	}
	return hex.EncodeToString(digest.Sum(nil))
}

func bootstrapDefinitionIdentityDigest(t *testing.T, definition commandcore.Definition) string {
	t.Helper()
	encoded, err := json.Marshal(definition)
	if err != nil {
		t.Fatal(err)
	}
	var value any
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		t.Fatal(err)
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:])
}
