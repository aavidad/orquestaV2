package orquestaruntimegemini

import "strings"

func BuildGeminiWrapperScriptV0(profile GeminiConnectorProfileV0) string {
	args := []string{geminiShellQuoteV0(profile.CommandPath)}
	args = append(args, geminiOptionalFlagArgsV0("--model", profile.Model)...)
	args = append(args, geminiOptionalFlagArgsV0("--approval-mode", profile.ApprovalMode)...)
	args = append(args, geminiOptionalFlagArgsV0("--output-format", profile.OutputFormat)...)
	args = append(args, geminiOptionalFlagArgsV0("--include-directories", profile.RuntimeWorkDir)...)
	args = append(args, "--prompt", geminiShellQuoteV0(""))
	for _, arg := range profile.ExtraArgs {
		args = append(args, geminiShellQuoteV0(arg))
	}
	command := strings.Join(args, " ")

	var b strings.Builder
	b.WriteString("#!/bin/sh\n")
	b.WriteString("set -eu\n")
	if profile.HomeDir != "" {
		b.WriteString("export HOME=")
		b.WriteString(geminiShellQuoteV0(profile.HomeDir))
		b.WriteString("\n")
	}
	if profile.PathEnv != "" {
		b.WriteString("export PATH=")
		b.WriteString(geminiShellQuoteV0(profile.PathEnv))
		b.WriteString("\n")
	}
	b.WriteString("cd ")
	b.WriteString(geminiShellQuoteV0(profile.ProjectWorkDir))
	b.WriteString("\n")
	b.WriteString("set +e\n")
	b.WriteString(command)
	b.WriteString(" < ")
	b.WriteString(geminiShellQuoteV0(profile.RuntimeWorkDir + "/" + GeminiAgentPromptFileNameV0))
	b.WriteString(" > ")
	b.WriteString(geminiShellQuoteV0(profile.RuntimeWorkDir + "/" + GeminiStdoutFileNameV0))
	b.WriteString(" 2> ")
	b.WriteString(geminiShellQuoteV0(profile.RuntimeWorkDir + "/" + GeminiStderrFileNameV0))
	b.WriteString(" &\n")
	b.WriteString("orquesta_gemini_child_v0=$!\n")
	b.WriteString("trap 'kill \"$orquesta_gemini_child_v0\" 2>/dev/null; ")
	b.WriteString("wait \"$orquesta_gemini_child_v0\" 2>/dev/null; exit 143' INT TERM\n")
	b.WriteString("wait \"$orquesta_gemini_child_v0\"\n")
	b.WriteString("orquesta_gemini_status_v0=$?\n")
	b.WriteString("trap - INT TERM\n")
	b.WriteString("set -e\n")
	b.WriteString("if [ \"$orquesta_gemini_status_v0\" -ne 0 ] && grep -q 'IneligibleTierError' ")
	b.WriteString(geminiShellQuoteV0(profile.RuntimeWorkDir + "/" + GeminiStderrFileNameV0))
	b.WriteString(" 2>/dev/null; then\n")
	b.WriteString("cat > ")
	b.WriteString(geminiShellQuoteV0(profile.RuntimeWorkDir + "/" + GeminiProviderDiagnosticFileNameV0))
	b.WriteString(" <<'JSON'\n")
	b.WriteString("{\"schema_version\":\"orquesta_provider_diagnostic.v0\",\"provider\":\"gemini\",\"status\":\"blocked\",\"code\":\"provider_auth_or_tier_blocked\",\"message\":\"gemini_cli_ineligible_tier\",\"next_actions\":[\"update_gemini_cli\",\"migrate_to_supported_provider\",\"configure_valid_credentials_or_tier\"],\"evidence_refs\":[\"evidence-ref-gemini-ineligible-tier\",\"evidence-ref-provider-auth-or-tier-blocked\"]}\n")
	b.WriteString("JSON\n")
	b.WriteString("fi\n")
	b.WriteString("exit \"$orquesta_gemini_status_v0\"\n")
	return b.String()
}

