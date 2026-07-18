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
)

const v15FixturePath = "acceptance/fixtures/v15_budgets_effects.json"
const v15ContractBaseGitCommitOID = "7aa91cbea80741c6d757ec17d09bc9d7aa7e3085"
const v15ProductDeltaSealedGitCommitOID = "0000000000000000000000000000000000000000"

type v15Fixture struct {
	SchemaVersion                  int         `json:"schema_version"`
	ReceiptSchemaVersion           int         `json:"receipt_schema_version"`
	ContractID                     string      `json:"contract_id"`
	TrustedBaseGitCommitOID        string      `json:"trusted_base_git_commit_oid"`
	ProductDeltaBaseGitCommitOID   string      `json:"product_delta_base_git_commit_oid"`
	ProductDeltaSealedGitCommitOID string      `json:"product_delta_sealed_git_commit_oid"`
	Command                        string      `json:"command"`
	ExecutionArgv                  []string    `json:"execution_argv"`
	OutputPath                     string      `json:"output_path"`
	ReceiptPath                    string      `json:"receipt_path"`
	CandidateSubjects              []string    `json:"candidate_subjects"`
	OwnedCapabilityIDs             []string    `json:"owned_capability_ids"`
	DependencyVerticals            []string    `json:"dependency_verticals"`
	BudgetDimensions               []string    `json:"budget_dimensions"`
	BudgetScopes                   []string    `json:"budget_scopes"`
	RiskLevels                     []string    `json:"risk_levels"`
	ReasoningEfforts               []string    `json:"reasoning_efforts"`
	EffectFacts                    []string    `json:"effect_facts"`
	EffectKinds                    []string    `json:"effect_kinds"`
	ApprovalOutcomes               []string    `json:"approval_outcomes"`
	ApprovalSources                []string    `json:"approval_sources"`
	DefaultParentLaunchesPerCycle  int         `json:"default_parent_launches_per_cycle"`
	DefaultMaxChildrenPerParent    int         `json:"default_max_children_per_parent"`
	RequiredUseCases               []string    `json:"required_use_cases"`
	RequiredRepositoryMethods      []string    `json:"required_repository_methods"`
	ForbiddenPrivateAuthorities    []string    `json:"forbidden_private_authorities"`
	RequiredBehaviorTests          []string    `json:"required_behavior_tests"`
	DeferredSurfaces               []string    `json:"deferred_surfaces"`
	Scenario                       v15Scenario `json:"scenario"`
}

type v15Scenario struct {
	BaseTime             string   `json:"base_time"`
	Projects             int      `json:"projects"`
	GoalsPerProject      int      `json:"goals_per_project"`
	ConcurrentClaims     int      `json:"concurrent_claims"`
	GlobalProcessSlots   int      `json:"global_process_slots"`
	ProjectProcessSlots  int      `json:"project_process_slots"`
	GoalProcessSlots     int      `json:"goal_process_slots"`
	EffectCrashFrontiers []string `json:"effect_crash_frontiers"`
}

func TestV15CandidateSubjectsCoverCommittedDelta(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v15Fixture](t, filepath.Join(repositoryRoot, filepath.FromSlash(v15FixturePath)))
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
		t.Fatalf("V15 candidate subjects differ from sealed product delta:\nchanged=%v\nfixture=%v", changed, fixture.CandidateSubjects)
	}
	numstat, err := evidenceGit(repositoryRoot, "diff", "--numstat",
		fixture.ProductDeltaBaseGitCommitOID, fixture.ProductDeltaSealedGitCommitOID, "--")
	if err != nil {
		t.Fatal(err)
	}
	v15AssertSimplicityBudget(t, numstat)
}

