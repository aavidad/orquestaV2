package orquesta_test

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// MEJ-106 / TAREA-8.3 — gobierno de variables de entorno.
//
// POLITICA DEL OPERADOR (2026-07-13), que sustituye al tope de conteo:
//
//   - Duplicidad semantica -> se PARA y se unifica, aunque quepa en el presupuesto.
//   - Env nueva, no duplicada y justificada por programacion nueva -> VERDE.
//     No se corta por un numero.
//   - Superficie de seguridad -> autorizacion explicita del operador.
//
// Por que desaparece el tope de conteo (426/103): un contador no distingue una
// variable imprescindible de una chapuza, y ademas NO estaba contando variables.
// Contaba apariciones del patron ORQUESTA_[A-Z0-9_]+ en el fuente, incluidos 16
// PREFIJOS que no son envs (ORQUESTA_CODEX_, ORQUESTA_OPES_, ORQUESTA_SERVER_...)
// usados para casar nombres. Bloqueaba trabajo legitimo con una cifra que no
// medía lo que decía medir.
//
// Lo que SI gobierna de verdad, y sigue vigente:
//
//   - Declaracion obligatoria: toda env leida en produccion por el servidor debe
//     estar en serverEffectiveEnvRegistryV0. Lo exige por AST, con baseline 0,
//     TestServerEnvRegistryASTV0LecturasORQUESTARegistradas.
//   - Este fichero: duplicados y superficie de seguridad.
const envSecurityAuthorizedNoteV0 = `superficie de seguridad: alta autorizada por el operador, una por una`

// Envs de superficie de seguridad autorizadas. Anadir una entrada aqui es una
// DECISION DEL OPERADOR, no del agente que programa: son las variables que
// pueden relajar el sandbox, abrir egress o portar credenciales.
var envSecuritySurfaceAuthorizedV0 = map[string]string{
	// Sandbox y limites de ejecucion.
	"ORQUESTA_CODEX_CONTAINER_SANDBOX_BOUNDARY_CONFIRMED":                                  "opt-in danger-full-access cuando el limite de seguridad es el propio contenedor (autorizado 2026-07-12)",
	"ORQUESTA_CODEX_SANDBOX":                                                               "politica sandbox del runtime Codex",
	"ORQUESTA_CODEX_DIRECTOR_SANDBOX":                                                      "politica sandbox del director",
	"ORQUESTA_CODEX_WAVE_SANDBOX":                                                          "politica sandbox de las olas de autoprogramacion",
	"ORQUESTA_GUARDIAN_REPAIR_CODEX_SANDBOX":                                               "politica sandbox entregada al guardian (entorno del proceso hijo)",
	"ORQUESTA_GUARDIAN_REPAIR_CODEX_ALLOW_BROAD_SANDBOX":                                   "opt-in de sandbox amplio en reparacion guardian",
	"ORQUESTA_GUARDIAN_REPAIR_CODEX_SANDBOX_EVIDENCE_REF":                                  "evidencia durable del sandbox usado en reparacion",
	"ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_GUARDIAN_REPAIR_CODEX_SANDBOX":              "ajuste del servidor: sandbox de reparacion guardian",
	"ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_GUARDIAN_REPAIR_CODEX_ALLOW_BROAD_SANDBOX":  "ajuste del servidor: opt-in de sandbox amplio",
	"ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_GUARDIAN_REPAIR_CODEX_SANDBOX_EVIDENCE_REF": "ajuste del servidor: evidencia del sandbox de reparacion",

	// Credenciales y control. Portan secreto o abren el plano de control.
	"ORQUESTA_SERVER_CONTROL_TOKEN":              "token del plano de control",
	"ORQUESTA_PRIVATE_API_KEY":                   "credencial de proveedor",
	"ORQUESTA_EGRESS_PRIVATE_API_KEY":            "credencial de proveedor para egress",
	"ORQUESTA_HERMES_API_KEY":                    "credencial de Hermes",
	"ORQUESTA_OLLAMA_MODEL_MANAGER_BEARER_TOKEN": "credencial del gestor de modelos locales",

	// Prontitud de auth observada (no porta secreto, pero decide si se sale a red).
	"ORQUESTA_OPES_BRIDGE_REMOTE_QA_AUTH_STATE_READY": "estado de auth observado por el puente OPES",
}

// Un nombre que sea otro nombre + uno de estos sufijos es, salvo prueba en
// contrario, la MISMA variable escrita dos veces: el patron clasico de anadir
// una variante en vez de dar un valor mas a la que ya existe.
//
// Caso real que motiva la regla (2026-07-13): ORQUESTA_TEST_TMUX_INVALID_IDENTITY
// y ORQUESTA_TEST_TMUX_INVALID_IDENTITY_ONCE convivieron para el mismo concepto
// ("el tmux falso devuelve identidad invalida"), una para "siempre" y otra para
// "solo la primera vez". Es UNA variable con dos valores (1 / once).
var envDuplicateModifierSuffixesV0 = []string{
	"ONCE", "2", "NEW", "OLD", "V2", "ALT", "TMP", "COPY", "BIS", "EXTRA", "FIX",
}

