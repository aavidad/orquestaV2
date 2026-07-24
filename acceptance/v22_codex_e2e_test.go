package acceptance_test

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

const (
	v22FixturePath              = "acceptance/fixtures/v22_codex_e2e.json"
	v22ContractBaseGitCommitOID = "665ef446a32a7f0512255640fa99b2e7edd9f29d"
)

type v22Fixture struct {
	SchemaVersion                  int                 `json:"schema_version"`
	ReceiptSchemaVersion           int                 `json:"receipt_schema_version"`
	ContractID                     string              `json:"contract_id"`
	TrustedBaseGitCommitOID        string              `json:"trusted_base_git_commit_oid"`
	ProductDeltaBaseGitCommitOID   string              `json:"product_delta_base_git_commit_oid"`
	ProductDeltaSealedGitCommitOID string              `json:"product_delta_sealed_git_commit_oid"`
	SealStatus                     string              `json:"seal_status"`
	SealNote                       string              `json:"seal_note"`
	Command                        string              `json:"command"`
	ExecutionArgv                  []string            `json:"execution_argv"`
	OutputPath                     string              `json:"output_path"`
	ReceiptPath                    string              `json:"receipt_path"`
	CandidateSubjects              []string            `json:"candidate_subjects"`
	ImplementationStatus           string              `json:"implementation_status"`
	LifecycleGate                  string              `json:"lifecycle_gate"`
	PreflightStatus                string              `json:"preflight_status"`
	PreflightNote                  string              `json:"preflight_note"`
	BlockingDependency             string              `json:"blocking_dependency_vertical"`
	DependencyReceiptPath          string              `json:"dependency_receipt_path"`
	DependencyActivation           v22DependencyGate   `json:"dependency_activation_contract"`
	OwnedCapabilityIDs             []string            `json:"owned_capability_ids"`
	DependencyVerticals            []string            `json:"dependency_verticals"`
	PublicInterface                string              `json:"public_interface"`
	ExecutionMailboxCommands       []string            `json:"execution_bound_mailbox_commands"`
	RequiredBehaviorTests          []string            `json:"required_behavior_tests"`
	RequiredRealE2ETests           []string            `json:"required_real_e2e_tests"`
	RequiredProductClosures        []string            `json:"required_product_closures"`
	GoalScenarios                  []v22GoalScenario   `json:"goal_scenarios"`
	ServiceIdentityAssertions      []string            `json:"service_identity_assertions"`
	FinalConsistencyAssertions     []string            `json:"final_consistency_assertions"`
	SimplicityBudget               v22SimplicityBudget `json:"simplicity_budget"`
	DeferredOwnership              []string            `json:"deferred_ownership"`
	KnownBugRefs                   []string            `json:"known_bug_refs"`
	PSEContract                    v22PSEContract      `json:"pse_contract"`
}

type v22DependencyGate struct {
	Contract            string `json:"contract"`
	Result              string `json:"result"`
	SourceWorktreeState string `json:"source_worktree_state"`
	ProductOID          string `json:"product_oid"`
	SealedOID           string `json:"sealed_oid"`
	EvidenceOID         string `json:"evidence_oid"`
	IntegrationHeadOID  string `json:"integration_head_oid"`
}

type v22GoalScenario struct {
	ID               string `json:"id"`
	Purpose          string `json:"purpose"`
	ExpectedTerminal string `json:"expected_terminal"`
}

type v22SimplicityBudget struct {
	ProductDeltaLOCMax        int `json:"product_delta_loc_max"`
	TestAndHarnessDeltaLOCMax int `json:"test_and_harness_delta_loc_max"`
	NewTopLevelStoreMax       int `json:"new_top_level_store_max"`
	LifecycleWriterMax        int `json:"lifecycle_writer_max"`
	SchedulerMax              int `json:"scheduler_max"`
	CommandRegistryMax        int `json:"command_registry_max"`
	ActiveProviderAdapterMax  int `json:"active_provider_adapter_max"`
	ProductFileLOCMax         int `json:"product_file_loc_max"`
}

