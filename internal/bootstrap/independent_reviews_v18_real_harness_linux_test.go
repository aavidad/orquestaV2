//go:build linux && v18_real_e2e

package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"orquesta/internal/adapters/agent/codex"
	"orquesta/internal/application"
	"orquesta/internal/config"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
	"orquesta/internal/review"
)

type v18RealE2EHarness struct {
	root, seed, workspaceRoot, configPath, git string
	external                                   *v18ExternalAgent
	runtime                                    *Runtime
	access                                     application.Access
}

func newV18RealE2EHarness(t *testing.T) *v18RealE2EHarness {
	t.Helper()
	root := t.TempDir()
	v17RequirePrivateMode(t, root, 0o700)
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatalf("V18_GATE_REAL_E2E_GIT_UNAVAILABLE: %v", err)
	}
	gitPath, err = filepath.Abs(gitPath)
	v17NoError(t, err)
	seed := filepath.Join(root, "seed")
	v17NoError(t, os.Mkdir(seed, 0o700))
	v16Git(t, gitPath, seed, "init", "-b", "main")
	v17NoError(t, os.Chmod(filepath.Join(seed, ".git"), 0o700))
	v16WriteFile(t, seed, "README.md", "V18 independent review seed\n")
	v16Git(t, gitPath, seed, "add", "--all")
	v16GitCommit(t, gitPath, seed, "V18 Fixture", "v18@orquesta.local",
		time.Date(2026, 7, 23, 1, 0, 0, 0, time.UTC), "v18 review base")
	harness := &v18RealE2EHarness{
		root: root, seed: seed, workspaceRoot: filepath.Join(root, "workspaces"), git: gitPath,
		external: newV18ExternalAgent(v18PassingCandidateWrites()),
	}
	harness.configPath = v16WriteConfig(t, root, seed, harness.workspaceRoot, "refs/heads/main")
	configureBubblewrapTestAttestor(t, harness.configPath)
	harness.build(t)
	return harness
}

func (harness *v18RealE2EHarness) build(t *testing.T) {
	t.Helper()
	built, err := Build(context.Background(), Options{
		ConfigPath: harness.configPath, Version: "v18-real-independent-reviews-e2e",
		AgentFactory: func(_ config.Snapshot, clock application.Clock) (AgentAdapter, error) {
			return &v18PersistentAgentAdapter{external: harness.external, now: clock.Now}, nil
		},
	})
	if err != nil {
		t.Fatalf("build real V18 production composition: %v cause=%v", err, errors.Unwrap(err))
	}
	harness.runtime, harness.access = built, testRuntimeAccess(t, built)
}

func (harness *v18RealE2EHarness) shutdown(t *testing.T) {
	t.Helper()
	if harness.runtime == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := harness.runtime.Shutdown(ctx); err != nil {
		t.Errorf("shutdown V18 runtime: %v", err)
	}
	harness.runtime = nil
}

func (harness *v18RealE2EHarness) restart(t *testing.T) {
	t.Helper()
	harness.shutdown(t)
	harness.build(t)
}

type v18ExternalAgent struct {
	mu           sync.Mutex
	writes       v16Write
	requests     map[goal.ExecutionRef]ports.AgentLaunchRequest
	receipts     map[goal.ExecutionRef]ports.AgentLaunchReceipt
	observations map[goal.ExecutionRef]ports.AgentObservation
	launches     int
}

func newV18ExternalAgent(writes v16Write) *v18ExternalAgent {
	return &v18ExternalAgent{
		writes: writes, requests: make(map[goal.ExecutionRef]ports.AgentLaunchRequest),
		receipts:     make(map[goal.ExecutionRef]ports.AgentLaunchReceipt),
		observations: make(map[goal.ExecutionRef]ports.AgentObservation),
	}
}

type v18PersistentAgentAdapter struct {
	mu       sync.Mutex
	external *v18ExternalAgent
	now      func() time.Time
	resolver codex.WorkspacePathResolver
	closed   bool
}

func (agent *v18PersistentAgentAdapter) BindWorkspacePathResolver(resolver codex.WorkspacePathResolver) error {
	if resolver == nil {
		return errors.New("v18_test.workspace_resolver_required")
	}
	agent.mu.Lock()
	defer agent.mu.Unlock()
	if agent.closed || agent.resolver != nil {
		return errors.New("v18_test.workspace_resolver_rebound")
	}
	agent.resolver = resolver
	return nil
}

func (*v18PersistentAgentAdapter) Capabilities(context.Context) (ports.AgentCapabilities, error) {
	return ports.AgentCapabilities{
		ProviderRef: "provider:v18-fake", ModelRef: "model:v18-fake",
		AgentRef: "agent:v18-fake", Unrestricted: true,
	}, nil
}

