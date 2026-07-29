package networkauth

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"orquesta/internal/credentials"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

var errAttestationDenied = errors.New("attestation denied")
var errBeginFailed = errors.New("begin failed")
var errOpeningFailed = errors.New("opening failed")
var errCommitFailed = errors.New("commit failed")
var errRollbackFailed = errors.New("rollback failed")

type credentialStoreFake struct {
	credentials.Store
	material []byte
	at       time.Time
	useCount atomic.Int32
	deny     bool
}

func (store *credentialStoreFake) Use(
	_ context.Context,
	request credentials.UseRequest,
	consume func(credentials.Secret) error,
) (credentials.Receipt, error) {
	store.useCount.Add(1)
	if store.deny {
		return credentials.Receipt{}, credentials.NewError(credentials.ErrorRevoked, "credential_ref")
	}
	secret, err := credentials.NewSecret(store.material)
	if err != nil {
		return credentials.Receipt{}, err
	}
	defer secret.Destroy()
	if err := consume(secret); err != nil {
		return credentials.Receipt{}, credentials.WrapError(credentials.ErrorConsumerFailed, "consumer", err)
	}
	return credentials.Receipt{
		CredentialRef: request.CredentialRef, OwnerRef: request.OwnerRef,
		ScopeRef: request.ScopeRef, PurposeRef: request.PurposeRef, Version: request.Version,
		RequestRef: request.RequestRef, ActorRef: request.ActorRef, Operation: "use",
		OccurredAt: store.at,
	}, nil
}

type attestationVerifierFake struct {
	want      LaunchAttestationCheck
	deny      bool
	callCount atomic.Int32
}

func (verifier *attestationVerifierFake) VerifyLaunchAttestation(
	_ context.Context,
	check LaunchAttestationCheck,
) error {
	verifier.callCount.Add(1)
	if verifier.deny || check != verifier.want {
		return errAttestationDenied
	}
	return nil
}

type launchTransactionSnapshot struct {
	OpenCount               int
	CommitCount             int
	RollbackCount           int
	Partial                 bool
	Reachable               bool
	RollbackContextCanceled bool
}

type launchTransactionFake struct {
	mu                      sync.Mutex
	openCount               int
	commitCount             int
	rollbackCount           int
	partial                 bool
	reachable               bool
	openErr                 error
	commitErr               error
	rollbackErr             error
	blockRollback           bool
	cancelAfterOpen         context.CancelFunc
	rollbackContextCanceled bool
}

func (transaction *launchTransactionFake) Open(_ context.Context) error {
	transaction.mu.Lock()
	transaction.openCount++
	transaction.partial = true
	openErr := transaction.openErr
	cancel := transaction.cancelAfterOpen
	transaction.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	return openErr
}

func (transaction *launchTransactionFake) Commit(_ context.Context) error {
	transaction.mu.Lock()
	defer transaction.mu.Unlock()
	transaction.commitCount++
	if transaction.commitErr != nil {
		return transaction.commitErr
	}
	transaction.partial = false
	transaction.reachable = true
	return nil
}

func (transaction *launchTransactionFake) Rollback(ctx context.Context) error {
	transaction.mu.Lock()
	transaction.rollbackCount++
	transaction.rollbackContextCanceled = ctx.Err() != nil
	block := transaction.blockRollback
	transaction.mu.Unlock()
	if block {
		<-ctx.Done()
		return ctx.Err()
	}
	transaction.mu.Lock()
	defer transaction.mu.Unlock()
	if transaction.rollbackErr != nil {
		return transaction.rollbackErr
	}
	transaction.partial = false
	transaction.reachable = false
	return nil
}

func (transaction *launchTransactionFake) Snapshot() launchTransactionSnapshot {
	transaction.mu.Lock()
	defer transaction.mu.Unlock()
	return launchTransactionSnapshot{
		OpenCount: transaction.openCount, CommitCount: transaction.commitCount,
		RollbackCount: transaction.rollbackCount, Partial: transaction.partial,
		Reachable:               transaction.reachable,
		RollbackContextCanceled: transaction.rollbackContextCanceled,
	}
}

