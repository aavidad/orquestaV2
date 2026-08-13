// Este contrato conserva una propuesta trazable; no crea trabajo, cierre, admisión ni estado canónico.
package orquesta_test

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

const behaviorBatch07Fixture = "product/traceability/fixtures/behavior_characterization_agent_batch_07.jsonl"

func TestBehaviorCharacterizationAgentBatch07(t *testing.T) {
	fixture := readBehaviorBatch07File(t, behaviorBatch07Fixture)
	ledger := readBehaviorBatch07File(t, "product/traceability/task_entries.jsonl")
	if err := validateBehaviorBatch07(fixture, ledger, behaviorBatch07PreviousFixtures(t)); err != nil {
		t.Fatal(err)
	}
}

func TestBehaviorCharacterizationAgentBatch07RejectsMutations(t *testing.T) {
	fixture := readBehaviorBatch07File(t, behaviorBatch07Fixture)
	ledger := readBehaviorBatch07File(t, "product/traceability/task_entries.jsonl")
	previous := behaviorBatch07PreviousFixtures(t)
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
		{"capacidad incorrecta", `"capability_id":"ORC-13"`, `"capability_id":"ORC-12"`},
		{"referencia anterior", `"TASKENTRY-c514901a23a3da2d638c2ace"`, `"TASKENTRY-d563352c3719db45f14a8ce5"`},
		{"contrato vacío", `"users":["motor de aplicación"`, `"users":[""`},
		{"ancla literal alterada", `Rails neutrales y outbox/replay sin producto.`, `rails sin ancla`},
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
			if err := validateBehaviorBatch07(changed, ledger, previous); err == nil {
				t.Fatal("la mutación autoritativa, duplicada o ambigua fue aceptada")
			}
		})
	}
	for _, size := range []int{3, 5} {
		t.Run("conducta con "+string(rune('0'+size))+" refs", func(t *testing.T) {
			changed := resizeBehaviorBatch07Record(t, fixture, "mutacion_y_outbox_coordinadas", size)
			if err := validateBehaviorBatch07(changed, ledger, previous); err == nil {
				t.Fatalf("la conducta con %d refs fue aceptada", size)
			}
		})
	}
	behaviors := []string{
		"mutacion_y_outbox_coordinadas", "outbox_pendiente_bloquea_avance",
		"recuperacion_outbox_sin_duplicar_efecto", "claim_lease_ack_correlacionados",
		"puerto_ledger_y_adaptadores_separados",
	}
	for left := 0; left < len(behaviors); left++ {
		for right := left + 1; right < len(behaviors); right++ {
			t.Run("intercambio "+behaviors[left]+" "+behaviors[right], func(t *testing.T) {
				changed := swapBehaviorBatch07Evidence(t, fixture, behaviors[left], behaviors[right])
				if err := validateBehaviorBatch07(changed, ledger, previous); err == nil {
					t.Fatal("el intercambio entre conductas de la misma capacidad fue aceptado")
				}
			})
		}
	}
	t.Run("LF final ausente", func(t *testing.T) {
		changed := bytes.TrimSuffix(fixture, []byte("\n"))
		if err := validateBehaviorBatch07(changed, ledger, previous); err == nil {
			t.Fatal("el fixture sin LF final fue aceptado")
		}
	})
}

func TestBehaviorCharacterizationAgentBatch07QualifiesHistoricalClaims(t *testing.T) {
	records, err := decodeBehaviorBatchRecords(readBehaviorBatch07File(t, behaviorBatch07Fixture))
	if err != nil {
		t.Fatal(err)
	}
	qualifiers := []string{"declar", "document", "propuest", "descrit", "registr", "inform", "históric", "no qued", "no consta"}
	for _, record := range records {
		for _, claims := range [][]string{record.Worked, record.Failed, record.Attempts} {
			for _, claim := range claims {
				normalized := strings.ToLower(claim)
				qualified := false
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

func behaviorBatch07PreviousFixtures(t *testing.T) [][]byte {
	t.Helper()
	return [][]byte{
		readBehaviorBatch07File(t, behaviorBatchFixture),
		readBehaviorBatch07File(t, behaviorBatch02Fixture),
		readBehaviorBatch07File(t, behaviorBatch03Fixture),
		readBehaviorBatch07File(t, behaviorBatch04Fixture),
		readBehaviorBatch07File(t, behaviorBatch05Fixture),
		readBehaviorBatch07File(t, behaviorBatch06Fixture),
	}
}

func readBehaviorBatch07File(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func resizeBehaviorBatch07Record(t *testing.T, raw []byte, behavior string, size int) []byte {
	t.Helper()
	records := decodeBehaviorBatch07ForMutation(t, raw)
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
		return encodeBehaviorBatch07Records(t, records)
	}
	t.Fatalf("conducta no encontrada: %s", behavior)
	return nil
}

func swapBehaviorBatch07Evidence(t *testing.T, raw []byte, leftBehavior, rightBehavior string) []byte {
	t.Helper()
	records := decodeBehaviorBatch07ForMutation(t, raw)
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
	records[left].EntryRefs[0], records[right].EntryRefs[0] = records[right].EntryRefs[0], records[left].EntryRefs[0]
	records[left].Evidence[0], records[right].Evidence[0] = records[right].Evidence[0], records[left].Evidence[0]
	return encodeBehaviorBatch07Records(t, records)
}

func decodeBehaviorBatch07ForMutation(t *testing.T, raw []byte) []behaviorBatchRecord {
	t.Helper()
	records, err := decodeBehaviorBatchRecords(raw)
	if err != nil {
		t.Fatal(err)
	}
	return records
}

func encodeBehaviorBatch07Records(t *testing.T, records []behaviorBatchRecord) []byte {
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
