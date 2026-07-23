//go:build linux && v17_real_e2e

package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/config"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

const (
	v17SystemdInnerMarker = "V17_SYSTEMD_INNER_E2E_PASSED:"
)

var v17SystemdInnerUnit = flag.String(
	"v17-systemd-inner-unit", "", "internal V17 real-E2E systemd unit",
)

// Opt-in real Linux gate:
// go test -mod=vendor -tags=v17_real_e2e -count=1 -timeout=180s ./internal/bootstrap -run '^TestRealGitSQLiteCASBubblewrapAttestationEndToEnd$'
func TestRealGitSQLiteCASBubblewrapAttestationEndToEnd(t *testing.T) {
	if *v17SystemdInnerUnit == "" {
		v17RunRealE2EInDelegatedUnit(t)
		return
	}
	v17RequireDelegatedSystemdHarness(t, *v17SystemdInnerUnit)
	v17RequireRealAttestorHost(t)
	passed := t.Run("PASS_survives_restart_without_integrating_then_closes_explicitly", v17TestPassingAttestation)
	passed = t.Run("failed_test_keeps_change_pending_and_goal_open", v17TestFailedAttestation) && passed
	if passed {
		fmt.Fprintln(os.Stdout, v17SystemdInnerMarker+*v17SystemdInnerUnit)
	}
}

func v17RunRealE2EInDelegatedUnit(t *testing.T) {
	runRealE2EInDelegatedUnit(t, realE2ESystemdSpec{
		gate: "V17_GATE_REAL_E2E", unitPrefix: "orquesta-v17-e2e-",
		testName:  "TestRealGitSQLiteCASBubblewrapAttestationEndToEnd",
		innerFlag: "v17-systemd-inner-unit", marker: v17SystemdInnerMarker,
	})
}

func v17TestPassingAttestation(t *testing.T) {
	harness := newV17E2EHarness(t, "v17-pass-candidate", false)
	goalRef := harness.submit(t, "request:v17-real-pass")
	before := harness.target(t)
	harness.process(t, application.ActionPrepareWorkspace, application.ActionLaunchAgent,
		application.ActionObserveAgent, application.ActionCommitChange, application.ActionAttestTest)
	record := harness.get(t, goalRef)
	v17AssertPassingPending(t, harness, record, before)
	v17AssertAttestationCAS(t, harness, record, application.AttestationVerdictPassed)
	harness.process(t, application.ActionLaunchAgent, application.ActionLaunchAgent,
		application.ActionObserveAgent, application.ActionObserveAgent)
	record = harness.get(t, goalRef)
	v17AssertReviewCompatibility(t, harness, record, before)

	harness.restart(t)
	restarted := harness.get(t, goalRef)
	if !reflect.DeepEqual(restarted, record) {
		t.Fatal("terminal PASS facts changed across SQLite/CAS restart")
	}
	v17AssertAttestationCAS(t, harness, restarted, application.AttestationVerdictPassed)
	if result, err := harness.runtime.Orchestrator().ProcessNext(
		context.Background(), "worker:v17-no-rerun",
	); err != nil || result.Processed || harness.launches.Load() != 3 {
		t.Fatalf("restart reran terminal work: result=%+v launches=%d err=%v", result, harness.launches.Load(), err)
	}

	change := restarted.ChangeSets[0]
	admitted, err := harness.runtime.Orchestrator().IntegrateChange(
		context.Background(), harness.access, application.IntegrateChangeRequest{
			RequestRef: "request:v17-real-integrate", GoalRef: goalRef,
			ChangeRef: change.Ref, ExpectedTargetOID: before,
		})
	if err != nil || !admitted.Created {
		t.Fatalf("authorized integration admission=%+v err=%v", admitted, err)
	}
	harness.process(t, application.ActionIntegrateChange)
	v17AssertIntegrated(t, harness, harness.get(t, goalRef), before)
}