type launchTransactionFactoryFake struct {
	mu          sync.Mutex
	beginCount  int
	beginErr    error
	transaction *launchTransactionFake
}

func newLaunchTransactionFactoryFake() *launchTransactionFactoryFake {
	return &launchTransactionFactoryFake{transaction: &launchTransactionFake{}}
}

func (factory *launchTransactionFactoryFake) Begin(
	_ context.Context,
) (ports.AgentMicroVMLaunchTransaction, error) {
	factory.mu.Lock()
	defer factory.mu.Unlock()
	factory.beginCount++
	if factory.beginErr != nil {
		return nil, factory.beginErr
	}
	return factory.transaction, nil
}

func (factory *launchTransactionFactoryFake) BeginCount() int {
	factory.mu.Lock()
	defer factory.mu.Unlock()
	return factory.beginCount
}

type testClock struct {
	mu  sync.RWMutex
	now time.Time
}

func (clock *testClock) Now() time.Time {
	clock.mu.RLock()
	defer clock.mu.RUnlock()
	return clock.now
}

func (clock *testClock) Set(now time.Time) {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	clock.now = now
}

func validNetworkPolicy(t *testing.T) ports.AgentMicroVMNetworkPolicy {
	t.Helper()
	projectRef, _ := goal.NewProjectRef("project:network-auth")
	goalRef, _ := goal.NewGoalRef("goal:network-auth")
	workItemRef, _ := goal.NewWorkItemRef("work:network-auth")
	executionRef, _ := goal.NewExecutionRef("execution:network-auth")
	attestationRef, _ := goal.NewAttestationRef("attestation:network-auth")
	scope := ports.AgentMicroVMNetworkScope{
		ProjectRef: projectRef, GoalRef: goalRef, WorkItemRef: workItemRef, ExecutionRef: executionRef,
		PlanGeneration: 2, AppSpecGeneration: 3, ExecutionAttempt: 1,
		SpecHash: bytesToDigest([]byte("spec")), AgentRef: "agent:codex",
	}
	policy := ports.AgentMicroVMNetworkPolicy{
		Ref: "policy:agent-network:auth", Scope: scope, GuestCID: 44,
		LaunchIdentityRef: "launch-identity:network-auth",
		LaunchCredential: ports.AgentMicroVMLaunchCredentialBinding{
			Ref: "credential:network-auth", OwnerRef: projectRef.String(),
			ScopeRef:   ports.AgentMicroVMLaunchCredentialScopeRef(scope),
			PurposeRef: ports.AgentMicroVMLaunchCredentialPurpose, Version: 1,
		},
		LaunchAttestationRef: attestationRef,
		EgressMode:           ports.AgentMicroVMEgressBrokerOnly,
		Broker: ports.AgentMicroVMVsockService{
			Port: 62001, IdentityRef: "service:orquesta-broker",
			IdentityDigest: bytesToDigest([]byte("broker")),
		},
	}
	policy.LaunchBindingDigest = ports.AgentMicroVMLaunchBindingDigest(
		scope, policy.LaunchIdentityRef, policy.LaunchCredential, policy.LaunchAttestationRef,
	)
	return policy
}

type verifierFixture struct {
	verifier     *Verifier
	challenges   *MemoryChallengeStore
	credentials  *credentialStoreFake
	attestations *attestationVerifierFake
	clock        *testClock
	now          time.Time
	key          []byte
}

