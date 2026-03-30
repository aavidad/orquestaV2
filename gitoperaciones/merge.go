/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package gitoperaciones

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type MergeResult struct {
	SourceCommit string
	MergeCommit  string
}

func MergeBranchIsolated(repoPath, sourceBranch, targetBranch string) (*MergeResult, error) {
	repoPath = strings.TrimSpace(repoPath)
	sourceBranch = strings.TrimSpace(sourceBranch)
	targetBranch = strings.TrimSpace(targetBranch)
	if repoPath == "" || sourceBranch == "" || targetBranch == "" {
		return nil, fmt.Errorf("repoPath, sourceBranch y targetBranch son obligatorios")
	}

	sourceCommit, err := gitOutput(repoPath, "rev-parse", sourceBranch)
	if err != nil {
		return nil, fmt.Errorf("resolver commit origen: %w", err)
	}
	targetCommit, err := gitOutput(repoPath, "rev-parse", targetBranch)
	if err != nil {
		return nil, fmt.Errorf("resolver commit destino: %w", err)
	}
	tempRoot, err := os.MkdirTemp("", "orquesta-merge-")
	if err != nil {
		return nil, err
	}
	worktreePath := filepath.Join(tempRoot, "worktree")
	manager := WorktreeManager{}
	if err := manager.CreateDetachedWorktree(repoPath, worktreePath, targetCommit); err != nil {
		_ = os.RemoveAll(tempRoot)
		return nil, err
	}
	tempBranch := buildMergeTempBranch(targetBranch)
	cleanup := func(abort bool) {
		if abort {
			_, _ = exec.Command("git", "-C", worktreePath, "merge", "--abort").CombinedOutput()
		}
		_ = manager.RemoveWorktree(repoPath, worktreePath)
		_, _ = exec.Command("git", "-C", repoPath, "branch", "-D", tempBranch).CombinedOutput()
		_ = os.RemoveAll(tempRoot)
	}
	if out, err := exec.Command("git", "-C", worktreePath, "checkout", "-b", tempBranch).CombinedOutput(); err != nil {
		cleanup(false)
		return nil, fmt.Errorf("crear rama temporal de merge: %w: %s", err, strings.TrimSpace(string(out)))
	}

	cmd := exec.Command("git", "-C", worktreePath, "merge", "--no-ff", "--no-edit", sourceBranch)
	if out, err := cmd.CombinedOutput(); err != nil {
		cleanup(true)
		return nil, fmt.Errorf("git merge %s -> %s: %w: %s", sourceBranch, targetBranch, err, strings.TrimSpace(string(out)))
	}

	mergeCommit, err := gitOutput(worktreePath, "rev-parse", "HEAD")
	if err != nil {
		cleanup(false)
		return nil, fmt.Errorf("resolver commit merge: %w", err)
	}
	if err := promoteIsolatedMerge(repoPath, worktreePath, targetBranch, targetCommit, mergeCommit); err != nil {
		cleanup(false)
		return nil, err
	}
	cleanup(false)
	return &MergeResult{
		SourceCommit: sourceCommit,
		MergeCommit:  mergeCommit,
	}, nil
}

func buildMergeTempBranch(targetBranch string) string {
	targetBranch = strings.NewReplacer("/", "-", "\\", "-", " ", "-", ":", "-").Replace(strings.TrimSpace(targetBranch))
	targetBranch = strings.Trim(targetBranch, "-")
	if targetBranch == "" {
		targetBranch = "target"
	}
	return fmt.Sprintf("orquesta/merge-tmp/%s-%d", targetBranch, time.Now().UTC().UnixNano())
}

func promoteIsolatedMerge(repoPath, primaryWorktreePath, targetBranch, targetCommit, mergeCommit string) error {
	if currentBranch, err := gitCurrentBranch(repoPath); err == nil && strings.TrimSpace(currentBranch) == targetBranch {
		clean, err := gitWorktreeClean(repoPath)
		if err != nil {
			return err
		}
		if !clean {
			return fmt.Errorf("la rama destino %s está activa en %s con cambios locales", targetBranch, filepath.Clean(strings.TrimSpace(repoPath)))
		}
		if out, err := exec.Command("git", "-C", repoPath, "merge", "--ff-only", mergeCommit).CombinedOutput(); err != nil {
			return fmt.Errorf("promocionar merge aislado a %s: %w: %s", targetBranch, err, strings.TrimSpace(string(out)))
		}
		return nil
	}
	worktrees, err := listGitWorktrees(repoPath)
	if err != nil {
		return err
	}
	ocupadas := make([]string, 0, 1)
	for _, wt := range worktrees {
		if strings.TrimSpace(wt.Branch) != targetBranch {
			continue
		}
		normalized := filepath.Clean(strings.TrimSpace(wt.Path))
		if normalized == filepath.Clean(strings.TrimSpace(primaryWorktreePath)) {
			continue
		}
		ocupadas = append(ocupadas, normalized)
	}
	sort.Strings(ocupadas)
	switch len(ocupadas) {
	case 0:
		if _, err := gitOutput(repoPath, "update-ref", "refs/heads/"+targetBranch, mergeCommit, targetCommit); err != nil {
			return fmt.Errorf("actualizar rama objetivo %s: %w", targetBranch, err)
		}
		return nil
	case 1:
		return fmt.Errorf("la rama destino %s está ocupada por la worktree %s", targetBranch, ocupadas[0])
	default:
		return fmt.Errorf("la rama destino %s está ocupada por múltiples worktrees: %s", targetBranch, strings.Join(ocupadas, ", "))
	}
}

type gitWorktreeRef struct {
	Path   string
	Branch string
}

func listGitWorktrees(repoPath string) ([]gitWorktreeRef, error) {
	out, err := gitOutput(repoPath, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("listar worktrees git: %w", err)
	}
	lines := strings.Split(out, "\n")
	worktrees := make([]gitWorktreeRef, 0, 4)
	current := gitWorktreeRef{}
	flush := func() {
		if strings.TrimSpace(current.Path) == "" {
			return
		}
		worktrees = append(worktrees, current)
		current = gitWorktreeRef{}
	}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			flush()
			continue
		}
		switch {
		case strings.HasPrefix(line, "worktree "):
			flush()
			current.Path = strings.TrimSpace(strings.TrimPrefix(line, "worktree "))
		case strings.HasPrefix(line, "branch "):
			current.Branch = strings.TrimPrefix(strings.TrimSpace(strings.TrimPrefix(line, "branch ")), "refs/heads/")
		}
	}
	flush()
	return worktrees, nil
}

func gitWorktreeClean(repoPath string) (bool, error) {
	out, err := gitOutput(repoPath, "status", "--porcelain")
	if err != nil {
		return false, fmt.Errorf("comprobar limpieza de worktree: %w", err)
	}
	return strings.TrimSpace(out) == "", nil
}

func gitCurrentBranch(repoPath string) (string, error) {
	out, err := gitOutput(repoPath, "symbolic-ref", "--quiet", "--short", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func gitOutput(repoPath string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", repoPath}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}
