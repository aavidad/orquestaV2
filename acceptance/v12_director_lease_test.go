package acceptance_test

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

const v12FixturePath = "acceptance/fixtures/v12_director_lease.json"
const v12TrustedBaseGitCommitOID = "3108caa7e7f3f0a7b693e3027ef7cb3e56f462d9"
const v12ProductDeltaBaseGitCommitOID = "3108caa7e7f3f0a7b693e3027ef7cb3e56f462d9"
const v12ProductDeltaSealedGitCommitOID = "0000000000000000000000000000000000000000"

type v12Fixture struct {
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
	DeferredCapabilities           []v12DeferredCapability `json:"deferred_capabilities"`
	DeferredSurfaces               []string                `json:"deferred_surfaces"`
	Scenario                       v12Scenario             `json:"scenario"`
	Assertions                     []string                `json:"assertions"`
}

type v12DeferredCapability struct {
	ID                 string `json:"id"`
	Owner              string `json:"owner"`
	AcceptanceContract string `json:"acceptance_contract"`
}

type v12Scenario struct {
	BaseTime                string   `json:"base_time"`
	LeaseDuration           string   `json:"lease_duration"`
	RenewAfter              string   `json:"renew_after"`
	TakeoverAfter           string   `json:"takeover_after"`
	ProjectRef              string   `json:"project_ref"`
	GoalRef                 string   `json:"goal_ref"`
	HumanDirectorActorRef   string   `json:"human_director_actor_ref"`
	ServiceDirectorActorRef string   `json:"service_director_actor_ref"`
	DirectorPermission      string   `json:"director_permission"`
	AllowedRoles            []string `json:"allowed_roles"`
	DeniedRoles             []string `json:"denied_roles"`
	FirstFence              uint64   `json:"first_fence"`
	TakeoverFence           uint64   `json:"takeover_fence"`
	AdvisoryReason          string   `json:"advisory_reason"`
}

