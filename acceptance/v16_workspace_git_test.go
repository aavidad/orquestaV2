package acceptance_test

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const v16FixturePath = "acceptance/fixtures/v16_workspace_git.json"
const v16ContractBaseGitCommitOID = "3820df2ae89f1a217de1b14d5b88abf8e86c898b"

// P-stage placeholder. Closure replaces this with the immutable product commit
// before creating the source and evidence commits.
const v16ProductDeltaSealedGitCommitOID = "b48162b0433dd32b6369ee324358e5f87af325ad"

type v16Fixture struct {
	SchemaVersion                  int               `json:"schema_version"`
	ReceiptSchemaVersion           int               `json:"receipt_schema_version"`
	ContractID                     string            `json:"contract_id"`
	TrustedBaseGitCommitOID        string            `json:"trusted_base_git_commit_oid"`
	ProductDeltaBaseGitCommitOID   string            `json:"product_delta_base_git_commit_oid"`
	ProductDeltaSealedGitCommitOID string            `json:"product_delta_sealed_git_commit_oid"`
	Command                        string            `json:"command"`
	ExecutionArgv                  []string          `json:"execution_argv"`
	OutputPath                     string            `json:"output_path"`
	ReceiptPath                    string            `json:"receipt_path"`
	CandidateSubjects              []string          `json:"candidate_subjects"`
	OwnedCapabilityIDs             []string          `json:"owned_capability_ids"`
	DependencyVerticals            []string          `json:"dependency_verticals"`
	RequiredApplicationTypes       []v16RequiredType `json:"required_application_types"`
	RequiredOutboundPorts          []v16RequiredPort `json:"required_outbound_ports"`
	RequiredUseCases               []string          `json:"required_use_cases"`
	RequiredActions                []string          `json:"required_actions"`
	RequiredEffectKinds            []string          `json:"required_effect_kinds"`
	RequiredStatuses               []string          `json:"required_statuses"`
	ForbiddenAuthorities           []string          `json:"forbidden_private_authorities"`
	RequiredBehaviorTests          []string          `json:"required_behavior_tests"`
	GitFixture                     v16GitFixture     `json:"git_fixture"`
	Actors                         []v16Actor        `json:"actors"`
	Changes                        []v16Change       `json:"changes"`
	CrashFrontiers                 []string          `json:"crash_frontiers"`
	SecurityCases                  []string          `json:"security_cases"`
	PrivateLeakMarkers             []string          `json:"private_leak_markers"`
	DeferredSurfaces               []string          `json:"deferred_surfaces"`
}

type v16RequiredType struct {
	Name   string   `json:"name"`
	Fields []string `json:"fields"`
}

type v16RequiredPort struct {
	Name    string   `json:"name"`
	Methods []string `json:"methods"`
}

type v16GitFixture struct {
	MinimumVersion string            `json:"minimum_version"`
	ObjectFormat   string            `json:"object_format"`
	TargetRef      string            `json:"target_ref"`
	AuthorName     string            `json:"author_name"`
	AuthorEmail    string            `json:"author_email"`
	BaseOID        string            `json:"base_oid"`
	BaseTreeOID    string            `json:"base_tree_oid"`
	TargetOID      string            `json:"target_oid"`
	TargetTreeOID  string            `json:"target_tree_oid"`
	BaseUnixTime   int64             `json:"base_unix_time"`
	TargetUnixTime int64             `json:"target_unix_time"`
	BaseFiles      []v16FixtureWrite `json:"base_files"`
	TargetChanges  []v16FixtureWrite `json:"target_changes"`
}

type v16Actor struct {
	PrincipalRef string          `json:"principal_ref"`
	ActorRef     string          `json:"actor_ref"`
	Memberships  []v16Membership `json:"memberships"`
}

type v16Membership struct {
	ProjectRef    string `json:"project_ref"`
	RepositoryRef string `json:"repository_ref"`
	Role          string `json:"role"`
}

type v16Change struct {
	Name         string            `json:"name"`
	ExecutionRef string            `json:"execution_ref"`
	WriteSet     []string          `json:"write_set"`
	Writes       []v16FixtureWrite `json:"writes"`
	Expected     string            `json:"expected"`
}