var envSecuritySurfacePatternV0 = regexp.MustCompile(
	`SANDBOX|DANGER|ALLOW_BROAD|SECRET|API_KEY|BEARER_TOKEN|CONTROL_TOKEN|AUTH`,
)

var envNamePatternV0 = regexp.MustCompile(`ORQUESTA_[A-Z0-9_]+`)

func TestEnvVarsBudgetMEJ106V0(t *testing.T) {
	root := findRepoRootForEnvVarsBudgetMEJ106V0(t)
	envs := collectEnvNamesV0(t, root)
	if len(envs) == 0 {
		t.Fatalf("no se encontro ninguna env ORQUESTA_*: el escaneo esta roto")
	}

	t.Run("sin_duplicados_semanticos", func(t *testing.T) {
		var dups []string
		for _, name := range envs {
			for _, suffix := range envDuplicateModifierSuffixesV0 {
				variant := name + "_" + suffix
				if envContainsV0(envs, variant) {
					dups = append(dups, variant+" duplica a "+name)
				}
			}
		}
		if len(dups) > 0 {
			sort.Strings(dups)
			t.Fatalf(
				"variables duplicadas (mismo concepto, dos nombres): %s\n"+
					"No anadas una variante: da un valor mas a la variable que ya existe.",
				strings.Join(dups, "; "),
			)
		}
	})

	t.Run("superficie_seguridad_autorizada", func(t *testing.T) {
		var noAutorizadas []string
		for _, name := range envs {
			if !envSecuritySurfacePatternV0.MatchString(name) {
				continue
			}
			if _, ok := envSecuritySurfaceAuthorizedV0[name]; !ok {
				noAutorizadas = append(noAutorizadas, name)
			}
		}
		if len(noAutorizadas) > 0 {
			sort.Strings(noAutorizadas)
			t.Fatalf(
				"envs de superficie de seguridad sin autorizacion del operador: %s\n"+
					"%s. Anadirlas al allowlist es decision del operador, no del agente: "+
					"anunciala en commit propio, no la cueles en otro cambio.",
				strings.Join(noAutorizadas, ", "),
				envSecurityAuthorizedNoteV0,
			)
		}
	})

	t.Run("allowlist_seguridad_sin_entradas_muertas", func(t *testing.T) {
		var muertas []string
		for name := range envSecuritySurfaceAuthorizedV0 {
			if !envContainsV0(envs, name) {
				muertas = append(muertas, name)
			}
		}
		if len(muertas) > 0 {
			sort.Strings(muertas)
			t.Fatalf(
				"envs autorizadas de seguridad que ya no existen en el codigo: %s\n"+
					"Bajalas del allowlist: una autorizacion viva sobre algo que no existe es una puerta abierta a nada.",
				strings.Join(muertas, ", "),
			)
		}
	})

	// Diagnostico, no puerta. El numero se publica para ver la tendencia; ya no
	// bloquea, porque una env nueva y justificada debe poder entrar (orden del
	// operador 2026-07-13).
	t.Logf("inventario ORQUESTA_*: %d nombres reales (prefijos excluidos)", len(envs))
}

// collectEnvNamesV0 devuelve los nombres de env reales del arbol: excluye los
// PREFIJOS (tokens terminados en _), que no son variables sino patrones de
// coincidencia y eran la mitad de la inflacion del contador viejo.
func collectEnvNamesV0(t *testing.T, root string) []string {
	t.Helper()
	seen := map[string]bool{}
	for _, dir := range []string{"cmd", "modulos"} {
		walkErr := filepath.Walk(filepath.Join(root, dir), func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			content, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			for _, name := range envNamePatternV0.FindAllString(string(content), -1) {
				if strings.HasSuffix(name, "_") {
					continue
				}
				seen[name] = true
			}
			return nil
		})
		if walkErr != nil {
			t.Fatalf("recorrer %s: %v", dir, walkErr)
		}
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func envContainsV0(envs []string, name string) bool {
	index := sort.SearchStrings(envs, name)
	return index < len(envs) && envs[index] == name
}

func findRepoRootForEnvVarsBudgetMEJ106V0(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if hasDirEnvVarsBudgetMEJ106V0(dir, "cmd") &&
			hasDirEnvVarsBudgetMEJ106V0(dir, "modulos") &&
			hasDirEnvVarsBudgetMEJ106V0(dir, "scripts") {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("repo root no encontrado desde %s", dir)
		}
		dir = parent
	}
}

func hasDirEnvVarsBudgetMEJ106V0(root string, name string) bool {
	info, err := os.Stat(filepath.Join(root, name))
	return err == nil && info.IsDir()
}
