//go:build linux && v18_real_e2e

package bootstrap

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
	"orquesta/internal/review"
)

const v18SystemdInnerMarker = "V18_SYSTEMD_INNER_E2E_PASSED:"

var v18SystemdInnerUnit = flag.String(
	"v18-systemd-inner-unit", "", "internal V18 real-E2E systemd unit",
)

// Opt-in real Linux gate:
// go test -mod=vendor -tags=v18_real_e2e -count=1 -timeout=180s ./internal/bootstrap -run '^TestRealGitSQLiteFilesystemCASBubblewrapIndependentReviewsEndToEnd$'
func TestRealGitSQLiteFilesystemCASBubblewrapIndependentReviewsEndToEnd(t *testing.T) {
	if *v18SystemdInnerUnit == "" {
		runRealE2EInDelegatedUnit(t, realE2ESystemdSpec{
			gate: "V18_GATE_REAL_E2E", unitPrefix: "orquesta-v18-e2e-",
			testName:  "TestRealGitSQLiteFilesystemCASBubblewrapIndependentReviewsEndToEnd",
			innerFlag: "v18-systemd-inner-unit", marker: v18SystemdInnerMarker,
		})
		return
	}
	v17RequireDelegatedSystemdHarness(t, *v18SystemdInnerUnit)
	v17RequireRealAttestorHost(t)
	v18TestIndependentReviewsRealE2E(t)
	fmt.Fprintln(os.Stdout, v18SystemdInnerMarker+*v18SystemdInnerUnit)
}

func v18TestIndependentReviewsRealE2E(t *testing.T) {
	harness := newV18RealE2EHarness(t)
	defer harness.shutdown(t)

	goalRef := harness.submit(t)
	targetBefore := harness.target(t)
	harness.process(t, application.ActionPrepareWorkspace, application.ActionLaunchAgent,
		application.ActionObserveAgent, application.ActionCommitChange, application.ActionAttestTest)
	v18AssertBeforeReviewsFrontier(t, harness, harness.get(t, goalRef), targetBefore)
	harness.restart(t)
	v18AssertBeforeReviewsFrontier(t, harness, harness.get(t, goalRef), targetBefore)

	harness.process(t, application.ActionLaunchAgent, application.ActionLaunchAgent)
	harness.processUntilReviewRole(t, goalRef, review.RolePrimary)
	afterPrimary := harness.get(t, goalRef)
	v18AssertPrimaryFrontier(t, harness, afterPrimary, targetBefore)
	harness.restart(t)
	v18AssertPrimaryFrontier(t, harness, harness.get(t, goalRef), targetBefore)

	harness.process(t, application.ActionObserveAgent)
	afterReviews := harness.get(t, goalRef)
	v18AssertReviewPair(t, harness, afterReviews, targetBefore)
	v18AssertNoAutomaticIntegration(t, harness, afterReviews, targetBefore)
	harness.restart(t)
	afterReviews = harness.get(t, goalRef)
	v18AssertReviewPair(t, harness, afterReviews, targetBefore)
	v18AssertNoAutomaticIntegration(t, harness, afterReviews, targetBefore)

	change := v18One(t, afterReviews.ChangeSets, "change set", func(application.ChangeSet) bool { return true })
	admitted, err := harness.runtime.Orchestrator().IntegrateChange(
		context.Background(), harness.access, application.IntegrateChangeRequest{
			RequestRef: "request:v18-real-integrate", GoalRef: goalRef,
			ChangeRef: change.Ref, ExpectedTargetOID: targetBefore,
		})
	if err != nil || !admitted.Created || admitted.Action.Kind != application.ActionIntegrateChange {
		t.Fatalf("explicit V18 integration admission=%+v err=%v", admitted, err)
	}
	v18AssertIntegrationAdmissionFrontier(t, harness, harness.get(t, goalRef), targetBefore)
	harness.restart(t)
	v18AssertIntegrationAdmissionFrontier(t, harness, harness.get(t, goalRef), targetBefore)
	harness.process(t, application.ActionIntegrateChange)
	closed := harness.get(t, goalRef)
	v18AssertClosed(t, harness, closed, targetBefore)

	harness.restart(t)
	reopened := harness.get(t, goalRef)
	v18AssertClosed(t, harness, reopened, targetBefore)
	if result, err := harness.runtime.Orchestrator().ProcessNext(
		context.Background(), "worker:v18-recovery-idle",
	); err != nil || result.Processed {
		t.Fatalf("recovery repeated terminal work: result=%+v err=%v", result, err)
	}
	harness.shutdown(t)
	requireNoRealE2EOwnedProcesses(t)
}