func v17AssertPassingPending(
	t *testing.T, harness *v17E2EHarness, record application.GoalRecord, targetBefore string,
) {
	t.Helper()
	item := v17FindOne(t, record.Goal.WorkItems(), "work item", func(goal.WorkItem) bool { return true })
	if record.Goal.State() != goal.GoalStateRunning || item.State() != goal.WorkItemStateRunning ||
		len(record.Executions) != 3 || record.Executions[0].State != application.ExecutionAwaitingIntegration ||
		len(record.WorkspaceBindings) != 1 || len(record.ChangeSets) != 1 || len(record.Artifacts) != 3 ||
		len(record.IntegrationReceipts) != 0 || harness.target(t) != targetBefore {
		t.Fatalf("PASS advanced without integration: state=%s item=%s executions=%+v bindings=%d changes=%d artifacts=%d integrations=%d attestations=%+v",
			record.Goal.State(), item.State(), record.Executions, len(record.WorkspaceBindings),
			len(record.ChangeSets), len(record.Artifacts), len(record.IntegrationReceipts), record.Attestations)
	}
	status, err := harness.runtime.Orchestrator().Status(context.Background(), harness.access)
	if err != nil || status.PendingActions != 2 {
		t.Fatalf("PASS review actions status=%+v err=%v", status, err)
	}
}

func v17AssertReviewCompatibility(
	t *testing.T, harness *v17E2EHarness, record application.GoalRecord, targetBefore string,
) {
	t.Helper()
	if len(record.Reviews) != 2 || len(record.Artifacts) != 5 || harness.target(t) != targetBefore {
		t.Fatalf("V17 review compatibility reviews=%+v artifacts=%d target=%s",
			record.Reviews, len(record.Artifacts), harness.target(t))
	}
	v17AssertNoIntegrationAction(t, harness, record)
}

func v17AssertIntegrated(
	t *testing.T, harness *v17E2EHarness, record application.GoalRecord, targetBefore string,
) {
	t.Helper()
	targetAfter := harness.target(t)
	if record.Goal.State() != goal.GoalStateSucceeded || len(record.Executions) != 3 ||
		record.Executions[0].State != application.ExecutionSucceeded || len(record.IntegrationReceipts) != 1 ||
		record.IntegrationReceipts[0].Status != ports.IntegrationStatusIntegrated ||
		targetAfter == targetBefore || record.IntegrationReceipts[0].TargetBeforeOID != targetBefore ||
		record.IntegrationReceipts[0].TargetAfterOID != targetAfter {
		t.Fatalf("authorized integration did not close exact target: state=%s execution=%+v before=%s after=%s receipts=%+v",
			record.Goal.State(), record.Executions, targetBefore, targetAfter, record.IntegrationReceipts)
	}
}

func v17TestFailedAttestation(t *testing.T) {
	harness := newV17E2EHarness(t, "v17-failed-candidate", true)
	goalRef := harness.submit(t, "request:v17-real-failed")
	before := harness.target(t)
	harness.process(t, application.ActionPrepareWorkspace, application.ActionLaunchAgent,
		application.ActionObserveAgent, application.ActionCommitChange, application.ActionAttestTest)
	record := harness.get(t, goalRef)
	item := v17FindOne(t, record.Goal.WorkItems(), "work item", func(goal.WorkItem) bool { return true })
	attestation := v17AssertAttestationCAS(t, harness, record, application.AttestationVerdictFailed)
	if record.Goal.State() != goal.GoalStateRunning || record.Goal.IsTerminal() ||
		item.State() != goal.WorkItemStateInterrupted || len(record.Executions) != 1 ||
		record.Executions[0].State != application.ExecutionFailed ||
		record.Executions[0].FailureCode != "test_attestor.required_tests_failed" ||
		len(attestation.Tests) != 1 || attestation.Tests[0].ExitCode == 0 || harness.target(t) != before {
		t.Fatalf("failed test integrated or closed: state=%s item=%s executions=%+v attestation=%+v",
			record.Goal.State(), item.State(), record.Executions, attestation)
	}
	v17AssertNoIntegrationAction(t, harness, record)
	_, err := harness.runtime.Orchestrator().IntegrateChange(context.Background(), harness.access,
		application.IntegrateChangeRequest{
			RequestRef: "request:v17-reject-failed", GoalRef: goalRef,
			ChangeRef: record.ChangeSets[0].Ref, ExpectedTargetOID: before,
		})
	if !application.IsStateError(err, application.StateConflict) || harness.target(t) != before {
		t.Fatalf("failed attestation admitted integration: err=%v target=%s", err, harness.target(t))
	}
}

