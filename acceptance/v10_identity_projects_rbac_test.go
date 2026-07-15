package acceptance_test

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

const v10FixturePath = "acceptance/fixtures/v10_identity_projects_rbac.json"
const v10TrustedBaseGitCommitOID = "f48066832b3813e996bf990083cf13c78e98c506"
const v10ProductDeltaBaseGitCommitOID = "f48066832b3813e996bf990083cf13c78e98c506"
const v10ProductDeltaSealedGitCommitOID = "0000000000000000000000000000000000000000"

type v10Fixture struct {
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
	DeferredCapabilities           []v10DeferredCapability `json:"deferred_capabilities"`
	DeferredSurfaces               []string                `json:"deferred_surfaces"`
	Scenario                       v10Scenario             `json:"scenario"`
	Assertions                     []string                `json:"assertions"`
}

type v10DeferredCapability struct {
	ID                 string `json:"id"`
	Owner              string `json:"owner"`
	AcceptanceContract string `json:"acceptance_contract"`
}

type v10Scenario struct {
	BaseTime                       string   `json:"base_time"`
	WorkspaceRefs                  []string `json:"workspace_refs"`
	GroupRefs                      []string `json:"group_refs"`
	ProjectRefs                    []string `json:"project_refs"`
	RepositoryRefs                 []string `json:"repository_refs"`
	HumanActorRefs                 []string `json:"human_actor_refs"`
	ServiceActorRef                string   `json:"service_actor_ref"`
	Roles                          []string `json:"roles"`
	ProtectedActions               []string `json:"protected_actions"`
	SealedPredecessorSchemaVersion int      `json:"sealed_predecessor_schema_version"`
	LocalOwnerRole                 string   `json:"local_owner_role"`
}

func TestAcceptanceV10IdentityProjectsRBAC(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v10Fixture](t, filepath.Join(repositoryRoot, filepath.FromSlash(v10FixturePath)))
	v10AssertFixtureHeader(t, repositoryRoot, fixture)

	t.Run("logical_hierarchy_and_roles_are_product_contracts", func(t *testing.T) {
		identitySource := v10ReadProductionGo(t, filepath.Join(repositoryRoot, "internal", "identity"))
		for _, required := range append(
			append([]string(nil), fixture.Scenario.Roles...),
			"WorkspaceRef", "GroupRef", "RepositoryRef",
		) {
			if !strings.Contains(identitySource, required) {
				t.Errorf("V10_RED identity contract lacks %q", required)
			}
		}
	})

	t.Run("application_authorizes_instead_of_filtering_by_creator", func(t *testing.T) {
		applicationSource := v10ReadProductionGo(t, filepath.Join(repositoryRoot, "internal", "application"))
		if !strings.Contains(strings.ToLower(applicationSource), "authoriz") {
			t.Error("V10_RED application has no authorization decision seam")
		}
		if strings.Contains(applicationSource, "record.Goal.Actor() != query.ActorRef") ||
			strings.Contains(applicationSource, "ListGoals(ctx, actorRef, projectRef") {
			t.Error("V10_RED creator equality still substitutes for project authorization")
		}
	})

	t.Run("mcp_requires_explicit_project_and_request_principal", func(t *testing.T) {
		tools := v10ReadFile(t, filepath.Join(repositoryRoot, "internal", "interfaces", "mcp", "tools.go"))
		middleware := v10ReadFile(t, filepath.Join(repositoryRoot, "internal", "adapters", "auth", "localtoken", "middleware.go"))
		if !strings.Contains(tools, "ProjectRef string `json:\"project_ref") {
			t.Error("V10_RED MCP inputs do not carry explicit project_ref")
		}
		if !strings.Contains(strings.ToLower(middleware), "principal") {
			t.Error("V10_RED local token authenticates a bearer but does not bind a request principal")
		}
	})

	t.Run("sqlite_migrates_sealed_v09_identity_state", func(t *testing.T) {
		migration := filepath.Join(repositoryRoot, "internal", "adapters", "state", "sqlite", "migrations", "006_identity_projects_rbac.sql")
		if _, err := os.Stat(migration); err != nil {
			t.Errorf("V10_RED identity migration 006 missing: %v", err)
		}
		recovery := v10ReadFile(t, filepath.Join(repositoryRoot, "internal", "adapters", "state", "sqlite", "recovery_validation.go"))
		if strings.Contains(recovery, "current != len(migrations)") {
			t.Error("V10_RED recovery rejects the sealed V09 predecessor solely because it is not the latest schema")
		}
	})
}

func TestAcceptanceV10IdentityProjectsRBACReceipt(t *testing.T) {
	evidenceAssertReceiptV3(t, evidenceRepositoryRoot(t), evidenceReceiptV3Expectation{
		Contract: "AC-V10-IDENTITY-PROJECTS-RBAC", FixturePath: v10FixturePath,
		ReceiptPath:       "product/evidence/v10_identity_projects_rbac.json",
		ExecutedNotBefore: "2026-07-15T00:00:00Z", TrustedBaseGitCommitOID: v10TrustedBaseGitCommitOID,
	})
}

func TestV10CandidateSubjectsCoverCommittedDelta(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v10Fixture](t, filepath.Join(repositoryRoot, filepath.FromSlash(v10FixturePath)))
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
		t.Fatalf("V10 candidate subjects differ from sealed product delta:\nchanged=%v\nfixture=%v", changed, fixture.CandidateSubjects)
	}
}

