package sqlite

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
	"orquesta/internal/review"
)

const sqliteV15PolicyHash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func sqliteRuntimeTestPolicy() application.BudgetPolicy {
	return sqliteTestBudgetPolicy(time.Unix(1, 0).UTC())
}

func sqliteTestBudgetPolicy(at time.Time) application.BudgetPolicy {
	limit := governance.ResourceVector{
		Tokens: 10_000, MoneyMicros: 10_000, Currency: governance.Currency("USD"),
		ActiveTimeNS: int64(time.Hour), ProcessSlots: 70, DiskBytes: 1 << 30,
	}
	envelope := func(ref, subject string, scope governance.BudgetScope) governance.BudgetEnvelope {
		return governance.BudgetEnvelope{
			Ref: ref, SubjectRef: subject, Scope: scope, Limit: limit, Revision: 1,
			PolicyHash: sqliteV15PolicyHash, CreatedAt: at.UTC(),
		}
	}
	return application.BudgetPolicy{
		DeploymentEnvelope: envelope(
			"budget-envelope:deployment:test:"+sqliteV15PolicyHash,
			"deployment:test", governance.BudgetScopeDeployment,
		),
		ProjectEnvelopeTemplate: envelope(
			"budget-envelope:project:template:"+sqliteV15PolicyHash,
			"project:template", governance.BudgetScopeProject,
		),
		GoalEnvelopeTemplate: envelope(
			"budget-envelope:goal:template:"+sqliteV15PolicyHash,
			"goal:template", governance.BudgetScopeGoal,
		),
		DefaultWorkItemDemand: governance.ResourceVector{
			Tokens: 100, MoneyMicros: 100, Currency: governance.Currency("USD"),
			ActiveTimeNS: int64(time.Minute), ProcessSlots: 1, DiskBytes: 1 << 20,
		},
		QuotaRetryDelay: time.Second, EffectApprovalTTL: time.Hour, PolicyHash: sqliteV15PolicyHash,
	}
}

func sqliteBudgetPolicyWithSlots(at time.Time, slots int64) application.BudgetPolicy {
	policy := sqliteTestBudgetPolicy(at)
	policy.DeploymentEnvelope.Limit.ProcessSlots = slots
	policy.ProjectEnvelopeTemplate.Limit.ProcessSlots = slots
	policy.GoalEnvelopeTemplate.Limit.ProcessSlots = slots
	return policy
}

type sqliteV15IDs struct {
	mu   sync.Mutex
	next uint64
}

func (ids *sqliteV15IDs) NewID(ctx context.Context, prefix string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	ids.mu.Lock()
	defer ids.mu.Unlock()
	ids.next++
	return fmt.Sprintf("%s:sqlite-v15:%d", prefix, ids.next), nil
}

type sqliteV15External struct {
	mu               sync.Mutex
	clock            *sqliteMembershipClock
	launches         map[string]ports.AgentLaunchReceipt
	launchRequests   map[goal.ExecutionRef]ports.AgentLaunchRequest
	content          map[goal.ArtifactRef]ports.ArtifactContent
	launchErr        error
	launchStart      chan struct{}
	launchGate       chan struct{}
	launchCalls      int
	stopCalls        int
	observationUsage governance.ResourceUsage
	reviewContent    []byte
	reviewVerdict    review.Verdict
}

type sqliteV15DefinitelyUnapplied struct{}

func (sqliteV15DefinitelyUnapplied) Error() string              { return "sqlite.v15.capacity" }
func (sqliteV15DefinitelyUnapplied) Temporary() bool            { return true }
func (sqliteV15DefinitelyUnapplied) DefinitelyNotApplied() bool { return true }

func newSQLiteV15External(clock *sqliteMembershipClock) *sqliteV15External {
	return &sqliteV15External{
		clock: clock, launches: make(map[string]ports.AgentLaunchReceipt),
		launchRequests:   make(map[goal.ExecutionRef]ports.AgentLaunchRequest),
		content:          make(map[goal.ArtifactRef]ports.ArtifactContent),
		observationUsage: governance.ResourceUsage{Quality: governance.UsageQualityUnknown},
	}
}

func (external *sqliteV15External) Capabilities(context.Context) (ports.AgentCapabilities, error) {
	return sqliteTestCapabilities(), nil
}

