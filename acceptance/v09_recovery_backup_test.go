package acceptance_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	artifactfs "orquesta/internal/adapters/artifact/filesystem"
	jsonimport "orquesta/internal/adapters/config/jsonimport"
	statesqlite "orquesta/internal/adapters/state/sqlite"
	"orquesta/internal/application"
	"orquesta/internal/config"
	"orquesta/internal/goal"
)

const v09FixturePath = "acceptance/fixtures/v09_recovery_backup.json"
const v09TrustedBaseGitCommitOID = "e198a02fd3b1f5c0d57f9fe2c427401af65ac1ee"
const v09ProductDeltaBaseGitCommitOID = "41c4702db97ca148109ce26aafca3bcf85c49c4a"
const v09ProductDeltaSealedGitCommitOID = "5cc86a9b1b1153f381080e64afbc526d403df385"
const v09RealLegacyFixtureSHA256 = "sha256:491df37cdffa509d8a31cae5a7ec1416f7e35df8aac6af6d523f375439548538"

type v09Fixture struct {
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
	DeferredCapabilities           []v09DeferredCapability `json:"deferred_capabilities"`
	Scenario                       v09Scenario             `json:"scenario"`
	Assertions                     []string                `json:"assertions"`
}

type v09DeferredCapability struct {
	ID                 string `json:"id"`
	Owner              string `json:"owner"`
	AcceptanceContract string `json:"acceptance_contract"`
}

type v09Scenario struct {
	BaseTime                string   `json:"base_time"`
	BackupMediaType         string   `json:"backup_media_type"`
	SchemaRefPrefix         string   `json:"schema_ref_prefix"`
	BackupRefPrefix         string   `json:"backup_ref_prefix"`
	TargetRef               string   `json:"target_ref"`
	Failpoints              []string `json:"failpoints"`
	RealLegacyFixture       string   `json:"real_legacy_fixture"`
	RealLegacyFixtureSHA256 string   `json:"real_legacy_fixture_sha256"`
	RealLegacyLeafPaths     []string `json:"real_legacy_leaf_paths"`
}

type v09BackupManifest struct {
	SchemaVersion         int       `json:"schema_version"`
	BackupRef             string    `json:"backup_ref"`
	PayloadSHA256         string    `json:"payload_sha256"`
	LogicalSHA256         string    `json:"logical_sha256"`
	MediaType             string    `json:"media_type"`
	Size                  int64     `json:"size"`
	SchemaRef             string    `json:"schema_ref"`
	CreatedAt             time.Time `json:"created_at"`
	CredentialsIncluded   bool      `json:"credentials_included"`
	ArtifactBlobsIncluded bool      `json:"artifact_blobs_included"`
}

type v09ProbeRecovery struct{}

func (*v09ProbeRecovery) CreateBackup(context.Context) (application.BackupReceipt, error) {
	return application.BackupReceipt{}, errors.New("probe.create")
}

func (*v09ProbeRecovery) VerifyBackup(context.Context, application.BackupRef) (application.BackupVerification, error) {
	return application.BackupVerification{}, errors.New("probe.verify")
}

func (*v09ProbeRecovery) RestoreBackup(
	context.Context,
	application.BackupRef,
	application.RecoveryTargetRef,
) (application.RestoreReceipt, error) {
	return application.RestoreReceipt{}, errors.New("probe.restore")
}

var _ application.StateRecovery = (*v09ProbeRecovery)(nil)

func TestAcceptanceV09RecoveryBackup(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v09Fixture](t, filepath.Join(repositoryRoot, filepath.FromSlash(v09FixturePath)))
	v09AssertFixtureHeader(t, repositoryRoot, fixture)

	t.Run("neutral_state_recovery_port_without_second_authority", func(t *testing.T) {
		v09AssertPortShape(t)
	})
	t.Run("online_backup_and_offline_restore_preserve_semantics", func(t *testing.T) {
		v09AssertBackupRestoreRoundTrip(t, fixture)
	})
	t.Run("concurrent_commit_is_wholly_before_or_after_snapshot", func(t *testing.T) {
		v09AssertConcurrentBackup(t, fixture)
	})
	t.Run("corruption_cancellation_and_unsafe_paths_fail_before_publication", func(t *testing.T) {
		v09AssertRecoveryNegatives(t, fixture)
	})
	t.Run("real_schema_json_to_toml_is_exhaustive_and_one_shot", func(t *testing.T) {
		v09AssertJSONImport(t, repositoryRoot, fixture)
	})
}

func TestAcceptanceV09RecoveryBackupReceipt(t *testing.T) {
	evidenceAssertReceiptV3(t, evidenceRepositoryRoot(t), evidenceReceiptV3Expectation{
		Contract: "AC-V09-RECOVERY-BACKUP", FixturePath: v09FixturePath,
		ReceiptPath:       "product/evidence/v09_recovery_backup.json",
		ExecutedNotBefore: "2026-07-15T00:00:00Z", TrustedBaseGitCommitOID: v09TrustedBaseGitCommitOID,
	})
}

func TestV09CandidateSubjectsCoverCommittedDelta(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v09Fixture](t, filepath.Join(repositoryRoot, filepath.FromSlash(v09FixturePath)))
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
		t.Fatalf("V09 candidate subjects differ from sealed product delta:\nchanged=%v\nfixture=%v", changed, fixture.CandidateSubjects)
	}
}