func v18AssertBeforeReviewsFrontier(t *testing.T, harness *v18RealE2EHarness,
	record application.GoalRecord, targetBefore string,
) {
	t.Helper()
	launches, observations := harness.external.counts()
	if record.Goal.State() != goal.GoalStateRunning || harness.target(t) != targetBefore ||
		len(record.Executions) != 3 || len(record.Reviews) != 0 || len(record.IntegrationReceipts) != 0 ||
		launches != 1 || observations != 1 {
		t.Fatalf("V18 before-reviews frontier: state=%s executions=%d reviews=%d integrations=%d launches=%d observations=%d target=%s",
			record.Goal.State(), len(record.Executions), len(record.Reviews), len(record.IntegrationReceipts),
			launches, observations, harness.target(t))
	}
}

func (harness *v18RealE2EHarness) submit(t *testing.T) goal.GoalRef {
	t.Helper()
	result, err := harness.runtime.Orchestrator().Submit(context.Background(), harness.access,
		application.SubmitRequest{
			RequestRef: "request:v18-real-independent-reviews", Statement: "v18-author-candidate", Confirm: true,
			Plan: &application.PlanSpec{
				Phases: []application.PhaseSpec{{
					Ref: "phase-instance:v18-e2e", Key: "phase:v18-e2e", TemplateRef: "phase-template:v18-e2e",
				}},
				WorkItems: []application.WorkItemSpec{{
					Key: "author", Objective: "v18-author-candidate", Phase: "phase:v18-e2e", Role: "role:author",
					WriteSet: []string{"subject"}, RequiredTests: []application.RequiredTestSpec{{
						Ref: "required-test:v18-real-go", ToolRef: "tool:go",
						Arguments: []string{"test", "-buildvcs=false", "./...", "-count=1"}, WorkingDirectory: "subject",
					}}, OutputContract: goal.OutputContractEvidenceBundle,
				}},
			},
		})
	if err != nil {
		t.Fatalf("submit V18 E2E: %v", err)
	}
	return result.Record.Goal.Ref()
}