func (external *sqliteV15External) Launch(
	_ context.Context, request ports.AgentLaunchRequest,
) (ports.AgentLaunchReceipt, error) {
	external.mu.Lock()
	external.launchCalls++
	launchErr, start, gate := external.launchErr, external.launchStart, external.launchGate
	external.mu.Unlock()
	if start != nil {
		close(start)
	}
	if gate != nil {
		<-gate
	}
	if launchErr != nil {
		return ports.AgentLaunchReceipt{}, launchErr
	}
	external.mu.Lock()
	defer external.mu.Unlock()
	if receipt, found := external.launches[request.IdempotencyKey]; found {
		return receipt, nil
	}
	capabilities := sqliteTestCapabilities()
	receipt := ports.AgentLaunchReceipt{
		ExecutionRef: request.ExecutionRef, GoalRef: request.GoalRef, WorkItemRef: request.WorkItemRef,
		PlanGeneration: request.PlanGeneration, AppSpecGeneration: request.AppSpecGeneration,
		ExecutionAttempt: request.ExecutionAttempt, SpecHash: request.SpecHash,
		ProviderRef: capabilities.ProviderRef, ModelRef: capabilities.ModelRef, AgentRef: capabilities.AgentRef,
		ExternalRef: "external:" + request.ExecutionRef.String(), IdempotencyKey: request.IdempotencyKey,
		ReceiptRef: "provider-receipt:" + request.ExecutionRef.String(), AcceptedAt: external.clock.Now(),
	}
	external.launches[request.IdempotencyKey] = receipt
	external.launchRequests[request.ExecutionRef] = request
	return receipt, nil
}

func (external *sqliteV15External) Observe(
	_ context.Context, executionRef goal.ExecutionRef,
) (ports.AgentObservation, error) {
	external.mu.Lock()
	defer external.mu.Unlock()
	for _, receipt := range external.launches {
		if receipt.ExecutionRef == executionRef {
			request := external.launchRequests[executionRef]
			if request.ArtifactMediaType == review.AssessmentMediaType {
				if external.reviewContent != nil {
					return ports.AgentObservation{ExecutionRef: executionRef, SpecHash: receipt.SpecHash,
						Status: ports.AgentCompleted, MediaType: review.AssessmentMediaType,
						Content: append([]byte(nil), external.reviewContent...),
						Usage:   external.observationUsage, ObservedAt: external.clock.Now()}, nil
				}
				role := review.RolePrimary
				if strings.Contains(request.Objective, `"role":"adversarial"`) {
					role = review.RoleAdversarial
				}
				verdict := external.reviewVerdict
				if verdict == "" {
					verdict = review.VerdictApprove
				}
				var findings []review.Finding
				if verdict == review.VerdictChangesRequested {
					findings = []review.Finding{{
						Code: "sqlite.review.change", Severity: review.SeverityMedium,
						EvidenceRef: "evidence:sqlite-review-change",
					}}
				}
				payload, err := json.Marshal(review.Artifact{
					SchemaVersion: 1, SubjectDigest: reviewSubjectDigestFromObjective(request.Objective),
					Role: role, Verdict: verdict, Summary: "exact subject assessed", Findings: findings,
				})
				if err != nil {
					return ports.AgentObservation{}, err
				}
				return ports.AgentObservation{ExecutionRef: executionRef, SpecHash: receipt.SpecHash,
					Status: ports.AgentCompleted, MediaType: review.AssessmentMediaType, Content: payload,
					Usage: external.observationUsage, ObservedAt: external.clock.Now()}, nil
			}
			if request.ArtifactMediaType == council.ContributionMediaType {
				subjectDigest, reviewGate, role, err := sqliteCouncilEvidenceFromObjective(request.Objective)
				if err != nil {
					return ports.AgentObservation{}, err
				}
				payload, err := json.Marshal(council.Contribution{
					Schema: council.ContributionSchema, SubjectDigest: subjectDigest, Role: role,
					Body: "sqlite exact contribution", Ballot: council.BallotAccept,
					Evidence: []council.Evidence{{Kind: "review_gate", Ref: reviewGate}},
				})
				if err != nil {
					return ports.AgentObservation{}, err
				}
				return ports.AgentObservation{ExecutionRef: executionRef, SpecHash: receipt.SpecHash,
					Status: ports.AgentCompleted, MediaType: council.ContributionMediaType, Content: payload,
					Usage: external.observationUsage, ObservedAt: external.clock.Now()}, nil
			}
			return ports.AgentObservation{
				ExecutionRef: executionRef, SpecHash: receipt.SpecHash, Status: ports.AgentCompleted,
				MediaType: "text/plain", Content: []byte("v15 evidence"),
				Usage:      external.observationUsage,
				ObservedAt: external.clock.Now(),
			}, nil
		}
	}
	return ports.AgentObservation{}, fmt.Errorf("sqlite.v15.execution_missing")
}

