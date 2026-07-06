package orquestaappcodexstack

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

var defaultGoalRequiredTestDependencyPlatformImportsV0 = []string{
	"orquesta/modulos/orquesta-orchestration-core",
	"orquesta/modulos/orquesta-app-director-service",
	"orquesta/modulos/orquesta-app-codex-stack",
	"orquesta/cmd/orquesta-server",
}

type GoalRequiredTestDependencyResolverPortV0 interface {
	ResolveGoalRequiredTestDependenciesV0(
		ctx context.Context,
		request GoalRequiredTestDependencyRequestV0,
	) (GoalRequiredTestDependencyResultV0, error)
}

type GoalRequiredTestDependencyRequestV0 struct {
	GoalRef          string
	WriteSet         []string
	ExistingCommands []string
}

type GoalRequiredTestDependencyResultV0 struct {
	Commands     []string
	EvidenceRefs []string
}

type GoalRequiredTestPackageGraphV0 struct {
	Packages            []GoalRequiredTestPackageV0
	PlatformImportPaths []string
	MaxPackages         int
}

type GoalRequiredTestPackageV0 struct {
	ImportPath string
	TestPath   string
	Deps       []string
}

type goalLauncherWithRequiredTestsDependenciesV0 struct {
	Inner    orquestagoal.GoalWorkLauncherPortV0
	Resolver GoalRequiredTestDependencyResolverPortV0
}

func goalLauncherWithRequiredTestsDependenciesFromConfigV0(
	launcher orquestagoal.GoalWorkLauncherPortV0,
	resolver GoalRequiredTestDependencyResolverPortV0,
) orquestagoal.GoalWorkLauncherPortV0 {
	if launcher == nil || resolver == nil {
		return launcher
	}
	return goalLauncherWithRequiredTestsDependenciesV0{
		Inner:    launcher,
		Resolver: resolver,
	}
}

func (launcher goalLauncherWithRequiredTestsDependenciesV0) LaunchGoalWorkV0(
	ctx context.Context,
	spec orquestagoal.GoalWorkSpecV0,
) (orquestagoal.GoalLaunchReceiptV0, error) {
	spec = orquestagoal.NormalizeGoalWorkSpecV0(spec)
	if goalSpecHasCodeWriteSetV0(spec) {
		result, err := launcher.Resolver.ResolveGoalRequiredTestDependenciesV0(ctx, GoalRequiredTestDependencyRequestV0{
			GoalRef:          spec.GoalRef,
			WriteSet:         goalRequiredTestDependencyWriteSetPathsV0(spec.WriteSet),
			ExistingCommands: goalRequiredTestCommandsV0(spec.RequiredTests),
		})
		if err != nil {
			// Fail-open gobernado: un fallo de resolucion (go list roto,
			// workdir sin repo) no puede congelar todos los lanzamientos;
			// el goal sale con sus tests declarados y evidencia de la
			// degradacion, y el nightly completo sigue siendo la red dura.
			spec.EvidenceRefs = uniqueDependencyRequiredTestStringsV0(append(
				spec.EvidenceRefs,
				"evidence-ref-goal-required-test-dependency-resolution-unavailable",
			))
			spec = orquestagoal.NormalizeGoalWorkSpecV0(spec)
		} else {
			spec = goalSpecWithDependencyRequiredTestsV0(spec, result)
		}
	}
	return launcher.Inner.LaunchGoalWorkV0(ctx, spec)
}

func ResolveGoalRequiredTestDependencyCommandsV0(
	request GoalRequiredTestDependencyRequestV0,
	graph GoalRequiredTestPackageGraphV0,
) GoalRequiredTestDependencyResultV0 {
	changed := goalRequiredTestChangedPackagesV0(request.WriteSet, graph.Packages)
	if len(changed) == 0 {
		return GoalRequiredTestDependencyResultV0{}
	}
	platform := goalRequiredTestPlatformSetV0(graph.PlatformImportPaths)
	maxPackages := graph.MaxPackages
	if maxPackages <= 0 {
		maxPackages = 4
	}
	affected := make([]GoalRequiredTestPackageV0, 0, len(graph.Packages))
	seen := map[string]bool{}
	for _, pkg := range graph.Packages {
		importPath := strings.TrimSpace(pkg.ImportPath)
		if importPath == "" || seen[importPath] {
			continue
		}
		if !changed[importPath] && !platform[importPath] {
			continue
		}
		if !changed[importPath] && !goalRequiredTestPackageDependsOnAnyV0(pkg, changed) {
			continue
		}
		pkg.TestPath = goalRequiredTestPackageTestPathV0(pkg)
		if pkg.TestPath == "" {
			continue
		}
		seen[importPath] = true
		affected = append(affected, pkg)
	}
	sort.Slice(affected, func(i, j int) bool {
		return affected[i].TestPath < affected[j].TestPath
	})
	if len(affected) > maxPackages {
		affected = affected[:maxPackages]
	}
	var commands []string
	for _, pkg := range affected {
		if goalRequiredTestCommandAlreadyCoversPathV0(request.ExistingCommands, pkg.TestPath) {
			continue
		}
		commands = append(commands, "go test -count=1 "+pkg.TestPath)
	}
	commands = uniqueDependencyRequiredTestStringsV0(commands)
	return GoalRequiredTestDependencyResultV0{
		Commands:     commands,
		EvidenceRefs: goalRequiredTestDependencyEvidenceRefsV0(commands),
	}
}

