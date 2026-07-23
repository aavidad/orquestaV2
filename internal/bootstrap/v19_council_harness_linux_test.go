//go:build linux && v19_real_e2e

package bootstrap

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/config"
	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
	"orquesta/internal/review"
)

const v19SystemdInnerMarker = "V19_COUNCIL_REAL_E2E_PASSED:"

var v19SystemdInnerUnit = flag.String("v19-systemd-inner-unit", "", "internal V19 Council systemd unit")

func v19RealGate(t *testing.T, name string) bool {
	t.Helper()
	if *v19SystemdInnerUnit == "" {
		runRealE2EInDelegatedUnit(t, realE2ESystemdSpec{gate: "V19_COUNCIL_REAL_E2E", unitPrefix: "orquesta-v19-council-", testName: name, innerFlag: "v19-systemd-inner-unit", marker: v19SystemdInnerMarker})
		return false
	}
	v17RequireDelegatedSystemdHarness(t, *v19SystemdInnerUnit)
	v17RequireRealAttestorHost(t)
	t.Cleanup(func() {
		if !t.Failed() {
			fmt.Fprintln(os.Stdout, v19SystemdInnerMarker+*v19SystemdInnerUnit)
		}
	})
	return true
}

type v19CouncilHarness struct {
	base    *v18RealE2EHarness
	policy  council.Policy
	ballots map[council.Role]council.Ballot
	agent   *v19CouncilAgent
}

func newV19CouncilHarness(t *testing.T, policy council.Policy, ballots map[council.Role]council.Ballot) *v19CouncilHarness {
	t.Helper()
	base := newV18RealE2EHarness(t)
	base.shutdown(t)
	harness := &v19CouncilHarness{base: base, policy: policy, ballots: ballots}
	harness.build(t)
	return harness
}

func (h *v19CouncilHarness) build(t *testing.T) {
	t.Helper()
	h.agent = newV19CouncilAgent(h.base.external, h.ballots)
	built, err := Build(context.Background(), Options{ConfigPath: h.base.configPath, Version: "v19-council-sqlite-fs-e2e",
		AgentFactory: func(_ config.Snapshot, clock application.Clock) (AgentAdapter, error) {
			h.agent.now = clock.Now
			h.agent.v18PersistentAgentAdapter.now = clock.Now
			return h.agent, nil
		}})
	if err != nil {
		t.Fatalf("build V19 Council runtime: %v", err)
	}
	h.base.runtime, h.base.access = built, testRuntimeAccess(t, built)
}

func (h *v19CouncilHarness) shutdown(t *testing.T) { h.base.shutdown(t) }
func (h *v19CouncilHarness) restart(t *testing.T)  { h.base.shutdown(t); h.build(t) }
func (h *v19CouncilHarness) get(t *testing.T, ref goal.GoalRef) application.GoalRecord {
	return h.base.get(t, ref)
}
func (h *v19CouncilHarness) target(t *testing.T) string { return h.base.target(t) }
func (h *v19CouncilHarness) process(t *testing.T, actions ...application.ActionKind) {
	h.base.process(t, actions...)
}

func (h *v19CouncilHarness) submit(t *testing.T, requestRef string) goal.GoalRef {
	t.Helper()
	result, err := h.base.runtime.Orchestrator().Submit(context.Background(), h.base.access, application.SubmitRequest{
		RequestRef: requestRef, Statement: "v19 council candidate", Confirm: true,
		Plan: &application.PlanSpec{
			Phases: []application.PhaseSpec{{Ref: "phase-instance:v19", Key: "phase:v19", TemplateRef: "phase-template:v19"}},
			WorkItems: []application.WorkItemSpec{{
				Key: "author", Objective: "v19 council candidate", Phase: "phase:v19", Role: "role:author", WriteSet: []string{"subject"}, CouncilPolicy: h.policy,
				RequiredTests:  []application.RequiredTestSpec{{Ref: "required-test:v19", ToolRef: "tool:go", Arguments: []string{"test", "-buildvcs=false", "./...", "-count=1"}, WorkingDirectory: "subject"}},
				OutputContract: goal.OutputContractEvidenceBundle,
			}},
		},
	})
	if err != nil {
		t.Fatalf("submit V19 Council: %v", err)
	}
	return result.Record.Goal.Ref()
}

func (h *v19CouncilHarness) driveApprovedGate(t *testing.T, ref goal.GoalRef) {
	t.Helper()
	h.process(t, application.ActionPrepareWorkspace, application.ActionLaunchAgent, application.ActionObserveAgent, application.ActionCommitChange, application.ActionAttestTest,
		application.ActionLaunchAgent, application.ActionLaunchAgent)
	h.base.processUntilReviewRole(t, ref, review.RolePrimary)
	h.process(t, application.ActionObserveAgent)
}

