package acceptance_test

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
)

const v14FixturePath = "acceptance/fixtures/v14_controls.json"
const v14ContractBaseGitCommitOID = "df695543b6486dae51f7eab40417543b48d71de7"
const v14ProductDeltaBaseGitCommitOID = "b7a4672a251b85904d213b7b147a0442a6d906a8"
const v14ProductDeltaSealedGitCommitOID = "6dbc0d808de63973305914b002c3bc2b8a806bb0"

type v14Fixture struct {
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
	DeferredCapabilities           []v14DeferredCapability `json:"deferred_capabilities"`
	DeferredSurfaces               []string                `json:"deferred_surfaces"`
	Operations                     []string                `json:"operations"`
	TargetMatrix                   map[string][]string     `json:"target_matrix"`
	StopModes                      []string                `json:"stop_modes"`
	RequiredUseCases               []string                `json:"required_use_cases"`
	RequiredRepositoryMethods      []string                `json:"required_repository_methods"`
	ForbiddenPrivateAuthorities    []string                `json:"forbidden_private_authorities"`
	RequiredBehaviorTests          []string                `json:"required_behavior_tests"`
	Scenario                       v14Scenario             `json:"scenario"`
	Assertions                     []string                `json:"assertions"`
}

func TestV14CandidateSubjectsCoverCommittedDelta(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v14Fixture](t, filepath.Join(repositoryRoot, filepath.FromSlash(v14FixturePath)))
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
		t.Fatalf("V14 candidate subjects differ from sealed product delta:\nchanged=%v\nfixture=%v", changed, fixture.CandidateSubjects)
	}
	numstat, err := evidenceGit(
		repositoryRoot, "diff", "--numstat",
		fixture.ProductDeltaBaseGitCommitOID, fixture.ProductDeltaSealedGitCommitOID, "--",
	)
	if err != nil {
		t.Fatal(err)
	}
	v14AssertSimplicityBudget(t, numstat)
}

func v14AssertSimplicityBudget(t *testing.T, numstat []byte) {
	t.Helper()
	type limit struct {
		name string
		max  int
		net  int
	}
	limits := map[string]*limit{
		"core":      {name: "domain/application/ports production", max: 2500},
		"adapters":  {name: "adapters/bootstrap production", max: 4700},
		"vendor":    {name: "modernc SQLite patch", max: 160},
		"migration": {name: "SQLite migration", max: 825},
		"tests":     {name: "tests/acceptance", max: 8600},
	}
	for _, row := range strings.Split(strings.TrimSpace(string(numstat)), "\n") {
		if row == "" {
			continue
		}
		fields := strings.SplitN(row, "\t", 3)
		if len(fields) != 3 || fields[0] == "-" || fields[1] == "-" {
			t.Fatalf("V14 simplicity budget cannot classify numstat row %q", row)
		}
		added, addErr := strconv.Atoi(fields[0])
		deleted, deleteErr := strconv.Atoi(fields[1])
		if addErr != nil || deleteErr != nil {
			t.Fatalf("V14 simplicity budget invalid numstat row %q", row)
		}
		if class := v14SimplicityClass(fields[2]); class != "" {
			limits[class].net += added - deleted
		}
	}
	for _, class := range []string{"core", "adapters", "vendor", "migration", "tests"} {
		budget := limits[class]
		if budget.net > budget.max {
			t.Errorf("V14 simplicity budget %s net LOC=%d, max=%d", budget.name, budget.net, budget.max)
		}
	}
}

func v14SimplicityClass(relative string) string {
	switch {
	case strings.HasSuffix(relative, ".sql"):
		return "migration"
	case strings.HasSuffix(relative, "_test.go"), strings.HasPrefix(relative, "acceptance/"):
		return "tests"
	case strings.HasPrefix(relative, "internal/goal/"),
		strings.HasPrefix(relative, "internal/application/"),
		strings.HasPrefix(relative, "internal/ports/"):
		if strings.HasSuffix(relative, ".go") {
			return "core"
		}
	case strings.HasPrefix(relative, "internal/adapters/"),
		strings.HasPrefix(relative, "internal/bootstrap/"):
		if strings.HasSuffix(relative, ".go") {
			return "adapters"
		}
	case strings.HasPrefix(relative, "vendor/modernc.org/sqlite/"):
		if strings.HasSuffix(relative, ".go") {
			return "vendor"
		}
	}
	return ""
}

type v14DeferredCapability struct {
	ID                 string `json:"id"`
	Owner              string `json:"owner"`
	AcceptanceContract string `json:"acceptance_contract"`
}

