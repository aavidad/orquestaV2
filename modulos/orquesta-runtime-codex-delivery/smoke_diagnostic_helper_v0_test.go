package orquestaruntimecodexdelivery

import (
	"os"
	"path/filepath"
	"strings"

	orquestarails "orquesta/modulos/orquesta-rails"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

const codexRealSmokeDiagnosticMaxBytesV0 = 1600

func codexRealSmokeDiagnosticsV0(runtimeDir string) string {
	parts := []string{}
	for _, name := range []string{
		orquestaruntimecodex.CodexAgentAckFileNameV0,
		orquestaruntimecodex.CodexLastMessageFileNameV0,
		orquestaruntimecodex.CodexStdoutFileNameV0,
		orquestaruntimecodex.CodexStderrFileNameV0,
	} {
		data, err := os.ReadFile(filepath.Join(runtimeDir, name))
		if err != nil {
			continue
		}
		parts = append(parts, name+": "+codexRealSmokeTruncateV0(string(data)))
	}
	return strings.Join(parts, "\n")
}

func codexRealSmokeTruncateV0(value string) string {
	return realSmokeDiagnosticTextV0("codex_real_smoke", value)
}

func realSmokeDiagnosticTextV0(boundary string, value string) string {
	redacted, _ := orquestarails.RedactOperationalTextForFieldV0(boundary, "diagnostic", value)
	redacted = strings.TrimSpace(redacted)
	if len(redacted) <= codexRealSmokeDiagnosticMaxBytesV0 {
		return redacted
	}
	return redacted[:codexRealSmokeDiagnosticMaxBytesV0] + "\n[truncated]"
}
