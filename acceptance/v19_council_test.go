package acceptance_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

const v19FixturePath = "fixtures/v19_council.json"

const (
	v19SealManifestPath = "product/evidence/v19_council_seal.json"
	v19ReceiptPath      = "product/evidence/v19_council.json"
	v19OutputPath       = "product/evidence/v19_council.output.txt"
)

type v19Dependency struct {
	Vertical                 string `json:"vertical"`
	Contract                 string `json:"contract"`
	ReceiptPath              string `json:"receipt_path"`
	ReceiptResult            string `json:"receipt_result"`
	SealedSourceGitCommitOID string `json:"sealed_source_git_commit_oid"`
	Status                   string `json:"status"`
}

type v19PolicyContract struct {
	Policy, Opener, Behavior, PromotionGate string
}

func (value *v19PolicyContract) UnmarshalJSON(data []byte) error {
	type wire struct {
		Policy        string `json:"policy"`
		Opener        string `json:"opener"`
		Behavior      string `json:"behavior"`
		PromotionGate string `json:"promotion_gate"`
	}
	var decoded wire
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*value = v19PolicyContract(decoded)
	return nil
}

type v19Fixture struct {
	SchemaVersion        int                 `json:"schema_version"`
	ContractID           string              `json:"contract_id"`
	Vertical             string              `json:"vertical"`
	OwnedCapabilityIDs   []string            `json:"owned_capability_ids"`
	Dependency           v19Dependency       `json:"dependency"`
	ImplementationStatus string              `json:"implementation_status"`
	PSourcePaths         []string            `json:"p_source_paths"`
	PTestPaths           []string            `json:"p_test_paths"`
	Policies             []string            `json:"policies"`
	PolicyContracts      []v19PolicyContract `json:"policy_contracts"`
	PolicySourceContract struct {
		Input        string `json:"input"`
		DomainType   string `json:"domain_type"`
		DurablePath  string `json:"durable_path"`
		WriterRule   string `json:"writer_rule"`
		ReadOnlyRule string `json:"read_only_rule"`
		ReworkRule   string `json:"rework_rule"`
	} `json:"policy_source_contract"`
	SubjectContract struct {
		DigestFields      []string `json:"digest_fields"`
		PersistedBindings []string `json:"persisted_bindings"`
		DigestEncoding    string   `json:"digest_encoding"`
		V18Revalidation   string   `json:"v18_revalidation"`
	} `json:"subject_contract"`
	Roles        []string `json:"roles"`
	RoleContract struct {
		RequiredLaunches                        int      `json:"required_launches"`
		RequiredBallots                         int      `json:"required_ballots"`
		DistinctFromEachOther                   bool     `json:"distinct_from_each_other"`
		DistinctFromV18AuthorPrimaryAdversarial bool     `json:"distinct_from_v18_author_primary_adversarial"`
		FactsRequireLaunchReceipt               bool     `json:"facts_require_launch_receipt"`
		ArtifactSchemas                         []string `json:"artifact_schemas"`
		ArtifactRule                            string   `json:"artifact_rule"`
	} `json:"role_contract"`
	Ballots          []string `json:"ballots"`
	DecisionContract struct {
		NormalQuorum                 int      `json:"normal_quorum"`
		Accepted                     string   `json:"accepted"`
		Rejected                     string   `json:"rejected"`
		NoConsensus                  string   `json:"no_consensus"`
		BlockedSecurity              string   `json:"blocked_security"`
		EarlyNormalDecisionForbidden bool     `json:"early_normal_decision_forbidden"`
		SecurityVetoDecisionTiming   string   `json:"security_veto_decision_timing"`
		Dissent                      string   `json:"dissent"`
		Integration                  string   `json:"integration"`
		ReplanCausality              string   `json:"replan_causality"`
		RequiredTestOutcomes         []string `json:"required_test_outcomes"`
	} `json:"decision_contract"`
	ResolutionContract struct {
		CommonBinding          string   `json:"common_binding"`
		AcceptedRoundBinding   []string `json:"accepted_round_binding"`
		SkipBinding            []string `json:"skip_binding"`
		MutualExclusion        string   `json:"mutual_exclusion"`
		TargetDigest           string   `json:"target_digest"`
		SyntheticSkipForbidden bool     `json:"synthetic_skip_decision_forbidden"`
	} `json:"resolution_contract"`
	AuthorityContract struct {
		RequiredOpenPermission string   `json:"required_open_permission"`
		SkipPermission         string   `json:"skip_permission"`
		SkipRoles              []string `json:"skip_roles"`
		SkipPrincipalKind      string   `json:"skip_principal_kind"`
		SkipRequiredFields     []string `json:"skip_required_fields"`
		SkipTemporalBoundary   string   `json:"skip_temporal_boundary"`
	} `json:"authority_contract"`
	ExecutionContract struct {
		Purposes               []string `json:"purposes"`
		ActionKinds            []string `json:"action_kinds"`
		NewActionKindForbidden bool     `json:"new_action_kind_forbidden"`
		ACKOrTextIsEvidence    bool     `json:"ack_or_text_is_evidence"`
	} `json:"execution_contract"`
	PersistenceContract struct {
		Authority string   `json:"authority"`
		Atomicity string   `json:"atomicity"`
		Replay    string   `json:"replay"`
		Recovery  string   `json:"recovery"`
		Facts     []string `json:"facts"`
	} `json:"persistence_contract"`
	MigrationContract struct {
		Version                              int    `json:"version"`
		SourceVersion                        int    `json:"source_version"`
		LiveV18CandidateWithoutDurablePolicy string `json:"live_v18_candidate_without_durable_policy"`
		CompletedV18Records                  string `json:"completed_v18_records"`
		Backfill                             string `json:"backfill"`
	} `json:"migration_contract"`
	ForbiddenPrivateAuthorities []string `json:"forbidden_private_authorities"`
	Budgets                     struct {
		Stage                string `json:"stage"`
		ProductLOCMax        int    `json:"product_loc_max"`
		DomainLOCMax         int    `json:"domain_loc_max"`
		ApplicationLOCMax    int    `json:"application_loc_max"`
		SQLiteRecoveryLOCMax int    `json:"sqlite_recovery_loc_max"`
		BootstrapLOCMax      int    `json:"bootstrap_loc_max"`
		FileLOCMax           int    `json:"file_loc_max"`
		MigrationException   string `json:"migration_exception"`
		QualityAccreditation string `json:"quality_accreditation"`
		SealRequirement      string `json:"seal_requirement"`
		PostV22Debt          string `json:"post_v22_debt"`
	} `json:"budgets"`
	Invariants []string `json:"invariants"`
	E2ECases   []struct {
		Policy          string   `json:"policy"`
		IsolatedRuntime bool     `json:"isolated_runtime"`
		Assertions      []string `json:"assertions"`
	} `json:"e2e_cases"`
	PSEContract   map[string]string `json:"pse_contract"`
	LifecycleGate string            `json:"lifecycle_gate"`
}