func newVerifierFixture(t *testing.T) verifierFixture {
	t.Helper()
	now := time.Unix(500, 0).UTC()
	clock := &testClock{now: now}
	policy := validNetworkPolicy(t)
	policyDigest, err := ports.AgentMicroVMNetworkPolicyDigest(policy)
	if err != nil {
		t.Fatal(err)
	}
	challenges, err := NewMemoryChallengeStore(ChallengeStoreOptions{
		Now: clock.Now, Random: bytes.NewReader(bytes.Repeat([]byte{0x42}, 4096)),
		TTL: time.Minute, MaxActive: 8,
	})
	if err != nil {
		t.Fatal(err)
	}
	credentialStore := &credentialStoreFake{material: []byte("launch-proof-key"), at: now}
	launchPlanDigest := bytesToDigest([]byte("launch-plan:adapter:firecracker-network-plan"))
	attestations := &attestationVerifierFake{want: LaunchAttestationCheck{
		AttestationRef: policy.LaunchAttestationRef.String(),
		PolicyDigest:   policyDigest, LaunchPlanDigest: launchPlanDigest,
		LaunchBindingDigest: policy.LaunchBindingDigest,
		ProjectRef:          policy.Scope.ProjectRef.String(), GoalRef: policy.Scope.GoalRef.String(),
		WorkItemRef: policy.Scope.WorkItemRef.String(), ExecutionRef: policy.Scope.ExecutionRef.String(),
		AgentRef: policy.Scope.AgentRef, ExecutionAttempt: policy.Scope.ExecutionAttempt,
	}}
	verifier, err := New(Config{
		CredentialStore: credentialStore, Attestations: attestations, Challenges: challenges,
		Now: clock.Now, CleanupTimeout: 100 * time.Millisecond,
		VerifierRef:        "verifier:agent-firecracker-network",
		CredentialActorRef: "actor:agent-firecracker-network",
	})
	if err != nil {
		t.Fatal(err)
	}
	return verifierFixture{
		verifier: verifier, challenges: challenges, credentials: credentialStore,
		attestations: attestations, clock: clock, now: now,
		key: append([]byte(nil), credentialStore.material...),
	}
}

