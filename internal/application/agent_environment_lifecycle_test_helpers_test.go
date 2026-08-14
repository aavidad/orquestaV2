package application

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

type agentEnvironmentLifecycleFixture struct {
	request ports.AgentLaunchRequest
	launch  ports.AgentLaunchReceipt
	record  GoalRecord
	base    ActionClaim
	initial AgentEnvironmentLifecycleSnapshot
	now     time.Time
}

func newAgentEnvironmentLifecycleFixture(t *testing.T) agentEnvironmentLifecycleFixture {
	t.Helper()
	baseFixture := newAgentLaunchRecoveryFixtureWithPreservation(t, true)
	request := baseFixture.request
	acceptedAt := baseFixture.attempt.StartedAt.Add(time.Second).UTC()
	effectConfirmedAt := acceptedAt.Add(500 * time.Millisecond)
	now := acceptedAt.Add(time.Second)
	launch := ports.AgentLaunchReceipt{
		ExecutionRef: request.ExecutionRef, GoalRef: request.GoalRef, WorkItemRef: request.WorkItemRef,
		PlanGeneration: request.PlanGeneration, AppSpecGeneration: request.AppSpecGeneration,
		ExecutionAttempt: request.ExecutionAttempt, LaunchActionFence: request.EffectAuthority.ActionFence,
		SpecHash:    request.SpecHash,
		ProviderRef: "provider:lifecycle", ModelRef: "model:lifecycle", AgentRef: "agent:lifecycle",
		ExternalRef: "external:lifecycle", IdempotencyKey: request.IdempotencyKey,
		ReceiptRef: "receipt:launch:lifecycle", AcceptedAt: acceptedAt,
		RequierePreservacionEntorno: request.RequierePreservacionEntorno,
	}
	subject := ports.AgentEnvironmentLifecycleSubject{
		ExecutionRef: launch.ExecutionRef, GoalRef: launch.GoalRef, WorkItemRef: launch.WorkItemRef,
		PlanGeneration: launch.PlanGeneration, AppSpecGeneration: launch.AppSpecGeneration,
		ExecutionAttempt: launch.ExecutionAttempt, SpecHash: launch.SpecHash,
		ProviderRef: launch.ProviderRef, ModelRef: launch.ModelRef, AgentRef: launch.AgentRef,
		ExternalRef: launch.ExternalRef,
	}
	initial, err := NewAgentEnvironmentLifecycleSnapshot(request, launch, ports.AgentEnvironmentInspectReceipt{
		Subject: subject, Token: lifecycleToken(t, launch.ExternalRef, ports.AgentEnvironmentActive, "revision:1"),
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	record := baseFixture.record
	launchEffect, err := effectReceipt(
		baseFixture.claim, baseFixture.attempt, launch.ReceiptRef,
		EffectStatusAccepted, unknownUsage(), effectConfirmedAt,
	)
	if err != nil {
		t.Fatal(err)
	}
	for index := range record.Executions {
		if record.Executions[index].Ref != launch.ExecutionRef {
			continue
		}
		record.Executions[index].ProviderRef, record.Executions[index].ModelRef = launch.ProviderRef, launch.ModelRef
		record.Executions[index].AgentRef, record.Executions[index].ExternalRef = launch.AgentRef, launch.ExternalRef
		record.Executions[index].EffectIntentRef = baseFixture.claim.Action.EffectIntent.Ref
		record.Executions[index].LaunchReceiptRef = launchEffect.Ref
		record.Executions[index].StartedAt = acceptedAt
		record.Executions[index].ProviderAcceptedAt = acceptedAt
	}
	record.EffectReceipts = append(record.EffectReceipts, launchEffect)
	record.ConsumptionReceipts = append(record.ConsumptionReceipts, ActionConsumptionReceipt{
		Kind: ActionLaunchAgent, ExecutionRef: launch.ExecutionRef, Fence: 7, Outcome: ActionConsumedCompleted,
	})
	base := baseFixture.claim
	base.Disposition, base.RecoveryEffectAttemptRef = ActionClaimDispositionNormal, ""
	base.BudgetReservationRef = ""
	base.BudgetReservation = governance.BudgetReservation{}
	if currentItem, found := record.Goal.WorkItem(subject.WorkItemRef); found {
		base.Action.WorkItemGeneration = currentItem.Revision()
	}
	return agentEnvironmentLifecycleFixture{
		request: request, launch: launch, record: record, base: base, initial: initial, now: now,
	}
}

func lifecyclePreparedPreserve(t *testing.T, fixture agentEnvironmentLifecycleFixture) (
	AgentEnvironmentLifecyclePreEffectState, ports.AgentPreserveReceipt, GoalRecord,
) {
	t.Helper()
	record := fixture.record
	quiesceClaim := lifecycleClaimFor(t, fixture.base, fixture.initial, ActionQuiesceAgent, 61, fixture.now)
	quiescePre := mustPrepareLifecycle(t, fixture.initial, quiesceClaim, fixture.now)
	appendLifecyclePreparedFacts(&record, quiescePre)
	quiesced := lifecycleToken(t, fixture.initial.Subject.ExternalRef, ports.AgentEnvironmentQuiesced, "revision:2")
	quiesceOutcome, err := RecordAgentEnvironmentQuiesceOutcome(quiescePre, ports.AgentQuiesceReceipt{
		Subject: fixture.initial.Subject, PreviousToken: fixture.initial.Token, NextToken: quiesced,
		IdempotencyKey: quiescePre.Attempt.IdempotencyKey, ReceiptRef: "receipt:quiesce:prepare-preserve",
		ConfirmedAt: fixture.now.Add(time.Second),
	}, fixture.now.Add(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	quiescePost := mustLifecycleTerminalOutcome(t, quiesceOutcome)
	appendLifecycleTerminalFacts(&record, quiescePost)
	preserveClaim := lifecycleClaimFor(t, fixture.base, quiescePost.Snapshot,
		ActionPreserveAgentEnvironment, 62, fixture.now.Add(3*time.Second))
	prepared := mustPrepareLifecycle(t, quiescePost.Snapshot, preserveClaim, fixture.now.Add(3*time.Second))
	appendLifecyclePreparedFacts(&record, prepared)
	manifest := lifecycleManifest(t, fixture.initial.Subject, fixture.now.Add(4*time.Second))
	receipt := ports.AgentPreserveReceipt{
		Subject: fixture.initial.Subject, PreviousToken: quiesced,
		NextToken:      lifecycleToken(t, fixture.initial.Subject.ExternalRef, ports.AgentEnvironmentPreserved, "revision:3"),
		IdempotencyKey: prepared.Attempt.IdempotencyKey, Manifest: manifest,
		ReceiptRef: "receipt:preserve", ConfirmedAt: fixture.now.Add(5 * time.Second),
	}
	return prepared, receipt, record
}

func lifecycleClaimFor(t *testing.T, base ActionClaim, snapshot AgentEnvironmentLifecycleSnapshot,
	kind ActionKind, fence uint64, now time.Time,
) ActionClaim {
	t.Helper()
	claim := base
	claim.Token, claim.WorkerRef, claim.DeliveryAttempt = "claim:"+string(kind), "worker:lifecycle", 1
	claim.Fence, claim.LeaseUntil = fence, now.Add(time.Hour)
	claim.Action.Ref, claim.Action.Kind = "action:"+string(kind)+":execution", kind
	claim.Action.GoalRef, claim.Action.WorkItemRef = snapshot.Subject.GoalRef, snapshot.Subject.WorkItemRef
	claim.Action.ExecutionRef, claim.Action.PlanGeneration = snapshot.Subject.ExecutionRef, snapshot.Subject.PlanGeneration
	intent := claim.Action.EffectIntent
	intent.Ref, intent.ActionRef = "effect-intent:"+string(kind)+":execution", claim.Action.Ref
	intent.ActionKind, intent.Kind = kind, lifecycleEffectKind(kind)
	intent.Subject.GoalRef, intent.Subject.WorkItemRef = snapshot.Subject.GoalRef, snapshot.Subject.WorkItemRef
	intent.Subject.ExecutionRef, intent.Subject.PlanGeneration = snapshot.Subject.ExecutionRef, snapshot.Subject.PlanGeneration
	intent.Subject.AppSpecGeneration, intent.Subject.SpecHash = snapshot.Subject.AppSpecGeneration, snapshot.Subject.SpecHash
	intent.IdempotencyKey = "idempotency:" + string(kind) + ":execution"
	intent.CreatedAt = now.UTC()
	intent.Digest = EffectIntentDigest(intent)
	claim.Action.EffectIntentRef, claim.Action.EffectIntent = intent.Ref, intent
	approval := claim.EffectApproval
	approval.Ref, approval.IntentRef = "effect-approval:"+string(kind)+":execution", intent.Ref
	approval.IntentDigest, approval.Subject, approval.IdempotencyKey = intent.Digest, intent.Subject, intent.IdempotencyKey
	approval.ProposedBy, approval.SecurityCriticality = intent.ProposedBy, intent.SecurityCriticality
	approval.PolicyHash, approval.PolicyRevision, approval.TargetDigest = intent.PolicyHash, intent.PolicyRevision, intent.TargetDigest
	approval.DecidedAt = now.UTC()
	if approval.Source == EffectApprovalSourceExplicitDecision {
		approval.ExpiresAt = approval.DecidedAt.Add(intent.ApprovalTTL)
	} else {
		approval.ExpiresAt = time.Time{}
	}
	claim.EffectApproval, claim.Action.EffectApproval = approval, &approval
	return claim
}

func lifecycleEffectKind(kind ActionKind) EffectKind {
	switch kind {
	case ActionQuiesceAgent:
		return EffectKindAgentQuiesce
	case ActionPreserveAgentEnvironment:
		return EffectKindAgentPreserve
	case ActionCloseAgentEnvironment:
		return EffectKindAgentClose
	default:
		return ""
	}
}

func lifecycleToken(t *testing.T, externalRef string, state ports.AgentEnvironmentLifecycleState,
	revisionValue string,
) ports.AgentEnvironmentLifecycleToken {
	t.Helper()
	token, err := ports.NewAgentPhysicalToken(externalRef)
	if err != nil {
		t.Fatal(err)
	}
	revision, err := ports.NewAgentPhysicalRevision(revisionValue)
	if err != nil {
		t.Fatal(err)
	}
	fence, err := ports.NewAgentPhysicalFence("physical-fence:lifecycle")
	if err != nil {
		t.Fatal(err)
	}
	return ports.AgentEnvironmentLifecycleToken{PhysicalToken: token, Revision: revision, Fence: fence, State: state}
}

func lifecycleManifest(t *testing.T, subject ports.AgentEnvironmentLifecycleSubject,
	sealedAt time.Time,
) ports.AgentPhysicalPreservationManifest {
	t.Helper()
	content := []byte("sealed lifecycle manifest")
	digest := sha256.Sum256(content)
	revision, err := ports.NewAgentPhysicalRevision("work-revision:lifecycle")
	if err != nil {
		t.Fatal(err)
	}
	hash := strings.Repeat("a", 64)
	return ports.AgentPhysicalPreservationManifest{
		Ref: "physical-manifest:" + subject.ExecutionRef.String(), SHA256: fmt.Sprintf("%x", digest),
		Content: content, ContentBytes: uint64(len(content)), WorkRevision: revision,
		Causality: ports.AgentPhysicalPreservationCausality{
			PlanSHA256: hash, GrantSHA256: hash, KernelSHA256: hash, InitramfsSHA256: hash,
		},
		SealedAt: sealedAt,
	}
}

func lifecyclePreservationFact(t *testing.T, record GoalRecord, receipt ports.AgentPreserveReceipt,
	registeredAt time.Time,
) ComprobantePreservacionEntornoAgente {
	t.Helper()
	digest := strings.Repeat("b", 64)
	artifactPackage, _ := newArtifactRefForLifecycle(digest)
	artifactInventory, _ := newArtifactRefForLifecycle(digest)
	result := ports.ResultadoPreservacionEntornoAgente{
		Estado: ports.EntornoAgentePreservadoPendienteRevision, EjecucionRef: receipt.Subject.ExecutionRef,
		IntentoEjecucion: receipt.Subject.ExecutionAttempt, Cerca: 7, IdentidadExterna: receipt.Subject.ExternalRef,
		PaqueteRef: artifactPackage, PaqueteDigest: digest, InventarioRef: artifactInventory, InventarioDigest: digest,
		ConfiguracionDigest: digest, RootFSDigest: digest, ComprobanteRef: receipt.ReceiptRef,
		SelladoEn: receipt.Manifest.SealedAt, PreservadoEn: receipt.ConfirmedAt,
	}
	result.SelloDigest = ports.ResumenSelloPreservacionEntorno(result)
	fact := ComprobantePreservacionEntornoAgente{
		Ref: "environment-receipt:lifecycle", ClaveIdempotencia: "environment-idempotency:lifecycle",
		ManifiestoFisicoRef: receipt.Manifest.Ref, ManifiestoFisicoDigest: receipt.Manifest.SHA256,
		ProyectoRef: record.Goal.Project(), ObjetivoRef: record.Goal.Ref(), ItemRef: receipt.Subject.WorkItemRef,
		EjecucionRef: receipt.Subject.ExecutionRef, AlcanceEspacio: PreservacionEntornoSinEspacioTrabajo,
		Resultado: result, RegistradoEn: registeredAt,
	}
	if err := ValidarCausalidadPreservacionEntornoAgente(fact, record); err != nil {
		t.Fatalf("invalid preservation fixture: %v", err)
	}
	return fact
}

func newArtifactRefForLifecycle(digest string) (ref goal.ArtifactRef, err error) {
	return goal.NewArtifactRef("artifact:sha256:" + digest)
}

func mustPrepareLifecycle(t *testing.T, snapshot AgentEnvironmentLifecycleSnapshot,
	claim ActionClaim, at time.Time,
) AgentEnvironmentLifecyclePreEffectState {
	t.Helper()
	state, err := PrepareAgentEnvironmentLifecycleEffect(snapshot, claim, at)
	if err != nil {
		t.Fatal(err)
	}
	return state
}

func appendLifecyclePreparedFacts(record *GoalRecord, prepared AgentEnvironmentLifecyclePreEffectState) {
	record.EffectIntents = append(record.EffectIntents, prepared.Claim.Action.EffectIntent)
	record.EffectApprovals = append(record.EffectApprovals, prepared.Claim.EffectApproval)
	record.EffectAttempts = append(record.EffectAttempts, prepared.Attempt)
}

func appendLifecycleTerminalFacts(record *GoalRecord, terminal AgentEnvironmentLifecyclePostEffectState) {
	record.EffectReceipts = append(record.EffectReceipts, *terminal.EffectReceipt)
	record.ConsumptionReceipts = append(record.ConsumptionReceipts, terminal.ConsumptionReceipt)
}

func mustLifecycleTerminalOutcome(
	t *testing.T,
	outcome AgentEnvironmentLifecycleOutcome,
) AgentEnvironmentLifecyclePostEffectState {
	t.Helper()
	if outcome.Terminal == nil || outcome.Reconciliation != nil {
		t.Fatalf("lifecycle outcome is not terminal: %+v", outcome)
	}
	return *outcome.Terminal
}

func mustLifecycleReconciliationOutcome(
	t *testing.T,
	outcome AgentEnvironmentLifecycleOutcome,
) AgentEnvironmentLifecycleReconciliation {
	t.Helper()
	if outcome.Terminal != nil || outcome.Reconciliation == nil {
		t.Fatalf("lifecycle outcome is not reconciliation-only: %+v", outcome)
	}
	return *outcome.Reconciliation
}

func lifecycleFixtureExecution(
	t *testing.T,
	fixture *agentEnvironmentLifecycleFixture,
) *ExecutionRecord {
	t.Helper()
	for index := range fixture.record.Executions {
		if fixture.record.Executions[index].Ref == fixture.initial.Subject.ExecutionRef {
			return &fixture.record.Executions[index]
		}
	}
	t.Fatal("lifecycle fixture execution missing")
	return nil
}