func (agent *v18PersistentAgentAdapter) Launch(
	ctx context.Context, request ports.AgentLaunchRequest,
) (ports.AgentLaunchReceipt, error) {
	if err := ctx.Err(); err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	if err := ports.ValidateAgentLaunchRequest(request); err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	agent.mu.Lock()
	resolver, closed := agent.resolver, agent.closed
	agent.mu.Unlock()
	if closed || resolver == nil {
		return ports.AgentLaunchReceipt{}, errors.New("v18_test.agent_unavailable")
	}
	agent.external.mu.Lock()
	if previous, found := agent.external.requests[request.ExecutionRef]; found {
		receipt := agent.external.receipts[request.ExecutionRef]
		agent.external.mu.Unlock()
		if !reflect.DeepEqual(previous, request) {
			return ports.AgentLaunchReceipt{}, errors.New("v18_test.launch_conflict")
		}
		return receipt, nil
	}
	agent.external.mu.Unlock()

	if request.ArtifactMediaType != review.AssessmentMediaType {
		workspace, err := resolver.ResolveExecutionWorkspace(ctx, request.ExecutionWorkspaceRef)
		if err != nil {
			return ports.AgentLaunchReceipt{}, err
		}
		for relative, content := range agent.external.writes {
			target := filepath.Join(workspace, filepath.FromSlash(relative))
			if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
				return ports.AgentLaunchReceipt{}, err
			}
			if err := os.WriteFile(target, []byte(content), 0o600); err != nil {
				return ports.AgentLaunchReceipt{}, err
			}
		}
	}
	receipt := v18LaunchReceipt(request, agent.now())
	agent.external.mu.Lock()
	defer agent.external.mu.Unlock()
	if previous, found := agent.external.requests[request.ExecutionRef]; found {
		if !reflect.DeepEqual(previous, request) {
			return ports.AgentLaunchReceipt{}, errors.New("v18_test.launch_conflict")
		}
		return agent.external.receipts[request.ExecutionRef], nil
	}
	agent.external.requests[request.ExecutionRef], agent.external.receipts[request.ExecutionRef] = request, receipt
	agent.external.launches++
	return receipt, nil
}

func v18LaunchReceipt(request ports.AgentLaunchRequest, acceptedAt time.Time) ports.AgentLaunchReceipt {
	participant := "author"
	if request.ArtifactMediaType == review.AssessmentMediaType {
		_, role := v16ReviewEvidence(request.Objective)
		participant = string(role)
	}
	return ports.AgentLaunchReceipt{
		ExecutionRef: request.ExecutionRef, GoalRef: request.GoalRef, WorkItemRef: request.WorkItemRef,
		PlanGeneration: request.PlanGeneration, AppSpecGeneration: request.AppSpecGeneration,
		ExecutionAttempt: request.ExecutionAttempt, SpecHash: request.SpecHash,
		ProviderRef: "provider:v18-fake", ModelRef: "model:v18-fake",
		AgentRef:       "agent:v18-fake",
		ExternalRef:    "execution:v18-fake:" + participant + ":" + request.ExecutionRef.String(),
		IdempotencyKey: request.IdempotencyKey,
		ReceiptRef:     "receipt:v18-launch:" + participant + ":" + request.ExecutionRef.String(),
		AcceptedAt:     acceptedAt,
	}
}

func (agent *v18PersistentAgentAdapter) Observe(
	ctx context.Context, executionRef goal.ExecutionRef,
) (ports.AgentObservation, error) {
	if err := ctx.Err(); err != nil {
		return ports.AgentObservation{}, err
	}
	agent.mu.Lock()
	closed := agent.closed
	agent.mu.Unlock()
	if closed {
		return ports.AgentObservation{}, errors.New("v18_test.agent_unavailable")
	}
	agent.external.mu.Lock()
	request, found := agent.external.requests[executionRef]
	if previous, observed := agent.external.observations[executionRef]; observed {
		agent.external.mu.Unlock()
		return previous, nil
	}
	agent.external.mu.Unlock()
	if !found {
		return ports.AgentObservation{}, errors.New("v18_test.observation_not_found")
	}
	subjectDigest, role := "", review.Role("")
	if request.ArtifactMediaType == review.AssessmentMediaType {
		subjectDigest, role = v16ReviewEvidence(request.Objective)
		if role == review.RoleAdversarial && !agent.external.primaryObserved() {
			return ports.AgentObservation{
				ExecutionRef: executionRef, SpecHash: request.SpecHash, Status: ports.AgentRunning,
				Usage: unknownTestUsage(), ObservedAt: agent.now(),
			}, nil
		}
	}
	observation := ports.AgentObservation{
		ExecutionRef: executionRef, SpecHash: request.SpecHash, Status: ports.AgentCompleted,
		MediaType: "text/plain", Content: []byte("V18 author wrote the real Git workspace"),
		Usage: unknownTestUsage(), ObservedAt: agent.now(),
	}
	if request.ArtifactMediaType == review.AssessmentMediaType {
		payload, err := json.Marshal(review.Artifact{
			SchemaVersion: 1, SubjectDigest: subjectDigest, Role: role,
			Verdict: review.VerdictApprove, Summary: "Exact immutable V18 subject approved",
			Findings: []review.Finding{},
		})
		if err != nil {
			return ports.AgentObservation{}, err
		}
		observation.MediaType, observation.Content = review.AssessmentMediaType, payload
	}
	agent.external.mu.Lock()
	agent.external.observations[executionRef] = observation
	agent.external.mu.Unlock()
	return observation, nil
}

func (external *v18ExternalAgent) primaryObserved() bool {
	external.mu.Lock()
	defer external.mu.Unlock()
	for executionRef := range external.observations {
		request := external.requests[executionRef]
		_, role := v16ReviewEvidence(request.Objective)
		if request.ArtifactMediaType == review.AssessmentMediaType && role == review.RolePrimary {
			return true
		}
	}
	return false
}

func (agent *v18PersistentAgentAdapter) Shutdown(context.Context) error {
	agent.mu.Lock()
	agent.closed = true
	agent.mu.Unlock()
	return nil
}

func v18PassingCandidateWrites() v16Write {
	return v16Write{
		"subject/go.mod":       "module example.test/orquesta/v18candidate\n\ngo 1.25\n",
		"subject/candidate.go": "package candidate\n\nfunc Value() int { return 18 }\n",
		"subject/candidate_test.go": `package candidate

import "testing"

func TestCandidate(t *testing.T) {
	if Value() != 18 { t.Fatalf("Value=%d", Value()) }
}
`,
	}
}

func (external *v18ExternalAgent) counts() (int, int) {
	external.mu.Lock()
	defer external.mu.Unlock()
	return external.launches, len(external.observations)
}