func TestAcceptanceV19Council(t *testing.T) {
	fixture := loadV19Fixture(t)
	if fixture.SchemaVersion != 2 || fixture.ContractID != "AC-V19-COUNCIL" || fixture.Vertical != "council" ||
		fixture.ImplementationStatus != "implemented_unsealed" || fixture.LifecycleGate != "V19_IMPLEMENTED_UNSEALED" {
		t.Fatalf("invalid V19 implemented-unsealed identity: %+v", fixture)
	}
	wantCapabilities := []string{"EVD-07", "GOV-11", "GOV-13", "GOV-14", "STG-06", "STG-08"}
	assertV19Strings(t, "capabilities", sortedV19(fixture.OwnedCapabilityIDs), wantCapabilities)
	assertV19Dependency(t, fixture.Dependency)
	assertV19Policies(t, fixture)
	assertV19CouncilShape(t, fixture)
	assertV19SafetyAndPersistence(t, fixture)
	assertV19ProductPresence(t, fixture)
	assertV19E2E(t, fixture)
}

// TestV19PSEImplementedUnsealedLifecycle accepts P only: product and tests are
// present, but neither a seal nor an execution receipt exists.
func TestV19PSEImplementedUnsealedLifecycle(t *testing.T) {
	fixture := loadV19Fixture(t)
	if fixture.ImplementationStatus != "implemented_unsealed" || fixture.LifecycleGate != "V19_IMPLEMENTED_UNSEALED" {
		t.Fatalf("V19 is not exactly P: %+v", fixture)
	}
	for _, path := range []string{v19SealManifestPath, v19ReceiptPath, v19OutputPath} {
		if _, err := os.Lstat(filepath.Join("..", filepath.FromSlash(path))); err == nil {
			t.Fatalf("V19 pre-P evidence exists: %s", path)
		} else if !os.IsNotExist(err) {
			t.Fatal(err)
		}
	}
	matches, err := filepath.Glob("../product/evidence/v19_council*")
	if err != nil || len(matches) != 0 {
		t.Fatalf("V19 pre-P evidence paths=%v err=%v", matches, err)
	}
	command := v19E2EValidationShellBody()
	for _, value := range []string{
		"-tags=v18_real_e2e,v19_real_e2e",
		"TestRealGitSQLiteFilesystemCASBubblewrapIndependentReviewsEndToEnd",
		"TestV19CouncilAutoSQLiteFilesystemRestartE2E",
		"TestV19CouncilRequiredStaleOpenAndVetoWaitsThreeE2E",
		"TestV19CouncilSkipHumanReplayAndIntegrationE2E",
		"\"Action\":\"run\"", "\"Action\":\"pass\"",
	} {
		if !strings.Contains(command, value) {
			t.Fatalf("V19 E2E command lacks %q", value)
		}
	}
}

