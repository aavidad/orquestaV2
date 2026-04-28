package gitestadisticasapp

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Stats struct {
	RepoRoot              string   `json:"repo_root,omitempty"`
	CWD                   string   `json:"cwd,omitempty"`
	Branch                string   `json:"branch,omitempty"`
	PendingFiles          []string `json:"pending_files,omitempty"`
	RecentCommitFiles     []string `json:"recent_commit_files,omitempty"`
	TouchedFiles          []string `json:"touched_files,omitempty"`
	PendingAddedLines     int      `json:"pending_added_lines"`
	PendingDeletedLines   int      `json:"pending_deleted_lines"`
	CommittedAddedLines   int      `json:"committed_added_lines"`
	CommittedDeletedLines int      `json:"committed_deleted_lines"`
	PendingShortStat      string   `json:"pending_shortstat,omitempty"`
	RecentCommitCount     int      `json:"recent_commit_count"`
}

type Aggregate struct {
	RepoRoots             []string `json:"repo_roots,omitempty"`
	TouchedFiles          []string `json:"touched_files,omitempty"`
	PendingAddedLines     int      `json:"pending_added_lines"`
	PendingDeletedLines   int      `json:"pending_deleted_lines"`
	CommittedAddedLines   int      `json:"committed_added_lines"`
	CommittedDeletedLines int      `json:"committed_deleted_lines"`
	RecentCommitCount     int      `json:"recent_commit_count"`
}

type gitRunner interface {
	Run(dir string, args ...string) (string, error)
}

type shellRunner struct{}

func (shellRunner) Run(dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	gitArgs := append([]string{"-C", dir}, args...)
	cmd := exec.CommandContext(ctx, "git", gitArgs...)
	cmd.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
		"GCM_INTERACTIVE=Never",
		"GIT_OPTIONAL_LOCKS=0",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

type Service struct {
	runner gitRunner
}

func NewService() *Service {
	return &Service{runner: shellRunner{}}
}

func (s *Service) Collect(cwd string, since time.Time) (*Stats, error) {
	cwd = strings.TrimSpace(cwd)
	if cwd == "" {
		return nil, nil
	}
	root, err := s.runner.Run(cwd, "rev-parse", "--show-toplevel")
	if err != nil {
		return &Stats{CWD: cwd}, nil
	}
	root = strings.TrimSpace(root)
	if !gitRootLooksUsable(root) {
		return &Stats{CWD: cwd}, nil
	}
	branch, _ := s.runner.Run(root, "rev-parse", "--abbrev-ref", "HEAD")
	pendingShortStat, _ := s.runner.Run(root, "diff", "--shortstat")
	pendingNumstat, _ := s.runner.Run(root, "diff", "--numstat")
	pendingStatus, _ := s.runner.Run(root, "status", "--porcelain", "--untracked-files=all")
	commitNumstat, _ := s.runner.Run(root, "log", "--since", since.Format(time.RFC3339), "--numstat", "--format=tformat:")
	commitFilesRaw, _ := s.runner.Run(root, "log", "--since", since.Format(time.RFC3339), "--name-only", "--format=tformat:")
	commitCountRaw, _ := s.runner.Run(root, "rev-list", "--count", "--since="+since.Format(time.RFC3339), "HEAD")

	pendingFiles := parseGitStatusFiles(pendingStatus)
	pendingAdded, pendingDeleted, pendingNumstatFiles := parseGitNumstatTotals(pendingNumstat)
	committedAdded, committedDeleted, recentCommitFiles := parseGitNumstatAndFiles(commitNumstat, commitFilesRaw)
	touchedFiles := mergeStringSets(pendingFiles, pendingNumstatFiles, recentCommitFiles)

	commitCount, _ := strconv.Atoi(strings.TrimSpace(commitCountRaw))
	return &Stats{
		RepoRoot:              root,
		CWD:                   cwd,
		Branch:                strings.TrimSpace(branch),
		PendingFiles:          pendingFiles,
		RecentCommitFiles:     recentCommitFiles,
		TouchedFiles:          touchedFiles,
		PendingAddedLines:     pendingAdded,
		PendingDeletedLines:   pendingDeleted,
		CommittedAddedLines:   committedAdded,
		CommittedDeletedLines: committedDeleted,
		PendingShortStat:      strings.TrimSpace(pendingShortStat),
		RecentCommitCount:     commitCount,
	}, nil
}

func (s *Service) CollectAggregate(cwds []string, since time.Time) (*Aggregate, error) {
	items := make([]*Stats, 0, len(cwds))
	for _, cwd := range cwds {
		item, err := s.Collect(cwd, since)
		if err != nil {
			return nil, err
		}
		if item != nil {
			items = append(items, item)
		}
	}
	agg := AggregateStats(items...)
	return &agg, nil
}

func AggregateStats(items ...*Stats) Aggregate {
	repoRoots := map[string]struct{}{}
	touchedFiles := map[string]struct{}{}
	out := Aggregate{}
	for _, item := range items {
		if item == nil {
			continue
		}
		if repo := strings.TrimSpace(item.RepoRoot); repo != "" {
			repoRoots[repo] = struct{}{}
		}
		for _, file := range item.TouchedFiles {
			file = strings.TrimSpace(file)
			if file == "" {
				continue
			}
			touchedFiles[file] = struct{}{}
		}
		out.PendingAddedLines += item.PendingAddedLines
		out.PendingDeletedLines += item.PendingDeletedLines
		out.CommittedAddedLines += item.CommittedAddedLines
		out.CommittedDeletedLines += item.CommittedDeletedLines
		out.RecentCommitCount += item.RecentCommitCount
	}
	out.RepoRoots = mapKeysSorted(repoRoots)
	out.TouchedFiles = mapKeysSorted(touchedFiles)
	return out
}

func parseGitStatusFiles(raw string) []string {
	set := map[string]struct{}{}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimRight(line, "\r")
		if len(line) < 4 {
			continue
		}
		path := strings.TrimSpace(line[3:])
		if path == "" {
			continue
		}
		if parts := strings.Split(path, " -> "); len(parts) == 2 {
			path = strings.TrimSpace(parts[1])
		}
		set[filepath.Clean(path)] = struct{}{}
	}
	return mapKeysSorted(set)
}

