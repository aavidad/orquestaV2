package acceptance_test

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
)

const v18FixturePath = "acceptance/fixtures/v18_independent_reviews.json"
const v18ContractBaseGitCommitOID = "eb272b6645928d800619709c9afd272440b0dabf"

type v18Fixture struct {
	SchemaVersion                  int                    `json:"schema_version"`
	ContractID                     string                 `json:"contract_id"`
	TrustedBaseGitCommitOID        string                 `json:"trusted_base_git_commit_oid"`
	ProductDeltaSealedGitCommitOID string                 `json:"product_delta_sealed_git_commit_oid"`
	SealStatus                     string                 `json:"seal_status"`
	SealNote                       string                 `json:"seal_note"`
	OwnedCapabilityIDs             []string               `json:"owned_capability_ids"`
	DependencyVerticals            []string               `json:"dependency_verticals"`
	DependencyReceipts             []v18DependencyReceipt `json:"dependency_receipts"`
	ReviewRoles                    []string               `json:"review_roles"`
	ReviewVerdicts                 []string               `json:"review_verdicts"`
	SubjectContract                v18TypeContract        `json:"subject_contract"`
	RequiredTypeContracts          []v18TypeContract      `json:"required_type_contracts"`
	RequiredReviewMarkers          []string               `json:"required_review_markers"`
	RequiredApplicationMarkers     []string               `json:"required_application_markers"`
	ForbiddenPrivateAuthorities    []string               `json:"forbidden_private_authorities"`
	AcceptedScenario               v18AcceptedScenario    `json:"accepted_scenario"`
	DriftMutations                 []v18DriftMutation     `json:"drift_mutations"`
	ReworkScenario                 v18ReworkScenario      `json:"rework_scenario"`
	RequiredBehaviorTests          []string               `json:"required_behavior_tests"`
	MutationGates                  []string               `json:"mutation_gates"`
	RecoveryReplayRequirements     []string               `json:"recovery_replay_requirements"`
	E2EContract                    v18E2EContract         `json:"e2e_contract"`
	SimplicityBudget               v18SimplicityBudget    `json:"simplicity_budget"`
	DeferredSurfaces               []string               `json:"deferred_surfaces"`
}

type v18DependencyReceipt struct {
	Vertical        string `json:"vertical"`
	Path            string `json:"path"`
	PreflightStatus string `json:"preflight_status"`
}

type v18TypeContract struct {
	Directory    string   `json:"directory"`
	Name         string   `json:"name"`
	Constructor  string   `json:"constructor"`
	DigestMethod string   `json:"digest_method"`
	Fields       []string `json:"fields"`
}

type v18SubjectFixture struct {
	GoalRef                string `json:"goal_ref"`
	WorkItemRef            string `json:"work_item_ref"`
	AuthorExecutionRef     string `json:"author_execution_ref"`
	AuthorExecutionAttempt uint64 `json:"author_execution_attempt"`
	PlanGeneration         uint64 `json:"plan_generation"`
	WorkItemGeneration     uint64 `json:"work_item_generation"`
	AppSpecGeneration      uint64 `json:"app_spec_generation"`
	SpecHash               string `json:"spec_hash"`
	AuthorLaunchReceiptRef string `json:"author_launch_receipt_ref"`
	WorkspaceBindingDigest string `json:"workspace_binding_digest"`
	ChangeSetRef           string `json:"change_set_ref"`
	ChangeSetDigest        string `json:"change_set_digest"`
	TreeOID                string `json:"tree_oid"`
	DiffDigest             string `json:"diff_digest"`
	WriteSetDigest         string `json:"write_set_digest"`
	RequiredTestsDigest    string `json:"required_tests_digest"`
	TestAttestationRef     string `json:"test_attestation_ref"`
	TestSubjectDigest      string `json:"test_subject_digest"`
	TestPolicyDigest       string `json:"test_policy_digest"`
	SubjectDigest          string `json:"subject_digest"`
}

type v18ParticipantFixture struct {
	Role             string `json:"role"`
	ExecutionRef     string `json:"execution_ref"`
	LaunchReceiptRef string `json:"launch_receipt_ref"`
	SubjectDigest    string `json:"subject_digest"`
	Decision         string `json:"decision"`
}

type v18AcceptedScenario struct {
	Subject                v18SubjectFixture       `json:"subject"`
	Participants           []v18ParticipantFixture `json:"participants"`
	IntegrationIsExplicit  bool                    `json:"integration_is_explicit"`
	IntegrationIsAutomatic bool                    `json:"integration_is_automatic"`
}