func TestAcceptanceV15BudgetsEffects(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v15Fixture](t, filepath.Join(repositoryRoot, filepath.FromSlash(v15FixturePath)))
	v15AssertFixtureHeader(t, repositoryRoot, fixture)

	t.Run("one_writer_state_outbox_and_scheduler", func(t *testing.T) {
		statePort := reflect.TypeOf((*application.StateRepository)(nil)).Elem()
		dependencies := reflect.TypeOf(application.Dependencies{})
		stateFields := 0
		for index := 0; index < dependencies.NumField(); index++ {
			if dependencies.Field(index).Type == statePort {
				stateFields++
			}
		}
		if stateFields != 1 {
			t.Errorf("V15_RED Dependencies StateRepository fields=%d, want 1", stateFields)
		}
		for _, method := range fixture.RequiredRepositoryMethods {
			if _, ok := statePort.MethodByName(method); !ok {
				t.Errorf("V15_RED StateRepository lacks %s", method)
			}
		}
		production := strings.ToLower(v15ReadProductionGo(t,
			filepath.Join(repositoryRoot, "internal", "application"),
			filepath.Join(repositoryRoot, "internal", "adapters", "state", "sqlite"),
		))
		for _, forbidden := range fixture.ForbiddenPrivateAuthorities {
			if strings.Contains(production, strings.ToLower("type "+forbidden+" ")) {
				t.Errorf("V15 private authority %s is forbidden", forbidden)
			}
		}
	})

	t.Run("typed_budget_plan_and_risk_contract", func(t *testing.T) {
		governance := v15ReadProductionGo(t, filepath.Join(repositoryRoot, "internal", "governance"))
		applicationSource := v15ReadProductionGo(t, filepath.Join(repositoryRoot, "internal", "application"))
		for _, typeName := range []string{
			"ResourceVector", "BudgetEnvelope", "BudgetDemand", "BudgetReservation", "BudgetSettlement",
			"SecurityCriticality", "ReasoningEffort",
		} {
			if !strings.Contains(governance+applicationSource, "type "+typeName+" ") {
				t.Errorf("V15_RED typed governance fact %s missing", typeName)
			}
		}
		for _, field := range []string{
			"Tokens", "MoneyMicros", "Currency", "ActiveTimeNS", "ProcessSlots", "DiskBytes",
		} {
			if !strings.Contains(governance+applicationSource, field) {
				t.Errorf("V15_RED resource field %s missing", field)
			}
		}
		planning := filepath.Join(repositoryRoot, "internal", "application")
		for _, field := range []string{"BudgetDemand", "SecurityCriticality", "ReasoningEffort"} {
			fields, found := v13ProductionTypeFields(t, planning, "WorkItemSpec")
			if !found || !fields[field] {
				t.Errorf("V15_RED WorkItemSpec lacks %s", field)
			}
		}
		if strings.Contains(strings.ToLower(governance), "codex") ||
			strings.Contains(strings.ToLower(governance), "keyword") {
			t.Error("V15 neutral governance contains provider or keyword policy")
		}
	})

	t.Run("effect_facts_and_exact_approval_are_distinct", func(t *testing.T) {
		applicationSource := v15ReadProductionGo(t, filepath.Join(repositoryRoot, "internal", "application"))
		for _, typeName := range []string{"EffectIntent", "EffectApproval", "EffectAttempt", "EffectReceipt"} {
			if !strings.Contains(applicationSource, "type "+typeName+" ") {
				t.Errorf("V15_RED effect fact %s missing", typeName)
			}
		}
		orchestrator := reflect.TypeOf((*application.Orchestrator)(nil))
		for _, useCase := range fixture.RequiredUseCases {
			method, ok := orchestrator.MethodByName(useCase)
			if !ok {
				t.Errorf("V15_RED Orchestrator lacks %s", useCase)
				continue
			}
			contextType := reflect.TypeOf((*context.Context)(nil)).Elem()
			errorType := reflect.TypeOf((*error)(nil)).Elem()
			if method.Type.NumIn() != 4 || method.Type.In(1) != contextType ||
				method.Type.In(2) != reflect.TypeOf(application.Access{}) ||
				method.Type.NumOut() != 2 || method.Type.Out(1) != errorType {
				t.Errorf("V15_RED %s signature=%s", useCase, method.Type)
			}
		}
		v14RequireProductionFields(t, filepath.Join(repositoryRoot, "internal", "application"),
			"ActionRecord", []string{"EffectIntentRef"})
		v14RequireProductionFields(t, filepath.Join(repositoryRoot, "internal", "application"),
			"ActionClaim", []string{"BudgetReservationRef"})
		fields, found := v13ProductionTypeFields(t, filepath.Join(repositoryRoot, "internal", "application"),
			"ActionConsumptionReceipt")
		if !found || !fields["EffectReceiptRef"] || fields["EffectStatus"] || fields["EffectConfirmedAt"] {
			t.Errorf("V15_RED action receipt must reference, not duplicate, effect receipt: fields=%v", fields)
		}
	})

	t.Run("ports_permissions_sqlite_and_config_are_wired", func(t *testing.T) {
		portsDirectory := filepath.Join(repositoryRoot, "internal", "ports")
		v14RequireProductionFields(t, portsDirectory, "AgentLaunchRequest", []string{"BudgetDemand", "SecurityCriticality", "ReasoningEffort"})
		v14RequireProductionFields(t, portsDirectory, "AgentLaunchReceipt", []string{"ReceiptRef"})
		v14RequireProductionFields(t, portsDirectory, "AgentObservation", []string{"Usage"})
		identitySource := v15ReadProductionGo(t, filepath.Join(repositoryRoot, "internal", "identity"))
		for _, permission := range []string{`"budgets.manage"`, `"effects.approve"`} {
			if !strings.Contains(identitySource, permission) {
				t.Errorf("V15_RED identity permission %s missing", permission)
			}
		}
		migrationPath := filepath.Join(repositoryRoot, "internal", "adapters", "state", "sqlite", "migrations", "010_budgets_effects.sql")
		migration, err := os.ReadFile(migrationPath)
		if err != nil {
			t.Errorf("V15_RED SQLite migration 010 missing: %v", err)
		} else {
			lower := strings.ToLower(string(migration))
			for _, required := range []string{"budget", "reservation", "fairness", "effect_intent", "effect_approval", "effect_attempt", "effect_receipt", "outbox"} {
				if !strings.Contains(lower, required) {
					t.Errorf("V15_RED migration lacks %s", required)
				}
			}
		}
		registry, err := os.ReadFile(filepath.Join(repositoryRoot, "config", "registry.json"))
		if err != nil {
			t.Fatal(err)
		}
		registryText := string(registry)
		if !strings.Contains(registryText, `"runtime.codex.max_concurrent_executions"`) ||
			!strings.Contains(registryText, `"default": 70`) {
			t.Error("V15 Codex parent-per-cycle default 70 is not canonical")
		}
		if !strings.Contains(registryText, `"scheduler.max_children_per_parent"`) ||
			!strings.Contains(registryText, `"default": 6`) {
			t.Error("V15_RED child fanout default 6 is not canonical")
		}
	})

	t.Run("behavior_gates_are_executable", func(t *testing.T) {
		tests := v13ReadGoTests(t,
			filepath.Join(repositoryRoot, "internal", "governance"),
			filepath.Join(repositoryRoot, "internal", "goal"),
			filepath.Join(repositoryRoot, "internal", "application"),
			filepath.Join(repositoryRoot, "internal", "ports"),
			filepath.Join(repositoryRoot, "internal", "adapters", "agent", "fake"),
			filepath.Join(repositoryRoot, "internal", "adapters", "agent", "codex"),
			filepath.Join(repositoryRoot, "internal", "adapters", "state", "sqlite"),
			filepath.Join(repositoryRoot, "internal", "bootstrap"),
		)
		for _, name := range fixture.RequiredBehaviorTests {
			if !strings.Contains(tests, "func "+name+"(") {
				t.Errorf("V15_RED executable behavior test missing: %s", name)
			}
		}
	})
}

