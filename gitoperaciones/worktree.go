/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package gitoperaciones

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type WorktreeManager struct{}

func (WorktreeManager) CreateWorktree(repoPath, worktreePath, branch, baseRef string) error {
	if err := os.MkdirAll(filepath.Dir(worktreePath), 0o755); err != nil {
		return err
	}
	args := []string{"-C", repoPath, "worktree", "add", "-B", branch, worktreePath}
	if baseRef != "" {
		args = append(args, baseRef)
	}
	cmd := exec.Command("git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git worktree add: %w: %s", err, string(out))
	}
	return nil
}

func (WorktreeManager) CreateDetachedWorktree(repoPath, worktreePath, baseRef string) error {
	if err := os.MkdirAll(filepath.Dir(worktreePath), 0o755); err != nil {
		return err
	}
	args := []string{"-C", repoPath, "worktree", "add", "--detach", worktreePath}
	if strings.TrimSpace(baseRef) != "" {
		args = append(args, baseRef)
	}
	cmd := exec.Command("git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git worktree add --detach: %w: %s", err, string(out))
	}
	return nil
}

func (WorktreeManager) RemoveWorktree(repoPath, worktreePath string) error {
	cmd := exec.Command("git", "-C", repoPath, "worktree", "remove", "--force", worktreePath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git worktree remove: %w: %s", err, string(out))
	}
	return nil
}

func (WorktreeManager) PruneWorktrees(repoPath string) error {
	cmd := exec.Command("git", "-C", repoPath, "worktree", "prune")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git worktree prune: %w: %s", err, string(out))
	}
	return nil
}

func (WorktreeManager) DeleteBranchDescendants(repoPath, branch string) error {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return nil
	}
	cmd := exec.Command("git", "-C", repoPath, "for-each-ref", "--format=%(refname:short)", "refs/heads/"+branch+"/")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git for-each-ref descendants: %w: %s", err, string(out))
	}
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		ref := strings.TrimSpace(scanner.Text())
		if ref == "" {
			continue
		}
		deleteCmd := exec.Command("git", "-C", repoPath, "branch", "-D", ref)
		deleteOut, deleteErr := deleteCmd.CombinedOutput()
		if deleteErr != nil {
			return fmt.Errorf("git branch -D %s: %w: %s", ref, deleteErr, string(deleteOut))
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}
