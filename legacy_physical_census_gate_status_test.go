// Este contrato fija el NO-GO sin consultar infraestructura y liga sus sujetos.
package orquesta_test

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const (
	legacyPhysicalCensusGatePath = "docs/reconstruccion/estado_compuerta_censo_fisico_2026-07-30.md"
	sealedCensusPlanPath         = "docs/reconstruccion/plan_ejecucion_censo_fisico_historico_2026-07-30.md"
	sealedCensusPlanTestPath     = "legacy_physical_census_plan_test.go"
	acceptedPhysicalCensorDigest = "33898168cdf41b6ece7702a47d4bd7a073a0652f5b1e0f7a395c91f6d434901f"
	sealedCensusPlanDigest       = "2a863bb5d2f648ea1844a0653484b060e295a8fd4372f3c48ddb46317ff7dc96"
	sealedCensusPlanTestDigest   = "87d8a010dc82d49ec823100fa91522abda28498681f90ed308fc7260af276de4"
)

func TestLegacyPhysicalCensusGateRemainsClosed(t *testing.T) {
	content, err := os.ReadFile(legacyPhysicalCensusGatePath)
	if err != nil {
		t.Fatalf("leer estado de compuerta: %v", err)
	}
	document := string(content)
	for _, fragment := range []string{
		"Estado de ejecución real: **NO-GO**.",
		"sha256:" + sealedCensusPlanDigest,
		"sha256:" + sealedCensusPlanTestDigest,
		"02c4ad641d83b656a3cd58fbde090faed66ca3e0",
		acceptedPhysicalCensorDigest,
		"41 pruebas",
		"`P0 = 0`, `P1 = 0` y `P2 = 0`",
		"product/traceability/rebuild_bugs.jsonl",
		"d154d3e2b53b1eea786dd043f956af994d23826a",
		"`reviews:two_independent_accepts`",
		"No existe en el árbol un",
		"artefacto durable individual por revisor",
		"`closed: false` permanece",
		"Ninguna raíz real fue abierta, enumerada o censada",
		"Es una observación fechada, no una garantía futura.",
		"se deben reobservar fuente, dispositivo",
		"prohibido_sin_orden: detener|desmontar|montar|reformatear|reparticionar|copiar|borrar|ejecutar",
	} {
		if !strings.Contains(document, fragment) {
			t.Errorf("el estado perdió el contrato %q", fragment)
		}
	}
	for _, forbidden := range []string{
		"Estado de ejecución real: **GO**",
		"`GOV-16` queda acreditada",
		"la ejecución real queda autorizada",
		"el inventario histórico queda cerrado",
	} {
		if strings.Contains(document, forbidden) {
			t.Fatalf("el estado introdujo el falso cierre %q", forbidden)
		}
	}
	wantGates := map[string]string{
		"Expansión lógica": "en construcción, no acreditada", "Mapeo físico": "bloqueado",
		"Vista estable": "bloqueada", "Recibos": "bloqueados",
		"Salida física": "bloqueada", "Ejecución real": "bloqueada",
	}
	assertStatusMap(t, document, "| Compuerta | Estado factual |", wantGates)
	wantPlanConditions := map[string]string{
		"`censador_minimo_acreditado == true`": "satisfecha", "`expansion_97_15_285_382_sellada == true`": "pendiente",
		"`members_sha256_invalidos == 0`": "pendiente de verificar", "`vista_estable_y_cercado_acreditados == true`": "pendiente",
		"`mapeo_1_1_y_1_n_acreditado == true`":                             "pendiente",
		"`reobservacion_382_en_vista_estable_acreditada == true`":          "pendiente",
		"`presencias_actuales_inferidas_desde_v3 == 0`":                    "pendiente de verificar",
		"`limites_y_reserva_acreditados == true`":                          "pendiente",
		"`recibos_e_idempotencia_acreditados == true`":                     "pendiente",
		"`publicacion_y_recuperacion_acreditadas == true`":                 "satisfecha en el censador",
		"`salida_fuera_de_fuentes == true`":                                "pendiente",
		"`salida_en_dispositivo_distinto_o_vista_externa_cercada == true`": "pendiente",
		"`recuperacion_conservadora_sin_borrado_automatico == true`":       "satisfecha en el censador",
		"`quinta_auditoria_independiente_superada == true`":                "satisfecha para el censador",
		"`revisor_independiente_asignado == true`":                         "pendiente para la ejecución real",
	}
	assertStatusMap(t, document, "| Condición exacta del plan | Estado del corte |", wantPlanConditions)
	assertFileDigest(t, sealedCensusPlanPath, sealedCensusPlanDigest)
	assertFileDigest(t, sealedCensusPlanTestPath, sealedCensusPlanTestDigest)
	assertConsolidatedReviews(t)
	if got := physicalCensorTreeDigest(t); got != acceptedPhysicalCensorDigest {
		t.Fatalf("sujeto del censador = %s; se esperaba %s", got, acceptedPhysicalCensorDigest)
	}
}