func (harness *v18RealE2EHarness) process(t *testing.T, expected ...application.ActionKind) {
	t.Helper()
	for _, want := range expected {
		deadline := time.Now().Add(45 * time.Second)
		for {
			ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
			result, err := harness.runtime.Orchestrator().ProcessNext(ctx, "worker:v18-real-e2e")
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

func (harness *v18RealE2EHarness) processUntilReviewRole(
	t *testing.T, goalRef goal.GoalRef, wanted review.Role,
) {
	t.Helper()
	for attempt := 0; attempt < 3; attempt++ {
		harness.process(t, application.ActionObserveAgent)
		for _, fact := range harness.get(t, goalRef).Reviews {
			if fact.Role == wanted {
				return
			}
		}
	}
	t.Fatalf("V18 review role %s was not observed", wanted)
}

func (harness *v18RealE2EHarness) get(t *testing.T, ref goal.GoalRef) application.GoalRecord {
	t.Helper()
	record, err := harness.runtime.Orchestrator().GetGoal(context.Background(), harness.access, ref)
	if err != nil {
		t.Fatalf("get V18 Goal %s: %v", ref.String(), err)
	}
	return record
}

func (harness *v18RealE2EHarness) target(t *testing.T) string {
	t.Helper()
	return v16Git(t, harness.git, harness.seed, "rev-parse", "refs/heads/main")
}

func v18AssertPrimaryFrontier(t *testing.T, harness *v18RealE2EHarness,
	record application.GoalRecord, targetBefore string,
) {
	t.Helper()
	if record.Goal.State() != goal.GoalStateRunning || harness.target(t) != targetBefore ||
		len(record.Executions) != 3 || len(record.Reviews) != 1 ||
		record.Reviews[0].Role != review.RolePrimary || len(record.IntegrationReceipts) != 0 {
		t.Fatalf("V18 primary frontier invalid: state=%s executions=%d reviews=%+v integrations=%d target=%s",
			record.Goal.State(), len(record.Executions), record.Reviews, len(record.IntegrationReceipts), harness.target(t))
	}
	launches, observations := harness.external.counts()
	if launches != 3 || observations != 2 {
		t.Fatalf("V18 live primary frontier launches=%d observations=%d want 3/2", launches, observations)
	}
}

func v18AssertReviewPair(t *testing.T, harness *v18RealE2EHarness,
	record application.GoalRecord, targetBefore string,
) {
	t.Helper()
	roles := map[review.Role]bool{}
	executions, launches, processes := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, fact := range record.Reviews {
		if roles[fact.Role] || executions[fact.ReviewerExecutionRef.String()] || launches[fact.LaunchReceiptRef] ||
			processes[fact.ExternalRef] || fact.Verdict != review.VerdictApprove {
			t.Fatalf("V18 duplicate/non-independent review: %+v", record.Reviews)
		}
		roles[fact.Role], executions[fact.ReviewerExecutionRef.String()] = true, true
		launches[fact.LaunchReceiptRef], processes[fact.ExternalRef] = true, true
	}
	if len(record.Reviews) != 2 || !roles[review.RolePrimary] || !roles[review.RoleAdversarial] ||
		record.Goal.State() != goal.GoalStateRunning || harness.target(t) != targetBefore {
		t.Fatalf("V18 review pair invalid: state=%s reviews=%+v target=%s",
			record.Goal.State(), record.Reviews, harness.target(t))
	}
	launchCount, observationCount := harness.external.counts()
	if launchCount != 3 || observationCount != 3 {
		t.Fatalf("V18 participant lifecycle launches=%d observations=%d want 3/3", launchCount, observationCount)
	}
}

func v18AssertNoAutomaticIntegration(t *testing.T, harness *v18RealE2EHarness,
	record application.GoalRecord, targetBefore string,
) {
	t.Helper()
	intents, consumptions := 0, 0
	for _, intent := range record.EffectIntents {
		if intent.ActionKind == application.ActionIntegrateChange {
			intents++
		}
	}
	for _, receipt := range record.ConsumptionReceipts {
		if receipt.Kind == application.ActionIntegrateChange {
			consumptions++
		}
	}
	status, err := harness.runtime.Orchestrator().Status(context.Background(), harness.access)
	if err != nil || status.PendingActions != 0 || intents != 0 || consumptions != 0 ||
		len(record.IntegrationReceipts) != 0 || harness.target(t) != targetBefore {
		t.Fatalf("V18 automatic integration facts: status=%+v intents=%d consumptions=%d receipts=%d target=%s err=%v",
			status, intents, consumptions, len(record.IntegrationReceipts), harness.target(t), err)
	}
}

func v18AssertIntegrationAdmissionFrontier(t *testing.T, harness *v18RealE2EHarness,
	record application.GoalRecord, targetBefore string,
) {
	t.Helper()
	intents := 0
	for _, intent := range record.EffectIntents {
		if intent.ActionKind == application.ActionIntegrateChange {
			intents++
		}
	}
	status, err := harness.runtime.Orchestrator().Status(context.Background(), harness.access)
	if err != nil || status.PendingActions != 1 || intents != 1 || len(record.IntegrationReceipts) != 0 ||
		harness.target(t) != targetBefore {
		t.Fatalf("V18 integration-admission frontier: status=%+v intents=%d receipts=%d target=%s err=%v",
			status, intents, len(record.IntegrationReceipts), harness.target(t), err)
	}
}

func v18AssertClosed(t *testing.T, harness *v18RealE2EHarness,
	record application.GoalRecord, targetBefore string,
) {
	t.Helper()
	targetAfter := harness.target(t)
	if record.Goal.State() != goal.GoalStateSucceeded || len(record.Reviews) != 2 ||
		len(record.IntegrationReceipts) != 1 || targetAfter == targetBefore ||
		record.IntegrationReceipts[0].Status != ports.IntegrationStatusIntegrated ||
		record.IntegrationReceipts[0].TargetBeforeOID != targetBefore ||
		record.IntegrationReceipts[0].TargetAfterOID != targetAfter {
		t.Fatalf("V18 closure invalid: state=%s reviews=%d integrations=%+v before=%s after=%s",
			record.Goal.State(), len(record.Reviews), record.IntegrationReceipts, targetBefore, targetAfter)
	}
	attestation := v18One(t, record.Attestations, "required tests attestation",
		func(value application.AttestationRecord) bool {
			return value.Kind == application.AttestationKindRequiredTests
		})
	if attestation.Verdict != application.AttestationVerdictPassed ||
		attestation.AttestorRef != "attestor:bubblewrap:local:v3" || len(attestation.Tests) != 1 ||
		attestation.Tests[0].ExitCode != 0 {
		t.Fatalf("V18 real Bubblewrap PASS invalid: %+v", attestation)
	}
	if len(record.Artifacts) != 5 {
		t.Fatalf("V18 durable CAS occurrences=%d want author+manifest+report+reviews", len(record.Artifacts))
	}
	for _, artifact := range record.Artifacts {
		content, err := harness.runtime.Orchestrator().GetArtifact(
			context.Background(), harness.access, record.Goal.Ref(), artifact.Stored.Ref,
		)
		if err != nil || content.Ref != artifact.Stored.Ref || content.Digest != artifact.Stored.Digest ||
			content.Size != artifact.Stored.Size || content.MediaType != artifact.Stored.MediaType {
			t.Fatalf("V18 filesystem CAS reread mismatch: artifact=%+v content=%+v err=%v", artifact, content, err)
		}
	}
}

func v18One[T any](t *testing.T, values []T, label string, match func(T) bool) T {
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