type v14Scenario struct {
	BaseTime             string   `json:"base_time"`
	ProjectRef           string   `json:"project_ref"`
	GoalRef              string   `json:"goal_ref"`
	PlanGeneration       uint64   `json:"plan_generation"`
	AppSpecGeneration    uint64   `json:"app_spec_generation"`
	SpecHash             string   `json:"spec_hash"`
	WorkItemRefs         []string `json:"work_item_refs"`
	ExecutionRefs        []string `json:"execution_refs"`
	MaxExecutionAttempts uint64   `json:"max_execution_attempts"`
	ControlContenders    int      `json:"control_contenders"`
	LaunchCrashFrontiers []string `json:"launch_crash_frontiers"`
	StopCrashFrontiers   []string `json:"stop_crash_frontiers"`
}

func TestAcceptanceV14Controls(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v14Fixture](t, filepath.Join(repositoryRoot, filepath.FromSlash(v14FixturePath)))
	v14AssertFixtureHeader(t, repositoryRoot, fixture)

	t.Run("generic_control_use_case_and_same_state_authority", func(t *testing.T) {
		orchestrator := reflect.TypeOf((*application.Orchestrator)(nil))
		v14RequireUseCase(t, orchestrator, "Control", "ControlRequest", "ControlResult")
		propose, ok := v14RequireUseCase(t, orchestrator, "ProposeDirectorPlan", "ProposeDirectorPlanRequest", "DirectorPlanResult")
		if ok {
			v14RequireReflectFields(t, propose.Type.In(3), []string{
				"ExpectedGoalRevision", "ExpectedPlanGeneration", "LeaseToken", "LeaseFence",
				"Cause", "SourceWorkItemRef", "ExpectedWorkItemRevision", "SourceExecutionRef", "SourceExecutionAttempt",
			})
		}
		statePort := reflect.TypeOf((*application.StateRepository)(nil)).Elem()
		for _, method := range fixture.RequiredRepositoryMethods {
			if _, ok := statePort.MethodByName(method); !ok {
				t.Errorf("V14_RED StateRepository lacks %s", method)
			}
		}
		dependencies := reflect.TypeOf(application.Dependencies{})
		stateFields, controllerFields := 0, 0
		for index := 0; index < dependencies.NumField(); index++ {
			field := dependencies.Field(index)
			if field.Type == statePort {
				stateFields++
			}
			if field.Type.Name() == "AgentController" {
				controllerFields++
			}
		}
		if stateFields != 1 || controllerFields != 1 {
			t.Errorf("V14_RED Dependencies state/controller fields=%d/%d, want 1/1", stateFields, controllerFields)
		}
	})

	t.Run("typed_exact_controls_and_replan_fences", func(t *testing.T) {
		applicationDirectory := filepath.Join(repositoryRoot, "internal", "application")
		v14RequireProductionFields(t, applicationDirectory, "ControlRequest", []string{
			"RequestRef", "Operation", "Target", "GoalRef", "ExpectedGoalRevision",
			"ExpectedPlanGeneration", "ExpectedAppSpecGeneration", "ExpectedSpecHash",
			"WorkItemRef", "ExpectedWorkItemRevision", "ExecutionRef", "ExpectedExecutionAttempt", "Mode", "Reason",
		})
		v14RequireProductionFields(t, applicationDirectory, "ControlRecord", []string{
			"Ref", "RequestRef", "RequestFingerprint", "PrincipalRef", "ProjectRef", "GoalRef",
			"WorkItemRef", "WorkItemRevision", "ExecutionRef", "ExecutionAttempt", "Operation", "Target", "Mode",
			"Reason", "GoalRevision", "PlanGeneration", "AppSpecGeneration", "SpecHash", "Status", "RequestedAt", "ConfirmedAt",
			"ReceiptRef", "SupersedesControlRef", "SupersededAt", "SupersededByControlRef", "AuthorizationReceipt",
		})
		v14RequireProductionFields(t, applicationDirectory, "ControlResult", []string{"Control", "Created"})

		applicationSource := v10ReadProductionGo(t, applicationDirectory)
		goalSource := v10ReadProductionGo(t, filepath.Join(repositoryRoot, "internal", "goal"))
		for _, required := range []string{
			`"pause"`, `"resume"`, `"cancel"`, `"stop"`, `"retry"`,
			`"stop_agent"`, "ActionStopAgent", "ControlSequence", "ReworkOf", "ReplanCause",
		} {
			if !strings.Contains(applicationSource+goalSource, required) {
				t.Errorf("V14_RED causal control contract lacks %q", required)
			}
		}
		for _, state := range []string{`"canceled"`, `"stopped"`, `"interrupted"`, `"superseded"`} {
			if !strings.Contains(applicationSource+goalSource, state) {
				t.Errorf("V14_RED lifecycle lacks %s", state)
			}
		}
		if strings.Contains(goalSource, "GoalStatePaused") {
			t.Error("V14 pause became a parallel Goal lifecycle state")
		}
	})

	t.Run("neutral_agent_controller_is_selective_not_shutdown", func(t *testing.T) {
		applicationSource := v10ReadProductionGo(t, filepath.Join(repositoryRoot, "internal", "application"))
		goalSource := v10ReadProductionGo(t, filepath.Join(repositoryRoot, "internal", "goal"))
		portsDirectory := filepath.Join(repositoryRoot, "internal", "ports")
		portsSource := v10ReadProductionGo(t, portsDirectory)
		for _, required := range []string{"type AgentController interface", "ControlCapabilities(", "Stop("} {
			if !strings.Contains(applicationSource, required) {
				t.Errorf("V14_RED application port lacks %q", required)
			}
		}
		v14RequireProductionFields(t, portsDirectory, "AgentControlCapabilities", []string{"CooperativeStop", "ForcedStop"})
		v14RequireProductionFields(t, portsDirectory, "AgentStopRequest", []string{
			"ExecutionRef", "GoalRef", "WorkItemRef", "PlanGeneration", "AppSpecGeneration",
			"ExecutionAttempt", "SpecHash", "ProviderRef", "ModelRef", "AgentRef", "ExternalRef", "Mode", "IdempotencyKey",
		})
		v14RequireProductionFields(t, portsDirectory, "AgentStopReceipt", []string{
			"ExecutionRef", "GoalRef", "WorkItemRef", "PlanGeneration", "AppSpecGeneration",
			"ExecutionAttempt", "SpecHash", "ProviderRef", "ModelRef", "AgentRef", "ExternalRef",
			"Mode", "IdempotencyKey", "Status", "ReceiptRef", "ConfirmedAt",
		})
		if strings.Contains(applicationSource, "AgentController interface {\n\tShutdown(") ||
			strings.Contains(portsSource, "AgentStopReceipt struct {\n\tPID ") {
			t.Error("V14 selective control leaks shutdown or PID through neutral public evidence")
		}
		codexSource := v10ReadProductionGo(t, filepath.Join(repositoryRoot, "internal", "adapters", "agent", "codex"))
		for _, required := range []string{`"process.json"`, `"owner.lock"`, "RuntimeScope", "ExtraFiles"} {
			if !strings.Contains(codexSource, required) {
				t.Errorf("V14_RED Codex private crash gate lacks %q", required)
			}
		}
		for _, forbidden := range []string{"RuntimeScope", "ProcessDescriptor", "owner.lock", "process.json"} {
			if strings.Contains(goalSource+applicationSource+portsSource, forbidden) {
				t.Errorf("V14 neutral core leaks Codex private detail %q", forbidden)
			}
		}
	})

	t.Run("sqlite_extends_existing_authority_without_private_engine", func(t *testing.T) {
		migrationPath := filepath.Join(repositoryRoot, "internal", "adapters", "state", "sqlite", "migrations", "009_controls.sql")
		migration, err := os.ReadFile(migrationPath)
		if err != nil {
			t.Errorf("V14_RED SQLite migration 009 missing: %v", err)
		} else {
			text := strings.ToLower(string(migration))
			for _, required := range []string{
				"control", "control_sequence", "stop_agent", "rework_of", "outbox", "mailbox",
			} {
				if !strings.Contains(text, required) {
					t.Errorf("V14_RED migration lacks %s", required)
				}
			}
			for _, forbidden := range []string{"runtime_orders", "control_outbox", "control_queue", "control_scheduler"} {
				if strings.Contains(text, forbidden) {
					t.Errorf("V14 migration creates private authority %s", forbidden)
				}
			}
			for _, forbidden := range []string{"pid", "pgid", "process_owner", "runtime_ownership"} {
				if strings.Contains(text, forbidden) {
					t.Errorf("V14 neutral migration leaks Codex process detail %s", forbidden)
				}
			}
		}
		production := strings.ToLower(v10ReadProductionGo(t, filepath.Join(repositoryRoot, "internal", "application")))
		for _, forbidden := range fixture.ForbiddenPrivateAuthorities {
			if strings.Contains(production, strings.ToLower("type "+forbidden+" ")) {
				t.Errorf("V14 adds private authority %s", forbidden)
			}
		}
		stateSource := strings.ToLower(v10ReadProductionGo(t, filepath.Join(repositoryRoot, "internal", "adapters", "state", "sqlite")))
		for _, forbidden := range []string{"process.json", "owner.lock", "runtime_ownership", "process_owner", "runtime_scope", "pgid"} {
			if strings.Contains(stateSource, forbidden) {
				t.Errorf("V14 SQLite adapter contains Codex-private ownership %q", forbidden)
			}
		}
		migrationEntries, err := os.ReadDir(filepath.Join(repositoryRoot, "internal", "adapters", "state", "sqlite", "migrations"))
		if err != nil {
			t.Fatal(err)
		}
		var allMigrations strings.Builder
		for _, entry := range migrationEntries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
				continue
			}
			payload, readErr := os.ReadFile(filepath.Join(repositoryRoot, "internal", "adapters", "state", "sqlite", "migrations", entry.Name()))
			if readErr != nil {
				t.Fatal(readErr)
			}
			allMigrations.Write(payload)
		}
		migrationSource := strings.ToLower(allMigrations.String())
		for _, forbidden := range []string{"process.json", "owner.lock", "runtime_ownership", "process_owner", "runtime_scope", "pgid"} {
			if strings.Contains(migrationSource, forbidden) {
				t.Errorf("V14 SQLite migrations contain Codex-private ownership %q", forbidden)
			}
		}
	})

	t.Run("races_restart_mailbox_and_real_codex_are_executable", func(t *testing.T) {
		testSource := v13ReadGoTests(t,
			repositoryRoot,
			filepath.Join(repositoryRoot, "internal", "goal"),
			filepath.Join(repositoryRoot, "internal", "application"),
			filepath.Join(repositoryRoot, "internal", "ports"),
			filepath.Join(repositoryRoot, "internal", "adapters", "state", "sqlite"),
			filepath.Join(repositoryRoot, "internal", "adapters", "agent", "fake"),
			filepath.Join(repositoryRoot, "internal", "adapters", "agent", "codex"),
			filepath.Join(repositoryRoot, "internal", "bootstrap"),
		)
		for _, name := range fixture.RequiredBehaviorTests {
			if !strings.Contains(testSource, "func "+name+"(") {
				t.Errorf("V14_RED executable behavior test missing: %s", name)
			}
		}
		if strings.Contains(testSource, "func TestExecutionAttemptPolicyFailsWorkItemOnlyAfterExhaustion(") {
			t.Error("V14_RED obsolete V06 immediate-failure expectation remains active")
		}
	})

	t.Run("prior_authority_and_handoff_ratchets_remain_intact", func(t *testing.T) {
		authorityFixture := evidenceDecodeStrictJSON[v02Fixture](
			t, filepath.Join(repositoryRoot, filepath.FromSlash(v02FixturePath)),
		)
		v02AssertSingleWriterAndScheduler(
			t, v02LoadSources(t, repositoryRoot, authorityFixture.ProductModule, authorityFixture.ProductRoots),
			authorityFixture.Lifecycle,
		)
		v05AssertContractualLineageDoesNotCloseGoalEarly(t)
		mailboxTests := v13ReadGoTests(t,
			filepath.Join(repositoryRoot, "internal", "goal"),
			filepath.Join(repositoryRoot, "internal", "application"),
		)
		for _, required := range []string{
			"TestMailboxRecipientFailureRetiresWithoutReplacementOrForgedResolution",
			"TestRequiredDependencySkippedChildClosesWithoutSyntheticMailboxFact",
		} {
			if !strings.Contains(mailboxTests, "func "+required+"(") {
				t.Errorf("V14 regresses V13 handoff ratchet %s", required)
			}
		}
	})
}