type v22PSEContract struct {
	P string `json:"P"`
	S string `json:"S"`
	E string `json:"E"`
}

func TestV22PreflightContractIsStructurallyValid(t *testing.T) {
	root := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v22Fixture](t, filepath.Join(root, v22FixturePath))
	v22AssertFixture(t, root, fixture)
	v22AssertRoadmapBoundary(t, root, fixture)
}

func TestAcceptanceV22CodexE2E(t *testing.T) {
	root := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v22Fixture](t, filepath.Join(root, v22FixturePath))
	v22AssertFixture(t, root, fixture)
	v22AssertRoadmapBoundary(t, root, fixture)

	missing := v22MissingRequiredProductTests(t, root, fixture)
	if len(missing) != 0 {
		t.Fatalf("V22_PRODUCT_PENDING: executable product/E2E gates are absent: %v", missing)
	}
	v22AssertRealE2ETestSources(t, root, fixture)
}

func v22AssertFixture(t *testing.T, root string, fixture v22Fixture) {
	t.Helper()
	if fixture.SchemaVersion != 1 || fixture.ReceiptSchemaVersion != 3 ||
		fixture.ContractID != "AC-V22-CODEX-E2E" ||
		fixture.TrustedBaseGitCommitOID != v22ContractBaseGitCommitOID ||
		fixture.ProductDeltaBaseGitCommitOID != v22ContractBaseGitCommitOID ||
		fixture.Command == "" || len(fixture.ExecutionArgv) != 3 ||
		fixture.ExecutionArgv[0] != "sh" || fixture.ExecutionArgv[1] != "-c" ||
		fixture.Command != "sh -c '"+fixture.ExecutionArgv[2]+"'" ||
		fixture.OutputPath != "product/evidence/v22_codex_e2e.output.txt" ||
		fixture.ReceiptPath != "product/evidence/v22_codex_e2e.json" ||
		fixture.SealNote == "" || fixture.PreflightNote == "" ||
		fixture.PSEContract.P == "" || fixture.PSEContract.S == "" || fixture.PSEContract.E == "" {
		t.Fatalf("invalid V22 identity/PSE envelope: %+v", fixture)
	}
	if fixture.BlockingDependency != "none" ||
		fixture.DependencyReceiptPath != "product/evidence/v21_i18n.json" ||
		fixture.DependencyActivation != (v22DependencyGate{
			Contract: "AC-V21-I18N", Result: "PASS", SourceWorktreeState: "detached_clean",
			ProductOID:         "e25f95bbd72b26adb6e81c9ca759582b0734cfb4",
			SealedOID:          "216e61e80c1b46ae049fbf2c5caefb1a69e5f7b3",
			EvidenceOID:        "665ef446a32a7f0512255640fa99b2e7edd9f29d",
			IntegrationHeadOID: v22ContractBaseGitCommitOID,
		}) {
		t.Fatalf("invalid V22 dependency activation: %+v", fixture.DependencyActivation)
	}
	v22AssertExactSet(t, "owned capabilities", fixture.OwnedCapabilityIDs,
		[]string{"AGT-01", "AGT-03", "EVD-09", "STG-11"})
	v22AssertExactSet(t, "dependency verticals", fixture.DependencyVerticals,
		[]string{"command_registry", "council", "i18n", "recovery_backup"})
	if fixture.PublicInterface != "mcp" {
		t.Fatalf("V22 public interface=%q want mcp", fixture.PublicInterface)
	}
	v22AssertExactSet(t, "execution-bound mailbox commands", fixture.ExecutionMailboxCommands, []string{
		"orquesta.mailbox.acknowledge", "orquesta.mailbox.admit",
		"orquesta.mailbox.block", "orquesta.mailbox.claim",
		"orquesta.mailbox.consume", "orquesta.mailbox.get",
		"orquesta.mailbox.list", "orquesta.mailbox.mark_delivered",
	})
	v22AssertExactSet(t, "required behavior tests", fixture.RequiredBehaviorTests, []string{
		"TestCodexChildDeliveryUsesSameExecutionServicePrincipalAfterArtifactPersistence",
		"TestCodexRuntimeUsesCatalogOwnedPrompt",
		"TestExecutionServicePrincipalExactScopeRevocationAndRestart",
		"TestProductionBuildCreatesDurableExecutionAuthorityResolver",
		"TestV22RealCodexFourGoalsSelectiveStopCrashRestartAndCloseThroughMCP",
		"TestV22RealCodexNoTerminalContradictionAndNoOwnedProcess",
	})
	v22AssertExactSet(t, "required real E2E tests", fixture.RequiredRealE2ETests, []string{
		"TestV22RealCodexFourGoalsSelectiveStopCrashRestartAndCloseThroughMCP",
		"TestV22RealCodexNoTerminalContradictionAndNoOwnedProcess",
	})
	v22AssertExactSet(t, "required product closures", fixture.RequiredProductClosures, []string{
		"catalog_owned_codex_prompt", "causal_post_artifact_delivery",
		"durable_execution_authority_binding",
		"execution_service_credential_lifecycle", "four_goal_real_codex_mcp_e2e",
		"owned_process_census", "real_backup_crash_restart",
	})
	if !reflect.DeepEqual(fixture.GoalScenarios, []v22GoalScenario{
		{ID: "A", Purpose: "parent_child_dag_and_execution_bound_mailbox", ExpectedTerminal: "completed"},
		{ID: "B", Purpose: "selective_stop_without_cross_goal_damage", ExpectedTerminal: "controlled_stop"},
		{ID: "C", Purpose: "workspace_tests_review_integration_and_close", ExpectedTerminal: "completed"},
		{ID: "D", Purpose: "backup_crash_restart_exactly_once_resume_and_close", ExpectedTerminal: "completed"},
	}) {
		t.Fatalf("invalid V22 A/B/C/D scenario matrix: %+v", fixture.GoalScenarios)
	}
	if len(fixture.ServiceIdentityAssertions) != 8 || len(fixture.FinalConsistencyAssertions) != 6 ||
		len(fixture.DeferredOwnership) != 5 ||
		!reflect.DeepEqual(fixture.KnownBugRefs,
			[]string{
				"BUG-ORQ-20260723-366", "BUG-ORQ-20260723-390", "BUG-ORQ-20260723-391",
				"BUG-ORQ-20260723-394", "BUG-ORQ-20260723-395", "BUG-ORQ-20260723-396",
			}) {
		t.Fatalf("invalid V22 closure/deferment ratchets: %+v", fixture)
	}
	if fixture.SimplicityBudget != (v22SimplicityBudget{
		ProductDeltaLOCMax: 3000, TestAndHarnessDeltaLOCMax: 4200,
		NewTopLevelStoreMax: 0, LifecycleWriterMax: 1, SchedulerMax: 1,
		CommandRegistryMax: 1, ActiveProviderAdapterMax: 1, ProductFileLOCMax: 500,
	}) {
		t.Fatalf("invalid V22 simplicity budget: %+v", fixture.SimplicityBudget)
	}
	switch fixture.ImplementationStatus {
	case "development_unsealed":
		if fixture.ProductDeltaSealedGitCommitOID != "" ||
			fixture.SealStatus != "development_unsealed_candidate_pending" ||
			fixture.LifecycleGate != "V22_DEVELOPMENT_UNSEALED" ||
			fixture.PreflightStatus != "contract_ready_product_pending" ||
			len(fixture.CandidateSubjects) != 0 {
			t.Fatalf("invalid V22 red lifecycle: %+v", fixture)
		}
		if v22ExecutionEvidencePresent(t, root, fixture) {
			t.Fatal("V22 development state contains execution evidence")
		}
	case "implemented_unsealed":
		if fixture.ProductDeltaSealedGitCommitOID != "" ||
			fixture.SealStatus != "p_implemented_unsealed_pending_seal" ||
			fixture.LifecycleGate != "V22_IMPLEMENTED_UNSEALED" ||
			fixture.PreflightStatus != "integrated_implementation_complete" {
			t.Fatalf("invalid V22 P lifecycle: %+v", fixture)
		}
		v22AssertCandidateSubjects(t, root, fixture, "")
		if v22ExecutionEvidencePresent(t, root, fixture) {
			t.Fatal("V22 P contains execution evidence")
		}
	case "sealed_unexecuted":
		if fixture.ProductDeltaSealedGitCommitOID == "" ||
			fixture.SealStatus != "s_product_delta_sealed_pending_execution" ||
			fixture.LifecycleGate != "V22_SEALED_UNEXECUTED" ||
			len(fixture.CandidateSubjects) == 0 ||
			!sort.StringsAreSorted(fixture.CandidateSubjects) {
			t.Fatalf("invalid V22 sealed lifecycle: %+v", fixture)
		}
		v22AssertCandidateSubjects(t, root, fixture, fixture.ProductDeltaSealedGitCommitOID)
		if !v22ExecutionEvidencePresent(t, root, fixture) {
			head, err := evidenceGit(root, "rev-parse", "HEAD")
			if err != nil {
				t.Fatal(err)
			}
			v22AssertExactSeal(t, root, fixture, strings.TrimSpace(string(head)))
		}
	default:
		t.Fatalf("invalid V22 implementation status %q", fixture.ImplementationStatus)
	}
	if _, err := evidenceGitCanonicalCommit(root, fixture.TrustedBaseGitCommitOID); err != nil {
		t.Fatalf("invalid V22 trusted base: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(fixture.DependencyReceiptPath))); err != nil {
		t.Fatalf("V22 dependency receipt is absent: %v", err)
	}
	evidenceAssertReceiptV3(t, root, evidenceReceiptV3Expectation{
		Contract:                "AC-V21-I18N",
		FixturePath:             "acceptance/fixtures/v21_i18n.json",
		ReceiptPath:             fixture.DependencyReceiptPath,
		ExecutedNotBefore:       "2026-07-23T00:00:00Z",
		TrustedBaseGitCommitOID: "8063d8ce3ed75dd5ec92108ef6e7e73ee8e11cb8",
	})
}

func TestV22ReceiptV3AndPSESealAreExact(t *testing.T) {
	root := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v22Fixture](t, filepath.Join(root, v22FixturePath))
	if !v22ExecutionEvidencePresent(t, root, fixture) {
		return
	}
	if fixture.ImplementationStatus != "sealed_unexecuted" {
		t.Fatalf("V22 evidence exists outside S lifecycle: %s", fixture.ImplementationStatus)
	}
	evidenceAssertReceiptV3(t, root, evidenceReceiptV3Expectation{
		Contract: "AC-V22-CODEX-E2E", FixturePath: v22FixturePath,
		ReceiptPath: fixture.ReceiptPath, ExecutedNotBefore: "2026-07-24T00:00:00Z",
		TrustedBaseGitCommitOID: v22ContractBaseGitCommitOID,
	})
	receipt := evidenceDecodeStrictJSON[evidenceReceiptV3](t,
		filepath.Join(root, filepath.FromSlash(fixture.ReceiptPath)))
	v22AssertExactSeal(t, root, fixture, receipt.SealedSource.GitCommitOID)
}

