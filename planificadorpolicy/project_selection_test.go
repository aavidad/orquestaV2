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

func TestPreferFreeTaskCandidatePrioritizesMicroprogramming(t *testing.T) {
	a := &FreeTaskCandidateSnapshot{
		ID:              1,
		Module:          "core",
		Priority:        "media",
		ContractDefined: true,
		HasActiveSpec:   true,
	}
	b := &FreeTaskCandidateSnapshot{
		ID:              2,
		Module:          "core",
		Priority:        "alta",
		ContractDefined: false,
	}
	if !PreferFreeTaskCandidate(a, b, "", map[string]bool{}, true) {
		t.Fatal("microprogrammed task should win")
	}
}

func TestPreferFreeTaskCandidatePrefersFreePreferredModule(t *testing.T) {
	a := &FreeTaskCandidateSnapshot{
		ID:       1,
		Module:   "web",
		Priority: "alta",
	}
	b := &FreeTaskCandidateSnapshot{
		ID:       2,
		Module:   "runtime",
		Priority: "alta",
	}
	occupied := map[string]bool{"runtime": true}
	if !PreferFreeTaskCandidate(a, b, "web", occupied, false) {
		t.Fatal("preferred free module should win")
	}
}

func TestPrioritizePlannableCandidates(t *testing.T) {
	candidates := []PlannableAgentCandidateSnapshot{
		{AgentName: "Codex2", PoolSlug: "", Priority: 1, ProjectSlug: "zzz"},
		{AgentName: "GemmaB", PoolSlug: "ollama", Priority: 1, ProjectSlug: "bbb"},
		{AgentName: "GemmaA", PoolSlug: "ollama", Priority: 3, ProjectSlug: "aaa"},
	}
	got := PrioritizePlannableCandidates(candidates, map[string]int{"ollama": 1})
	if len(got) != 2 || got[0] != "GemmaA" || got[1] != "Codex2" {
		t.Fatalf("unexpected prioritized agents: %#v", got)
	}
}

func TestResolvePreferredProject(t *testing.T) {
	if got := ResolvePreferredProject(10, true, 20, 30, true); got.ProjectID != 10 || got.Priority != 3 {
		t.Fatalf("unexpected active resolution: %+v", got)
	}
	if got := ResolvePreferredProject(10, false, 20, 30, true); got.ProjectID != 20 || got.Priority != 2 {
		t.Fatalf("unexpected paused resolution: %+v", got)
	}
	if got := ResolvePreferredProject(0, false, 0, 30, true); got.ProjectID != 30 || got.Priority != 1 {
		t.Fatalf("unexpected automatic resolution: %+v", got)
	}
	if got := ResolvePreferredProject(0, false, 0, 30, false); got.ProjectID != 0 || got.Priority != 0 {
		t.Fatalf("unexpected empty resolution: %+v", got)
	}
}

func TestLooksLikeManualOperatorOutsideFleet(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{name: "", want: false},
		{name: "codex-worker", want: false},
		{name: " Claude-op ", want: false},
		{name: "gemini-runtime", want: false},
		{name: "ollama-local", want: false},
		{name: "antigravity-bot", want: false},
		{name: "alberto", want: true},
		{name: "manual-supervisor", want: true},
	}
	for _, tc := range cases {
		if got := LooksLikeManualOperatorOutsideFleet(tc.name); got != tc.want {
			t.Fatalf("LooksLikeManualOperatorOutsideFleet(%q) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestPoolUsesSharedLocalConnector(t *testing.T) {
	cases := []struct {
		name     string
		runtime  string
		metadata string
		want     bool
	}{
		{name: "nil runtime", runtime: "", metadata: `{"conector_canonico":"ollama_pool_local"}`, want: false},
		{name: "wrong runtime", runtime: "openai", metadata: `{"conector_canonico":"ollama_pool_local"}`, want: false},
		{name: "empty metadata", runtime: "ollama", metadata: "", want: false},
		{name: "empty json", runtime: "ollama", metadata: "{}", want: false},
		{name: "invalid json", runtime: "ollama", metadata: "{", want: false},
		{name: "wrong connector", runtime: "ollama", metadata: `{"conector_canonico":"otro"}`, want: false},
		{name: "matching connector", runtime: "ollama", metadata: `{"conector_canonico":"ollama_pool_local"}`, want: true},
	}
	for _, tc := range cases {
		if got := PoolUsesSharedLocalConnector(tc.runtime, tc.metadata); got != tc.want {
			t.Fatalf("%s: PoolUsesSharedLocalConnector(%q, %q) = %v, want %v", tc.name, tc.runtime, tc.metadata, got, tc.want)
		}
	}
}
