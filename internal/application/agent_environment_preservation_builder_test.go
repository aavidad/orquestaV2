package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

func TestAgentEnvironmentPreservationBuilderPublishesExactManifestAndCanonicalInventory(t *testing.T) {
	fixture := newAgentEnvironmentLifecycleFixture(t)
	receipt := agentEnvironmentPreservationBuilderReceipt(t, fixture)
	artifacts := newMemoryArtifactStore()
	builder, err := NewAgentEnvironmentPreservationBuilder(artifacts)
	if err != nil {
		t.Fatal(err)
	}
	registeredAt := receipt.ConfirmedAt.Add(time.Second)
	fact, err := builder(context.Background(), receipt, fixture.record, registeredAt)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidarCausalidadPreservacionEntornoAgente(fact, fixture.record); err != nil {
		t.Fatal(err)
	}
	if fact.AlcanceEspacio != PreservacionEntornoSinEspacioTrabajo ||
		fact.ManifiestoFisicoRef != receipt.Manifest.Ref ||
		fact.ManifiestoFisicoDigest != receipt.Manifest.SHA256 ||
		fact.Resultado.PaqueteDigest != receipt.Manifest.SHA256 ||
		fact.Resultado.ConfiguracionDigest != receipt.Manifest.Causality.ProfileSHA256 ||
		fact.Resultado.RootFSDigest != receipt.Manifest.Causality.InitramfsSHA256 {
		t.Fatalf("physical binding was not preserved exactly: %+v", fact)
	}
	storedManifest, err := artifacts.Get(context.Background(), fact.Resultado.PaqueteRef,
		int64(receipt.Manifest.ContentBytes))
	if err != nil || !bytes.Equal(storedManifest.Content, receipt.Manifest.Content) {
		t.Fatalf("manifest content changed: artifact=%+v err=%v", storedManifest, err)
	}
	storedInventory, err := artifacts.Get(context.Background(), fact.Resultado.InventarioRef,
		int64(artifacts.content[fact.Resultado.InventarioRef].Size))
	if err != nil {
		t.Fatal(err)
	}
	var inventory agentEnvironmentPreservationInventory
	if err := json.Unmarshal(storedInventory.Content, &inventory); err != nil {
		t.Fatal(err)
	}
	if inventory.SchemaVersion != 1 || inventory.Package.ArtifactRef != fact.Resultado.PaqueteRef.String() ||
		inventory.Package.SHA256 != receipt.Manifest.SHA256 || inventory.Execution.Fence != 7 ||
		inventory.Causality != (agentEnvironmentPreservationCausality{
			PlanSHA256:      receipt.Manifest.Causality.PlanSHA256,
			GrantSHA256:     receipt.Manifest.Causality.GrantSHA256,
			KernelSHA256:    receipt.Manifest.Causality.KernelSHA256,
			InitramfsSHA256: receipt.Manifest.Causality.InitramfsSHA256,
			ProfileSHA256:   receipt.Manifest.Causality.ProfileSHA256,
		}) || inventory.Workspace != nil || inventory.Change != nil {
		t.Fatalf("inventory omitted or invented facts: %+v", inventory)
	}
	replayed, err := builder(context.Background(), receipt, fixture.record, registeredAt.Add(time.Minute))
	if err != nil || replayed.Ref != fact.Ref || replayed.Resultado.PaqueteRef != fact.Resultado.PaqueteRef ||
		replayed.Resultado.InventarioRef != fact.Resultado.InventarioRef || len(artifacts.content) != 2 {
		t.Fatalf("content-addressed replay diverged: first=%+v replay=%+v blobs=%d err=%v",
			fact, replayed, len(artifacts.content), err)
	}
}

