package planificadorpolicy

import (
	"sort"
	"strings"
)

type ProjectOperationSnapshot struct {
	ObjectivePct int
	MinAgents    int
	MaxAgents    int
	Priority     int
}

type AutomaticProjectCandidateSnapshot struct {
	ProjectID      int64
	Operation      *ProjectOperationSnapshot
	Deficit        int
	HasMin         bool
	LoadRatio      int
	MicroClosed    bool
	ContractClosed bool
}

type FreeTaskCandidateSnapshot struct {
	ID              int64
	Module          string
	Priority        string
	ContractDefined bool
	HasActiveSpec   bool
}

type PlannableAgentCandidateSnapshot struct {
	AgentName   string
	PoolSlug    string
	Priority    int
	ProjectSlug string
}

func DesiredProjectQuota(op *ProjectOperationSnapshot, totalAgents int) int {
	if totalAgents <= 0 {
		totalAgents = 1
	}
	if op == nil {
		return 1
	}
	target := op.ObjectivePct
	if target <= 0 {
		target = 100
	}
	desired := (totalAgents*target + 99) / 100
	if desired <= 0 {
		desired = 1
	}
	if op.MinAgents > desired {
		desired = op.MinAgents
	}
	if op.MaxAgents > 0 && desired > op.MaxAgents {
		desired = op.MaxAgents
	}
	if desired <= 0 {
		return 1
	}
	return desired
}

func ProjectLoad(active, desired int) int {
	if desired <= 0 {
		desired = 1
	}
	return (active * 10000) / desired
}

func PreferAutomaticProjectCandidate(candidate *AutomaticProjectCandidateSnapshot, current *AutomaticProjectCandidateSnapshot) bool {
	if candidate == nil || candidate.ProjectID == 0 {
		return false
	}
	if current == nil || current.ProjectID == 0 {
		return true
	}
	if candidate.HasMin != current.HasMin {
		return candidate.HasMin
	}
	if candidate.MicroClosed != current.MicroClosed {
		return candidate.MicroClosed
	}
	if candidate.ContractClosed != current.ContractClosed {
		return candidate.ContractClosed
	}
	if candidate.Deficit > 0 || current.Deficit > 0 {
		if candidate.Deficit != current.Deficit {
			return candidate.Deficit > current.Deficit
		}
	}
	if candidate.LoadRatio != current.LoadRatio {
		return candidate.LoadRatio < current.LoadRatio
	}
	if candidate.Operation != nil && current.Operation != nil && candidate.Operation.Priority != current.Operation.Priority {
		return candidate.Operation.Priority > current.Operation.Priority
	}
	return candidate.ProjectID < current.ProjectID
}

func SplitAgentList(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n'
	})
	seen := make(map[string]struct{}, len(parts))
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key := strings.ToLower(part)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, part)
	}
	return out
}

func AgentAllowedForAutobootstrapProject(agent, projectSlug, configuredProjectSlug string, allowedAgents []string) bool {
	agent = strings.TrimSpace(agent)
	projectSlug = strings.TrimSpace(projectSlug)
	configuredProjectSlug = strings.TrimSpace(configuredProjectSlug)
	if agent == "" {
		return true
	}
	if projectSlug == "" || configuredProjectSlug == "" || !strings.EqualFold(projectSlug, configuredProjectSlug) {
		return true
	}
	if len(allowedAgents) == 0 {
		return true
	}
	for _, name := range allowedAgents {
		if strings.EqualFold(agent, name) {
			return true
		}
	}
	return false
}

func ScoreFreeTaskCandidate(task *FreeTaskCandidateSnapshot, preferredModule string, occupiedModules map[string]bool, preferMicroprogramming bool) int {
	if task == nil {
		return -1 << 30
	}
	score := 0
	if preferMicroprogramming {
		if task.HasActiveSpec {
			score += 2000
		} else if task.ContractDefined {
			score += 800
		}
	}
	module := strings.TrimSpace(task.Module)
	if module != "" && preferredModule != "" && strings.EqualFold(module, preferredModule) {
		score += 1000
	}
	if module != "" && !occupiedModules[strings.ToLower(module)] {
		score += 200
	}
	switch strings.ToLower(strings.TrimSpace(task.Priority)) {
	case "alta":
		score += 30
	case "media":
		score += 20
	default:
		score += 10
	}
	return score
}

func PreferFreeTaskCandidate(candidate *FreeTaskCandidateSnapshot, current *FreeTaskCandidateSnapshot, preferredModule string, occupiedModules map[string]bool, preferMicroprogramming bool) bool {
	if candidate == nil {
		return false
	}
	if current == nil {
		return true
	}
	scoreCandidate := ScoreFreeTaskCandidate(candidate, preferredModule, occupiedModules, preferMicroprogramming)
	scoreCurrent := ScoreFreeTaskCandidate(current, preferredModule, occupiedModules, preferMicroprogramming)
	if scoreCandidate != scoreCurrent {
		return scoreCandidate > scoreCurrent
	}
	return candidate.ID < current.ID
}

func PrioritizePlannableCandidates(candidates []PlannableAgentCandidateSnapshot, poolCapacities map[string]int) []string {
	if len(candidates) == 0 {
		return nil
	}
	items := append([]PlannableAgentCandidateSnapshot(nil), candidates...)
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		if (a.PoolSlug != "") != (b.PoolSlug != "") {
			return a.PoolSlug != ""
		}
		if a.Priority != b.Priority {
			return a.Priority > b.Priority
		}
		if a.ProjectSlug != b.ProjectSlug {
			return a.ProjectSlug < b.ProjectSlug
		}
		return strings.ToLower(strings.TrimSpace(a.AgentName)) < strings.ToLower(strings.TrimSpace(b.AgentName))
	})

	remainingByPool := make(map[string]int, len(poolCapacities))
	for key, value := range poolCapacities {
		remainingByPool[strings.TrimSpace(key)] = value
	}

	out := make([]string, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.PoolSlug) == "" {
			out = append(out, item.AgentName)
			continue
		}
		if remainingByPool[item.PoolSlug] <= 0 {
			continue
		}
		remainingByPool[item.PoolSlug]--
		out = append(out, item.AgentName)
	}
	return out
}