func assertConsolidatedReviews(t *testing.T) {
	t.Helper()
	content, err := os.ReadFile("product/traceability/rebuild_bugs.jsonl")
	if err != nil {
		t.Fatalf("leer ledger de defectos: %v", err)
	}
	lines := strings.Split(string(content), "\n")
	for _, suffix := range []string{"004", "005", "008", "009", "010", "011"} {
		id, found := "BUG-REBUILD-20260730-"+suffix, false
		for _, line := range lines {
			if !strings.Contains(line, `"bug_id":"`+id+`"`) {
				continue
			}
			if !strings.Contains(line, `"status":"closed"`) ||
				!strings.Contains(line, acceptedPhysicalCensorDigest) ||
				!strings.Contains(line, "reviews:two_independent_accepts") {
				t.Fatalf("dictamen consolidado incoherente para %s", id)
			}
			found = true
		}
		if !found {
			t.Fatalf("falta el dictamen consolidado de %s", id)
		}
	}
}

func assertStatusMap(t *testing.T, document, header string, want map[string]string) {
	t.Helper()
	start := strings.Index(document, header)
	if start < 0 {
		t.Fatalf("falta la tabla %q", header)
	}
	got := map[string]string{}
	for _, line := range strings.Split(document[start+len(header):], "\n")[2:] {
		if !strings.HasPrefix(line, "|") {
			break
		}
		fields := strings.Split(line, "|")
		if len(fields) < 4 {
			t.Fatalf("fila mal formada en %q: %q", header, line)
		}
		got[strings.TrimSpace(fields[1])] = strings.TrimSpace(fields[2])
	}
	if !maps.Equal(got, want) {
		t.Fatalf("estados de %q = %#v; se esperaban %#v", header, got, want)
	}
}

func assertFileDigest(t *testing.T, path, want string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("leer %s: %v", path, err)
	}
	got := sha256.Sum256(content)
	if hex.EncodeToString(got[:]) != want {
		t.Fatalf("SHA-256 de %s = %x; se esperaba %s", path, got, want)
	}
}

func physicalCensorTreeDigest(t *testing.T) string {
	t.Helper()
	var paths []string
	err := filepath.WalkDir("scripts/legacy_physical_inventory", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type().IsRegular() {
			paths = append(paths, filepath.ToSlash(path))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("enumerar sujeto del censador: %v", err)
	}
	sort.Strings(paths)
	aggregate := sha256.New()
	for _, path := range paths {
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatalf("leer %s: %v", path, readErr)
		}
		sum := sha256.Sum256(content)
		_, _ = aggregate.Write([]byte(hex.EncodeToString(sum[:]) + "  " + path + "\n"))
	}
	return hex.EncodeToString(aggregate.Sum(nil))
}