type v16FixtureWrite struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func TestV16CandidateSubjectsCoverCommittedDelta(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v16Fixture](t, filepath.Join(repositoryRoot, filepath.FromSlash(v16FixturePath)))
	if err := evidenceValidateSealedCommit(
		repositoryRoot, fixture.ProductDeltaBaseGitCommitOID, fixture.ProductDeltaSealedGitCommitOID,
	); err != nil {
		t.Fatal(err)
	}
	output, err := evidenceGit(repositoryRoot, "diff", "--name-only",
		fixture.ProductDeltaBaseGitCommitOID, fixture.ProductDeltaSealedGitCommitOID, "--")
	if err != nil {
		t.Fatal(err)
	}
	var changed []string
	if value := strings.TrimSpace(string(output)); value != "" {
		changed = strings.Split(value, "\n")
	}
	sort.Strings(changed)
	if !reflect.DeepEqual(changed, fixture.CandidateSubjects) {
		t.Fatalf("V16 candidate subjects differ from sealed product delta:\nchanged=%v\nfixture=%v", changed, fixture.CandidateSubjects)
	}
	numstat, err := evidenceGit(repositoryRoot, "diff", "--numstat",
		fixture.ProductDeltaBaseGitCommitOID, fixture.ProductDeltaSealedGitCommitOID, "--")
	if err != nil {
		t.Fatal(err)
	}
	v16AssertSimplicityBudget(
		t, repositoryRoot, fixture.ProductDeltaBaseGitCommitOID, fixture.ProductDeltaSealedGitCommitOID, numstat,
	)
}

func TestAcceptanceV16WorkspaceGit(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v16Fixture](t, filepath.Join(repositoryRoot, filepath.FromSlash(v16FixturePath)))
	v16AssertFixture(t, repositoryRoot, fixture)

	applicationDirectory := filepath.Join(repositoryRoot, "internal", "application")
	portsDirectory := filepath.Join(repositoryRoot, "internal", "ports")

	t.Run("one_writer_and_two_outbound_ports", func(t *testing.T) {
		v16AssertOneWriterAndOutboundPorts(t, applicationDirectory, fixture)
	})

	t.Run("immutable_facts_are_causal_and_path_free", func(t *testing.T) {
		v16AssertImmutableFacts(t, applicationDirectory, portsDirectory, fixture)
	})

	t.Run("public_use_cases_admit_and_query_but_do_not_create_lifecycle", func(t *testing.T) {
		v16AssertPublicUseCases(t, applicationDirectory, fixture)
	})

	t.Run("sqlite_config_git_adapter_and_bootstrap_are_real", func(t *testing.T) {
		v16AssertConcreteAdapters(t, repositoryRoot)
	})

	t.Run("all_behavior_security_restart_e2e_and_race_gates_exist", func(t *testing.T) {
		v16AssertBehaviorTests(t, repositoryRoot, applicationDirectory, portsDirectory, fixture)
	})
}

func TestAcceptanceV16WorkspaceGitReceipt(t *testing.T) {
	evidenceAssertReceiptV3(t, evidenceRepositoryRoot(t), evidenceReceiptV3Expectation{
		Contract: "AC-V16-WORKSPACE-GIT", FixturePath: v16FixturePath,
		ReceiptPath:       "product/evidence/v16_workspace_git.json",
		ExecutedNotBefore: "2026-07-21T00:00:00Z", TrustedBaseGitCommitOID: v16ContractBaseGitCommitOID,
	})
}

