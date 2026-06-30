package orquestacontext

import (
	"context"
	"testing"
	"time"
)

func TestCodeContextToolLeaseV0StoreCompletaLease(t *testing.T) {
	store := NewInMemoryCodeContextToolLeaseStoreV0()
	started := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	lease, err := store.BeginCodeContextToolLeaseV0(context.Background(), CodeContextToolLeaseRequestV0{
		RequestRef:      "request-ref-codebase-lease",
		RepositoryRef:   "repo-ref-orquesta",
		QueryHash:       "code-context-sha256-test",
		ToolRef:         "tool-ref-codebase-central",
		ProviderKind:    CodeContextProviderKindCodebaseMCPV0,
		StartedAt:       started.Format(time.RFC3339),
		LeaseTTLSeconds: 10,
	})
	if err != nil {
		t.Fatalf("begin lease: %v", err)
	}
	if lease.Status != CodeContextToolLeaseStatusActiveV0 ||
		lease.LeaseUntil != started.Add(10*time.Second).Format(time.RFC3339) {
		t.Fatalf("lease=%+v", lease)
	}

	if err := store.FinishCodeContextToolLeaseV0(context.Background(), CodeContextToolLeaseCompletionV0{
		LeaseRef:    lease.LeaseRef,
		ToolRef:     lease.ToolRef,
		CompletedAt: started.Add(time.Second).Format(time.RFC3339),
		Status:      CodeContextToolLeaseCompletionCompletedV0,
	}); err != nil {
		t.Fatalf("finish lease: %v", err)
	}
	completed, err := store.ListCodeContextToolLeasesV0(context.Background(), CodeContextToolLeaseListFilterV0{
		Status: CodeContextToolLeaseStatusCompletedV0,
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(completed) != 1 || completed[0].LeaseRef != lease.LeaseRef {
		t.Fatalf("completed=%+v", completed)
	}
}

func TestCodeContextToolLeaseV0StoreMarcaLeaseStopped(t *testing.T) {
	store := NewInMemoryCodeContextToolLeaseStoreV0()
	started := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	lease, err := store.BeginCodeContextToolLeaseV0(context.Background(), CodeContextToolLeaseRequestV0{
		RequestRef:      "request-ref-codebase-lease",
		RepositoryRef:   "repo-ref-orquesta",
		QueryHash:       "query-hash-test",
		ToolRef:         "tool-ref-codebase-central",
		ProviderKind:    CodeContextProviderKindCodebaseMCPV0,
		StartedAt:       started.Format(time.RFC3339),
		LeaseTTLSeconds: 10,
	})
	if err != nil {
		t.Fatalf("BeginCodeContextToolLeaseV0: %v", err)
	}
	if err := store.FinishCodeContextToolLeaseV0(context.Background(), CodeContextToolLeaseCompletionV0{
		LeaseRef:    lease.LeaseRef,
		ToolRef:     lease.ToolRef,
		CompletedAt: started.Add(2 * time.Second).Format(time.RFC3339),
		Status:      CodeContextToolLeaseCompletionStoppedV0,
	}); err != nil {
		t.Fatalf("FinishCodeContextToolLeaseV0 stopped: %v", err)
	}
	stopped, err := store.ListCodeContextToolLeasesV0(context.Background(), CodeContextToolLeaseListFilterV0{
		Status: CodeContextToolLeaseStatusStoppedV0,
	})
	if err != nil {
		t.Fatalf("ListCodeContextToolLeasesV0 stopped: %v", err)
	}
	if len(stopped) != 1 || stopped[0].LeaseRef != lease.LeaseRef {
		t.Fatalf("stopped=%+v", stopped)
	}
}

func TestEvaluateCodeContextToolLeaseV0PideParadaSiLeaseExpiraSinPeticiones(t *testing.T) {
	started := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	lease := validCodeContextToolLeaseTestV0(started, 10)
	assessment, err := EvaluateCodeContextToolLeaseV0(CodeContextToolLeaseObservationV0{
		ObservedAt:     started.Add(11 * time.Second).Format(time.RFC3339),
		Lease:          lease,
		CPUPercent:     95,
		CPUHighPercent: 75,
		EvidenceRefs:   []string{"evidence-ref-proc-cpu"},
	})
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if !assessment.ShouldRequestStop ||
		assessment.Decision != CodeContextToolLeaseDecisionRequestStopV0 ||
		assessment.ReasonCode != CodeContextToolLeaseReasonLeaseExpiredHighCPUV0 {
		t.Fatalf("assessment=%+v", assessment)
	}
}

func TestEvaluateCodeContextToolLeaseV0NoParaSiHayPeticionesActivas(t *testing.T) {
	started := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	lease := validCodeContextToolLeaseTestV0(started, 10)
	assessment, err := EvaluateCodeContextToolLeaseV0(CodeContextToolLeaseObservationV0{
		ObservedAt:     started.Add(30 * time.Second).Format(time.RFC3339),
		Lease:          lease,
		CPUPercent:     95,
		CPUHighPercent: 75,
		ActiveRequests: 1,
	})
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if assessment.ShouldRequestStop ||
		assessment.Decision != CodeContextToolLeaseDecisionContinueV0 ||
		assessment.ReasonCode != CodeContextToolLeaseReasonActiveRequestsV0 {
		t.Fatalf("assessment=%+v", assessment)
	}
}

func TestEvaluateCodeContextToolLeaseV0NoParaLeaseTerminal(t *testing.T) {
	started := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	lease := validCodeContextToolLeaseTestV0(started, 10)
	lease.Status = CodeContextToolLeaseStatusCompletedV0
	assessment, err := EvaluateCodeContextToolLeaseV0(CodeContextToolLeaseObservationV0{
		ObservedAt: started.Add(time.Minute).Format(time.RFC3339),
		Lease:      lease,
		CPUPercent: 95,
	})
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if assessment.ShouldRequestStop || assessment.ReasonCode != CodeContextToolLeaseReasonTerminalV0 {
		t.Fatalf("assessment=%+v", assessment)
	}
}

func validCodeContextToolLeaseTestV0(started time.Time, ttl int) CodeContextToolLeaseV0 {
	return CodeContextToolLeaseV0{
		SchemaVersion:   CodeContextToolLeaseSchemaVersionV0,
		LeaseRef:        "code-context-tool-lease-test",
		RequestRef:      "request-ref-codebase-lease",
		RepositoryRef:   "repo-ref-orquesta",
		QueryHash:       "code-context-sha256-test",
		ToolRef:         "tool-ref-codebase-central",
		ProviderKind:    CodeContextProviderKindCodebaseMCPV0,
		StartedAt:       started.Format(time.RFC3339),
		LeaseUntil:      started.Add(time.Duration(ttl) * time.Second).Format(time.RFC3339),
		LeaseTTLSeconds: ttl,
		Status:          CodeContextToolLeaseStatusActiveV0,
		EvidenceRefs:    []string{"evidence-ref-lease"},
	}
}
