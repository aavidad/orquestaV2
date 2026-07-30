// Contrato documental del futuro censo físico: solo compara el manifiesto V3
// comprometido con su plan y nunca consulta raíces físicas del equipo.
package orquesta_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"maps"
	"os"
	"strings"
	"testing"
)

const (
	legacyPhysicalCensusPlanPath = "docs/reconstruccion/plan_ejecucion_censo_fisico_historico_2026-07-30.md"
	legacyPendingIDsDigest       = "2d2b81b27bfccad5a0b3a6dfabdec299a516dc3b31400f2cb2ecd6948c1e7834"
	legacyWrongPendingIDsDigest  = "c21fc65ff620ec1539646546787e6358a5319343b543749d46cc4fa16c4417aa"
	legacyRootIDsSortedLFDigest  = "sha256:c4a93e4679a045ac68d835accb7bfd0ff0b6cb918bc2a7c9ab1248d3803291ec"
)

func TestLegacyPhysicalCensusPlanBindsPendingUniverse(t *testing.T) {
	var document legacyRootsDocument
	traceDecodeStrict(t, legacySourceRootsPath, &document)

	pending := document.BlockingPhysicalCensusRootIDs
	if len(pending) != 112 {
		t.Fatalf("censos físicos pendientes = %d; se esperaban 112", len(pending))
	}

	roots := make(map[string]legacyRootsRoot, len(document.Roots))
	rootIDs := make([]string, 0, len(document.Roots))
	for _, root := range document.Roots {
		if _, duplicate := roots[root.ID]; duplicate {
			t.Fatalf("referencia raíz duplicada: %q", root.ID)
		}
		roots[root.ID] = root
		rootIDs = append(rootIDs, root.ID)
	}
	if got := legacyLinesDigest(rootIDs); got != legacyRootIDsSortedLFDigest ||
		document.RootIDsSHA256 != legacyRootIDsSortedLFDigest {
		t.Fatalf("dominio sorted-LF de raíces incoherente: calculado=%s documento=%s", got, document.RootIDsSHA256)
	}
	collections := make(map[string]legacyRootsCollection, len(document.Collections))
	for _, collection := range document.Collections {
		if _, duplicate := collections[collection.RootID]; duplicate {
			t.Fatalf("registro de colección duplicado: %q", collection.RootID)
		}
		collections[collection.RootID] = collection
	}
	if len(document.Collections) != 15 || len(collections) != 15 {
		t.Fatalf("colecciones = %d/%d; se esperaban exactamente 15", len(document.Collections), len(collections))
	}
	seen := make(map[string]struct{}, len(pending))
	originCounts := make(map[string]int)
	decisionCounts := make(map[string]int)
	simpleCount, collectionCount := 0, 0
	simplePresentV3, simpleAbsentV3 := 0, 0
	memberCount, memberPresentV3, memberAbsentV3 := 0, 0, 0
	digest := sha256.New()
	for _, id := range pending {
		if _, duplicate := seen[id]; duplicate {
			t.Fatalf("identificador físico pendiente duplicado: %q", id)
		}
		seen[id] = struct{}{}
		root, exists := roots[id]
		if !exists || !root.PhysicalCensusRequired || root.PhysicalCensusStatus != "pendiente" {
			t.Fatalf("referencia pendiente incoherente: %q", id)
		}
		originCounts[root.ObservationBatchID]++
		decisionCounts[root.SourceDecision]++
		switch root.ScopeKind {
		case "single":
			simpleCount++
			if root.ExistsAtObservation {
				simplePresentV3++
			} else {
				simpleAbsentV3++
			}
		case "collection":
			collection, found := collections[id]
			if !found {
				t.Fatalf("colección pendiente sin membresía sellada: %q", id)
			}
			collectionCount++
			memberCount += len(collection.Members)
			encoded, marshalErr := json.Marshal(collection.Members)
			if marshalErr != nil {
				t.Fatalf("codificar miembros de %q: %v", id, marshalErr)
			}
			memberDigest := sha256.New()
			_, _ = memberDigest.Write(encoded)
			_, _ = memberDigest.Write([]byte{'\n'})
			if got := "sha256:" + hex.EncodeToString(memberDigest.Sum(nil)); got != collection.MembersSHA256 {
				t.Fatalf("members_sha256 de %q = %s; se esperaba %s", id, got, collection.MembersSHA256)
			}
			aliases := make(map[string]struct{}, len(collection.Members))
			for _, member := range collection.Members {
				if member.PathAlias == "" {
					t.Fatalf("miembro sin alias en %q", id)
				}
				if _, duplicate := aliases[member.PathAlias]; duplicate {
					t.Fatalf("alias de miembro duplicado en %q: %q", id, member.PathAlias)
				}
				aliases[member.PathAlias] = struct{}{}
				if member.ExistsAtObservation {
					memberPresentV3++
				} else {
					memberAbsentV3++
				}
			}
		default:
			t.Fatalf("alcance inesperado en %q: %q", id, root.ScopeKind)
		}
		_, _ = digest.Write([]byte(id))
		_, _ = digest.Write([]byte{0})
	}

	wantOrigins := map[string]int{
		"base_git_023829":                 95,
		"huecos_worktrees_023829_033143":  4,
		"externos_metadata_023829_033143": 11,
		"vec_vivo_035814_035815":          2,
	}
	wantDecisions := map[string]int{"incluir": 36, "excluir": 49, "pendiente": 27}
	if !maps.Equal(originCounts, wantOrigins) {
		t.Fatalf("partición por origen = %#v; se esperaba %#v", originCounts, wantOrigins)
	}
	if !maps.Equal(decisionCounts, wantDecisions) {
		t.Fatalf("partición por decisión = %#v; se esperaba %#v", decisionCounts, wantDecisions)
	}
	if simpleCount != 97 || collectionCount != 15 || memberCount != 285 ||
		simpleCount+memberCount != 382 ||
		simplePresentV3 != 96 || simpleAbsentV3 != 1 ||
		memberPresentV3 != 268 || memberAbsentV3 != 17 ||
		simplePresentV3+memberPresentV3 != 364 ||
		simpleAbsentV3+memberAbsentV3 != 18 {
		t.Fatalf(
			"expansión incoherente: simples=%d (%d/%d V3), colecciones=%d, miembros=%d (%d/%d V3), sujetos=%d (%d/%d V3)",
			simpleCount, simplePresentV3, simpleAbsentV3,
			collectionCount, memberCount, memberPresentV3, memberAbsentV3,
			simpleCount+memberCount,
			simplePresentV3+memberPresentV3,
			simpleAbsentV3+memberAbsentV3,
		)
	}
	if got := hex.EncodeToString(digest.Sum(nil)); got != legacyPendingIDsDigest {
		t.Fatalf("SHA-256 con NUL por identificador = %s; se esperaba %s", got, legacyPendingIDsDigest)
	}

	planBytes, err := os.ReadFile(legacyPhysicalCensusPlanPath)
	if err != nil {
		t.Fatalf("leer plan del censo físico: %v", err)
	}
	plan := string(planBytes)
	if strings.Count(plan, legacyPendingIDsDigest) != 2 ||
		strings.Count(plan, legacyRootIDsSortedLFDigest) != 1 ||
		strings.Contains(plan, legacyWrongPendingIDsDigest) {
		t.Fatal("el plan no conserva exactamente las dos huellas correctas o recuperó la huella errónea")
	}
	for _, required := range []string{
		`jq -j '.blocking_physical_census_root_ids[] | ., "\u0000"'`,
		"`B01`–`B05`", "`B06`", "`H01`", "`E01`", "`V01`", "**9 particiones**",
		"97 simples", "15 colecciones", "285 miembros", "`members_sha256`",
		"| `single` | 97 | 97 | 96 | 1 |",
		"| `collection` | 15 | 285 miembros | 268 | 17 |",
		"| **Total expandido** | **112** | **382** | **364** | **18** |",
		"`exists_at_v3_observation`", "`present_in_stable_view`",
		"no preasignan ausencias", "debe reobservar los 382 sujetos",
		"orquesta.legacy-root-ids.sorted-lf.v1", "LC_ALL=C sort",
		"V3 -> vista_estable_y_cercado -> mapeo -> censo -> copia -> liberar_vista -> censo_Git_sobre_copia",
		"Estado: plan condicionado.", "No autoriza todavía",
		"no se declara compatible ni aceptado", "quinta auditoría independiente",
		"dispositivo físico distinto", "recuperación conservadora",
		"sin borrado automático", "`recovery_conflict`",
		"| `max_regular_file_bytes` | no aplica a lectura; tamaño solo observado | 2 GiB |",
		"| `max_scratch_bytes` | 512 MiB | 2 GiB |",
		"| `max_total_attempt_bytes` | 1 GiB | 4 GiB |",
		"dominio_utf8 || 0x00 || json_canonico_sin_digest || 0x0a",
		"UID/GID sí se emiten; faltan tiempos, `nlink`, xattrs, ACL y capacidades",
	} {
		if !strings.Contains(plan, required) {
			t.Errorf("el plan no contiene el contrato %q", required)
		}
	}
	for _, forbidden := range []string{
		"V3 -> lote físico -> congelación",
		"miembros_existentes != 268",
		"Los 17 miembros y la referencia simple no existentes",
		"| `max_regular_file_bytes` | 16 GiB",
		"este plan no promete conservarlas ni borrarlas",
	} {
		if strings.Contains(plan, forbidden) {
			t.Fatalf("el plan recuperó el contrato descartado %q", forbidden)
		}
	}
}
