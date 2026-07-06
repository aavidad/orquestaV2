package main

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestaserver "orquesta/modulos/orquesta-server"
)

const (
	serverGoalRequiredTestsGoListTimeoutV0  = 15 * time.Second
	serverGoalRequiredTestsGoListTemplateV0 = "{{.ImportPath}}|{{.Dir}}|{{join .Deps \"\\t\"}}"
)

type serverGoListGoalRequiredTestDependencyResolverV0 struct {
	projectWorkDir string
	mu             sync.Mutex
	graph          *orquestaappcodexstack.GoalRequiredTestPackageGraphV0
}

func serverGoalRequiredTestDependencyResolverFromConfigV0(
	config orquestaserver.ConfigV0,
) orquestaappcodexstack.GoalRequiredTestDependencyResolverPortV0 {
	root := strings.TrimSpace(config.ProjectWorkDir)
	if root == "" {
		return nil
	}
	return &serverGoListGoalRequiredTestDependencyResolverV0{projectWorkDir: root}
}

func (resolver *serverGoListGoalRequiredTestDependencyResolverV0) ResolveGoalRequiredTestDependenciesV0(
	ctx context.Context,
	request orquestaappcodexstack.GoalRequiredTestDependencyRequestV0,
) (orquestaappcodexstack.GoalRequiredTestDependencyResultV0, error) {
	graph, err := resolver.loadGraphV0(ctx)
	if err != nil {
		return orquestaappcodexstack.GoalRequiredTestDependencyResultV0{}, err
	}
	return orquestaappcodexstack.ResolveGoalRequiredTestDependencyCommandsV0(request, graph), nil
}

func (resolver *serverGoListGoalRequiredTestDependencyResolverV0) loadGraphV0(
	ctx context.Context,
) (orquestaappcodexstack.GoalRequiredTestPackageGraphV0, error) {
	resolver.mu.Lock()
	defer resolver.mu.Unlock()
	if resolver.graph != nil {
		return *resolver.graph, nil
	}
	// Solo se cachea el grafo bueno: un fallo transitorio de go list no debe
	// quedar pegado hasta el reinicio del servidor.
	ctx, cancel := context.WithTimeout(ctx, serverGoalRequiredTestsGoListTimeoutV0)
	defer cancel()
	cmd := exec.CommandContext(
		ctx,
		"go",
		"list",
		"-f",
		serverGoalRequiredTestsGoListTemplateV0,
		"./cmd/...",
		"./modulos/...",
	)
	cmd.Dir = resolver.projectWorkDir
	output, err := cmd.Output()
	if err != nil {
		return orquestaappcodexstack.GoalRequiredTestPackageGraphV0{}, fmt.Errorf("goal_required_tests_go_list_failed: %w", err)
	}
	graph, err := serverGoalRequiredTestDependencyGraphFromGoListOutputV0(
		resolver.projectWorkDir,
		string(output),
	)
	if err != nil {
		return orquestaappcodexstack.GoalRequiredTestPackageGraphV0{}, err
	}
	resolver.graph = &graph
	return graph, nil
}

func serverGoalRequiredTestDependencyGraphFromGoListOutputV0(
	projectWorkDir string,
	output string,
) (orquestaappcodexstack.GoalRequiredTestPackageGraphV0, error) {
	projectWorkDir = strings.TrimSpace(projectWorkDir)
	if projectWorkDir == "" {
		return orquestaappcodexstack.GoalRequiredTestPackageGraphV0{}, fmt.Errorf("goal_required_tests_project_workdir_required")
	}
	var packages []orquestaappcodexstack.GoalRequiredTestPackageV0
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 3)
		if len(parts) < 2 {
			return orquestaappcodexstack.GoalRequiredTestPackageGraphV0{}, fmt.Errorf("goal_required_tests_go_list_line_invalid")
		}
		importPath := strings.TrimSpace(parts[0])
		if !strings.HasPrefix(importPath, "orquesta/") {
			continue
		}
		testPath := serverGoalRequiredTestPathFromDirV0(projectWorkDir, strings.TrimSpace(parts[1]))
		if testPath == "" {
			continue
		}
		deps := []string{}
		if len(parts) == 3 {
			for _, dep := range strings.Split(parts[2], "\t") {
				dep = strings.TrimSpace(dep)
				if strings.HasPrefix(dep, "orquesta/") {
					deps = append(deps, dep)
				}
			}
		}
		packages = append(packages, orquestaappcodexstack.GoalRequiredTestPackageV0{
			ImportPath: importPath,
			TestPath:   testPath,
			Deps:       deps,
		})
	}
	return orquestaappcodexstack.GoalRequiredTestPackageGraphV0{
		Packages:    packages,
		MaxPackages: 4,
	}, nil
}

func serverGoalRequiredTestPathFromDirV0(projectWorkDir string, dir string) string {
	rel, err := filepath.Rel(projectWorkDir, dir)
	if err != nil {
		return ""
	}
	rel = filepath.ToSlash(filepath.Clean(rel))
	if rel == "." || strings.HasPrefix(rel, "../") || strings.HasPrefix(rel, "/") {
		return ""
	}
	return "./" + rel
}