func v09AssertFixtureHeader(t *testing.T, repositoryRoot string, fixture v09Fixture) {
	t.Helper()
	wantDeferred := []v09DeferredCapability{
		{ID: "OPS-11", Owner: "postgres_s3_multihost", AcceptanceContract: "AC-V31-POSTGRES-S3-MULTIHOST"},
		{ID: "OPS-13", Owner: "postgres_s3_multihost", AcceptanceContract: "AC-V31-POSTGRES-S3-MULTIHOST"},
		{ID: "OPS-15", Owner: "operations_telemetry", AcceptanceContract: "AC-V32-OPERATIONS-TELEMETRY"},
		{ID: "OPS-16", Owner: "operations_telemetry", AcceptanceContract: "AC-V32-OPERATIONS-TELEMETRY"},
	}
	wantLegacyPaths := []string{
		"autoprogramming.checkpoint_only_high_consumption_tokens",
		"control_plane.remote_access_opt_in",
		"operator_director_mailbox.enabled",
		"runtime_models.allowed_models",
		"runtime_models.enabled",
		"schema_version",
		"server.addr",
		"server.state_dir",
	}
	if fixture.SchemaVersion != 1 || fixture.ReceiptSchemaVersion != 3 ||
		fixture.ContractID != "AC-V09-RECOVERY-BACKUP" || fixture.TrustedBaseGitCommitOID != v09TrustedBaseGitCommitOID ||
		fixture.ProductDeltaBaseGitCommitOID != v09ProductDeltaBaseGitCommitOID ||
		fixture.ProductDeltaSealedGitCommitOID != v09ProductDeltaSealedGitCommitOID ||
		fixture.OutputPath != "product/evidence/v09_recovery_backup.output.txt" ||
		fixture.ReceiptPath != "product/evidence/v09_recovery_backup.json" ||
		!reflect.DeepEqual(fixture.OwnedCapabilityIDs, []string{"EVD-15", "OPS-14"}) ||
		!reflect.DeepEqual(fixture.DeferredCapabilities, wantDeferred) || len(fixture.Assertions) != 12 ||
		fixture.Scenario.BackupMediaType != "application/vnd.sqlite3" ||
		fixture.Scenario.SchemaRefPrefix != "state-schema:sqlite:sha256:" ||
		fixture.Scenario.BackupRefPrefix != "backup:sha256:" || len(fixture.Scenario.Failpoints) != 5 ||
		fixture.Scenario.RealLegacyFixture != "acceptance/fixtures/v09_orquesta_config_v0_current_shape.json" ||
		fixture.Scenario.RealLegacyFixtureSHA256 != v09RealLegacyFixtureSHA256 ||
		!reflect.DeepEqual(fixture.Scenario.RealLegacyLeafPaths, wantLegacyPaths) {
		t.Fatalf("invalid V09 fixture header: %+v", fixture)
	}
	if _, err := time.Parse(time.RFC3339Nano, fixture.Scenario.BaseTime); err != nil {
		t.Fatalf("invalid V09 base time: %v", err)
	}
	legacyShape, err := os.ReadFile(filepath.Join(repositoryRoot, filepath.FromSlash(fixture.Scenario.RealLegacyFixture)))
	if err != nil || evidenceBytesSHA256(legacyShape) != fixture.Scenario.RealLegacyFixtureSHA256 {
		t.Fatalf("V09 legacy shape fixture drifted: digest=%s err=%v", evidenceBytesSHA256(legacyShape), err)
	}
	if fixture.Command != "sh -c '"+v09ValidationShellBody()+"'" ||
		!reflect.DeepEqual(fixture.ExecutionArgv, []string{"sh", "-c", v09ValidationShellBody()}) {
		t.Fatalf("invalid V09 command/argv: %q %#v", fixture.Command, fixture.ExecutionArgv)
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

func v09ValidationShellBody() string {
	return "go test -mod=vendor -count=1 . ./acceptance -run \"^(TestProductRoadmapIsExhaustiveAndCausal|TestProductRoadmapV09ScopeAndExecutableContract|TestV09EvidenceBelongsOnlyToRecoveryCapabilities|TestV09AcceptanceCommandRunsRecoveryConsumers|TestRebuildArchitecture|TestTraceabilityRebuildBugLessons|TestAcceptanceV09RecoveryBackup|TestV09CandidateSubjectsCoverCommittedDelta)$\"" +
		" && go test -mod=vendor -count=1 ./internal/application ./internal/ports ./internal/adapters/state/sqlite ./internal/adapters/artifact/filesystem ./internal/config ./internal/adapters/config/toml ./internal/adapters/config/jsonimport ./cmd/orquesta ./internal/bootstrap"
}

func v09AssertPortShape(t *testing.T) {
	t.Helper()
	port := reflect.TypeOf((*application.StateRecovery)(nil)).Elem()
	want := []string{"CreateBackup", "RestoreBackup", "VerifyBackup"}
	if port.NumMethod() != len(want) {
		t.Fatalf("StateRecovery methods=%d, want exactly %v", port.NumMethod(), want)
	}
	for _, method := range want {
		if _, found := port.MethodByName(method); !found {
			t.Errorf("StateRecovery lacks %s", method)
		}
	}
	state := reflect.TypeOf((*application.StateRepository)(nil)).Elem()
	for _, forbidden := range want {
		if _, found := state.MethodByName(forbidden); found {
			t.Errorf("StateRepository became an operational god interface through %s", forbidden)
		}
	}
	var interchangeable application.StateRecovery = &v09ProbeRecovery{}
	if interchangeable == nil {
		t.Fatal("StateRecovery cannot be replaced")
	}
	v09RequireExactPublicFields(t, reflect.TypeOf(application.BackupReceipt{}), []string{
		"CreatedAt", "ManifestSHA256", "MediaType", "Ref", "SchemaRef", "Size",
	})
	v09RequireExactPublicFields(t, reflect.TypeOf(application.BackupVerification{}), []string{
		"BackupRef", "ManifestSHA256", "SchemaRef", "VerifiedAt",
	})
	v09RequireExactPublicFields(t, reflect.TypeOf(application.RestoreReceipt{}), []string{
		"BackupRef", "ManifestSHA256", "RestoredAt", "SchemaRef", "TargetRef",
	})
	for _, value := range []any{
		application.BackupReceipt{}, application.BackupVerification{}, application.RestoreReceipt{},
	} {
		typ := reflect.TypeOf(value)
		for index := 0; index < typ.NumField(); index++ {
			field := typ.Field(index)
			lower := strings.ToLower(field.Name + " " + field.Type.String() + " " + field.Type.PkgPath())
			for _, forbidden := range []string{"path", "sqlite", "database/sql", "*sql.", "page"} {
				if strings.Contains(lower, forbidden) {
					t.Errorf("neutral %s exposes concrete field %s %s", typ.Name(), field.Name, field.Type)
				}
			}
		}
	}
}

func v09RequireExactPublicFields(t *testing.T, typ reflect.Type, want []string) {
	t.Helper()
	got := make([]string, 0, typ.NumField())
	for index := 0; index < typ.NumField(); index++ {
		field := typ.Field(index)
		if field.PkgPath != "" {
			t.Fatalf("%s contains unexported contract field %s", typ.Name(), field.Name)
		}
		got = append(got, field.Name)
	}
	sort.Strings(got)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s fields=%v want exact %v", typ.Name(), got, want)
	}
}

type v09StateFixture struct {
	clock        *v06Clock
	v06          v06Fixture
	databasePath string
	repository   *statesqlite.Repository
	orchestrator *application.Orchestrator
	artifacts    *artifactfs.Store
	terminalRef  goal.GoalRef
	pendingRef   goal.GoalRef
	pendingClaim application.ActionClaim
}

func v09CreateStateFixture(t *testing.T, fixture v09Fixture, root string) v09StateFixture {
	t.Helper()
	v06 := evidenceDecodeStrictJSON[v06Fixture](t, filepath.Join(evidenceRepositoryRoot(t), "acceptance/fixtures/v06_atomic_state_outbox.json"))
	clock := v06ClockFromFixture(t, v06)
	databasePath := filepath.Join(root, "state", "orquesta.sqlite")
	repository := v06OpenSQLite(t, context.Background(), databasePath, clock)
	artifacts, err := artifactfs.Open(filepath.Join(root, "artifacts"))
	if err != nil {
		t.Fatal(err)
	}
	orchestrator := v09NewOrchestrator(t, repository, clock, artifacts, v06)
	terminal, err := orchestrator.Submit(context.Background(), v06SubmitRequest(t, "request:v09-terminal", nil))
	if err != nil {
		t.Fatal(err)
	}
	if result, err := orchestrator.ProcessNext(context.Background(), "worker:v09"); err != nil || !result.Processed {
		t.Fatalf("launch terminal fixture: result=%+v err=%v", result, err)
	}
	clock.Advance(v06Duration(t, v06.ClockAndRetry.ObservationDelay))
	if result, err := orchestrator.ProcessNext(context.Background(), "worker:v09"); err != nil || !result.Processed {
		t.Fatalf("observe terminal fixture: result=%+v err=%v", result, err)
	}
	pending, err := orchestrator.Submit(context.Background(), v06SubmitRequest(t, "request:v09-pending", nil))
	if err != nil {
		t.Fatal(err)
	}
	pendingClaim, claimed, err := repository.ClaimNextAction(context.Background(), application.ClaimRequest{
		WorkerRef: "worker:v09-lease", Token: "claim-token:v09-lease",
		LeaseDuration: v06Duration(t, v06.ClockAndRetry.ClaimLease),
		Capabilities:  v06Capabilities(v06.OpaqueRequirements, true),
	})
	if err != nil || !claimed || pendingClaim.Action.GoalRef != pending.Record.Goal.Ref() {
		t.Fatalf("claim pending fixture: claim=%+v claimed=%v err=%v", pendingClaim, claimed, err)
	}
	return v09StateFixture{
		clock: clock, v06: v06, databasePath: databasePath,
		repository: repository, orchestrator: orchestrator, artifacts: artifacts,
		terminalRef: terminal.Record.Goal.Ref(), pendingRef: pending.Record.Goal.Ref(), pendingClaim: pendingClaim,
	}
}

func v09NewOrchestrator(
	t *testing.T,
	repository application.StateRepository,
	clock *v06Clock,
	artifacts application.ArtifactStore,
	fixture v06Fixture,
) *application.Orchestrator {
	t.Helper()
	agent := newV06Agent(clock, v06Capabilities(fixture.OpaqueRequirements, true), "accepted")
	orchestrator, err := application.New(application.Dependencies{
		State: repository, Launcher: agent, Observer: agent, Artifacts: artifacts,
		Clock: clock, IDs: &v06IDs{}, MaxOutputBytes: 1 << 20,
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

func (state v09StateFixture) close(t *testing.T) {
	t.Helper()
	if err := state.artifacts.Close(); err != nil {
		t.Errorf("close artifacts: %v", err)
	}
	if err := state.repository.Close(); err != nil {
		t.Errorf("close repository: %v", err)
	}
}

func v09AssertBackupRestoreRoundTrip(t *testing.T, fixture v09Fixture) {
	t.Helper()
	ctx := context.Background()
	root := t.TempDir()
	state := v09CreateStateFixture(t, fixture, root)
	defer state.close(t)
	recovery, err := statesqlite.NewRecovery(statesqlite.RecoveryOptions{
		Repository: state.repository, BackupRoot: filepath.Join(root, "backups"),
		RestoreRoot: filepath.Join(root, "restores"), Now: state.clock.Now,
	})
	if err != nil {
		t.Fatal(err)
	}

	beforeTerminal := v06GetGoal(t, state.repository, state.terminalRef)
	beforePending := v06GetGoal(t, state.repository, state.pendingRef)
	beforeStatus, err := state.repository.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}
	beforeLogicalState := v09SQLiteLogicalDump(t, state.databasePath)
	credentialSentinel := []byte("v09-private-credential-material-7f6d")
	credentialPath := filepath.Join(root, "secrets", "credentials.json")
	if err := os.MkdirAll(filepath.Dir(credentialPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(credentialPath, credentialSentinel, 0o600); err != nil {
		t.Fatal(err)
	}
	receipt, err := recovery.CreateBackup(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(receipt.Ref.String(), fixture.Scenario.BackupRefPrefix) ||
		receipt.ManifestSHA256 == "" || receipt.MediaType != fixture.Scenario.BackupMediaType ||
		receipt.Size <= 0 || !strings.HasPrefix(receipt.SchemaRef, fixture.Scenario.SchemaRefPrefix) || receipt.CreatedAt.IsZero() {
		t.Fatalf("incomplete backup receipt: %+v", receipt)
	}
	v09AssertBackupManifest(t, filepath.Join(root, "backups"), receipt, fixture)
	v09AssertBackupExclusions(t, filepath.Join(root, "backups"), beforeTerminal, credentialSentinel)
	verified, err := recovery.VerifyBackup(ctx, receipt.Ref)
	if err != nil || verified.BackupRef != receipt.Ref || verified.ManifestSHA256 != receipt.ManifestSHA256 ||
		verified.SchemaRef != receipt.SchemaRef || verified.VerifiedAt.IsZero() {
		t.Fatalf("backup verification mismatch: %+v err=%v", verified, err)
	}
	target, err := application.NewRecoveryTargetRef(fixture.Scenario.TargetRef)
	if err != nil {
		t.Fatal(err)
	}
	preexistingPath, err := recovery.TargetPath(target)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(preexistingPath), 0o700); err != nil {
		t.Fatal(err)
	}
	preexisting := []byte("v09-preexisting-target-must-survive")
	if err := os.WriteFile(preexistingPath, preexisting, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := recovery.RestoreBackup(ctx, receipt.Ref, target); err == nil {
		t.Fatal("restore overwrote a preexisting arbitrary target")
	}
	if content, err := os.ReadFile(preexistingPath); err != nil || !bytes.Equal(content, preexisting) {
		t.Fatalf("rejected restore changed preexisting target: %q err=%v", content, err)
	}
	if err := os.Remove(preexistingPath); err != nil {
		t.Fatal(err)
	}
	restored, err := recovery.RestoreBackup(ctx, receipt.Ref, target)
	if err != nil || restored.BackupRef != receipt.Ref || restored.TargetRef != target ||
		restored.ManifestSHA256 != receipt.ManifestSHA256 || restored.SchemaRef != receipt.SchemaRef || restored.RestoredAt.IsZero() {
		t.Fatalf("restore receipt mismatch: %+v err=%v", restored, err)
	}
	restoredPath, err := recovery.TargetPath(target)
	if err != nil {
		t.Fatal(err)
	}
	restoredRepository := v06OpenSQLite(t, ctx, restoredPath, state.clock)
	defer restoredRepository.Close()
	afterTerminal := v06GetGoal(t, restoredRepository, state.terminalRef)
	afterPending := v06GetGoal(t, restoredRepository, state.pendingRef)
	afterStatus, err := restoredRepository.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}
	afterLogicalState := v09SQLiteLogicalDump(t, restoredPath)
	if !reflect.DeepEqual(beforeTerminal, afterTerminal) || !reflect.DeepEqual(beforePending, afterPending) {
		t.Fatalf("restored semantic state differs:\nterminal=%+v\npending=%+v", afterTerminal, afterPending)
	}
	if !afterTerminal.Goal.IsTerminal() || afterPending.Goal.IsTerminal() {
		t.Fatalf("terminality changed: terminal=%s pending=%s", afterTerminal.Goal.State(), afterPending.Goal.State())
	}
	if !reflect.DeepEqual(beforeStatus, afterStatus) {
		t.Fatalf("repository status changed across restore: before=%+v after=%+v", beforeStatus, afterStatus)
	}
	if !bytes.Equal(beforeLogicalState, afterLogicalState) {
		t.Fatalf("events, outbox, claims, receipts or lifecycle rows changed across restore:\nbefore=%s\nafter=%s", beforeLogicalState, afterLogicalState)
	}
	if len(afterTerminal.Artifacts) == 0 {
		t.Fatal("terminal artifact metadata was lost")
	}
	artifact := afterTerminal.Artifacts[0].Stored
	if _, err := state.artifacts.Get(ctx, artifact.Ref, artifact.Size); err != nil {
		t.Fatalf("external artifact CAS dependency no longer resolves: %v", err)
	}
	state.clock.Advance(v06Duration(t, state.v06.ClockAndRetry.ClaimLease) + time.Nanosecond)
	if err := restoredRepository.RequeueAction(ctx, application.ActionRequeuedState{
		Claim: state.pendingClaim, Execution: beforePending.Executions[0],
		AvailableAt: state.clock.Now(), ErrorCode: "v09.stale_claim", OperationAt: state.clock.Now(),
	}); err == nil {
		t.Fatal("restored repository accepted the expired pre-backup claim")
	}
	restoredOrchestrator := v09NewOrchestrator(t, restoredRepository, state.clock, state.artifacts, state.v06)
	if processed, err := restoredOrchestrator.ProcessNext(ctx, "worker:v09-restored"); err != nil || !processed.Processed {
		t.Fatalf("restored pending action was not recoverable: result=%+v err=%v", processed, err)
	}
	state.clock.Advance(v06Duration(t, state.v06.ClockAndRetry.ObservationDelay))
	if processed, err := restoredOrchestrator.ProcessNext(ctx, "worker:v09-restored"); err != nil || !processed.Processed {
		t.Fatalf("restored pending observation did not close: result=%+v err=%v", processed, err)
	}
	if replay, err := restoredOrchestrator.ProcessNext(ctx, "worker:v09-restored"); err != nil || replay.Processed {
		t.Fatalf("restored state redelivered consumed work: result=%+v err=%v", replay, err)
	}
	if terminalAgain := v06GetGoal(t, restoredRepository, state.terminalRef); !reflect.DeepEqual(afterTerminal, terminalAgain) {
		t.Fatalf("processing pending work reexecuted terminal Goal: before=%+v after=%+v", afterTerminal, terminalAgain)
	}
	if _, err := recovery.RestoreBackup(ctx, receipt.Ref, target); err == nil {
		t.Fatal("restore overwrote an existing target")
	}
	if err := recovery.Close(); err != nil {
		t.Fatalf("close recovery: %v", err)
	}
	if _, err := recovery.CreateBackup(ctx); err == nil {
		t.Fatal("closed recovery accepted a new operation")
	}
	if _, err := recovery.VerifyBackup(ctx, receipt.Ref); err == nil {
		t.Fatal("closed recovery verified a backup")
	}
	closedTarget, err := application.NewRecoveryTargetRef("recovery-target:v09-after-close")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := recovery.RestoreBackup(ctx, receipt.Ref, closedTarget); err == nil {
		t.Fatal("closed recovery restored a backup")
	}
	if err := recovery.Close(); err != nil {
		t.Fatalf("idempotent recovery close: %v", err)
	}
}

func v09AssertConcurrentBackup(t *testing.T, fixture v09Fixture) {
	t.Helper()
	ctx := context.Background()
	root := t.TempDir()
	state := v09CreateStateFixture(t, fixture, root)
	defer state.close(t)
	var once sync.Once
	var concurrentRef goal.GoalRef
	var concurrentErr error
	recovery, err := statesqlite.NewRecovery(statesqlite.RecoveryOptions{
		Repository: state.repository, BackupRoot: filepath.Join(root, "backups"),
		RestoreRoot: filepath.Join(root, "restores"), Now: state.clock.Now,
		Failpoint: func(stage string) error {
			if stage != "after_first_backup_step" {
				return nil
			}
			once.Do(func() {
				created, err := state.orchestrator.Submit(ctx, v06SubmitRequest(t, "request:v09-concurrent", nil))
				concurrentErr = err
				if err == nil {
					concurrentRef = created.Record.Goal.Ref()
				}
			})
			return concurrentErr
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer recovery.Close()
	receipt, err := recovery.CreateBackup(ctx)
	if err != nil || concurrentErr != nil || concurrentRef.String() == "" {
		t.Fatalf("online backup/concurrent commit failed: receipt=%+v concurrent_ref=%s err=%v writer=%v", receipt, concurrentRef, err, concurrentErr)
	}
	target, err := application.NewRecoveryTargetRef("recovery-target:v09-concurrent")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := recovery.RestoreBackup(ctx, receipt.Ref, target); err != nil {
		t.Fatal(err)
	}
	targetPath, err := recovery.TargetPath(target)
	if err != nil {
		t.Fatal(err)
	}
	restored := v06OpenSQLite(t, ctx, targetPath, state.clock)
	defer restored.Close()
	status, err := restored.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if status.Goals != 2 && status.Goals != 3 {
		t.Fatalf("concurrent transaction was partially captured: status=%+v", status)
	}
	_, err = restored.GetGoal(ctx, concurrentRef)
	if status.Goals == 2 && !application.IsStateError(err, application.StateNotFound) {
		t.Fatalf("pre-commit snapshot contains partial concurrent Goal: %v", err)
	}
	if status.Goals == 3 && err != nil {
		t.Fatalf("post-commit snapshot lacks complete concurrent Goal: %v", err)
	}
}

func v09AssertRecoveryNegatives(t *testing.T, fixture v09Fixture) {
	t.Helper()
	ctx := context.Background()
	root := t.TempDir()
	state := v09CreateStateFixture(t, fixture, root)
	defer state.close(t)
	backupRoot := filepath.Join(root, "backups")
	restoreRoot := filepath.Join(root, "restores")
	recovery, err := statesqlite.NewRecovery(statesqlite.RecoveryOptions{
		Repository: state.repository, BackupRoot: backupRoot, RestoreRoot: restoreRoot, Now: state.clock.Now,
	})
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := recovery.CreateBackup(ctx)
	if err != nil {
		t.Fatal(err)
	}
	backupPath := v09FindOnlyFile(t, backupRoot, ".sqlite")
	manifestPath := v09FindOnlyFile(t, backupRoot, ".manifest.json")
	originalBackup, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	originalManifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := recovery.Close(); err != nil {
		t.Fatal(err)
	}
	linked := backupPath + ".hardlink"
	if err := os.Link(backupPath, linked); err != nil {
		t.Fatal(err)
	}
	recovery, err = statesqlite.NewRecovery(statesqlite.RecoveryOptions{
		Repository: state.repository, BackupRoot: backupRoot, RestoreRoot: restoreRoot, Now: state.clock.Now,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer recovery.Close()
	if _, err := recovery.VerifyBackup(ctx, receipt.Ref); err == nil {
		t.Fatal("hardlinked backup passed verification")
	}
	if err := os.Remove(linked); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, append(append([]byte(nil), originalManifest...), ' '), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := recovery.VerifyBackup(ctx, receipt.Ref); err == nil {
		t.Fatal("noncanonical or tampered manifest passed verification")
	}
	if err := os.WriteFile(manifestPath, originalManifest, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(backupPath, originalBackup[:len(originalBackup)/2], 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := recovery.VerifyBackup(ctx, receipt.Ref); err == nil {
		t.Fatal("truncated backup passed verification")
	}
	if err := os.WriteFile(backupPath, originalBackup, 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(backupPath, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteAt([]byte{0xff}, 128); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := recovery.VerifyBackup(ctx, receipt.Ref); err == nil {
		t.Fatal("corrupt backup passed verification")
	}
	target, err := application.NewRecoveryTargetRef("recovery-target:v09-corrupt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := recovery.RestoreBackup(ctx, receipt.Ref, target); err == nil {
		t.Fatal("corrupt backup was restored")
	}
	if files := v09FilesWithSuffix(t, restoreRoot, ".sqlite"); len(files) != 0 {
		t.Fatalf("failed restore published targets: %v", files)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := recovery.CreateBackup(canceled); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled backup err=%v", err)
	}
	if _, err := application.NewRecoveryTargetRef("../escape"); err == nil {
		t.Fatal("traversal-shaped recovery target accepted")
	}
	symlinkRoot := filepath.Join(root, "backup-link")
	if err := os.Symlink(backupRoot, symlinkRoot); err != nil {
		t.Fatal(err)
	}
	if unsafeRecovery, err := statesqlite.NewRecovery(statesqlite.RecoveryOptions{
		Repository: state.repository, BackupRoot: symlinkRoot,
		RestoreRoot: filepath.Join(root, "other-restores"), Now: state.clock.Now,
	}); err == nil {
		_ = unsafeRecovery.Close()
		t.Fatal("symlink recovery root accepted")
	}
	unsafeModeRoot := filepath.Join(root, "unsafe-mode-backups")
	if err := os.Mkdir(unsafeModeRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if unsafeRecovery, err := statesqlite.NewRecovery(statesqlite.RecoveryOptions{
		Repository: state.repository, BackupRoot: unsafeModeRoot,
		RestoreRoot: filepath.Join(root, "mode-restores"), Now: state.clock.Now,
	}); err == nil {
		_ = unsafeRecovery.Close()
		t.Fatal("unsafe existing recovery mode was silently repaired")
	}
	realRestoreRoot := filepath.Join(root, "real-restore-root")
	if err := os.Mkdir(realRestoreRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	restoreLink := filepath.Join(root, "restore-link")
	if err := os.Symlink(realRestoreRoot, restoreLink); err != nil {
		t.Fatal(err)
	}
	privateBackupRoot := filepath.Join(root, "private-overlap-backups")
	if err := os.Mkdir(privateBackupRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if unsafeRecovery, err := statesqlite.NewRecovery(statesqlite.RecoveryOptions{
		Repository: state.repository, BackupRoot: privateBackupRoot, RestoreRoot: restoreLink, Now: state.clock.Now,
	}); err == nil {
		_ = unsafeRecovery.Close()
		t.Fatal("symlink restore root accepted")
	}
	if unsafeRecovery, err := statesqlite.NewRecovery(statesqlite.RecoveryOptions{
		Repository: state.repository, BackupRoot: privateBackupRoot,
		RestoreRoot: filepath.Dir(state.databasePath), Now: state.clock.Now,
	}); err == nil {
		_ = unsafeRecovery.Close()
		t.Fatal("active SQLite namespace accepted as restore root")
	}
	v09AssertRecoveryFailpointCleanup(t, fixture)
}

func v09AssertRecoveryFailpointCleanup(t *testing.T, fixture v09Fixture) {
	t.Helper()
	for _, stage := range fixture.Scenario.Failpoints {
		t.Run(stage, func(t *testing.T) {
			ctx := context.Background()
			root := t.TempDir()
			state := v09CreateStateFixture(t, fixture, root)
			defer state.close(t)
			backupRoot := filepath.Join(root, "backups")
			restoreRoot := filepath.Join(root, "restores")
			var receipt application.BackupReceipt
			var target application.RecoveryTargetRef
			var targetPath string
			if strings.HasPrefix(stage, "after_restore_") {
				stable, err := statesqlite.NewRecovery(statesqlite.RecoveryOptions{
					Repository: state.repository, BackupRoot: backupRoot, RestoreRoot: restoreRoot, Now: state.clock.Now,
				})
				if err != nil {
					t.Fatal(err)
				}
				receipt, err = stable.CreateBackup(ctx)
				if closeErr := stable.Close(); err != nil || closeErr != nil {
					t.Fatalf("prepare backup: err=%v close=%v", err, closeErr)
				}
			}
			injected := errors.New("v09.injected_crash_boundary")
			recovery, err := statesqlite.NewRecovery(statesqlite.RecoveryOptions{
				Repository: state.repository, BackupRoot: backupRoot, RestoreRoot: restoreRoot, Now: state.clock.Now,
				Failpoint: func(current string) error {
					if current == stage {
						return injected
					}
					return nil
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			if strings.HasPrefix(stage, "after_restore_") {
				target, err = application.NewRecoveryTargetRef("recovery-target:v09-failpoint-" + strings.TrimPrefix(stage, "after_restore_"))
				if err != nil {
					t.Fatal(err)
				}
				targetPath, err = recovery.TargetPath(target)
				if err != nil {
					t.Fatal(err)
				}
				_, err = recovery.RestoreBackup(ctx, receipt.Ref, target)
				if !errors.Is(err, injected) {
					t.Fatalf("restore failpoint %s err=%v", stage, err)
				}
			} else {
				if _, err := recovery.CreateBackup(ctx); !errors.Is(err, injected) {
					t.Fatalf("backup failpoint %s err=%v", stage, err)
				}
			}
			if err := recovery.Close(); err != nil {
				t.Fatal(err)
			}
			for _, suffix := range []string{".next", ".partial", ".tmp", "-wal", "-shm", "-journal"} {
				files := append(v09FilesWithSuffix(t, backupRoot, suffix), v09FilesWithSuffix(t, restoreRoot, suffix)...)
				if len(files) != 0 {
					t.Errorf("failpoint %s left owned %s residue: %v", stage, suffix, files)
				}
			}
			backupPayloads := v09FilesWithSuffix(t, backupRoot, ".sqlite")
			backupManifests := v09FilesWithSuffix(t, backupRoot, ".manifest.json")
			if len(backupPayloads) != len(backupManifests) || len(backupPayloads) > 1 {
				t.Fatalf("failpoint %s left incomplete backup publication: payloads=%v manifests=%v", stage, backupPayloads, backupManifests)
			}
			if len(backupManifests) == 1 {
				content, err := os.ReadFile(backupManifests[0])
				if err != nil {
					t.Fatal(err)
				}
				var manifest v09BackupManifest
				decoder := json.NewDecoder(bytes.NewReader(content))
				decoder.DisallowUnknownFields()
				if err := decoder.Decode(&manifest); err != nil {
					t.Fatalf("failpoint %s left invalid manifest: %v", stage, err)
				}
				backupRef, err := application.NewBackupRef(manifest.BackupRef)
				if err != nil {
					t.Fatalf("failpoint %s left invalid backup ref: %v", stage, err)
				}
				stable, err := statesqlite.NewRecovery(statesqlite.RecoveryOptions{
					Repository: state.repository, BackupRoot: backupRoot, RestoreRoot: restoreRoot, Now: state.clock.Now,
				})
				if err != nil {
					t.Fatal(err)
				}
				if _, err := stable.VerifyBackup(ctx, backupRef); err != nil {
					_ = stable.Close()
					t.Fatalf("failpoint %s published unverifiable backup: %v", stage, err)
				}
				if err := stable.Close(); err != nil {
					t.Fatal(err)
				}
			}
			if strings.HasPrefix(stage, "after_restore_") {
				_, statErr := os.Stat(targetPath)
				if statErr == nil {
					restored := v06OpenSQLite(t, ctx, targetPath, state.clock)
					status, err := restored.Status(ctx)
					closeErr := restored.Close()
					if err != nil || closeErr != nil || status.Goals != 2 {
						t.Fatalf("failpoint %s published invalid restore: status=%+v err=%v close=%v", stage, status, err, closeErr)
					}
				} else if !errors.Is(statErr, os.ErrNotExist) {
					t.Fatal(statErr)
				}
			}
		})
	}
}

func v09AssertBackupExclusions(
	t *testing.T,
	backupRoot string,
	terminal application.GoalRecord,
	credentialMaterial []byte,
) {
	t.Helper()
	if len(terminal.Artifacts) == 0 {
		t.Fatal("fixture lacks artifact metadata")
	}
	artifactMaterial := []byte("V06 acreditado")
	signatures := [][]byte{
		credentialMaterial,
		[]byte(base64.StdEncoding.EncodeToString(credentialMaterial)),
		[]byte(base64.RawURLEncoding.EncodeToString(credentialMaterial)),
		[]byte(hex.EncodeToString(credentialMaterial)),
		[]byte(strings.ToUpper(hex.EncodeToString(credentialMaterial))),
		artifactMaterial,
	}
	err := filepath.WalkDir(backupRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, signature := range signatures {
			if bytes.Contains(content, signature) {
				return fmt.Errorf("excluded material present in %s", filepath.Base(path))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func v09AssertBackupManifest(
	t *testing.T,
	backupRoot string,
	receipt application.BackupReceipt,
	fixture v09Fixture,
) {
	t.Helper()
	payloadPath := v09FindOnlyFile(t, backupRoot, ".sqlite")
	manifestPath := v09FindOnlyFile(t, backupRoot, ".manifest.json")
	payload, err := os.ReadFile(payloadPath)
	if err != nil {
		t.Fatal(err)
	}
	manifestContent, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(bytes.NewReader(manifestContent))
	decoder.DisallowUnknownFields()
	var manifest v09BackupManifest
	if err := decoder.Decode(&manifest); err != nil {
		t.Fatalf("decode backup manifest: %v", err)
	}
	payloadDigest := evidenceBytesSHA256(payload)
	wantRef := fixture.Scenario.BackupRefPrefix + strings.TrimPrefix(payloadDigest, "sha256:")
	if manifest.SchemaVersion != 1 || manifest.BackupRef != receipt.Ref.String() ||
		manifest.PayloadSHA256 != payloadDigest || receipt.Ref.String() != wantRef ||
		manifest.LogicalSHA256 == "" || manifest.LogicalSHA256 == manifest.PayloadSHA256 ||
		manifest.MediaType != receipt.MediaType || manifest.Size != int64(len(payload)) || receipt.Size != manifest.Size ||
		manifest.SchemaRef != receipt.SchemaRef || !manifest.CreatedAt.Equal(receipt.CreatedAt) ||
		manifest.CreatedAt.Location() != time.UTC || receipt.CreatedAt.Location() != time.UTC ||
		manifest.CredentialsIncluded || manifest.ArtifactBlobsIncluded ||
		receipt.ManifestSHA256 != evidenceBytesSHA256(manifestContent) {
		t.Fatalf("backup manifest is not independently bound: manifest=%+v receipt=%+v", manifest, receipt)
	}
}

func v09AssertJSONImport(t *testing.T, repositoryRoot string, fixture v09Fixture) {
	t.Helper()
	root := t.TempDir()
	sourcePath := v07WriteSource(t, root, "")
	active := v07Load(t, sourcePath)
	v07 := evidenceDecodeStrictJSON[v07Fixture](t, filepath.Join(repositoryRoot, "acceptance/fixtures/v07_config.json"))
	manager, _ := v07OpenManager(t, sourcePath, active, nil, nil, v07)
	mappings := jsonimport.CanonicalMappings()
	v09AssertCanonicalJSONMappings(t, mappings)
	importer, err := jsonimport.New(jsonimport.Options{Manager: manager})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	compatible := []byte(`{"schema_version":"orquesta_config.v0","server":{"addr":"127.0.0.1:19090"}}`)
	before := v07DirectorySnapshot(t, root)
	plan, err := importer.Preview(ctx, compatible)
	if err != nil || !plan.Ready || plan.SourceSHA256 == "" || plan.PlanSHA256 == "" ||
		len(plan.Entries) != 2 || len(plan.UnresolvedPaths) != 0 || len(plan.SecretRequiredPaths) != 0 {
		t.Fatalf("compatible preview invalid: %+v err=%v", plan, err)
	}
	if after := v07DirectorySnapshot(t, root); !reflect.DeepEqual(before, after) {
		t.Fatalf("dry-run wrote files: before=%v after=%v", before, after)
	}
	view, err := manager.View(ctx)
	if err != nil {
		t.Fatal(err)
	}
	applied, err := importer.Apply(ctx, jsonimport.ApplyRequest{
		Source: compatible, ExpectedPlanSHA256: plan.PlanSHA256,
		ExpectedRevision: view.SourceRevision, ActorRef: "actor:v09-migrator",
		RequestRef: "request:v09-migrate-compatible", Confirm: true,
	})
	if err != nil || applied.Receipt.AfterRevision == applied.Receipt.BeforeRevision {
		t.Fatalf("compatible import did not commit through Manager: %+v err=%v", applied, err)
	}
	replayed, err := importer.Apply(ctx, jsonimport.ApplyRequest{
		Source: compatible, ExpectedPlanSHA256: plan.PlanSHA256,
		ExpectedRevision: view.SourceRevision, ActorRef: "actor:v09-migrator",
		RequestRef: "request:v09-migrate-compatible", Confirm: true,
	})
	if err != nil || !replayed.Replayed || !reflect.DeepEqual(replayed.Receipt, applied.Receipt) {
		t.Fatalf("Manager-backed import replay mutated again: first=%+v replay=%+v err=%v", applied, replayed, err)
	}
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	explicit, err := config.ParseExplicit(content)
	if err != nil || explicit[config.KeyServerListen] != "127.0.0.1:19090" {
		t.Fatalf("imported TOML differs: values=%v err=%v", explicit, err)
	}
	actualResolved := v07Load(t, sourcePath)
	expectedResolved, err := config.Resolve(config.ResolveOptions{
		TOML: []byte("server.listen = \"127.0.0.1:19090\"\n"), SourcePath: sourcePath,
	})
	if err != nil {
		t.Fatal(err)
	}
	actualEffective, actualErr := actualResolved.EffectiveJSON()
	expectedEffective, expectedErr := expectedResolved.EffectiveJSON()
	if actualErr != nil || expectedErr != nil || actualResolved.Hash() != expectedResolved.Hash() ||
		!bytes.Equal(actualEffective, expectedEffective) {
		t.Fatalf("imported resolved/effective config differs: actual=%s expected=%s err=%v/%v", actualResolved.Hash(), expectedResolved.Hash(), actualErr, expectedErr)
	}

	realShape, err := os.ReadFile(filepath.Join(repositoryRoot, filepath.FromSlash(fixture.Scenario.RealLegacyFixture)))
	if err != nil || evidenceBytesSHA256(realShape) != fixture.Scenario.RealLegacyFixtureSHA256 {
		t.Fatalf("load pinned real-shape fixture: digest=%s err=%v", evidenceBytesSHA256(realShape), err)
	}
	before = v07DirectorySnapshot(t, root)
	realPlan, err := importer.Preview(ctx, realShape)
	wantUnresolved := []string{
		"autoprogramming.checkpoint_only_high_consumption_tokens",
		"control_plane.remote_access_opt_in",
		"operator_director_mailbox.enabled",
		"runtime_models.allowed_models",
		"runtime_models.enabled",
		"server.state_dir",
	}
	resolved := 0
	for _, entry := range realPlan.Entries {
		if entry.Disposition == jsonimport.DispositionMapped || entry.Disposition == jsonimport.DispositionSchema {
			resolved++
		}
	}
	if err != nil || realPlan.Ready || realPlan.SourceSHA256 != fixture.Scenario.RealLegacyFixtureSHA256 ||
		len(realPlan.Entries) != 8 || resolved != 2 || len(realPlan.SecretRequiredPaths) != 0 ||
		!reflect.DeepEqual(realPlan.UnresolvedPaths, wantUnresolved) ||
		!reflect.DeepEqual(v09PlanPaths(realPlan), fixture.Scenario.RealLegacyLeafPaths) {
		t.Fatalf("real legacy shape was not exhaustively blocked: %+v err=%v", realPlan, err)
	}
	if _, err := importer.Apply(ctx, jsonimport.ApplyRequest{
		Source: realShape, ExpectedPlanSHA256: realPlan.PlanSHA256,
		ExpectedRevision: applied.Receipt.AfterRevision, ActorRef: "actor:v09-migrator",
		RequestRef: "request:v09-migrate-not-ready", Confirm: true,
	}); !jsonimport.HasErrorCode(err, jsonimport.ErrorPlanNotReady) {
		t.Fatalf("not-ready legacy plan error=%v", err)
	}
	if after := v07DirectorySnapshot(t, root); !reflect.DeepEqual(before, after) {
		t.Fatalf("blocked legacy apply wrote files: before=%v after=%v", before, after)
	}
	secret := "v09-secret-must-never-project"
	secretPlan, err := importer.Preview(ctx, []byte(fmt.Sprintf(
		`{"schema_version":"orquesta_config.v0","control_plane":{"token":%q}}`, secret,
	)))
	if err != nil || secretPlan.Ready || !reflect.DeepEqual(secretPlan.SecretRequiredPaths, []string{"control_plane.token"}) ||
		strings.Contains(fmt.Sprintf("%+v", secretPlan), secret) {
		t.Fatalf("secret disposition leaked or became ready: plan=%+v err=%v", secretPlan, err)
	}
	for _, invalid := range [][]byte{
		[]byte(`{"schema_version":"orquesta_config.v0","server":{"addr":"a","addr":"b"}}`),
		[]byte(`{"schema_version":"orquesta_config.v0"} trailing`),
		[]byte(`{"schema_version":"unknown.v0","server":{"addr":"a"}}`),
		[]byte(`{"schema_version":"orquesta_config.v0","unknown":true}`),
	} {
		if _, err := importer.Preview(ctx, invalid); err == nil {
			t.Errorf("strict legacy JSON accepted %q", invalid)
		}
	}
	v09AssertNoLegacyJSONRuntimeWiring(t, repositoryRoot)
}

func v09AssertNoLegacyJSONRuntimeWiring(t *testing.T, repositoryRoot string) {
	t.Helper()
	for _, relativeRoot := range []string{"internal/bootstrap", "cmd/orquesta"} {
		root := filepath.Join(repositoryRoot, filepath.FromSlash(relativeRoot))
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
				return nil
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			for _, forbidden := range [][]byte{
				[]byte("orquesta.config.json"),
				[]byte("orquesta/internal/adapters/config/jsonimport"),
				[]byte("legacyjson"),
			} {
				if bytes.Contains(content, forbidden) {
					t.Errorf("runtime composition %s references one-shot legacy JSON authority %q", filepath.ToSlash(path), forbidden)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("inspect runtime composition %s: %v", relativeRoot, err)
		}
	}
}

func v09AssertCanonicalJSONMappings(t *testing.T, mappings []jsonimport.Mapping) {
	t.Helper()
	wantPaths := []string{
		"autoprogramming.checkpoint_only_high_consumption_tokens",
		"control_plane.remote_access_opt_in",
		"control_plane.token",
		"operator_director_mailbox.enabled",
		"runtime_models.allowed_models",
		"runtime_models.enabled",
		"schema_version",
		"server.addr",
		"server.state_dir",
	}
	wantKinds := map[string]jsonimport.SourceKind{
		"autoprogramming.checkpoint_only_high_consumption_tokens": jsonimport.SourceKindInteger,
		"control_plane.remote_access_opt_in":                      jsonimport.SourceKindBool,
		"control_plane.token":                                     jsonimport.SourceKindString,
		"operator_director_mailbox.enabled":                       jsonimport.SourceKindBool,
		"runtime_models.allowed_models":                           jsonimport.SourceKindStringArray,
		"runtime_models.enabled":                                  jsonimport.SourceKindBool,
		"schema_version":                                          jsonimport.SourceKindString,
		"server.addr":                                             jsonimport.SourceKindString,
		"server.state_dir":                                        jsonimport.SourceKindString,
	}
	gotPaths := make([]string, 0, len(mappings))
	seen := make(map[string]struct{}, len(mappings))
	for _, mapping := range mappings {
		if _, duplicate := seen[mapping.LegacyPath]; duplicate {
			t.Fatalf("duplicate canonical legacy mapping %q", mapping.LegacyPath)
		}
		seen[mapping.LegacyPath] = struct{}{}
		gotPaths = append(gotPaths, mapping.LegacyPath)
		if mapping.SourceKind != wantKinds[mapping.LegacyPath] {
			t.Fatalf("legacy mapping source kind differs: %+v want=%q", mapping, wantKinds[mapping.LegacyPath])
		}
		switch mapping.Disposition {
		case jsonimport.DispositionMapped:
			if mapping.TargetKey == "" || mapping.Transform == "" {
				t.Fatalf("mapped legacy path lacks target/transform: %+v", mapping)
			}
		case jsonimport.DispositionSchema, jsonimport.DispositionDeferred, jsonimport.DispositionSecretRequired:
			if strings.TrimSpace(mapping.Reason) == "" {
				t.Fatalf("non-mapped legacy disposition lacks reason: %+v", mapping)
			}
		default:
			t.Fatalf("unsupported legacy disposition: %+v", mapping)
		}
	}
	if !reflect.DeepEqual(gotPaths, wantPaths) {
		t.Fatalf("canonical orquesta_config.v0 table paths=%v want=%v", gotPaths, wantPaths)
	}
}

func v09PlanPaths(plan jsonimport.Plan) []string {
	result := make([]string, 0, len(plan.Entries))
	for _, entry := range plan.Entries {
		result = append(result, entry.LegacyPath)
	}
	sort.Strings(result)
	return result
}

func v09FindOnlyFile(t *testing.T, root, suffix string) string {
	t.Helper()
	files := v09FilesWithSuffix(t, root, suffix)
	if len(files) != 1 {
		t.Fatalf("files with suffix %q under %s = %v", suffix, root, files)
	}
	return files[0]
}

func v09FilesWithSuffix(t *testing.T, root, suffix string) []string {
	t.Helper()
	var result []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if path != root && strings.HasSuffix(path, suffix) {
			result = append(result, path)
		}
		return nil
	})
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	sort.Strings(result)
	return result
}

type v09SQLiteTableDump struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Rows    [][]any  `json:"rows"`
}

func v09SQLiteLogicalDump(t *testing.T, filename string) []byte {
	t.Helper()
	dsn := (&url.URL{Scheme: "file", Path: filename, RawQuery: "mode=ro&_pragma=query_only(1)"}).String()
	database, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	tableRows, err := database.Query(`SELECT name FROM sqlite_schema WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for tableRows.Next() {
		var name string
		if err := tableRows.Scan(&name); err != nil {
			_ = tableRows.Close()
			t.Fatal(err)
		}
		names = append(names, name)
	}
	if err := tableRows.Err(); err != nil {
		_ = tableRows.Close()
		t.Fatal(err)
	}
	if err := tableRows.Close(); err != nil {
		t.Fatal(err)
	}
	dump := make([]v09SQLiteTableDump, 0, len(names))
	for _, name := range names {
		quotedTable := v09SQLiteQuoteIdentifier(name)
		probe, err := database.Query("SELECT * FROM " + quotedTable + " LIMIT 0")
		if err != nil {
			t.Fatal(err)
		}
		columns, err := probe.Columns()
		closeErr := probe.Close()
		if err != nil || closeErr != nil || len(columns) == 0 {
			t.Fatalf("inspect SQLite table %s: columns=%v err=%v close=%v", name, columns, err, closeErr)
		}
		order := make([]string, len(columns))
		for index, column := range columns {
			order[index] = v09SQLiteQuoteIdentifier(column)
		}
		rows, err := database.Query("SELECT * FROM " + quotedTable + " ORDER BY " + strings.Join(order, ", "))
		if err != nil {
			t.Fatal(err)
		}
		table := v09SQLiteTableDump{Name: name, Columns: columns, Rows: make([][]any, 0)}
		for rows.Next() {
			values := make([]any, len(columns))
			destinations := make([]any, len(columns))
			for index := range values {
				destinations[index] = &values[index]
			}
			if err := rows.Scan(destinations...); err != nil {
				_ = rows.Close()
				t.Fatal(err)
			}
			table.Rows = append(table.Rows, values)
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			t.Fatal(err)
		}
		if err := rows.Close(); err != nil {
			t.Fatal(err)
		}
		dump = append(dump, table)
	}
	content, err := json.Marshal(dump)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func v09SQLiteQuoteIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

func TestV09RecoveryResidueScannerIncludesDirectories(t *testing.T) {
	root := t.TempDir()
	orphan := filepath.Join(root, "orphan.next")
	if err := os.Mkdir(orphan, 0o700); err != nil {
		t.Fatal(err)
	}
	if got := v09FilesWithSuffix(t, root, ".next"); !reflect.DeepEqual(got, []string{orphan}) {
		t.Fatalf("owned transient directory escaped residue scan: %v", got)
	}
}

func TestV09JSONImportUsesManagerWithoutOwningStoreLifecycle(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v09Fixture](t, filepath.Join(repositoryRoot, filepath.FromSlash(v09FixturePath)))
	v09AssertJSONImport(t, repositoryRoot, fixture)
}

func TestV09FixtureIsStrictJSON(t *testing.T) {
	filename := filepath.Join(evidenceRepositoryRoot(t), filepath.FromSlash(v09FixturePath))
	_ = evidenceDecodeStrictJSON[v09Fixture](t, filename)
}