func v17AssertNoIntegrationAction(t *testing.T, harness *v17E2EHarness, record application.GoalRecord) {
	t.Helper()
	integrationIntents, integrationConsumptions := 0, 0
	for _, intent := range record.EffectIntents {
		if intent.ActionKind == application.ActionIntegrateChange {
			integrationIntents++
		}
	}
	for _, receipt := range record.ConsumptionReceipts {
		if receipt.Kind == application.ActionIntegrateChange {
			integrationConsumptions++
		}
	}
	status, err := harness.runtime.Orchestrator().Status(context.Background(), harness.access)
	if err != nil || status.PendingActions != 0 || integrationIntents != 0 || integrationConsumptions != 0 ||
		len(record.IntegrationReceipts) != 0 {
		t.Fatalf("implicit integration facts: status=%+v intents=%d consumptions=%d receipts=%d err=%v",
			status, integrationIntents, integrationConsumptions, len(record.IntegrationReceipts), err)
	}
}

func v17AssertAttestationCAS(
	t *testing.T,
	harness *v17E2EHarness,
	record application.GoalRecord,
	want application.AttestationVerdict,
) application.AttestationRecord {
	t.Helper()
	attestation := v17FindOne(t, record.Attestations, "required_tests attestation",
		func(value application.AttestationRecord) bool {
			return value.Kind == application.AttestationKindRequiredTests
		})
	if attestation.Verdict != want {
		t.Fatalf("required test verdict=%s want=%s", attestation.Verdict, want)
	}
	item := v17FindOne(t, record.Goal.WorkItems(), "work item", func(goal.WorkItem) bool { return true })
	execution := v17FindOne(t, record.Executions, "execution", func(value application.ExecutionRecord) bool {
		return value.Ref == attestation.ExecutionRef
	})
	binding := v17FindOne(t, record.WorkspaceBindings, "workspace binding", func(value application.WorkspaceBinding) bool {
		return value.ExecutionRef == execution.Ref
	})
	change := v17FindOne(t, record.ChangeSets, "change", func(value application.ChangeSet) bool {
		return value.Ref == attestation.ChangeSetRef
	})
	subject := v17AttestationSubject(item, execution, binding, change, attestation)
	if attestation.SubjectDigest != ports.TestSubjectDigest(subject) ||
		attestation.WorkspaceBindingDigest != binding.Digest() ||
		attestation.ChangeSetDigest != change.Digest() ||
		attestation.RequiredTestsDigest != item.RequiredTestsDigest() {
		t.Fatalf("attestation not bound to exact causal subject: %+v", attestation)
	}
	v17AssertAttestationEffect(t, record, attestation, want)
	v17AssertCanonicalCASBytes(t, harness, record, subject, attestation, want)
	return attestation
}

func v17AttestationSubject(
	item goal.WorkItem,
	execution application.ExecutionRecord,
	binding application.WorkspaceBinding,
	change application.ChangeSet,
	attestation application.AttestationRecord,
) ports.TestSubject {
	return ports.TestSubject{
		GoalRef: change.GoalRef, WorkItemRef: item.Ref(), ExecutionRef: execution.Ref,
		ExecutionAttempt: execution.AttemptNo, PlanGeneration: execution.PlanGeneration,
		WorkItemGeneration: attestation.WorkItemGeneration,
		AppSpecGeneration:  execution.AppSpecGeneration, AppSpecHash: execution.SpecHash,
		WorkspaceRef: binding.Ref, WorkspaceBindingDigest: binding.Digest(),
		ChangeSetRef: change.Ref, ChangeSetDigest: change.Digest(), RepositoryRef: change.RepositoryRef,
		ObjectFormat: change.ObjectFormat, BaseOID: change.BaseOID, ParentOID: change.ParentOID,
		HeadOID: change.HeadOID, TreeOID: change.TreeOID, DiffDigest: change.DiffDigest,
		WriteSetDigest: change.WriteSetDigest, RequiredTestsDigest: item.RequiredTestsDigest(),
		PolicyRef: attestation.PolicyRef, PolicyDigest: attestation.PolicyDigest,
	}
}

