package acceptance_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"syscall"
	"testing"
	"time"

	credentiallocal "orquesta/internal/adapters/credentials/local"
	"orquesta/internal/config"
	"orquesta/internal/credentials"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

const v08FixturePath = "acceptance/fixtures/v08_credentials.json"
const v08TrustedBaseGitCommitOID = "6b7bdd2002da73b11265f2c61e1046d6cd60b110"
const v08ProductDeltaBaseGitCommitOID = "6b7bdd2002da73b11265f2c61e1046d6cd60b110"
const v08ProductDeltaSealedGitCommitOID = "6b7bdd2002da73b11265f2c61e1046d6cd60b110"

type v08Fixture struct {
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
	DeferredCapabilities           []v08DeferredCapability `json:"deferred_capabilities"`
	Scenario                       v08Scenario             `json:"scenario"`
	Assertions                     []string                `json:"assertions"`
}

type v08DeferredCapability struct {
	ID                 string `json:"id"`
	Owner              string `json:"owner"`
	AcceptanceContract string `json:"acceptance_contract"`
}

type v08Scenario struct {
	BaseTime               string            `json:"base_time"`
	CredentialRef          string            `json:"credential_ref"`
	OwnerRef               string            `json:"owner_ref"`
	OtherOwnerRef          string            `json:"other_owner_ref"`
	ScopeRefs              []string          `json:"scope_refs"`
	PurposeRef             string            `json:"purpose_ref"`
	InitialVersion         uint64            `json:"initial_version"`
	MaxStoreBytes          int64             `json:"max_store_bytes"`
	DirectoryMode          string            `json:"directory_mode"`
	SourceMode             string            `json:"source_mode"`
	Failpoints             []string          `json:"failpoints"`
	ChildPublicEnvironment map[string]string `json:"child_public_environment"`
	ChildBindings          []v08ChildBinding `json:"child_bindings"`
	ForbiddenAmbientNames  []string          `json:"forbidden_ambient_names"`
}

type v08ChildBinding struct {
	EnvironmentName string `json:"environment_name"`
	CredentialRef   string `json:"credential_ref"`
	Version         uint64 `json:"version"`
}

type v08ProbeStore struct{}

func (*v08ProbeStore) Create(context.Context, credentials.CreateRequest) (credentials.MutationResult, error) {
	return credentials.MutationResult{}, errors.New("probe.create")
}

func (*v08ProbeStore) Use(context.Context, credentials.UseRequest, func(credentials.Secret) error) (credentials.Receipt, error) {
	return credentials.Receipt{}, errors.New("probe.use")
}

func (*v08ProbeStore) Rotate(context.Context, credentials.RotateRequest) (credentials.MutationResult, error) {
	return credentials.MutationResult{}, errors.New("probe.rotate")
}

func (*v08ProbeStore) Revoke(context.Context, credentials.RevokeRequest) (credentials.MutationResult, error) {
	return credentials.MutationResult{}, errors.New("probe.revoke")
}

var _ credentials.Store = (*v08ProbeStore)(nil)

func TestAcceptanceV08Credentials(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v08Fixture](t, filepath.Join(repositoryRoot, filepath.FromSlash(v08FixturePath)))
	v08AssertFixtureHeader(t, repositoryRoot, fixture)

	t.Run("neutral_port_opaque_refs_and_no_deferred_surfaces", func(t *testing.T) {
		v08AssertPortAndScope(t, fixture)
	})
	t.Run("exact_owner_scope_purpose_version_rotation_and_revocation", func(t *testing.T) {
		v08AssertLifecycle(t, fixture)
	})
	t.Run("private_file_negatives_and_fsync_crash_recovery", func(t *testing.T) {
		v08AssertFilesystemAndRecovery(t, fixture)
	})
	t.Run("redaction_leak_scan_and_ref_only_durable_surfaces", func(t *testing.T) {
		v08AssertLeakGuardAndDurableRefs(t, fixture)
	})
	t.Run("child_environment_is_exact_callback_scoped_and_nonambient", func(t *testing.T) {
		v08AssertChildEnvironment(t, fixture)
	})
}

func TestAcceptanceV08CredentialsReceipt(t *testing.T) {
	evidenceAssertReceiptV3(t, evidenceRepositoryRoot(t), evidenceReceiptV3Expectation{
		Contract: "AC-V08-CREDENTIALS", FixturePath: v08FixturePath,
		ReceiptPath:       "product/evidence/v08_credentials.json",
		ExecutedNotBefore: "2026-07-15T00:00:00Z", TrustedBaseGitCommitOID: v08TrustedBaseGitCommitOID,
	})
}

