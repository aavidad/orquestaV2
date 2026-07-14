package acceptance_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"orquesta/internal/adapters/state/sqlite"
	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

const v06FixturePath = "acceptance/fixtures/v06_atomic_state_outbox.json"
const v06TrustedBaseGitCommitOID = "238ebc59025d3dd2bdd9593febd878650af6a660"
const v06ProductDeltaBaseGitCommitOID = "00b760c5989300c205eda9d3a12e0a86d1f096c2"
const v06ProductDeltaSealedGitCommitOID = "57ad1986255af99f60bca1a9c59d6da1772adf77"

type v06Fixture struct {
	SchemaVersion                  int                     `json:"schema_version"`
	ReceiptSchemaVersion           int                     `json:"receipt_schema_version"`
	ContractID                     string                  `json:"contract_id"`
	TrustedBaseGitCommitOID        string                  `json:"trusted_base_git_commit_oid"`
	ProductDeltaBaseGitCommitOID   string                  `json:"product_delta_base_git_commit_oid"`
	ProductDeltaSealedGitCommitOID string                  `json:"product_delta_sealed_git_commit_oid"`
	Command                        string                  `json:"command"`
	ExecutionArgv                  []string                `json:"execution_argv"`
	OutputPath                     string                  `json:"output_path"`
	ReceiptPath                    string                  `json:"receipt_path"`
	CandidateSubjects              []string                `json:"candidate_subjects"`
	OwnedCapabilityIDs             []string                `json:"owned_capability_ids"`
	DeferredCapabilities           []v06DeferredCapability `json:"deferred_capabilities"`
	ClockAndRetry                  v06ClockAndRetry        `json:"clock_and_retry"`
	OpaqueRequirements             v06OpaqueRequirements   `json:"opaque_requirements"`
	LaunchReceiptRequiredFields    []string                `json:"launch_receipt_required_fields"`
	Assertions                     []string                `json:"assertions"`
}

type v06DeferredCapability struct {
	ID                 string `json:"id"`
	Owner              string `json:"owner"`
	AcceptanceContract string `json:"acceptance_contract"`
}

type v06ClockAndRetry struct {
	BaseTime             string   `json:"base_time"`
	ClaimLease           string   `json:"claim_lease"`
	ObservationDelay     string   `json:"observation_delay"`
	ExecutionTimeout     string   `json:"execution_timeout"`
	MaxExecutionAttempts uint64   `json:"max_execution_attempts"`
	ClaimContenders      int      `json:"claim_contenders"`
	LaunchOutcomes       []string `json:"launch_outcomes"`
}

type v06OpaqueRequirements struct {
	RoleKey        string   `json:"role_key"`
	SkillRefs      []string `json:"skill_refs"`
	ToolRefs       []string `json:"tool_refs"`
	CapabilityRefs []string `json:"capability_refs"`
}

func TestAcceptanceV06AtomicStateOutbox(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v06Fixture](t, filepath.Join(repositoryRoot, filepath.FromSlash(v06FixturePath)))
	v06AssertFixtureHeader(t, repositoryRoot, fixture)

	t.Run("public_contract_separates_execution_delivery_fence_and_state_port", func(t *testing.T) {
		v06AssertPublicContractShape(t, fixture)
	})
	t.Run("snapshot_event_and_outbox_are_atomic_and_idempotent", func(t *testing.T) {
		v06AssertAtomicCreate(t, fixture)
	})
	t.Run("delivery_retry_provider_replacement_receipts_and_terminal_restart", func(t *testing.T) {
		v06AssertReplaceableAttemptsAndRestart(t, fixture)
	})
	t.Run("trusted_clock_race_reclaim_stale_fence_and_immutable_receipt", func(t *testing.T) {
		v06AssertFencedClaimAndReceipt(t, fixture)
	})
	t.Run("opaque_capability_matching_uses_durable_queue", func(t *testing.T) {
		v06AssertCapabilityMatchingAndNoPrivateQueue(t, fixture)
	})
}

func TestAcceptanceV06AtomicStateOutboxReceipt(t *testing.T) {
	evidenceAssertReceiptV3(t, evidenceRepositoryRoot(t), evidenceReceiptV3Expectation{
		Contract: "AC-V06-ATOMIC-STATE-OUTBOX", FixturePath: v06FixturePath,
		ReceiptPath:       "product/evidence/v06_atomic_state_outbox.json",
		ExecutedNotBefore: "2026-07-14T00:00:00Z", TrustedBaseGitCommitOID: v06TrustedBaseGitCommitOID,
	})
}

func TestV06CandidateSubjectsCoverCommittedDelta(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v06Fixture](t, filepath.Join(repositoryRoot, filepath.FromSlash(v06FixturePath)))
	if err := evidenceValidateSealedCommit(
		repositoryRoot, fixture.ProductDeltaBaseGitCommitOID, fixture.ProductDeltaSealedGitCommitOID,
	); err != nil {
		t.Fatal(err)
	}
	output, err := evidenceGit(
		repositoryRoot, "diff", "--name-only",
		fixture.ProductDeltaBaseGitCommitOID, fixture.ProductDeltaSealedGitCommitOID, "--",
	)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.TrimSpace(string(output))
	var changed []string
	if text != "" {
		changed = strings.Split(text, "\n")
	}
	sort.Strings(changed)
	if !reflect.DeepEqual(changed, fixture.CandidateSubjects) {
		t.Fatalf("V06 candidate subjects differ from sealed product delta:\nchanged=%v\nfixture=%v", changed, fixture.CandidateSubjects)
	}
}