func (fixture verifierFixture) proofRequest(t *testing.T) ports.AgentMicroVMLaunchProofRequest {
	t.Helper()
	policy := validNetworkPolicy(t)
	policyDigest, err := ports.AgentMicroVMNetworkPolicyDigest(policy)
	if err != nil {
		t.Fatal(err)
	}
	launchPlanDigest := bytesToDigest([]byte("launch-plan:adapter:firecracker-network-plan"))
	challenge, err := fixture.challenges.Issue(context.Background(), ChallengeIssueRequest{
		PolicyDigest: policyDigest, LaunchBindingDigest: policy.LaunchBindingDigest,
		LaunchPlanDigest: launchPlanDigest,
		ExecutionRef:     policy.Scope.ExecutionRef.String(), AgentRef: policy.Scope.AgentRef,
	})
	if err != nil {
		t.Fatal(err)
	}
	message, err := ports.AgentMicroVMLaunchProofMessage(
		policy, policyDigest, launchPlanDigest,
		ports.AgentMicroVMLaunchProofScheme, challenge.Ref, challenge.Value,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer clear(message)
	mac := hmac.New(sha256.New, fixture.key)
	_, _ = mac.Write(message)
	proof, err := ports.NewAgentMicroVMLaunchProof(mac.Sum(nil))
	if err != nil {
		t.Fatal(err)
	}
	return ports.AgentMicroVMLaunchProofRequest{
		Policy: policy, ExpectedPolicyDigest: policyDigest, LaunchPlanDigest: launchPlanDigest,
		Scheme:       ports.AgentMicroVMLaunchProofScheme,
		ChallengeRef: challenge.Ref, Challenge: challenge.Value, Proof: proof,
		RequestedAt: fixture.now.Add(-time.Second),
	}
}

func TestNewRequiresExplicitPositiveCleanupTimeout(t *testing.T) {
	fixture := newVerifierFixture(t)
	config := fixture.verifier.config
	config.CleanupTimeout = 0
	if _, err := New(config); ErrorCode(err) != "agent_firecracker_network_auth.config_invalid" {
		t.Fatalf("zero cleanup timeout error = %v", err)
	}
}

func TestVerifierAuthorizesOnceAndRejectsIdenticalReplay(t *testing.T) {
	fixture := newVerifierFixture(t)
	request := fixture.proofRequest(t)
	defer request.Proof.Destroy()
	transactions := newLaunchTransactionFactoryFake()
	receipt, err := fixture.verifier.Authorize(context.Background(), request, transactions)
	if err != nil {
		t.Fatal(err)
	}
	if err := ports.ValidateAgentMicroVMLaunchAuthorizationReceipt(request, receipt); err != nil {
		t.Fatalf("receipt invalid: %v", err)
	}
	if _, err := fixture.verifier.Authorize(context.Background(), request, transactions); ErrorCode(err) !=
		"agent_firecracker_network_auth.challenge_denied" {
		t.Fatalf("identical replay error = %v", err)
	}
	if got := fixture.credentials.useCount.Load(); got != 1 {
		t.Fatalf("credential uses = %d, want 1", got)
	}
	if got := fixture.attestations.callCount.Load(); got != 1 {
		t.Fatalf("attestation calls = %d, want 1", got)
	}
	if got := transactions.BeginCount(); got != 1 {
		t.Fatalf("transaction begins = %d, want 1", got)
	}
	if got := transactions.transaction.Snapshot(); got != (launchTransactionSnapshot{
		OpenCount: 1, CommitCount: 1, Reachable: true,
	}) {
		t.Fatalf("transaction state = %#v", got)
	}
}

func TestVerifierBadProofConsumesChallengeAndCannotRetry(t *testing.T) {
	fixture := newVerifierFixture(t)
	request := fixture.proofRequest(t)
	defer request.Proof.Destroy()
	transactions := newLaunchTransactionFactoryFake()
	badProof, _ := ports.NewAgentMicroVMLaunchProof(bytes.Repeat([]byte{0xff}, sha256.Size))
	bad := request
	bad.Proof = badProof
	defer bad.Proof.Destroy()
	if _, err := fixture.verifier.Authorize(context.Background(), bad, transactions); ErrorCode(err) !=
		"agent_firecracker_network_auth.proof_invalid" {
		t.Fatalf("bad proof error = %v", err)
	}
	if _, err := fixture.verifier.Authorize(context.Background(), request, transactions); ErrorCode(err) !=
		"agent_firecracker_network_auth.challenge_denied" {
		t.Fatalf("retry after bad proof error = %v", err)
	}
	if got := fixture.credentials.useCount.Load(); got != 1 {
		t.Fatalf("credential uses = %d, want 1", got)
	}
	if got := transactions.BeginCount(); got != 0 {
		t.Fatalf("transaction begins = %d, want 0", got)
	}
}

func TestVerifierRejectsAttestationBeforeCredentialUse(t *testing.T) {
	fixture := newVerifierFixture(t)
	fixture.attestations.deny = true
	request := fixture.proofRequest(t)
	defer request.Proof.Destroy()
	transactions := newLaunchTransactionFactoryFake()
	if _, err := fixture.verifier.Authorize(context.Background(), request, transactions); ErrorCode(err) !=
		"agent_firecracker_network_auth.attestation_denied" {
		t.Fatalf("attestation error = %v", err)
	}
	if got := fixture.credentials.useCount.Load(); got != 0 {
		t.Fatalf("credential used before attestation: %d", got)
	}
	fixture.attestations.deny = false
	if _, err := fixture.verifier.Authorize(context.Background(), request, transactions); ErrorCode(err) !=
		"agent_firecracker_network_auth.challenge_denied" {
		t.Fatalf("retry after attestation denial error = %v", err)
	}
	if got := fixture.attestations.callCount.Load(); got != 1 {
		t.Fatalf("attestation calls = %d, want 1", got)
	}
	if got := transactions.BeginCount(); got != 0 {
		t.Fatalf("transaction begins = %d, want 0", got)
	}
}

func TestVerifierMissingAndExpiredChallengesDoNotCrossLedgers(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		fixture := newVerifierFixture(t)
		request := fixture.proofRequest(t)
		defer request.Proof.Destroy()
		request.ChallengeRef = "challenge:agent-microvm:missing"
		transactions := newLaunchTransactionFactoryFake()
		if _, err := fixture.verifier.Authorize(context.Background(), request, transactions); ErrorCode(err) !=
			"agent_firecracker_network_auth.challenge_denied" {
			t.Fatalf("missing challenge error = %v", err)
		}
		if fixture.credentials.useCount.Load() != 0 || fixture.attestations.callCount.Load() != 0 ||
			transactions.BeginCount() != 0 {
			t.Fatalf("missing challenge crossed a ledger or opening")
		}
	})
	t.Run("expiry_is_inclusive", func(t *testing.T) {
		fixture := newVerifierFixture(t)
		request := fixture.proofRequest(t)
		defer request.Proof.Destroy()
		fixture.clock.Set(fixture.now.Add(time.Minute))
		transactions := newLaunchTransactionFactoryFake()
		if _, err := fixture.verifier.Authorize(context.Background(), request, transactions); ErrorCode(err) !=
			"agent_firecracker_network_auth.challenge_denied" {
			t.Fatalf("expired challenge error = %v", err)
		}
		if fixture.credentials.useCount.Load() != 0 || fixture.attestations.callCount.Load() != 0 ||
			transactions.BeginCount() != 0 {
			t.Fatalf("expired challenge crossed a ledger or opening")
		}
	})
}

