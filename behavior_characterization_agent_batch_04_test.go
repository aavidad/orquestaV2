// Este contrato conserva una propuesta trazable; no crea trabajo, cierre, admisión ni estado canónico.
package orquesta_test

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

const behaviorBatch04Fixture = "product/traceability/fixtures/behavior_characterization_agent_batch_04.jsonl"

func TestBehaviorCharacterizationAgentBatch04(t *testing.T) {
	fixture := readBehaviorBatch04File(t, behaviorBatch04Fixture)
	ledger := readBehaviorBatch04File(t, "product/traceability/task_entries.jsonl")
	previous := [][]byte{
		readBehaviorBatch04File(t, behaviorBatchFixture),
		readBehaviorBatch04File(t, behaviorBatch02Fixture),
		readBehaviorBatch04File(t, behaviorBatch03Fixture),
	}
	if err := validateBehaviorBatch04(fixture, ledger, previous); err != nil {
		t.Fatal(err)
	}
}

func TestBehaviorCharacterizationAgentBatch04RejectsMutations(t *testing.T) {
	fixture := readBehaviorBatch04File(t, behaviorBatch04Fixture)
	ledger := readBehaviorBatch04File(t, "product/traceability/task_entries.jsonl")
	previous := [][]byte{
		readBehaviorBatch04File(t, behaviorBatchFixture),
		readBehaviorBatch04File(t, behaviorBatch02Fixture),
		readBehaviorBatch04File(t, behaviorBatch03Fixture),
	}
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
		{"capacidad incorrecta", `"capability_id":"AGT-01"`, `"capability_id":"AGT-03"`},
		{"referencia de lote anterior", `"TASKENTRY-a45cf00438c4e53899e03cc2"`, `"TASKENTRY-d1e54d14ccd52a1241c38235"`},
		{"contrato vacío", `"users":["motor de aplicación"`, `"users":[""`},
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
			if err := validateBehaviorBatch04(changed, ledger, previous); err == nil {
				t.Fatal("la mutación autoritativa, duplicada o ambigua fue aceptada")
			}
		})
	}
	for _, size := range []int{3, 5} {
		t.Run("conducta con "+string(rune('0'+size))+" refs", func(t *testing.T) {
			changed := resizeBehaviorBatch04Record(t, fixture, "protocolo_comun_de_dispatch", size)
			if err := validateBehaviorBatch04(changed, ledger, previous); err == nil {
				t.Fatalf("la conducta con %d refs fue aceptada", size)
			}
		})
	}
	behaviors := []string{
		"protocolo_comun_de_dispatch",
		"peticion_neutral_de_runtime",
		"solicitud_durable_a_launch_validado",
		"runtime_de_proceso_controlado_y_aislado",
		"adaptador_externo_opt_in",
	}
	for left := 0; left < len(behaviors); left++ {
		for right := left + 1; right < len(behaviors); right++ {
			t.Run("intercambio "+behaviors[left]+" "+behaviors[right], func(t *testing.T) {
				changed := swapBehaviorBatch04Evidence(t, fixture, behaviors[left], behaviors[right])
				if err := validateBehaviorBatch04(changed, ledger, previous); err == nil {
					t.Fatal("el intercambio entre conductas de la misma capacidad fue aceptado")
				}
			})
		}
	}
	t.Run("LF final ausente", func(t *testing.T) {
		changed := bytes.TrimSuffix(fixture, []byte("\n"))
		if err := validateBehaviorBatch04(changed, ledger, previous); err == nil {
			t.Fatal("el fixture sin LF final fue aceptado")
		}
	})
}

func TestBehaviorCharacterizationAgentBatch04QualifiesHistoricalClaims(t *testing.T) {
	records, err := decodeBehaviorBatchRecords(readBehaviorBatch04File(t, behaviorBatch04Fixture))
	if err != nil {
		t.Fatal(err)
	}
	qualifiers := []string{"declar", "document", "propuest", "descrit", "registr", "inform", "históric",
		"no qued", "no consta"}
	for _, record := range records {
		for _, claims := range [][]string{record.Worked, record.Failed, record.Attempts} {
			for _, claim := range claims {
				qualified := false
				for _, qualifier := range qualifiers {
					qualified = qualified || strings.Contains(claim, qualifier)
				}
				if !qualified {
					t.Errorf("afirmación histórica sin calificar: %q", claim)
				}
			}
		}
	}
}

func readBehaviorBatch04File(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func resizeBehaviorBatch04Record(t *testing.T, raw []byte, behavior string, size int) []byte {
	t.Helper()
	records := decodeBehaviorBatch04ForMutation(t, raw)
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
		return encodeBehaviorBatch04Records(t, records)
	}
	t.Fatalf("conducta no encontrada: %s", behavior)
	return nil
}

func swapBehaviorBatch04Evidence(t *testing.T, raw []byte, leftBehavior, rightBehavior string) []byte {
	t.Helper()
	records := decodeBehaviorBatch04ForMutation(t, raw)
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
	return encodeBehaviorBatch04Records(t, records)
}

func decodeBehaviorBatch04ForMutation(t *testing.T, raw []byte) []behaviorBatchRecord {
	t.Helper()
	records, err := decodeBehaviorBatchRecords(raw)
	if err != nil {
		t.Fatal(err)
	}
	return records
}

func encodeBehaviorBatch04Records(t *testing.T, records []behaviorBatchRecord) []byte {
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
