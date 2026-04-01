package cmd

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"orquesta/coordinacion"
	"orquesta/db"
)

type apiOpenClawWorktreeDrift struct {
	Agente       string `json:"agente"`
	Branch       string `json:"branch,omitempty"`
	Path         string `json:"path,omitempty"`
	CurrentHead  string `json:"current_head,omitempty"`
	ExpectedHead string `json:"expected_head,omitempty"`
	Dirty        bool   `json:"dirty,omitempty"`
	DirtySummary string `json:"dirty_summary,omitempty"`
}

type gitWorktreeHeadRef struct {
	Path string
	Head string
}

func parseGitWorktreeListPorcelain(raw string) []gitWorktreeHeadRef {
	lines := strings.Split(raw, "\n")
	out := make([]gitWorktreeHeadRef, 0, 4)
	current := gitWorktreeHeadRef{}
	flush := func() {
		if strings.TrimSpace(current.Path) == "" {
			return
		}
		current.Path = filepath.Clean(strings.TrimSpace(current.Path))
		current.Head = strings.TrimSpace(current.Head)
		out = append(out, current)
		current = gitWorktreeHeadRef{}
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
		case strings.HasPrefix(line, "HEAD "):
			current.Head = strings.TrimSpace(strings.TrimPrefix(line, "HEAD "))
		}
	}
	flush()
	return out
}

func currentGitWorktreeHeads() (string, string, map[string]string, error) {
	rootOut, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", "", nil, fmt.Errorf("git root: %w", err)
	}
	repoRoot := filepath.Clean(strings.TrimSpace(string(rootOut)))
	listOut, err := exec.Command("git", "worktree", "list", "--porcelain").Output()
	if err != nil {
		return "", "", nil, fmt.Errorf("git worktree list: %w", err)
	}
	refs := parseGitWorktreeListPorcelain(string(listOut))
	heads := make(map[string]string, len(refs))
	repoHead := ""
	for _, ref := range refs {
		if ref.Path == "" || ref.Head == "" {
			continue
		}
		heads[ref.Path] = ref.Head
		if filepath.Clean(ref.Path) == repoRoot {
			repoHead = ref.Head
		}
	}
	if repoHead == "" {
		return "", "", nil, fmt.Errorf("no se pudo resolver HEAD del repo principal")
	}
	return repoRoot, repoHead, heads, nil
}