func parseGitNumstatTotals(raw string) (int, int, []string) {
	added := 0
	deleted := 0
	files := map[string]struct{}{}
	for _, line := range strings.Split(raw, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 3 {
			continue
		}
		path := filepath.Clean(strings.Join(fields[2:], " "))
		if n, err := strconv.Atoi(fields[0]); err == nil {
			added += n
		}
		if n, err := strconv.Atoi(fields[1]); err == nil {
			deleted += n
		}
		files[path] = struct{}{}
	}
	return added, deleted, mapKeysSorted(files)
}

func parseGitNumstatAndFiles(numstatRaw, filesRaw string) (int, int, []string) {
	added, deleted, files := parseGitNumstatTotals(numstatRaw)
	fileSet := map[string]struct{}{}
	for _, file := range files {
		fileSet[file] = struct{}{}
	}
	for _, line := range strings.Split(filesRaw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fileSet[filepath.Clean(line)] = struct{}{}
	}
	return added, deleted, mapKeysSorted(fileSet)
}

func mergeStringSets(groups ...[]string) []string {
	set := map[string]struct{}{}
	for _, group := range groups {
		for _, item := range group {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			set[item] = struct{}{}
		}
	}
	return mapKeysSorted(set)
}

func mapKeysSorted(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for key := range set {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func gitRootLooksUsable(root string) bool {
	root = strings.TrimSpace(root)
	if root == "" {
		return false
	}
	info, err := os.Stat(filepath.Join(root, ".git"))
	if err != nil {
		return false
	}
	return info.IsDir() || info.Mode().IsRegular()
}
