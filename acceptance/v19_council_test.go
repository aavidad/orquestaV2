package acceptance_test

import (
	"encoding/json"
	"os"
	"reflect"
	"sort"
	"testing"
)

const v19FixturePath = "fixtures/v19_council.json"

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
	Policies             []string            `json:"policies"`
	PolicyContracts      []v19PolicyContract `json:"policy_contracts"`
	SubjectContract      struct {
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
	} `json:"role_contract"`
	Ballots          []string `json:"ballots"`
	DecisionContract struct {
		NormalQuorum                 int    `json:"normal_quorum"`
		Accepted                     string `json:"accepted"`
		Rejected                     string `json:"rejected"`
		NoConsensus                  string `json:"no_consensus"`
		BlockedSecurity              string `json:"blocked_security"`
		EarlyNormalDecisionForbidden bool   `json:"early_normal_decision_forbidden"`
		Dissent                      string `json:"dissent"`
		Integration                  string `json:"integration"`
	} `json:"decision_contract"`
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
		ProductLOCMax        int `json:"product_loc_max"`
		DomainLOCMax         int `json:"domain_loc_max"`
		ApplicationLOCMax    int `json:"application_loc_max"`
		SQLiteRecoveryLOCMax int `json:"sqlite_recovery_loc_max"`
		BootstrapLOCMax      int `json:"bootstrap_loc_max"`
		FileLOCMax           int `json:"file_loc_max"`
	} `json:"budgets"`
	Invariants []string `json:"invariants"`
	E2ECases   []struct {
		Policy          string   `json:"policy"`
		IsolatedRuntime bool     `json:"isolated_runtime"`
		Assertions      []string `json:"assertions"`
	} `json:"e2e_cases"`
	PSEContract map[string]string `json:"pse_contract"`
	RedGate     string            `json:"red_gate"`
}

func TestAcceptanceV19Council(t *testing.T) {
	fixture := loadV19Fixture(t)
	if fixture.SchemaVersion != 2 || fixture.ContractID != "AC-V19-COUNCIL" || fixture.Vertical != "council" ||
		fixture.ImplementationStatus != "awaiting_product" || fixture.RedGate != "V19_PRODUCT_PENDING" {
		t.Fatalf("invalid V19 controlled-red identity: %+v", fixture)
	}
	wantCapabilities := []string{"EVD-07", "GOV-11", "GOV-13", "GOV-14", "STG-06", "STG-08"}
	assertV19Strings(t, "capabilities", sortedV19(fixture.OwnedCapabilityIDs), wantCapabilities)
	assertV19Dependency(t, fixture.Dependency)
	assertV19Policies(t, fixture)
	assertV19CouncilShape(t, fixture)
	assertV19SafetyAndPersistence(t, fixture)
	assertV19E2E(t, fixture)
	t.Fatalf("%s: V18 is accredited; only product V19 is absent", fixture.RedGate)
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
	openers := map[string]string{}
	for _, policy := range fixture.PolicyContracts {
		if policy.Policy == "" || policy.Opener == "" || policy.Behavior == "" || policy.PromotionGate == "" {
			t.Fatalf("incomplete policy: %+v", policy)
		}
		openers[policy.Policy] = policy.Opener
	}
	if openers["auto"] != "application_after_exact_v18_gate" ||
		openers["required"] != "director_with_live_goal_lease_and_fence" ||
		openers["skip_by_operator"] != "authorized_human_with_council_skip" {
		t.Fatalf("policy openers overlap or drift: %v", openers)
	}
}

func assertV19CouncilShape(t *testing.T, fixture v19Fixture) {
	t.Helper()
	assertV19Strings(t, "subject digest", fixture.SubjectContract.DigestFields,
		[]string{"ProjectRef", "ReviewSubjectDigest", "ReviewGateDigest", "CouncilPolicy"})
	assertV19Strings(t, "roles", fixture.Roles, []string{"proposer", "critic", "arbiter"})
	assertV19Strings(t, "ballots", sortedV19(fixture.Ballots), []string{"abstain", "accept", "reject", "security_veto"})
	role := fixture.RoleContract
	if role.RequiredLaunches != 3 || role.RequiredBallots != 3 || !role.DistinctFromEachOther ||
		!role.DistinctFromV18AuthorPrimaryAdversarial || !role.FactsRequireLaunchReceipt || len(role.ArtifactSchemas) != 3 {
		t.Fatalf("invalid council cardinality/independence: %+v", role)
	}
	decision := fixture.DecisionContract
	if decision.NormalQuorum != 3 || !decision.EarlyNormalDecisionForbidden || decision.Accepted == "" ||
		decision.Rejected == "" || decision.NoConsensus == "" || decision.BlockedSecurity == "" ||
		decision.Dissent == "" || decision.Integration == "" {
		t.Fatalf("incomplete deterministic decision contract: %+v", decision)
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
		len(fixture.ExecutionContract.Purposes) != 3 {
		t.Fatalf("council introduced execution authority: %+v", fixture.ExecutionContract)
	}
	if fixture.PersistenceContract.Authority != "GoalRecord_through_StateRepository" || len(fixture.PersistenceContract.Facts) != 8 ||
		fixture.MigrationContract.Version != 14 || fixture.MigrationContract.SourceVersion != 13 ||
		fixture.MigrationContract.LiveV18CandidateWithoutDurablePolicy != "fail_closed" || fixture.MigrationContract.Backfill != "forbidden" {
		t.Fatalf("invalid persistence/migration contract: %+v %+v", fixture.PersistenceContract, fixture.MigrationContract)
	}
	if fixture.Budgets.ProductLOCMax != 3400 || fixture.Budgets.FileLOCMax != 350 ||
		fixture.Budgets.DomainLOCMax+fixture.Budgets.ApplicationLOCMax+fixture.Budgets.SQLiteRecoveryLOCMax+fixture.Budgets.BootstrapLOCMax > fixture.Budgets.ProductLOCMax {
		t.Fatalf("invalid simplicity budget: %+v", fixture.Budgets)
	}
}

func assertV19E2E(t *testing.T, fixture v19Fixture) {
	t.Helper()
	if len(fixture.E2ECases) != 3 || len(fixture.PSEContract) != 3 {
		t.Fatalf("V19 E2E/PSE count invalid")
	}
	var policies []string
	for _, scenario := range fixture.E2ECases {
		policies = append(policies, scenario.Policy)
		if !scenario.IsolatedRuntime || len(scenario.Assertions) != 6 {
			t.Errorf("V19 %s lacks isolated six-assertion E2E", scenario.Policy)
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