func TestAcceptanceV14ControlsReceipt(t *testing.T) {
	evidenceAssertReceiptV3(t, evidenceRepositoryRoot(t), evidenceReceiptV3Expectation{
		Contract: "AC-V14-CONTROLS", FixturePath: v14FixturePath,
		ReceiptPath:       "product/evidence/v14_controls.json",
		ExecutedNotBefore: "2026-07-16T00:00:00Z", TrustedBaseGitCommitOID: v14ContractBaseGitCommitOID,
	})
}

func v14AssertFixtureHeader(t *testing.T, repositoryRoot string, fixture v14Fixture) {
	t.Helper()
	wantDeferred := []v14DeferredCapability{
		{ID: "GOV-15", Owner: "budgets_effects", AcceptanceContract: "AC-V15-BUDGETS-EFFECTS"},
		{ID: "GOV-17", Owner: "command_registry", AcceptanceContract: "AC-V20-COMMAND-REGISTRY"},
		{ID: "UI-02", Owner: "command_registry", AcceptanceContract: "AC-V20-COMMAND-REGISTRY"},
		{ID: "OPS-11", Owner: "postgres_s3_multihost", AcceptanceContract: "AC-V31-POSTGRES-S3-MULTIHOST"},
	}
	wantDeferredSurfaces := []string{
		"approval_budget_fairness_and_effect_retry_idempotency",
		"public_http_mcp_cli_control_bindings_and_command_registry",
		"distributed_process_ownership_and_multihost_adoption",
	}
	if fixture.SchemaVersion != 1 || fixture.ReceiptSchemaVersion != 3 ||
		fixture.ContractID != "AC-V14-CONTROLS" ||
		fixture.TrustedBaseGitCommitOID != v14ContractBaseGitCommitOID ||
		fixture.ProductDeltaBaseGitCommitOID != v14ProductDeltaBaseGitCommitOID ||
		fixture.ProductDeltaSealedGitCommitOID != v14ProductDeltaSealedGitCommitOID ||
		fixture.OutputPath != "product/evidence/v14_controls.output.txt" ||
		fixture.ReceiptPath != "product/evidence/v14_controls.json" ||
		!reflect.DeepEqual(fixture.OwnedCapabilityIDs, []string{"GOV-07", "STG-15", "ORC-03", "ORC-16"}) ||
		!reflect.DeepEqual(fixture.DeferredCapabilities, wantDeferred) ||
		!reflect.DeepEqual(fixture.DeferredSurfaces, wantDeferredSurfaces) ||
		!reflect.DeepEqual(fixture.Operations, []string{"pause", "resume", "cancel", "stop", "retry", "replan"}) ||
		!reflect.DeepEqual(fixture.StopModes, []string{"cooperative", "forced"}) ||
		!reflect.DeepEqual(fixture.RequiredUseCases, []string{"Control", "ProposeDirectorPlan"}) ||
		!reflect.DeepEqual(fixture.RequiredRepositoryMethods, []string{"ControlReplay", "ApplyControl"}) ||
		!reflect.DeepEqual(fixture.RequiredBehaviorTests, v14ExpectedBehaviorTests()) ||
		!reflect.DeepEqual(fixture.Assertions, v14ExpectedAssertions()) {
		t.Fatalf("invalid V14 fixture header: %+v", fixture)
	}
	wantTargets := map[string][]string{
		"pause": {"goal", "work_item"}, "resume": {"goal", "work_item"},
		"cancel": {"goal", "work_item"}, "stop": {"execution"},
		"retry": {"work_item", "execution_fence"}, "replan": {"goal", "work_item", "execution_fence"},
	}
	if !reflect.DeepEqual(fixture.TargetMatrix, wantTargets) {
		t.Fatalf("invalid V14 target matrix: %+v", fixture.TargetMatrix)
	}
	if _, err := time.Parse(time.RFC3339Nano, fixture.Scenario.BaseTime); err != nil ||
		fixture.Scenario.PlanGeneration == 0 || fixture.Scenario.AppSpecGeneration == 0 ||
		!goal.IsCanonicalAppSpecHash(fixture.Scenario.SpecHash) || len(fixture.Scenario.WorkItemRefs) != 4 ||
		len(fixture.Scenario.ExecutionRefs) != 4 || fixture.Scenario.MaxExecutionAttempts < 2 ||
		fixture.Scenario.ControlContenders < 2 ||
		!reflect.DeepEqual(fixture.Scenario.LaunchCrashFrontiers, []string{
			"after_wrapper_start_before_descriptor", "after_descriptor_fsync_before_gate_release",
		}) || !reflect.DeepEqual(fixture.Scenario.StopCrashFrontiers, []string{
		"before_stop_effect", "after_stop_effect_before_receipt",
	}) {
		t.Fatalf("invalid V14 scenario: %+v", fixture.Scenario)
	}
	if fixture.Command != "sh -c '"+v14ValidationShellBody()+"'" ||
		!reflect.DeepEqual(fixture.ExecutionArgv, []string{"sh", "-c", v14ValidationShellBody()}) {
		t.Fatalf("invalid V14 command/argv: %q %#v", fixture.Command, fixture.ExecutionArgv)
	}
	v14AssertRaceGate(t, v14ValidationShellBody())
	if _, err := evidenceGitCanonicalCommit(repositoryRoot, fixture.TrustedBaseGitCommitOID); err != nil {
		t.Fatalf("invalid V14 real contract base: %v", err)
	}
	if err := evidenceValidateCandidateSubjects(fixture.CandidateSubjects, fixture.ReceiptPath, fixture.OutputPath); err != nil {
		t.Fatal(err)
	}
	for _, relative := range fixture.CandidateSubjects {
		if _, err := os.Stat(filepath.Join(repositoryRoot, filepath.FromSlash(relative))); err != nil {
			t.Fatalf("candidate subject %q is not readable: %v", relative, err)
		}
	}
}