func v15AssertFixtureHeader(t *testing.T, repositoryRoot string, fixture v15Fixture) {
	t.Helper()
	wantOwned := []string{"GOV-15", "STG-09", "ORC-08", "ORC-09", "ORC-10", "ORC-11", "EVD-03", "EVD-14"}
	if fixture.SchemaVersion != 1 || fixture.ReceiptSchemaVersion != 3 ||
		fixture.ContractID != "AC-V15-BUDGETS-EFFECTS" ||
		fixture.TrustedBaseGitCommitOID != v15ContractBaseGitCommitOID ||
		fixture.ProductDeltaBaseGitCommitOID != v15ContractBaseGitCommitOID ||
		fixture.ProductDeltaSealedGitCommitOID != v15ProductDeltaSealedGitCommitOID ||
		fixture.OutputPath != "product/evidence/v15_budgets_effects.output.txt" ||
		fixture.ReceiptPath != "product/evidence/v15_budgets_effects.json" ||
		!reflect.DeepEqual(fixture.OwnedCapabilityIDs, wantOwned) ||
		!reflect.DeepEqual(fixture.DependencyVerticals, []string{"atomic_state_outbox", "identity_projects_rbac", "director_lease", "controls"}) ||
		!reflect.DeepEqual(fixture.BudgetDimensions, []string{"tokens", "money_micros", "active_time_ns", "process_slots", "disk_bytes"}) ||
		!reflect.DeepEqual(fixture.BudgetScopes, []string{"deployment", "project", "goal"}) ||
		!reflect.DeepEqual(fixture.RiskLevels, []string{"normal", "sensitive", "critical"}) ||
		!reflect.DeepEqual(fixture.EffectFacts, []string{"intent", "approval", "attempt", "receipt"}) ||
		fixture.DefaultParentLaunchesPerCycle != 70 || fixture.DefaultMaxChildrenPerParent != 6 ||
		len(fixture.RequiredBehaviorTests) == 0 || len(fixture.DeferredSurfaces) == 0 {
		t.Fatalf("invalid V15 fixture header: %+v", fixture)
	}
	if len(fixture.ExecutionArgv) != 3 || fixture.ExecutionArgv[0] != "sh" ||
		fixture.ExecutionArgv[1] != "-c" || fixture.ExecutionArgv[2] == "" ||
		fixture.Command != "sh -c '"+fixture.ExecutionArgv[2]+"'" {
		t.Fatal("invalid V15 execution argv")
	}
	if parsed, err := time.Parse(time.RFC3339Nano, fixture.Scenario.BaseTime); err != nil || parsed.IsZero() ||
		fixture.Scenario.Projects != 2 || fixture.Scenario.GoalsPerProject != 2 ||
		fixture.Scenario.ConcurrentClaims < 100 || fixture.Scenario.GlobalProcessSlots != 3 ||
		fixture.Scenario.ProjectProcessSlots != 2 || fixture.Scenario.GoalProcessSlots != 1 ||
		len(fixture.Scenario.EffectCrashFrontiers) != 3 {
		t.Fatalf("invalid V15 scenario: %+v", fixture.Scenario)
	}
	if root, err := filepath.Abs(repositoryRoot); err != nil || root == "" {
		t.Fatal("repository root unavailable")
	}
}