func TestAcceptanceV12DirectorLease(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v12Fixture](t, filepath.Join(repositoryRoot, filepath.FromSlash(v12FixturePath)))
	v12AssertFixtureHeader(t, repositoryRoot, fixture)

	t.Run("transferable_director_uses_one_permission_and_protocol", func(t *testing.T) {
		identitySource := v10ReadProductionGo(t, filepath.Join(repositoryRoot, "internal", "identity"))
		if !strings.Contains(identitySource, "PermissionGoalsDirect") ||
			!strings.Contains(identitySource, `"goals.direct"`) {
			t.Error("V12_RED identity lacks canonical goals.direct permission")
		}
		for _, role := range fixture.Scenario.AllowedRoles {
			if !strings.Contains(identitySource, role) {
				t.Errorf("V12_RED director policy lacks allowed role %q", role)
			}
		}
	})

	t.Run("lease_and_proposal_contracts_are_typed", func(t *testing.T) {
		orchestratorType := reflect.TypeOf((*application.Orchestrator)(nil))
		claim, claimOK := v12RequireUseCase(t, orchestratorType, "ClaimDirector", "ClaimDirectorRequest", "DirectorLeaseResult")
		renew, renewOK := v12RequireUseCase(t, orchestratorType, "RenewDirector", "RenewDirectorRequest", "DirectorLeaseResult")
		propose, proposeOK := v12RequireUseCase(t, orchestratorType, "ProposeDirectorPlan", "ProposeDirectorPlanRequest", "DirectorPlanResult")
		if claimOK {
			v12RequireFields(t, claim.Type.In(3), map[string]reflect.Type{
				"RequestRef": reflect.TypeOf(""), "GoalRef": reflect.TypeOf(goal.GoalRef{}),
			})
			v12RequireDirectorLeaseResult(t, claim.Type.Out(0))
		}
		if renewOK {
			v12RequireFields(t, renew.Type.In(3), map[string]reflect.Type{
				"RequestRef": reflect.TypeOf(""), "GoalRef": reflect.TypeOf(goal.GoalRef{}),
				"Token": reflect.TypeOf(""), "Fence": reflect.TypeOf(uint64(0)),
			})
			v12RequireDirectorLeaseResult(t, renew.Type.Out(0))
		}
		if proposeOK {
			v12RequireFields(t, propose.Type.In(3), map[string]reflect.Type{
				"RequestRef": reflect.TypeOf(""), "GoalRef": reflect.TypeOf(goal.GoalRef{}),
				"ExpectedGoalRevision":   reflect.TypeOf(goal.Revision(0)),
				"ExpectedPlanGeneration": reflect.TypeOf(goal.PlanGeneration(0)),
				"LeaseToken":             reflect.TypeOf(""), "LeaseFence": reflect.TypeOf(uint64(0)),
				"Reason": reflect.TypeOf(""), "Plan": reflect.TypeOf(application.PlanSpec{}),
			})
			v12RequireDirectorPlanResult(t, propose.Type.Out(0))
		}
	})

	t.Run("one_state_authority_owns_atomic_director_mutations", func(t *testing.T) {
		dependencies := reflect.TypeOf(application.Dependencies{})
		statePort := reflect.TypeOf((*application.StateRepository)(nil)).Elem()
		stateFields := 0
		for index := 0; index < dependencies.NumField(); index++ {
			field := dependencies.Field(index)
			if field.Type == statePort {
				stateFields++
			}
			if strings.Contains(strings.ToLower(field.Name), "director") && field.Name != "DirectorLeaseDuration" {
				t.Errorf("V12_RED dependencies add private Director authority %s", field.Name)
			}
		}
		if stateFields != 1 {
			t.Fatalf("application.Dependencies has %d StateRepository values, want one", stateFields)
		}
		leaseField, ok := dependencies.FieldByName("DirectorLeaseDuration")
		if !ok || leaseField.Type != reflect.TypeOf(time.Duration(0)) {
			t.Error("V12_RED dependencies lack typed DirectorLeaseDuration")
		}
		for _, method := range []string{"ClaimDirector", "RenewDirector", "ApplyDirectorPlan"} {
			if _, ok := statePort.MethodByName(method); !ok {
				t.Errorf("V12_RED StateRepository lacks atomic %s mutation", method)
			}
		}
	})

	t.Run("goal_and_existing_outbox_remain_authoritative", func(t *testing.T) {
		if _, ok := reflect.TypeOf(goal.Goal{}).MethodByName("ApplyPlan"); !ok {
			t.Fatal("Goal.ApplyPlan authority missing")
		}
		applicationSource := v10ReadProductionGo(t, filepath.Join(repositoryRoot, "internal", "application"))
		for _, required := range []string{"ProposeDirectorPlan", ".ApplyPlan(", "scheduleReady("} {
			if !strings.Contains(applicationSource, required) {
				t.Errorf("V12_RED proposal path does not reuse %q", required)
			}
		}
		for _, forbidden := range []string{"DirectorStore", "DirectorLoop", "DirectorScheduler", "DirectorQueue", "go func(", "time.NewTicker("} {
			if strings.Contains(applicationSource, forbidden) {
				t.Errorf("private Director engine found: %s", forbidden)
			}
		}
	})

	t.Run("sqlite_persists_lease_and_decision_without_second_database", func(t *testing.T) {
		migrationPath := filepath.Join(repositoryRoot, "internal", "adapters", "state", "sqlite", "migrations", "007_director_lease.sql")
		migration, err := os.ReadFile(migrationPath)
		if err != nil {
			t.Errorf("V12_RED SQLite migration 007 missing: %v", err)
			return
		}
		text := string(migration)
		for _, required := range []string{"director_leases", "director_decisions"} {
			if !strings.Contains(text, required) {
				t.Errorf("V12_RED migration lacks %s", required)
			}
		}
		for _, forbidden := range []string{"director_goals", "director_work_items", "director_outbox", "director_queue"} {
			if strings.Contains(text, forbidden) {
				t.Errorf("V12_RED migration creates private lifecycle table %s", forbidden)
			}
		}
	})

	t.Run("recoverable_content_stays_advisory", func(t *testing.T) {
		if !strings.Contains(fixture.Scenario.AdvisoryReason, "alias") ||
			!strings.Contains(fixture.Scenario.AdvisoryReason, "*.md") {
			t.Fatal("V12 scenario does not exercise recoverable alias/glob input")
		}
		applicationSource := strings.ToLower(v10ReadProductionGo(t, filepath.Join(repositoryRoot, "internal", "application")))
		for _, forbidden := range []string{"classifygarbage", "detailforbidden", "capacitylimited", "rejectcontent"} {
			if strings.Contains(applicationSource, forbidden) {
				t.Errorf("V12 Director path contains hard content rail %q", forbidden)
			}
		}
	})
}

func TestAcceptanceV12DirectorLeaseReceipt(t *testing.T) {
	evidenceAssertReceiptV3(t, evidenceRepositoryRoot(t), evidenceReceiptV3Expectation{
		Contract: "AC-V12-DIRECTOR-LEASE", FixturePath: v12FixturePath,
		ReceiptPath:       "product/evidence/v12_director_lease.json",
		ExecutedNotBefore: "2026-07-15T00:00:00Z", TrustedBaseGitCommitOID: v12TrustedBaseGitCommitOID,
	})
}

