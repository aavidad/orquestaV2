package orquestaruntimecodex

import (
	"path/filepath"
	"strconv"
	"strings"
)

func BuildCodexWrapperScriptV0(profile CodexConnectorProfileV0) string {
	args := []string{shellQuoteV0(profile.CommandPath)}
	args = append(args, codexOptionalFlagArgsV0("--ask-for-approval", profile.ApprovalPolicy)...)
	args = append(args, "exec")
	args = append(args, codexSkipGitRepoCheckArgsV0(profile)...)
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
	b.WriteString(codexSharedStartupLockShellV0())
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
	b.WriteString(codexSharedStartupLockReleaseAfterLaunchShellV0(codexStartupLockDefaultSecondsV0(profile)))
	b.WriteString("trap 'kill \"$orquesta_codex_child_v0\" 2>/dev/null; orquesta_codex_release_startup_lock_v0; ")
	b.WriteString("wait \"$orquesta_codex_child_v0\" 2>/dev/null; exit 143' INT TERM\n")
	b.WriteString("wait \"$orquesta_codex_child_v0\"\n")
	b.WriteString("orquesta_codex_status_v0=$?\n")
	b.WriteString("trap - INT TERM\n")
	b.WriteString("set -e\n")
	b.WriteString("orquesta_codex_write_usage_accounting_v0 || true\n")
	b.WriteString("exit \"$orquesta_codex_status_v0\"\n")
	return b.String()
}

func codexSharedStartupLockShellV0() string {
	return strings.Join([]string{
		"orquesta_codex_startup_lock_dir_v0=\"\"",
		"orquesta_codex_release_startup_lock_v0() {",
		"  if [ -n \"$orquesta_codex_startup_lock_dir_v0\" ]; then",
		"    rmdir \"$orquesta_codex_startup_lock_dir_v0\" 2>/dev/null || true",
		"    orquesta_codex_startup_lock_dir_v0=\"\"",
		"  fi",
		"}",
		"orquesta_codex_reap_stale_startup_lock_v0() {",
		"  orquesta_codex_lock_stale_seconds_v0=\"${ORQUESTA_CODEX_STARTUP_LOCK_STALE_SECONDS:-900}\"",
		"  case \"$orquesta_codex_lock_stale_seconds_v0\" in ''|*[!0-9]*) orquesta_codex_lock_stale_seconds_v0=900;; esac",
		"  if [ \"$orquesta_codex_lock_stale_seconds_v0\" -le 0 ]; then return 1; fi",
		"  orquesta_codex_lock_now_v0=\"$(date +%s 2>/dev/null || printf '%s' 0)\"",
		"  orquesta_codex_lock_mtime_v0=\"$(stat -c %Y \"$orquesta_codex_startup_lock_dir_v0\" 2>/dev/null || printf '%s' 0)\"",
		"  case \"$orquesta_codex_lock_now_v0:$orquesta_codex_lock_mtime_v0\" in *[!0-9:]*|0:*|*:0) return 1;; esac",
		"  orquesta_codex_lock_age_v0=$((orquesta_codex_lock_now_v0 - orquesta_codex_lock_mtime_v0))",
		"  if [ \"$orquesta_codex_lock_age_v0\" -ge \"$orquesta_codex_lock_stale_seconds_v0\" ]; then",
		"    if rmdir \"$orquesta_codex_startup_lock_dir_v0\" 2>/dev/null; then",
		"      printf '%s\\n' 'orquesta_codex_startup_lock_stale_reaped' >&2",
		"      return 0",
		"    fi",
		"  fi",
		"  return 1",
		"}",
		"orquesta_codex_lock_root_v0=\"${CODEX_HOME:-${HOME:-}}\"",
		"if [ -n \"$orquesta_codex_lock_root_v0\" ]; then",
		"  orquesta_codex_startup_lock_dir_v0=\"$orquesta_codex_lock_root_v0/.orquesta-codex-startup.lock\"",
		"  orquesta_codex_lock_timeout_v0=\"${ORQUESTA_CODEX_STARTUP_LOCK_TIMEOUT_SECONDS:-120}\"",
		"  case \"$orquesta_codex_lock_timeout_v0\" in ''|*[!0-9]*) orquesta_codex_lock_timeout_v0=120;; esac",
		"  orquesta_codex_lock_waited_v0=0",
		"  while ! mkdir \"$orquesta_codex_startup_lock_dir_v0\" 2>/dev/null; do",
		"    if orquesta_codex_reap_stale_startup_lock_v0; then",
		"      continue",
		"    fi",
		"    if [ \"$orquesta_codex_lock_waited_v0\" -ge \"$orquesta_codex_lock_timeout_v0\" ]; then",
		"      printf '%s\\n' 'orquesta_codex_startup_lock_timeout' >&2",
		"      exit 75",
		"    fi",
		"    sleep 1",
		"    orquesta_codex_lock_waited_v0=$((orquesta_codex_lock_waited_v0 + 1))",
		"  done",
		"fi",
		"",
	}, "\n")
}

func codexSharedStartupLockReleaseAfterLaunchShellV0(defaultSeconds string) string {
	if strings.TrimSpace(defaultSeconds) == "" {
		defaultSeconds = "8"
	}
	return strings.Join([]string{
		"orquesta_codex_startup_lock_seconds_v0=\"${ORQUESTA_CODEX_STARTUP_LOCK_SECONDS:-" + defaultSeconds + "}\"",
		"case \"$orquesta_codex_startup_lock_seconds_v0\" in ''|*[!0-9]*) orquesta_codex_startup_lock_seconds_v0=" + defaultSeconds + ";; esac",
		"orquesta_codex_startup_waited_v0=0",
		"while [ \"$orquesta_codex_startup_waited_v0\" -lt \"$orquesta_codex_startup_lock_seconds_v0\" ]; do",
		"  if ! kill -0 \"$orquesta_codex_child_v0\" 2>/dev/null; then",
		"    break",
		"  fi",
		"  sleep 1",
		"  orquesta_codex_startup_waited_v0=$((orquesta_codex_startup_waited_v0 + 1))",
		"done",
		"orquesta_codex_release_startup_lock_v0",
		"",
	}, "\n")
}

func codexStartupLockDefaultSecondsV0(profile CodexConnectorProfileV0) string {
	if filepath.Base(strings.TrimSpace(profile.CommandPath)) != "codex" {
		return "0"
	}
	codeHome := strings.TrimSpace(profile.CodeHomeDir)
	runtimeDir := strings.TrimSpace(profile.RuntimeWorkDir)
	if codeHome != "" && runtimeDir != "" && codexPathInsideV0(codeHome, runtimeDir) {
		return "0"
	}
	return "8"
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

func codexSkipGitRepoCheckArgsV0(profile CodexConnectorProfileV0) []string {
	for _, arg := range profile.ExtraArgs {
		if strings.TrimSpace(arg) == "--skip-git-repo-check" {
			return nil
		}
	}
	return []string{"--skip-git-repo-check"}
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
