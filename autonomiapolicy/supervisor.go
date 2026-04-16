package autonomiapolicy

import "strings"

type SupervisorCandidateSnapshot struct {
	AgentName string
	Preferred bool
	Active    bool
	RoleScore int
}

func SupervisorRoleScore(role string) int {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "supervisor", "orquestador":
		return 0
	case "admin":
		return 1
	case "programador":
		return 2
	case "revisor", "reviewer":
		return 3
	default:
		return 4
	}
}

func PreferSupervisorCandidate(candidate *SupervisorCandidateSnapshot, current *SupervisorCandidateSnapshot) bool {
	if candidate == nil || strings.TrimSpace(candidate.AgentName) == "" {
		return false
	}
	if current == nil || strings.TrimSpace(current.AgentName) == "" {
		return true
	}
	if candidate.Preferred != current.Preferred {
		return candidate.Preferred
	}
	if candidate.Active != current.Active {
		return candidate.Active
	}
	if candidate.RoleScore != current.RoleScore {
		return candidate.RoleScore < current.RoleScore
	}
	return strings.TrimSpace(candidate.AgentName) < strings.TrimSpace(current.AgentName)
}
