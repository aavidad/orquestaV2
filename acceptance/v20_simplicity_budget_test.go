package acceptance_test

import (
	"bufio"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// These subjects are owned by the acceptance test, not by the fixture being
// measured. Tests are excluded. Shared pre-V20 hosts are charged by their
// physical added lines from the accredited V19 base; new V20 files are charged
// by their complete owner-revision line count. Generated Go and migration SQL remain
// separate from manual product LOC.
var (
	v20GeneratedProductPaths = []string{
		"internal/commands/definitions_generated.go",
	}
	v20CommandPortPaths = []string{
		"internal/ports/command_audit.go",
		"internal/ports/command_audit_projection.go",
	}
	v20StateBootstrapPaths = []string{
		"internal/adapters/state/sqlite/command_audit.go",
		"internal/adapters/state/sqlite/command_audit_governance_projection.go",
		"internal/adapters/state/sqlite/recovery_validation_v20.go",
		"internal/bootstrap/command_cli.go",
		"internal/bootstrap/command_surfaces.go",
		"cmd/orquesta/command.go",
	}
	v20ApplicationIntegrationPaths = []string{
		"internal/application/access.go",
		"internal/application/effect_decision.go",
		"internal/application/effect_model.go",
		"internal/application/planning.go",
	}
	v20BindingIntegrationPaths = []string{
		"internal/i18n/catalogs/en.json",
		"internal/i18n/catalogs/es.json",
	}
	v20StateIntegrationPaths = []string{
		"internal/adapters/state/sqlite/errors.go",
		"internal/adapters/state/sqlite/recovery_validation.go",
		"internal/adapters/state/sqlite/recovery_validation_versions.go",
		"internal/bootstrap/runtime.go",
		"cmd/orquesta/main.go",
	}
	v20ManualContractDataPaths = []string{
		"internal/commands/registry.json",
	}
	v20MigrationProductPaths = []string{
		"internal/adapters/state/sqlite/migrations/015_command_registry.sql",
	}
)

func TestV20SimplicityBudgetMeasuresProductionInsteadOfTrustingFixture(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v20Fixture](t, filepath.Join(repositoryRoot, v20FixturePath))
	v20AssertObservedSimplicityBudget(t, repositoryRoot, fixture)
}

func v20AssertObservedSimplicityBudget(t *testing.T, repositoryRoot string, fixture v20Fixture) {
	t.Helper()
	owner := fixture.ProductDeltaSealedGitCommitOID
	generated := v20PathSet(v20GeneratedProductPaths)
	commandOwned := simplicityCountAddedGoLines(t, repositoryRoot, v20ContractBaseGitCommitOID,
		owner, []string{"internal/commands"}, generated) +
		simplicityCountAddedLines(t, repositoryRoot, v20ContractBaseGitCommitOID, owner, v20CommandPortPaths)
	commandIntegration := simplicityCountAddedLines(t, repositoryRoot, v20ContractBaseGitCommitOID,
		owner, v20ApplicationIntegrationPaths)
	commands := commandOwned + commandIntegration

	bindings := simplicityCountAddedGoLines(t, repositoryRoot, v20ContractBaseGitCommitOID, owner, []string{
		"internal/interfaces/httpapi", "internal/interfaces/mcp", "internal/interfaces/cli",
	}, nil) + simplicityCountAddedLines(t, repositoryRoot, v20ContractBaseGitCommitOID,
		owner, v20BindingIntegrationPaths)
	sdk := simplicityCountAddedGoLines(t, repositoryRoot, v20ContractBaseGitCommitOID,
		owner, []string{"sdk/commands"}, nil)
	stateOwned := simplicityCountAddedLines(t, repositoryRoot, v20ContractBaseGitCommitOID,
		owner, v20StateBootstrapPaths)
	stateIntegration := simplicityCountAddedLines(t, repositoryRoot, v20ContractBaseGitCommitOID,
		owner, v20StateIntegrationPaths)
	stateAndBootstrap := stateOwned + stateIntegration
	manualContractData := simplicityCountAddedLines(t, repositoryRoot, v20ContractBaseGitCommitOID,
		owner, v20ManualContractDataPaths)
	generatedLines := simplicityCountAddedLines(t, repositoryRoot, v20ContractBaseGitCommitOID,
		owner, v20GeneratedProductPaths)
	migrationLines := simplicityCountAddedLines(t, repositoryRoot, v20ContractBaseGitCommitOID,
		owner, v20MigrationProductPaths)
	manual := commands + bindings + sdk + stateAndBootstrap + manualContractData

	if !reflect.DeepEqual(fixture.GeneratedFiles, v20GeneratedProductPaths) {
		t.Errorf("V20 fixture generated subjects=%v, physical inventory=%v",
			fixture.GeneratedFiles, v20GeneratedProductPaths)
	}
	migrationMatches := v20MatchingPaths(t, repositoryRoot, owner, fixture.MigrationContract.FileGlob)
	if !reflect.DeepEqual(migrationMatches, v20MigrationProductPaths) {
		t.Errorf("V20 fixture migration subjects=%v, physical inventory=%v",
			migrationMatches, v20MigrationProductPaths)
	}

	observed := map[string]struct{ got, limit int }{
		"manual product":            {manual, fixture.SimplicityBudget.ManualProductLOC},
		"commands/application":      {commands, fixture.SimplicityBudget.CommandsApplicationLOC},
		"bindings":                  {bindings, fixture.SimplicityBudget.BindingAdapterLOC},
		"SDK":                       {sdk, fixture.SimplicityBudget.SDKLOC},
		"SQLite/recovery/bootstrap": {stateAndBootstrap, fixture.SimplicityBudget.SQLiteRecoveryBootstrapLOC},
		"generated":                 {generatedLines, fixture.SimplicityBudget.GeneratedLOC},
		"migration SQL":             {migrationLines, fixture.SimplicityBudget.MigrationSQLLOC},
	}
	t.Logf("V20 physical LOC manual=%d commands=%d (owned=%d integration=%d) bindings=%d SDK=%d state/bootstrap=%d (owned=%d integration=%d) contract_data=%d generated=%d SQL=%d",
		manual, commands, commandOwned, commandIntegration, bindings, sdk,
		stateAndBootstrap, stateOwned, stateIntegration, manualContractData, generatedLines, migrationLines)
	for name, measurement := range observed {
		if measurement.got > measurement.limit {
			t.Errorf("V20 %s LOC=%d exceeds budget=%d", name, measurement.got, measurement.limit)
		}
	}
}