func v14ValidationShellBody() string {
	return "go test -mod=vendor -count=1 . ./acceptance -run \"^(TestProductRoadmapIsExhaustiveAndCausal|TestProductRoadmapV14ScopeAndExecutableContract|TestV14EvidenceBelongsOnlyToControlCapabilities|TestV14AcceptanceCommandRunsControlConsumers|TestRebuildArchitecture|TestTraceabilityRebuildBugLessons|TestAcceptanceV14Controls|TestV14CandidateSubjectsCoverCommittedDelta)$\"" +
		" && go test -mod=vendor -count=1 ./internal/goal ./internal/identity ./internal/config ./internal/credentials ./internal/application ./internal/ports ./internal/adapters/agent/fake ./internal/adapters/agent/codex ./internal/adapters/state/sqlite ./internal/bootstrap ./cmd/orquesta" +
		" && " + v14RaceValidationShellBody()
}

func v14RaceValidationShellBody() string {
	return "go test -mod=vendor -race -count=1 ./internal/application ./internal/adapters/state/sqlite ./internal/adapters/agent/codex ./internal/bootstrap" +
		" -run \"^(TestConcurrentIdenticalControlCASLoserReturnsExactReplay|TestControlsGoalAndWorkItemCancelCompletionCASBothOrders|TestControlsStopCompletionCASAndUnsupportedMode|TestControlsStopCrashReplayConvergesWithoutDuplicateEffect|TestClaimedRetryRevalidatesPauseBeforeLaunchPreparation|TestClaimedAutomaticReplacementRevalidatesPauseBeforeLaunchPreparation|TestSQLiteControlsRestartAndConcurrentCAS|TestSQLiteForcedStopSupersessionIsAtomicConcurrentAndRestartSafe|TestSQLiteTerminalStopSettlesAfterRestartWithReplacementAgentRouting|TestV14RecoveryAcceptsClaimedTerminalStopThenReclaimsAndSettlesOnce|TestCodexSelectiveStopPreservesSiblingProcessTrees|TestCodexSelectiveStopAdoptsAfterCrashAndRejectsReusedPID|TestCodexLaunchGateCrashNeverOrphansProcess|TestCodexOwnerLockIsExclusiveAndCLOEXEC|TestCodexRestoredDatabaseCannotAdoptSourceProcess|TestBuildBindsAgentToOpenedRepositoryIdentity|TestRealCodexControlsThroughProductionComposition|TestRealCodexCooperativeStopLeavesResidentSchedulerLive)$\""
}

