package orquestaappcodexstack

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestarails "orquesta/modulos/orquesta-rails"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

func codexStackRealSmokeWriteProjectContextV0(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	files := map[string]string{
		"AGENTS.md": strings.Join([]string{
			"# Contexto local",
			"- Usa el write-set recibido como alcance cerrado; si falta alcance, usa CONSULTA AL DIRECTOR.",
			"- Arquitectura hexagonal e i18n por defecto.",
			"- Persistencia solo por puerto/conector; no elijas una base concreta.",
			"- Ficheros pequenos; divide responsabilidades si crecen.",
			"- Si falta informacion, escribe CONSULTA AL DIRECTOR.",
		}, "\n"),
		"README.md": "# Agenda API Web\n\nProyecto temporal para smoke real de Orquesta.\n",
	}
	for path, data := range files {
		if err := os.WriteFile(filepath.Join(dir, path), []byte(data), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
}

func codexStackRealSmokeDiagnosticsV0(runtimeDir string) string {
	parts := []string{}
	_ = filepath.WalkDir(runtimeDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		switch entry.Name() {
		case orquestaruntimecodex.CodexAgentAckFileNameV0,
			orquestaruntimecodex.CodexLastMessageFileNameV0,
			orquestaruntimecodex.CodexStdoutFileNameV0,
			orquestaruntimecodex.CodexStderrFileNameV0:
			data, readErr := os.ReadFile(path)
			if readErr == nil {
				parts = append(parts, entry.Name()+": "+codexStackRealSmokeTruncateV0(string(data)))
			}
		}
		return nil
	})
	return strings.Join(parts, "\n")
}

func codexStackRealSmokeTruncateV0(value string) string {
	value, _ = orquestarails.RedactOperationalTextForFieldV0("codex_stack_real_smoke", "diagnostic", value)
	value = strings.TrimSpace(value)
	if len(value) <= 1600 {
		return value
	}
	return value[:1600] + "\n[truncated]"
}