func v06AssertFixtureHeader(t *testing.T, repositoryRoot string, fixture v06Fixture) {
	t.Helper()
	wantCommand := "sh -c '" + v06ValidationShellBody() + "'"
	wantArgv := []string{"sh", "-c", v06ValidationShellBody()}
	wantCapabilities := []string{
		"EVD-02", "GOV-05", "GOV-06", "OPS-09", "OPS-10", "OPS-12", "ORC-12", "ORC-13", "ORC-17",
	}
	wantDeferred := []v06DeferredCapability{
		{ID: "EVD-01", Owner: "test_attestor", AcceptanceContract: "AC-V17-TEST-ATTESTOR"},
		{ID: "GOV-17", Owner: "command_registry", AcceptanceContract: "AC-V20-COMMAND-REGISTRY"},
		{ID: "OPS-13", Owner: "postgres_s3_multihost", AcceptanceContract: "AC-V31-POSTGRES-S3-MULTIHOST"},
	}
	wantReceiptFields := []string{
		"execution_ref", "goal_ref", "work_item_ref", "plan_generation", "app_spec_generation",
		"execution_attempt", "spec_hash", "provider_ref", "model_ref", "agent_ref", "external_ref",
		"idempotency_key", "accepted_at",
	}
	if fixture.SchemaVersion != 1 || fixture.ReceiptSchemaVersion != 3 ||
		fixture.ContractID != "AC-V06-ATOMIC-STATE-OUTBOX" || fixture.TrustedBaseGitCommitOID != v06TrustedBaseGitCommitOID ||
		fixture.ProductDeltaBaseGitCommitOID != v06ProductDeltaBaseGitCommitOID ||
		fixture.ProductDeltaSealedGitCommitOID != v06ProductDeltaSealedGitCommitOID ||
		fixture.Command != wantCommand || !reflect.DeepEqual(fixture.ExecutionArgv, wantArgv) ||
		fixture.OutputPath != "product/evidence/v06_atomic_state_outbox.output.txt" ||
		fixture.ReceiptPath != "product/evidence/v06_atomic_state_outbox.json" ||
		!reflect.DeepEqual(fixture.OwnedCapabilityIDs, wantCapabilities) ||
		!reflect.DeepEqual(fixture.DeferredCapabilities, wantDeferred) ||
		!reflect.DeepEqual(fixture.LaunchReceiptRequiredFields, wantReceiptFields) || len(fixture.Assertions) != 9 {
		t.Fatalf("invalid V06 fixture header: %+v", fixture)
	}
	if _, err := time.Parse(time.RFC3339Nano, fixture.ClockAndRetry.BaseTime); err != nil {
		t.Fatalf("invalid base time: %v", err)
	}
	for name, raw := range map[string]string{
		"claim lease": fixture.ClockAndRetry.ClaimLease, "observation delay": fixture.ClockAndRetry.ObservationDelay,
		"execution timeout": fixture.ClockAndRetry.ExecutionTimeout,
	} {
		if duration, err := time.ParseDuration(raw); err != nil || duration <= 0 {
			t.Fatalf("invalid %s %q: %v", name, raw, err)
		}
	}
	if fixture.ClockAndRetry.MaxExecutionAttempts < 2 ||
		fixture.ClockAndRetry.ClaimContenders < 2 ||
		!reflect.DeepEqual(fixture.ClockAndRetry.LaunchOutcomes, []string{"temporary_error", "permanent_error", "accepted"}) {
		t.Fatalf("invalid V06 retry/race scenario: %+v", fixture.ClockAndRetry)
	}
	if err := evidenceValidateCandidateSubjects(fixture.CandidateSubjects, fixture.ReceiptPath, fixture.OutputPath); err != nil {
		t.Fatal(err)
	}
	for _, relative := range fixture.CandidateSubjects {
		if _, err := os.Stat(filepath.Join(repositoryRoot, filepath.FromSlash(relative))); err != nil {
			t.Errorf("candidate subject %q is not readable: %v", relative, err)
		}
	}
}

func v06ValidationShellBody() string {
	return "go test -mod=vendor -count=1 . ./acceptance -run \"^(TestProductRoadmapIsExhaustiveAndCausal|TestProductRoadmapV06ScopeAndExecutableContract|TestAcceptanceV06AtomicStateOutbox|TestV06CandidateSubjectsCoverCommittedDelta)$\"" +
		" && go test -mod=vendor -count=1 ./internal/goal ./internal/application ./internal/ports ./internal/adapters/agent/fake ./internal/adapters/agent/codex ./internal/adapters/state/sqlite ./internal/interfaces/mcp ./internal/bootstrap ./cmd/orquesta"
}

