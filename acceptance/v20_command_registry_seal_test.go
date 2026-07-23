package acceptance_test

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

const v20CandidateFreezeRecipe = "git add -f cmd/orquesta/command.go; { git diff --name-only " +
	v20ContractBaseGitCommitOID + " --; git ls-files --others --exclude-standard; } | LC_ALL=C sort -u"

func TestV20PSELifecycleIsExact(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v20Fixture](t, filepath.Join(repositoryRoot, v20FixturePath))
	if fixture.ProductDeltaBaseGitCommitOID != v20ContractBaseGitCommitOID ||
		fixture.SealNote == "" || fixture.Command == "" || len(fixture.ExecutionArgv) != 3 {
		t.Fatalf("invalid V20 P/S/E envelope: %+v", fixture)
	}

	switch fixture.ImplementationStatus {
	case "development_unsealed":
		if fixture.LifecycleGate != "V20_DEVELOPMENT_UNSEALED" ||
			fixture.SealStatus != "development_unsealed_candidate_pending" ||
			fixture.ProductDeltaSealedGitCommitOID != "" || len(fixture.CandidateSubjects) != 0 {
			t.Fatalf("V20 development state is not exact: %+v", fixture)
		}
		if v20ExecutionEvidencePresent(t, repositoryRoot, fixture) {
			t.Fatal("V20 development state cannot contain execution evidence")
		}
		t.Logf("V20 candidate is intentionally pending; freeze exact sorted subjects with: %s", v20CandidateFreezeRecipe)
	case "implemented_unsealed":
		if fixture.LifecycleGate != "V20_IMPLEMENTED_UNSEALED" ||
			fixture.SealStatus != "p_implemented_unsealed_pending_seal" ||
			fixture.ProductDeltaSealedGitCommitOID != "" {
			t.Fatalf("V20 P state is not exact: %+v", fixture)
		}
		v20AssertCandidateSubjects(t, repositoryRoot, fixture, false)
		if v20ExecutionEvidencePresent(t, repositoryRoot, fixture) {
			t.Fatal("V20 P state cannot contain execution evidence")
		}
	case "sealed_unexecuted":
		if fixture.LifecycleGate != "V20_SEALED_UNEXECUTED" ||
			fixture.SealStatus != "s_product_delta_sealed_pending_execution" ||
			fixture.ProductDeltaSealedGitCommitOID == "" {
			t.Fatalf("V20 S state is not exact: %+v", fixture)
		}
		v20AssertCandidateSubjects(t, repositoryRoot, fixture, true)
	default:
		t.Fatalf("unsupported V20 implementation_status %q", fixture.ImplementationStatus)
	}
}

func TestV20ReceiptV3Lifecycle(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v20Fixture](t, filepath.Join(repositoryRoot, v20FixturePath))
	if !v20ExecutionEvidencePresent(t, repositoryRoot, fixture) {
		return
	}
	if fixture.ImplementationStatus != "sealed_unexecuted" {
		t.Fatalf("V20 execution evidence exists outside S state: %q", fixture.ImplementationStatus)
	}
	evidenceAssertReceiptV3(t, repositoryRoot, evidenceReceiptV3Expectation{
		Contract: "AC-V20-COMMAND-REGISTRY", FixturePath: v20FixturePath,
		ReceiptPath: fixture.ReceiptPath, ExecutedNotBefore: "2026-07-23T00:00:00Z",
		TrustedBaseGitCommitOID: v20ContractBaseGitCommitOID,
	})
}

func TestV20CandidateSubjectsRetainGenericReceiptAndExcludeOwnEvidence(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v20Fixture](t, filepath.Join(repositoryRoot, v20FixturePath))
	if fixture.ImplementationStatus == "development_unsealed" {
		return
	}
	if err := evidenceValidateCandidateSubjects(fixture.CandidateSubjects, fixture.ReceiptPath, fixture.OutputPath); err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		v20FixturePath,
		"acceptance/v20_command_registry_seal_test.go",
		"product/evidence/real_codex_mcp_e2e.json",
		"product_roadmap_v20_test.go",
	} {
		if !v20ContainsSubject(fixture.CandidateSubjects, required) {
			t.Fatalf("V20 candidate lacks required subject %q", required)
		}
	}
}