func assertV19ProductPresence(t *testing.T, fixture v19Fixture) {
	t.Helper()
	wantSources := []string{
		"internal/council/council.go",
		"internal/application/council_commands.go",
		"internal/adapters/state/sqlite/migrations/014_council.sql",
		"internal/bootstrap/v19_council_auto_e2e_linux_test.go",
	}
	wantTests := []string{
		"internal/application/council_commands_replay_test.go",
		"internal/adapters/state/sqlite/v19_council_test.go",
		"internal/bootstrap/v19_council_required_e2e_linux_test.go",
	}
	assertV19Strings(t, "P source paths", fixture.PSourcePaths, wantSources)
	assertV19Strings(t, "P test paths", fixture.PTestPaths, wantTests)
	for _, path := range append(append([]string(nil), fixture.PSourcePaths...), fixture.PTestPaths...) {
		if info, err := os.Stat(filepath.Join("..", filepath.FromSlash(path))); err != nil || info.IsDir() {
			t.Fatalf("V19 P path=%q info=%v err=%v", path, info, err)
		}
	}
}

func v19E2EValidationShellBody() string {
	return `e2e_events=$(CGO_ENABLED=0 go test -mod=vendor -tags=v18_real_e2e,v19_real_e2e -json -count=1 -timeout=240s ./internal/bootstrap -run "^(TestRealGitSQLiteFilesystemCASBubblewrapIndependentReviewsEndToEnd|TestV19CouncilAutoSQLiteFilesystemRestartE2E|TestV19CouncilRequiredStaleOpenAndVetoWaitsThreeE2E|TestV19CouncilSkipHumanReplayAndIntegrationE2E)$" 2>&1); e2e_status=$?; printf '%s\n' "$e2e_events"; [ "$e2e_status" -eq 0 ] && for test_name in TestRealGitSQLiteFilesystemCASBubblewrapIndependentReviewsEndToEnd TestV19CouncilAutoSQLiteFilesystemRestartE2E TestV19CouncilRequiredStaleOpenAndVetoWaitsThreeE2E TestV19CouncilSkipHumanReplayAndIntegrationE2E; do printf '%s\n' "$e2e_events" | grep -F '"Action":"run"' | grep -F '"Package":"orquesta/internal/bootstrap"' | grep -F "\"Test\":\"$test_name\"" >/dev/null && printf '%s\n' "$e2e_events" | grep -F '"Action":"pass"' | grep -F '"Package":"orquesta/internal/bootstrap"' | grep -F "\"Test\":\"$test_name\"" >/dev/null || exit 1; done`
}

