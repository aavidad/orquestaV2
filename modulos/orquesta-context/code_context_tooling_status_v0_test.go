package orquestacontext

import (
	"testing"
	"time"
)

func TestBuildCodeContextToolingStatusV0PideParadaParaLeaseExpirado(t *testing.T) {
	started := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	lease := validCodeContextToolLeaseTestV0(started, 10)

	status, err := BuildCodeContextToolingStatusV0(CodeContextToolingStatusRequestV0{
		RequestRef:    "request-ref-codebase-status",
		RepositoryRef: lease.RepositoryRef,
		ObservedAt:    started.Add(time.Minute).Format(time.RFC3339),
		Leases:        []CodeContextToolLeaseV0{lease},
		Observations: []CodeContextToolLeaseObservationV0{{
			ObservedAt:     started.Add(time.Minute).Format(time.RFC3339),
			Lease:          lease,
			CPUPercent:     91,
			CPUHighPercent: 75,
			ActiveRequests: 0,
			EvidenceRefs:   []string{"evidence-ref-proc-snapshot"},
			ObservationRef: "observation-ref-codebase-process",
		}},
	})
	if err != nil {
		t.Fatalf("build status: %v", err)
	}
	if status.Estado != CodeContextToolingEstadoAttentionRequiredV0 ||
		status.StopRequested != 1 ||
		status.HighCPUStopRequested != 1 ||
		status.ActiveLeases != 1 {
		t.Fatalf("status=%+v", status)
	}
	if len(status.Entries) != 1 ||
		!status.Entries[0].ShouldRequestStop ||
		status.Entries[0].ReasonCode != CodeContextToolLeaseReasonLeaseExpiredHighCPUV0 {
		t.Fatalf("entries=%+v", status.Entries)
	}
	if !containsContextStringV0(status.NextActions, CodeContextToolingActionStopExpiredLeaseV0) {
		t.Fatalf("next_actions=%+v", status.NextActions)
	}
}

func TestBuildCodeContextToolingStatusV0NoParaLeaseConPeticionesActivas(t *testing.T) {
	started := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	lease := validCodeContextToolLeaseTestV0(started, 10)

	status, err := BuildCodeContextToolingStatusV0(CodeContextToolingStatusRequestV0{
		ObservedAt: started.Add(time.Minute).Format(time.RFC3339),
		Leases:     []CodeContextToolLeaseV0{lease},
		Observations: []CodeContextToolLeaseObservationV0{{
			Lease:          lease,
			CPUPercent:     95,
			ActiveRequests: 2,
		}},
	})
	if err != nil {
		t.Fatalf("build status: %v", err)
	}
	if status.Estado != CodeContextToolingEstadoOKV0 ||
		status.StopRequested != 0 ||
		status.Entries[0].Decision != CodeContextToolLeaseDecisionContinueV0 ||
		status.Entries[0].ReasonCode != CodeContextToolLeaseReasonActiveRequestsV0 {
		t.Fatalf("status=%+v", status)
	}
}

func TestBuildCodeContextToolingStatusV0CuentaLeaseTerminal(t *testing.T) {
	started := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	lease := validCodeContextToolLeaseTestV0(started, 10)
	lease.Status = CodeContextToolLeaseStatusCompletedV0

	status, err := BuildCodeContextToolingStatusV0(CodeContextToolingStatusRequestV0{
		ObservedAt: started.Add(time.Minute).Format(time.RFC3339),
		Leases:     []CodeContextToolLeaseV0{lease},
	})
	if err != nil {
		t.Fatalf("build status: %v", err)
	}
	if status.Estado != CodeContextToolingEstadoOKV0 ||
		status.TerminalLeases != 1 ||
		status.ActiveLeases != 0 ||
		status.Entries[0].ReasonCode != CodeContextToolLeaseReasonTerminalV0 {
		t.Fatalf("status=%+v", status)
	}
}