func simplicityCountAddedGoLines(
	t *testing.T, repositoryRoot, baseOID, ownerOID string,
	relativeRoots []string, excluded map[string]struct{},
) int {
	t.Helper()
	return simplicityCountAddedLinesFiltered(t, repositoryRoot, baseOID, ownerOID, relativeRoots,
		func(relative string) bool {
			_, skip := excluded[relative]
			return strings.HasSuffix(relative, ".go") &&
				!strings.HasSuffix(relative, "_test.go") && !skip
		})
}

func v20MatchingPaths(t *testing.T, repositoryRoot, ownerOID, relativeGlob string) []string {
	t.Helper()
	output, err := evidenceGit(repositoryRoot, "ls-tree", "-r", "--name-only", ownerOID)
	if err != nil {
		t.Fatal(err)
	}
	matches := make([]string, 0)
	for _, relative := range strings.Fields(string(output)) {
		match, err := filepath.Match(filepath.FromSlash(relativeGlob), filepath.FromSlash(relative))
		if err != nil {
			t.Fatal(err)
		}
		if match {
			matches = append(matches, filepath.ToSlash(relative))
		}
	}
	return matches
}

func v20PathSet(paths []string) map[string]struct{} {
	set := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		set[filepath.ToSlash(path)] = struct{}{}
	}
	return set
}

func simplicityCountAddedLines(
	t *testing.T, repositoryRoot, baseOID, ownerOID string, relativePaths []string,
) int {
	t.Helper()
	return simplicityCountAddedLinesFiltered(t, repositoryRoot, baseOID, ownerOID, relativePaths, nil)
}

func simplicityCountAddedLinesFiltered(
	t *testing.T, repositoryRoot, baseOID, ownerOID string, relativePaths []string,
	include func(string) bool,
) int {
	t.Helper()
	args := []string{"-C", repositoryRoot, "diff", "--numstat", "--no-renames", baseOID, ownerOID, "--"}
	args = append(args, relativePaths...)
	output, err := exec.Command("git", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("measure owned physical delta %s..%s: %v: %s",
			baseOID, ownerOID, err, strings.TrimSpace(string(output)))
	}
	lines := 0
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		fields := strings.SplitN(scanner.Text(), "\t", 3)
		if len(fields) != 3 {
			t.Fatalf("invalid owned numstat row %q", scanner.Text())
		}
		relative := filepath.ToSlash(fields[2])
		if include != nil && !include(relative) {
			continue
		}
		added, err := strconv.Atoi(fields[0])
		if err != nil {
			t.Fatalf("owned subject %s is not textual: %v", relative, err)
		}
		lines += added
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return lines
}