func assertV19Dependency(t *testing.T, dependency v19Dependency) {
	t.Helper()
	if dependency.Vertical != "independent_reviews" || dependency.Contract != "AC-V18-INDEPENDENT-REVIEWS" ||
		dependency.ReceiptPath != "product/evidence/v18_independent_reviews.json" || dependency.ReceiptResult != "PASS" ||
		dependency.SealedSourceGitCommitOID != "32ee17e407006d9e0aeb46557b1e160769dd4848" || dependency.Status != "verified_accredited" {
		t.Fatalf("invalid V18 dependency: %+v", dependency)
	}
	content, err := os.ReadFile("../" + dependency.ReceiptPath)
	if err != nil {
		t.Fatal(err)
	}
	var receipt struct {
		SchemaVersion int    `json:"schema_version"`
		Contract      string `json:"contract"`
		Result        string `json:"result"`
		SealedSource  struct {
			GitCommitOID string `json:"git_commit_oid"`
		} `json:"sealed_source"`
	}
	if json.Unmarshal(content, &receipt) != nil || receipt.SchemaVersion != 3 || receipt.Contract != dependency.Contract ||
		receipt.Result != dependency.ReceiptResult || receipt.SealedSource.GitCommitOID != dependency.SealedSourceGitCommitOID {
		t.Fatalf("V18 receipt does not match V19 dependency: %+v", receipt)
	}
}

func assertV19Policies(t *testing.T, fixture v19Fixture) {
	t.Helper()
	assertV19Strings(t, "policies", sortedV19(fixture.Policies), []string{"auto", "required", "skip_by_operator"})
	if len(fixture.PolicyContracts) != 3 {
		t.Fatalf("policy contracts=%d want=3", len(fixture.PolicyContracts))
	}
	openers, behaviors, gates := map[string]string{}, map[string]string{}, map[string]string{}
	for _, policy := range fixture.PolicyContracts {
		if policy.Policy == "" || policy.Opener == "" || policy.Behavior == "" || policy.PromotionGate == "" {
			t.Fatalf("incomplete policy: %+v", policy)
		}
		openers[policy.Policy] = policy.Opener
		behaviors[policy.Policy] = policy.Behavior
		gates[policy.Policy] = policy.PromotionGate
	}
	if openers["auto"] != "application_after_exact_v18_gate" ||
		openers["required"] != "director_with_live_goal_lease_and_fence" ||
		openers["skip_by_operator"] != "authorized_human_with_council_skip" {
		t.Fatalf("policy openers overlap or drift: %v", openers)
	}
	if behaviors["auto"] != "open once and schedule exactly proposer critic and arbiter" ||
		behaviors["required"] != "block integration until explicit governed open then schedule exactly proposer critic and arbiter" ||
		behaviors["skip_by_operator"] != "record exact skip before any council round fact or launch and schedule no council execution" ||
		gates["auto"] != "accepted_decision_only" || gates["required"] != "accepted_decision_only" ||
		gates["skip_by_operator"] != "valid_skip_plus_exact_v18_gate" {
		t.Fatalf("policy behavior/gate drift: behaviors=%v gates=%v", behaviors, gates)
	}
	source := fixture.PolicySourceContract
	if source.Input != "WorkItemSpec.CouncilPolicy" || source.DomainType != "council.Policy" ||
		source.DurablePath != "WorkItem_and_GoalSnapshot_before_author_launch" ||
		source.WriterRule != "required_for_every_work_item_with_non_empty_write_set" ||
		source.ReadOnlyRule != "empty_policy_allowed_only_without_change_or_council_subject" ||
		source.ReworkRule != "new_work_item_declares_policy_explicitly_and_never_infers_from_mutable_config_or_text" {
		t.Fatalf("council policy source is not durable and pre-launch: %+v", source)
	}
}