func v06AssertPublicContractShape(t *testing.T, fixture v06Fixture) {
	t.Helper()
	v06RequireFields(t, reflect.TypeOf(application.ExecutionRecord{}), map[string]reflect.Type{
		"AttemptNo":            reflect.TypeOf(uint64(0)),
		"MaxExecutionAttempts": reflect.TypeOf(uint64(0)),
		"ReplacesExecutionRef": reflect.TypeOf(goal.ExecutionRef{}),
		"PlanGeneration":       reflect.TypeOf(goal.PlanGeneration(0)),
		"AppSpecGeneration":    reflect.TypeOf(goal.AppSpecGeneration(0)),
		"SpecHash":             reflect.TypeOf(""),
		"ModelRef":             reflect.TypeOf(""),
		"AgentRef":             reflect.TypeOf(""),
	})
	if _, legacyLimit := reflect.TypeOf(application.ExecutionRecord{}).FieldByName("MaxAttempts"); legacyLimit {
		t.Fatal("ExecutionRecord still carries the conflated legacy delivery limit")
	}
	v06RequireFields(t, reflect.TypeOf(application.ActionClaim{}), map[string]reflect.Type{
		"DeliveryAttempt": reflect.TypeOf(uint64(0)),
		"Fence":           reflect.TypeOf(uint64(0)),
	})
	if _, conflated := reflect.TypeOf(application.ActionClaim{}).FieldByName("Attempt"); conflated {
		t.Fatal("ActionClaim still conflates delivery attempt with execution attempt")
	}
	v06RequireFields(t, reflect.TypeOf(application.ClaimRequest{}), map[string]reflect.Type{
		"Capabilities": reflect.TypeOf(ports.AgentCapabilities{}),
	})
	if _, callerClock := reflect.TypeOf(application.ClaimRequest{}).FieldByName("Now"); callerClock {
		t.Fatal("ClaimRequest lets callers control the repository lease clock")
	}
	v06RequireFields(t, reflect.TypeOf(application.ActionConsumptionReceipt{}), map[string]reflect.Type{
		"PlanGeneration":     reflect.TypeOf(goal.PlanGeneration(0)),
		"WorkItemGeneration": reflect.TypeOf(goal.Revision(0)),
		"Fence":              reflect.TypeOf(uint64(0)),
		"DeliveryAttempt":    reflect.TypeOf(uint64(0)),
		"Outcome":            reflect.TypeOf(application.ActionConsumptionOutcome("")),
	})
	v06RequireFields(t, reflect.TypeOf(ports.AgentLaunchReceipt{}), map[string]reflect.Type{
		"GoalRef":           reflect.TypeOf(goal.GoalRef{}),
		"WorkItemRef":       reflect.TypeOf(goal.WorkItemRef{}),
		"PlanGeneration":    reflect.TypeOf(goal.PlanGeneration(0)),
		"AppSpecGeneration": reflect.TypeOf(goal.AppSpecGeneration(0)),
		"ExecutionAttempt":  reflect.TypeOf(uint64(0)),
		"ModelRef":          reflect.TypeOf(""),
		"AgentRef":          reflect.TypeOf(""),
	})
	statePort := reflect.TypeOf((*application.StateRepository)(nil)).Elem()
	dependencies := reflect.TypeOf(application.Dependencies{})
	if _, deadLimit := dependencies.FieldByName("MaxActionAttempts"); deadLimit {
		t.Fatal("application.Dependencies still exposes an ineffective delivery-attempt limit")
	}
	stateFields := 0
	for index := 0; index < dependencies.NumField(); index++ {
		if dependencies.Field(index).Type == statePort {
			stateFields++
		}
	}
	if stateFields != 1 {
		t.Fatalf("application.Dependencies injects %d StateRepository values, want exactly one", stateFields)
	}
	if _, exists := statePort.MethodByName("RecordExecutionReplaced"); !exists {
		t.Fatal("StateRepository lacks atomic execution replacement mutation")
	}
	var repository application.StateRepository = (*sqlite.Repository)(nil)
	if reflect.TypeOf(repository) != reflect.TypeOf((*sqlite.Repository)(nil)) {
		t.Fatal("SQLite is not selected through StateRepository port")
	}
	requirements := v06AgentRequirements(fixture.OpaqueRequirements)
	missing := v06Capabilities(fixture.OpaqueRequirements, false)
	matching := v06Capabilities(fixture.OpaqueRequirements, true)
	if ports.MatchAgentCapabilities(missing, requirements) || !ports.MatchAgentCapabilities(matching, requirements) {
		t.Fatal("opaque capability matching failed closed/open for wrong worker")
	}
}

func v06RequireFields(t *testing.T, value reflect.Type, required map[string]reflect.Type) {
	t.Helper()
	for name, wantType := range required {
		field, found := value.FieldByName(name)
		if !found || field.Type != wantType {
			t.Errorf("%s.%s type = %v/found=%v, want %v", value, name, field.Type, found, wantType)
		}
	}
}