func TestVerifierRejectsLaunchPlanSubstitutionBeforeLedgers(t *testing.T) {
	fixture := newVerifierFixture(t)
	request := fixture.proofRequest(t)
	defer request.Proof.Destroy()
	request.LaunchPlanDigest = bytesToDigest([]byte("launch-plan:adapter:substitute"))
	transactions := newLaunchTransactionFactoryFake()
	if _, err := fixture.verifier.Authorize(
		context.Background(),
		request,
		transactions,
	); ErrorCode(err) != "agent_firecracker_network_auth.challenge_denied" {
		t.Fatalf("launch plan substitution error = %v", err)
	}
	if fixture.credentials.useCount.Load() != 0 || fixture.attestations.callCount.Load() != 0 ||
		transactions.BeginCount() != 0 {
		t.Fatal("substituted LaunchPlanDigest crossed challenge boundary")
	}
}

func TestVerifierConcurrentReplayHasSingleWinner(t *testing.T) {
	fixture := newVerifierFixture(t)
	request := fixture.proofRequest(t)
	defer request.Proof.Destroy()
	transactions := newLaunchTransactionFactoryFake()
	var wait sync.WaitGroup
	var successes atomic.Int32
	wait.Add(2)
	for range 2 {
		go func() {
			defer wait.Done()
			if _, err := fixture.verifier.Authorize(context.Background(), request, transactions); err == nil {
				successes.Add(1)
			}
		}()
	}
	wait.Wait()
	if got := successes.Load(); got != 1 {
		t.Fatalf("concurrent winners = %d, want 1", got)
	}
	if got := fixture.credentials.useCount.Load(); got != 1 {
		t.Fatalf("concurrent credential uses = %d, want 1", got)
	}
	if got := fixture.attestations.callCount.Load(); got != 1 {
		t.Fatalf("concurrent attestation calls = %d, want 1", got)
	}
	if got := transactions.BeginCount(); got != 1 {
		t.Fatalf("concurrent transaction begins = %d, want 1", got)
	}
	if got := transactions.transaction.Snapshot(); got != (launchTransactionSnapshot{
		OpenCount: 1, CommitCount: 1, Reachable: true,
	}) {
		t.Fatalf("concurrent transaction state = %#v", got)
	}
}

func TestVerifierOpenPartialFailureRollsBackAndConsumesAuthorization(t *testing.T) {
	fixture := newVerifierFixture(t)
	request := fixture.proofRequest(t)
	defer request.Proof.Destroy()
	transactions := newLaunchTransactionFactoryFake()
	transactions.transaction.openErr = errOpeningFailed
	if _, err := fixture.verifier.Authorize(context.Background(), request, transactions); ErrorCode(err) !=
		"agent_firecracker_network_auth.opening_failed" {
		t.Fatalf("opening error = %v", err)
	}
	transactions.transaction.openErr = nil
	if _, err := fixture.verifier.Authorize(context.Background(), request, transactions); ErrorCode(err) !=
		"agent_firecracker_network_auth.challenge_denied" {
		t.Fatalf("retry after opening failure error = %v", err)
	}
	if got := fixture.credentials.useCount.Load(); got != 1 {
		t.Fatalf("credential uses = %d, want 1", got)
	}
	if got := transactions.BeginCount(); got != 1 {
		t.Fatalf("transaction begins = %d, want 1", got)
	}
	if got := transactions.transaction.Snapshot(); got != (launchTransactionSnapshot{
		OpenCount: 1, RollbackCount: 1,
	}) {
		t.Fatalf("transaction state after partial open = %#v", got)
	}
}

