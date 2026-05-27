package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const guardianCommandTemplateChangedReasonV0 = "command_template_changed"

type guardianCommandPublicIdentityV0 struct {
	CommandRef          string
	CommandProfile      string
	TemplateRef         string
	TemplateStatus      string
	TemplateEvidenceRef []string
}

func guardianReasonCodesV0(result guardianResultV0) []string {
	codes := []string{"guardian_public_result_redacted"}
	if result.Status != "" {
		codes = append(codes, "guardian_status_"+sanitizeFilenamePartV0(result.Status))
	}
	if result.Phase != "" {
		codes = append(codes, "guardian_phase_"+sanitizeFilenamePartV0(result.Phase))
	}
	if result.ArtifactManifest != nil && result.ArtifactManifest.ReasonCode != "" {
		codes = append(codes, result.ArtifactManifest.ReasonCode)
	}
	if strings.TrimSpace(result.Message) == "guardian_healthcheck_required" {
		codes = append(codes, "guardian_healthcheck_required")
	}
	codes = append(codes, guardianSkipHealthReasonCodesV0(result)...)
	if result.Lease != nil && result.Lease.ReasonCode != "" {
		codes = append(codes, result.Lease.ReasonCode)
	}
	if strings.TrimSpace(result.RepairBlockReason) != "" {
		codes = append(codes, strings.TrimSpace(result.RepairBlockReason))
	}
	if strings.TrimSpace(result.FailurePacketHash) != "" {
		codes = append(codes, "guardian_repair_failure_packet_hashed")
	}
	for _, command := range result.Commands {
		if strings.TrimSpace(command.ReasonCode) != "" {
			codes = append(codes, command.ReasonCode)
		}
		if command.StopReceipt != nil && command.StopReceipt.ReasonCode != "" {
			codes = append(codes, command.StopReceipt.ReasonCode)
		}
	}
	return codes
}

func guardianSkipHealthReasonCodesV0(result guardianResultV0) []string {
	refs := strings.Join(result.EvidenceRefs, "\n")
	if !strings.Contains(refs, "evidence-ref-guardian-candidate-not-live-checked") {
		return nil
	}
	codes := []string{"candidate_built"}
	if strings.Contains(refs, "evidence-ref-guardian-candidate-tests-passed") {
		codes = append(codes, "candidate_tests_passed")
	}
	return append(codes, "candidate_not_live_checked", "guardian_skip_health_breakglass_authorized")
}

func guardianCommandReasonCodeV0(command guardianCommandResultV0) string {
	if command.ExitCode == 0 {
		return "command_passed"
	}
	return "command_failed_exit_" + strconv.Itoa(command.ExitCode)
}

func guardianCommandIdentityV0(
	config guardianConfigV0,
	command guardianCommandResultV0,
) guardianCommandPublicIdentityV0 {
	profile := strings.TrimSpace(command.EffectProfile)
	if profile == "" {
		profile = guardianCommandProfileV0(command.Phase)
	}
	templateKey, changed := guardianCommandTemplateKeyV0(config, profile, command.Command)
	templateRef := "guardian-command-template-ref-" + guardianShortHashV0(templateKey)
	attemptRef := strings.TrimSpace(config.AttemptRef)
	if attemptRef == "" {
		attemptRef = "guardian-attempt-ref-missing"
	}
	identity := guardianCommandPublicIdentityV0{
		CommandProfile: profile,
		TemplateRef:    templateRef,
		CommandRef: "guardian-command-ref-" + guardianShortHashV0(strings.Join([]string{
			strings.TrimSpace(command.Phase),
			profile,
			templateRef,
			attemptRef,
		}, "|")),
	}
	if changed {
		identity.TemplateStatus = guardianCommandTemplateChangedReasonV0
		identity.TemplateEvidenceRef = []string{
			"evidence-ref-guardian-command-template-changed-" + guardianShortHashV0(strings.Join([]string{
				strings.TrimSpace(command.Phase),
				profile,
				attemptRef,
			}, "|")),
		}
	}
	return identity
}

func guardianCommandTemplateKeyV0(
	config guardianConfigV0,
	profile string,
	command string,
) (string, bool) {
	switch profile {
	case "build":
		if guardianCommandCanonicalV0(config, profile, command) {
			return "guardian-command-template|build|canonical-go-build", false
		}
	case "required_test":
		if guardianCommandCanonicalV0(config, profile, command) {
			return "guardian-command-template|required_test|go-test", false
		}
	case "healthcheck":
		return "guardian-command-template|healthcheck|candidate-readiness", false
	case "repair":
		if strings.Contains(" "+strings.ToLower(command)+" ", " codex-launch-director-wave ") {
			return "guardian-command-template|repair|codex-launch-director-wave", false
		}
	}
	return "guardian-command-template|" + profile + "|operator-changed", true
}

func guardianManifestRefV0(config guardianConfigV0, result guardianResultV0) string {
	return guardianCausalPublicRefV0(config, result, "guardian-manifest-ref", "manifest")
}

func guardianRepairPacketRefV0(config guardianConfigV0, result guardianResultV0) string {
	return guardianCausalPublicRefV0(config, result, "guardian-repair-packet-ref", "repair_packet",
		result.Phase)
}

func guardianRepairLaunchRefV0(config guardianConfigV0, result guardianResultV0) string {
	return guardianCausalPublicRefV0(config, result, "guardian-repair-launch-packet-ref", "repair_launch_packet",
		result.Phase)
}

func guardianCommandOutputRefV0(config guardianConfigV0, command guardianCommandResultV0) string {
	contentHash := guardianFileContentHashV0(command.OutputPath)
	if contentHash == "" {
		return ""
	}
	return guardianCausalPublicRefV0(config, guardianResultV0{}, "guardian-output-ref", "command_output",
		command.Phase, strconv.Itoa(command.ExitCode), strconv.FormatInt(command.OutputBytes, 10), contentHash)
}

func guardianLocalDiagnosticPublicRefV0(
	config guardianConfigV0,
	result guardianResultV0,
	kind string,
	evidenceRef string,
) string {
	if strings.TrimSpace(evidenceRef) == "" {
		return ""
	}
	return guardianCausalPublicRefV0(config, result, "guardian-local-diagnostic-ref", kind, evidenceRef)
}

func guardianCausalPublicRefV0(
	config guardianConfigV0,
	result guardianResultV0,
	prefix string,
	kind string,
	parts ...string,
) string {
	causal := guardianPublicCausalRefV0(config, result)
	if strings.TrimSpace(causal) == "" {
		return ""
	}
	seed := append([]string{prefix, kind, causal}, compactStringsV0(parts)...)
	return prefix + "-" + guardianShortHashV0(strings.Join(seed, "|"))
}

func guardianPublicCausalRefV0(config guardianConfigV0, result guardianResultV0) string {
	for _, ref := range []string{
		result.AttemptRef,
		config.AttemptRef,
		result.PromotionRef,
		config.PromotionRef,
		result.RunRef,
		config.RunRef,
		result.ShutdownRef,
		config.ShutdownRef,
	} {
		if strings.TrimSpace(ref) != "" {
			return strings.TrimSpace(ref)
		}
	}
	return ""
}

func guardianFileContentHashV0(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return guardianShortHashV0(string(body))
}

func guardianPathRefV0(prefix string, path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	return prefix + "-" + guardianShortHashV0(filepath.Clean(path))
}

func guardianPublicErrorV0(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return "command_error_redacted"
}

func guardianShortHashV0(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])[:16]
}