func v06AssertAtomicCreate(t *testing.T, fixture v06Fixture) {
	t.Helper()
	ctx := context.Background()
	clock := v06ClockFromFixture(t, fixture)
	databasePath := v06PrivateDatabasePath(t, "atomic.db")
	repository := v06OpenSQLite(t, ctx, databasePath, clock)
	defer repository.Close()
	v06AssertSQLiteWAL(t, databasePath)

	raw, err := sql.Open("sqlite", databasePath)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	if _, err := raw.ExecContext(ctx, `
CREATE TRIGGER v06_abort_event BEFORE INSERT ON events
BEGIN
  SELECT RAISE(ABORT, 'v06.atomic.failpoint');
END`); err != nil {
		t.Fatalf("install atomic failpoint: %v", err)
	}

	agent := newV06Agent(clock, v06Capabilities(fixture.OpaqueRequirements, true), "accepted")
	orchestrator := v06NewOrchestrator(t, repository, clock, &v06IDs{}, agent, newV06ArtifactStore(), fixture)
	request := v06SubmitRequest(t, "request:v06-atomic", nil)
	if _, err := orchestrator.Submit(ctx, request); err == nil {
		t.Fatal("event failpoint allowed partial Goal creation")
	}
	for _, table := range []string{"goals", "work_items", "executions", "events", "outbox"} {
		if count := v06TableCount(t, raw, table); count != 0 {
			t.Errorf("atomic rollback left %s rows = %d", table, count)
		}
	}
	if _, err := raw.ExecContext(ctx, `DROP TRIGGER v06_abort_event`); err != nil {
		t.Fatal(err)
	}
	created, err := orchestrator.Submit(ctx, request)
	if err != nil || !created.Created {
		t.Fatalf("create after rollback: created=%v err=%v", created.Created, err)
	}
	wantCounts := map[string]int{"goals": 1, "work_items": 1, "executions": 1, "events": 2, "outbox": 1}
	for table, want := range wantCounts {
		if got := v06TableCount(t, raw, table); got != want {
			t.Errorf("%s rows = %d, want %d", table, got, want)
		}
	}
	replayed, err := orchestrator.Submit(ctx, request)
	if err != nil || replayed.Created || replayed.Record.Goal.Ref() != created.Record.Goal.Ref() {
		t.Fatalf("idempotent replay created another Goal: %+v err=%v", replayed, err)
	}
	for table, want := range wantCounts {
		if got := v06TableCount(t, raw, table); got != want {
			t.Errorf("replay changed %s rows = %d, want %d", table, got, want)
		}
	}
}

func v06AssertReplaceableAttemptsAndRestart(t *testing.T, fixture v06Fixture) {
	t.Helper()
	ctx := context.Background()
	clock := v06ClockFromFixture(t, fixture)
	databasePath := v06PrivateDatabasePath(t, "attempts.db")
	repository := v06OpenSQLite(t, ctx, databasePath, clock)
	ids := &v06IDs{}
	artifacts := newV06ArtifactStore()
	agent := newV06Agent(clock, v06Capabilities(fixture.OpaqueRequirements, true), fixture.ClockAndRetry.LaunchOutcomes...)
	orchestrator := v06NewOrchestrator(t, repository, clock, ids, agent, artifacts, fixture)

	created, err := orchestrator.Submit(ctx, v06SubmitRequest(t, "request:v06-attempts", nil))
	if err != nil {
		t.Fatal(err)
	}
	goalRef := created.Record.Goal.Ref()
	firstResult, err := orchestrator.ProcessNext(ctx, "worker:v06")
	if err != nil || !firstResult.Processed {
		t.Fatalf("temporary delivery: result=%+v err=%v", firstResult, err)
	}
	record := v06GetGoal(t, repository, goalRef)
	if len(record.Executions) != 1 || record.Executions[0].AttemptNo != 1 || len(record.ConsumptionReceipts) != 0 {
		t.Fatalf("delivery retry became execution/receipt: executions=%+v receipts=%+v", record.Executions, record.ConsumptionReceipts)
	}

	clock.Advance(v06Duration(t, fixture.ClockAndRetry.ObservationDelay))
	secondResult, err := orchestrator.ProcessNext(ctx, "worker:v06")
	if err != nil || !secondResult.Processed {
		t.Fatalf("provider failure replacement: result=%+v err=%v", secondResult, err)
	}
	record = v06GetGoal(t, repository, goalRef)
	executions := v06ExecutionsByAttempt(record.Executions)
	if len(executions) != 2 || executions[0].AttemptNo != 1 || executions[1].AttemptNo != 2 ||
		executions[0].State != application.ExecutionFailed || executions[1].State != application.ExecutionDispatching ||
		executions[1].ReplacesExecutionRef != executions[0].Ref || record.Goal.IsTerminal() {
		t.Fatalf("provider failure did not create replaceable attempt: goal=%s executions=%+v", record.Goal.State(), executions)
	}
	firstConsumption := v06ReceiptForExecution(t, record.ConsumptionReceipts, executions[0].Ref, application.ActionLaunchAgent)
	if firstConsumption.DeliveryAttempt != 2 || firstConsumption.Fence < 2 || firstConsumption.Outcome != application.ActionConsumedCompleted {
		t.Fatalf("delivery and execution attempts conflated: %+v", firstConsumption)
	}

	clock.Advance(v06Duration(t, fixture.ClockAndRetry.ObservationDelay))
	thirdResult, err := orchestrator.ProcessNext(ctx, "worker:v06")
	if err != nil || !thirdResult.Processed {
		t.Fatalf("replacement launch: result=%+v err=%v", thirdResult, err)
	}
	record = v06GetGoal(t, repository, goalRef)
	executions = v06ExecutionsByAttempt(record.Executions)
	accepted := executions[1]
	if accepted.State != application.ExecutionRunning || accepted.ProviderRef != "provider:v06" ||
		accepted.ModelRef != "model:v06" || accepted.AgentRef != "agent:v06" ||
		accepted.PlanGeneration != record.Goal.PlanGeneration() ||
		accepted.AppSpecGeneration != record.Goal.AppSpec().Generation() || accepted.SpecHash != record.Goal.SpecHash() {
		t.Fatalf("incomplete durable launch receipt: %+v", accepted)
	}
	if err := repository.Close(); err != nil {
		t.Fatal(err)
	}

	repository = v06OpenSQLite(t, ctx, databasePath, clock)
	defer repository.Close()
	restarted := v06GetGoal(t, repository, goalRef)
	restartedExecutions := v06ExecutionsByAttempt(restarted.Executions)
	if !reflect.DeepEqual(restartedExecutions, executions) || len(restarted.ConsumptionReceipts) != 2 {
		t.Fatalf("restart changed attempts/receipts: executions=%+v receipts=%+v", restartedExecutions, restarted.ConsumptionReceipts)
	}

	clock.Advance(v06Duration(t, fixture.ClockAndRetry.ObservationDelay))
	orchestrator = v06NewOrchestrator(t, repository, clock, ids, agent, artifacts, fixture)
	observed, err := orchestrator.ProcessNext(ctx, "worker:v06-restart")
	if err != nil || !observed.Processed || observed.Action != application.ActionObserveAgent {
		t.Fatalf("restart observation: result=%+v err=%v", observed, err)
	}
	terminal := v06GetGoal(t, repository, goalRef)
	if terminal.Goal.State() != goal.GoalStateSucceeded || len(terminal.Executions) != 2 || len(terminal.ConsumptionReceipts) != 3 {
		t.Fatalf("terminal closure/recovery mismatch: state=%s executions=%d receipts=%d", terminal.Goal.State(), len(terminal.Executions), len(terminal.ConsumptionReceipts))
	}
	if replay, err := orchestrator.ProcessNext(ctx, "worker:v06-restart"); err != nil || replay.Processed {
		t.Fatalf("terminal work reexecuted: result=%+v err=%v", replay, err)
	}
}