func v16AssertFixture(t *testing.T, repositoryRoot string, fixture v16Fixture) {
	t.Helper()
	if fixture.SchemaVersion != 1 || fixture.ReceiptSchemaVersion != 3 ||
		fixture.ContractID != "AC-V16-WORKSPACE-GIT" ||
		fixture.TrustedBaseGitCommitOID != v16ContractBaseGitCommitOID ||
		fixture.ProductDeltaBaseGitCommitOID != v16ContractBaseGitCommitOID ||
		fixture.ProductDeltaSealedGitCommitOID != v16ProductDeltaSealedGitCommitOID ||
		fixture.OutputPath != "product/evidence/v16_workspace_git.output.txt" ||
		fixture.ReceiptPath != "product/evidence/v16_workspace_git.json" ||
		!reflect.DeepEqual(fixture.OwnedCapabilityIDs, []string{"STG-02", "STG-10", "EXT-10"}) ||
		!reflect.DeepEqual(fixture.DependencyVerticals, []string{
			"goal_dag_phases", "atomic_state_outbox", "identity_projects_rbac", "controls", "budgets_effects",
		}) ||
		!reflect.DeepEqual(fixture.RequiredUseCases, []string{"ListPendingChanges", "IntegrateChange"}) ||
		!reflect.DeepEqual(fixture.RequiredActions, []string{"prepare_workspace", "commit_change", "integrate_change"}) ||
		!reflect.DeepEqual(fixture.RequiredEffectKinds, fixture.RequiredActions) ||
		!reflect.DeepEqual(fixture.RequiredStatuses, []string{"clean", "conflicted", "stale", "integrated"}) {
		t.Fatalf("invalid V16 fixture identity: %+v", fixture)
	}
	if fixture.Command != "sh -c '"+v16ValidationShellBody()+"'" ||
		!reflect.DeepEqual(fixture.ExecutionArgv, []string{"sh", "-c", v16ValidationShellBody()}) {
		t.Fatalf("invalid V16 command/argv: %q %#v", fixture.Command, fixture.ExecutionArgv)
	}
	v16AssertValidationGates(t, v16ValidationShellBody())
	if err := evidenceValidateCandidateSubjects(
		fixture.CandidateSubjects, fixture.ReceiptPath, fixture.OutputPath,
	); err != nil {
		t.Fatal(err)
	}
	for _, relative := range fixture.CandidateSubjects {
		if _, err := os.Stat(filepath.Join(repositoryRoot, filepath.FromSlash(relative))); err != nil {
			t.Fatalf("candidate subject %q is not readable: %v", relative, err)
		}
	}
	if len(fixture.RequiredApplicationTypes) != 4 || len(fixture.RequiredOutboundPorts) != 2 ||
		len(fixture.RequiredBehaviorTests) != 29 || len(fixture.CrashFrontiers) != 7 ||
		len(fixture.SecurityCases) != 22 || len(fixture.PrivateLeakMarkers) != 5 {
		t.Fatalf("invalid V16 fixture coverage counts: %+v", fixture)
	}
	if fixture.GitFixture.MinimumVersion != "2.38.0" || fixture.GitFixture.ObjectFormat != "sha1" ||
		fixture.GitFixture.TargetRef != "refs/heads/main" || fixture.GitFixture.AuthorName == "" ||
		fixture.GitFixture.AuthorEmail == "" || fixture.GitFixture.BaseUnixTime <= 0 ||
		fixture.GitFixture.TargetUnixTime <= fixture.GitFixture.BaseUnixTime ||
		len(fixture.GitFixture.BaseFiles) != 2 || len(fixture.GitFixture.TargetChanges) != 2 {
		t.Fatalf("invalid V16 real Git fixture: %+v", fixture.GitFixture)
	}
	oid := regexp.MustCompile(`^[0-9a-f]{40}$`)
	for _, value := range []string{
		fixture.GitFixture.BaseOID, fixture.GitFixture.BaseTreeOID,
		fixture.GitFixture.TargetOID, fixture.GitFixture.TargetTreeOID,
	} {
		if !oid.MatchString(value) {
			t.Fatalf("invalid V16 sha1 object id %q", value)
		}
	}
	if len(fixture.Actors) != 2 || len(fixture.Actors[0].Memberships) != 1 || len(fixture.Actors[1].Memberships) != 2 ||
		len(fixture.Changes) != 3 || fixture.Changes[0].Expected != "clean" ||
		fixture.Changes[1].Expected != "conflicted" || fixture.Changes[2].Expected != "rejected_without_git_mutation" {
		t.Fatalf("invalid V16 actor/change scenarios: actors=%+v changes=%+v", fixture.Actors, fixture.Changes)
	}
	for _, change := range fixture.Changes {
		if change.Name == "" || change.ExecutionRef == "" || len(change.WriteSet) == 0 || len(change.Writes) == 0 {
			t.Fatalf("incomplete V16 change scenario: %+v", change)
		}
	}
	if !reflect.DeepEqual(fixture.DeferredSurfaces, []string{
		"remote_push", "github_gitlab_gitea_forge_adapters", "hostile_process_sandbox_and_attestation",
		"automatic_retention_cleanup", "postgres_s3_multihost",
	}) {
		t.Fatalf("invalid V16 deferred boundary: %v", fixture.DeferredSurfaces)
	}
	if _, err := evidenceGitCanonicalCommit(repositoryRoot, fixture.TrustedBaseGitCommitOID); err != nil {
		t.Fatalf("invalid V16 contract base: %v", err)
	}
}