func TestV10FixtureIsStrictJSON(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v10Fixture](t, filepath.Join(repositoryRoot, filepath.FromSlash(v10FixturePath)))
	if fixture.ContractID != "AC-V10-IDENTITY-PROJECTS-RBAC" {
		t.Fatalf("unexpected V10 contract: %q", fixture.ContractID)
	}
}

func v10AssertFixtureHeader(t *testing.T, repositoryRoot string, fixture v10Fixture) {
	t.Helper()
	wantRoles := []string{
		"platform_admin", "project_owner", "project_admin", "contributor", "reviewer", "operator", "viewer",
	}
	wantDeferredCapabilities := []v10DeferredCapability{
		{ID: "ORC-11", Owner: "budgets_effects", AcceptanceContract: "AC-V15-BUDGETS-EFFECTS"},
		{ID: "STG-02", Owner: "workspace_git", AcceptanceContract: "AC-V16-WORKSPACE-GIT"},
		{ID: "EXT-10", Owner: "workspace_git", AcceptanceContract: "AC-V16-WORKSPACE-GIT"},
		{ID: "UI-04", Owner: "web_admin", AcceptanceContract: "AC-V24-WEB-ADMIN"},
		{ID: "OPS-11", Owner: "postgres_s3_multihost", AcceptanceContract: "AC-V31-POSTGRES-S3-MULTIHOST"},
	}
	wantDeferredSurfaces := []string{
		"oidc_ad_ldap", "fairness_budgets_effects", "physical_workspace_git_forge", "web_admin", "postgres_s3_multihost",
	}
	if fixture.SchemaVersion != 1 || fixture.ReceiptSchemaVersion != 3 ||
		fixture.ContractID != "AC-V10-IDENTITY-PROJECTS-RBAC" ||
		fixture.TrustedBaseGitCommitOID != v10TrustedBaseGitCommitOID ||
		fixture.ProductDeltaBaseGitCommitOID != v10ProductDeltaBaseGitCommitOID ||
		fixture.ProductDeltaSealedGitCommitOID != v10ProductDeltaSealedGitCommitOID ||
		fixture.OutputPath != "product/evidence/v10_identity_projects_rbac.output.txt" ||
		fixture.ReceiptPath != "product/evidence/v10_identity_projects_rbac.json" ||
		!reflect.DeepEqual(fixture.OwnedCapabilityIDs, []string{"GOV-19", "GOV-20", "GOV-22"}) ||
		!reflect.DeepEqual(fixture.DeferredCapabilities, wantDeferredCapabilities) ||
		!reflect.DeepEqual(fixture.DeferredSurfaces, wantDeferredSurfaces) ||
		!reflect.DeepEqual(fixture.Scenario.Roles, wantRoles) ||
		fixture.Scenario.SealedPredecessorSchemaVersion != 5 ||
		fixture.Scenario.LocalOwnerRole != "project_owner" || len(fixture.Assertions) != 12 {
		t.Fatalf("invalid V10 fixture header: %+v", fixture)
	}
	if len(fixture.Scenario.WorkspaceRefs) != 2 || len(fixture.Scenario.GroupRefs) != 2 ||
		len(fixture.Scenario.ProjectRefs) != 2 || len(fixture.Scenario.RepositoryRefs) != 2 ||
		len(fixture.Scenario.HumanActorRefs) != 4 || fixture.Scenario.ServiceActorRef == "" ||
		len(fixture.Scenario.ProtectedActions) != 8 {
		t.Fatalf("incomplete V10 scenario: %+v", fixture.Scenario)
	}
	if _, err := time.Parse(time.RFC3339Nano, fixture.Scenario.BaseTime); err != nil {
		t.Fatalf("invalid V10 base time: %v", err)
	}
	if fixture.Command != "sh -c '"+v10ValidationShellBody()+"'" ||
		!reflect.DeepEqual(fixture.ExecutionArgv, []string{"sh", "-c", v10ValidationShellBody()}) {
		t.Fatalf("invalid V10 command/argv: %q %#v", fixture.Command, fixture.ExecutionArgv)
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

func v10ValidationShellBody() string {
	return "go test -mod=vendor -count=1 . ./acceptance -run \"^(TestProductRoadmapIsExhaustiveAndCausal|TestProductRoadmapV10ScopeAndExecutableContract|TestV10EvidenceBelongsOnlyToIdentityCapabilities|TestV10AcceptanceCommandRunsIdentityConsumers|TestRebuildArchitecture|TestTraceabilityRebuildBugLessons|TestAcceptanceV10IdentityProjectsRBAC|TestV10CandidateSubjectsCoverCommittedDelta)$\"" +
		" && go test -mod=vendor -count=1 ./internal/goal ./internal/identity ./internal/application ./internal/config ./internal/i18n ./internal/adapters/auth/localtoken ./internal/adapters/state/sqlite ./internal/interfaces/mcp ./internal/bootstrap ./cmd/orquesta"
}

func v10ReadProductionGo(t *testing.T, directory string) string {
	t.Helper()
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("read production package %s: %v", directory, err)
	}
	var source strings.Builder
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		source.WriteString(v10ReadFile(t, filepath.Join(directory, entry.Name())))
		source.WriteByte('\n')
	}
	return source.String()
}

func v10ReadFile(t *testing.T, filename string) string {
	t.Helper()
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("read %s: %v", filename, err)
	}
	return string(content)
}