func v17AssertAttestationEffect(
	t *testing.T,
	record application.GoalRecord,
	attestation application.AttestationRecord,
	want application.AttestationVerdict,
) {
	t.Helper()
	wantStatus := application.EffectStatusAttestedPassed
	if want == application.AttestationVerdictFailed {
		wantStatus = application.EffectStatusAttestedFailed
	}
	var intents, attempts, receipts int
	for _, intent := range record.EffectIntents {
		if intent.Ref == attestation.EffectIntentRef && intent.Kind == application.EffectKindAttestTest &&
			intent.ActionKind == application.ActionAttestTest && intent.TargetDigest == attestation.SubjectDigest {
			intents++
		}
	}
	for _, attempt := range record.EffectAttempts {
		if attempt.Ref == attestation.EffectAttemptRef && attempt.IntentRef == attestation.EffectIntentRef &&
			attempt.ActionFence == attestation.EffectFence {
			attempts++
		}
	}
	for _, receipt := range record.EffectReceipts {
		if receipt.Ref == attestation.EffectReceiptRef && receipt.IntentRef == attestation.EffectIntentRef &&
			receipt.AttemptRef == attestation.EffectAttemptRef && receipt.ActionFence == attestation.EffectFence &&
			receipt.Status == wantStatus && receipt.ExternalRef == attestation.ReceiptRef {
			receipts++
		}
	}
	if intents != 1 || attempts != 1 || receipts != 1 {
		t.Fatalf("attestation effect chain intents=%d attempts=%d receipts=%d", intents, attempts, receipts)
	}
}

func v17AssertCanonicalCASBytes(
	t *testing.T,
	harness *v17E2EHarness,
	record application.GoalRecord,
	subject ports.TestSubject,
	attestation application.AttestationRecord,
	want application.AttestationVerdict,
) {
	t.Helper()
	manifest, err := ports.BuildTestSubjectManifest(subject)
	v17NoError(t, err)
	verdict := ports.TestAttestationPassed
	if want == application.AttestationVerdictFailed {
		verdict = ports.TestAttestationFailed
	}
	report, err := ports.BuildTestAttestationReport(ports.TestAttestationReportInput{
		SubjectDigest: attestation.SubjectDigest, Verdict: verdict, Tests: attestation.Tests,
		AttestorRef: attestation.AttestorRef, PolicyRef: attestation.PolicyRef,
		PolicyDigest: attestation.PolicyDigest,
	})
	v17NoError(t, err)
	v17AssertCASArtifact(t, harness, record, attestation.ManifestArtifactRef,
		application.ArtifactKindTestSubjectManifest, manifest)
	v17AssertCASArtifact(t, harness, record, attestation.ReportArtifactRef,
		application.ArtifactKindTestReport, report)
}

func v17AssertCASArtifact(
	t *testing.T,
	harness *v17E2EHarness,
	record application.GoalRecord,
	ref goal.ArtifactRef,
	kind application.ArtifactKind,
	want ports.PutArtifactRequest,
) {
	t.Helper()
	var occurrence application.ArtifactRecord
	count := 0
	for _, candidate := range record.Artifacts {
		if candidate.Stored.Ref == ref && candidate.Kind == kind {
			occurrence, count = candidate, count+1
		}
	}
	content, err := harness.runtime.Orchestrator().GetArtifact(
		context.Background(), harness.access, record.Goal.Ref(), ref,
	)
	if err != nil || count != 1 || content.Ref != occurrence.Stored.Ref ||
		content.Digest != occurrence.Stored.Digest || content.Size != occurrence.Stored.Size ||
		content.MediaType != want.MediaType || !bytes.Equal(content.Content, want.Content) {
		t.Fatalf("CAS reread mismatch kind=%s count=%d content=%+v stored=%+v err=%v",
			kind, count, content, occurrence.Stored, err)
	}
}

