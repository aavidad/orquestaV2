package autonomiapolicy

import "testing"

func TestSupervisorRoleScore(t *testing.T) {
	if got := SupervisorRoleScore("supervisor"); got != 0 {
		t.Fatalf("unexpected score for supervisor: %d", got)
	}
	if got := SupervisorRoleScore("Programador"); got != 2 {
		t.Fatalf("unexpected score for programador: %d", got)
	}
	if got := SupervisorRoleScore("desconocido"); got != 4 {
		t.Fatalf("unexpected default score: %d", got)
	}
}

func TestPreferSupervisorCandidate(t *testing.T) {
	if got := PreferSupervisorCandidate(
		&SupervisorCandidateSnapshot{AgentName: "a", Preferred: true},
		&SupervisorCandidateSnapshot{AgentName: "b"},
	); !got {
		t.Fatalf("prefered candidate should win")
	}
	if got := PreferSupervisorCandidate(
		&SupervisorCandidateSnapshot{AgentName: "a", Preferred: true, Active: false},
		&SupervisorCandidateSnapshot{AgentName: "b", Active: true},
	); got {
		t.Fatalf("active candidate should outrank inactive preferred")
	}

	if got := PreferSupervisorCandidate(
		&SupervisorCandidateSnapshot{AgentName: "a", Active: true},
		&SupervisorCandidateSnapshot{AgentName: "b", Active: false},
	); !got {
		t.Fatalf("active candidate should win")
	}
}

func TestPreferSupervisorCandidateFallsBackToRoleAndName(t *testing.T) {
	if got := PreferSupervisorCandidate(
		&SupervisorCandidateSnapshot{AgentName: "CodexWorker", RoleScore: 2},
		&SupervisorCandidateSnapshot{AgentName: "CodexSupervisor", RoleScore: 0},
	); got {
		t.Fatalf("lower role score should lose")
	}
	if got := PreferSupervisorCandidate(
		&SupervisorCandidateSnapshot{AgentName: "CodexA", RoleScore: 1},
		&SupervisorCandidateSnapshot{AgentName: "CodexB", RoleScore: 1},
	); !got {
		t.Fatalf("lexicographic tie-break should win")
	}
}

func TestPreferSupervisorCandidatePrefiereNoPrimeATieBreak(t *testing.T) {
	if got := PreferSupervisorCandidate(
		&SupervisorCandidateSnapshot{AgentName: "Codex1", RoleScore: 2, CostTier: AgentCostTier("Codex1")},
		&SupervisorCandidateSnapshot{AgentName: "CodexPg1", RoleScore: 2, CostTier: AgentCostTier("CodexPg1")},
	); !got {
		t.Fatalf("worker no-prime deberia ganar frente a prime con el mismo rol")
	}
}

func TestAgentCostTier(t *testing.T) {
	if got := AgentCostTier("Codex1"); got != 0 {
		t.Fatalf("cost tier inesperado para codex normal: %d", got)
	}
	if got := AgentCostTier("CodexPg1"); got != 1 {
		t.Fatalf("cost tier inesperado para prime: %d", got)
	}
}