func v15ReadProductionGo(t *testing.T, directories ...string) string {
	t.Helper()
	var result strings.Builder
	for _, directory := range directories {
		entries, err := os.ReadDir(directory)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
				continue
			}
			content, err := os.ReadFile(filepath.Join(directory, entry.Name()))
			if err != nil {
				t.Fatal(err)
			}
			result.Write(content)
		}
	}
	return result.String()
}

func v15AssertSimplicityBudget(t *testing.T, numstat []byte) {
	t.Helper()
	type limit struct{ net, max int }
	limits := map[string]*limit{
		"core": {max: 2200}, "adapters": {max: 2400}, "migration": {max: 650}, "tests": {max: 6500},
	}
	for _, row := range strings.Split(strings.TrimSpace(string(numstat)), "\n") {
		if row == "" {
			continue
		}
		fields := strings.SplitN(row, "\t", 3)
		if len(fields) != 3 || fields[0] == "-" || fields[1] == "-" {
			t.Fatalf("V15 simplicity budget cannot classify %q", row)
		}
		added, addErr := strconv.Atoi(fields[0])
		deleted, deleteErr := strconv.Atoi(fields[1])
		if addErr != nil || deleteErr != nil {
			t.Fatalf("V15 invalid numstat %q", row)
		}
		if class := v15SimplicityClass(fields[2]); class != "" {
			limits[class].net += added - deleted
		}
	}
	for class, budget := range limits {
		if budget.net > budget.max {
			t.Errorf("V15 simplicity budget %s net LOC=%d, max=%d", class, budget.net, budget.max)
		}
	}
	if limits["core"].net+limits["adapters"].net+limits["migration"].net > 5250 {
		t.Errorf("V15 production net LOC exceeds 5250")
	}
}

func v15SimplicityClass(relative string) string {
	switch {
	case strings.HasSuffix(relative, "_test.go"), strings.HasPrefix(relative, "acceptance/"):
		return "tests"
	case strings.HasSuffix(relative, ".sql"):
		return "migration"
	case strings.HasPrefix(relative, "internal/adapters/"), strings.HasPrefix(relative, "internal/bootstrap/"),
		strings.HasPrefix(relative, "internal/config/"), strings.HasPrefix(relative, "config/"):
		if strings.HasSuffix(relative, ".go") || strings.HasSuffix(relative, ".json") {
			return "adapters"
		}
	case strings.HasPrefix(relative, "internal/governance/"), strings.HasPrefix(relative, "internal/goal/"),
		strings.HasPrefix(relative, "internal/application/"), strings.HasPrefix(relative, "internal/identity/"),
		strings.HasPrefix(relative, "internal/ports/"):
		if strings.HasSuffix(relative, ".go") {
			return "core"
		}
	}
	return ""
}
