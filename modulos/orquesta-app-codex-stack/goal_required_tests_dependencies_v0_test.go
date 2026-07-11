package orquestaappcodexstack

import (
	"context"
	"errors"
	"strings"
	"testing"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

func TestResolveGoalRequiredTestDependencyCommandsV0TickInputExigeOrchestrationCoreV0(t *testing.T) {
	graph := goalRequiredTestDependencyGraphForTestV0()
	result := ResolveGoalRequiredTestDependencyCommandsV0(GoalRequiredTestDependencyRequestV0{
		WriteSet: []string{"modulos/orquesta-director-tick-input/tick_input_usecase_v0.go"},
		ExistingCommands: []string{
			"go test -count=1 ./modulos/orquesta-director-tick-input",
		},
	}, graph)

	if !dependencyRequiredTestCommandContainsForTestV0(result.Commands, "./modulos/orquesta-orchestration-core") {
		t.Fatalf("commands=%v, falta orchestration-core", result.Commands)
	}
	if dependencyRequiredTestCommandContainsForTestV0(result.Commands, "./modulos/orquesta-director-tick-input") {
		t.Fatalf("duplico test ya cubierto: %v", result.Commands)
	}
	if len(result.EvidenceRefs) == 0 {
		t.Fatalf("sin evidencia de dependencias: %+v", result)
	}
}

func TestResolveGoalRequiredTestDependencyCommandsV0WebNoArrastraMundoEnteroV0(t *testing.T) {
	graph := goalRequiredTestDependencyGraphForTestV0()
	result := ResolveGoalRequiredTestDependencyCommandsV0(GoalRequiredTestDependencyRequestV0{
		WriteSet: []string{"modulos/orquesta-web/nueva_app_wizard_turn_v0.go"},
		ExistingCommands: []string{
			"go test -count=1 ./modulos/orquesta-web",
		},
	}, graph)

	if dependencyRequiredTestCommandContainsForTestV0(result.Commands, "./...") ||
		dependencyRequiredTestCommandContainsForTestV0(result.Commands, "./modulos/orquesta-unrelated-001") ||
		len(result.Commands) > 3 {
		t.Fatalf("web arrastro demasiado: %v", result.Commands)
	}
	if dependencyRequiredTestCommandContainsForTestV0(result.Commands, "./modulos/orquesta-web") {
		t.Fatalf("duplico test web ya cubierto: %v", result.Commands)
	}
}

func TestGoalLauncherWithRequiredTestsDependenciesV0InyectaTestsAntesDeLanzarV0(t *testing.T) {
	inner := &codexStackGoalLauncherForTestV0{}
	resolver := &goalRequiredTestDependencyResolverForTestV0{
		result: GoalRequiredTestDependencyResultV0{
			Commands:     []string{"go test -count=1 ./modulos/orquesta-orchestration-core"},
			EvidenceRefs: []string{"evidence-ref-deps-orchestration-core"},
		},
	}
	launcher := goalLauncherWithRequiredTestsDependenciesFromConfigV0(inner, resolver)

	_, err := launcher.LaunchGoalWorkV0(context.Background(), orquestagoal.GoalWorkSpecV0{
		GoalRef:      "goal-ref-required-tests-deps-001",
		Objective:    "Cambiar tick-input.",
		DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
		WriteSet: []orquestagoal.GoalWriteScopeV0{{
			Path: "modulos/orquesta-director-tick-input/tick_input_usecase_v0.go",
		}},
		RequiredTests: []orquestagoal.GoalRequiredTestV0{{
			TestRef: "test-ref-existing",
			Command: "go test -count=1 ./modulos/orquesta-director-tick-input",
		}},
		RuleRefs:    []orquestagoal.GoalRuleRefV0{{Ref: "AGENTS.md"}},
		ContextRefs: []orquestagoal.GoalContextRefV0{{Ref: "context-ref-required-tests-deps"}},
	})
	if err != nil {
		t.Fatalf("LaunchGoalWorkV0: %v", err)
	}
	if inner.calls != 1 || resolver.calls != 1 {
		t.Fatalf("calls inner=%d resolver=%d", inner.calls, resolver.calls)
	}
	if !dependencyGoalSpecHasRequiredCommandForTestV0(inner.lastSpec, "go test -count=1 ./modulos/orquesta-orchestration-core") ||
		!inner.lastSpec.ClosurePolicy.RequireRequiredTests ||
		!stringInSetV0(inner.lastSpec.EvidenceRefs, "evidence-ref-deps-orchestration-core") {
		t.Fatalf("spec sin required deps: %+v", inner.lastSpec)
	}
	for _, test := range inner.lastSpec.RequiredTests {
		if test.Command == "go test -count=1 ./modulos/orquesta-orchestration-core" &&
			(test.CommandRef == "" || test.CommandSHA256 == "" ||
				test.DefinitionSHA256 != orquestagoal.FreezeGoalRequiredTestV0(test).DefinitionSHA256) {
			t.Fatalf("required test dependiente sin congelar: %+v", test)
		}
	}
}

func TestGoalLauncherWithRequiredTestsDependenciesV0FailOpenConEvidenciaSiResolverFallaV0(t *testing.T) {
	inner := &codexStackGoalLauncherForTestV0{}
	resolver := &goalRequiredTestDependencyResolverForTestV0{err: errors.New("go_list_failed")}
	launcher := goalLauncherWithRequiredTestsDependenciesFromConfigV0(inner, resolver)

	_, err := launcher.LaunchGoalWorkV0(context.Background(), orquestagoal.GoalWorkSpecV0{
		GoalRef:      "goal-ref-required-tests-deps-fail-001",
		Objective:    "Cambiar tick-input.",
		DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
		WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "modulos/orquesta-director-tick-input"}},
		RuleRefs:     []orquestagoal.GoalRuleRefV0{{Ref: "AGENTS.md"}},
		ContextRefs:  []orquestagoal.GoalContextRefV0{{Ref: "context-ref-required-tests-deps-fail"}},
	})
	if err != nil {
		t.Fatalf("un fallo de resolucion no debe bloquear el lanzamiento: %v", err)
	}
	if inner.calls != 1 {
		t.Fatalf("el lanzamiento debe continuar con los tests declarados: inner_calls=%d", inner.calls)
	}
	if !stringInSetV0(inner.lastSpec.EvidenceRefs, "evidence-ref-goal-required-test-dependency-resolution-unavailable") {
		t.Fatalf("falta evidencia de degradacion: %+v", inner.lastSpec.EvidenceRefs)
	}
}

