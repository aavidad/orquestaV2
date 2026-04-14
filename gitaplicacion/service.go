package gitaplicacion

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"orquesta/db"
)

type Store interface {
	ListWorktrees(estado, agente string) ([]*db.Worktree, error)
	ListLocks(estado, agente string) ([]*db.Lock, error)
	ListMerges(proyectoSlug, estado string) ([]*db.GitMergeRequest, error)
	SaveMerge(item *db.GitMergeRequest) (int64, error)
	ResolveProjectID(slug string) (*int64, error)
}

type Service struct {
	store Store
}

type DirtyWorkspaceSyncResult struct {
	SyncedPaths  []string
	RemovedPaths []string
	SkippedPaths []string
}

var validateActiveWorktree = activeWorktreeUsable

func NewService(store Store) *Service {
	return &Service{store: store}
}

type CreateMergeInput struct {
	ProyectoSlug string
	SourceBranch string
	TargetBranch string
	RequestedBy  string
	Estado       string
	SourceCommit string
	MergeCommit  string
	Notas        string
	MetadataJSON string
}

func (s *Service) ListWorktrees(estado, agente string) ([]*db.Worktree, error) {
	return s.store.ListWorktrees(estado, agente)
}

func (s *Service) ListLocks(estado, agente string) ([]*db.Lock, error) {
	return s.store.ListLocks(estado, agente)
}

func (s *Service) ListMerges(proyectoSlug, estado string) ([]*db.GitMergeRequest, error) {
	return s.store.ListMerges(proyectoSlug, estado)
}

func (s *Service) CreateMerge(input CreateMergeInput) (int64, error) {
	proyectoID, err := s.store.ResolveProjectID(input.ProyectoSlug)
	if err != nil {
		return 0, err
	}
	return s.store.SaveMerge(&db.GitMergeRequest{
		ProyectoID:   *proyectoID,
		SourceBranch: input.SourceBranch,
		TargetBranch: input.TargetBranch,
		RequestedBy:  input.RequestedBy,
		Estado:       input.Estado,
		SourceCommit: input.SourceCommit,
		MergeCommit:  input.MergeCommit,
		Notas:        input.Notas,
		MetadataJSON: input.MetadataJSON,
	})
}

func (s *Service) ResolveActiveWorktree(proyectoSlug, agente string) (*db.Worktree, error) {
	worktrees, err := s.store.ListWorktrees("activa", strings.TrimSpace(agente))
	if err != nil {
		return nil, err
	}
	proyectoSlug = strings.TrimSpace(proyectoSlug)
	var proyectoID *int64
	if proyectoSlug != "" {
		proyectoID, err = s.store.ResolveProjectID(proyectoSlug)
		if err != nil {
			return nil, err
		}
	}
	for _, item := range worktrees {
		if item == nil {
			continue
		}
		if proyectoSlug != "" && !worktreeBelongsToProject(item, proyectoSlug, proyectoID) {
			continue
		}
		if !validateActiveWorktree(item) {
			continue
		}
		return item, nil
	}
	if proyectoSlug != "" {
		return nil, fmt.Errorf("no existe worktree activa para agente=%s proyecto=%s", strings.TrimSpace(agente), proyectoSlug)
	}
	return nil, fmt.Errorf("no existe worktree activa para agente=%s", strings.TrimSpace(agente))
}

func (s *Service) IsWorktreeClean(worktreePath string) (bool, error) {
	worktreePath = filepath.Clean(strings.TrimSpace(worktreePath))
	if worktreePath == "" {
		return false, fmt.Errorf("ruta worktree obligatoria")
	}
	raw, err := gitStatusPorcelain(worktreePath)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(raw) == "", nil
}

