package acceptance_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	effectivefile "orquesta/internal/adapters/config/effectivefile"
	tomlstore "orquesta/internal/adapters/config/toml"
	"orquesta/internal/config"
)

const v07FixturePath = "acceptance/fixtures/v07_config.json"
const v07TrustedBaseGitCommitOID = "5ff9234d7ef1c8650c1f21ad21a288246c5d027c"

type v07Fixture struct {
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
	DeferredCapabilities           []v07DeferredCapability `json:"deferred_capabilities"`
	Scenario                       v07Scenario             `json:"scenario"`
	Assertions                     []string                `json:"assertions"`
}

type v07DeferredCapability struct {
	ID                 string `json:"id"`
	Owner              string `json:"owner"`
	AcceptanceContract string `json:"acceptance_contract"`
}

type v07Scenario struct {
	BaseTime          string   `json:"base_time"`
	ConcurrentWriters int      `json:"concurrent_writers"`
	MaxSourceBytes    int64    `json:"max_source_bytes"`
	MaxReceiptBytes   int64    `json:"max_receipt_bytes"`
	SourceMode        string   `json:"source_mode"`
	EffectiveMode     string   `json:"effective_mode"`
	Failpoints        []string `json:"failpoints"`
}

type v07PersistedReceiptEnvelope struct {
	SchemaVersion int                   `json:"schema_version"`
	DocumentType  string                `json:"document_type"`
	Receipt       v07PersistedReceiptV1 `json:"receipt"`
}

type v07PersistedReceiptV1 struct {
	ReceiptRef         string          `json:"receipt_ref"`
	ActorRef           string          `json:"actor_ref"`
	RequestRef         string          `json:"request_ref"`
	Fingerprint        string          `json:"fingerprint"`
	BeforeRevision     config.Revision `json:"before_revision"`
	AfterRevision      config.Revision `json:"after_revision"`
	ChangedKeys        []config.Key    `json:"changed_keys"`
	PendingRestartKeys []config.Key    `json:"pending_restart_keys"`
	ChangedAt          time.Time       `json:"changed_at"`
}

type v07ReadOnlyDocumentStore struct {
	document config.StoredDocument
	reads    atomic.Int64
}

func (store *v07ReadOnlyDocumentStore) Read(context.Context) (config.StoredDocument, error) {
	store.reads.Add(1)
	document := store.document
	document.Content = append([]byte(nil), document.Content...)
	return document, nil
}

func (*v07ReadOnlyDocumentStore) Commit(context.Context, config.CommitRequest) (config.CommitResult, error) {
	return config.CommitResult{}, errors.New("v07_read_only_store")
}

func TestAcceptanceV07Config(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v07Fixture](t, filepath.Join(repositoryRoot, filepath.FromSlash(v07FixturePath)))
	v07AssertFixtureHeader(t, repositoryRoot, fixture)

	t.Run("port_shape_precedence_and_generated_surfaces", func(t *testing.T) {
		v07AssertPortAndGeneratedShape(t, repositoryRoot, fixture)
	})
	t.Run("mutable_toml_effective_pending_restart_and_credentials", func(t *testing.T) {
		v07AssertMutationAndRestart(t, fixture)
	})
	t.Run("concurrent_CAS_and_idempotent_receipt", func(t *testing.T) {
		v07AssertCASAndReplay(t, fixture)
	})
	t.Run("journal_recovers_each_crash_boundary", func(t *testing.T) {
		v07AssertCrashRecovery(t, fixture)
	})
	t.Run("doctor_child_environment_and_anti_scope", func(t *testing.T) {
		v07AssertDoctorChildEnvironmentAndScope(t, repositoryRoot, fixture)
	})
}

