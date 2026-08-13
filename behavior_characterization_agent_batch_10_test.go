// Este contrato conserva una propuesta trazable; no crea trabajo, cierre, admisión ni estado canónico.
package orquesta_test

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

const behaviorBatch10Fixture = "product/traceability/fixtures/behavior_characterization_agent_batch_10.jsonl"

func TestBehaviorCharacterizationAgentBatch10(t *testing.T) {
	fixture := readBehaviorBatch10File(t, behaviorBatch10Fixture)
	ledger := readBehaviorBatch10File(t, "product/traceability/task_entries.jsonl")
	if err := validateBehaviorBatch10(fixture, ledger, behaviorBatch10PreviousFixtures(t)); err != nil {
		t.Fatal(err)
	}
}

func TestBehaviorCharacterizationAgentBatch10RejectsMutations(t *testing.T) {
	fixture := readBehaviorBatch10File(t, behaviorBatch10Fixture)
	ledger := readBehaviorBatch10File(t, "product/traceability/task_entries.jsonl")
	previous := behaviorBatch10PreviousFixtures(t)
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
		{"capacidad incorrecta", `"capability_id":"ORC-12"`, `"capability_id":"ORC-13"`},
		{"referencia del lote 06", `"TASKENTRY-01f2042a0abbe1756bb04e99"`, `"TASKENTRY-d563352c3719db45f14a8ce5"`},
		{"fuentes tratadas como independientes", `no son evidencia independiente`, `son evidencia independiente`},
		{"ancla literal alterada", `non_idempotent_mutation_retry_blocked`, `retry_mutation_unknown`},
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
			if err := validateBehaviorBatch10(changed, ledger, previous); err == nil {
				t.Fatal("la mutación autoritativa, duplicada o ambigua fue aceptada")
			}
		})
	}
	for _, size := range []int{3, 5} {
		t.Run("conducta con "+string(rune('0'+size))+" refs", func(t *testing.T) {
			changed := resizeBehaviorBatch10Record(t, fixture, "retry_saliente_con_presupuesto_y_deadline", size)
			if err := validateBehaviorBatch10(changed, ledger, previous); err == nil {
				t.Fatalf("la conducta con %d refs fue aceptada", size)
			}
		})
	}
	behaviors := []string{
		"retry_saliente_con_presupuesto_y_deadline", "reintento_publico_segun_clase_de_operacion",
		"reconciliacion_antes_de_repetir_efecto_externo", "presupuesto_y_reentrada_de_reparaciones",
	}
	for left := 0; left < len(behaviors); left++ {
		for right := left + 1; right < len(behaviors); right++ {
			t.Run("intercambio "+behaviors[left]+" "+behaviors[right], func(t *testing.T) {
				changed := swapBehaviorBatch10Evidence(t, fixture, behaviors[left], behaviors[right])
				if err := validateBehaviorBatch10(changed, ledger, previous); err == nil {
					t.Fatal("el intercambio entre conductas de la misma capacidad fue aceptado")
				}
			})
		}
	}
	t.Run("LF final ausente", func(t *testing.T) {
		changed := bytes.TrimSuffix(fixture, []byte("\n"))
		if err := validateBehaviorBatch10(changed, ledger, previous); err == nil {
			t.Fatal("el fixture sin LF final fue aceptado")
		}
	})
}

func TestBehaviorCharacterizationAgentBatch10QualifiesHistoricalClaims(t *testing.T) {
	records, err := decodeBehaviorBatchRecords(readBehaviorBatch10File(t, behaviorBatch10Fixture))
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

func TestBehaviorCharacterizationAgentBatch10DoesNotRepeatBatch06Behaviors(t *testing.T) {
	records, err := decodeBehaviorBatchRecords(readBehaviorBatch10File(t, behaviorBatch10Fixture))
	if err != nil {
		t.Fatal(err)
	}
	batch06 := map[string]struct{}{
		"lease_y_timeout_sin_reloj_oculto":   {},
		"identidad_fuerte_para_replay_y_cas": {},
		"tick_y_supervision_idempotentes":    {},
	}
	for _, record := range records {
		if _, duplicate := batch06[record.Behavior]; duplicate {
			t.Fatalf("conducta del lote 06 repetida: %s", record.Behavior)
		}
	}
}

func behaviorBatch10PreviousFixtures(t *testing.T) [][]byte {
	t.Helper()
	return [][]byte{
		readBehaviorBatch10File(t, behaviorBatchFixture),
		readBehaviorBatch10File(t, behaviorBatch02Fixture),
		readBehaviorBatch10File(t, behaviorBatch03Fixture),
		readBehaviorBatch10File(t, behaviorBatch04Fixture),
		readBehaviorBatch10File(t, behaviorBatch05Fixture),
		readBehaviorBatch10File(t, behaviorBatch06Fixture),
		readBehaviorBatch10File(t, behaviorBatch07Fixture),
	}
}

func readBehaviorBatch10File(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func resizeBehaviorBatch10Record(t *testing.T, raw []byte, behavior string, size int) []byte {
	t.Helper()
	records := decodeBehaviorBatch10ForMutation(t, raw)
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
		return encodeBehaviorBatch10Records(t, records)
	}
	t.Fatalf("conducta no encontrada: %s", behavior)
	return nil
}

func swapBehaviorBatch10Evidence(t *testing.T, raw []byte, leftBehavior, rightBehavior string) []byte {
	t.Helper()
	records := decodeBehaviorBatch10ForMutation(t, raw)
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
	return encodeBehaviorBatch10Records(t, records)
}

func decodeBehaviorBatch10ForMutation(t *testing.T, raw []byte) []behaviorBatchRecord {
	t.Helper()
	records, err := decodeBehaviorBatchRecords(raw)
	if err != nil {
		t.Fatal(err)
	}
	return records
}

func encodeBehaviorBatch10Records(t *testing.T, records []behaviorBatchRecord) []byte {
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
