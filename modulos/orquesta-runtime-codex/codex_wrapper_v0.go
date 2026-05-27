package orquestaruntimecodex

import (
	"strconv"
	"strings"
)

func BuildCodexWrapperScriptV0(profile CodexConnectorProfileV0) string {
	args := []string{shellQuoteV0(profile.CommandPath)}
	args = append(args, codexOptionalFlagArgsV0("--ask-for-approval", profile.ApprovalPolicy)...)
	args = append(args, "exec")
	args = append(args, codexOptionalFlagArgsV0("-m", profile.Model)...)
	args = append(args, codexConfigValueArgsV0("model_reasoning_effort", profile.ReasoningEffort)...)
	args = append(args, codexOptionalFlagArgsV0("-p", profile.Profile)...)
	args = append(args, codexOptionalFlagArgsV0("--sandbox", profile.Sandbox)...)
	args = append(args, codexRuntimeWritableArgsV0(profile)...)
	args = append(args, codexOutputLastMessageArgsV0(profile)...)
	for _, arg := range profile.ExtraArgs {
		args = append(args, shellQuoteV0(arg))
	}
	args = append(args, "-C", shellQuoteV0(profile.ProjectWorkDir), "-")
	command := strings.Join(args, " ")

	var b strings.Builder
	b.WriteString("#!/bin/sh\n")
	b.WriteString("set -eu\n")
	b.WriteString(codexUsageAccountingShellFunctionV0(profile))
	if profile.HomeDir != "" {
		b.WriteString("export HOME=")
		b.WriteString(shellQuoteV0(profile.HomeDir))
		b.WriteString("\n")
	}
	if profile.CodeHomeDir != "" {
		b.WriteString("export CODEX_HOME=")
		b.WriteString(shellQuoteV0(profile.CodeHomeDir))
		b.WriteString("\n")
	}
	if profile.PathEnv != "" {
		b.WriteString("export PATH=")
		b.WriteString(shellQuoteV0(profile.PathEnv))
		b.WriteString("\n")
	}
	b.WriteString("cd ")
	b.WriteString(shellQuoteV0(profile.ProjectWorkDir))
	b.WriteString("\n")
	b.WriteString("set +e\n")
	b.WriteString(command)
	b.WriteString(" < ")
	b.WriteString(shellQuoteV0(profile.RuntimeWorkDir + "/" + CodexAgentPromptFileNameV0))
	b.WriteString(" > ")
	b.WriteString(shellQuoteV0(profile.RuntimeWorkDir + "/" + CodexStdoutFileNameV0))
	b.WriteString(" 2> ")
	b.WriteString(shellQuoteV0(profile.RuntimeWorkDir + "/" + CodexStderrFileNameV0))
	b.WriteString(" &\n")
	b.WriteString("orquesta_codex_child_v0=$!\n")
	b.WriteString("trap 'kill \"$orquesta_codex_child_v0\" 2>/dev/null; ")
	b.WriteString("wait \"$orquesta_codex_child_v0\" 2>/dev/null; exit 143' INT TERM\n")
	b.WriteString("wait \"$orquesta_codex_child_v0\"\n")
	b.WriteString("orquesta_codex_status_v0=$?\n")
	b.WriteString("trap - INT TERM\n")
	b.WriteString("set -e\n")
	b.WriteString("orquesta_codex_write_usage_accounting_v0 || true\n")
	b.WriteString("exit \"$orquesta_codex_status_v0\"\n")
	return b.String()
}

func codexOptionalFlagArgsV0(flag, value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return []string{flag, shellQuoteV0(value)}
}

func codexConfigValueArgsV0(key, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return []string{"-c", shellQuoteV0(key + "=" + strconv.Quote(value))}
}

func codexOutputLastMessageArgsV0(profile CodexConnectorProfileV0) []string {
	return []string{
		"--output-last-message",
		shellQuoteV0(profile.RuntimeWorkDir + "/" + CodexLastMessageFileNameV0),
	}
}

func codexRuntimeWritableArgsV0(profile CodexConnectorProfileV0) []string {
	if strings.TrimSpace(profile.Sandbox) != "workspace-write" ||
		strings.TrimSpace(profile.RuntimeWorkDir) == "" {
		return nil
	}
	return []string{"--add-dir", shellQuoteV0(profile.RuntimeWorkDir)}
}

func shellQuoteV0(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