func v14AssertRaceGate(t *testing.T, command string) {
	t.Helper()
	const requiredPrefix = "go test -mod=vendor -race -count=1"
	if !strings.Contains(command, " && "+requiredPrefix) {
		t.Fatalf("V14 acceptance command omits race detector gate: %q", command)
	}
	for _, required := range []string{
		"./internal/application", "./internal/adapters/state/sqlite",
		"./internal/adapters/agent/codex", "./internal/bootstrap",
		"TestConcurrentIdenticalControlCASLoserReturnsExactReplay",
		"TestSQLiteForcedStopSupersessionIsAtomicConcurrentAndRestartSafe",
		"TestCodexSelectiveStopAdoptsAfterCrashAndRejectsReusedPID",
		"TestRealCodexControlsThroughProductionComposition",
		"TestRealCodexCooperativeStopLeavesResidentSchedulerLive",
	} {
		if !strings.Contains(v14RaceValidationShellBody(), required) {
			t.Errorf("V14 race gate omits %q: %q", required, v14RaceValidationShellBody())
		}
	}
}

func v14RequireUseCase(t *testing.T, owner reflect.Type, name, requestName, resultName string) (reflect.Method, bool) {
	t.Helper()
	method, ok := owner.MethodByName(name)
	if !ok {
		t.Errorf("V14_RED Orchestrator lacks %s", name)
		return reflect.Method{}, false
	}
	contextType := reflect.TypeOf((*context.Context)(nil)).Elem()
	errorType := reflect.TypeOf((*error)(nil)).Elem()
	if method.Type.NumIn() != 4 || method.Type.In(1) != contextType ||
		method.Type.In(2) != reflect.TypeOf(application.Access{}) || method.Type.In(3).Name() != requestName ||
		method.Type.NumOut() != 2 || method.Type.Out(0).Name() != resultName || method.Type.Out(1) != errorType {
		t.Errorf("V14_RED %s signature=%s", name, method.Type)
		return method, false
	}
	return method, true
}