func (s *Service) SyncDirtyWorkspaceToWorktree(repoPath, worktreePath string) (*DirtyWorkspaceSyncResult, error) {
	repoPath = filepath.Clean(strings.TrimSpace(repoPath))
	worktreePath = filepath.Clean(strings.TrimSpace(worktreePath))
	if repoPath == "" {
		return nil, fmt.Errorf("ruta repo obligatoria")
	}
	if worktreePath == "" {
		return nil, fmt.Errorf("ruta worktree obligatoria")
	}
	if repoPath == worktreePath {
		return &DirtyWorkspaceSyncResult{}, nil
	}
	raw, err := gitStatusPorcelain(repoPath)
	if err != nil {
		return nil, err
	}
	result := &DirtyWorkspaceSyncResult{}
	for _, item := range parseDirtyWorkspaceEntries(raw) {
		if item.Path != "" && skipDirtyWorkspacePath(item.Path) {
			result.SkippedPaths = append(result.SkippedPaths, item.Path)
			continue
		}
		if item.OldPath != "" && skipDirtyWorkspacePath(item.OldPath) {
			result.SkippedPaths = append(result.SkippedPaths, item.OldPath)
			item.OldPath = ""
		}
		if item.OldPath != "" && item.OldPath != item.Path {
			removed, err := removeDirtyWorkspacePath(worktreePath, item.OldPath)
			if err != nil {
				return nil, err
			}
			if removed {
				result.RemovedPaths = append(result.RemovedPaths, item.OldPath)
			}
		}
		if item.Path == "" {
			continue
		}
		src := filepath.Join(repoPath, filepath.FromSlash(item.Path))
		if item.Deleted {
			removed, err := removeDirtyWorkspacePath(worktreePath, item.Path)
			if err != nil {
				return nil, err
			}
			if removed {
				result.RemovedPaths = append(result.RemovedPaths, item.Path)
			}
			continue
		}
		synced, err := copyDirtyWorkspacePath(src, filepath.Join(worktreePath, filepath.FromSlash(item.Path)))
		if err != nil {
			return nil, err
		}
		if synced {
			result.SyncedPaths = append(result.SyncedPaths, item.Path)
		}
	}
	sort.Strings(result.SyncedPaths)
	sort.Strings(result.RemovedPaths)
	sort.Strings(result.SkippedPaths)
	return result, nil
}

func worktreeBelongsToProject(item *db.Worktree, proyectoSlug string, proyectoID *int64) bool {
	if item == nil {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(item.ProyectoSlug), strings.TrimSpace(proyectoSlug)) {
		return true
	}
	return proyectoID != nil && *proyectoID > 0 && item.ProyectoID == *proyectoID
}

func activeWorktreeUsable(item *db.Worktree) bool {
	if item == nil {
		return false
	}
	path := filepath.Clean(strings.TrimSpace(item.RutaAbs))
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return false
	}
	cmd := exec.Command("git", "-C", path, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	top := filepath.Clean(strings.TrimSpace(string(out)))
	return top == path
}

type dirtyWorkspaceEntry struct {
	Path    string
	OldPath string
	Deleted bool
}

func gitStatusPorcelain(repoPath string) (string, error) {
	cmd := exec.Command("git", "-C", repoPath, "status", "--porcelain", "--untracked-files=all")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git status porcelain: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

func parseDirtyWorkspaceEntries(raw string) []dirtyWorkspaceEntry {
	lines := strings.Split(strings.TrimRight(raw, "\n"), "\n")
	items := make([]dirtyWorkspaceEntry, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" || len(line) < 4 {
			continue
		}
		status := line[:2]
		rest := filepath.ToSlash(strings.TrimSpace(line[3:]))
		if rest == "" {
			continue
		}
		entry := dirtyWorkspaceEntry{}
		if idx := strings.LastIndex(rest, " -> "); idx >= 0 {
			entry.OldPath = strings.TrimSpace(rest[:idx])
			entry.Path = strings.TrimSpace(rest[idx+4:])
		} else {
			entry.Path = rest
		}
		entry.Deleted = status[0] == 'D' || status[1] == 'D'
		items = append(items, entry)
	}
	return items
}

func skipDirtyWorkspacePath(path string) bool {
	path = filepath.ToSlash(strings.TrimSpace(path))
	if path == "" {
		return true
	}
	switch path {
	case ".git", ".orquesta-runtime", ".orquesta-worktrees", ".orquesta-inbox.md", "logs":
		return true
	}
	for _, prefix := range []string{".git/", ".orquesta-runtime/", ".orquesta-worktrees/", "logs/"} {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func removeDirtyWorkspacePath(worktreePath, relative string) (bool, error) {
	target := filepath.Join(worktreePath, filepath.FromSlash(relative))
	info, err := os.Lstat(target)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if info.IsDir() {
		if err := os.RemoveAll(target); err != nil {
			return false, err
		}
		return true, nil
	}
	if err := os.Remove(target); err != nil {
		return false, err
	}
	return true, nil
}

func copyDirtyWorkspacePath(src, dst string) (bool, error) {
	info, err := os.Lstat(src)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if info.IsDir() {
		return false, nil
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return false, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(src)
		if err != nil {
			return false, err
		}
		_ = os.RemoveAll(dst)
		if err := os.Symlink(target, dst); err != nil {
			return false, err
		}
		return true, nil
	}
	in, err := os.Open(src)
	if err != nil {
		return false, err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return false, err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return false, err
	}
	if err := out.Chmod(info.Mode().Perm()); err != nil {
		return false, err
	}
	return true, nil
}
