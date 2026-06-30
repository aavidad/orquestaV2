package orquestacontext

import (
	"errors"
	"sort"
	"strings"
)

const (
	CodeContextToolingStatusSchemaVersionV0 = "code_context_tooling_status.v0"

	CodeContextToolingEstadoOKV0                = CodeContextEstadoOKV0
	CodeContextToolingEstadoAttentionRequiredV0 = "attention_required"
	CodeContextToolingEstadoErrorV0             = CodeContextEstadoErrorV0

	CodeContextToolingActionStopExpiredLeaseV0           = "stop_expired_code_context_tool_lease"
	CodeContextToolingActionInspectHighCPULeaseV0        = "inspect_high_cpu_code_context_tool_lease"
	CodeContextToolingActionKeepBrokerCentralizedV0      = "keep_code_context_queries_on_orquesta_broker"
	CodeContextToolingActionNoExternalIndexerByDefaultV0 = "do_not_spawn_codebase_memory_mcp_by_default"
)

type CodeContextToolingStatusRequestV0 struct {
	RequestRef            string                              `json:"request_ref,omitempty"`
	CorrelationID         string                              `json:"correlation_id,omitempty"`
	RepositoryRef         string                              `json:"repository_ref,omitempty"`
	ObservedAt            string                              `json:"observed_at"`
	Leases                []CodeContextToolLeaseV0            `json:"leases,omitempty"`
	Observations          []CodeContextToolLeaseObservationV0 `json:"observations,omitempty"`
	DefaultCPUHighPercent int                                 `json:"default_cpu_high_percent,omitempty"`
	EvidenceRefs          []string                            `json:"evidence_refs,omitempty"`
}

type CodeContextToolingStatusV0 struct {
	SchemaVersion        string                            `json:"schema_version"`
	Estado               string                            `json:"estado"`
	RequestRef           string                            `json:"request_ref,omitempty"`
	CorrelationID        string                            `json:"correlation_id,omitempty"`
	RepositoryRef        string                            `json:"repository_ref,omitempty"`
	ObservedAt           string                            `json:"observed_at,omitempty"`
	TotalLeases          int                               `json:"total_leases"`
	ActiveLeases         int                               `json:"active_leases"`
	TerminalLeases       int                               `json:"terminal_leases"`
	StopRequested        int                               `json:"stop_requested"`
	HighCPUStopRequested int                               `json:"high_cpu_stop_requested"`
	Entries              []CodeContextToolingStatusEntryV0 `json:"entries,omitempty"`
	NextActions          []string                          `json:"next_actions,omitempty"`
	Issues               []CodeContextIssueV0              `json:"issues,omitempty"`
	EvidenceRefs         []string                          `json:"evidence_refs,omitempty"`
}

type CodeContextToolingStatusEntryV0 struct {
	LeaseRef          string   `json:"lease_ref"`
	ToolRef           string   `json:"tool_ref"`
	ProviderKind      string   `json:"provider_kind,omitempty"`
	OwnerRef          string   `json:"owner_ref,omitempty"`
	StartedAt         string   `json:"started_at,omitempty"`
	LeaseUntil        string   `json:"lease_until,omitempty"`
	Status            string   `json:"status,omitempty"`
	Decision          string   `json:"decision"`
	ReasonCode        string   `json:"reason_code"`
	CPUPercent        int      `json:"cpu_percent"`
	ActiveRequests    int      `json:"active_requests"`
	ShouldRequestStop bool     `json:"should_request_stop"`
	EvidenceRefs      []string `json:"evidence_refs,omitempty"`
}

func BuildCodeContextToolingStatusV0(
	request CodeContextToolingStatusRequestV0,
) (CodeContextToolingStatusV0, error) {
	request = normalizeCodeContextToolingStatusRequestV0(request)
	if err := validateCodeContextToolingStatusRequestV0(request); err != nil {
		return newCodeContextToolingStatusErrorV0(request, err.Error(), "observed_at"), err
	}
	status := CodeContextToolingStatusV0{
		SchemaVersion: CodeContextToolingStatusSchemaVersionV0,
		Estado:        CodeContextToolingEstadoOKV0,
		RequestRef:    request.RequestRef,
		CorrelationID: request.CorrelationID,
		RepositoryRef: request.RepositoryRef,
		ObservedAt:    request.ObservedAt,
		TotalLeases:   len(request.Leases),
		NextActions: []string{
			CodeContextToolingActionKeepBrokerCentralizedV0,
			CodeContextToolingActionNoExternalIndexerByDefaultV0,
		},
		EvidenceRefs: compactContextStringsV0(request.EvidenceRefs),
	}
	observations := codeContextToolingObservationByLeaseV0(request)
	for _, lease := range request.Leases {
		observation := observations[lease.LeaseRef]
		if observation.Lease.LeaseRef == "" {
			observation = CodeContextToolLeaseObservationV0{
				ObservedAt:     request.ObservedAt,
				Lease:          lease,
				CPUHighPercent: request.DefaultCPUHighPercent,
			}
		}
		assessment, err := EvaluateCodeContextToolLeaseV0(observation)
		if err != nil {
			status.Estado = CodeContextToolingEstadoErrorV0
			status.Issues = append(status.Issues, codeContextIssueV0(
				ErrCodeContextLeaseErrorV0,
				"lease",
				"lease de herramienta de contexto no evaluable",
			))
			continue
		}
		if isCodeContextToolLeaseTerminalV0(lease.Status) {
			status.TerminalLeases++
		} else {
			status.ActiveLeases++
		}
		if assessment.ShouldRequestStop {
			status.StopRequested++
			if !containsContextStringV0(status.NextActions, CodeContextToolingActionStopExpiredLeaseV0) {
				status.NextActions = append(status.NextActions, CodeContextToolingActionStopExpiredLeaseV0)
			}
		}
		if assessment.ReasonCode == CodeContextToolLeaseReasonLeaseExpiredHighCPUV0 {
			status.HighCPUStopRequested++
			if !containsContextStringV0(status.NextActions, CodeContextToolingActionInspectHighCPULeaseV0) {
				status.NextActions = append(status.NextActions, CodeContextToolingActionInspectHighCPULeaseV0)
			}
		}
		status.Entries = append(status.Entries, CodeContextToolingStatusEntryV0{
			LeaseRef:          lease.LeaseRef,
			ToolRef:           lease.ToolRef,
			ProviderKind:      lease.ProviderKind,
			OwnerRef:          lease.OwnerRef,
			StartedAt:         lease.StartedAt,
			LeaseUntil:        lease.LeaseUntil,
			Status:            lease.Status,
			Decision:          assessment.Decision,
			ReasonCode:        assessment.ReasonCode,
			CPUPercent:        assessment.CPUPercent,
			ActiveRequests:    observation.ActiveRequests,
			ShouldRequestStop: assessment.ShouldRequestStop,
			EvidenceRefs:      compactContextStringsV0(assessment.EvidenceRefs),
		})
	}
	if status.Estado != CodeContextToolingEstadoErrorV0 && status.StopRequested > 0 {
		status.Estado = CodeContextToolingEstadoAttentionRequiredV0
	}
	status.NextActions = compactContextStringsV0(status.NextActions)
	sort.Slice(status.Entries, func(i int, j int) bool {
		return status.Entries[i].LeaseRef < status.Entries[j].LeaseRef
	})
	return status, nil
}