func v14RequireReflectFields(t *testing.T, owner reflect.Type, required []string) {
	t.Helper()
	for _, name := range required {
		if _, ok := owner.FieldByName(name); !ok {
			t.Errorf("V14_RED %s lacks %s", owner.Name(), name)
		}
	}
}

func v14RequireProductionFields(t *testing.T, directory, typeName string, required []string) {
	t.Helper()
	fields, found := v13ProductionTypeFields(t, directory, typeName)
	if !found {
		t.Errorf("V14_RED production type %s missing", typeName)
		return
	}
	for _, name := range required {
		if !fields[name] {
			t.Errorf("V14_RED %s lacks %s", typeName, name)
		}
	}
}

func v14ExpectedBehaviorTests() []string {
	return []string{
		"TestControlsPauseBeforeAndAfterLaunchPrepared",
		"TestControlsEffectivePauseRequiresBothScopesResumed",
		"TestPauseGatesRetryUntilRecordLaunchPrepared",
		"TestPauseGatesAutomaticReplacementThroughoutBackoff",
		"TestClaimedRetryRevalidatesPauseBeforeLaunchPreparation",
		"TestClaimedAutomaticReplacementRevalidatesPauseBeforeLaunchPreparation",
		"TestControlsReplayAndSemanticConflict",
		"TestConcurrentIdenticalControlCASLoserReturnsExactReplay",
		"TestControlsCancelBeforeAndAfterLaunchPrepared",
		"TestControlsGoalAndWorkItemCancelCompletionCASBothOrders",
		"TestControlsStopCompletionCASAndUnsupportedMode",
		"TestControlsForcedStopSupersedesOnlyExactPendingCooperativeStop",
		"TestControlsForcedEscalationSettlesWhenCooperativeAlreadyStoppedTarget",
		"TestControlsRetryCreatesFreshExecutionAndPreservesStoppedAttempt",
		"TestControlsRetryRejectsTerminalGoalRetiredMailboxAndAttemptLimit",
		"TestControlsReplanSplitStoppedAndFailedSources",
		"TestControlsNestedReplanResolvesLogicalOutcome",
		"TestControlsReplanRejectsSkippedDescendantAndHandoffEndpoints",
		"TestControlsCanceledHandoffChildFailsWithoutSyntheticResolution",
		"TestControlsExecutionExhaustionInterruptsWithoutClosingGoal",
		"TestControlsExactFencesAndInvalidTargetPairsLeaveNoEffects",
		"TestSQLiteControlsRestartAndConcurrentCAS",
		"TestSQLitePauseGatesQueuedRetryAcrossRestart",
		"TestSQLitePauseGatesAutomaticReplacementBackoffAcrossRestart",
		"TestSQLiteForcedStopSupersessionIsAtomicConcurrentAndRestartSafe",
		"TestSQLiteForcedStopRejectsQuarantinedCooperativeOwnerWithoutPartialWrite",
		"TestSQLiteTerminalStopSettlesAfterRestartWithReplacementAgentRouting",
		"TestV14RecoveryAndBackupRejectStopSupersessionTampering",
		"TestV14RecoveryRejectsStopActionAndEffectReceiptCausalTampering",
		"TestControlsStopCrashReplayConvergesWithoutDuplicateEffect",
		"TestCodexSelectiveStopPreservesSiblingProcessTrees",
		"TestCodexSelectiveStopAdoptsAfterCrashAndRejectsReusedPID",
		"TestCodexLaunchGateCrashNeverOrphansProcess",
		"TestCodexOwnerLockIsExclusiveAndCLOEXEC",
		"TestCodexRestoredDatabaseCannotAdoptSourceProcess",
		"TestSQLiteBackupExcludesCodexPrivateProcessJournal",
		"TestLocalStateConnectorRejectsConnectionOpenedAcrossABASwap",
		"TestLocalStateConnectorDoesNotCreateMissingCapturedPath",
		"TestLocalStateConnectorRejectsLazyConnectionAfterReplacement",
		"TestLocalStateConnectorKeepsOneCanonicalWALNamespace",
		"TestModerncSQLiteRuntimeScopePatchIsReproducible",
		"TestRealCodexCooperativeStopLeavesResidentSchedulerLive",
		"TestRealCodexControlsThroughProductionComposition",
	}
}