// TestV07CrashProcessHelper is entered only by v07AssertCrashRecovery. The
// process exits from the store failpoint, so transaction cleanup defers cannot
// turn a crash boundary into an ordinary returned error.
func TestV07CrashProcessHelper(t *testing.T) {
	if os.Getenv("ORQUESTA_V07_CRASH_HELPER") != "1" {
		return
	}
	path := os.Getenv("ORQUESTA_V07_CRASH_PATH")
	stage := os.Getenv("ORQUESTA_V07_CRASH_STAGE")
	maxSource, err := strconv.ParseInt(os.Getenv("ORQUESTA_V07_MAX_SOURCE"), 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	maxReceipt, err := strconv.ParseInt(os.Getenv("ORQUESTA_V07_MAX_RECEIPT"), 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	baseTime, err := time.Parse(time.RFC3339Nano, os.Getenv("ORQUESTA_V07_BASE_TIME"))
	if err != nil {
		t.Fatal(err)
	}
	active := v07Load(t, path)
	store, err := tomlstore.Open(tomlstore.Options{
		Path: path, MaxSourceBytes: maxSource, MaxReceiptBytes: maxReceipt,
		Failpoint: func(current string) error {
			if current == stage {
				os.Exit(86)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	manager, err := config.NewManager(config.ManagerOptions{
		Store: store, Active: active, Now: func() time.Time { return baseTime },
	})
	if err != nil {
		t.Fatal(err)
	}
	view, err := manager.View(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	_, err = manager.Update(context.Background(), config.UpdateRequest{
		ActorRef: "actor:v07", RequestRef: "request:v07-crash-" + stage,
		ExpectedRevision: view.SourceRevision, Confirm: true,
		Changes: []config.Change{{Key: config.KeySchedulerPollInterval, Value: "850ms"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Fatal("configured crash failpoint was not reached")
}

func TestAcceptanceV07ConfigReceipt(t *testing.T) {
	evidenceAssertReceiptV3(t, evidenceRepositoryRoot(t), evidenceReceiptV3Expectation{
		Contract: "AC-V07-CONFIG", FixturePath: v07FixturePath,
		ReceiptPath:       "product/evidence/v07_config.json",
		ExecutedNotBefore: "2026-07-14T00:00:00Z", TrustedBaseGitCommitOID: v07TrustedBaseGitCommitOID,
	})
}

func TestV07CandidateSubjectsCoverCommittedDelta(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v07Fixture](t, filepath.Join(repositoryRoot, filepath.FromSlash(v07FixturePath)))
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
		t.Fatalf("V07 candidate subjects differ from sealed product delta:\nchanged=%v\nfixture=%v", changed, fixture.CandidateSubjects)
	}
}

func v07AssertFixtureHeader(t *testing.T, repositoryRoot string, fixture v07Fixture) {
	t.Helper()
	wantCapabilities := []string{
		"OPS-01", "OPS-02", "OPS-04", "OPS-05", "OPS-06",
		"OPS-26", "OPS-27", "OPS-28", "OPS-29", "OPS-30",
	}
	wantDeferred := []v07DeferredCapability{{ID: "OPS-07", Owner: "web_admin", AcceptanceContract: "AC-V24-WEB-ADMIN"}}
	wantFailpoints := []string{
		"after_replacement_sync", "after_intent_sync", "after_source_rename",
		"after_source_directory_sync", "after_receipt_rename", "after_receipt_directory_sync",
	}
	if fixture.SchemaVersion != 1 || fixture.ReceiptSchemaVersion != 3 || fixture.ContractID != "AC-V07-CONFIG" ||
		fixture.TrustedBaseGitCommitOID != v07TrustedBaseGitCommitOID ||
		fixture.ProductDeltaBaseGitCommitOID != v07TrustedBaseGitCommitOID ||
		fixture.ProductDeltaSealedGitCommitOID != v07TrustedBaseGitCommitOID ||
		fixture.OutputPath != "product/evidence/v07_config.output.txt" ||
		fixture.ReceiptPath != "product/evidence/v07_config.json" ||
		!reflect.DeepEqual(fixture.OwnedCapabilityIDs, wantCapabilities) ||
		!reflect.DeepEqual(fixture.DeferredCapabilities, wantDeferred) || len(fixture.Assertions) != 13 ||
		fixture.Scenario.ConcurrentWriters < 2 || fixture.Scenario.MaxSourceBytes != 1<<20 ||
		fixture.Scenario.MaxReceiptBytes != 1<<16 || fixture.Scenario.SourceMode != "0600" ||
		fixture.Scenario.EffectiveMode != "0400" || !reflect.DeepEqual(fixture.Scenario.Failpoints, wantFailpoints) {
		t.Fatalf("invalid V07 fixture header: %+v", fixture)
	}
	if _, err := time.Parse(time.RFC3339Nano, fixture.Scenario.BaseTime); err != nil {
		t.Fatalf("invalid base time: %v", err)
	}
	if fixture.Command != "sh -c '"+v07ValidationShellBody()+"'" ||
		!reflect.DeepEqual(fixture.ExecutionArgv, []string{"sh", "-c", v07ValidationShellBody()}) {
		t.Fatalf("invalid V07 command/argv: %q %#v", fixture.Command, fixture.ExecutionArgv)
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

func v07ValidationShellBody() string {
	return "go test -mod=vendor -count=1 . ./acceptance -run \"^(TestProductRoadmapIsExhaustiveAndCausal|TestProductRoadmapV07ScopeAndExecutableContract|TestRebuildArchitecture|TestTraceabilityRebuildBugLessons|TestAcceptanceV07Config|TestV07CandidateSubjectsCoverCommittedDelta)$\"" +
		" && go test -mod=vendor -count=1 ./internal/config ./internal/adapters/config/effectivefile ./internal/adapters/config/toml ./internal/bootstrap ./cmd/orquesta"
}

func v07AssertPortAndGeneratedShape(t *testing.T, repositoryRoot string, fixture v07Fixture) {
	t.Helper()
	port := reflect.TypeOf((*config.DocumentStore)(nil)).Elem()
	if port.NumMethod() != 2 {
		t.Fatalf("DocumentStore methods = %d, want Read and Commit only", port.NumMethod())
	}
	for _, name := range []string{"Read", "Commit"} {
		if _, found := port.MethodByName(name); !found {
			t.Fatalf("DocumentStore lacks %s", name)
		}
	}
	var _ config.DocumentStore = (*tomlstore.Store)(nil)
	var _ config.DocumentStore = (*v07ReadOnlyDocumentStore)(nil)

	defaults, err := config.Resolve(config.ResolveOptions{})
	if err != nil || defaults.APILocale() != "es" {
		t.Fatalf("registry default resolution = %q err=%v", defaults.APILocale(), err)
	}
	fileSnapshot, err := config.Resolve(config.ResolveOptions{TOML: []byte("[api]\nlocale = \"en\"\n")})
	if err != nil || fileSnapshot.APILocale() != "en" {
		t.Fatalf("TOML precedence = %q err=%v", fileSnapshot.APILocale(), err)
	}
	environmentSnapshot, err := config.Resolve(config.ResolveOptions{
		TOML: []byte("[api]\nlocale = \"en\"\n"), Environment: map[string]string{"ORQUESTA_API_LOCALE": "es"},
	})
	if err != nil || environmentSnapshot.APILocale() != "es" {
		t.Fatalf("environment precedence = %q err=%v", environmentSnapshot.APILocale(), err)
	}
	for _, check := range []struct {
		snapshot config.Snapshot
		want     config.Source
	}{{defaults, config.SourceDefault}, {fileSnapshot, config.SourceFile}, {environmentSnapshot, config.SourceEnv}} {
		metadata, found := check.snapshot.Metadata(config.KeyAPILocale)
		if !found || metadata.Source != check.want {
			t.Fatalf("api.locale source = %q/%v, want %q", metadata.Source, found, check.want)
		}
	}
	if _, err := config.Resolve(config.ResolveOptions{Environment: map[string]string{"ORQUESTA_UNDECLARED_V07": "value"}}); err == nil {
		t.Fatal("undeclared bootstrap environment name was accepted")
	}

	root := t.TempDir()
	path := v07WriteSource(t, root, "[api]\nlocale = \"es\"\n")
	active := v07Load(t, path)
	concrete, err := tomlstore.Open(tomlstore.Options{
		Path: path, MaxSourceBytes: fixture.Scenario.MaxSourceBytes, MaxReceiptBytes: fixture.Scenario.MaxReceiptBytes,
	})
	if err != nil {
		t.Fatal(err)
	}
	document, err := concrete.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	memory := &v07ReadOnlyDocumentStore{document: document}
	baseTime, _ := time.Parse(time.RFC3339Nano, fixture.Scenario.BaseTime)
	neutralManager, err := config.NewManager(config.ManagerOptions{
		Store: memory, Active: active, Now: func() time.Time { return baseTime },
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := neutralManager.View(context.Background()); err != nil || memory.reads.Load() != 1 {
		t.Fatalf("Manager did not consume neutral DocumentStore: reads=%d err=%v", memory.reads.Load(), err)
	}
	first, _ := v07OpenManager(t, path, active, map[string]string{"ORQUESTA_RUNTIME_CODEX_MODEL": "first-model"}, nil, fixture)
	second, _ := v07OpenManager(t, path, active, map[string]string{"ORQUESTA_RUNTIME_CODEX_MODEL": "second-model"}, nil, fixture)
	firstView, err := first.View(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	secondView, err := second.View(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if firstView.DesiredHash == secondView.DesiredHash || firstView.ActiveHash != active.Hash() || secondView.ActiveHash != active.Hash() {
		t.Fatalf("declared environment precedence or active snapshot is not deterministic: %+v %+v", firstView, secondView)
	}

	generatedRoot := t.TempDir()
	generated := map[string]string{
		filepath.Join(generatedRoot, "keys_generated.go"):     "internal/config/keys_generated.go",
		filepath.Join(generatedRoot, "orquesta.schema.json"):  "config/generated/orquesta.schema.json",
		filepath.Join(generatedRoot, "ui.json"):               "config/generated/ui.json",
		filepath.Join(generatedRoot, "orquesta.toml.example"): "config/orquesta.toml.example",
		filepath.Join(generatedRoot, "reference.md"):          "config/generated/reference.md",
	}
	arguments := []string{
		"run", "./internal/config/cmd/configgen", "-registry", "config/registry.json",
		"-go-output", filepath.Join(generatedRoot, "keys_generated.go"),
		"-schema-output", filepath.Join(generatedRoot, "orquesta.schema.json"),
		"-ui-output", filepath.Join(generatedRoot, "ui.json"),
		"-example-output", filepath.Join(generatedRoot, "orquesta.toml.example"),
		"-doc-output", filepath.Join(generatedRoot, "reference.md"),
	}
	for run := 0; run < 2; run++ {
		command := exec.Command("go", arguments...)
		command.Dir = repositoryRoot
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("generate config surfaces: %v\n%s", err, output)
		}
		for generatedPath, trackedPath := range generated {
			got, readErr := os.ReadFile(generatedPath)
			if readErr != nil {
				t.Fatal(readErr)
			}
			want, readErr := os.ReadFile(filepath.Join(repositoryRoot, trackedPath))
			if readErr != nil || !bytes.Equal(got, want) {
				t.Fatalf("generated surface drift for %s: err=%v", trackedPath, readErr)
			}
		}
	}
}

func v07AssertMutationAndRestart(t *testing.T, fixture v07Fixture) {
	t.Helper()
	ctx := context.Background()
	root := t.TempDir()
	effectivePath := filepath.Join(root, "effective.json")
	path := v07WriteSource(t, root, "[config]\neffective_path = "+fmt.Sprintf("%q", effectivePath)+"\n")
	active := v07Load(t, path)
	manager, _ := v07OpenManager(t, path, active, nil, nil, fixture)
	view, err := manager.View(ctx)
	if err != nil || view.SourceRevision == "" || view.PendingRestart {
		t.Fatalf("initial view = %+v err=%v", view, err)
	}
	request := config.UpdateRequest{
		ActorRef: "actor:v07", RequestRef: "request:v07-mutation", ExpectedRevision: view.SourceRevision, Confirm: true,
		Changes: []config.Change{
			{Key: config.KeySchedulerPollInterval, Value: "750ms"},
			{Key: config.KeyRuntimeCodexCredentialRef, Value: config.CredentialRef("credential:v07-next")},
		},
	}
	result, err := manager.Update(ctx, request)
	if err != nil || result.Replayed || !result.View.PendingRestart || result.View.ActiveHash != active.Hash() ||
		result.Receipt.ActorRef != request.ActorRef || result.Receipt.RequestRef != request.RequestRef ||
		result.Receipt.BeforeRevision != view.SourceRevision || result.Receipt.AfterRevision != result.View.SourceRevision {
		t.Fatalf("update result = %+v err=%v", result, err)
	}
	v07AssertKeys(t, result.View.PendingRestartKeys, config.KeyRuntimeCodexCredentialRef, config.KeySchedulerPollInterval)
	receiptJSON, _ := json.Marshal(result.Receipt)
	if bytes.Contains(receiptJSON, []byte("credential:v07-next")) {
		t.Fatalf("audit receipt leaked credential ref value: %s", receiptJSON)
	}
	if _, err := manager.Update(ctx, config.UpdateRequest{
		ActorRef: "actor:v07", RequestRef: "request:v07-secret", ExpectedRevision: result.View.SourceRevision, Confirm: true,
		Changes: []config.Change{{Key: config.KeyRuntimeCodexCredentialRef, Value: "sk-live-secret-material"}},
	}); err == nil {
		t.Fatal("raw credential material was accepted as credential_ref")
	}

	desired := v07Load(t, path)
	effectiveJSON, err := desired.EffectiveJSON()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := config.ParseExplicit(effectiveJSON); err == nil {
		t.Fatal("effective JSON was accepted as human-editable TOML input")
	}
	activeEffective, err := active.EffectiveJSON()
	if err != nil {
		t.Fatal(err)
	}
	if err := effectivefile.Write(ctx, effectivefile.Options{
		Path: active.ConfigEffectivePath(), Content: activeEffective,
		MaxExistingBytes: active.ConfigEffectiveMaxExistingBytes(),
	}); err != nil {
		t.Fatal(err)
	}
	failpointReached := false
	err = effectivefile.Write(ctx, effectivefile.Options{
		Path: desired.ConfigEffectivePath(), Content: effectiveJSON,
		MaxExistingBytes: desired.ConfigEffectiveMaxExistingBytes(),
		Failpoint: func(stage string) error {
			if stage == "after_temp_sync" {
				failpointReached = true
				return errors.New("simulated effective crash")
			}
			return nil
		},
	})
	if err == nil || !failpointReached {
		t.Fatalf("effective pre-rename crash was not injected: %v", err)
	}
	unchanged, err := os.ReadFile(effectivePath)
	if err != nil || !bytes.Equal(unchanged, activeEffective) {
		t.Fatalf("effective pre-rename crash changed active output: %v", err)
	}
	if err := effectivefile.Write(ctx, effectivefile.Options{
		Path: desired.ConfigEffectivePath(), Content: effectiveJSON,
		MaxExistingBytes: desired.ConfigEffectiveMaxExistingBytes(),
	}); err != nil {
		t.Fatal(err)
	}
	effective, err := os.ReadFile(effectivePath)
	if err != nil || bytes.Contains(effective, []byte("credential:v07-next")) ||
		!bytes.Contains(effective, []byte("[REDACTED]")) {
		t.Fatalf("effective config is not redacted: %s err=%v", effective, err)
	}
	info, err := os.Stat(effectivePath)
	if err != nil || info.Mode().Perm() != 0o400 {
		t.Fatalf("effective mode = %v err=%v, want 0400", info.Mode().Perm(), err)
	}
	restarted, _ := v07OpenManager(t, path, desired, nil, nil, fixture)
	restartedView, err := restarted.View(ctx)
	if err != nil || restartedView.PendingRestart || restartedView.ActiveHash != restartedView.DesiredHash {
		t.Fatalf("restart did not activate desired snapshot: %+v err=%v", restartedView, err)
	}
	v07AssertHumanTOML(t, root, path, fixture.Scenario.MaxReceiptBytes, result.Receipt)
}

func v07AssertCASAndReplay(t *testing.T, fixture v07Fixture) {
	t.Helper()
	ctx := context.Background()
	root := t.TempDir()
	path := v07WriteSource(t, root, "[scheduler]\npoll_interval = \"500ms\"\n")
	active := v07Load(t, path)
	reader, _ := v07OpenManager(t, path, active, nil, nil, fixture)
	view, err := reader.View(ctx)
	if err != nil {
		t.Fatal(err)
	}
	type outcome struct {
		request config.UpdateRequest
		result  config.UpdateResult
		manager *config.Manager
		err     error
	}
	start := make(chan struct{})
	outcomes := make(chan outcome, fixture.Scenario.ConcurrentWriters)
	var wait sync.WaitGroup
	var transactionStages atomic.Int64
	for index := 0; index < fixture.Scenario.ConcurrentWriters; index++ {
		manager, _ := v07OpenManager(t, path, active, nil, func(string) error {
			transactionStages.Add(1)
			return nil
		}, fixture)
		request := config.UpdateRequest{
			ActorRef: "actor:v07", RequestRef: fmt.Sprintf("request:v07-race-%d", index),
			ExpectedRevision: view.SourceRevision, Confirm: true,
			Changes: []config.Change{{Key: config.KeySchedulerPollInterval, Value: fmt.Sprintf("%dms", 600+index)}},
		}
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			result, err := manager.Update(ctx, request)
			outcomes <- outcome{request: request, result: result, manager: manager, err: err}
		}()
	}
	close(start)
	wait.Wait()
	close(outcomes)
	var winner outcome
	successes := 0
	for candidate := range outcomes {
		if candidate.err == nil {
			successes++
			winner = candidate
			continue
		}
		if !config.IsDocumentStoreError(candidate.err, config.DocumentStoreRevisionConflict) {
			t.Fatalf("CAS loser error = %T %v, want revision conflict", candidate.err, candidate.err)
		}
	}
	if successes != 1 {
		t.Fatalf("CAS successes = %d, want exactly one", successes)
	}
	beforeReplay := v07DirectorySnapshot(t, root)
	transactionStages.Store(0)
	replayed, err := winner.manager.Update(ctx, winner.request)
	if err != nil || !replayed.Replayed || !reflect.DeepEqual(replayed.Receipt, winner.result.Receipt) {
		t.Fatalf("idempotent replay = %+v err=%v, want original receipt", replayed, err)
	}
	if transactionStages.Load() != 0 {
		t.Fatalf("idempotent replay entered %d write stages", transactionStages.Load())
	}
	if afterReplay := v07DirectorySnapshot(t, root); !reflect.DeepEqual(afterReplay, beforeReplay) {
		t.Fatal("idempotent replay rewrote source, journal, or receipt")
	}
	conflicting := winner.request
	conflicting.Changes = []config.Change{{Key: config.KeySchedulerPollInterval, Value: "9s"}}
	if _, err := winner.manager.Update(ctx, conflicting); err == nil {
		t.Fatal("same request_ref with a different fingerprint replayed")
	}
}

func v07AssertCrashRecovery(t *testing.T, fixture v07Fixture) {
	t.Helper()
	for _, stage := range fixture.Scenario.Failpoints {
		t.Run(stage, func(t *testing.T) {
			ctx := context.Background()
			root := t.TempDir()
			path := v07WriteSource(t, root, "[scheduler]\npoll_interval = \"500ms\"\n")
			active := v07Load(t, path)
			reader, _ := v07OpenManager(t, path, active, nil, nil, fixture)
			view, err := reader.View(ctx)
			if err != nil {
				t.Fatal(err)
			}
			request := config.UpdateRequest{
				ActorRef: "actor:v07", RequestRef: "request:v07-crash-" + stage,
				ExpectedRevision: view.SourceRevision, Confirm: true,
				Changes: []config.Change{{Key: config.KeySchedulerPollInterval, Value: "850ms"}},
			}
			command := exec.Command(os.Args[0], "-test.run=^TestV07CrashProcessHelper$")
			command.Env = append(os.Environ(),
				"ORQUESTA_V07_CRASH_HELPER=1",
				"ORQUESTA_V07_CRASH_PATH="+path,
				"ORQUESTA_V07_CRASH_STAGE="+stage,
				"ORQUESTA_V07_MAX_SOURCE="+strconv.FormatInt(fixture.Scenario.MaxSourceBytes, 10),
				"ORQUESTA_V07_MAX_RECEIPT="+strconv.FormatInt(fixture.Scenario.MaxReceiptBytes, 10),
				"ORQUESTA_V07_BASE_TIME="+fixture.Scenario.BaseTime,
			)
			output, crashErr := command.CombinedOutput()
			var exitError *exec.ExitError
			if !errors.As(crashErr, &exitError) || exitError.ExitCode() != 86 {
				t.Fatalf("failpoint %s did not terminate helper at crash boundary: %v\n%s", stage, crashErr, output)
			}
			recovered, _ := v07OpenManager(t, path, active, nil, nil, fixture)
			replayed, err := recovered.Update(ctx, request)
			wantReplayed := stage != "after_replacement_sync"
			if err != nil || replayed.Replayed != wantReplayed || !replayed.View.PendingRestart {
				t.Fatalf("recovery/replay at %s = %+v err=%v", stage, replayed, err)
			}
			if snapshot := v07Load(t, path); snapshot.SchedulerPollInterval() != 850*time.Millisecond {
				t.Fatalf("recovered TOML at %s has %v", stage, snapshot.SchedulerPollInterval())
			}
			v07AssertHumanTOML(t, root, path, fixture.Scenario.MaxReceiptBytes, replayed.Receipt)
		})
	}
}

func v07AssertDoctorChildEnvironmentAndScope(t *testing.T, repositoryRoot string, fixture v07Fixture) {
	t.Helper()
	root := t.TempDir()
	path := v07WriteSource(t, root, "[server]\nlisten = \"127.0.0.1:8080\"\n")
	active := v07Load(t, path)
	manager, _ := v07OpenManager(t, path, active, nil, nil, fixture)
	before, err := manager.View(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	report, err := manager.Doctor(context.Background(), config.DoctorRequest{Proposals: []config.DoctorProposal{
		{Key: "server.bind", TargetKey: config.KeyServerListen, SemanticRef: "orquesta.config.server.listen"},
		{Key: "server.address", TargetKey: config.KeyServerListen, SemanticRef: "network.listen", Alias: "server.listen", RemoveAfterRevision: "2027-01-01"},
		{Key: "telemetry.sample_interval", SemanticRef: "telemetry.sample_interval", GoName: "TelemetrySampleInterval", EnvAlias: "ORQUESTA_TELEMETRY_SAMPLE_INTERVAL", Type: "duration", Scope: "telemetry"},
	}})
	if err != nil || len(report.Accepted) != 3 || len(report.Conflicts) != 0 ||
		report.Accepted[0].Mode != config.DoctorReuse || report.Accepted[1].Mode != config.DoctorReplace ||
		report.Accepted[2].Mode != config.DoctorNew {
		t.Fatalf("doctor valid report = %+v err=%v", report, err)
	}
	conflicts, err := manager.Doctor(context.Background(), config.DoctorRequest{Proposals: []config.DoctorProposal{
		{Key: "server.other", SemanticRef: "other", GoName: "ServerListen", EnvAlias: "ORQUESTA_SERVER_LISTEN", Type: "string", Scope: "server"},
		{Key: "server.old", TargetKey: config.KeyServerListen, SemanticRef: "network.listen", Alias: "server.listen"},
	}})
	if err != nil || len(conflicts.Conflicts) < 3 {
		t.Fatalf("doctor failed to report generated-name/env/retirement conflicts: %+v err=%v", conflicts, err)
	}
	after, err := manager.View(context.Background())
	if err != nil || after.SourceRevision != before.SourceRevision {
		t.Fatal("doctor mutated configuration")
	}

	const allowed = "V07_ALLOWED_CHILD_ENV"
	const ambient = "V07_AMBIENT_CHILD_ENV"
	t.Setenv(allowed, "kept")
	t.Setenv(ambient, "must-not-leak")
	child, err := config.ResolveChildEnvironment([]string{allowed})
	if err != nil || !reflect.DeepEqual(child, map[string]string{allowed: "kept"}) {
		t.Fatalf("child environment = %#v err=%v", child, err)
	}

	v07AssertNoPublicConfigImports(t, repositoryRoot, "internal/interfaces", "cmd/orquesta")
}

func v07AssertNoPublicConfigImports(t *testing.T, repositoryRoot string, roots ...string) {
	t.Helper()
	for _, relative := range roots {
		err := filepath.WalkDir(filepath.Join(repositoryRoot, relative), func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return walkErr
			}
			fileSet := token.NewFileSet()
			syntax, err := parser.ParseFile(fileSet, path, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}
			for _, imported := range syntax.Imports {
				importPath, err := strconv.Unquote(imported.Path.Value)
				if err != nil {
					return err
				}
				if importPath == "orquesta/internal/config" || strings.HasPrefix(importPath, "orquesta/internal/config/") {
					position := fileSet.Position(imported.Pos())
					return fmt.Errorf("premature V24 public config dependency at %s:%d", path, position.Line)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func v07OpenManager(t *testing.T, path string, active config.Snapshot, environment map[string]string, failpoint func(string) error, fixture v07Fixture) (*config.Manager, *tomlstore.Store) {
	t.Helper()
	store, err := tomlstore.Open(tomlstore.Options{
		Path: path, MaxSourceBytes: fixture.Scenario.MaxSourceBytes,
		MaxReceiptBytes: fixture.Scenario.MaxReceiptBytes, Failpoint: failpoint,
	})
	if err != nil {
		t.Fatalf("open TOML store: %v", err)
	}
	baseTime, err := time.Parse(time.RFC3339Nano, fixture.Scenario.BaseTime)
	if err != nil {
		t.Fatal(err)
	}
	manager, err := config.NewManager(config.ManagerOptions{
		Store: store, Active: active, Environment: environment, Now: func() time.Time { return baseTime },
	})
	if err != nil {
		t.Fatalf("new config Manager: %v", err)
	}
	return manager, store
}

func v07WriteSource(t *testing.T, root, body string) string {
	t.Helper()
	path := filepath.Join(root, "orquesta.toml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func v07Load(t *testing.T, path string) config.Snapshot {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	snapshot, err := config.Resolve(config.ResolveOptions{TOML: content, SourcePath: path})
	if err != nil {
		t.Fatalf("resolve %s: %v", path, err)
	}
	return snapshot
}

func v07AssertKeys(t *testing.T, got []config.Key, want ...config.Key) {
	t.Helper()
	got = append([]config.Key(nil), got...)
	want = append([]config.Key(nil), want...)
	sort.Slice(got, func(i, j int) bool { return got[i] < got[j] })
	sort.Slice(want, func(i, j int) bool { return want[i] < want[j] })
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("keys = %v, want %v", got, want)
	}
}

func v07AssertHumanTOML(t *testing.T, root, sourcePath string, maxReceiptBytes int64, want config.ChangeReceipt) {
	t.Helper()
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"sk-live-secret-material", "actor:v07", "request:v07", "_orquesta", "audit_receipt", "registry_revision"} {
		if bytes.Contains(content, []byte(forbidden)) {
			t.Fatalf("human TOML contains private/schema/audit metadata %q: %s", forbidden, content)
		}
	}
	info, err := os.Stat(sourcePath)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("source mode = %v err=%v, want 0600", info.Mode().Perm(), err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	receipts := 0
	for _, entry := range entries {
		name := strings.ToLower(entry.Name())
		if strings.Contains(name, "receipt") {
			receipts++
			v07AssertReceiptDirectory(t, filepath.Join(root, entry.Name()), maxReceiptBytes, want)
		}
		if strings.Contains(name, "journal") || strings.Contains(name, "intent") || strings.Contains(name, "replacement") ||
			strings.Contains(name, "pending") || strings.HasSuffix(name, ".next") {
			t.Fatalf("recovery left transient sidecar %q", entry.Name())
		}
	}
	if receipts != 1 {
		t.Fatalf("durable audit receipt sidecars = %d, want exactly one", receipts)
	}
}

func v07AssertReceiptDirectory(t *testing.T, path string, maxReceiptBytes int64, want config.ChangeReceipt) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0o700 {
		t.Fatalf("receipt directory = %v err=%v, want private 0700 directory", info, err)
	}
	entries, err := os.ReadDir(path)
	if err != nil || len(entries) != 1 {
		t.Fatalf("receipt entries = %d err=%v, want exactly one", len(entries), err)
	}
	receiptInfo, err := entries[0].Info()
	if err != nil || !receiptInfo.Mode().IsRegular() || receiptInfo.Mode().Perm() != 0o400 ||
		receiptInfo.Size() <= 0 || receiptInfo.Size() > maxReceiptBytes {
		t.Fatalf("receipt file = %v err=%v, want bounded owner-read-only regular file", receiptInfo, err)
	}
	payload, err := os.ReadFile(filepath.Join(path, entries[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range [][]byte{[]byte(`"content"`), []byte(`"replacement"`), []byte("credential:"), []byte("sk-live")} {
		if bytes.Contains(bytes.ToLower(payload), forbidden) {
			t.Fatalf("receipt sidecar contains source or credential material %q: %s", forbidden, payload)
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var envelope v07PersistedReceiptEnvelope
	if err := decoder.Decode(&envelope); err != nil {
		t.Fatalf("decode receipt sidecar: %v", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		t.Fatalf("receipt sidecar has trailing JSON: %v", err)
	}
	wantPersisted := v07PersistedReceiptV1{
		ReceiptRef: want.ReceiptRef, ActorRef: want.ActorRef, RequestRef: want.RequestRef,
		Fingerprint: want.Fingerprint, BeforeRevision: want.BeforeRevision, AfterRevision: want.AfterRevision,
		ChangedKeys:        append([]config.Key(nil), want.ChangedKeys...),
		PendingRestartKeys: append([]config.Key(nil), want.PendingRestartKeys...), ChangedAt: want.ChangedAt,
	}
	if envelope.SchemaVersion != 1 || envelope.DocumentType != "orquesta.config_change_receipt" ||
		!reflect.DeepEqual(envelope.Receipt, wantPersisted) {
		t.Fatalf("receipt sidecar is not causal proof of returned receipt: got=%+v want=%+v", envelope, wantPersisted)
	}
}

func v07DirectorySnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	result := make(map[string]string)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return walkErr
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		result[filepath.ToSlash(relative)] = fmt.Sprintf("mode=%o size=%d mtime=%d\n%s", info.Mode(), info.Size(), info.ModTime().UnixNano(), content)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}