type v18DriftMutation struct {
	Field        string `json:"field"`
	Replacement  string `json:"replacement"`
	ExpectedCode string `json:"expected_code"`
}

type v18ReworkScenario struct {
	SourceSubjectDigest      string `json:"source_subject_digest"`
	ChangesRequestedBy       string `json:"changes_requested_by"`
	ReplanCause              string `json:"replan_cause"`
	SourceWorkItemRef        string `json:"source_work_item_ref"`
	SourceChangeSetRef       string `json:"source_change_set_ref"`
	SuccessorWorkItemRef     string `json:"successor_work_item_ref"`
	SuccessorReworkOf        string `json:"successor_rework_of"`
	SuccessorParentChangeRef string `json:"successor_parent_change_ref"`
	SuccessorSubjectDigest   string `json:"successor_subject_digest"`
	PriorReviewsReusable     bool   `json:"prior_reviews_reusable"`
	IntegrationIsExplicit    bool   `json:"integration_is_explicit"`
}

type v18E2EContract struct {
	Composition                     string   `json:"composition"`
	RealProviderRequired            bool     `json:"real_provider_required"`
	RequiredLaunchCountPerCandidate int      `json:"required_launch_count_per_candidate"`
	RestartFrontiers                []string `json:"restart_frontiers"`
	MaxWallTimeSeconds              int      `json:"max_wall_time_seconds"`
	ZeroOwnedProcessesAfter         bool     `json:"zero_owned_processes_after"`
}

type v18SimplicityBudget struct {
	ProductTotal        int `json:"product_total"`
	ReviewDomain        int `json:"review_domain"`
	Application         int `json:"application"`
	SQLiteRecovery      int `json:"sqlite_recovery"`
	AdaptersBootstrap   int `json:"adapters_bootstrap"`
	MaxFileLines        int `json:"max_file_lines"`
	NewLifecycleWriters int `json:"new_lifecycle_writers"`
	NewSchedulers       int `json:"new_schedulers"`
	NewStores           int `json:"new_stores"`
	NewDaemons          int `json:"new_daemons"`
	NewOutboundPorts    int `json:"new_outbound_ports"`
}

func TestAcceptanceV18IndependentReviews(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v18Fixture](t, filepath.Join(repositoryRoot, v18FixturePath))
	v18AssertFixture(t, repositoryRoot, fixture)

	t.Run("sealed_dependencies_visible_without_assuming_v17_api", func(t *testing.T) {
		v18AssertKnownDependencySeams(t, repositoryRoot)
	})
	t.Run("exact_subject_and_structured_review_facts", func(t *testing.T) {
		v18AssertProductModel(t, repositoryRoot, fixture)
	})
	t.Run("three_launches_one_subject_one_writer_and_explicit_integration", func(t *testing.T) {
		v18AssertFlowAuthority(t, repositoryRoot, fixture)
	})
	t.Run("drift_rework_replay_recovery_and_e2e_gates_exist", func(t *testing.T) {
		v18AssertBehaviorTests(t, repositoryRoot, fixture)
	})
}