func buildOpenClawWorktreeDriftFromRefs(worktrees []*coordinacion.Worktree, repoHead string, heads map[string]string, relevantAgents map[string]struct{}) []apiOpenClawWorktreeDrift {
	if len(worktrees) == 0 || strings.TrimSpace(repoHead) == "" || len(heads) == 0 {
		return nil
	}
	repoHead = strings.TrimSpace(repoHead)
	out := make([]apiOpenClawWorktreeDrift, 0, len(worktrees))
	for _, worktree := range worktrees {
		if worktree == nil {
			continue
		}
		agente := strings.TrimSpace(worktree.Agent)
		if agente == "" {
			continue
		}
		if len(relevantAgents) > 0 {
			if _, ok := relevantAgents[strings.ToLower(agente)]; !ok {
				continue
			}
		}
		path := filepath.Clean(strings.TrimSpace(worktree.Path))
		if path == "" {
			continue
		}
		head := strings.TrimSpace(heads[path])
		if head == "" || head == repoHead {
			continue
		}
		out = append(out, apiOpenClawWorktreeDrift{
			Agente:       agente,
			Branch:       strings.TrimSpace(worktree.Branch),
			Path:         path,
			CurrentHead:  shortGitHash(head),
			ExpectedHead: shortGitHash(repoHead),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Agente == out[j].Agente {
			return out[i].Path < out[j].Path
		}
		return strings.ToLower(out[i].Agente) < strings.ToLower(out[j].Agente)
	})
	return out
}

func buildOpenClawWorktreeDrift(status *estadoResumen) ([]apiOpenClawWorktreeDrift, error) {
	if status == nil {
		return nil, nil
	}
	_, repoHead, heads, err := currentGitWorktreeHeads()
	if err != nil {
		return nil, err
	}
	state := coordinacion.WorktreeActive
	worktrees, err := db.CoordinationWorktreeRepository().List(coordinacion.WorktreeFilter{State: &state})
	if err != nil {
		return nil, err
	}
	relevantAgents := relevantOpenClawWorktreeAgentsFromEstadoResumen(status)
	return enrichOpenClawWorktreeDriftWithDirty(buildOpenClawWorktreeDriftFromRefs(worktrees, repoHead, heads, relevantAgents)), nil
}

func buildOpenClawWorktreeDriftFromAPIStatus(status apiStatusResponse) ([]apiOpenClawWorktreeDrift, error) {
	_, repoHead, heads, err := currentGitWorktreeHeads()
	if err != nil {
		return nil, err
	}
	state := coordinacion.WorktreeActive
	worktrees, err := db.CoordinationWorktreeRepository().List(coordinacion.WorktreeFilter{State: &state})
	if err != nil {
		return nil, err
	}
	relevantAgents := relevantOpenClawWorktreeAgentsFromAPIStatus(status)
	return enrichOpenClawWorktreeDriftWithDirty(buildOpenClawWorktreeDriftFromRefs(worktrees, repoHead, heads, relevantAgents)), nil
}

func relevantOpenClawWorktreeAgentsFromEstadoResumen(status *estadoResumen) map[string]struct{} {
	relevantAgents := make(map[string]struct{}, len(status.AgentesActivos)+len(status.TareasActivas))
	for _, agente := range status.AgentesActivos {
		if agente == nil {
			continue
		}
		relevantAgents[strings.ToLower(strings.TrimSpace(agente.Nombre))] = struct{}{}
	}
	for _, tarea := range status.TareasActivas {
		agente := strings.ToLower(strings.TrimSpace(tarea.Agente))
		if agente == "" {
			continue
		}
		relevantAgents[agente] = struct{}{}
	}
	return relevantAgents
}

func relevantOpenClawWorktreeAgentsFromAPIStatus(status apiStatusResponse) map[string]struct{} {
	relevantAgents := make(map[string]struct{}, len(status.AgentesActivos)+len(status.TareasActivas))
	for _, agente := range status.AgentesActivos {
		if agente == nil {
			continue
		}
		relevantAgents[strings.ToLower(strings.TrimSpace(agente.Nombre))] = struct{}{}
	}
	for _, tarea := range status.TareasActivas {
		agente := strings.ToLower(strings.TrimSpace(tarea.Agente))
		if agente == "" {
			continue
		}
		relevantAgents[agente] = struct{}{}
	}
	return relevantAgents
}

func shortGitHash(hash string) string {
	hash = strings.TrimSpace(hash)
	if len(hash) > 8 {
		return hash[:8]
	}
	return hash
}

func parseGitStatusPorcelainSummary(raw string) (bool, string) {
	tracked := 0
	untracked := 0
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "??") {
			untracked++
			continue
		}
		tracked++
	}
	if tracked == 0 && untracked == 0 {
		return false, ""
	}
	parts := make([]string, 0, 2)
	if tracked > 0 {
		parts = append(parts, fmt.Sprintf("%d tracked", tracked))
	}
	if untracked > 0 {
		parts = append(parts, fmt.Sprintf("%d untracked", untracked))
	}
	return true, strings.Join(parts, " · ")
}

func inspectGitWorktreeDirty(path string) (bool, string) {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "" {
		return false, ""
	}
	out, err := exec.Command("git", "-C", path, "status", "--porcelain").Output()
	if err != nil {
		return false, ""
	}
	return parseGitStatusPorcelainSummary(string(out))
}

func enrichOpenClawWorktreeDriftWithDirty(items []apiOpenClawWorktreeDrift) []apiOpenClawWorktreeDrift {
	if len(items) == 0 {
		return nil
	}
	out := make([]apiOpenClawWorktreeDrift, 0, len(items))
	for _, item := range items {
		copyItem := item
		copyItem.Dirty, copyItem.DirtySummary = inspectGitWorktreeDirty(copyItem.Path)
		out = append(out, copyItem)
	}
	return out
}