func v06AssertFencedClaimAndReceipt(t *testing.T, fixture v06Fixture) {
	t.Helper()
	ctx := context.Background()
	clock := v06ClockFromFixture(t, fixture)
	lease := v06Duration(t, fixture.ClockAndRetry.ClaimLease)
	databasePath := v06PrivateDatabasePath(t, "fence.db")
	first := v06OpenSQLite(t, ctx, databasePath, clock)
	second := v06OpenSQLite(t, ctx, databasePath, clock)
	agent := newV06Agent(clock, v06Capabilities(fixture.OpaqueRequirements, true), "accepted")
	orchestrator := v06NewOrchestrator(t, first, clock, &v06IDs{}, agent, newV06ArtifactStore(), fixture)
	created, err := orchestrator.Submit(ctx, v06SubmitRequest(t, "request:v06-fence", nil))
	if err != nil {
		t.Fatal(err)
	}

	type claimResult struct {
		claim application.ActionClaim
		found bool
		err   error
	}
	start := make(chan struct{})
	results := make(chan claimResult, fixture.ClockAndRetry.ClaimContenders)
	capabilities := v06Capabilities(fixture.OpaqueRequirements, true)
	for index := 0; index < fixture.ClockAndRetry.ClaimContenders; index++ {
		repository := first
		if index%2 == 1 {
			repository = second
		}
		go func(index int, repository *sqlite.Repository) {
			<-start
			claim, found, err := repository.ClaimNextAction(ctx, application.ClaimRequest{
				WorkerRef: fmt.Sprintf("worker:v06-%02d", index), Token: fmt.Sprintf("claim:v06-%02d", index),
				LeaseDuration: lease, Capabilities: capabilities,
			})
			results <- claimResult{claim: claim, found: found, err: err}
		}(index, repository)
	}
	close(start)
	var winners []application.ActionClaim
	for index := 0; index < fixture.ClockAndRetry.ClaimContenders; index++ {
		result := <-results
		if result.err != nil {
			t.Errorf("claim contender failed: %v", result.err)
		}
		if result.found {
			winners = append(winners, result.claim)
		}
	}
	if len(winners) != 1 {
		t.Fatalf("active claim winners = %d, want 1: %+v", len(winners), winners)
	}
	stale := winners[0]
	clock.Advance(lease + time.Nanosecond)
	reclaimed, found, err := second.ClaimNextAction(ctx, application.ClaimRequest{
		WorkerRef: "worker:v06-reclaimer", Token: "claim:v06-reclaimer", LeaseDuration: lease, Capabilities: capabilities,
	})
	if err != nil || !found || reclaimed.Action.Ref != stale.Action.Ref || reclaimed.Fence != stale.Fence+1 ||
		reclaimed.DeliveryAttempt != stale.DeliveryAttempt+1 {
		t.Fatalf("reclaim is not monotonic/exact: stale=%+v reclaimed=%+v found=%v err=%v", stale, reclaimed, found, err)
	}
	operationAt := clock.Now()
	err = first.QuarantineAction(ctx, application.ActionQuarantinedState{
		Claim: stale, ErrorCode: "v06.stale",
		Event: application.EventRecord{
			Ref: "event:action-quarantined:v06-stale", Kind: "action.quarantined",
			GoalRef: stale.Action.GoalRef, WorkItemRef: stale.Action.WorkItemRef,
			ExecutionRef: stale.Action.ExecutionRef, OccurredAt: operationAt,
		},
		OperationAt: operationAt,
	})
	if !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("stale fence mutation = %v, want state conflict", err)
	}
	if receipts := v06GetGoal(t, first, created.Record.Goal.Ref()).ConsumptionReceipts; len(receipts) != 0 {
		t.Fatalf("stale mutation wrote partial receipt: %+v", receipts)
	}
	if status, err := first.Status(ctx); err != nil || status.PendingActions != 1 || status.QuarantinedActions != 0 {
		t.Fatalf("stale mutation changed durable outbox: status=%+v err=%v", status, err)
	}

	event := application.EventRecord{
		Ref: "event:action-quarantined:v06-fence", Kind: "action.quarantined",
		GoalRef: reclaimed.Action.GoalRef, WorkItemRef: reclaimed.Action.WorkItemRef,
		ExecutionRef: reclaimed.Action.ExecutionRef, OccurredAt: operationAt,
	}
	quarantine := application.ActionQuarantinedState{
		Claim: reclaimed, ErrorCode: "v06.test_quarantine", Event: event, OperationAt: operationAt,
	}
	if err := second.QuarantineAction(ctx, quarantine); err != nil {
		t.Fatalf("consume reclaimed action: %v", err)
	}
	after := v06GetGoal(t, first, created.Record.Goal.Ref())
	if len(after.ConsumptionReceipts) != 1 {
		t.Fatalf("consumption receipt count = %d, want 1", len(after.ConsumptionReceipts))
	}
	receipt := after.ConsumptionReceipts[0]
	if receipt.ActionRef != reclaimed.Action.Ref || receipt.Fence != reclaimed.Fence ||
		receipt.DeliveryAttempt != reclaimed.DeliveryAttempt || receipt.ClaimToken != reclaimed.Token ||
		receipt.WorkerRef != reclaimed.WorkerRef || receipt.Outcome != application.ActionConsumedQuarantined ||
		receipt.PlanGeneration != reclaimed.Action.PlanGeneration ||
		receipt.WorkItemGeneration != reclaimed.Action.WorkItemGeneration {
		t.Fatalf("incomplete causal consumption receipt: %+v", receipt)
	}
	if err := second.QuarantineAction(ctx, quarantine); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("consumption replay = %v, want conflict without duplicate", err)
	}
	if got := len(v06GetGoal(t, first, created.Record.Goal.Ref()).ConsumptionReceipts); got != 1 {
		t.Fatalf("consumption replay duplicated receipt: %d", got)
	}
	raw, err := sql.Open("sqlite", databasePath)
	if err != nil {
		t.Fatal(err)
	}
	for name, statement := range map[string]string{
		"update": `UPDATE action_consumption_receipts SET error_code = 'tampered' WHERE action_ref = ?`,
		"delete": `DELETE FROM action_consumption_receipts WHERE action_ref = ?`,
	} {
		if _, err := raw.ExecContext(ctx, statement, receipt.ActionRef); err == nil {
			t.Errorf("immutable receipt allowed %s", name)
		}
	}
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	if err := second.Close(); err != nil {
		t.Fatal(err)
	}
	restarted := v06OpenSQLite(t, ctx, databasePath, clock)
	defer restarted.Close()
	restartedReceipts := v06GetGoal(t, restarted, created.Record.Goal.Ref()).ConsumptionReceipts
	if len(restartedReceipts) != 1 || !reflect.DeepEqual(restartedReceipts[0], receipt) {
		t.Fatalf("restart changed immutable receipt: %+v want %+v", restartedReceipts, receipt)
	}
}