func TestV12CandidateSubjectsCoverCommittedDelta(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v12Fixture](t, filepath.Join(repositoryRoot, filepath.FromSlash(v12FixturePath)))
	if err := evidenceValidateSealedCommit(repositoryRoot, fixture.ProductDeltaBaseGitCommitOID, fixture.ProductDeltaSealedGitCommitOID); err != nil {
		t.Fatal(err)
	}
	output, err := evidenceGit(repositoryRoot, "diff", "--name-only", fixture.ProductDeltaBaseGitCommitOID, fixture.ProductDeltaSealedGitCommitOID, "--")
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
		t.Fatalf("V12 candidate subjects differ from sealed product delta:\nchanged=%v\nfixture=%v", changed, fixture.CandidateSubjects)
	}
}

func TestV12FixtureIsStrictJSON(t *testing.T) {
	fixture := evidenceDecodeStrictJSON[v12Fixture](t, filepath.Join(evidenceRepositoryRoot(t), filepath.FromSlash(v12FixturePath)))
	if fixture.ContractID != "AC-V12-DIRECTOR-LEASE" {
		t.Fatalf("unexpected V12 contract: %q", fixture.ContractID)
	}
}

func v12AssertFixtureHeader(t *testing.T, repositoryRoot string, fixture v12Fixture) {
	t.Helper()
	wantCapabilities := []string{"GOV-08", "GOV-09", "GOV-10", "ORC-24"}
	wantAllowed := []string{"platform_admin", "project_owner", "project_admin", "operator"}
	wantDenied := []string{"contributor", "reviewer", "viewer"}
	if fixture.SchemaVersion != 1 || fixture.ReceiptSchemaVersion != 3 ||
		fixture.ContractID != "AC-V12-DIRECTOR-LEASE" ||
		fixture.TrustedBaseGitCommitOID != v12TrustedBaseGitCommitOID ||
		fixture.ProductDeltaBaseGitCommitOID != v12ProductDeltaBaseGitCommitOID ||
		fixture.ProductDeltaSealedGitCommitOID != v12ProductDeltaSealedGitCommitOID ||
		fixture.OutputPath != "product/evidence/v12_director_lease.output.txt" ||
		fixture.ReceiptPath != "product/evidence/v12_director_lease.json" ||
		!reflect.DeepEqual(fixture.OwnedCapabilityIDs, wantCapabilities) ||
		!reflect.DeepEqual(fixture.Scenario.AllowedRoles, wantAllowed) ||
		!reflect.DeepEqual(fixture.Scenario.DeniedRoles, wantDenied) ||
		fixture.Scenario.DirectorPermission != "goals.direct" || fixture.Scenario.FirstFence != 1 ||
		fixture.Scenario.TakeoverFence != 2 || len(fixture.Assertions) != 14 {
		t.Fatalf("invalid V12 fixture header: %+v", fixture)
	}
	if _, err := time.Parse(time.RFC3339Nano, fixture.Scenario.BaseTime); err != nil {
		t.Fatalf("invalid V12 base time: %v", err)
	}
	lease := v12PositiveDuration(t, "lease_duration", fixture.Scenario.LeaseDuration)
	renew := v12PositiveDuration(t, "renew_after", fixture.Scenario.RenewAfter)
	takeover := v12PositiveDuration(t, "takeover_after", fixture.Scenario.TakeoverAfter)
	if renew >= lease || takeover <= lease {
		t.Fatalf("invalid V12 timing lease=%s renew=%s takeover=%s", lease, renew, takeover)
	}
	if fixture.Command != "sh -c '"+v12ValidationShellBody()+"'" ||
		!reflect.DeepEqual(fixture.ExecutionArgv, []string{"sh", "-c", v12ValidationShellBody()}) {
		t.Fatalf("invalid V12 command/argv: %q %#v", fixture.Command, fixture.ExecutionArgv)
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

func v12ValidationShellBody() string {
	return "go test -mod=vendor -count=1 . ./acceptance -run \"^(TestProductRoadmapIsExhaustiveAndCausal|TestProductRoadmapV12ScopeAndExecutableContract|TestV12EvidenceBelongsOnlyToDirectorCapabilities|TestV12AcceptanceCommandRunsDirectorConsumers|TestRebuildArchitecture|TestTraceabilityRebuildBugLessons|TestAcceptanceV12DirectorLease|TestV12CandidateSubjectsCoverCommittedDelta)$\"" +
		" && go test -mod=vendor -count=1 ./internal/goal ./internal/identity ./internal/application ./internal/config ./internal/adapters/state/sqlite ./internal/bootstrap ./cmd/orquesta"
}

func v12RequireUseCase(t *testing.T, owner reflect.Type, name, requestName, resultName string) (reflect.Method, bool) {
	t.Helper()
	method, ok := owner.MethodByName(name)
	if !ok {
		t.Errorf("V12_RED Orchestrator lacks %s", name)
		return reflect.Method{}, false
	}
	contextType := reflect.TypeOf((*context.Context)(nil)).Elem()
	errorType := reflect.TypeOf((*error)(nil)).Elem()
	if method.Type.NumIn() != 4 || method.Type.In(1) != contextType ||
		method.Type.In(2) != reflect.TypeOf(application.Access{}) || method.Type.In(3).Name() != requestName ||
		method.Type.NumOut() != 2 || method.Type.Out(0).Name() != resultName || method.Type.Out(1) != errorType {
		t.Errorf("V12_RED %s signature=%s", name, method.Type)
		return method, false
	}
	return method, true
}

func v12RequireDirectorLeaseResult(t *testing.T, result reflect.Type) {
	t.Helper()
	v12RequireFields(t, result, map[string]reflect.Type{"Changed": reflect.TypeOf(false)})
	field, ok := result.FieldByName("Lease")
	if !ok {
		t.Error("V12_RED DirectorLeaseResult lacks Lease")
		return
	}
	v12RequireFields(t, field.Type, map[string]reflect.Type{
		"GoalRef": reflect.TypeOf(goal.GoalRef{}), "PrincipalRef": reflect.TypeOf(identity.PrincipalRef{}),
		"Token": reflect.TypeOf(""), "Fence": reflect.TypeOf(uint64(0)), "LeaseUntil": reflect.TypeOf(time.Time{}),
	})
}

func v12RequireDirectorPlanResult(t *testing.T, result reflect.Type) {
	t.Helper()
	v12RequireFields(t, result, map[string]reflect.Type{
		"Record": reflect.TypeOf(application.GoalRecord{}), "Created": reflect.TypeOf(false),
	})
	field, ok := result.FieldByName("Decision")
	if !ok {
		t.Error("V12_RED DirectorPlanResult lacks Decision")
		return
	}
	v12RequireFields(t, field.Type, map[string]reflect.Type{
		"Ref": reflect.TypeOf(""), "RequestRef": reflect.TypeOf(""),
		"GoalRef": reflect.TypeOf(goal.GoalRef{}), "PrincipalRef": reflect.TypeOf(identity.PrincipalRef{}),
		"LeaseFence": reflect.TypeOf(uint64(0)), "SourceGoalRevision": reflect.TypeOf(goal.Revision(0)),
		"SourcePlanGeneration":  reflect.TypeOf(goal.PlanGeneration(0)),
		"AppliedGoalRevision":   reflect.TypeOf(goal.Revision(0)),
		"AppliedPlanGeneration": reflect.TypeOf(goal.PlanGeneration(0)),
		"Reason":                reflect.TypeOf(""), "DecidedAt": reflect.TypeOf(time.Time{}),
		"AuthorizationReceipt": reflect.TypeOf(identity.AuthorizationReceipt{}),
	})
}

func v12RequireFields(t *testing.T, owner reflect.Type, fields map[string]reflect.Type) {
	t.Helper()
	if owner.Kind() == reflect.Pointer {
		owner = owner.Elem()
	}
	if owner.Kind() != reflect.Struct {
		t.Errorf("V12_RED %s is not a struct", owner)
		return
	}
	for name, want := range fields {
		field, ok := owner.FieldByName(name)
		if !ok {
			t.Errorf("V12_RED %s lacks %s", owner.Name(), name)
			continue
		}
		if name == "Plan" && field.Type == reflect.PointerTo(want) {
			continue
		}
		if field.Type != want {
			t.Errorf("V12_RED %s.%s type=%s want=%s", owner.Name(), name, field.Type, want)
		}
	}
}

func v12PositiveDuration(t *testing.T, name, raw string) time.Duration {
	t.Helper()
	duration, err := time.ParseDuration(raw)
	if err != nil || duration <= 0 {
		t.Fatalf("invalid V12 %s %q: %v", name, raw, err)
	}
	return duration
}