func v18AssertFixture(t *testing.T, repositoryRoot string, fixture v18Fixture) {
	t.Helper()
	if fixture.SchemaVersion != 1 || fixture.ContractID != "AC-V18-INDEPENDENT-REVIEWS" ||
		fixture.TrustedBaseGitCommitOID != v18ContractBaseGitCommitOID ||
		fixture.ProductDeltaSealedGitCommitOID != "" ||
		fixture.SealStatus != "preflight_awaiting_v17_no_product_evidence" ||
		fixture.SealNote != "Preflight contract only. V17 is not sealed; empty product OID and this fixture never accredit V18." {
		t.Fatalf("invalid V18 preflight identity: %+v", fixture)
	}
	if !reflect.DeepEqual(fixture.OwnedCapabilityIDs, []string{"GOV-12", "STG-13", "STG-14", "STG-16", "EVD-06"}) ||
		!reflect.DeepEqual(fixture.DependencyVerticals, []string{"controls", "workspace_git", "test_attestor"}) ||
		!reflect.DeepEqual(fixture.ReviewRoles, []string{"author", "primary", "adversarial"}) ||
		!reflect.DeepEqual(fixture.ReviewVerdicts, []string{"approve", "changes_requested"}) {
		t.Fatalf("invalid V18 ownership, dependencies, roles or verdicts: %+v", fixture)
	}
	wantReceipts := []v18DependencyReceipt{
		{Vertical: "controls", Path: "product/evidence/v14_controls.json", PreflightStatus: "verified_present"},
		{Vertical: "workspace_git", Path: "product/evidence/v16_workspace_git.json", PreflightStatus: "verified_present"},
		{Vertical: "test_attestor", Path: "product/evidence/v17_test_attestor.json", PreflightStatus: "awaiting_dependency"},
	}
	if !reflect.DeepEqual(fixture.DependencyReceipts, wantReceipts) {
		t.Fatalf("invalid V18 dependency receipt state: %+v", fixture.DependencyReceipts)
	}
	for _, dependency := range fixture.DependencyReceipts[:2] {
		info, err := os.Lstat(filepath.Join(repositoryRoot, filepath.FromSlash(dependency.Path)))
		if err != nil || !info.Mode().IsRegular() {
			t.Fatalf("sealed V18 dependency receipt %s unavailable: %v", dependency.Path, err)
		}
	}
	if fixture.SubjectContract.Directory != "internal/review" || fixture.SubjectContract.Name != "Subject" ||
		fixture.SubjectContract.Constructor != "NewSubject" || fixture.SubjectContract.DigestMethod != "Digest" ||
		len(fixture.SubjectContract.Fields) != 19 || len(fixture.RequiredTypeContracts) != 4 ||
		len(fixture.RequiredReviewMarkers) != 8 || len(fixture.RequiredApplicationMarkers) != 7 ||
		len(fixture.ForbiddenPrivateAuthorities) != 11 || len(fixture.RequiredBehaviorTests) != 18 ||
		len(fixture.MutationGates) != 15 || len(fixture.RecoveryReplayRequirements) != 6 ||
		len(fixture.DeferredSurfaces) != 8 {
		t.Fatalf("invalid V18 contract coverage counts: %+v", fixture)
	}
	v18AssertAcceptedScenario(t, fixture.AcceptedScenario)
	v18AssertDriftMutations(t, fixture.AcceptedScenario.Subject, fixture.DriftMutations)
	v18AssertReworkScenario(t, fixture.AcceptedScenario, fixture.ReworkScenario)
	wantE2E := v18E2EContract{
		Composition: "fake_agents_git_sqlite_cas_test_attestor", RequiredLaunchCountPerCandidate: 3,
		RestartFrontiers:   []string{"before_reviews", "after_primary", "after_both", "after_integration_admission"},
		MaxWallTimeSeconds: 180, ZeroOwnedProcessesAfter: true,
	}
	if !reflect.DeepEqual(fixture.E2EContract, wantE2E) {
		t.Fatalf("invalid V18 E2E contract: %+v", fixture.E2EContract)
	}
	wantBudget := v18SimplicityBudget{
		ProductTotal: 4200, ReviewDomain: 700, Application: 1800, SQLiteRecovery: 1000,
		AdaptersBootstrap: 700, MaxFileLines: 350,
	}
	if fixture.SimplicityBudget != wantBudget {
		t.Fatalf("invalid V18 simplicity budget: %+v", fixture.SimplicityBudget)
	}
	if _, err := evidenceGitCanonicalCommit(repositoryRoot, fixture.TrustedBaseGitCommitOID); err != nil {
		t.Fatalf("invalid V18 preflight base: %v", err)
	}
}

func v18AssertAcceptedScenario(t *testing.T, scenario v18AcceptedScenario) {
	t.Helper()
	if scenario.Subject.SubjectDigest != v18FixtureSubjectDigest(scenario.Subject) ||
		!scenario.IntegrationIsExplicit || scenario.IntegrationIsAutomatic || len(scenario.Participants) != 3 {
		t.Fatalf("invalid V18 accepted scenario subject/integration: %+v", scenario)
	}
	wantRoles, wantDecisions := []string{"author", "primary", "adversarial"}, []string{"produced", "approve", "approve"}
	executions, launches := map[string]bool{}, map[string]bool{}
	for index, participant := range scenario.Participants {
		if participant.Role != wantRoles[index] || participant.Decision != wantDecisions[index] ||
			participant.SubjectDigest != scenario.Subject.SubjectDigest || participant.ExecutionRef == "" ||
			participant.LaunchReceiptRef == "" || executions[participant.ExecutionRef] || launches[participant.LaunchReceiptRef] {
			t.Fatalf("V18 fixture participants are not three distinct launches on one subject: %+v", scenario.Participants)
		}
		executions[participant.ExecutionRef], launches[participant.LaunchReceiptRef] = true, true
	}
	if scenario.Participants[0].ExecutionRef != scenario.Subject.AuthorExecutionRef ||
		scenario.Participants[0].LaunchReceiptRef != scenario.Subject.AuthorLaunchReceiptRef {
		t.Fatalf("V18 fixture author does not match subject: %+v", scenario.Participants[0])
	}
}

