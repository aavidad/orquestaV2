package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"orquesta/internal/adapters/system/local"
	gitlocal "orquesta/internal/adapters/workspace/gitlocal"
	"orquesta/internal/application"
	"orquesta/internal/config"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func newV16Harness(t *testing.T, fixture v16E2EFixture, writes map[string]v16Write) *v16Harness {
	t.Helper()
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatalf("Git CLI required by V16 E2E: %v", err)
	}
	gitPath, err = filepath.Abs(gitPath)
	if err != nil {
		t.Fatal(err)
	}
	seed := filepath.Join(root, "seed")
	if err := os.Mkdir(seed, 0o700); err != nil {
		t.Fatal(err)
	}
	v16Git(t, gitPath, seed, "init", "-b", "main")
	if err := os.Chmod(filepath.Join(seed, ".git"), 0o700); err != nil {
		t.Fatal(err)
	}
	v16RequireGitVersion(t, v16Git(t, gitPath, seed, "--version"), fixture.GitFixture.MinimumVersion)
	for _, file := range fixture.GitFixture.BaseFiles {
		v16WriteFile(t, seed, file.Path, file.Content)
	}
	v16Git(t, gitPath, seed, "add", "--all")
	v16GitCommit(t, gitPath, seed, fixture.GitFixture.AuthorName, fixture.GitFixture.AuthorEmail,
		time.Unix(fixture.GitFixture.BaseUnixTime, 0).UTC(), "v16 fixture base")
	commit := v16Git(t, gitPath, seed, "rev-parse", "HEAD")
	tree := v16Git(t, gitPath, seed, "rev-parse", "HEAD^{tree}")
	if commit != fixture.GitFixture.BaseOID || tree != fixture.GitFixture.BaseTreeOID {
		t.Fatalf("sealed base fixture mismatch: commit=%s/%s tree=%s/%s",
			commit, fixture.GitFixture.BaseOID, tree, fixture.GitFixture.BaseTreeOID)
	}
	if got := v16Git(t, gitPath, seed, "rev-parse", "--show-object-format"); got != fixture.GitFixture.ObjectFormat {
		t.Fatalf("Git object format=%s want=%s", got, fixture.GitFixture.ObjectFormat)
	}
	harness := &v16Harness{
		t: t, fixture: fixture, root: root, seed: seed,
		workspaceRoot: filepath.Join(root, "workspaces"), git: gitPath, writes: writes,
	}
	harness.configPath = v16WriteConfig(t, root, seed, harness.workspaceRoot, fixture.GitFixture.TargetRef)
	replaceTestConfigValue(t, harness.configPath, "[scheduler]\n", `[scheduler]
attest_test_claim_lease = "2s"
`)
	t.Cleanup(func() { harness.shutdown() })
	harness.build(t)
	return harness
}

func (harness *v16Harness) build(t *testing.T) {
	t.Helper()
	var workspaceAgent *v16WorkspaceAgent
	runtime, err := Build(context.Background(), Options{
		ConfigPath: harness.configPath, Version: "v16-real-git-sqlite-e2e",
		Listener: &runtimeEdgeFailingListener{err: errors.New("v16_test.listener_unused")},
		AgentFactory: func(_ config.Snapshot, clock application.Clock) (AgentAdapter, error) {
			workspaceAgent = &v16WorkspaceAgent{
				now: clock.Now, writes: harness.writes, launches: &harness.launches,
				requests: make(map[goal.ExecutionRef]ports.AgentLaunchRequest), allowReviews: true,
			}
			return workspaceAgent, nil
		},
	})
	if err != nil {
		t.Fatalf("build V16 composition: %v", err)
	}
	harness.runtime = runtime
	harness.installPassingTestAttestor(t, runtime, workspaceAgent)
	harness.access = testRuntimeAccess(t, runtime)
}