func v22AssertCandidateSubjects(t *testing.T, root string, fixture v22Fixture, sealed string) {
	t.Helper()
	if len(fixture.CandidateSubjects) == 0 || !sort.StringsAreSorted(fixture.CandidateSubjects) {
		t.Fatalf("V22 candidate subjects are empty or unsorted")
	}
	if err := evidenceValidateCandidateSubjects(
		fixture.CandidateSubjects, fixture.ReceiptPath, fixture.OutputPath,
	); err != nil {
		t.Fatal(err)
	}
	if sealed != "" {
		evidenceAssertCandidateDelta(t, "V22", root, v22ContractBaseGitCommitOID,
			sealed, fixture.CandidateSubjects)
		return
	}
	if got := v22CurrentCandidateSubjects(t, root); !reflect.DeepEqual(got, fixture.CandidateSubjects) {
		t.Fatalf("V22 P subjects differ from exact base..P delta:\ngot=%v\nwant=%v", got, fixture.CandidateSubjects)
	}
}

func v22CurrentCandidateSubjects(t *testing.T, root string) []string {
	t.Helper()
	unique := map[string]struct{}{}
	for _, arguments := range [][]string{
		{"diff", "--name-only", v22ContractBaseGitCommitOID, "--"},
		{"ls-files", "--others", "--exclude-standard"},
	} {
		output, err := evidenceGit(root, arguments...)
		if err != nil {
			t.Fatal(err)
		}
		for _, relative := range strings.Fields(string(output)) {
			unique[filepath.ToSlash(relative)] = struct{}{}
		}
	}
	result := make([]string, 0, len(unique))
	for relative := range unique {
		result = append(result, relative)
	}
	sort.Strings(result)
	return result
}