func v16ValidationShellBody() string {
	return "git diff --check " + v16ContractBaseGitCommitOID + " HEAD --" +
		" && go test -mod=vendor -count=1 . ./acceptance -run \"^(TestProductRoadmapIsExhaustiveAndCausal|TestProductRoadmapV16ScopeAndExecutableContract|TestV16EvidenceBelongsOnlyToWorkspaceGitCapabilities|TestV16AcceptanceCommandRunsWorkspaceGitConsumers|TestRebuildArchitecture|TestTraceabilityRebuildBugLessons|TestTraceabilityRebuildHistoricalBugIDs|TestTraceabilityRebuildHistoricalBugReviewBindings|TestTraceabilityRebuildSchemaValidatesCanonicalLedgers|TestHistoricalBugCapabilityCoverageNeverInfersLegacyClosure|TestAcceptanceV02AuthorityRules|TestAcceptanceV03CanonicalLedgers|TestAcceptanceV06AtomicStateOutbox|TestAcceptanceV09RecoveryBackup|TestAcceptanceV10IdentityProjectsRBAC|TestAcceptanceV16WorkspaceGit|TestV16CandidateSubjectsCoverCommittedDelta)$\"" +
		" && go test -mod=vendor -count=1 ./internal/goal ./internal/governance ./internal/identity ./internal/config ./internal/credentials ./internal/application ./internal/ports ./internal/adapters/agent/fake ./internal/adapters/agent/codex ./internal/adapters/state/sqlite ./internal/adapters/workspace/gitlocal ./internal/bootstrap ./cmd/orquesta" +
		" && timeout --kill-after=10s 180s go test -mod=vendor -race -count=1 -timeout=150s ./internal/application ./internal/ports ./internal/adapters/state/sqlite ./internal/adapters/workspace/gitlocal ./internal/adapters/agent/codex ./internal/bootstrap -run \"^(" +
		strings.Join(v16ExpectedRaceTests(), "|") + ")$\"" +
		" && GOFLAGS=-mod=vendor go vet ./internal/goal ./internal/governance ./internal/identity ./internal/config ./internal/credentials ./internal/application ./internal/ports ./internal/adapters/agent/fake ./internal/adapters/agent/codex ./internal/adapters/state/sqlite ./internal/adapters/workspace/gitlocal ./internal/bootstrap ./cmd/orquesta"
}

func v16ExpectedRaceTests() []string {
	return []string{
		"TestWorkspacePrepareIsIdempotentAndUniquePerExecution",
		"TestReplacementExecutionGetsDistinctWorkspace",
		"TestLaunchUsesExactOpaqueWorkspaceBinding",
		"TestCommitBindsBaseTreeDiffWriteSetAndExecution",
		"TestOutOfWriteSetChangeLeavesGitUnmodified",
		"TestReworkRequiresExplicitParentChangeRef",
		"TestConflictAndStaleIntegrationLeaveTargetUnchanged",
		"TestConcurrentIntegrationCASPreservesLoserPending",
		"TestWorkspaceEffectsReplayEveryCrashFrontierExactlyOnce",
		"TestPendingChangesAreRBACScopedAndSurviveRestart",
		"TestGitWorkspaceRejectsUnsafeFilesystemAndGitControls",
		"TestGitWorkspaceRejectsRepositoryOverlappingPrivateRoot",
		"TestIntegrationReplayRejectsSameKeyWithDifferentPayload",
		"TestWorkspaceEvidenceLeaksNoPrivateAdapterData",
		"TestWorkspaceArchitectureKeepsOneWriterStateOutboxScheduler",
		"TestRealGitSQLiteWorkspaceLifecycleEndToEnd",
		"TestWorkspaceConcurrentPrepareCommitIntegrateRace",
		"TestSQLiteWorkspaceGitRestartRaceAndReplay",
		"TestRecoveryV16RejectsWorkspaceCausalTampering",
		"TestRecoveryV16RejectsMissingIntegrationFacts",
		"TestIntegrateChangeRejectsMalformedTargetBeforeAuthorizationOrAdmission",
		"TestValidateGitOIDRejectsMalformedValues",
	}
}