type v17E2EHarness struct {
	root          string
	seed          string
	workspaceRoot string
	configPath    string
	git           string
	objective     string
	writes        map[string]v16Write
	launches      atomic.Int64
	runtime       *Runtime
	access        application.Access
}

func newV17E2EHarness(t *testing.T, objective string, failed bool) *v17E2EHarness {
	t.Helper()
	root := t.TempDir()
	v17RequirePrivateMode(t, root, 0o700)
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatalf("V17_GATE_REAL_E2E_GIT_UNAVAILABLE: %v", err)
	}
	gitPath, err = filepath.Abs(gitPath)
	v17NoError(t, err)
	seed := filepath.Join(root, "seed")
	if err := os.Mkdir(seed, 0o700); err != nil {
		t.Fatal(err)
	}
	v16Git(t, gitPath, seed, "init", "-b", "main")
	if err := os.Chmod(filepath.Join(seed, ".git"), 0o700); err != nil {
		t.Fatal(err)
	}
	v16WriteFile(t, seed, "README.md", "V17 real attestation seed\n")
	v16Git(t, gitPath, seed, "add", "--all")
	v16GitCommit(t, gitPath, seed, "V17 Fixture", "v17@orquesta.local",
		time.Date(2026, 7, 22, 10, 0, 0, 0, time.UTC), "v17 attestation base")
	harness := &v17E2EHarness{
		root: root, seed: seed, workspaceRoot: filepath.Join(root, "workspaces"), git: gitPath,
		objective: objective, writes: map[string]v16Write{objective: v17CandidateWrites(failed)},
	}
	harness.configPath = v17WriteE2EConfig(t, harness)
	harness.build(t)
	t.Cleanup(func() { harness.shutdown(t) })
	return harness
}

func v17CandidateWrites(failed bool) v16Write {
	testBody := `package candidate

import (
	"errors"
	"os"
	"syscall"
	"testing"
)

func TestCandidate(t *testing.T) {
	if Value() != 17 { t.Fatalf("Value=%d", Value()) }
	for resource, want := range map[int]uint64{
		syscall.RLIMIT_NOFILE: 8192, syscall.RLIMIT_FSIZE: 536870912,
		syscall.RLIMIT_AS: 4294967296, syscall.RLIMIT_CPU: 30,
	} {
		var got syscall.Rlimit
		if err := syscall.Getrlimit(resource, &got); err != nil || got.Cur != want || got.Max != want {
			t.Fatalf("resource=%d limit=%+v want=%d err=%v", resource, got, want, err)
		}
	}
	operations := []func() error{
		func() error { return os.WriteFile("new.go", []byte("x"), 0o600) },
		func() error { return os.Chmod("candidate.go", 0o600) },
		func() error { return os.Rename("candidate.go", "moved.go") },
	}
	for _, operation := range operations {
		if err := operation(); !errors.Is(err, syscall.EROFS) { t.Fatalf("subject mutation err=%v", err) }
	}
	if err := syscall.Unshare(syscall.CLONE_NEWUSER); !errors.Is(err, syscall.EPERM) && !errors.Is(err, syscall.EINVAL) {
		t.Fatalf("nested user namespace err=%v", err)
	}
}
`
	if failed {
		testBody = `package candidate

import "testing"

func TestCandidate(t *testing.T) { t.Fatal("intentional V17 failure") }
`
	}
	return v16Write{
		"subject/go.mod":            "module example.test/orquesta/v17candidate\n\ngo 1.25\n",
		"subject/candidate.go":      "package candidate\n\nfunc Value() int { return 17 }\n",
		"subject/candidate_test.go": testBody,
	}
}

