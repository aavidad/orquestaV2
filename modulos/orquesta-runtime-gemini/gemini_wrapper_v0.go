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
	b.WriteString("exit \"$orquesta_gemini_status_v0\"\n")
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