type v19CouncilAgent struct {
	*v18PersistentAgentAdapter
	now          func() time.Time
	mu           sync.Mutex
	ballots      map[council.Role]council.Ballot
	requests     map[goal.ExecutionRef]ports.AgentLaunchRequest
	receipts     map[goal.ExecutionRef]ports.AgentLaunchReceipt
	observations map[goal.ExecutionRef]ports.AgentObservation
}

func newV19CouncilAgent(external *v18ExternalAgent, ballots map[council.Role]council.Ballot) *v19CouncilAgent {
	return &v19CouncilAgent{v18PersistentAgentAdapter: &v18PersistentAgentAdapter{external: external}, ballots: ballots,
		requests: map[goal.ExecutionRef]ports.AgentLaunchRequest{}, receipts: map[goal.ExecutionRef]ports.AgentLaunchReceipt{}, observations: map[goal.ExecutionRef]ports.AgentObservation{}}
}

func (a *v19CouncilAgent) Launch(ctx context.Context, request ports.AgentLaunchRequest) (ports.AgentLaunchReceipt, error) {
	if request.ArtifactMediaType != council.ContributionMediaType {
		return a.v18PersistentAgentAdapter.Launch(ctx, request)
	}
	if err := ports.ValidateAgentLaunchRequest(request); err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if prior, ok := a.requests[request.ExecutionRef]; ok {
		if !reflect.DeepEqual(prior, request) {
			return ports.AgentLaunchReceipt{}, context.Canceled
		}
		return a.receipts[request.ExecutionRef], nil
	}
	receipt := ports.AgentLaunchReceipt{ExecutionRef: request.ExecutionRef, GoalRef: request.GoalRef, WorkItemRef: request.WorkItemRef,
		PlanGeneration: request.PlanGeneration, AppSpecGeneration: request.AppSpecGeneration, ExecutionAttempt: request.ExecutionAttempt,
		SpecHash: request.SpecHash, ProviderRef: "provider:v18-fake", ModelRef: "model:v18-fake", AgentRef: "agent:v18-fake",
		ExternalRef: "execution:v19-council:" + request.ExecutionRef.String(), ReceiptRef: "receipt:v19-council:" + request.ExecutionRef.String(), IdempotencyKey: request.IdempotencyKey, AcceptedAt: a.now().UTC()}
	a.requests[request.ExecutionRef], a.receipts[request.ExecutionRef] = request, receipt
	return receipt, nil
}

func (a *v19CouncilAgent) Observe(ctx context.Context, ref goal.ExecutionRef) (ports.AgentObservation, error) {
	a.mu.Lock()
	request, councilRequest := a.requests[ref]
	if observation, ok := a.observations[ref]; ok {
		a.mu.Unlock()
		return observation, nil
	}
	a.mu.Unlock()
	if !councilRequest {
		return a.v18PersistentAgentAdapter.Observe(ctx, ref)
	}
	var evidence struct {
		SubjectDigest string       `json:"subject_digest"`
		Role          council.Role `json:"role"`
	}
	start := len("Assess exact approved V18 evidence ")
	end := start
	for end < len(request.Objective) && request.Objective[end] != '.' {
		end++
	}
	if start >= end || json.Unmarshal([]byte(request.Objective[start:end]), &evidence) != nil {
		return ports.AgentObservation{}, context.Canceled
	}
	ballot := a.ballots[evidence.Role]
	refs := []council.Evidence{{Kind: "review_gate", Ref: "evidence:v19-review-gate"}}
	if ballot == council.BallotSecurityVeto {
		refs = append(refs, council.Evidence{Kind: "security_veto", Ref: "evidence:v19-security"})
	}
	content, err := json.Marshal(council.Contribution{Schema: council.ContributionSchema, SubjectDigest: evidence.SubjectDigest, Role: evidence.Role, Body: "V19 exact Council contribution", Ballot: ballot, Evidence: refs})
	if err != nil {
		return ports.AgentObservation{}, err
	}
	observation := ports.AgentObservation{ExecutionRef: ref, SpecHash: request.SpecHash, Status: ports.AgentCompleted, MediaType: council.ContributionMediaType, Content: content, Usage: unknownTestUsage(), ObservedAt: a.now().UTC()}
	a.mu.Lock()
	a.observations[ref] = observation
	a.mu.Unlock()
	return observation, nil
}
