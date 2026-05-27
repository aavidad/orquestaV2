package orquestaruntime

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

const (
	ProcessRuntimeLaunchReceiptSchemaVersionV0 = "process_runtime_launch_receipt.v0"
	ProcessRuntimeLaunchPolicyRefV0            = "process-runtime-launch-policy-v0"
	ProcessRuntimeLaunchIOPolicyDiscardV0      = "io_discarded"
	ProcessRuntimeLaunchEnvPolicyAllowlistV0   = "env_allowlist_refs_only"
)

type ProcessRuntimeLaunchReceiptV0 struct {
	SchemaVersion string                          `json:"schema_version"`
	ReceiptRef    string                          `json:"receipt_ref"`
	CommandRef    string                          `json:"command_ref"`
	ExecutableRef string                          `json:"executable_ref"`
	ArgRefs       []string                        `json:"arg_refs,omitempty"`
	EnvRefs       []string                        `json:"env_refs,omitempty"`
	WorkingDirRef string                          `json:"working_dir_ref,omitempty"`
	PolicyRef     string                          `json:"policy_ref"`
	PolicyHash    string                          `json:"policy_hash"`
	Cause         string                          `json:"cause"`
	EnvPolicy     string                          `json:"env_policy"`
	IO            ProcessRuntimeIOPolicyReceiptV0 `json:"io"`
}

type ProcessRuntimeIOPolicyReceiptV0 struct {
	StdoutPolicy   string `json:"stdout_policy"`
	StderrPolicy   string `json:"stderr_policy"`
	OutputRedacted bool   `json:"output_redacted"`
	MaxBytes       int    `json:"max_bytes"`
}

func NewProcessRuntimeLaunchReceiptFromSpecV0(
	spec ExternalAgentLaunchSpecV0,
	cause string,
) ProcessRuntimeLaunchReceiptV0 {
	receipt := processRuntimeLaunchReceiptBaseV0(cause)
	receipt.CommandRef = strings.TrimSpace(spec.Command.CommandRef)
	receipt.ExecutableRef = strings.TrimSpace(spec.Command.ExecutableRef)
	receipt.ArgRefs = processRuntimeCompactRefsV0(spec.Command.ArgRefs)
	receipt.EnvRefs = processRuntimeCompactRefsV0(spec.Command.EnvRefs)
	receipt.WorkingDirRef = strings.TrimSpace(spec.Command.WorkingDirRef)
	return finalizeProcessRuntimeLaunchReceiptV0(receipt)
}

func ProcessRuntimeLaunchReceiptRefsV0(
	receipt *ProcessRuntimeLaunchReceiptV0,
) []string {
	if receipt == nil {
		return nil
	}
	return processRuntimeCompactRefsV0([]string{
		receipt.ReceiptRef,
		receipt.PolicyRef,
		"policy-hash-ref-" + processRuntimeReceiptHashPartV0(receipt.PolicyHash),
	})
}

func ProcessRuntimeSnapshotEvidenceRefsV0(snapshot ProcessRuntimeSnapshotV0) []string {
	return processRuntimeCompactRefsV0(append(
		[]string{snapshot.ProcessRef, snapshot.SessionRef, snapshot.LaunchRef},
		snapshot.LaunchReceiptRefs...,
	))
}

func processRuntimeDirectLaunchReceiptV0(req ProcessRuntimeLaunchRequestV0) ProcessRuntimeLaunchReceiptV0 {
	receipt := processRuntimeLaunchReceiptBaseV0("direct_process_runtime_launch")
	receipt.CommandRef = "command-ref-process-runtime-direct"
	receipt.ExecutableRef = "executable-ref-process-runtime-redacted"
	receipt.WorkingDirRef = "working-dir-ref-process-runtime-redacted"
	receipt.ArgRefs = processRuntimeCountRefsV0("arg-ref-process-runtime-redacted", len(req.Args))
	receipt.EnvRefs = processRuntimeCountRefsV0("env-ref-process-runtime-redacted", len(req.Env))
	return finalizeProcessRuntimeLaunchReceiptV0(receipt)
}

func processRuntimeLaunchReceiptBaseV0(cause string) ProcessRuntimeLaunchReceiptV0 {
	if strings.TrimSpace(cause) == "" {
		cause = "command_resolver_resolution"
	}
	return ProcessRuntimeLaunchReceiptV0{
		SchemaVersion: ProcessRuntimeLaunchReceiptSchemaVersionV0,
		PolicyRef:     ProcessRuntimeLaunchPolicyRefV0,
		Cause:         strings.TrimSpace(cause),
		EnvPolicy:     ProcessRuntimeLaunchEnvPolicyAllowlistV0,
		IO: ProcessRuntimeIOPolicyReceiptV0{
			StdoutPolicy:   ProcessRuntimeLaunchIOPolicyDiscardV0,
			StderrPolicy:   ProcessRuntimeLaunchIOPolicyDiscardV0,
			OutputRedacted: true,
			MaxBytes:       0,
		},
	}
}

func finalizeProcessRuntimeLaunchReceiptV0(
	receipt ProcessRuntimeLaunchReceiptV0,
) ProcessRuntimeLaunchReceiptV0 {
	receipt.ArgRefs = processRuntimeCompactRefsV0(receipt.ArgRefs)
	receipt.EnvRefs = processRuntimeCompactRefsV0(receipt.EnvRefs)
	hash := processRuntimeLaunchReceiptHashV0(receipt)
	receipt.PolicyHash = "sha256:" + hash
	receipt.ReceiptRef = "receipt-ref-process-runtime-launch-" + hash[:16]
	return receipt
}

func processRuntimeLaunchReceiptHashV0(receipt ProcessRuntimeLaunchReceiptV0) string {
	parts := []string{
		receipt.SchemaVersion,
		receipt.CommandRef,
		receipt.ExecutableRef,
		strings.Join(receipt.ArgRefs, ","),
		strings.Join(receipt.EnvRefs, ","),
		receipt.WorkingDirRef,
		receipt.PolicyRef,
		receipt.Cause,
		receipt.EnvPolicy,
		receipt.IO.StdoutPolicy,
		receipt.IO.StderrPolicy,
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:])
}

func processRuntimeReceiptHashPartV0(policyHash string) string {
	part := strings.TrimPrefix(strings.TrimSpace(policyHash), "sha256:")
	if len(part) > 16 {
		return part[:16]
	}
	if part == "" {
		return "missing"
	}
	return part
}

func processRuntimeCountRefsV0(prefix string, count int) []string {
	if count <= 0 {
		return nil
	}
	return []string{fmt.Sprintf("%s-count-%03d", prefix, count)}
}

func processRuntimeCompactRefsV0(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}