func v22AssertExactSeal(t *testing.T, root string, fixture v22Fixture, sealed string) {
	t.Helper()
	line, err := evidenceGit(root, "rev-list", "--parents", "-n", "1", sealed)
	if err != nil {
		t.Fatal(err)
	}
	identity := strings.Fields(string(line))
	if len(identity) != 2 || identity[0] != sealed ||
		identity[1] != fixture.ProductDeltaSealedGitCommitOID {
		t.Fatalf("V22 S is not an exact child of P: %v", identity)
	}
	delta, err := evidenceGit(root, "diff", "--name-only",
		fixture.ProductDeltaSealedGitCommitOID, sealed, "--")
	if err != nil || strings.TrimSpace(string(delta)) != v22FixturePath {
		t.Fatalf("V22 P->S delta=%q err=%v, want only fixture", delta, err)
	}
	for _, relative := range []string{fixture.ReceiptPath, fixture.OutputPath} {
		inside, err := evidenceGit(root, "ls-tree", "-r", "--name-only", sealed, "--", relative)
		if err != nil || strings.TrimSpace(string(inside)) != "" {
			t.Fatalf("V22 evidence %q is inside S: output=%q err=%v", relative, inside, err)
		}
		if info, err := os.Lstat(filepath.Join(root, filepath.FromSlash(relative))); err == nil &&
			!info.Mode().IsRegular() {
			t.Fatalf("V22 evidence %q is not regular", relative)
		}
	}
}

func v22ExecutionEvidencePresent(t *testing.T, root string, fixture v22Fixture) bool {
	t.Helper()
	present := 0
	for _, relative := range []string{fixture.ReceiptPath, fixture.OutputPath} {
		if info, err := os.Lstat(filepath.Join(root, filepath.FromSlash(relative))); err == nil && info.Mode().IsRegular() {
			present++
		} else if err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
	}
	if present == 1 {
		t.Fatal("V22 receipt and output must appear atomically")
	}
	return present == 2
}

func v22AssertExactSet(t *testing.T, label string, got, want []string) {
	t.Helper()
	gotCopy, wantCopy := append([]string(nil), got...), append([]string(nil), want...)
	sort.Strings(gotCopy)
	sort.Strings(wantCopy)
	if !reflect.DeepEqual(gotCopy, wantCopy) || !sort.StringsAreSorted(got) {
		t.Fatalf("V22 %s=%v want sorted exact %v", label, got, wantCopy)
	}
}
