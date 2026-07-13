package orquesta_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// CABLE TRAMPA para el multiusuario.
//
// Hoy, con un solo operador, `scope_mode` vacio cae a `legacy` y devuelve TODO sin
// filtrar. Es compatibilidad hacia atras y esta justificado: no hay identidad
// autenticada ni ownership durable, asi que no hay nada que aislar.
//
// Pero el operador ha pedido multiusuario para la v1.0. En ese mundo, "no pedir
// scope = verlo todo" es escalada de privilegios POR OMISION, y un `legacy`
// invocable por su nombre es un bypass con nombre amable.
//
// El compromiso esta escrito en CODEX_LEEME. Un compromiso sin guard se olvida.
// Este test es el guard: EN CUANTO aparezca autenticacion real de propietario
// (owner_ref/tenant en el codigo de produccion), el modo legacy TIENE que haber
// desaparecido. No se puede tener las dos cosas a la vez ni un solo commit.
func TestElModoLegacyMuereCuandoLlegueElMultiusuarioV0(t *testing.T) {
	hayOwnership := false
	hayLegacy := false

	err := filepath.Walk(".", func(ruta string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			if info != nil && info.IsDir() && (info.Name() == "vendor" || info.Name() == ".git") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(ruta, ".go") || strings.HasSuffix(ruta, "_test.go") {
			return nil
		}
		contenido, err := os.ReadFile(ruta)
		if err != nil {
			return err
		}
		texto := string(contenido)
		// OJO: `OwnerRef` NO sirve como señal. Ya existe y significa otra cosa: el
		// dueño de un CLAIM (el lease de un goal), no un usuario humano. Usar ese
		// mismo nombre para el propietario del multiusuario seria darle dos
		// significados a la misma palabra en un contexto de seguridad, que es como
		// se fabrica un fallo de aislamiento sin querer.
		//
		// La señal es `TenantRef`: identidad del INQUILINO, inequivoca.
		if strings.Contains(texto, "TenantRef ") {
			hayOwnership = true
		}
		if strings.Contains(texto, "MCPAutoprogrammingStatusScopeLegacyV0") {
			hayLegacy = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("recorriendo el arbol: %v", err)
	}

	if hayOwnership && hayLegacy {
		t.Fatal(
			"ha aterrizado ownership (OwnerRef/TenantRef) y el modo legacy SIGUE VIVO: " +
				"con multiusuario, 'no pedir scope = verlo todo' es escalada de privilegios por omision, " +
				"y un legacy invocable por su nombre es un bypass. Borra legacy antes de activar el multiusuario.",
		)
	}
}
