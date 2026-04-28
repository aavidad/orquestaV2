package autonomiapolicy

import "testing"

func TestResidentSupervisorPolicyPrefersActiveSupervisorForNormalContinuity(t *testing.T) {
	supervisor := &SupervisorCandidateSnapshot{
		AgentName: "CodexSupervisor",
		Active:    true,
		RoleScore: SupervisorRoleScore("supervisor"),
		CostTier:  AgentCostTier("CodexSupervisor"),
	}
	worker := &SupervisorCandidateSnapshot{
		AgentName: "Codex1",
		Active:    true,
		RoleScore: SupervisorRoleScore("programador"),
		CostTier:  AgentCostTier("Codex1"),
	}

	if !PreferSupervisorCandidate(supervisor, worker) {
		t.Fatalf("el supervisor residente deberia ganar frente a un worker normal en continuidad ordinaria")
	}
}

func TestResidentSupervisorPolicyPrefersNonPrimeWorkerBeforePrimeEscalation(t *testing.T) {
	cheapWorker := &SupervisorCandidateSnapshot{
		AgentName: "Codex2",
		Active:    true,
		RoleScore: SupervisorRoleScore("programador"),
		CostTier:  AgentCostTier("Codex2"),
	}
	primeWorker := &SupervisorCandidateSnapshot{
		AgentName: "CodexPg1",
		Active:    true,
		RoleScore: SupervisorRoleScore("programador"),
		CostTier:  AgentCostTier("CodexPg1"),
	}

	if !PreferSupervisorCandidate(cheapWorker, primeWorker) {
		t.Fatalf("un worker no-prime equivalente deberia ganar antes de escalar a prime")
	}
}

func TestResidentSupervisorPolicyAllowsPrimeWhenItIsTheOnlyRemainingCandidate(t *testing.T) {
	primeWorker := &SupervisorCandidateSnapshot{
		AgentName: "CodexPg1",
		Active:    true,
		RoleScore: SupervisorRoleScore("programador"),
		CostTier:  AgentCostTier("CodexPg1"),
	}

	if !PreferSupervisorCandidate(primeWorker, nil) {
		t.Fatalf("prime deberia seguir siendo seleccionable cuando no hay alternativa previa")
	}
}