func (harness *v16Harness) installPassingTestAttestor(
	t *testing.T,
	runtime *Runtime,
	agent *v16WorkspaceAgent,
) {
	t.Helper()
	agent.mu.Lock()
	resolver := agent.resolver
	agent.mu.Unlock()
	workspace, ok := resolver.(*gitlocal.Adapter)
	if !ok || workspace == nil {
		t.Fatalf("V16 harness workspace resolver=%T", resolver)
	}
	clock := local.Clock{}
	budgetPolicy, err := buildBudgetPolicy(runtime.config, clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	capabilities, err := runtime.agent.Capabilities(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	policy := legacyTestAttestationPolicy()
	// V16 remains a legacy harness: Build opened the real Git/SQLite/CAS
	// adapters, then only its application wiring receives this canonical fake.
	// Production Bubblewrap ownership stays in the dedicated V17 E2E.
	orchestrator, err := newBuildOrchestrator(
		buildSetup{snapshot: runtime.config, clock: clock, budgetPolicy: budgetPolicy},
		runtime.repository, runtime.artifacts, runtime.agent, nil, capabilities, workspace,
		buildTestAttestorComposition{attestor: legacyPassingTestAttestor{}, policy: policy},
	)
	if err != nil {
		t.Fatalf("install V16 canonical test attestor: %v", err)
	}
	runtime.orchestrator = orchestrator
}

func (harness *v16Harness) shutdown() {
	if harness.runtime == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = harness.runtime.Shutdown(ctx)
	harness.runtime = nil
}

func (harness *v16Harness) restart(t *testing.T) {
	t.Helper()
	harness.shutdown()
	harness.build(t)
}

func (harness *v16Harness) restartAfterLease(t *testing.T) {
	t.Helper()
	harness.shutdown()
	time.Sleep(80 * time.Millisecond)
	harness.build(t)
}

func (harness *v16Harness) submit(
	t *testing.T,
	access application.Access,
	requestRef, objective string,
	writeSet []string,
) goal.GoalRef {
	t.Helper()
	result, err := harness.runtime.Orchestrator().Submit(context.Background(), access, application.SubmitRequest{
		RequestRef: requestRef, Statement: objective, Confirm: true,
		Plan: &application.PlanSpec{
			Phases: []application.PhaseSpec{{
				Ref: "phase-instance:v16-workspace", Key: "phase:v16-workspace",
				TemplateRef: "phase-template:v16-workspace",
			}},
			WorkItems: []application.WorkItemSpec{{
				Key: "writer", Objective: objective, Phase: "phase:v16-workspace", Role: "role:writer",
				WriteSet: append([]string(nil), writeSet...),
				RequiredTests: []application.RequiredTestSpec{{
					Ref: "required-test:v16-go", ToolRef: "tool:go",
					Arguments: []string{"test", "./..."}, WorkingDirectory: ".",
				}},
				OutputContract: goal.OutputContractEvidenceBundle,
			}},
		},
	})
	if err != nil {
		t.Fatalf("submit %s: %v", objective, err)
	}
	return result.Record.Goal.Ref()
}

func (harness *v16Harness) process(t *testing.T, expected ...application.ActionKind) {
	t.Helper()
	for _, want := range expected {
		deadline := time.Now().Add(5 * time.Second)
		for {
			result, err := harness.runtime.Orchestrator().ProcessNext(context.Background(), "worker:v16-e2e")
			if err != nil {
				t.Fatalf("process %s: result=%+v err=%v cause=%v", want, result, err, errors.Unwrap(err))
			}
			if result.Processed {
				if result.Action != want {
					t.Fatalf("processed action=%s want=%s", result.Action, want)
				}
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("action %s not available", want)
			}
			time.Sleep(5 * time.Millisecond)
		}
	}
}

func (harness *v16Harness) get(t *testing.T, access application.Access, ref goal.GoalRef) application.GoalRecord {
	t.Helper()
	record, err := harness.runtime.Orchestrator().GetGoal(context.Background(), access, ref)
	if err != nil {
		t.Fatalf("get Goal %s: %v cause=%v", ref.String(), err, errors.Unwrap(err))
	}
	return record
}

func (harness *v16Harness) pending(t *testing.T, access application.Access) []application.PendingChange {
	t.Helper()
	result, err := harness.runtime.Orchestrator().ListPendingChanges(context.Background(), access,
		application.ListPendingChangesRequest{Limit: 100})
	if err != nil {
		t.Fatalf("list pending: %v", err)
	}
	return result.Changes
}

func (harness *v16Harness) claim(t *testing.T, want application.ActionKind) application.ActionClaim {
	t.Helper()
	policy, err := buildBudgetPolicy(harness.runtime.config, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	claim, found, err := harness.runtime.repository.ClaimNextAction(context.Background(), application.ClaimRequest{
		WorkerRef:     "worker:v16-crash-injector",
		Token:         "claim-token:v16:" + string(want) + ":" + strconv.FormatInt(time.Now().UnixNano(), 10),
		LeaseDuration: 50 * time.Millisecond, Capabilities: v16AgentCapabilities(), BudgetPolicy: policy,
	})
	if err != nil || !found || claim.Action.Kind != want {
		t.Fatalf("claim=%+v found=%v want=%s err=%v", claim, found, want, err)
	}
	return claim
}

func (harness *v16Harness) workspacePath(ref ports.ExecutionWorkspaceRef) string {
	return filepath.Join(harness.workspaceRoot, "workspaces", v16OpaqueDigest(ref.String()))
}

func (harness *v16Harness) driveToIntegrated(t *testing.T, ref goal.GoalRef, requestRef string) {
	t.Helper()
	record := harness.get(t, harness.access, ref)
	if len(record.ChangeSets) == 0 || record.Executions[0].State != application.ExecutionAwaitingIntegration {
		harness.driveToAttested(t, record)
		record = harness.get(t, harness.access, ref)
	}
	harness.driveReviews(t, ref)
	record = harness.get(t, harness.access, ref)
	if len(record.IntegrationReceipts) == 0 {
		if _, err := harness.runtime.Orchestrator().IntegrateChange(context.Background(), harness.access,
			application.IntegrateChangeRequest{
				RequestRef: requestRef, GoalRef: ref, ChangeRef: record.ChangeSets[0].Ref,
				ExpectedTargetOID: record.WorkspaceBindings[0].BaseOID,
			}); err != nil {
			t.Fatal(err)
		}
		harness.process(t, application.ActionIntegrateChange)
	}
}

func (harness *v16Harness) driveReviews(t *testing.T, ref goal.GoalRef) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		record := harness.get(t, harness.access, ref)
		if len(record.Reviews) == 2 {
			return
		}
		result, err := harness.runtime.Orchestrator().ProcessNext(context.Background(), "worker:v16-reviews")
		if err != nil || !result.Processed ||
			(result.Action != application.ActionLaunchAgent && result.Action != application.ActionObserveAgent) {
			t.Fatalf("drive V16 reviews result=%+v err=%v", result, err)
		}
		if time.Now().After(deadline) {
			t.Fatal("drive V16 reviews timed out")
		}
	}
}

func (harness *v16Harness) driveToAttested(t *testing.T, record application.GoalRecord) {
	t.Helper()
	if len(record.Executions) != 1 {
		t.Fatalf("attestation driver executions=%d", len(record.Executions))
	}
	switch record.Executions[0].State {
	case application.ExecutionQueued:
		harness.process(t, application.ActionPrepareWorkspace, application.ActionLaunchAgent,
			application.ActionObserveAgent, application.ActionCommitChange, application.ActionAttestTest)
	case application.ExecutionAwaitingCommit:
		harness.process(t, application.ActionCommitChange, application.ActionAttestTest)
	case application.ExecutionAwaitingAttestation:
		harness.process(t, application.ActionAttestTest)
	case application.ExecutionAwaitingIntegration:
		return
	default:
		t.Fatalf("attestation driver unexpected execution state=%s", record.Executions[0].State)
	}
}

type legacyPassingTestAttestor struct{}

func (legacyPassingTestAttestor) Attest(
	ctx context.Context,
	run ports.TestAttestationRun,
) (ports.TestAttestationResult, error) {
	request := run.Request
	if err := ctx.Err(); err != nil {
		return ports.TestAttestationResult{}, err
	}
	if err := ports.ValidateTestAttestationRequest(request); err != nil {
		return ports.TestAttestationResult{}, err
	}
	outcomes := make([]ports.RequiredTestOutcome, len(request.RequiredTests))
	for index, spec := range request.RequiredTests {
		outcomes[index] = ports.RequiredTestOutcome{
			RequiredTestRef: spec.Ref(), ExitCode: 0,
			OutputDigest: legacyAttestationDigest("pass:" + spec.Ref().String()),
		}
	}
	manifest, err := ports.BuildTestSubjectManifest(request.Subject)
	if err != nil {
		return ports.TestAttestationResult{}, err
	}
	report, err := ports.BuildTestAttestationReport(ports.TestAttestationReportInput{
		SubjectDigest: request.SubjectDigest, Verdict: ports.TestAttestationPassed, Tests: outcomes,
		AttestorRef: "test-attestor:legacy-harness",
		PolicyRef:   request.Subject.PolicyRef, PolicyDigest: request.Subject.PolicyDigest,
	})
	if err != nil {
		return ports.TestAttestationResult{}, err
	}
	receiptRef, err := ports.TestAttestationReceiptRef(report)
	if err != nil {
		return ports.TestAttestationResult{}, err
	}
	result := ports.TestAttestationResult{
		Subject: request.Subject, SubjectDigest: request.SubjectDigest,
		Verdict: ports.TestAttestationPassed, Manifest: manifest, Report: report, Tests: outcomes,
		AttestorRef: "test-attestor:legacy-harness",
		ReceiptRef:  receiptRef,
		PolicyRef:   request.Subject.PolicyRef, PolicyDigest: request.Subject.PolicyDigest,
		StartedAt: request.RequestedAt, FinishedAt: request.RequestedAt,
	}
	if err := ports.ValidateTestAttestationResult(request, result); err != nil {
		return ports.TestAttestationResult{}, err
	}
	return result, nil
}

func legacyTestAttestationPolicy() application.TestAttestationPolicy {
	const ref = "test-attestation-policy:legacy-harness"
	return application.TestAttestationPolicy{Ref: ref, Digest: legacyAttestationDigest(ref)}
}

func legacyAttestationDigest(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}
