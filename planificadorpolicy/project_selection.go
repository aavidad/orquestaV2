package planificadorpolicy

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