func TestV08CandidateSubjectsCoverCommittedDelta(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v08Fixture](t, filepath.Join(repositoryRoot, filepath.FromSlash(v08FixturePath)))
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
		t.Fatalf("V08 candidate subjects differ from sealed product delta:\nchanged=%v\nfixture=%v", changed, fixture.CandidateSubjects)
	}
}

func v08AssertFixtureHeader(t *testing.T, repositoryRoot string, fixture v08Fixture) {
	t.Helper()
	wantCapabilities := []string{"EVD-11", "EVD-12", "OPS-03", "OPS-08"}
	wantDeferred := []v08DeferredCapability{{
		ID: "EVD-13", Owner: "test_attestor", AcceptanceContract: "AC-V17-TEST-ATTESTOR",
	}}
	wantFailpoints := []string{
		"after_replacement_sync", "after_store_rename", "after_store_directory_sync",
	}
	if fixture.SchemaVersion != 1 || fixture.ReceiptSchemaVersion != 3 ||
		fixture.ContractID != "AC-V08-CREDENTIALS" || fixture.TrustedBaseGitCommitOID != v08TrustedBaseGitCommitOID ||
		fixture.ProductDeltaBaseGitCommitOID != v08ProductDeltaBaseGitCommitOID ||
		fixture.ProductDeltaSealedGitCommitOID != v08ProductDeltaSealedGitCommitOID ||
		fixture.OutputPath != "product/evidence/v08_credentials.output.txt" ||
		fixture.ReceiptPath != "product/evidence/v08_credentials.json" ||
		!reflect.DeepEqual(fixture.OwnedCapabilityIDs, wantCapabilities) ||
		!reflect.DeepEqual(fixture.DeferredCapabilities, wantDeferred) || len(fixture.Assertions) != 11 ||
		fixture.Scenario.InitialVersion != 1 || fixture.Scenario.MaxStoreBytes != 1<<20 ||
		fixture.Scenario.DirectoryMode != "0700" || fixture.Scenario.SourceMode != "0600" ||
		!reflect.DeepEqual(fixture.Scenario.Failpoints, wantFailpoints) ||
		len(fixture.Scenario.ScopeRefs) != 2 || len(fixture.Scenario.ChildBindings) != 1 {
		t.Fatalf("invalid V08 fixture header: %+v", fixture)
	}
	if _, err := time.Parse(time.RFC3339Nano, fixture.Scenario.BaseTime); err != nil {
		t.Fatalf("invalid V08 base time: %v", err)
	}
	if fixture.Command != "sh -c '"+v08ValidationShellBody()+"'" ||
		!reflect.DeepEqual(fixture.ExecutionArgv, []string{"sh", "-c", v08ValidationShellBody()}) {
		t.Fatalf("invalid V08 command/argv: %q %#v", fixture.Command, fixture.ExecutionArgv)
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

func v08ValidationShellBody() string {
	return "go test -mod=vendor -count=1 . ./acceptance -run \"^(TestProductRoadmapIsExhaustiveAndCausal|TestProductRoadmapV08ScopeAndExecutableContract|TestRebuildArchitecture|TestTraceabilityRebuildBugLessons|TestAcceptanceV08Credentials|TestV08CandidateSubjectsCoverCommittedDelta)$\"" +
		" && go test -mod=vendor -count=1 ./internal/credentials ./internal/adapters/credentials/local ./internal/config ./internal/goal ./internal/bootstrap ./cmd/orquesta"
}

func v08AssertPortAndScope(t *testing.T, fixture v08Fixture) {
	t.Helper()
	port := reflect.TypeOf((*credentials.Store)(nil)).Elem()
	if port.NumMethod() != 4 {
		t.Fatalf("CredentialStore methods = %d, want Create Use Rotate Revoke only", port.NumMethod())
	}
	for _, method := range []string{"Create", "Use", "Rotate", "Revoke"} {
		if _, found := port.MethodByName(method); !found {
			t.Fatalf("CredentialStore lacks %s", method)
		}
	}
	var interchangeable credentials.Store = &v08ProbeStore{}
	if interchangeable == nil {
		t.Fatal("CredentialStore cannot be replaced")
	}

	root := t.TempDir()
	store := v08OpenStore(t, root, fixture, nil, time.Time{})
	_, err := store.Create(context.Background(), credentials.CreateRequest{
		ActorRef: "actor:v08-operator", RequestRef: "request:v08-invalid-ref",
		CredentialRef: credentials.CredentialRef("credential:../escape"),
		OwnerRef:      credentials.OwnerRef(fixture.Scenario.OwnerRef),
		ScopeRefs:     v08ScopeRefs(fixture), PurposeRef: credentials.PurposeRef(fixture.Scenario.PurposeRef),
		Material: v08Secret(t, v08InitialSecretBytes()),
	})
	if !credentials.HasErrorCode(err, credentials.ErrorInvalidRef) {
		t.Fatalf("traversal-shaped opaque ref accepted: %v", err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(root), "escape")); !os.IsNotExist(err) {
		t.Fatalf("invalid credential ref escaped store root: %v", err)
	}
}

func v08AssertLifecycle(t *testing.T, fixture v08Fixture) {
	t.Helper()
	root := t.TempDir()
	store := v08OpenStore(t, root, fixture, nil, time.Time{})
	ctx := context.Background()
	created := v08Create(t, ctx, store, fixture, "request:v08-create")
	if created.Metadata.CredentialRef != credentials.CredentialRef(fixture.Scenario.CredentialRef) ||
		created.Metadata.OwnerRef != credentials.OwnerRef(fixture.Scenario.OwnerRef) ||
		!reflect.DeepEqual(created.Metadata.ScopeRefs, v08ScopeRefs(fixture)) ||
		created.Metadata.PurposeRef != credentials.PurposeRef(fixture.Scenario.PurposeRef) ||
		created.Metadata.Version != credentials.Version(fixture.Scenario.InitialVersion) || created.Metadata.Revoked {
		t.Fatalf("created credential metadata = %+v", created.Metadata)
	}
	v08AssertReceiptRedacted(t, created.Receipt, v08InitialSecretBytes())

	used := v08Use(t, ctx, store, fixture, credentials.Version(1), v08InitialSecretBytes(), "request:v08-use-1")
	if used.Version != 1 || used.CredentialRef != credentials.CredentialRef(fixture.Scenario.CredentialRef) {
		t.Fatalf("use receipt = %+v", used)
	}

	negativeUses := []struct {
		name string
		edit func(*credentials.UseRequest)
		code credentials.ErrorCode
	}{
		{"owner_spoof", func(request *credentials.UseRequest) {
			request.OwnerRef = credentials.OwnerRef(fixture.Scenario.OtherOwnerRef)
		}, credentials.ErrorOwnerMismatch},
		{"scope_prefix", func(request *credentials.UseRequest) {
			request.ScopeRef = credentials.ScopeRef(fixture.Scenario.ScopeRefs[0] + ":child")
		}, credentials.ErrorScopeDenied},
		{"purpose_prefix", func(request *credentials.UseRequest) {
			request.PurposeRef = credentials.PurposeRef(fixture.Scenario.PurposeRef + ":other")
		}, credentials.ErrorPurposeDenied},
		{"stale_or_future_version", func(request *credentials.UseRequest) { request.Version = 2 }, credentials.ErrorVersionConflict},
	}
	for _, test := range negativeUses {
		t.Run(test.name, func(t *testing.T) {
			request := v08UseRequest(fixture, 1, "request:v08-negative-"+test.name)
			test.edit(&request)
			called := false
			_, err := store.Use(ctx, request, func(credentials.Secret) error { called = true; return nil })
			if !credentials.HasErrorCode(err, test.code) || called {
				t.Fatalf("negative use = err=%v callback=%v", err, called)
			}
		})
	}

	rotated, err := store.Rotate(ctx, v08RotateRequest(t, fixture, "request:v08-rotate"))
	if err != nil || rotated.Metadata.Version != 2 || rotated.Metadata.Revoked || rotated.Replayed {
		t.Fatalf("rotate = %+v err=%v", rotated, err)
	}
	replayed, err := store.Rotate(ctx, v08RotateRequest(t, fixture, "request:v08-rotate"))
	if err != nil || !replayed.Replayed || !reflect.DeepEqual(replayed, credentials.MutationResult{
		Metadata: rotated.Metadata, Receipt: rotated.Receipt, Replayed: true,
	}) {
		t.Fatalf("rotation replay = %+v err=%v", replayed, err)
	}
	conflictingReplay := v08RotateRequest(t, fixture, "request:v08-rotate")
	conflictingReplay.Material = v08Secret(t, v08InitialSecretBytes())
	if _, err := store.Rotate(ctx, conflictingReplay); !credentials.HasErrorCode(err, credentials.ErrorIdempotencyConflict) {
		t.Fatalf("same request ref with different material accepted: %v", err)
	}
	_, err = store.Use(ctx, v08UseRequest(fixture, 1, "request:v08-old-version"), func(credentials.Secret) error {
		t.Fatal("old credential material was exposed after rotation")
		return nil
	})
	if !credentials.HasErrorCode(err, credentials.ErrorVersionConflict) {
		t.Fatalf("old version use = %v", err)
	}
	v08Use(t, ctx, store, fixture, 2, v08RotatedSecretBytes(), "request:v08-use-2")

	revoked, err := store.Revoke(ctx, credentials.RevokeRequest{
		ActorRef: "actor:v08-operator", RequestRef: "request:v08-revoke",
		CredentialRef: credentials.CredentialRef(fixture.Scenario.CredentialRef),
		OwnerRef:      credentials.OwnerRef(fixture.Scenario.OwnerRef), ExpectedVersion: 2,
		Reason: "operator_rotation_complete",
	})
	if err != nil || !revoked.Metadata.Revoked || revoked.Metadata.Version != 2 {
		t.Fatalf("revoke = %+v err=%v", revoked, err)
	}
	source, err := os.ReadFile(filepath.Join(root, "credentials.json"))
	encoded := []byte(base64.StdEncoding.EncodeToString(v08RotatedSecretBytes()))
	if err != nil || bytes.Contains(source, v08RotatedSecretBytes()) || bytes.Contains(source, encoded) {
		t.Fatalf("revocation retained current material: err=%v", err)
	}
	for _, requestRef := range []string{"request:v08-use-after-revoke", "request:v08-use-2"} {
		called := false
		_, err := store.Use(ctx, v08UseRequest(fixture, 2, requestRef), func(credentials.Secret) error {
			called = true
			return nil
		})
		if !credentials.HasErrorCode(err, credentials.ErrorRevoked) || called {
			t.Fatalf("revocation did not dominate request %q replay: err=%v called=%v", requestRef, err, called)
		}
	}
	v08AssertConcurrentRotate(t, fixture)
}

func v08AssertConcurrentRotate(t *testing.T, fixture v08Fixture) {
	store := v08OpenStore(t, t.TempDir(), fixture, nil, time.Time{})
	v08Create(t, context.Background(), store, fixture, "request:v08-race-create")
	results := make(chan error, 2)
	for index := 0; index < 2; index++ {
		go func(index int) {
			_, err := store.Rotate(context.Background(), v08RotateRequest(t, fixture, fmt.Sprintf("request:v08-race-%d", index)))
			results <- err
		}(index)
	}
	succeeded, conflicted := 0, 0
	for range 2 {
		err := <-results
		if err == nil {
			succeeded++
		} else if credentials.HasErrorCode(err, credentials.ErrorVersionConflict) {
			conflicted++
		}
	}
	if succeeded != 1 || conflicted != 1 {
		t.Fatalf("concurrent expected-version rotation = succeeded %d conflicted %d", succeeded, conflicted)
	}
}

func v08AssertFilesystemAndRecovery(t *testing.T, fixture v08Fixture) {
	t.Helper()
	root := t.TempDir()
	store := v08OpenStore(t, root, fixture, nil, time.Time{})
	v08Create(t, context.Background(), store, fixture, "request:v08-private-store")
	v08AssertPrivateStore(t, root, filepath.Join(root, "credentials.json"), fixture)
	for _, test := range []struct {
		name    string
		prepare func(string, string, *credentiallocal.Options) error
	}{
		{"symlink", func(root, path string, _ *credentiallocal.Options) error {
			target := filepath.Join(root, "target.json")
			return errors.Join(os.WriteFile(target, []byte("{}\n"), 0o600), os.Symlink(target, path))
		}},
		{"parent_symlink", func(root, _ string, options *credentiallocal.Options) error {
			target, link := filepath.Join(root, "target"), filepath.Join(root, "link")
			options.Path = filepath.Join(link, "credentials.json")
			return errors.Join(os.Mkdir(target, 0o700), os.Symlink(target, link))
		}},
		{"hardlink", func(root, path string, _ *credentiallocal.Options) error {
			v08Create(t, context.Background(), v08OpenStore(t, root, fixture, nil, time.Time{}), fixture, "request:v08-hardlink")
			return os.Link(path, filepath.Join(root, "copy.json"))
		}},
		{"source_mode", func(root, path string, _ *credentiallocal.Options) error {
			v08Create(t, context.Background(), v08OpenStore(t, root, fixture, nil, time.Time{}), fixture, "request:v08-mode")
			return os.Chmod(path, 0o640)
		}},
		{"directory_mode", func(root, _ string, _ *credentiallocal.Options) error { return os.Chmod(root, 0o750) }},
		{"owner", func(root, path string, options *credentiallocal.Options) error {
			v08Create(t, context.Background(), v08OpenStore(t, root, fixture, nil, time.Time{}), fixture, "request:v08-owner")
			options.OwnerUID++
			return nil
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			root, path := t.TempDir(), ""
			path = filepath.Join(root, "credentials.json")
			options := v08LocalOptions(path, fixture, nil, time.Time{})
			if err := test.prepare(root, path, &options); err != nil {
				t.Fatal(err)
			}
			if _, err := credentiallocal.Open(options); !credentials.HasErrorCode(err, credentials.ErrorUnsafeFile) {
				t.Fatalf("unsafe %s store accepted: %v", test.name, err)
			}
		})
	}
	v08AssertFsyncBoundaries(t, fixture)
}

func v08AssertFsyncBoundaries(t *testing.T, fixture v08Fixture) {
	t.Helper()
	root := t.TempDir()
	options := v08LocalOptions(filepath.Join(root, "credentials.json"), fixture, nil, time.Time{})
	fsyncs := 0
	options.Sync = func(file *os.File) error { fsyncs++; return file.Sync() }
	store, err := credentiallocal.Open(options)
	if err != nil {
		t.Fatal(err)
	}
	v08Create(t, context.Background(), store, fixture, "request:v08-fsync-real-create")
	fsyncs = 0
	_, err = store.Rotate(context.Background(), v08RotateRequest(t, fixture, "request:v08-fsync-real-rotate"))
	if err != nil || fsyncs < 2 {
		t.Fatalf("atomic replacement did not execute injected real fsyncs: count=%d err=%v", fsyncs, err)
	}
	for _, stage := range fixture.Scenario.Failpoints {
		t.Run(stage, func(t *testing.T) {
			root := t.TempDir()
			failed := v08OpenStore(t, root, fixture, func(current string) error {
				if current == stage {
					return errors.New("simulated_interruption")
				}
				return nil
			}, time.Time{})
			v08Create(t, context.Background(), failed, fixture, "request:v08-fsync-create-"+stage)
			request := v08RotateRequest(t, fixture, "request:v08-fsync-rotate-"+stage)
			if _, err := failed.Rotate(context.Background(), request); err == nil {
				t.Fatalf("fsync failpoint %s was not reached", stage)
			}
			recovered := v08OpenStore(t, root, fixture, nil, time.Time{})
			result, err := recovered.Rotate(context.Background(), request)
			if err != nil || result.Metadata.Version != 2 {
				t.Fatalf("fsync recovery/replay at %s = %+v err=%v", stage, result, err)
			}
			v08Use(t, context.Background(), recovered, fixture, 2, v08RotatedSecretBytes(), "request:v08-fsync-use-"+stage)
		})
	}
}

func v08AssertLeakGuardAndDurableRefs(t *testing.T, fixture v08Fixture) {
	t.Helper()
	initial := v08Secret(t, v08InitialSecretBytes())
	guard, err := credentials.NewLeakGuard(initial)
	if err != nil {
		t.Fatalf("new leak guard: %v", err)
	}

	root := t.TempDir()
	store := v08OpenStore(t, root, fixture, nil, time.Time{})
	created := v08Create(t, context.Background(), store, fixture, "request:v08-leak-create")
	goalPayload := v08GoalPayload(t, fixture)
	promptPayload := v08JSON(t, ports.AgentLaunchRequest{
		Objective: "use " + fixture.Scenario.CredentialRef,
	})
	receiptPayload := v08JSON(t, created.Receipt)
	snapshot, err := config.Resolve(config.ResolveOptions{TOML: []byte(
		"[runtime.codex]\ncredential_ref = \"" + fixture.Scenario.CredentialRef + "\"\n",
	)})
	if err != nil {
		t.Fatal(err)
	}
	effectivePayload, err := snapshot.EffectiveJSON()
	if err != nil {
		t.Fatal(err)
	}

	surfaces := []credentials.LeakSurface{
		{Name: "goal", Content: goalPayload},
		{Name: "prompt", Content: promptPayload},
		{Name: "receipt", Content: receiptPayload},
		{Name: "effective_config", Content: effectivePayload},
	}
	if err := guard.Scan(surfaces); err != nil {
		t.Fatalf("clean surfaces reported secret leak: %v", err)
	}
	contaminated := append(append([]byte(nil), promptPayload...), v08InitialSecretBytes()...)
	if err := guard.Scan([]credentials.LeakSurface{{Name: "prompt", Content: contaminated}}); !credentials.HasErrorCode(err, credentials.ErrorSecretLeak) || bytes.Contains([]byte(err.Error()), v08InitialSecretBytes()) {
		t.Fatalf("prompt leak result = %v", err)
	}
	redacted := guard.Redact(contaminated)
	if bytes.Contains(redacted, v08InitialSecretBytes()) || !bytes.Contains(redacted, []byte("[REDACTED]")) {
		t.Fatalf("prompt was not safely redacted: %s", redacted)
	}
	for _, value := range []any{initial, created} {
		jsonPayload := v08JSON(t, value)
		formatted := []byte(fmt.Sprintf("%v %#v", value, value))
		if bytes.Contains(jsonPayload, v08InitialSecretBytes()) || bytes.Contains(formatted, v08InitialSecretBytes()) {
			t.Fatalf("public secret projection leaked for %T: json=%s fmt=%s", value, jsonPayload, formatted)
		}
	}
	if !bytes.Contains(goalPayload, []byte(fixture.Scenario.CredentialRef)) || bytes.Contains(goalPayload, v08InitialSecretBytes()) {
		t.Fatalf("Goal payload does not preserve ref-only credential use: %s", goalPayload)
	}
}

func v08AssertChildEnvironment(t *testing.T, fixture v08Fixture) {
	t.Helper()
	root := t.TempDir()
	store := v08OpenStore(t, root, fixture, nil, time.Time{})
	v08Create(t, context.Background(), store, fixture, "request:v08-child-create")
	_, err := store.Rotate(context.Background(), v08RotateRequest(t, fixture, "request:v08-child-rotate"))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range fixture.Scenario.ForbiddenAmbientNames {
		t.Setenv(name, "ambient-must-not-leak")
	}
	request := v08ChildEnvironmentRequest(fixture)
	called := false
	output, receipts, err := credentials.WithChildEnvironment(context.Background(), store, request, func(environment []string) (credentials.ChildOutput, error) {
		called = true
		want := []string{
			"PATH=" + fixture.Scenario.ChildPublicEnvironment["PATH"],
			"TMPDIR=" + fixture.Scenario.ChildPublicEnvironment["TMPDIR"],
			fixture.Scenario.ChildBindings[0].EnvironmentName + "=" + string(v08RotatedSecretBytes()),
		}
		sort.Strings(want)
		if !reflect.DeepEqual(environment, want) {
			return credentials.ChildOutput{}, fmt.Errorf("child environment = %q, want exact %q", environment, want)
		}
		for _, forbidden := range fixture.Scenario.ForbiddenAmbientNames {
			for _, entry := range environment {
				if strings.HasPrefix(entry, forbidden+"=") {
					return credentials.ChildOutput{}, fmt.Errorf("ambient environment %q leaked", forbidden)
				}
			}
		}
		return credentials.ChildOutput{Result: []byte("ok")}, nil
	})
	if err != nil || !called || string(output.Result) != "ok" || len(receipts) != len(fixture.Scenario.ChildBindings) {
		t.Fatalf("child delivery = called=%v output=%+v receipts=%+v err=%v", called, output, receipts, err)
	}
	if payload := v08JSON(t, receipts); bytes.Contains(payload, v08RotatedSecretBytes()) {
		t.Fatalf("child delivery receipt leaked material: %s", payload)
	}
	malicious, maliciousReceipts, err := credentials.WithChildEnvironment(context.Background(), store, request, func(environment []string) (credentials.ChildOutput, error) {
		material := []byte(strings.TrimPrefix(environment[len(environment)-1], fixture.Scenario.ChildBindings[0].EnvironmentName+"="))
		return credentials.ChildOutput{Result: material, Diagnostic: material, Artifact: material}, nil
	})
	if !credentials.HasErrorCode(err, credentials.ErrorSecretLeak) || len(malicious.Result)+len(malicious.Diagnostic)+len(malicious.Artifact) != 0 ||
		bytes.Contains([]byte(err.Error()), v08RotatedSecretBytes()) || bytes.Contains(v08JSON(t, maliciousReceipts), v08RotatedSecretBytes()) {
		t.Fatalf("malicious child output escaped leak gate: output=%+v receipts=%+v err=%v", malicious, maliciousReceipts, err)
	}

	negative := []struct {
		name string
		edit func(*credentials.ChildEnvironmentRequest)
		code credentials.ErrorCode
	}{
		{"owner", func(request *credentials.ChildEnvironmentRequest) {
			request.OwnerRef = credentials.OwnerRef(fixture.Scenario.OtherOwnerRef)
		}, credentials.ErrorOwnerMismatch},
		{"scope", func(request *credentials.ChildEnvironmentRequest) {
			request.ScopeRef = credentials.ScopeRef(fixture.Scenario.ScopeRefs[0] + ":child")
		}, credentials.ErrorScopeDenied},
		{"purpose", func(request *credentials.ChildEnvironmentRequest) {
			request.PurposeRef = credentials.PurposeRef(fixture.Scenario.PurposeRef + ":other")
		}, credentials.ErrorPurposeDenied},
		{"version", func(request *credentials.ChildEnvironmentRequest) { request.Bindings[0].Version = 1 }, credentials.ErrorVersionConflict},
		{"duplicate_environment", func(request *credentials.ChildEnvironmentRequest) {
			request.PublicEnvironment[request.Bindings[0].EnvironmentName] = "spoof"
		}, credentials.ErrorChildEnvironment},
	}
	for _, test := range negative {
		t.Run(test.name, func(t *testing.T) {
			request := v08ChildEnvironmentRequest(fixture)
			test.edit(&request)
			called := false
			_, _, err := credentials.WithChildEnvironment(context.Background(), store, request, func([]string) (credentials.ChildOutput, error) {
				called = true
				return credentials.ChildOutput{}, nil
			})
			if !credentials.HasErrorCode(err, test.code) || called {
				t.Fatalf("unsafe child request = err=%v called=%v", err, called)
			}
		})
	}
}

func v08OpenStore(t *testing.T, root string, fixture v08Fixture, failpoint func(string) error, now time.Time) *credentiallocal.Store {
	t.Helper()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "credentials.json")
	store, err := credentiallocal.Open(v08LocalOptions(path, fixture, failpoint, now))
	if err != nil {
		t.Fatalf("open local CredentialStore: %v", err)
	}
	return store
}

func v08LocalOptions(path string, fixture v08Fixture, failpoint func(string) error, now time.Time) credentiallocal.Options {
	if now.IsZero() {
		parsed, _ := time.Parse(time.RFC3339Nano, fixture.Scenario.BaseTime)
		now = parsed
	}
	return credentiallocal.Options{
		Path: path, OwnerUID: os.Getuid(), MaxStoreBytes: fixture.Scenario.MaxStoreBytes,
		Failpoint: failpoint, Now: func() time.Time { return now },
	}
}

func v08Create(t *testing.T, ctx context.Context, store credentials.Store, fixture v08Fixture, requestRef string) credentials.MutationResult {
	t.Helper()
	result, err := store.Create(ctx, credentials.CreateRequest{
		ActorRef: "actor:v08-operator", RequestRef: requestRef,
		CredentialRef: credentials.CredentialRef(fixture.Scenario.CredentialRef),
		OwnerRef:      credentials.OwnerRef(fixture.Scenario.OwnerRef), ScopeRefs: v08ScopeRefs(fixture),
		PurposeRef: credentials.PurposeRef(fixture.Scenario.PurposeRef), Material: v08Secret(t, v08InitialSecretBytes()),
	})
	if err != nil {
		t.Fatalf("create credential: %v", err)
	}
	return result
}

func v08Use(t *testing.T, ctx context.Context, store credentials.Store, fixture v08Fixture, version credentials.Version, want []byte, requestRef string) credentials.Receipt {
	t.Helper()
	called := 0
	receipt, err := store.Use(ctx, v08UseRequest(fixture, version, requestRef), func(secret credentials.Secret) error {
		called++
		got := secret.Bytes()
		if !bytes.Equal(got, want) {
			return fmt.Errorf("secret material mismatch")
		}
		got[0] ^= 0xff
		if bytes.Equal(secret.Bytes(), got) {
			return fmt.Errorf("secret Bytes returned mutable backing storage")
		}
		return nil
	})
	if err != nil || called != 1 {
		t.Fatalf("use credential = called=%d receipt=%+v err=%v", called, receipt, err)
	}
	v08AssertReceiptRedacted(t, receipt, want)
	return receipt
}

func v08UseRequest(fixture v08Fixture, version credentials.Version, requestRef string) credentials.UseRequest {
	return credentials.UseRequest{
		ActorRef: "actor:v08-worker", RequestRef: requestRef,
		CredentialRef: credentials.CredentialRef(fixture.Scenario.CredentialRef),
		OwnerRef:      credentials.OwnerRef(fixture.Scenario.OwnerRef),
		ScopeRef:      credentials.ScopeRef(fixture.Scenario.ScopeRefs[0]),
		PurposeRef:    credentials.PurposeRef(fixture.Scenario.PurposeRef), Version: version,
	}
}

func v08RotateRequest(t *testing.T, fixture v08Fixture, requestRef string) credentials.RotateRequest {
	return credentials.RotateRequest{
		ActorRef: "actor:v08-operator", RequestRef: requestRef,
		CredentialRef: credentials.CredentialRef(fixture.Scenario.CredentialRef),
		OwnerRef:      credentials.OwnerRef(fixture.Scenario.OwnerRef), ExpectedVersion: 1,
		Material: v08Secret(t, v08RotatedSecretBytes()),
	}
}

func v08ScopeRefs(fixture v08Fixture) []credentials.ScopeRef {
	refs := make([]credentials.ScopeRef, len(fixture.Scenario.ScopeRefs))
	for index, value := range fixture.Scenario.ScopeRefs {
		refs[index] = credentials.ScopeRef(value)
	}
	return refs
}

func v08ChildEnvironmentRequest(fixture v08Fixture) credentials.ChildEnvironmentRequest {
	bindings := make([]credentials.ChildBinding, len(fixture.Scenario.ChildBindings))
	for index, binding := range fixture.Scenario.ChildBindings {
		bindings[index] = credentials.ChildBinding{
			EnvironmentName: binding.EnvironmentName,
			CredentialRef:   credentials.CredentialRef(binding.CredentialRef),
			Version:         credentials.Version(binding.Version),
		}
	}
	public := make(map[string]string, len(fixture.Scenario.ChildPublicEnvironment))
	for name, value := range fixture.Scenario.ChildPublicEnvironment {
		public[name] = value
	}
	return credentials.ChildEnvironmentRequest{
		ActorRef: "actor:v08-worker", RequestRef: "request:v08-child-environment",
		OwnerRef:          credentials.OwnerRef(fixture.Scenario.OwnerRef),
		ScopeRef:          credentials.ScopeRef(fixture.Scenario.ScopeRefs[0]),
		PurposeRef:        credentials.PurposeRef(fixture.Scenario.PurposeRef),
		PublicEnvironment: public, Bindings: bindings,
	}
}

func v08Secret(t *testing.T, material []byte) credentials.Secret {
	t.Helper()
	secret, err := credentials.NewSecret(material)
	if err != nil {
		t.Fatalf("new secret: %v", err)
	}
	return secret
}

func v08InitialSecretBytes() []byte {
	return []byte(strings.Join([]string{"v08", "live", "initial", "material", "7dfe"}, "-"))
}

func v08RotatedSecretBytes() []byte {
	return []byte(strings.Join([]string{"v08", "live", "rotated", "material", "9ac1"}, "-"))
}

func v08AssertReceiptRedacted(t *testing.T, receipt credentials.Receipt, material []byte) {
	t.Helper()
	payload := v08JSON(t, receipt)
	formatted := []byte(fmt.Sprintf("%v %#v", receipt, receipt))
	if bytes.Contains(payload, material) || bytes.Contains(formatted, material) ||
		receipt.CredentialRef.String() == "" || receipt.OwnerRef.String() == "" ||
		receipt.PurposeRef.String() == "" || receipt.Version == 0 || receipt.RequestRef == "" || receipt.ActorRef == "" {
		t.Fatalf("credential receipt is incomplete or leaked material: json=%s fmt=%s receipt=%+v", payload, formatted, receipt)
	}
}

func v08AssertPrivateStore(t *testing.T, root, sourcePath string, fixture v08Fixture) {
	t.Helper()
	rootInfo, err := os.Stat(root)
	if err != nil || rootInfo.Mode().Perm() != 0o700 {
		t.Fatalf("credential root must be 0700: %v %v", rootInfo, err)
	}
	sourceInfo, err := os.Lstat(sourcePath)
	if err != nil || !sourceInfo.Mode().IsRegular() || sourceInfo.Mode().Perm() != 0o600 ||
		sourceInfo.Size() <= 0 || sourceInfo.Size() > fixture.Scenario.MaxStoreBytes {
		t.Fatalf("credential source must be bounded regular 0600: %v %v", sourceInfo, err)
	}
	stat, ok := sourceInfo.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != uint32(os.Getuid()) || stat.Nlink != 1 {
		t.Fatalf("credential source owner/link count = %#v, want uid=%d nlink=1", stat, os.Getuid())
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 1 || entries[0].Name() != filepath.Base(sourcePath) {
		t.Fatalf("CredentialStore must settle as one private document: %v %v", entries, err)
	}
}

func v08GoalPayload(t *testing.T, fixture v08Fixture) []byte {
	t.Helper()
	return v08JSON(t, goal.GoalSnapshot{
		SchemaVersion: goal.GoalSnapshotSchemaVersion, Ref: "goal:v08", ActorRef: "actor:v08-alice", ProjectRef: "project:v08",
		AppSpec: goal.AppSpecSnapshot{Ref: "app-spec:v08", Objective: "use " + fixture.Scenario.CredentialRef,
			Intent: goal.IntentManifestSnapshot{Ref: "intent:v08", Statement: "execute with " + fixture.Scenario.CredentialRef}},
	})
}

func v08JSON(t *testing.T, value any) []byte {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal %T: %v", value, err)
	}
	return payload
}