func v18AssertDriftMutations(t *testing.T, baseline v18SubjectFixture, mutations []v18DriftMutation) {
	t.Helper()
	wantFields := []string{
		"plan_generation", "work_item_generation", "author_launch_receipt_ref", "tree_oid",
		"diff_digest", "required_tests_digest", "test_subject_digest", "test_policy_digest",
	}
	if len(mutations) != len(wantFields) {
		t.Fatalf("V18 drift mutation count=%d", len(mutations))
	}
	for index, mutation := range mutations {
		if mutation.Field != wantFields[index] || mutation.Replacement == "" || mutation.ExpectedCode != "review.subject_mismatch" {
			t.Fatalf("invalid V18 drift mutation: %+v", mutation)
		}
		changed := baseline
		switch mutation.Field {
		case "plan_generation":
			changed.PlanGeneration = mustV18Uint(t, mutation.Replacement)
		case "work_item_generation":
			changed.WorkItemGeneration = mustV18Uint(t, mutation.Replacement)
		case "author_launch_receipt_ref":
			changed.AuthorLaunchReceiptRef = mutation.Replacement
		case "tree_oid":
			changed.TreeOID = mutation.Replacement
		case "diff_digest":
			changed.DiffDigest = mutation.Replacement
		case "required_tests_digest":
			changed.RequiredTestsDigest = mutation.Replacement
		case "test_subject_digest":
			changed.TestSubjectDigest = mutation.Replacement
		case "test_policy_digest":
			changed.TestPolicyDigest = mutation.Replacement
		}
		if v18FixtureSubjectDigest(changed) == baseline.SubjectDigest {
			t.Fatalf("V18 drift mutation %s did not change fixture subject", mutation.Field)
		}
	}
}

func v18AssertReworkScenario(t *testing.T, accepted v18AcceptedScenario, rework v18ReworkScenario) {
	t.Helper()
	if rework.SourceSubjectDigest != accepted.Subject.SubjectDigest || rework.ChangesRequestedBy != "primary" ||
		rework.ReplanCause != "review_changes_requested" || rework.SourceWorkItemRef != accepted.Subject.WorkItemRef ||
		rework.SourceChangeSetRef != accepted.Subject.ChangeSetRef || rework.SuccessorWorkItemRef == "" ||
		rework.SuccessorReworkOf != rework.SourceWorkItemRef || rework.SuccessorParentChangeRef != rework.SourceChangeSetRef ||
		rework.SuccessorSubjectDigest == rework.SourceSubjectDigest || rework.PriorReviewsReusable || !rework.IntegrationIsExplicit {
		t.Fatalf("invalid V18 causal rework fixture: %+v", rework)
	}
}

func v18FixtureSubjectDigest(subject v18SubjectFixture) string {
	fields := []string{
		"orquesta.review-subject.fixture.v1", subject.GoalRef, subject.WorkItemRef, subject.AuthorExecutionRef,
		strconv.FormatUint(subject.AuthorExecutionAttempt, 10), strconv.FormatUint(subject.PlanGeneration, 10),
		strconv.FormatUint(subject.WorkItemGeneration, 10), strconv.FormatUint(subject.AppSpecGeneration, 10),
		subject.SpecHash, subject.AuthorLaunchReceiptRef, subject.WorkspaceBindingDigest, subject.ChangeSetRef,
		subject.ChangeSetDigest, subject.TreeOID, subject.DiffDigest, subject.WriteSetDigest,
		subject.RequiredTestsDigest, subject.TestAttestationRef, subject.TestSubjectDigest, subject.TestPolicyDigest,
	}
	digest := sha256.New()
	for _, field := range fields {
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(len(field)))
		_, _ = digest.Write(size[:])
		_, _ = digest.Write([]byte(field))
	}
	return "sha256:" + hex.EncodeToString(digest.Sum(nil))
}

func mustV18Uint(t *testing.T, value string) uint64 {
	t.Helper()
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		t.Fatalf("invalid V18 fixture integer %q: %v", value, err)
	}
	return parsed
}