func assertV19CouncilShape(t *testing.T, fixture v19Fixture) {
	t.Helper()
	assertV19Strings(t, "subject digest", fixture.SubjectContract.DigestFields,
		[]string{"ProjectRef", "ReviewSubjectDigest", "ReviewGateDigest", "CouncilPolicy"})
	assertV19Strings(t, "subject bindings", fixture.SubjectContract.PersistedBindings,
		[]string{"GoalRef", "WorkItemRef", "ChangeSetRef", "SpecHash", "PlanGeneration", "WorkItemGeneration", "AppSpecGeneration"})
	if fixture.SubjectContract.DigestEncoding != "domain_separated_length_framed_sha256" ||
		fixture.SubjectContract.V18Revalidation != "rebuild exact author production provenance plus primary and adversarial approve gate at every council write and integration admission processing" {
		t.Fatalf("subject digest/revalidation drift: %+v", fixture.SubjectContract)
	}
	assertV19Strings(t, "roles", fixture.Roles, []string{"proposer", "critic", "arbiter"})
	assertV19Strings(t, "ballots", sortedV19(fixture.Ballots), []string{"abstain", "accept", "reject", "security_veto"})
	role := fixture.RoleContract
	if role.RequiredLaunches != 3 || role.RequiredBallots != 3 || !role.DistinctFromEachOther ||
		!role.DistinctFromV18AuthorPrimaryAdversarial || !role.FactsRequireLaunchReceipt ||
		!reflect.DeepEqual(role.ArtifactSchemas, []string{"orquesta.council.contribution.v1"}) ||
		role.ArtifactRule != "one strict envelope per role contains role body ballot and typed evidence; proposer derives proposal plus ballot critic derives critique plus ballot arbiter derives ballot; every derived fact references the same CAS artifact" {
		t.Fatalf("invalid council cardinality/independence: %+v", role)
	}
	decision := fixture.DecisionContract
	if decision.NormalQuorum != 3 || !decision.EarlyNormalDecisionForbidden ||
		decision.Accepted != "at_least_two_accept_after_three_ballots" ||
		decision.Rejected != "at_least_two_reject_after_three_ballots" ||
		decision.NoConsensus != "every_other_complete_three_ballot_result" ||
		decision.BlockedSecurity != "any_typed_security_veto_with_artifact_and_evidence_ref" ||
		decision.SecurityVetoDecisionTiming != "after_all_three_ballots_without_retiring_pending_roles" ||
		decision.Dissent != "accepted derives dissent for reject or abstain; rejected derives dissent for accept or abstain; no_consensus derives dissent for all three ballots; blocked_security derives dissent for every non_veto ballot" ||
		decision.Integration != "only explicit integration command may proceed and must revalidate V18 gate plus exact typed council resolution" ||
		decision.ReplanCausality != "every Director replan proposal and successor binds source CouncilDecisionRef CouncilDecisionDigest and CouncilSubjectDigest" {
		t.Fatalf("incomplete deterministic decision contract: %+v", decision)
	}
	assertV19Strings(t, "decision outcomes", decision.RequiredTestOutcomes,
		[]string{"accepted", "rejected", "no_consensus", "blocked_security"})
	resolution := fixture.ResolutionContract
	if resolution.CommonBinding != "CouncilSubjectDigest" ||
		!reflect.DeepEqual(resolution.AcceptedRoundBinding, []string{"CouncilDecisionRef", "CouncilDecisionDigest"}) ||
		!reflect.DeepEqual(resolution.SkipBinding, []string{"CouncilSkipRef", "CouncilSkipDigest"}) ||
		resolution.MutualExclusion != "exactly one accepted_round or skip binding is present on integration action intent admission and processing" ||
		resolution.TargetDigest != "bind resolution kind common subject digest and the selected ref plus digest" ||
		!resolution.SyntheticSkipForbidden {
		t.Fatalf("ambiguous Council integration resolution: %+v", resolution)
	}
}

