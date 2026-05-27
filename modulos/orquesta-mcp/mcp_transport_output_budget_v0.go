package orquestamcp

const (
	MCPTransportOutputModeFullV0    = "full"
	MCPTransportOutputModeSummaryV0 = "summary"
	MCPTransportOutputModeBlockedV0 = "blocked"

	MCPTransportOutputFreshnessStaticV0 = "static"
	MCPTransportOutputFreshnessLiveV0   = "live"

	MCPTransportOutputRedactionPublicV0 = "public_redacted"

	MCPTransportDefaultResourceOutputMaxBytesV0 = 64 << 10
	MCPTransportDefaultToolOutputMaxBytesV0     = 64 << 10

	MCPTransportOutputTooLargeV0         = MCPPublicErrOutputTooLargeV0
	MCPTransportResourcePayloadBlockedV0 = MCPPublicErrResourceBlockedV0
	MCPTransportToolPayloadBlockedV0     = MCPPublicErrToolBlockedV0
)

type MCPTransportOutputBudgetV0 struct {
	MaxBytes  int    `json:"max_bytes"`
	Mode      string `json:"mode"`
	Freshness string `json:"freshness"`
	Redaction string `json:"redaction"`
	Overflow  string `json:"overflow_code"`
}

func MCPTransportResourceOutputBudgetV0(freshness string) MCPTransportOutputBudgetV0 {
	return MCPTransportOutputBudgetV0{
		MaxBytes:  MCPTransportDefaultResourceOutputMaxBytesV0,
		Mode:      MCPTransportOutputModeFullV0,
		Freshness: normalizeMCPTransportFreshnessV0(freshness),
		Redaction: MCPTransportOutputRedactionPublicV0,
		Overflow:  MCPTransportResourcePayloadBlockedV0,
	}
}

func MCPTransportToolOutputBudgetV0(freshness string) MCPTransportOutputBudgetV0 {
	return MCPTransportOutputBudgetV0{
		MaxBytes:  MCPTransportDefaultToolOutputMaxBytesV0,
		Mode:      MCPTransportOutputModeFullV0,
		Freshness: normalizeMCPTransportFreshnessV0(freshness),
		Redaction: MCPTransportOutputRedactionPublicV0,
		Overflow:  MCPTransportToolPayloadBlockedV0,
	}
}

func NormalizeMCPTransportOutputBudgetV0(
	budget MCPTransportOutputBudgetV0,
	defaultMaxBytes int,
	defaultOverflow string,
) MCPTransportOutputBudgetV0 {
	if budget.MaxBytes <= 0 {
		budget.MaxBytes = defaultMaxBytes
	}
	switch budget.Mode {
	case MCPTransportOutputModeFullV0, MCPTransportOutputModeSummaryV0, MCPTransportOutputModeBlockedV0:
	default:
		budget.Mode = MCPTransportOutputModeFullV0
	}
	budget.Freshness = normalizeMCPTransportFreshnessV0(budget.Freshness)
	if budget.Redaction == "" {
		budget.Redaction = MCPTransportOutputRedactionPublicV0
	}
	if budget.Overflow == "" {
		budget.Overflow = defaultOverflow
	}
	return budget
}

func normalizeMCPTransportFreshnessV0(value string) string {
	switch value {
	case MCPTransportOutputFreshnessLiveV0:
		return MCPTransportOutputFreshnessLiveV0
	default:
		return MCPTransportOutputFreshnessStaticV0
	}
}
