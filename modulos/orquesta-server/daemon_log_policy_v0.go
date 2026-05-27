package orquestaserver

import "strings"

const (
	DaemonLogOwnerV0                = "orquesta-server-daemon-operational-log"
	DaemonLogPolicyRefV0            = "daemon-log-policy-v0"
	DaemonLogAccessLocalOnlyV0      = "local_state_dir_only"
	DaemonLogModeSummaryRedactedV0  = "summary_redacted"
	DaemonLogModeLocalRawOptInV0    = "local_raw_opt_in"
	DefaultDaemonLogMaxBytesV0      = 1 << 20
	DefaultDaemonLogMaxRotatedV0    = 3
	DefaultDaemonLogRetentionDaysV0 = 7
)

type DaemonLogPolicyV0 struct {
	Owner             string `json:"owner"`
	PolicyRef         string `json:"policy_ref"`
	Mode              string `json:"mode"`
	Access            string `json:"access"`
	MaxBytes          int64  `json:"max_bytes"`
	MaxRotatedFiles   int    `json:"max_rotated_files"`
	RetentionDays     int    `json:"retention_days"`
	LocalRawEnabled   bool   `json:"local_raw_enabled"`
	LocalRawReason    string `json:"local_raw_reason,omitempty"`
	TerminalEvidence  bool   `json:"terminal_evidence"`
	RedactionRequired bool   `json:"redaction_required"`
}

func NormalizeDaemonLogPolicyV0(policy DaemonLogPolicyV0) DaemonLogPolicyV0 {
	policy.Owner = strings.TrimSpace(policy.Owner)
	if policy.Owner == "" {
		policy.Owner = DaemonLogOwnerV0
	}
	policy.PolicyRef = strings.TrimSpace(policy.PolicyRef)
	if policy.PolicyRef == "" {
		policy.PolicyRef = DaemonLogPolicyRefV0
	}
	policy.Access = strings.TrimSpace(policy.Access)
	if policy.Access == "" {
		policy.Access = DaemonLogAccessLocalOnlyV0
	}
	if policy.MaxBytes <= 0 {
		policy.MaxBytes = DefaultDaemonLogMaxBytesV0
	}
	if policy.MaxRotatedFiles <= 0 {
		policy.MaxRotatedFiles = DefaultDaemonLogMaxRotatedV0
	}
	if policy.RetentionDays <= 0 {
		policy.RetentionDays = DefaultDaemonLogRetentionDaysV0
	}
	policy.LocalRawReason = strings.TrimSpace(policy.LocalRawReason)
	if policy.LocalRawEnabled {
		policy.Mode = DaemonLogModeLocalRawOptInV0
		if !daemonLogPublicReasonV0(policy.LocalRawReason) {
			policy.LocalRawReason = "local_diagnostic"
		}
	} else {
		policy.Mode = DaemonLogModeSummaryRedactedV0
		policy.LocalRawReason = ""
	}
	policy.TerminalEvidence = false
	policy.RedactionRequired = true
	return policy
}

func daemonLogPublicReasonV0(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) < 3 || len(value) > 80 {
		return false
	}
	for _, char := range value {
		if char >= 'a' && char <= 'z' {
			continue
		}
		if char >= '0' && char <= '9' {
			continue
		}
		if char == '_' || char == '-' || char == ':' {
			continue
		}
		return false
	}
	return true
}
