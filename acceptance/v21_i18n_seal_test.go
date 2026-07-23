package acceptance_test

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

const v21CandidateFreezeRecipe = "{ git diff --name-only " +
	v21ContractBaseGitCommitOID + " --; git ls-files --others --exclude-standard; } | LC_ALL=C sort -u"

func TestV21PSELifecycleIsExact(t *testing.T) {
	root := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v21Fixture](t, filepath.Join(root, v21FixturePath))
	if fixture.ProductDeltaBaseGitCommitOID != v21ContractBaseGitCommitOID ||
		fixture.SealNote == "" || fixture.Command == "" || len(fixture.ExecutionArgv) != 3 {
		t.Fatalf("invalid V21 P/S/E envelope: %+v", fixture)
	}
	switch fixture.ImplementationStatus {
	case "development_unsealed":
		if fixture.LifecycleGate != "V21_DEVELOPMENT_UNSEALED" ||
			fixture.SealStatus != "development_unsealed_candidate_pending" ||
			fixture.ProductDeltaSealedGitCommitOID != "" || len(fixture.CandidateSubjects) != 0 ||
			fixture.PreflightStatus != "contract_ready_product_pending" {
			t.Fatalf("V21 development state is not exact: %+v", fixture)
		}
		if v21ExecutionEvidencePresent(t, root, fixture) {
			t.Fatal("V21 development state cannot contain execution evidence")
		}
	case "implemented_unsealed":
		if fixture.LifecycleGate != "V21_IMPLEMENTED_UNSEALED" ||
			fixture.SealStatus != "p_implemented_unsealed_pending_seal" ||
			fixture.ProductDeltaSealedGitCommitOID != "" ||
			fixture.PreflightStatus != "integrated_implementation_complete" {
			t.Fatalf("V21 P state is not exact: %+v", fixture)
		}
		v21AssertCandidateSubjects(t, root, fixture, false)
		if v21ExecutionEvidencePresent(t, root, fixture) {
			t.Fatal("V21 P state cannot contain execution evidence")
		}
	case "sealed_unexecuted":
		if fixture.LifecycleGate != "V21_SEALED_UNEXECUTED" ||
			fixture.SealStatus != "s_product_delta_sealed_pending_execution" ||
			fixture.ProductDeltaSealedGitCommitOID == "" ||
			fixture.PreflightStatus != "integrated_implementation_complete" {
			t.Fatalf("V21 S state is not exact: %+v", fixture)
		}
		v21AssertCandidateSubjects(t, root, fixture, true)
	default:
		t.Fatalf("unsupported V21 implementation_status %q", fixture.ImplementationStatus)
	}
}

func TestV21ReceiptV3Lifecycle(t *testing.T) {
	root := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v21Fixture](t, filepath.Join(root, v21FixturePath))
	if !v21ExecutionEvidencePresent(t, root, fixture) {
		return
	}
	if fixture.ImplementationStatus != "sealed_unexecuted" {
		t.Fatalf("V21 execution evidence exists outside S state: %q", fixture.ImplementationStatus)
	}
	evidenceAssertReceiptV3(t, root, evidenceReceiptV3Expectation{
		Contract: "AC-V21-I18N", FixturePath: v21FixturePath,
		ReceiptPath: fixture.ReceiptPath, ExecutedNotBefore: "2026-07-23T00:00:00Z",
		TrustedBaseGitCommitOID: v21ContractBaseGitCommitOID,
	})
	receipt := evidenceDecodeStrictJSON[evidenceReceiptV3](t,
		filepath.Join(root, filepath.FromSlash(fixture.ReceiptPath)))
	v21AssertReceiptSealedExactS(t, root, fixture, receipt.SealedSource.GitCommitOID)
	v21AssertEvidenceOutsideSealedTree(t, root, fixture, receipt.SealedSource.GitCommitOID)
}

func TestV21CandidateSubjectsExcludeOwnEvidenceAndRetainPublicContract(t *testing.T) {
	root := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v21Fixture](t, filepath.Join(root, v21FixturePath))
	if fixture.ImplementationStatus == "development_unsealed" {
		return
	}
	if err := evidenceValidateCandidateSubjects(
		fixture.CandidateSubjects, fixture.ReceiptPath, fixture.OutputPath,
	); err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		v21FixturePath,
		"acceptance/v21_i18n_seal_test.go",
		"docs/public/en/README.md",
		"docs/public/es/README.md",
		"internal/i18n/manifest.json",
		"product_roadmap_v21_test.go",
	} {
		if !v21ContainsSubject(fixture.CandidateSubjects, required) {
			t.Fatalf("V21 candidate lacks required subject %q", required)
		}
	}
}

