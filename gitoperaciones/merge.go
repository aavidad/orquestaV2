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
	"strings"
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
	tempRoot := filepath.Join(repoPath, ".orquesta-worktrees", ".merge-tmp")
	if err := os.MkdirAll(tempRoot, 0o755); err != nil {
		return nil, err
	}
	worktreePath, err := os.MkdirTemp(tempRoot, "merge-")
	if err != nil {
		return nil, err
	}
	manager := WorktreeManager{}
	if err := manager.CreateWorktree(repoPath, worktreePath, targetBranch, targetBranch); err != nil {
		_ = os.RemoveAll(worktreePath)
		return nil, err
	}
	cleanup := func(abort bool) {
		if abort {
			_, _ = exec.Command("git", "-C", worktreePath, "merge", "--abort").CombinedOutput()
		}
		_ = manager.RemoveWorktree(repoPath, worktreePath)
		_ = os.RemoveAll(worktreePath)
	}

	cmd := exec.Command("git", "-C", worktreePath, "merge", "--no-ff", "--no-edit", sourceBranch)
	if out, err := cmd.CombinedOutput(); err != nil {
		cleanup(true)
		return nil, fmt.Errorf("git merge %s -> %s: %w: %s", sourceBranch, targetBranch, err, strings.TrimSpace(string(out)))
	}

	mergeCommit, err := gitOutput(repoPath, "rev-parse", targetBranch)
	if err != nil {
		cleanup(false)
		return nil, fmt.Errorf("resolver commit merge: %w", err)
	}
	cleanup(false)
	return &MergeResult{
		SourceCommit: sourceCommit,
		MergeCommit:  mergeCommit,
	}, nil
}

func gitOutput(repoPath string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", repoPath}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}