func v16AssertValidationGates(t *testing.T, command string) {
	t.Helper()
	if strings.Contains(command, "./...") {
		t.Fatalf("V16 acceptance command uses forbidden broad package wildcard: %q", command)
	}
	for _, required := range []string{
		"git diff --check " + v16ContractBaseGitCommitOID + " HEAD --",
		"TestProductRoadmapV16ScopeAndExecutableContract",
		"TestV16EvidenceBelongsOnlyToWorkspaceGitCapabilities",
		"TestV16AcceptanceCommandRunsWorkspaceGitConsumers",
		"TestAcceptanceV16WorkspaceGit",
		"TestV16CandidateSubjectsCoverCommittedDelta",
		"TestAcceptanceV02AuthorityRules", "TestAcceptanceV03CanonicalLedgers",
		"TestAcceptanceV06AtomicStateOutbox", "TestAcceptanceV09RecoveryBackup",
		"TestAcceptanceV10IdentityProjectsRBAC",
		"./internal/application", "./internal/ports", "./internal/adapters/state/sqlite",
		"./internal/adapters/workspace/gitlocal", "./internal/adapters/agent/codex",
		"./internal/bootstrap", "./cmd/orquesta",
		"timeout --kill-after=10s 180s", "-timeout=150s", "GOFLAGS=-mod=vendor go vet",
	} {
		if !strings.Contains(command, required) {
			t.Errorf("V16 acceptance command omits %q", required)
		}
	}
	for _, name := range v16ExpectedRaceTests() {
		if !strings.Contains(command, name) {
			t.Errorf("V16 race gate omits %q", name)
		}
	}
}

func v16AssertSimplicityBudget(t *testing.T, repositoryRoot, baseOID, sealedOID string, numstat []byte) {
	t.Helper()
	type limit struct{ net, max int }
	limits := map[string]*limit{
		"core": {max: 2050}, "adapters": {max: 3650}, "migration": {max: 650}, "tests": {max: 5600},
	}
	for _, row := range strings.Split(strings.TrimSpace(string(numstat)), "\n") {
		if row == "" {
			continue
		}
		fields := strings.SplitN(row, "\t", 3)
		if len(fields) != 3 || fields[0] == "-" || fields[1] == "-" {
			t.Fatalf("V16 simplicity budget cannot classify %q", row)
		}
		added, addErr := strconv.Atoi(fields[0])
		deleted, deleteErr := strconv.Atoi(fields[1])
		if addErr != nil || deleteErr != nil {
			t.Fatalf("V16 invalid numstat %q", row)
		}
		class := v16SimplicityClass(fields[2])
		if class == "" {
			t.Fatalf("V16 simplicity budget cannot classify changed path %q", fields[2])
		}
		if class != "metadata" {
			limits[class].net += added - deleted
		}
	}
	for class, budget := range limits {
		if budget.net > budget.max {
			t.Errorf("V16 simplicity budget %s net LOC=%d, max=%d", class, budget.net, budget.max)
		}
	}
	production := limits["core"].net + limits["adapters"].net + limits["migration"].net
	if production > 6250 {
		t.Errorf("V16 production net LOC=%d, max=6250", production)
	}
	v15AssertStructuralSimplicityWithClassifier(t, repositoryRoot, baseOID, sealedOID, v16SimplicityClass)
}

func v16SimplicityClass(relative string) string {
	switch {
	case strings.HasSuffix(relative, "_test.go"), strings.HasPrefix(relative, "acceptance/"):
		return "tests"
	case relative == "internal/adapters/state/sqlite/migrations/011_workspace_git.sql":
		// One normalized transactional schema migration. Count separately so its
		// declarative DDL does not conceal executable adapter growth.
		return "migration"
	case strings.HasSuffix(relative, ".sql"):
		return ""
	case strings.HasPrefix(relative, "internal/adapters/"), strings.HasPrefix(relative, "internal/bootstrap/"),
		strings.HasPrefix(relative, "internal/config/"), strings.HasPrefix(relative, "config/"),
		strings.HasPrefix(relative, "cmd/orquesta/"):
		return "adapters"
	case strings.HasPrefix(relative, "internal/governance/"), strings.HasPrefix(relative, "internal/goal/"),
		strings.HasPrefix(relative, "internal/application/"), strings.HasPrefix(relative, "internal/identity/"),
		strings.HasPrefix(relative, "internal/ports/"):
		return "core"
	case strings.HasPrefix(relative, "docs/reconstruccion/"), relative == "product/roadmap.json",
		strings.HasPrefix(relative, "product/traceability/"), strings.HasPrefix(relative, "product/evidence/"):
		return "metadata"
	}
	return ""
}
