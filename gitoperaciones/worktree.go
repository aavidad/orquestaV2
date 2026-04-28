/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package gitoperaciones

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type WorktreeManager struct{}

const worktreeCommandTimeout = 8 * time.Second

func (WorktreeManager) CreateWorktree(repoPath, worktreePath, branch, baseRef string) error {
	start := time.Now()
	worktreeGitDebugf("CreateWorktree step=start repo=%s path=%s branch=%s base=%s", strings.TrimSpace(repoPath), strings.TrimSpace(worktreePath), strings.TrimSpace(branch), strings.TrimSpace(baseRef))
	repoRoot, err := canonicalGitRepoPath(repoPath)
	if err != nil {
		return err
	}
	worktreeGitDebugf("CreateWorktree step=canonical_repo duration=%s repo=%s", time.Since(start).Round(time.Millisecond), repoRoot)
	if err := os.MkdirAll(filepath.Dir(worktreePath), 0o755); err != nil {
		return err
	}
	args := []string{"-C", repoRoot, "worktree", "add", "-B", branch, worktreePath}
	if baseRef != "" {
		args = append(args, baseRef)
	}
	ctx, cancel := context.WithTimeout(context.Background(), worktreeCommandTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("git worktree add timeout tras %s: %s", worktreeCommandTimeout, string(out))
		}
		return fmt.Errorf("git worktree add: %w: %s", err, string(out))
	}
	worktreeGitDebugf("CreateWorktree step=git_add duration=%s repo=%s path=%s branch=%s", time.Since(start).Round(time.Millisecond), repoRoot, worktreePath, branch)
	return nil
}

func (WorktreeManager) CreateDetachedWorktree(repoPath, worktreePath, baseRef string) error {
	repoRoot, err := canonicalGitRepoPath(repoPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(worktreePath), 0o755); err != nil {
		return err
	}
	args := []string{"-C", repoRoot, "worktree", "add", "--detach", worktreePath}
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

func canonicalGitRepoPath(repoPath string) (string, error) {
	repoPath = strings.TrimSpace(repoPath)
	if repoPath == "" {
		return "", fmt.Errorf("repoPath obligatorio")
	}
	ctx, cancel := context.WithTimeout(context.Background(), worktreeCommandTimeout)
	defer cancel()
	root, err := gitOutputContext(ctx, repoPath, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("ruta repo inválida %s: %w", filepath.Clean(repoPath), err)
	}
	root = filepath.Clean(strings.TrimSpace(root))
	if root == string(filepath.Separator) {
		return "", fmt.Errorf("ruta repo inválida %s: toplevel git inesperado /", filepath.Clean(repoPath))
	}
	return root, nil
}

func gitOutputContext(ctx context.Context, repoPath string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", repoPath}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("%s timeout tras %s: %s", strings.Join(args, " "), worktreeCommandTimeout, strings.TrimSpace(string(out)))
		}
		return "", fmt.Errorf("%s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func worktreeGitDebugf(format string, args ...any) {
	for _, key := range []string{"ORQUESTA_DEBUG_PREPARE", "ORQUESTA_DEBUG"} {
		value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
		switch value {
		case "1", "true", "yes", "on", "si", "sí":
			log.Printf("orquesta[prepare-git] "+format, args...)
			return
		}
	}
}