func TestV21DevelopmentCandidateRecipeIsDeterministic(t *testing.T) {
	if !strings.Contains(v21CandidateFreezeRecipe, "git diff --name-only "+v21ContractBaseGitCommitOID) ||
		!strings.Contains(v21CandidateFreezeRecipe, "git ls-files --others --exclude-standard") ||
		!strings.HasSuffix(v21CandidateFreezeRecipe, "LC_ALL=C sort -u") {
		t.Fatalf("unsafe V21 candidate freeze recipe: %q", v21CandidateFreezeRecipe)
	}
	root := evidenceRepositoryRoot(t)
	first := v21CurrentCandidateSubjects(t, root)
	second := v21CurrentCandidateSubjects(t, root)
	if !reflect.DeepEqual(first, second) || !sort.StringsAreSorted(first) {
		t.Fatalf("V21 candidate recipe is nondeterministic: first=%v second=%v", first, second)
	}
}

func v21AssertCandidateSubjects(
	t *testing.T, root string, fixture v21Fixture, sealed bool,
) {
	t.Helper()
	if len(fixture.CandidateSubjects) == 0 || !sort.StringsAreSorted(fixture.CandidateSubjects) {
		t.Fatalf("V21 candidate subjects are empty or unsorted; run: %s", v21CandidateFreezeRecipe)
	}
	if sealed {
		evidenceAssertCandidateDelta(t, "V21", root, fixture.ProductDeltaBaseGitCommitOID,
			fixture.ProductDeltaSealedGitCommitOID, fixture.CandidateSubjects)
		return
	}
	got := v21CurrentCandidateSubjects(t, root)
	if !reflect.DeepEqual(got, fixture.CandidateSubjects) {
		t.Fatalf("V21_GATE_PRE_P_SUBJECTS: current=%v fixture=%v; regenerate with: %s",
			got, fixture.CandidateSubjects, v21CandidateFreezeRecipe)
	}
}

func v21CurrentCandidateSubjects(t *testing.T, root string) []string {
	t.Helper()
	unique := map[string]struct{}{}
	for _, arguments := range [][]string{
		{"diff", "--name-only", v21ContractBaseGitCommitOID, "--"},
		{"ls-files", "--others", "--exclude-standard"},
	} {
		output, err := evidenceGit(root, arguments...)
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

func v21ContainsSubject(subjects []string, wanted string) bool {
	index := sort.SearchStrings(subjects, wanted)
	return index < len(subjects) && subjects[index] == wanted
}

func v21AssertReceiptSealedExactS(
	t *testing.T, root string, fixture v21Fixture, sealedSourceOID string,
) {
	t.Helper()
	line, err := evidenceGit(root, "rev-list", "--parents", "-n", "1", sealedSourceOID)
	if err != nil {
		t.Fatal(err)
	}
	identity := strings.Fields(string(line))
	if len(identity) != 2 || identity[0] != sealedSourceOID ||
		identity[1] != fixture.ProductDeltaSealedGitCommitOID {
		t.Fatalf("V21 S identity=%v want exact single-parent [S=%q P=%q]",
			identity, sealedSourceOID, fixture.ProductDeltaSealedGitCommitOID)
	}
	delta, err := evidenceGit(root, "diff", "--name-only",
		fixture.ProductDeltaSealedGitCommitOID, sealedSourceOID, "--")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(delta)); got != v21FixturePath {
		t.Fatalf("V21 P->S delta=%q want only %q", got, v21FixturePath)
	}
}

func v21AssertEvidenceOutsideSealedTree(
	t *testing.T, root string, fixture v21Fixture, sealedSourceOID string,
) {
	t.Helper()
	for _, relative := range []string{fixture.ReceiptPath, fixture.OutputPath} {
		output, err := evidenceGit(root, "ls-tree", "-r", "--name-only",
			sealedSourceOID, "--", relative)
		if err != nil {
			t.Fatal(err)
		}
		if strings.TrimSpace(string(output)) != "" {
			t.Fatalf("V21 evidence %q is inside sealed source tree %q", relative,
				sealedSourceOID)
		}
		info, err := os.Lstat(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil || !info.Mode().IsRegular() {
			t.Fatalf("V21 E evidence %q must be a regular current-tree file: info=%v err=%v", relative, info, err)
		}
	}
}
