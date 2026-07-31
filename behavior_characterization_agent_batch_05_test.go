// Este contrato conserva una propuesta trazable; no crea trabajo, cierre, admisión ni estado canónico.
package orquesta_test

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

const behaviorBatch05Fixture = "product/traceability/fixtures/behavior_characterization_agent_batch_05.jsonl"

func TestBehaviorCharacterizationAgentBatch05(t *testing.T) {
	fixture := readBehaviorBatch05File(t, behaviorBatch05Fixture)
	ledger := readBehaviorBatch05File(t, "product/traceability/task_entries.jsonl")
	previous := behaviorBatch05PreviousFixtures(t)
	if err := validateBehaviorBatch05(fixture, ledger, previous); err != nil {
		t.Fatal(err)
	}
}

func TestBehaviorCharacterizationAgentBatch05RejectsMutations(t *testing.T) {
	fixture := readBehaviorBatch05File(t, behaviorBatch05Fixture)
	ledger := readBehaviorBatch05File(t, "product/traceability/task_entries.jsonl")
	previous := behaviorBatch05PreviousFixtures(t)
	mutations := []struct {
		name, old, replacement string
	}{
		{"autoridad canónica", `"authority":"proposal_fixture_not_canonical_ledger"`, `"authority":"canonical_ledger"`},
		{"cambio canónico", `"canonical_state_change":false`, `"canonical_state_change":true`},
		{"crea trabajo", `"creates_work_item":false`, `"creates_work_item":true`},
		{"cierra capacidad", `"closes_capability":false`, `"closes_capability":true`},
		{"afirma acreditación", `"claims_accreditation":false`, `"claims_accreditation":true`},
		{"revisión aceptada", `"review_state":"bootstrap_first_review_pending_independent_counterreview"`, `"review_state":"accepted"`},
		{"disposición evaluada", `"disposition":"not_evaluated"`, `"disposition":"accepted"`},
		{"capacidad incorrecta", `"capability_id":"AGT-03"`, `"capability_id":"AGT-01"`},
		{"referencia de lote anterior", `"TASKENTRY-4ca8a7b17d4cd9e364bafdd9"`, `"TASKENTRY-a45cf00438c4e53899e03cc2"`},
		{"contrato vacío", `"users":["composición autorizada"`, `"users":[""`},
		{"ancla literal alterada", `Composicion Codex opt-in, closure source y supervisor`, `composición inespecífica`},
		{"campo desconocido", `{"schema_version":1`, `{"unknown":true,"schema_version":1`},
		{"clave JSON duplicada", `{"schema_version":1`, `{"schema_version":2,"schema_version":1`},
		{"valor posterior", `}` + "\n", `} {}` + "\n"},
		{"representación no canónica", `{"schema_version":1`, ` {"schema_version":1`},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			changed := bytes.Replace(fixture, []byte(mutation.old), []byte(mutation.replacement), 1)
			if bytes.Equal(changed, fixture) {
				t.Fatal("la mutación no cambió el fixture")
			}
			if err := validateBehaviorBatch05(changed, ledger, previous); err == nil {
				t.Fatal("la mutación autoritativa, duplicada o ambigua fue aceptada")
			}
		})
	}
	for _, size := range []int{3, 5} {
		t.Run("conducta con "+string(rune('0'+size))+" refs", func(t *testing.T) {
			changed := resizeBehaviorBatch05Record(t, fixture, "composicion_codex_opt_in_y_compatibilidad", size)
			if err := validateBehaviorBatch05(changed, ledger, previous); err == nil {
				t.Fatalf("la conducta con %d refs fue aceptada", size)
			}
		})
	}
	for _, pair := range [][2]string{
		{"composicion_codex_opt_in_y_compatibilidad", "codex_real_aislamiento_y_apagado"},
		{"normalizacion_y_contrato_runtime_neutrales", "observacion_recuperacion_y_paridad_de_runtime"},
	} {
		t.Run("intercambio "+pair[0]+" "+pair[1], func(t *testing.T) {
			changed := swapBehaviorBatch05Evidence(t, fixture, pair[0], pair[1])
			if err := validateBehaviorBatch05(changed, ledger, previous); err == nil {
				t.Fatal("el intercambio entre conductas de la misma capacidad fue aceptado")
			}
		})
	}
	t.Run("LF final ausente", func(t *testing.T) {
		changed := bytes.TrimSuffix(fixture, []byte("\n"))
		if err := validateBehaviorBatch05(changed, ledger, previous); err == nil {
			t.Fatal("el fixture sin LF final fue aceptado")
		}
	})
}