func TestV20DevelopmentCandidateRecipeIsDeterministic(t *testing.T) {
	if !strings.Contains(v20CandidateFreezeRecipe, "git add -f cmd/orquesta/command.go") ||
		!strings.Contains(v20CandidateFreezeRecipe, "git diff --name-only "+v20ContractBaseGitCommitOID) ||
		!strings.HasSuffix(v20CandidateFreezeRecipe, "LC_ALL=C sort -u") {
		t.Fatalf("unsafe V20 candidate freeze recipe: %q", v20CandidateFreezeRecipe)
	}
	repositoryRoot := evidenceRepositoryRoot(t)
	first := v20CurrentCandidateSubjects(t, repositoryRoot)
	second := v20CurrentCandidateSubjects(t, repositoryRoot)
	if !reflect.DeepEqual(first, second) || !sort.StringsAreSorted(first) {
		t.Fatalf("V20 candidate recipe is nondeterministic: first=%v second=%v", first, second)
	}
}

func v20AssertCandidateSubjects(t *testing.T, repositoryRoot string, fixture v20Fixture, sealed bool) {
	t.Helper()
	if len(fixture.CandidateSubjects) == 0 || !sort.StringsAreSorted(fixture.CandidateSubjects) {
		t.Fatalf("V20 candidate subjects are empty or unsorted; run: %s", v20CandidateFreezeRecipe)
	}
	if sealed {
		evidenceAssertCandidateDelta(t, "V20", repositoryRoot, fixture.ProductDeltaBaseGitCommitOID,
			fixture.ProductDeltaSealedGitCommitOID, fixture.CandidateSubjects)
		return
	}
	got := v20CurrentCandidateSubjects(t, repositoryRoot)
	if !reflect.DeepEqual(got, fixture.CandidateSubjects) {
		t.Fatalf("V20_GATE_PRE_P_SUBJECTS: current=%v fixture=%v; regenerate with: %s",
			got, fixture.CandidateSubjects, v20CandidateFreezeRecipe)
	}
}

func v20CurrentCandidateSubjects(t *testing.T, repositoryRoot string) []string {
	t.Helper()
	unique := map[string]struct{}{}
	for _, args := range [][]string{
		{"diff", "--name-only", v20ContractBaseGitCommitOID, "--"},
		{"ls-files", "--others", "--exclude-standard"},
	} {
		output, err := evidenceGit(repositoryRoot, args...)
		if err != nil {
			t.Fatal(err)
		}
		for _, relative := range strings.Split(strings.TrimSpace(string(output)), "\n") {
			if relative != "" {
				unique[filepath.ToSlash(relative)] = struct{}{}
			}
		}
	}
	result := make([]string, 0, len(unique))
	for relative := range unique {
		result = append(result, relative)
	}
	sort.Strings(result)
	return result
}

func v20ExecutionEvidencePresent(t *testing.T, repositoryRoot string, fixture v20Fixture) bool {
	t.Helper()
	present := make([]bool, 0, 2)
	for _, relative := range []string{fixture.ReceiptPath, fixture.OutputPath} {
		if err := evidenceValidateRepositoryPath(relative); err != nil {
			t.Fatal(err)
		}
		_, err := os.Lstat(filepath.Join(repositoryRoot, filepath.FromSlash(relative)))
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}
		present = append(present, err == nil)
	}
	if present[0] != present[1] {
		t.Fatal("V20 execution evidence must contain both receipt and output or neither")
	}
	return present[0]
}

func v20ContainsSubject(subjects []string, wanted string) bool {
	index := sort.SearchStrings(subjects, wanted)
	return index < len(subjects) && subjects[index] == wanted
}