type goalRequiredTestDependencyResolverForTestV0 struct {
	calls  int
	result GoalRequiredTestDependencyResultV0
	err    error
}

func (resolver *goalRequiredTestDependencyResolverForTestV0) ResolveGoalRequiredTestDependenciesV0(
	context.Context,
	GoalRequiredTestDependencyRequestV0,
) (GoalRequiredTestDependencyResultV0, error) {
	resolver.calls++
	if resolver.err != nil {
		return GoalRequiredTestDependencyResultV0{}, resolver.err
	}
	return resolver.result, nil
}

func goalRequiredTestDependencyGraphForTestV0() GoalRequiredTestPackageGraphV0 {
	packages := []GoalRequiredTestPackageV0{
		{ImportPath: "orquesta/modulos/orquesta-director-tick-input", TestPath: "./modulos/orquesta-director-tick-input"},
		{ImportPath: "orquesta/modulos/orquesta-orchestration-core", TestPath: "./modulos/orquesta-orchestration-core", Deps: []string{"orquesta/modulos/orquesta-director-tick-input"}},
		{ImportPath: "orquesta/modulos/orquesta-app-director-service", TestPath: "./modulos/orquesta-app-director-service", Deps: []string{"orquesta/modulos/orquesta-orchestration-core", "orquesta/modulos/orquesta-director-tick-input"}},
		{ImportPath: "orquesta/modulos/orquesta-app-codex-stack", TestPath: "./modulos/orquesta-app-codex-stack", Deps: []string{"orquesta/modulos/orquesta-web", "orquesta/modulos/orquesta-orchestration-core"}},
		{ImportPath: "orquesta/cmd/orquesta-server", TestPath: "./cmd/orquesta-server", Deps: []string{"orquesta/modulos/orquesta-web", "orquesta/modulos/orquesta-app-codex-stack", "orquesta/modulos/orquesta-director-tick-input"}},
		{ImportPath: "orquesta/modulos/orquesta-web", TestPath: "./modulos/orquesta-web"},
	}
	for i := 0; i < 8; i++ {
		packages = append(packages, GoalRequiredTestPackageV0{
			ImportPath: "orquesta/modulos/orquesta-unrelated-00" + string(rune('0'+i)),
			TestPath:   "./modulos/orquesta-unrelated-00" + string(rune('0'+i)),
			Deps:       []string{"orquesta/modulos/orquesta-web"},
		})
	}
	return GoalRequiredTestPackageGraphV0{Packages: packages, MaxPackages: 4}
}

func dependencyRequiredTestCommandContainsForTestV0(commands []string, value string) bool {
	for _, command := range commands {
		if strings.Contains(command, value) {
			return true
		}
	}
	return false
}

func dependencyGoalSpecHasRequiredCommandForTestV0(spec orquestagoal.GoalWorkSpecV0, command string) bool {
	for _, test := range spec.RequiredTests {
		if test.Command == command {
			return true
		}
	}
	return false
}