func sqliteCouncilEvidenceFromObjective(objective string) (string, string, council.Role, error) {
	const prefix = "Assess exact approved V18 evidence "
	value := strings.TrimPrefix(objective, prefix)
	end := strings.Index(value, ". Return exactly")
	var evidence struct {
		SubjectDigest string `json:"subject_digest"`
		ReviewGate    string `json:"review_gate_digest"`
	}
	if end < 0 || json.Unmarshal([]byte(value[:end]), &evidence) != nil {
		return "", "", "", fmt.Errorf("sqlite.v19_council_evidence_invalid")
	}
	role := council.RoleProposer
	for _, candidate := range council.Roles() {
		if strings.Contains(objective, `"role":"`+string(candidate)+`"`) {
			role = candidate
		}
	}
	return evidence.SubjectDigest, evidence.ReviewGate, role, nil
}

func reviewSubjectDigestFromObjective(objective string) string {
	const evidencePrefix = "Review exact immutable evidence "
	if strings.HasPrefix(objective, evidencePrefix) {
		value := strings.TrimPrefix(objective, evidencePrefix)
		if end := strings.Index(value, ". Inspect with "); end >= 0 {
			var evidence struct {
				SubjectDigest string `json:"subject_digest"`
			}
			if json.Unmarshal([]byte(value[:end]), &evidence) == nil {
				return evidence.SubjectDigest
			}
		}
	}
	const prefix = "Review the exact immutable subject "
	value := strings.TrimPrefix(objective, prefix)
	if end := strings.IndexByte(value, ' '); end >= 0 {
		return value[:end]
	}
	return value
}

func (external *sqliteV15External) ControlCapabilities(context.Context) (ports.AgentControlCapabilities, error) {
	return ports.AgentControlCapabilities{CooperativeStop: true, ForcedStop: true}, nil
}

func (external *sqliteV15External) Stop(
	_ context.Context, request ports.AgentStopRequest,
) (ports.AgentStopReceipt, error) {
	external.mu.Lock()
	external.stopCalls++
	external.mu.Unlock()
	return ports.AgentStopReceipt{
		ExecutionRef: request.ExecutionRef, GoalRef: request.GoalRef, WorkItemRef: request.WorkItemRef,
		PlanGeneration: request.PlanGeneration, AppSpecGeneration: request.AppSpecGeneration,
		ExecutionAttempt: request.ExecutionAttempt, SpecHash: request.SpecHash,
		ProviderRef: request.ProviderRef, ModelRef: request.ModelRef, AgentRef: request.AgentRef,
		ExternalRef: request.ExternalRef, Mode: request.Mode, IdempotencyKey: request.IdempotencyKey,
		Status: ports.AgentStopped, ReceiptRef: "provider-stop:" + request.ExecutionRef.String(),
		ConfirmedAt: external.clock.Now(),
	}, nil
}

func (external *sqliteV15External) Put(
	_ context.Context, request ports.PutArtifactRequest,
) (ports.StoredArtifact, error) {
	digest := sha256.Sum256(request.Content)
	value := hex.EncodeToString(digest[:])
	ref, err := goal.NewArtifactRef("artifact:sha256:" + value)
	if err != nil {
		return ports.StoredArtifact{}, err
	}
	stored := ports.StoredArtifact{Ref: ref, Digest: value, MediaType: request.MediaType, Size: int64(len(request.Content))}
	external.mu.Lock()
	external.content[ref] = ports.ArtifactContent{
		Ref: ref, Digest: value, MediaType: request.MediaType,
		Size: int64(len(request.Content)), Content: append([]byte(nil), request.Content...),
	}
	external.mu.Unlock()
	return stored, nil
}

func (external *sqliteV15External) Get(
	_ context.Context, ref goal.ArtifactRef, _ int64,
) (ports.ArtifactContent, error) {
	external.mu.Lock()
	defer external.mu.Unlock()
	content, found := external.content[ref]
	if !found {
		return ports.ArtifactContent{}, fmt.Errorf("sqlite.v15.artifact_missing")
	}
	content.Content = append([]byte(nil), content.Content...)
	return content, nil
}

type sqliteV15System struct {
	repository   *Repository
	path         string
	clock        *sqliteMembershipClock
	orchestrator *application.Orchestrator
	access       application.Access
	project      goal.ProjectRef
	external     *sqliteV15External
	policy       application.BudgetPolicy
	ids          *sqliteV15IDs
}

