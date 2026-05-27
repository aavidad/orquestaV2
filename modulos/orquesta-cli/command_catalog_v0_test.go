package orquestacli

import (
	"strings"
	"testing"
)

func TestCliCommandCatalogV0AlimentaHelpYDispatch(t *testing.T) {
	helpES := cliHelpTextForArgsV0([]string{"--help"})
	helpEN := cliHelpTextForArgsV0([]string{"--help", "--locale", "en"})

	seen := map[string]bool{}
	for _, entry := range cliCommandCatalogV0 {
		path := strings.Join(entry.Path, " ")
		if path == "" {
			t.Fatalf("catalogo con path vacio: %+v", entry)
		}
		if seen[path] {
			t.Fatalf("path duplicado en catalogo: %s", path)
		}
		seen[path] = true
		if entry.Handler == nil || entry.ClientTarget == "" || entry.EffectProfile == "" || entry.State == "" {
			t.Fatalf("metadata incompleta para %s: %+v", path, entry)
		}
		switch entry.State {
		case CliCommandStateActiveV0, CliCommandStateLegacyV0, CliCommandStateHiddenV0:
		default:
			t.Fatalf("estado invalido para %s: %s", path, entry.State)
		}
		if entry.State == CliCommandStateHiddenV0 {
			continue
		}
		if found, rest := findCLICommandCatalogEntryV0(append(append([]string{}, entry.Path...), "--json")); found == nil || len(rest) != 1 {
			t.Fatalf("dispatch no sale del catalogo para %s", path)
		}
		if !strings.Contains(helpES, cliCommandUsageV0(entry, "es")) {
			t.Fatalf("help ES no contiene uso de catalogo para %s", path)
		}
		if !strings.Contains(helpEN, cliCommandUsageV0(entry, "en")) {
			t.Fatalf("help EN no contiene uso de catalogo para %s", path)
		}
	}
}

func TestCliUnsupportedCommandDetailV0RedactaYSugiereDesdeCatalogo(t *testing.T) {
	detail := cliUnsupportedCommandDetailV0([]string{
		"https://usuario:secreto@example.test/path",
		"--input",
		`{"token":"abc"}`,
	})
	if strings.Contains(detail, "usuario") || strings.Contains(detail, "secreto") ||
		strings.Contains(detail, "token") || !strings.Contains(detail, "path=<redacted>") {
		t.Fatalf("detalle no redactado: %s", detail)
	}

	detail = cliUnsupportedCommandDetailV0([]string{"autoprogramacionx", "estado"})
	if !strings.Contains(detail, "path=autoprogramacionx estado") ||
		!strings.Contains(detail, "autoprogramacion estado ver") {
		t.Fatalf("detalle sin path/sugerencia acotada: %s", detail)
	}
}