func assertV19SafetyAndPersistence(t *testing.T, fixture v19Fixture) {
	t.Helper()
	authority := fixture.AuthorityContract
	if authority.RequiredOpenPermission != "goals.direct" || authority.SkipPermission != "council.skip" ||
		authority.SkipPrincipalKind != "human" || authority.SkipTemporalBoundary != "before_round_fact_or_council_launch" {
		t.Fatalf("invalid council authority: %+v", authority)
	}
	assertV19Strings(t, "skip roles", authority.SkipRoles, []string{"platform_admin", "project_owner", "operator"})
	if len(authority.SkipRequiredFields) != 6 || len(fixture.Invariants) != 12 || len(fixture.ForbiddenPrivateAuthorities) != 7 {
		t.Fatalf("incomplete authority/invariants/forbidden contract")
	}
	if !fixture.ExecutionContract.NewActionKindForbidden || fixture.ExecutionContract.ACKOrTextIsEvidence ||
		!reflect.DeepEqual(fixture.ExecutionContract.ActionKinds, []string{"launch_agent", "observe_agent"}) ||
		!reflect.DeepEqual(fixture.ExecutionContract.Purposes, []string{"council_proposer", "council_critic", "council_arbiter"}) {
		t.Fatalf("council introduced execution authority: %+v", fixture.ExecutionContract)
	}
	if fixture.PersistenceContract.Authority != "GoalRecord_through_StateRepository" ||
		fixture.PersistenceContract.Atomicity != "same_CAS_snapshot_event_outbox_transaction" ||
		!reflect.DeepEqual(fixture.PersistenceContract.Facts, []string{"round", "proposal", "critique", "ballot", "dissent", "security_veto", "decision", "skip"}) ||
		fixture.PersistenceContract.Replay == "" || fixture.PersistenceContract.Recovery == "" ||
		fixture.MigrationContract.Version != 14 || fixture.MigrationContract.SourceVersion != 13 ||
		fixture.MigrationContract.LiveV18CandidateWithoutDurablePolicy != "fail_closed" ||
		fixture.MigrationContract.CompletedV18Records != "preserve_without_retroactive_council" ||
		fixture.MigrationContract.Backfill != "forbidden" {
		t.Fatalf("invalid persistence/migration contract: %+v %+v", fixture.PersistenceContract, fixture.MigrationContract)
	}
	budget := fixture.Budgets
	if budget.Stage != "pre_P_operational_ceiling" || budget.ProductLOCMax != 3650 ||
		budget.DomainLOCMax != 350 || budget.ApplicationLOCMax != 1300 ||
		budget.SQLiteRecoveryLOCMax != 1600 || budget.BootstrapLOCMax != 400 || budget.FileLOCMax != 1450 ||
		budget.MigrationException != "forbidden" ||
		budget.QualityAccreditation != "P_cannot_declare_simplicity_green_from_this_ceiling" ||
		budget.SealRequirement != "S_records_LOC_by_layer_and_files_over_350" ||
		budget.PostV22Debt != "separate_SQLite_claim_read_validate_write_and_application_adapters_over_350_then_restore_compact_per_module_budget" ||
		budget.DomainLOCMax+budget.ApplicationLOCMax+budget.SQLiteRecoveryLOCMax+budget.BootstrapLOCMax != budget.ProductLOCMax {
		t.Fatalf("invalid V19 pre-P budget/debt contract: %+v", budget)
	}
}

func assertV19E2E(t *testing.T, fixture v19Fixture) {
	t.Helper()
	if len(fixture.E2ECases) != 3 || len(fixture.PSEContract) != 3 {
		t.Fatalf("V19 E2E/PSE count invalid")
	}
	var policies []string
	wantAssertions := map[string]int{"auto": 6, "required": 7, "skip_by_operator": 6}
	for _, scenario := range fixture.E2ECases {
		policies = append(policies, scenario.Policy)
		if !scenario.IsolatedRuntime || len(scenario.Assertions) != wantAssertions[scenario.Policy] {
			t.Errorf("V19 %s E2E assertions=%d want=%d", scenario.Policy, len(scenario.Assertions), wantAssertions[scenario.Policy])
		}
	}
	assertV19Strings(t, "E2E policies", sortedV19(policies), []string{"auto", "required", "skip_by_operator"})
	for _, phase := range []string{"P", "S", "E"} {
		if fixture.PSEContract[phase] == "" {
			t.Errorf("missing P/S/E phase %s", phase)
		}
	}
}

func loadV19Fixture(t *testing.T) v19Fixture {
	t.Helper()
	content, err := os.ReadFile(v19FixturePath)
	if err != nil {
		t.Fatal(err)
	}
	var fixture v19Fixture
	if err := json.Unmarshal(content, &fixture); err != nil {
		t.Fatal(err)
	}
	return fixture
}

func assertV19Strings(t *testing.T, name string, got, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("V19 %s=%v want=%v", name, got, want)
	}
}

func sortedV19(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	return result
}