func newSQLiteV15System(t *testing.T, slots int64) *sqliteV15System {
	t.Helper()
	repository, path := openTestRepository(t)
	clock := &sqliteMembershipClock{now: time.Date(2026, 7, 18, 8, 0, 0, 0, time.UTC)}
	repository.now = clock.Now
	project := mustRef(t, "project:v15", goal.NewProjectRef)
	owner := testPrincipal(t, "principal:v15-owner", "actor:v15-owner", identity.PrincipalKindHuman)
	provisionTestAccess(t, repository, owner, project, identity.RoleProjectOwner, clock.Now())
	access, err := application.NewAccess(owner, project)
	sqliteTestNoError(t, err)
	external := newSQLiteV15External(clock)
	policy := sqliteBudgetPolicyWithSlots(clock.Now(), slots)
	ids := &sqliteV15IDs{}
	orchestrator := newSQLiteV15Orchestrator(t, repository, clock, external, policy, ids)
	return &sqliteV15System{
		repository: repository, path: path, clock: clock, orchestrator: orchestrator,
		access: access, project: project, external: external, policy: policy, ids: ids,
	}
}

func openSQLiteV15Repository(t *testing.T, path string, now func() time.Time) *Repository {
	t.Helper()
	return openFastTestRepository(t, Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 8, Now: now,
	})
}

func newSQLiteV15Orchestrator(t *testing.T, repository *Repository, clock *sqliteMembershipClock, external *sqliteV15External, policy application.BudgetPolicy, ids *sqliteV15IDs) *application.Orchestrator {
	return newSQLiteV15OrchestratorWithState(t, repository, repository, clock, external, policy, ids)
}

func newSQLiteV15OrchestratorWithState(
	t *testing.T,
	state application.StateRepository,
	repository *Repository,
	clock *sqliteMembershipClock,
	external *sqliteV15External,
	policy application.BudgetPolicy,
	ids *sqliteV15IDs,
) *application.Orchestrator {
	t.Helper()
	orchestrator, err := application.New(application.Dependencies{
		State: state, Access: repository, Launcher: external, Observer: external,
		Controller: external, Artifacts: external, Clock: clock, IDs: ids,
		MaxOutputBytes: 1024, MaxMailboxEnvelopeBytes: 64 << 10,
		MaxExecutionAttempts: 3, MaxChildrenPerParent: 6, ClaimLease: time.Minute,
		DirectorLeaseDuration: 30 * time.Second, EffectApprovalTTL: policy.EffectApprovalTTL,
		BudgetPolicy: policy, ObservationDelay: time.Second, ExecutionTimeout: time.Hour,
		AgentCapabilities: sqliteTestCapabilities(),
	})
	sqliteTestNoError(t, err)
	return orchestrator
}

func (system *sqliteV15System) submit(t *testing.T, ref string) application.SubmitResult {
	t.Helper()
	result, err := system.orchestrator.Submit(context.Background(), system.access, application.SubmitRequest{
		RequestRef: ref, Statement: "produce governed evidence " + ref, Confirm: true,
	})
	if err != nil {
		t.Fatalf("submit %s: %v", ref, err)
	}
	return result
}

func prepareSQLiteV15Launch(t *testing.T, system *sqliteV15System, claim application.ActionClaim) {
	t.Helper()
	record, err := system.repository.GetGoal(context.Background(), claim.Action.GoalRef)
	sqliteTestNoError(t, err)
	item, found := record.Goal.WorkItem(claim.Action.WorkItemRef)
	if !found {
		t.Fatal("claimed WorkItem missing")
	}
	started, err := record.Goal.StartWorkItem(
		record.Goal.Revision(), item.Revision(), item.Ref(), claim.Action.ExecutionRef, system.clock.Now(),
	)
	sqliteTestNoError(t, err)
	execution, found := sqliteExecutionByRef(record.Executions, claim.Action.ExecutionRef)
	if !found {
		t.Fatal("claimed execution missing")
	}
	execution.State, execution.BudgetReservationRef, execution.EffectIntentRef =
		application.ExecutionDispatching, claim.BudgetReservationRef, claim.Action.EffectIntentRef
	state := application.LaunchPreparedState{
		Claim: claim, ExpectedGoalRevision: record.Goal.Revision(), Goal: started, Execution: execution,
		OperationAt: system.clock.Now(), Event: application.EventRecord{
			Ref: "event:dispatching:" + claim.Token, Kind: "execution.dispatching", GoalRef: claim.Action.GoalRef,
			WorkItemRef: claim.Action.WorkItemRef, ExecutionRef: claim.Action.ExecutionRef, OccurredAt: system.clock.Now(),
		},
	}
	if err := validateLaunchPrepared(state); err != nil {
		t.Fatalf("invalid prepared fixture: %v", err)
	}
	if err := system.repository.RecordLaunchPrepared(context.Background(), state); err != nil {
		t.Fatal(err)
	}
}