func TestVerifierBeginFailureHasNoTransactionEffectsAndConsumesAuthorization(t *testing.T) {
	fixture := newVerifierFixture(t)
	request := fixture.proofRequest(t)
	defer request.Proof.Destroy()
	transactions := newLaunchTransactionFactoryFake()
	transactions.beginErr = errBeginFailed

	receipt, err := fixture.verifier.Authorize(context.Background(), request, transactions)
	if ErrorCode(err) != "agent_firecracker_network_auth.transaction_begin_failed" {
		t.Fatalf("begin error = %v", err)
	}
	if receipt != (ports.AgentMicroVMLaunchAuthorizationReceipt{}) {
		t.Fatalf("begin failure returned receipt: %#v", receipt)
	}
	if got := transactions.BeginCount(); got != 1 {
		t.Fatalf("transaction begins = %d, want 1", got)
	}
	if got := transactions.transaction.Snapshot(); got != (launchTransactionSnapshot{}) {
		t.Fatalf("begin failure produced transaction effects: %#v", got)
	}
	if _, err := fixture.verifier.Authorize(context.Background(), request, transactions); ErrorCode(err) !=
		"agent_firecracker_network_auth.challenge_denied" {
		t.Fatalf("retry after begin failure error = %v", err)
	}
}

func TestVerifierCancellationAfterOpenRollsBackBeforeCommit(t *testing.T) {
	fixture := newVerifierFixture(t)
	request := fixture.proofRequest(t)
	defer request.Proof.Destroy()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	transactions := newLaunchTransactionFactoryFake()
	transactions.transaction.cancelAfterOpen = cancel

	receipt, err := fixture.verifier.Authorize(ctx, request, transactions)
	if ErrorCode(err) != "agent_firecracker_network_auth.unavailable" {
		t.Fatalf("cancellation error = %v", err)
	}
	if receipt != (ports.AgentMicroVMLaunchAuthorizationReceipt{}) {
		t.Fatalf("cancellation returned receipt: %#v", receipt)
	}
	if got := transactions.transaction.Snapshot(); got != (launchTransactionSnapshot{
		OpenCount: 1, RollbackCount: 1,
	}) {
		t.Fatalf("transaction state after cancellation = %#v", got)
	}
}

func TestVerifierCommitPartialFailureRollsBack(t *testing.T) {
	fixture := newVerifierFixture(t)
	request := fixture.proofRequest(t)
	defer request.Proof.Destroy()
	transactions := newLaunchTransactionFactoryFake()
	transactions.transaction.commitErr = errCommitFailed

	receipt, err := fixture.verifier.Authorize(context.Background(), request, transactions)
	if ErrorCode(err) != "agent_firecracker_network_auth.commit_failed" {
		t.Fatalf("commit error = %v", err)
	}
	if receipt != (ports.AgentMicroVMLaunchAuthorizationReceipt{}) {
		t.Fatalf("failed commit returned receipt: %#v", receipt)
	}
	if got := transactions.transaction.Snapshot(); got != (launchTransactionSnapshot{
		OpenCount: 1, CommitCount: 1, RollbackCount: 1,
	}) {
		t.Fatalf("transaction state after commit failure = %#v", got)
	}
}

func TestVerifierRollbackFailureIsVisibleAndNeverReturnsReceipt(t *testing.T) {
	fixture := newVerifierFixture(t)
	request := fixture.proofRequest(t)
	defer request.Proof.Destroy()
	transactions := newLaunchTransactionFactoryFake()
	transactions.transaction.openErr = errOpeningFailed
	transactions.transaction.rollbackErr = errRollbackFailed

	receipt, err := fixture.verifier.Authorize(context.Background(), request, transactions)
	if ErrorCode(err) != "agent_firecracker_network_auth.cleanup_failed" {
		t.Fatalf("rollback error = %v", err)
	}
	if receipt != (ports.AgentMicroVMLaunchAuthorizationReceipt{}) {
		t.Fatalf("rollback failure returned receipt: %#v", receipt)
	}
	if got := transactions.transaction.Snapshot(); got != (launchTransactionSnapshot{
		OpenCount: 1, RollbackCount: 1, Partial: true,
	}) {
		t.Fatalf("transaction state after rollback failure = %#v", got)
	}
}

