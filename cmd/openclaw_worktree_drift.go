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
	return buildOpenClawWorktreeDriftFromRefs(worktrees, repoHead, heads, relevantAgents), nil
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
	return buildOpenClawWorktreeDriftFromRefs(worktrees, repoHead, heads, relevantAgents), nil
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