func goalSpecWithDependencyRequiredTestsV0(
	spec orquestagoal.GoalWorkSpecV0,
	result GoalRequiredTestDependencyResultV0,
) orquestagoal.GoalWorkSpecV0 {
	existing := goalRequiredTestCommandsV0(spec.RequiredTests)
	for _, command := range result.Commands {
		command = strings.TrimSpace(command)
		if command == "" || goalRequiredTestCommandAlreadyExistsV0(existing, command) {
			continue
		}
		spec.RequiredTests = append(spec.RequiredTests, orquestagoal.GoalRequiredTestV0{
			TestRef: "test-ref-goal-dependency-" + shortGoalRequiredTestDependencyHashV0(command),
			Command: command,
		})
		existing = append(existing, command)
	}
	if len(result.Commands) > 0 {
		spec.ClosurePolicy.RequireRequiredTests = true
		spec.EvidenceRefs = uniqueDependencyRequiredTestStringsV0(append(spec.EvidenceRefs, result.EvidenceRefs...))
	}
	return orquestagoal.NormalizeGoalWorkSpecV0(spec)
}

func goalRequiredTestDependencyWriteSetPathsV0(values []orquestagoal.GoalWriteScopeV0) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value.Path)
	}
	return uniqueDependencyRequiredTestStringsV0(out)
}

func goalRequiredTestCommandsV0(values []orquestagoal.GoalRequiredTestV0) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value.Command)
	}
	return uniqueDependencyRequiredTestStringsV0(out)
}

func goalRequiredTestChangedPackagesV0(
	writeSet []string,
	packages []GoalRequiredTestPackageV0,
) map[string]bool {
	changed := map[string]bool{}
	for _, path := range writeSet {
		path = cleanDependencyRequiredTestPathV0(path)
		if path == "" {
			continue
		}
		for _, pkg := range packages {
			testPath := strings.TrimPrefix(goalRequiredTestPackageTestPathV0(pkg), "./")
			if testPath == "" {
				continue
			}
			if path == testPath || strings.HasPrefix(path, testPath+"/") {
				changed[strings.TrimSpace(pkg.ImportPath)] = true
			}
		}
	}
	return changed
}

func goalRequiredTestPackageDependsOnAnyV0(
	pkg GoalRequiredTestPackageV0,
	changed map[string]bool,
) bool {
	for _, dep := range pkg.Deps {
		if changed[strings.TrimSpace(dep)] {
			return true
		}
	}
	return false
}

func goalRequiredTestPlatformSetV0(values []string) map[string]bool {
	if len(values) == 0 {
		values = defaultGoalRequiredTestDependencyPlatformImportsV0
	}
	out := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out[value] = true
		}
	}
	return out
}

func goalRequiredTestPackageTestPathV0(pkg GoalRequiredTestPackageV0) string {
	if path := cleanDependencyRequiredTestPathV0(pkg.TestPath); path != "" {
		return "./" + path
	}
	importPath := strings.TrimSpace(pkg.ImportPath)
	if strings.HasPrefix(importPath, "orquesta/") {
		return "./" + strings.TrimPrefix(importPath, "orquesta/")
	}
	return ""
}

func goalRequiredTestCommandAlreadyCoversPathV0(commands []string, testPath string) bool {
	testPath = strings.TrimSpace(testPath)
	if testPath == "" {
		return false
	}
	for _, command := range commands {
		for _, field := range strings.Fields(command) {
			field = strings.TrimSpace(field)
			if field == "./..." || field == testPath || field == testPath+"/..." {
				return true
			}
			if strings.HasSuffix(field, "/...") &&
				strings.HasPrefix(testPath, strings.TrimSuffix(field, "/...")+"/") {
				return true
			}
		}
	}
	return false
}

func goalRequiredTestCommandAlreadyExistsV0(commands []string, command string) bool {
	command = strings.TrimSpace(command)
	for _, existing := range commands {
		if strings.TrimSpace(existing) == command {
			return true
		}
	}
	return false
}

func goalRequiredTestDependencyEvidenceRefsV0(commands []string) []string {
	out := make([]string, 0, len(commands))
	for _, command := range commands {
		out = append(out, "evidence-ref-goal-required-test-dependency-"+shortGoalRequiredTestDependencyHashV0(command))
	}
	return uniqueDependencyRequiredTestStringsV0(out)
}

func cleanDependencyRequiredTestPathV0(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	value = strings.TrimPrefix(value, "./")
	for strings.Contains(value, "//") {
		value = strings.ReplaceAll(value, "//", "/")
	}
	if value == "." || value == "" || strings.HasPrefix(value, "/") || strings.HasPrefix(value, "../") {
		return ""
	}
	return strings.TrimSuffix(value, "/")
}

func uniqueDependencyRequiredTestStringsV0(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	if out == nil {
		return []string{}
	}
	return out
}

func shortGoalRequiredTestDependencyHashV0(value string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return hex.EncodeToString(sum[:])[:12]
}