func normalizeCodeContextToolingStatusRequestV0(
	request CodeContextToolingStatusRequestV0,
) CodeContextToolingStatusRequestV0 {
	request.RequestRef = trimContextV0(request.RequestRef)
	request.CorrelationID = trimContextV0(request.CorrelationID)
	request.RepositoryRef = trimContextV0(request.RepositoryRef)
	request.ObservedAt = trimContextV0(request.ObservedAt)
	request.EvidenceRefs = compactContextStringsV0(request.EvidenceRefs)
	request.DefaultCPUHighPercent = nonNegativeContextIntV0(request.DefaultCPUHighPercent)
	if request.DefaultCPUHighPercent == 0 {
		request.DefaultCPUHighPercent = 75
	}
	for idx := range request.Leases {
		request.Leases[idx].LeaseRef = trimContextV0(request.Leases[idx].LeaseRef)
		request.Leases[idx].ToolRef = trimContextV0(request.Leases[idx].ToolRef)
		request.Leases[idx].ProviderKind = trimContextV0(request.Leases[idx].ProviderKind)
		request.Leases[idx].OwnerRef = trimContextV0(request.Leases[idx].OwnerRef)
		request.Leases[idx].Status = trimContextV0(request.Leases[idx].Status)
	}
	return request
}

func validateCodeContextToolingStatusRequestV0(request CodeContextToolingStatusRequestV0) error {
	if request.ObservedAt == "" {
		return errors.New(ErrCodeContextCampoRequeridoV0)
	}
	if _, err := parseCodeContextToolLeaseTimeV0(request.ObservedAt); err != nil {
		return errors.New(ErrCodeContextToolLeaseInvalidV0)
	}
	for _, lease := range request.Leases {
		if strings.TrimSpace(lease.LeaseRef) == "" ||
			strings.TrimSpace(lease.ToolRef) == "" ||
			strings.TrimSpace(lease.LeaseUntil) == "" {
			return errors.New(ErrCodeContextToolLeaseInvalidV0)
		}
	}
	return nil
}

func codeContextToolingObservationByLeaseV0(
	request CodeContextToolingStatusRequestV0,
) map[string]CodeContextToolLeaseObservationV0 {
	out := map[string]CodeContextToolLeaseObservationV0{}
	leases := map[string]CodeContextToolLeaseV0{}
	for _, lease := range request.Leases {
		leases[lease.LeaseRef] = lease
	}
	for _, observation := range request.Observations {
		observation = normalizeCodeContextToolLeaseObservationV0(observation)
		leaseRef := observation.Lease.LeaseRef
		if leaseRef == "" {
			continue
		}
		if lease, ok := leases[leaseRef]; ok {
			observation.Lease = lease
		}
		if observation.ObservedAt == "" {
			observation.ObservedAt = request.ObservedAt
		}
		if observation.CPUHighPercent == 0 {
			observation.CPUHighPercent = request.DefaultCPUHighPercent
		}
		out[leaseRef] = observation
	}
	return out
}

func newCodeContextToolingStatusErrorV0(
	request CodeContextToolingStatusRequestV0,
	code string,
	field string,
) CodeContextToolingStatusV0 {
	return CodeContextToolingStatusV0{
		SchemaVersion: CodeContextToolingStatusSchemaVersionV0,
		Estado:        CodeContextToolingEstadoErrorV0,
		RequestRef:    request.RequestRef,
		CorrelationID: request.CorrelationID,
		RepositoryRef: request.RepositoryRef,
		ObservedAt:    request.ObservedAt,
		Issues: []CodeContextIssueV0{{
			Code:  strings.TrimSpace(code),
			Field: strings.TrimSpace(field),
		}},
	}
}

func containsContextStringV0(values []string, needle string) bool {
	needle = strings.TrimSpace(needle)
	for _, value := range values {
		if strings.TrimSpace(value) == needle {
			return true
		}
	}
	return false
}