func v17WriteE2EConfig(t *testing.T, harness *v17E2EHarness) string {
	t.Helper()
	path := v16WriteConfig(t, harness.root, harness.seed, harness.workspaceRoot, "refs/heads/main")
	configureBubblewrapTestAttestor(t, path)
	return path
}

func (harness *v17E2EHarness) build(t *testing.T) {
	t.Helper()
	built, err := Build(context.Background(), Options{
		ConfigPath: harness.configPath, Version: "v17-real-git-sqlite-cas-bubblewrap-e2e",
		AgentFactory: func(_ config.Snapshot, clock application.Clock) (AgentAdapter, error) {
			return &v16WorkspaceAgent{
				now: clock.Now, writes: harness.writes, launches: &harness.launches,
				requests: make(map[goal.ExecutionRef]ports.AgentLaunchRequest), allowReviews: true,
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("build real V17 production composition: %v cause=%v", err, errors.Unwrap(err))
	}
	harness.runtime = built
	harness.access = testRuntimeAccess(t, built)
}

func (harness *v17E2EHarness) shutdown(t *testing.T) {
	t.Helper()
	if harness.runtime == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := harness.runtime.Shutdown(ctx); err != nil {
		t.Errorf("shutdown V17 runtime: %v", err)
	}
	harness.runtime = nil
}

func (harness *v17E2EHarness) restart(t *testing.T) {
	t.Helper()
	harness.shutdown(t)
	harness.build(t)
}

func (harness *v17E2EHarness) submit(t *testing.T, requestRef string) goal.GoalRef {
	t.Helper()
	result, err := harness.runtime.Orchestrator().Submit(context.Background(), harness.access,
		application.SubmitRequest{
			RequestRef: requestRef, Statement: harness.objective, Confirm: true,
			Plan: &application.PlanSpec{
				Phases: []application.PhaseSpec{{
					Ref: "phase-instance:v17-e2e", Key: "phase:v17-e2e",
					TemplateRef: "phase-template:v17-e2e",
				}},
				WorkItems: []application.WorkItemSpec{{
					Key: "writer", Objective: harness.objective, Phase: "phase:v17-e2e", Role: "role:writer",
					WriteSet: []string{"subject"}, RequiredTests: []application.RequiredTestSpec{{
						Ref: "required-test:v17-real-go", ToolRef: "tool:go",
						Arguments: []string{"test", "-buildvcs=false", "./...", "-count=1"}, WorkingDirectory: "subject",
					}}, OutputContract: goal.OutputContractEvidenceBundle,
				}},
			},
		})
	if err != nil {
		t.Fatalf("submit V17 E2E: %v", err)
	}
	return result.Record.Goal.Ref()
}

func (harness *v17E2EHarness) process(t *testing.T, expected ...application.ActionKind) {
	t.Helper()
	for _, want := range expected {
		deadline := time.Now().Add(45 * time.Second)
		for {
			ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
			result, err := harness.runtime.Orchestrator().ProcessNext(ctx, "worker:v17-real-e2e")
			cancel()
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
				t.Fatalf("action %s unavailable", want)
			}
			time.Sleep(5 * time.Millisecond)
		}
	}
}

func (harness *v17E2EHarness) get(t *testing.T, ref goal.GoalRef) application.GoalRecord {
	t.Helper()
	record, err := harness.runtime.Orchestrator().GetGoal(context.Background(), harness.access, ref)
	if err != nil {
		t.Fatalf("get V17 Goal %s: %v", ref.String(), err)
	}
	return record
}

func (harness *v17E2EHarness) target(t *testing.T) string {
	t.Helper()
	return v16Git(t, harness.git, harness.seed, "rev-parse", "refs/heads/main")
}

func v17FindOne[T any](t *testing.T, values []T, label string, match func(T) bool) T {
	t.Helper()
	var found T
	count := 0
	for _, value := range values {
		if match(value) {
			found, count = value, count+1
		}
	}
	if count != 1 {
		t.Fatalf("%s count=%d in %+v", label, count, values)
	}
	return found
}
