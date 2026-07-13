package orquestaruntimerequiredtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// La resolucion de la lista blanca de comandos esta HOY duplicada en cuatro
// sitios: preflight, validacion congelada, race/CGO y el ejecutor. Codex va a
// consolidarla en un unico helper interno.
//
// Este guard es un TOPE, no una bendicion: impide que aparezca una quinta copia
// mientras tanto. Consolidar arregla el hoy, pero no impide que mañana alguien
// resuelva la allowlist a mano "porque es una linea" y volvamos a estar donde
// estabamos, esta vez sin que nadie lo note porque el problema parecia resuelto.
//
// Cuando la consolidacion termine, este numero baja a 1 y el guard pasa a
// significar lo que debe: la allowlist se resuelve en UN sitio y solo en uno.
const copiasDeLaAllowlistPermitidasV0 = 4

func TestLaAllowlistNoSeResuelveEnMasSitiosV0(t *testing.T) {
	ficheros, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	copias := map[string]int{}
	total := 0
	for _, fichero := range ficheros {
		if strings.HasSuffix(fichero, "_test.go") {
			continue
		}
		contenido, err := os.ReadFile(fichero)
		if err != nil {
			t.Fatalf("leyendo %s: %v", fichero, err)
		}
		n := strings.Count(string(contenido), "AllowedCommands[tokens[0]]")
		if n > 0 {
			copias[fichero] = n
			total += n
		}
	}
	if total > copiasDeLaAllowlistPermitidasV0 {
		t.Fatalf(
			"la allowlist se resuelve en %d sitios (tope %d): una copia nueva es un sitio mas donde el control puede divergir. Copias: %v",
			total, copiasDeLaAllowlistPermitidasV0, copias,
		)
	}
	if total == 0 {
		t.Fatal("no se encontro ninguna resolucion de la allowlist: ¿se ha desactivado el control?")
	}
}