func TestAgentEnvironmentPreservationBuilderIncludesExactWorkspaceAndChangeFacts(t *testing.T) {
	fixture := newAgentEnvironmentLifecycleFixture(t)
	receipt := agentEnvironmentPreservationBuilderReceipt(t, fixture)
	binding, change := attachAgentEnvironmentPreservationWorkspace(t, &fixture, receipt.ConfirmedAt)
	artifacts := newMemoryArtifactStore()
	builder, err := NewAgentEnvironmentPreservationBuilder(artifacts)
	if err != nil {
		t.Fatal(err)
	}
	fact, err := builder(context.Background(), receipt, fixture.record, receipt.ConfirmedAt.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if fact.AlcanceEspacio != PreservacionEntornoConEspacioTrabajo ||
		fact.EspacioTrabajoRef != binding.Ref || fact.DigestBindingEspacio != binding.Digest() ||
		fact.CambioRef != change.Ref || fact.DigestCambio != change.Digest() {
		t.Fatalf("workspace/change facts diverged: %+v", fact)
	}
	stored := artifacts.content[fact.Resultado.InventarioRef]
	var inventory agentEnvironmentPreservationInventory
	if err := json.Unmarshal(stored.Content, &inventory); err != nil {
		t.Fatal(err)
	}
	if inventory.Workspace == nil || inventory.Workspace.Ref != binding.Ref.String() ||
		inventory.Workspace.BindingSHA256 != binding.Digest() || inventory.Change == nil ||
		inventory.Change.Ref != change.Ref.String() || inventory.Change.ChangeSHA256 != change.Digest() {
		t.Fatalf("inventory lost exact workspace/change facts: %+v", inventory)
	}
}

func TestAgentEnvironmentPreservationBuilderRejectsMissingProfileBeforeCAS(t *testing.T) {
	fixture := newAgentEnvironmentLifecycleFixture(t)
	receipt := agentEnvironmentPreservationBuilderReceipt(t, fixture)
	receipt.Manifest.Causality.ProfileSHA256 = ""
	artifacts := newMemoryArtifactStore()
	builder, err := NewAgentEnvironmentPreservationBuilder(artifacts)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := builder(context.Background(), receipt, fixture.record,
		receipt.ConfirmedAt.Add(time.Second)); !errorsIsAgentEnvironmentPreservationBuilderInvalid(err) ||
		len(artifacts.content) != 0 {
		t.Fatalf("missing physical configuration crossed CAS: blobs=%d err=%v", len(artifacts.content), err)
	}
}

func errorsIsAgentEnvironmentPreservationBuilderInvalid(err error) bool {
	return err == errAgentEnvironmentPreservationBuilderInvalid
}

func agentEnvironmentPreservationBuilderReceipt(
	t *testing.T,
	fixture agentEnvironmentLifecycleFixture,
) ports.AgentPreserveReceipt {
	t.Helper()
	token, err := ports.NewAgentPhysicalToken(fixture.initial.Subject.ExternalRef)
	if err != nil {
		t.Fatal(err)
	}
	previousRevision, err := ports.NewAgentPhysicalRevision("2")
	if err != nil {
		t.Fatal(err)
	}
	nextRevision, err := ports.NewAgentPhysicalRevision("3")
	if err != nil {
		t.Fatal(err)
	}
	workRevision, err := ports.NewAgentPhysicalRevision("4")
	if err != nil {
		t.Fatal(err)
	}
	fence, err := ports.NewAgentPhysicalFence("7")
	if err != nil {
		t.Fatal(err)
	}
	content := []byte(`{"protocolo":"agentmicrovm.preservacion.v1","contexto":{},"artefactos":[]}`)
	digest := sha256.Sum256(content)
	digestText := hex.EncodeToString(digest[:])
	sealedAt := fixture.now.Add(10 * time.Second)
	causalDigest := func(marker string) string { return strings.Repeat(marker, sha256.Size*2) }
	return ports.AgentPreserveReceipt{
		Subject: fixture.initial.Subject,
		PreviousToken: ports.AgentEnvironmentLifecycleToken{
			PhysicalToken: token, Revision: previousRevision, Fence: fence,
			State: ports.AgentEnvironmentQuiesced,
		},
		NextToken: ports.AgentEnvironmentLifecycleToken{
			PhysicalToken: token, Revision: nextRevision, Fence: fence,
			State: ports.AgentEnvironmentPreserved,
		},
		IdempotencyKey: "idempotency:preserve:builder",
		Manifest: ports.AgentPhysicalPreservationManifest{
			Ref: digestText, SHA256: digestText, Content: content, ContentBytes: uint64(len(content)),
			WorkRevision: workRevision,
			Causality: ports.AgentPhysicalPreservationCausality{
				PlanSHA256: causalDigest("a"), GrantSHA256: causalDigest("b"),
				KernelSHA256: causalDigest("c"), InitramfsSHA256: causalDigest("d"),
				ProfileSHA256: causalDigest("e"),
			},
			SealedAt: sealedAt,
		},
		ReceiptRef: digestText, ConfirmedAt: sealedAt,
	}
}

func attachAgentEnvironmentPreservationWorkspace(
	t *testing.T,
	fixture *agentEnvironmentLifecycleFixture,
	at time.Time,
) (WorkspaceBinding, ChangeSet) {
	t.Helper()
	workspaceRef, err := ports.NewExecutionWorkspaceRef("execution-workspace:preservation-builder")
	if err != nil {
		t.Fatal(err)
	}
	repositoryRef, err := identity.NewRepositoryRef("repository:preservation-builder")
	if err != nil {
		t.Fatal(err)
	}
	changeRef, err := ports.NewChangeSetRef("change-set:preservation-builder")
	if err != nil {
		t.Fatal(err)
	}
	writeSet := []string{"internal/application/agent_environment.go"}
	writeSetDigest := ports.WorkspaceWriteSetDigest(writeSet)
	binding := WorkspaceBinding{
		Ref: workspaceRef, PrincipalRef: fixture.record.RequestedBy,
		ActorRef: fixture.record.Goal.Actor(), ProjectRef: fixture.record.Goal.Project(), RepositoryRef: repositoryRef,
		GoalRef: fixture.initial.Subject.GoalRef, WorkItemRef: fixture.initial.Subject.WorkItemRef,
		ExecutionRef: fixture.initial.Subject.ExecutionRef, ExecutionAttempt: fixture.initial.Subject.ExecutionAttempt,
		PlanGeneration:    fixture.initial.Subject.PlanGeneration,
		AppSpecGeneration: fixture.initial.Subject.AppSpecGeneration, SpecHash: fixture.initial.Subject.SpecHash,
		WriteSet: writeSet, WriteSetDigest: writeSetDigest, TargetRef: "refs/heads/main",
		BaseOID: testGitOID('a'), ObjectFormat: ports.GitObjectFormatSHA1,
		AdapterRef:       "workspace-adapter:preservation-builder",
		EffectIntentRef:  "effect-intent:workspace:preservation-builder",
		EffectAttemptRef: "effect-attempt:workspace:preservation-builder", EffectFence: 8,
		ReceiptRef: "effect-receipt:workspace:preservation-builder", PreparedAt: at.Add(-time.Second),
	}
	if err := ValidateWorkspaceBinding(binding); err != nil {
		t.Fatal(err)
	}
	change := ChangeSet{
		Ref: changeRef, WorkspaceRef: workspaceRef, PrincipalRef: binding.PrincipalRef,
		ActorRef: binding.ActorRef, ProjectRef: binding.ProjectRef, RepositoryRef: repositoryRef,
		GoalRef: binding.GoalRef, WorkItemRef: binding.WorkItemRef, ExecutionRef: binding.ExecutionRef,
		ExecutionAttempt: binding.ExecutionAttempt, PlanGeneration: binding.PlanGeneration,
		AppSpecGeneration: binding.AppSpecGeneration, SpecHash: binding.SpecHash,
		BaseOID: binding.BaseOID, ParentOID: binding.BaseOID, HeadOID: testGitOID('b'), TreeOID: testGitOID('c'),
		ObjectFormat: binding.ObjectFormat, DiffDigest: testDigest("preservation-builder-diff"),
		ChangedPaths: writeSet, WriteSet: writeSet, WriteSetDigest: writeSetDigest,
		EffectIntentRef:  "effect-intent:commit:preservation-builder",
		EffectAttemptRef: "effect-attempt:commit:preservation-builder", EffectFence: 9,
		IdempotencyKey: "idempotency:commit:preservation-builder",
		AdapterRef:     "version-control:preservation-builder",
		ReceiptRef:     "effect-receipt:commit:preservation-builder", CommittedAt: at,
	}
	if err := ValidateChangeSet(change); err != nil {
		t.Fatal(err)
	}
	fixture.record.WorkspaceBindings = append(fixture.record.WorkspaceBindings, binding)
	fixture.record.ChangeSets = append(fixture.record.ChangeSets, change)
	return binding, change
}
