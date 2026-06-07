package orquestaruntimeclaude

import "strings"

func BuildClaudeWrapperScriptV0(profile ClaudeConnectorProfileV0) string {
	args := []string{claudeShellQuoteV0(profile.CommandPath), "-p"}
	args = append(args, claudeOptionalFlagArgsV0("--model", profile.Model)...)
	args = append(args, claudeOptionalFlagArgsV0("--permission-mode", profile.PermissionMode)...)
	args = append(args, claudeOptionalFlagArgsV0("--output-format", profile.OutputFormat)...)
	args = append(args, claudeOptionalFlagArgsV0("--effort", profile.Effort)...)
	for _, arg := range profile.ExtraArgs {
		args = append(args, claudeShellQuoteV0(arg))
	}
	command := strings.Join(args, " ")

	var b strings.Builder
	b.WriteString("#!/bin/sh\n")
	b.WriteString("set -eu\n")
	if profile.HomeDir != "" {
		b.WriteString("export HOME=")
		b.WriteString(claudeShellQuoteV0(profile.HomeDir))
		b.WriteString("\n")
	}
	if profile.PathEnv != "" {
		b.WriteString("export PATH=")
		b.WriteString(claudeShellQuoteV0(profile.PathEnv))
		b.WriteString("\n")
	}
	b.WriteString("cd ")
	b.WriteString(claudeShellQuoteV0(profile.ProjectWorkDir))
	b.WriteString("\n")
	b.WriteString("set +e\n")
	b.WriteString(command)
	b.WriteString(" < ")
	b.WriteString(claudeShellQuoteV0(profile.RuntimeWorkDir + "/" + ClaudeAgentPromptFileNameV0))
	b.WriteString(" > ")
	b.WriteString(claudeShellQuoteV0(profile.RuntimeWorkDir + "/" + ClaudeStdoutFileNameV0))
	b.WriteString(" 2> ")
	b.WriteString(claudeShellQuoteV0(profile.RuntimeWorkDir + "/" + ClaudeStderrFileNameV0))
	b.WriteString(" &\n")
	b.WriteString("orquesta_claude_child_v0=$!\n")
	b.WriteString("trap 'kill \"$orquesta_claude_child_v0\" 2>/dev/null; ")
	b.WriteString("wait \"$orquesta_claude_child_v0\" 2>/dev/null; exit 143' INT TERM\n")
	b.WriteString("wait \"$orquesta_claude_child_v0\"\n")
	b.WriteString("orquesta_claude_status_v0=$?\n")
	b.WriteString("trap - INT TERM\n")
	b.WriteString("set -e\n")
	b.WriteString("exit \"$orquesta_claude_status_v0\"\n")
	return b.String()
}

func claudeOptionalFlagArgsV0(flag, value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return []string{flag, claudeShellQuoteV0(value)}
}

func claudeShellQuoteV0(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
