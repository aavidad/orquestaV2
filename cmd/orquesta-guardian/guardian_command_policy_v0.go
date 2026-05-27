package main

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"time"
)

const (
	envGuardianCommandEffectEvidenceRefsV0 = "ORQUESTA_GUARDIAN_COMMAND_EFFECT_EVIDENCE_REFS"

	guardianCommandEffectPolicyRefV0     = "guardian-command-effect-profile-policy-v0"
	guardianCommandEffectAllowedV0       = "allowed"
	guardianCommandEffectBlockedV0       = "blocked"
	guardianCommandEffectBlockedReasonV0 = "guardian_command_effect_policy_blocked"
	guardianCommandExpansionReasonV0     = "guardian_command_placeholder_invalid"
)

type guardianCommandEffectDecisionV0 struct {
	Profile       string
	Status        string
	ReasonCode    string
	External      []string
	EvidenceRefs  []string
	EffectPolicy  string
	CommandRef    string
	CommandPublic string
}

func runGuardianPolicyShellCommandV0(
	ctx context.Context,
	config guardianConfigV0,
	phase string,
	rawCommand string,
) guardianCommandResultV0 {
	command, err := expandGuardianCommandCheckedV0(rawCommand, config)
	if err != nil {
		return blockedGuardianCommandResultV0(config, phase, rawCommand, guardianCommandExpansionReasonV0)
	}
	return runGuardianShellCommandV0(ctx, config, phase, command)
}

func authorizeGuardianCommandEffectV0(
	config guardianConfigV0,
	phase string,
	command string,
) guardianCommandEffectDecisionV0 {
	decision := guardianCommandEffectDecisionV0{
		Profile:      guardianCommandProfileV0(phase),
		Status:       guardianCommandEffectAllowedV0,
		EffectPolicy: guardianCommandEffectPolicyRefV0,
		EvidenceRefs: append([]string(nil), config.CommandEffectEvidenceRefs...),
	}
	if !guardianCommandCanonicalV0(config, decision.Profile, command) {
		decision.External = append(decision.External, "custom_shell")
	}
	decision.External = append(decision.External, guardianCommandExternalEffectsV0(command, decision.Profile)...)
	decision.External = compactStringsV0(decision.External)
	if len(decision.External) > 0 && len(decision.EvidenceRefs) == 0 {
		decision.Status = guardianCommandEffectBlockedV0
		decision.ReasonCode = guardianCommandEffectBlockedReasonV0
	}
	return decision
}

func guardianCommandProfileV0(phase string) string {
	phase = strings.TrimSpace(phase)
	switch {
	case phase == "build":
		return "build"
	case strings.HasPrefix(phase, "test-"):
		return "required_test"
	case phase == "healthcheck":
		return "healthcheck"
	case phase == "repair":
		return "repair"
	default:
		return "command"
	}
}

func guardianCommandCanonicalV0(config guardianConfigV0, profile string, command string) bool {
	command = guardianCommandCompactSpaceV0(command)
	switch profile {
	case "build":
		return command == guardianCommandCompactSpaceV0(guardianCanonicalBuildCommandV0(config))
	case "required_test":
		return guardianCommandStartsWithTokenV0(command, "go test")
	case "healthcheck":
		return true
	default:
		return false
	}
}

func guardianCanonicalBuildCommandV0(config guardianConfigV0) string {
	return "go build -o " + shellQuoteV0(config.CandidateBin) + " ./cmd/orquesta-server"
}

func guardianCommandExternalEffectsV0(command string, profile string) []string {
	lower := strings.ToLower(command)
	effects := []string{}
	switch profile {
	case "repair":
		effects = append(effects, "repair_shell")
	}
	for _, item := range []struct {
		needle string
		effect string
	}{
		{"curl ", "network"},
		{"wget ", "network"},
		{"ssh ", "network"},
		{"scp ", "network"},
		{"rsync ", "network"},
		{"git push", "git_remote"},
		{"git pull", "git_remote"},
		{"git fetch", "git_remote"},
		{"git clone", "git_remote"},
		{" gh ", "git_remote"},
		{"opes", "opes"},
		{"codex-launch-wave", "provider_runtime"},
		{"openai", "provider_runtime"},
		{"rm -rf", "destructive"},
		{"sudo ", "destructive"},
		{"docker push", "network"},
	} {
		if strings.Contains(" "+lower+" ", item.needle) {
			effects = append(effects, item.effect)
		}
	}
	return effects
}

func expandGuardianCommandCheckedV0(command string, config guardianConfigV0) (string, error) {
	expanded := expandGuardianCommandV0(command, config)
	for _, placeholder := range guardianCommandPathPlaceholdersV0() {
		if strings.Contains(expanded, placeholder) {
			return "", errors.New(guardianCommandExpansionReasonV0)
		}
	}
	return expanded, nil
}

func guardianCommandPathPlaceholdersV0() []string {
	return []string{
		"{project_dir}",
		"{state_dir}",
		"{current_bin}",
		"{candidate_bin}",
		"{last_good_bin}",
		"{repair_packet}",
	}
}

func blockedGuardianCommandResultV0(
	config guardianConfigV0,
	phase string,
	command string,
	reason string,
) guardianCommandResultV0 {
	outputPath := filepath.Join(config.StateDir, "logs", sanitizeFilenamePartV0(phase)+"-blocked.log")
	_ = writeGuardianRedactedLogV0(config, outputPath, reason)
	decision := authorizeGuardianCommandEffectV0(config, phase, command)
	if reason != guardianCommandEffectBlockedReasonV0 {
		decision.Status = guardianCommandEffectBlockedV0
		decision.ReasonCode = reason
	}
	return guardianCommandResultV0{
		Phase:                   phase,
		Command:                 command,
		ExitCode:                1,
		DurationMS:              int64((0 * time.Second).Milliseconds()),
		OutputPath:              outputPath,
		Error:                   reason,
		ReasonCode:              decision.ReasonCode,
		EffectProfile:           decision.Profile,
		EffectPolicyRef:         decision.EffectPolicy,
		EffectAuthorization:     decision.Status,
		ExternalEffects:         decision.External,
		EffectEvidenceRefs:      decision.EvidenceRefs,
		CommandExpansionChecked: true,
	}
}

func applyGuardianCommandEffectV0(
	result guardianCommandResultV0,
	decision guardianCommandEffectDecisionV0,
) guardianCommandResultV0 {
	result.EffectProfile = decision.Profile
	result.EffectPolicyRef = decision.EffectPolicy
	result.EffectAuthorization = decision.Status
	result.ExternalEffects = decision.External
	result.EffectEvidenceRefs = decision.EvidenceRefs
	result.CommandExpansionChecked = true
	if result.ReasonCode == "" && decision.ReasonCode != "" {
		result.ReasonCode = decision.ReasonCode
	}
	return result
}

func guardianCommandCompactSpaceV0(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func guardianCommandStartsWithTokenV0(value string, prefix string) bool {
	value = guardianCommandCompactSpaceV0(value)
	return value == prefix || strings.HasPrefix(value, prefix+" ")
}