func TestVerifierBoundsBlockedRollbackAndNeverReturnsReceipt(t *testing.T) {
	fixture := newVerifierFixture(t)
	fixture.verifier.config.CleanupTimeout = 10 * time.Millisecond
	request := fixture.proofRequest(t)
	defer request.Proof.Destroy()
	transactions := newLaunchTransactionFactoryFake()
	transactions.transaction.openErr = errOpeningFailed
	transactions.transaction.blockRollback = true

	type authorizationResult struct {
		receipt ports.AgentMicroVMLaunchAuthorizationReceipt
		err     error
	}
	result := make(chan authorizationResult, 1)
	startedAt := time.Now()
	go func() {
		receipt, err := fixture.verifier.Authorize(context.Background(), request, transactions)
		result <- authorizationResult{receipt: receipt, err: err}
	}()
	var authorization authorizationResult
	select {
	case authorization = <-result:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("blocked rollback exceeded its cleanup bound")
	}
	receipt, err := authorization.receipt, authorization.err
	if ErrorCode(err) != "agent_firecracker_network_auth.cleanup_failed" {
		t.Fatalf("blocked rollback error = %v", err)
	}
	if elapsed := time.Since(startedAt); elapsed < fixture.verifier.config.CleanupTimeout ||
		elapsed > 500*time.Millisecond {
		t.Fatalf("blocked rollback elapsed = %v", elapsed)
	}
	if receipt != (ports.AgentMicroVMLaunchAuthorizationReceipt{}) {
		t.Fatalf("blocked rollback returned receipt: %#v", receipt)
	}
	if got := transactions.transaction.Snapshot(); got != (launchTransactionSnapshot{
		OpenCount: 1, RollbackCount: 1, Partial: true,
	}) {
		t.Fatalf("transaction state after blocked rollback = %#v", got)
	}
}

func TestMemoryChallengeStoreBindsScopeAndExpires(t *testing.T) {
	now := time.Unix(700, 0).UTC()
	store, err := NewMemoryChallengeStore(ChallengeStoreOptions{
		Now: func() time.Time { return now }, Random: bytes.NewReader(bytes.Repeat([]byte{0x33}, 64)),
		TTL: time.Second, MaxActive: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	issue := ChallengeIssueRequest{
		PolicyDigest:        bytesToDigest([]byte("policy")),
		LaunchBindingDigest: bytesToDigest([]byte("binding")),
		LaunchPlanDigest:    bytesToDigest([]byte("launch-plan")),
		ExecutionRef:        "execution:challenge", AgentRef: "agent:codex",
	}
	challenge, err := store.Issue(context.Background(), issue)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Issue(context.Background(), issue); ErrorCode(err) !=
		"agent_firecracker_network_auth.challenge_capacity_reached" {
		t.Fatalf("capacity error = %v", err)
	}
	mismatched := ChallengeConsumeRequest{
		Ref: challenge.Ref, Value: challenge.Value, PolicyDigest: issue.PolicyDigest,
		LaunchBindingDigest: issue.LaunchBindingDigest,
		LaunchPlanDigest:    issue.LaunchPlanDigest, ExecutionRef: "execution:other",
		AgentRef: issue.AgentRef,
	}
	if err := store.Consume(context.Background(), mismatched); ErrorCode(err) !=
		"agent_firecracker_network_auth.challenge_binding_mismatch" {
		t.Fatalf("binding error = %v", err)
	}
	now = now.Add(time.Second)
	valid := mismatched
	valid.ExecutionRef = issue.ExecutionRef
	if err := store.Consume(context.Background(), valid); ErrorCode(err) !=
		"agent_firecracker_network_auth.challenge_expired" {
		t.Fatalf("expiry error = %v", err)
	}
}

func bytesToDigest(value []byte) string {
	digest := sha256.Sum256(value)
	return hexString(digest[:])
}

func hexString(value []byte) string {
	const digits = "0123456789abcdef"
	result := make([]byte, len(value)*2)
	for index, octet := range value {
		result[index*2] = digits[octet>>4]
		result[index*2+1] = digits[octet&0x0f]
	}
	return string(result)
}
