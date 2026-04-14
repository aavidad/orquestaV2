package planificadorpolicy

import "testing"

func TestDesiredProjectQuotaRespectsObjectiveMinAndMax(t *testing.T) {
	if got := DesiredProjectQuota(&ProjectOperationSnapshot{ObjectivePct: 40}, 5); got != 2 {
		t.Fatalf("desired quota=%d, want 2", got)
	}
	if got := DesiredProjectQuota(&ProjectOperationSnapshot{ObjectivePct: 10, MinAgents: 2}, 3); got != 2 {
		t.Fatalf("desired quota=%d, want 2", got)
	}
	if got := DesiredProjectQuota(&ProjectOperationSnapshot{ObjectivePct: 100, MaxAgents: 1}, 4); got != 1 {
		t.Fatalf("desired quota=%d, want 1", got)
	}
}

func TestPreferAutomaticProjectCandidatePrioritizesRealDeficit(t *testing.T) {
	a := &AutomaticProjectCandidateSnapshot{
		ProjectID: 1,
		Operation: &ProjectOperationSnapshot{Priority: 100},
		Deficit:   2,
		LoadRatio: ProjectLoad(2, 4),
	}
	b := &AutomaticProjectCandidateSnapshot{
		ProjectID: 2,
		Operation: &ProjectOperationSnapshot{Priority: 200},
		Deficit:   1,
		LoadRatio: ProjectLoad(1, 2),
	}
	if !PreferAutomaticProjectCandidate(a, b) {
		t.Fatal("should prioritize larger real deficit")
	}
}

func TestPreferAutomaticProjectCandidatePrioritizesMicroClosedWork(t *testing.T) {
	a := &AutomaticProjectCandidateSnapshot{
		ProjectID:      1,
		Operation:      &ProjectOperationSnapshot{Priority: 100},
		Deficit:        1,
		LoadRatio:      ProjectLoad(1, 2),
		MicroClosed:    true,
		ContractClosed: true,
	}
	b := &AutomaticProjectCandidateSnapshot{
		ProjectID:      2,
		Operation:      &ProjectOperationSnapshot{Priority: 200},
		Deficit:        3,
		LoadRatio:      ProjectLoad(0, 3),
		MicroClosed:    false,
		ContractClosed: false,
	}
	if !PreferAutomaticProjectCandidate(a, b) {
		t.Fatal("should prioritize micro-directed work before raw deficit")
	}
}

func TestSplitAgentListDeduplicatesAndTrims(t *testing.T) {
	got := SplitAgentList(" Codex2, codex2 ; Codex3\nCodex4 ")
	if len(got) != 3 || got[0] != "Codex2" || got[1] != "Codex3" || got[2] != "Codex4" {
		t.Fatalf("unexpected list: %#v", got)
	}
}

func TestAgentAllowedForAutobootstrapProject(t *testing.T) {
	allowed := SplitAgentList("Codex2,Codex3")
	if !AgentAllowedForAutobootstrapProject("Codex2", "orquestador", "orquestador", allowed) {
		t.Fatal("Codex2 should be allowed")
	}
	if AgentAllowedForAutobootstrapProject("Codex9", "orquestador", "orquestador", allowed) {
		t.Fatal("Codex9 should not be allowed")
	}
	if !AgentAllowedForAutobootstrapProject("Codex9", "otro", "orquestador", allowed) {
		t.Fatal("other projects should not be restricted")
	}
}