func v06AssertCapabilityMatchingAndNoPrivateQueue(t *testing.T, fixture v06Fixture) {
	t.Helper()
	ctx := context.Background()
	clock := v06ClockFromFixture(t, fixture)
	databasePath := v06PrivateDatabasePath(t, "capability.db")
	repository := v06OpenSQLite(t, ctx, databasePath, clock)
	ids := &v06IDs{}
	artifacts := newV06ArtifactStore()
	missingCapabilities := v06Capabilities(fixture.OpaqueRequirements, false)
	missingAgent := newV06Agent(clock, missingCapabilities, "accepted")
	orchestrator := v06NewOrchestrator(t, repository, clock, ids, missingAgent, artifacts, fixture)
	created, err := orchestrator.Submit(ctx, v06SubmitRequest(t, "request:v06-capability", &fixture.OpaqueRequirements))
	if err != nil {
		t.Fatal(err)
	}
	if result, err := orchestrator.ProcessNext(ctx, "worker:v06-missing"); err != nil || result.Processed {
		t.Fatalf("worker without opaque requirement claimed work: result=%+v err=%v", result, err)
	}
	if err := repository.Close(); err != nil {
		t.Fatal(err)
	}

	repository = v06OpenSQLite(t, ctx, databasePath, clock)
	defer repository.Close()
	matchingAgent := newV06Agent(clock, v06Capabilities(fixture.OpaqueRequirements, true), "accepted")
	orchestrator = v06NewOrchestrator(t, repository, clock, ids, matchingAgent, artifacts, fixture)
	result, err := orchestrator.ProcessNext(ctx, "worker:v06-matching")
	if err != nil || !result.Processed || result.GoalRef != created.Record.Goal.Ref() || result.Action != application.ActionLaunchAgent {
		t.Fatalf("matching worker did not recover durable work: result=%+v err=%v", result, err)
	}
	record := v06GetGoal(t, repository, created.Record.Goal.Ref())
	if len(record.Executions) != 1 || record.Executions[0].State != application.ExecutionRunning {
		t.Fatalf("durable capability-selected action not launched: %+v", record.Executions)
	}
}