func TestBehaviorCharacterizationAgentBatch05QualifiesHistoricalClaims(t *testing.T) {
	records, err := decodeBehaviorBatchRecords(readBehaviorBatch05File(t, behaviorBatch05Fixture))
	if err != nil {
		t.Fatal(err)
	}
	qualifiers := []string{
		"declar", "document", "propuest", "descrit", "registr", "inform", "históric",
		"no qued", "no consta",
	}
	for _, record := range records {
		for _, claims := range [][]string{record.Worked, record.Failed, record.Attempts} {
			for _, claim := range claims {
				qualified := false
				normalized := strings.ToLower(claim)
				for _, qualifier := range qualifiers {
					qualified = qualified || strings.Contains(normalized, qualifier)
				}
				if !qualified {
					t.Errorf("afirmación histórica sin calificar: %q", claim)
				}
			}
		}
	}
}

func behaviorBatch05PreviousFixtures(t *testing.T) [][]byte {
	t.Helper()
	return [][]byte{
		readBehaviorBatch05File(t, behaviorBatchFixture),
		readBehaviorBatch05File(t, behaviorBatch02Fixture),
		readBehaviorBatch05File(t, behaviorBatch03Fixture),
		readBehaviorBatch05File(t, behaviorBatch04Fixture),
	}
}

func readBehaviorBatch05File(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func resizeBehaviorBatch05Record(t *testing.T, raw []byte, behavior string, size int) []byte {
	t.Helper()
	records := decodeBehaviorBatch05ForMutation(t, raw)
	for index := range records {
		if records[index].Behavior != behavior {
			continue
		}
		switch size {
		case 3:
			records[index].EntryRefs = records[index].EntryRefs[:3]
			records[index].Evidence = records[index].Evidence[:3]
		case 5:
			records[index].EntryRefs = append(records[index].EntryRefs, records[index].EntryRefs[0])
			records[index].Evidence = append(records[index].Evidence, records[index].Evidence[0])
		default:
			t.Fatalf("tamaño de mutación no soportado: %d", size)
		}
		return encodeBehaviorBatch05Records(t, records)
	}
	t.Fatalf("conducta no encontrada: %s", behavior)
	return nil
}

func swapBehaviorBatch05Evidence(t *testing.T, raw []byte, leftBehavior, rightBehavior string) []byte {
	t.Helper()
	records := decodeBehaviorBatch05ForMutation(t, raw)
	left, right := -1, -1
	for index := range records {
		switch records[index].Behavior {
		case leftBehavior:
			left = index
		case rightBehavior:
			right = index
		}
	}
	if left < 0 || right < 0 {
		t.Fatal("no se encontraron dos conductas intercambiables")
	}
	records[left].EntryRefs[0], records[right].EntryRefs[0] =
		records[right].EntryRefs[0], records[left].EntryRefs[0]
	records[left].Evidence[0], records[right].Evidence[0] =
		records[right].Evidence[0], records[left].Evidence[0]
	return encodeBehaviorBatch05Records(t, records)
}

func decodeBehaviorBatch05ForMutation(t *testing.T, raw []byte) []behaviorBatchRecord {
	t.Helper()
	records, err := decodeBehaviorBatchRecords(raw)
	if err != nil {
		t.Fatal(err)
	}
	return records
}

func encodeBehaviorBatch05Records(t *testing.T, records []behaviorBatchRecord) []byte {
	t.Helper()
	var encoded bytes.Buffer
	for _, record := range records {
		line, err := json.Marshal(record)
		if err != nil {
			t.Fatal(err)
		}
		encoded.Write(line)
		encoded.WriteByte('\n')
	}
	return encoded.Bytes()
}
