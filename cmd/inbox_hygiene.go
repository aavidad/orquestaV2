package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"orquesta/coordinacion"
	"orquesta/db"
)

var inboxTaskIDPattern = regexp.MustCompile("`?#([0-9]+)\\b")

func higienizarInboxesWorktree(proyecto *db.Proyecto, currentBase string, taskID int64) error {
	if proyecto == nil {
		return nil
	}
	repoRoot := strings.TrimSpace(proyecto.RutaAbs)
	if repoRoot == "" {
		return nil
	}
	currentBase = filepath.Clean(strings.TrimSpace(currentBase))
	candidates, err := listInboxHygieneCandidateDirs(proyecto)
	if err != nil {
		return err
	}
	for _, dir := range candidates {
		if currentBase != "" && filepath.Clean(dir) == currentBase {
			continue
		}
		inboxPath := filepath.Join(dir, ".orquesta-inbox.md")
		raw, err := os.ReadFile(inboxPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		if !worktreeInboxPathUsable(dir) {
			if err := os.Remove(inboxPath); err != nil && !os.IsNotExist(err) {
				return err
			}
			continue
		}
		if taskID > 0 && inboxTaskIDMatches(raw, taskID) {
			if err := os.Remove(inboxPath); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
	}
	return nil
}

func listInboxHygieneCandidateDirs(proyecto *db.Proyecto) ([]string, error) {
	if proyecto == nil {
		return nil, nil
	}
	repoRoot := filepath.Clean(strings.TrimSpace(proyecto.RutaAbs))
	if repoRoot == "" {
		return nil, nil
	}
	out := []string{repoRoot}
	worktreesRoot := filepath.Join(repoRoot, ".orquesta-worktrees")
	if entries, err := os.ReadDir(worktreesRoot); err == nil {
		for _, entry := range entries {
			if entry == nil || !entry.IsDir() {
				continue
			}
			out = append(out, filepath.Join(worktreesRoot, entry.Name()))
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if proyecto.ID > 0 {
		estado := coordinacion.WorktreeActive
		items, err := db.ListarWorktreesCoordRaw(coordinacion.WorktreeFilter{
			ProjectID: &proyecto.ID,
			State:     &estado,
		})
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if item == nil || strings.TrimSpace(item.Path) == "" {
				continue
			}
			out = append(out, filepath.Clean(strings.TrimSpace(item.Path)))
		}
	}
	unique := make([]string, 0, len(out))
	for _, dir := range out {
		dir = filepath.Clean(strings.TrimSpace(dir))
		if dir == "" || slices.Contains(unique, dir) {
			continue
		}
		unique = append(unique, dir)
	}
	return unique, nil
}

func inboxTaskIDMatches(raw []byte, taskID int64) bool {
	if taskID <= 0 || len(raw) == 0 {
		return false
	}
	matches := inboxTaskIDPattern.FindSubmatch(bytes.TrimSpace(raw))
	if len(matches) < 2 {
		return false
	}
	got, err := strconv.ParseInt(strings.TrimSpace(string(matches[1])), 10, 64)
	if err != nil {
		return false
	}
	return got == taskID
}

func worktreeInboxPathUsable(dir string) bool {
	dir = filepath.Clean(strings.TrimSpace(dir))
	if dir == "" {
		return false
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return false
	}
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return filepath.Clean(strings.TrimSpace(string(out))) == dir
}

func escribirInboxArchivoConHigiene(proyecto *db.Proyecto, base string, tareaID int64, contenido string) error {
	base = strings.TrimSpace(base)
	if base == "" {
		return nil
	}
	if err := os.MkdirAll(base, 0o755); err != nil {
		return err
	}
	path := filepath.Join(base, ".orquesta-inbox.md")
	if err := os.WriteFile(path, []byte(contenido), 0o644); err != nil {
		return err
	}
	if err := higienizarInboxesWorktree(proyecto, base, tareaID); err != nil {
		return fmt.Errorf("higienizar inboxes worktree: %w", err)
	}
	return nil
}