func BuildGeminiGoalWrapperScriptV0(
	profile GeminiConnectorProfileV0,
	promptPath string,
	stdoutPath string,
	stderrPath string,
) string {
	args := []string{geminiShellQuoteV0(profile.CommandPath)}
	args = append(args, geminiOptionalFlagArgsV0("--model", profile.Model)...)
	args = append(args, geminiOptionalFlagArgsV0("--approval-mode", profile.ApprovalMode)...)
	args = append(args, geminiOptionalFlagArgsV0("--output-format", profile.OutputFormat)...)
	args = append(args, geminiOptionalFlagArgsV0("--include-directories", profile.RuntimeWorkDir)...)
	args = append(args, "--prompt", geminiShellQuoteV0(""))
	for _, arg := range profile.ExtraArgs {
		args = append(args, geminiShellQuoteV0(arg))
	}
	command := strings.Join(args, " ")

	var b strings.Builder
	b.WriteString("#!/bin/sh\n")
	b.WriteString("set -eu\n")
	if profile.HomeDir != "" {
		b.WriteString("export HOME=")
		b.WriteString(geminiShellQuoteV0(profile.HomeDir))
		b.WriteString("\n")
	}
	if profile.PathEnv != "" {
		b.WriteString("export PATH=")
		b.WriteString(geminiShellQuoteV0(profile.PathEnv))
		b.WriteString("\n")
	}
	b.WriteString("cd ")
	b.WriteString(geminiShellQuoteV0(profile.ProjectWorkDir))
	b.WriteString("\n")
	b.WriteString("set +e\n")
	b.WriteString(command)
	b.WriteString(" < ")
	b.WriteString(geminiShellQuoteV0(promptPath))
	b.WriteString(" > ")
	b.WriteString(geminiShellQuoteV0(stdoutPath))
	b.WriteString(" 2> ")
	b.WriteString(geminiShellQuoteV0(stderrPath))
	b.WriteString(" &\n")
	b.WriteString("orquesta_gemini_goal_child_v0=$!\n")
	b.WriteString("trap 'kill \"$orquesta_gemini_goal_child_v0\" 2>/dev/null; ")
	b.WriteString("wait \"$orquesta_gemini_goal_child_v0\" 2>/dev/null; exit 143' INT TERM\n")
	b.WriteString("wait \"$orquesta_gemini_goal_child_v0\"\n")
	b.WriteString("orquesta_gemini_goal_status_v0=$?\n")
	b.WriteString("trap - INT TERM\n")
	b.WriteString("set -e\n")
	b.WriteString("if [ \"$orquesta_gemini_goal_status_v0\" -ne 0 ] && grep -q 'IneligibleTierError' ")
	b.WriteString(geminiShellQuoteV0(stderrPath))
	b.WriteString(" 2>/dev/null; then\n")
	b.WriteString("cat > ")
	b.WriteString(geminiShellQuoteV0(profile.RuntimeWorkDir + "/" + GeminiProviderDiagnosticFileNameV0))
	b.WriteString(" <<'JSON'\n")
	b.WriteString("{\"schema_version\":\"orquesta_provider_diagnostic.v0\",\"provider\":\"gemini\",\"status\":\"blocked\",\"code\":\"provider_auth_or_tier_blocked\",\"message\":\"gemini_cli_ineligible_tier\",\"next_actions\":[\"update_gemini_cli\",\"migrate_to_supported_provider\",\"configure_valid_credentials_or_tier\"],\"evidence_refs\":[\"evidence-ref-gemini-ineligible-tier\",\"evidence-ref-provider-auth-or-tier-blocked\"]}\n")
	b.WriteString("JSON\n")
	b.WriteString("fi\n")
	b.WriteString("exit \"$orquesta_gemini_goal_status_v0\"\n")
	return b.String()
}

func geminiOptionalFlagArgsV0(flag, value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return []string{flag, geminiShellQuoteV0(value)}
}

func geminiShellQuoteV0(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
