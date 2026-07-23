package acceptance_test

import (
	"bufio"
	"fmt"
	"os"
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
// by their complete current line count. Generated Go and migration SQL remain
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
	generated := v20PathSet(v20GeneratedProductPaths)
	commandOwned := v20CountGoLines(t, repositoryRoot, []string{"internal/commands"}, generated) +
		v20CountPaths(t, repositoryRoot, v20CommandPortPaths)
	commandIntegration := v20CountAddedLines(t, repositoryRoot, v20ApplicationIntegrationPaths)
	commands := commandOwned + commandIntegration

	bindings := v20CountGoLines(t, repositoryRoot, []string{
		"internal/interfaces/httpapi", "internal/interfaces/mcp", "internal/interfaces/cli",
	}, nil) + v20CountAddedLines(t, repositoryRoot, v20BindingIntegrationPaths)
	sdk := v20CountGoLines(t, repositoryRoot, []string{"sdk/commands"}, nil)
	stateOwned := v20CountPaths(t, repositoryRoot, v20StateBootstrapPaths)
	stateIntegration := v20CountAddedLines(t, repositoryRoot, v20StateIntegrationPaths)
	stateAndBootstrap := stateOwned + stateIntegration
	manualContractData := v20CountPaths(t, repositoryRoot, v20ManualContractDataPaths)
	generatedLines := v20CountPaths(t, repositoryRoot, v20GeneratedProductPaths)
	migrationLines := v20CountPaths(t, repositoryRoot, v20MigrationProductPaths)
	manual := commands + bindings + sdk + stateAndBootstrap + manualContractData

	if !reflect.DeepEqual(fixture.GeneratedFiles, v20GeneratedProductPaths) {
		t.Errorf("V20 fixture generated subjects=%v, physical inventory=%v",
			fixture.GeneratedFiles, v20GeneratedProductPaths)
	}
	migrationMatches := v20MatchingPaths(t, repositoryRoot, fixture.MigrationContract.FileGlob)
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

func v20CountGoLines(t *testing.T, repositoryRoot string, relativeRoots []string, excluded map[string]struct{}) int {
	t.Helper()
	lines := 0
	for _, relativeRoot := range relativeRoots {
		absoluteRoot := filepath.Join(repositoryRoot, filepath.FromSlash(relativeRoot))
		info, err := os.Stat(absoluteRoot)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			t.Fatal(err)
		}
		if !info.IsDir() {
			if strings.HasSuffix(absoluteRoot, ".go") && !strings.HasSuffix(absoluteRoot, "_test.go") {
				lines += v20CountFileLines(t, absoluteRoot)
			}
			continue
		}
		err = filepath.WalkDir(absoluteRoot, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			relative, err := filepath.Rel(repositoryRoot, path)
			if err != nil {
				return err
			}
			if _, skip := excluded[filepath.ToSlash(relative)]; !skip {
				lines += v20CountFileLines(t, path)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return lines
}

func v20MatchingPaths(t *testing.T, repositoryRoot, relativeGlob string) []string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(repositoryRoot, filepath.FromSlash(relativeGlob)))
	if err != nil {
		t.Fatal(err)
	}
	relative := make([]string, 0, len(matches))
	for _, match := range matches {
		path, err := filepath.Rel(repositoryRoot, match)
		if err != nil {
			t.Fatal(err)
		}
		relative = append(relative, filepath.ToSlash(path))
	}
	return relative
}

func v20CountPaths(t *testing.T, repositoryRoot string, relativePaths []string) int {
	t.Helper()
	lines := 0
	for _, relative := range relativePaths {
		lines += v20CountFileLines(t, filepath.Join(repositoryRoot, filepath.FromSlash(relative)))
	}
	return lines
}

func v20CountFileLines(t *testing.T, path string) int {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	lines := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines++
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return lines
}

func v20PathSet(paths []string) map[string]struct{} {
	set := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		set[filepath.ToSlash(path)] = struct{}{}
	}
	return set
}

func v20CountAddedLines(t *testing.T, repositoryRoot string, relativePaths []string) int {
	t.Helper()
	args := []string{"-C", repositoryRoot, "diff", "--numstat", v20ContractBaseGitCommitOID, "--"}
	args = append(args, relativePaths...)
	output, err := exec.Command("git", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("measure V20 shared integration delta: %v: %s", err, strings.TrimSpace(string(output)))
	}
	lines := 0
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 3 {
			t.Fatalf("invalid V20 numstat row %q", scanner.Text())
		}
		added, err := strconv.Atoi(fields[0])
		if err != nil {
			t.Fatalf("V20 integration subject %s is not textual: %v", fields[2], err)
		}
		lines += added
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	for _, relative := range relativePaths {
		if _, err := os.Stat(filepath.Join(repositoryRoot, filepath.FromSlash(relative))); err != nil {
			t.Fatal(fmt.Errorf("V20 integration subject %s: %w", relative, err))
		}
	}
	return lines
}