func v06SubmitRequest(t *testing.T, requestRef string, requirements *v06OpaqueRequirements) application.SubmitRequest {
	t.Helper()
	actor, err := goal.NewActorRef("actor:v06")
	if err != nil {
		t.Fatal(err)
	}
	project, err := goal.NewProjectRef("project:v06")
	if err != nil {
		t.Fatal(err)
	}
	request := application.SubmitRequest{
		RequestRef: requestRef, ActorRef: actor, ProjectRef: project,
		Statement: "acreditar estado y outbox atómicos", Confirm: true,
	}
	if requirements != nil {
		request.Plan = &application.PlanSpec{
			Phases: []application.PhaseSpec{{
				Ref: "phase-instance:v06", Key: "phase:v06", TemplateRef: "phase-template:v06",
			}},
			WorkItems: []application.WorkItemSpec{{
				Key: "work:v06", Objective: "probar matching opaco", Phase: "phase:v06", Role: requirements.RoleKey,
				SkillRefs: requirements.SkillRefs, ToolRefs: requirements.ToolRefs,
				CapabilityRefs: requirements.CapabilityRefs, OutputContract: goal.OutputContractEvidenceBundle,
			}},
		}
	}
	return request
}

func v06NewOrchestrator(
	t *testing.T,
	repository application.StateRepository,
	clock *v06Clock,
	ids *v06IDs,
	agent *v06Agent,
	artifacts *v06ArtifactStore,
	fixture v06Fixture,
) *application.Orchestrator {
	t.Helper()
	orchestrator, err := application.New(application.Dependencies{
		State: repository, Launcher: agent, Observer: agent, Artifacts: artifacts, Clock: clock, IDs: ids,
		MaxOutputBytes:       1 << 20,
		MaxExecutionAttempts: fixture.ClockAndRetry.MaxExecutionAttempts,
		ClaimLease:           v06Duration(t, fixture.ClockAndRetry.ClaimLease),
		ObservationDelay:     v06Duration(t, fixture.ClockAndRetry.ObservationDelay),
		ExecutionTimeout:     v06Duration(t, fixture.ClockAndRetry.ExecutionTimeout),
		AgentCapabilities:    agent.capabilities,
	})
	if err != nil {
		t.Fatal(err)
	}
	return orchestrator
}

func v06OpenSQLite(t *testing.T, ctx context.Context, path string, clock *v06Clock) *sqlite.Repository {
	t.Helper()
	repository, err := sqlite.Open(ctx, sqlite.Options{
		Path: path, BusyTimeout: 5 * time.Second, MaxOpenConnections: 4, Now: clock.Now,
	})
	if err != nil {
		t.Fatalf("open V06 SQLite repository: %v: %v", err, errors.Unwrap(err))
	}
	return repository
}

func v06PrivateDatabasePath(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "private-state", name)
}

func v06AssertSQLiteWAL(t *testing.T, path string) {
	t.Helper()
	database, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	var mode string
	if err := database.QueryRow(`PRAGMA journal_mode`).Scan(&mode); err != nil || strings.ToLower(mode) != "wal" {
		t.Fatalf("SQLite journal mode = %q err=%v, want wal", mode, err)
	}
}