func v14ExpectedAssertions() []string {
	return []string{
		"controls are authenticated request-idempotent application mutations bound to exact project Goal AppSpec generation and hash PlanGeneration WorkItem revision Execution attempt request ref and fingerprint",
		"pause and resume are reversible Goal or WorkItem dispatch gates whose effective union blocks new launch_agent claims while observe_agent mailbox and in-flight work continue",
		"pause before RecordLaunchPrepared invalidates the launch claim while a durable prepared launch remains in flight and records its exact acceptance or rejection before stop can claim",
		"resume reclaims the existing pending action without creating another Execution outbox action or effect",
		"stop targets one exact Execution and generation through the existing outbox scheduler and AgentController and separates stop requested from exact stop confirmed",
		"cooperative and forced stop execute only when adapter capabilities advertise them and unsupported never becomes stopped or invokes global Shutdown",
		"forced stop supersedes only the exact still-active cooperative stop for the same Execution; atomic lineage retires the old action before the new one while completed retired or quarantined owners reject without partial writes",
		"four disjoint A B C D executions prove that stopping B preserves A C D processes state and progress before and after crash restart without global Shutdown",
		"stop requested permits late V13 delivery consume and acknowledgement until confirmation; confirmation retires only unresolved exact-recipient mailbox without readdress synthetic ACK or ChildHandoffResolution",
		"completion and stop race through one CAS; already completed is observed rather than falsified as stopped and every terminal Execution remains immutable and never restarts",
		"execution retry after confirmed stop creates a new Execution ref idempotency key and attempt with replaces_execution_ref while terminal Goal retired mailbox or exhausted attempt policy rejects without effects",
		"V14 execution retry preserves V06 attempt receipts fences and idempotency but does not claim V15 external-effect retry approvals budgets quotas or fairness",
		"cancel is irreversible at Goal or WorkItem scope, blocks new launches and delivery claims after its CAS, cancels queued work locally and stops every exact prepared or running Execution before terminal publication",
		"Goal and WorkItem cancel race with completion by CAS and preserve exact terminal Execution evidence without converting it into another terminal state or leaving an orphan process or launch",
		"WorkItem cancel preserves independent work, derives dependency_canceled skips, fails the resolved Goal, and treats a canceled HandoffRequired endpoint as a causal failure without synthetic resolution",
		"ProposeDirectorPlan remains the only replan entry and requires live Director lease token fence authenticated principal exact revisions generations source WorkItem and causal Execution or assessment evidence",
		"split_pending atomically cancels the queued Execution consumes its launch marks the source superseded and appends one or more successors without a reclaimable source action",
		"exhausted execution attempts leave the last Execution failed and the WorkItem interrupted with cause execution_failed while the Goal stays open and dependents stay pending for replan or cancel",
		"append-only replan records only successor rework_of source relations and recursively derives logical source success or failure across nested successors without rewriting history",
		"replan rejects stale or revoked authority invalid target pairs cycles write-set conflicts already skipped descendants and every HandoffRequired endpoint with zero partial snapshot event outbox receipt or effect",
		"one StateRepository transaction persists control snapshot audit event outbox consumption and receipt while replay returns the same frontier and conflicting request semantics fail",
		"SQLite restart backup restore and races at pause claim cancel launch stop completion retry and replan preserve one fenced WorkItem lease and never repeat a terminal effect",
		"the neutral AgentController contract passes one fake suite and Codex proves exact process-tree stop crash adoption and PID PGID birth-identity checks while clean Shutdown remains separate",
		"Codex persists process identity only in its existing private WorkRoot journal behind an FD3 launch gate holds one CLOEXEC owner.lock and binds local runtime scope to non-backup-clonable StateRepository file identity while distributed ownership remains V31",
		"every SQLite physical connection validates the retained local file identity before configuration so path replacement lazy open missing-path recreation and alternate WAL namespaces fail closed",
		"control receipts expose opaque identities without PID argv environment prompt or secrets and V14 adds no undeclared configuration key",
		"V14 preserves the V02 single writer V05 DAG and dependency rules V06 atomic retries and receipts V07 config V08 credentials V09 recovery V10 RBAC V12 Director and V13 mailbox ratchets without another store scheduler loop daemon database or lifecycle",
		"V14 exposes application use cases only; HTTP MCP CLI command registry and full i18n bindings remain V20 and V21 while budgets effects workspace reviews provider parity generic messages UI and later surfaces remain deferred",
	}
}