func v06TableCount(t *testing.T, database *sql.DB, table string) int {
	t.Helper()
	allowed := map[string]bool{"goals": true, "work_items": true, "executions": true, "events": true, "outbox": true}
	if !allowed[table] {
		t.Fatalf("non-allowlisted table %q", table)
	}
	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func v06GetGoal(t *testing.T, repository application.StateRepository, ref goal.GoalRef) application.GoalRecord {
	t.Helper()
	record, err := repository.GetGoal(context.Background(), ref)
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func v06ExecutionsByAttempt(values []application.ExecutionRecord) []application.ExecutionRecord {
	result := append([]application.ExecutionRecord(nil), values...)
	sort.Slice(result, func(left, right int) bool { return result[left].AttemptNo < result[right].AttemptNo })
	return result
}

func v06ReceiptForExecution(
	t *testing.T,
	values []application.ActionConsumptionReceipt,
	ref goal.ExecutionRef,
	kind application.ActionKind,
) application.ActionConsumptionReceipt {
	t.Helper()
	for _, receipt := range values {
		if receipt.ExecutionRef == ref && receipt.Kind == kind {
			return receipt
		}
	}
	t.Fatalf("consumption receipt for %s/%s not found: %+v", ref.String(), kind, values)
	return application.ActionConsumptionReceipt{}
}

func v06ClockFromFixture(t *testing.T, fixture v06Fixture) *v06Clock {
	t.Helper()
	value, err := time.Parse(time.RFC3339Nano, fixture.ClockAndRetry.BaseTime)
	if err != nil {
		t.Fatal(err)
	}
	return &v06Clock{now: value}
}

func v06Duration(t *testing.T, raw string) time.Duration {
	t.Helper()
	value, err := time.ParseDuration(raw)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func v06AgentRequirements(requirements v06OpaqueRequirements) ports.AgentRequirements {
	return ports.AgentRequirements{
		RoleKey: requirements.RoleKey, SkillRefs: append([]string(nil), requirements.SkillRefs...),
		ToolRefs:       append([]string(nil), requirements.ToolRefs...),
		CapabilityRefs: append([]string(nil), requirements.CapabilityRefs...),
	}
}

func v06Capabilities(requirements v06OpaqueRequirements, matching bool) ports.AgentCapabilities {
	capabilities := ports.AgentCapabilities{
		ProviderRef: "provider:v06", ModelRef: "model:v06", AgentRef: "agent:v06",
		RoleKeys:  []string{"role:worker", requirements.RoleKey},
		SkillRefs: append([]string(nil), requirements.SkillRefs...), ToolRefs: append([]string(nil), requirements.ToolRefs...),
	}
	if matching {
		capabilities.CapabilityRefs = append([]string(nil), requirements.CapabilityRefs...)
	}
	return capabilities
}

type v06Clock struct {
	mu  sync.Mutex
	now time.Time
}

func (clock *v06Clock) Now() time.Time {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	return clock.now
}

func (clock *v06Clock) Advance(duration time.Duration) {
	clock.mu.Lock()
	clock.now = clock.now.Add(duration)
	clock.mu.Unlock()
}

type v06IDs struct {
	mu   sync.Mutex
	next uint64
}

func (ids *v06IDs) NewID(ctx context.Context, prefix string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	ids.mu.Lock()
	defer ids.mu.Unlock()
	ids.next++
	return fmt.Sprintf("%s:v06-%04d", prefix, ids.next), nil
}

type v06TemporaryError struct{}

func (v06TemporaryError) Error() string   { return "v06.temporary" }
func (v06TemporaryError) Temporary() bool { return true }

type v06Agent struct {
	mu           sync.Mutex
	clock        *v06Clock
	capabilities ports.AgentCapabilities
	outcomes     []string
	launches     int
	requests     map[goal.ExecutionRef]ports.AgentLaunchRequest
}

func newV06Agent(clock *v06Clock, capabilities ports.AgentCapabilities, outcomes ...string) *v06Agent {
	return &v06Agent{
		clock: clock, capabilities: capabilities, outcomes: append([]string(nil), outcomes...),
		requests: make(map[goal.ExecutionRef]ports.AgentLaunchRequest),
	}
}

func (agent *v06Agent) Capabilities(context.Context) (ports.AgentCapabilities, error) {
	return agent.capabilities, nil
}

func (agent *v06Agent) Launch(_ context.Context, request ports.AgentLaunchRequest) (ports.AgentLaunchReceipt, error) {
	agent.mu.Lock()
	defer agent.mu.Unlock()
	agent.launches++
	outcome := "accepted"
	if agent.launches <= len(agent.outcomes) {
		outcome = agent.outcomes[agent.launches-1]
	}
	switch outcome {
	case "temporary_error":
		return ports.AgentLaunchReceipt{}, v06TemporaryError{}
	case "permanent_error":
		return ports.AgentLaunchReceipt{}, errors.New("v06.permanent")
	case "accepted":
	default:
		return ports.AgentLaunchReceipt{}, fmt.Errorf("v06.unknown_launch_outcome:%s", outcome)
	}
	agent.requests[request.ExecutionRef] = request
	return ports.AgentLaunchReceipt{
		ExecutionRef: request.ExecutionRef, GoalRef: request.GoalRef, WorkItemRef: request.WorkItemRef,
		PlanGeneration: request.PlanGeneration, AppSpecGeneration: request.AppSpecGeneration,
		ExecutionAttempt: request.ExecutionAttempt, SpecHash: request.SpecHash,
		ProviderRef: "provider:v06", ModelRef: "model:v06", AgentRef: "agent:v06",
		ExternalRef: "external:" + request.ExecutionRef.String(), IdempotencyKey: request.IdempotencyKey,
		AcceptedAt: agent.clock.Now(),
	}, nil
}

func (agent *v06Agent) Observe(_ context.Context, executionRef goal.ExecutionRef) (ports.AgentObservation, error) {
	agent.mu.Lock()
	request, found := agent.requests[executionRef]
	agent.mu.Unlock()
	if !found {
		return ports.AgentObservation{}, errors.New("v06.execution_not_found")
	}
	return ports.AgentObservation{
		ExecutionRef: executionRef, SpecHash: request.SpecHash, Status: ports.AgentCompleted,
		MediaType: "text/plain", Content: []byte("V06 acreditado"), ObservedAt: agent.clock.Now(),
	}, nil
}

type v06ArtifactStore struct {
	mu      sync.Mutex
	content map[goal.ArtifactRef]ports.ArtifactContent
}

func newV06ArtifactStore() *v06ArtifactStore {
	return &v06ArtifactStore{content: make(map[goal.ArtifactRef]ports.ArtifactContent)}
}

func (store *v06ArtifactStore) Put(_ context.Context, request ports.PutArtifactRequest) (ports.StoredArtifact, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	digest := sha256.Sum256(request.Content)
	digestText := hex.EncodeToString(digest[:])
	ref, err := goal.NewArtifactRef("artifact:sha256:" + digestText)
	if err != nil {
		return ports.StoredArtifact{}, err
	}
	stored := ports.StoredArtifact{
		Ref: ref, Digest: digestText, MediaType: request.MediaType, Size: int64(len(request.Content)),
	}
	store.content[ref] = ports.ArtifactContent{
		Ref: ref, Digest: digestText, MediaType: request.MediaType, Size: stored.Size,
		Content: append([]byte(nil), request.Content...),
	}
	return stored, nil
}

func (store *v06ArtifactStore) Get(_ context.Context, ref goal.ArtifactRef, expectedSize int64) (ports.ArtifactContent, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	content, found := store.content[ref]
	if !found {
		return ports.ArtifactContent{}, errors.New("v06.artifact_not_found")
	}
	if content.Size != expectedSize {
		return ports.ArtifactContent{}, errors.New("v06.artifact_size_mismatch")
	}
	content.Content = append([]byte(nil), content.Content...)
	return content, nil
}
