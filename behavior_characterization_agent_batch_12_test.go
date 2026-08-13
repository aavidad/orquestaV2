// Este contrato conserva una propuesta trazable; no crea trabajo, cierre, admisión ni estado canónico.
package orquesta_test

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

const behaviorBatch12Fixture = "product/traceability/fixtures/behavior_characterization_agent_batch_12.jsonl"

func TestBehaviorCharacterizationAgentBatch12(t *testing.T) {
	fixture := readBehaviorBatch12File(t, behaviorBatch12Fixture)
	ledger := readBehaviorBatch12File(t, "product/traceability/task_entries.jsonl")
	roadmap := readBehaviorBatch12File(t, "product/roadmap.json")
	if err := validateBehaviorBatch12(fixture, ledger, roadmap, behaviorBatch12PreviousFixtures(t)); err != nil {
		t.Fatal(err)
	}
}

func TestBehaviorCharacterizationAgentBatch12RejectsMutations(t *testing.T) {
	fixture := readBehaviorBatch12File(t, behaviorBatch12Fixture)
	ledger := readBehaviorBatch12File(t, "product/traceability/task_entries.jsonl")
	roadmap := readBehaviorBatch12File(t, "product/roadmap.json")
	previous := behaviorBatch12PreviousFixtures(t)
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
		{"referencia del lote 10", `"TASKENTRY-ad9fc8e67198a91192637027"`, `"TASKENTRY-01f2042a0abbe1756bb04e99"`},
		{"fuentes tratadas como independientes", `no son evidencia independiente`, `son evidencia independiente`},
		{"ancla literal alterada", `backlogTaskIDAllocationV0`, `claimInventadaV0`},
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
			if err := validateBehaviorBatch12(changed, ledger, roadmap, previous); err == nil {
				t.Fatal("la mutación autoritativa, duplicada o ambigua fue aceptada")
			}
		})
	}
	for _, size := range []int{3, 5} {
		t.Run("cantidad de refs", func(t *testing.T) {
			changed := resizeBehaviorBatch12Record(t, fixture, "reserva_causal_antes_de_lanzar_o_fusionar", size)
			if err := validateBehaviorBatch12(changed, ledger, roadmap, previous); err == nil {
				t.Fatalf("la conducta con %d refs fue aceptada", size)
			}
		})
	}
	t.Run("intercambio entre conductas ORC-12", func(t *testing.T) {
		changed := swapBehaviorBatch12Evidence(t, fixture,
			"progreso_expiracion_y_guardian_con_lease_vivo", "identidad_orden_y_dedupe_documental")
		if err := validateBehaviorBatch12(changed, ledger, roadmap, previous); err == nil {
			t.Fatal("el intercambio semántico fue aceptado")
		}
	})
	t.Run("LF final ausente", func(t *testing.T) {
		changed := bytes.TrimSuffix(fixture, []byte("\n"))
		if err := validateBehaviorBatch12(changed, ledger, roadmap, previous); err == nil {
			t.Fatal("el fixture sin LF final fue aceptado")
		}
	})
	t.Run("roadmap degrada ORC-12 acreditada", func(t *testing.T) {
		changed := mutateBehaviorBatch12RoadmapStatus(t, roadmap, "declared")
		if err := validateBehaviorBatch12(fixture, ledger, changed, previous); err == nil {
			t.Fatal("el estado roadmap mutado fue aceptado")
		}
	})
}

func TestBehaviorCharacterizationAgentBatch12QualifiesHistoricalClaims(t *testing.T) {
	records, err := decodeBehaviorBatchRecords(readBehaviorBatch12File(t, behaviorBatch12Fixture))
	if err != nil {
		t.Fatal(err)
	}
	qualifiers := []string{"declar", "document", "propuest", "descrit", "registr", "inform", "históric", "no qued", "no consta", "código histórico"}
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

func behaviorBatch12PreviousFixtures(t *testing.T) [][]byte {
	t.Helper()
	paths := []string{
		"product/traceability/fixtures/behavior_characterization_agent_batch_01.jsonl",
		"product/traceability/fixtures/behavior_characterization_agent_batch_02.jsonl",
		"product/traceability/fixtures/behavior_characterization_agent_batch_03.jsonl",
		"product/traceability/fixtures/behavior_characterization_agent_batch_04.jsonl",
		"product/traceability/fixtures/behavior_characterization_agent_batch_05.jsonl",
		"product/traceability/fixtures/behavior_characterization_agent_batch_06.jsonl",
		"product/traceability/fixtures/behavior_characterization_agent_batch_07.jsonl",
		"product/traceability/fixtures/behavior_characterization_agent_batch_08.jsonl",
		"product/traceability/fixtures/behavior_characterization_agent_batch_09.jsonl",
		"product/traceability/fixtures/behavior_characterization_agent_batch_10.jsonl",
		"product/traceability/fixtures/behavior_characterization_agent_batch_11.jsonl",
	}
	result := make([][]byte, 0, len(paths))
	for _, path := range paths {
		result = append(result, readBehaviorBatch12File(t, path))
	}
	return result
}

func readBehaviorBatch12File(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func resizeBehaviorBatch12Record(t *testing.T, raw []byte, behavior string, size int) []byte {
	t.Helper()
	records := decodeBehaviorBatch12ForMutation(t, raw)
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
		return encodeBehaviorBatch12Records(t, records)
	}
	t.Fatalf("conducta no encontrada: %s", behavior)
	return nil
}

func swapBehaviorBatch12Evidence(t *testing.T, raw []byte, leftBehavior, rightBehavior string) []byte {
	t.Helper()
	records := decodeBehaviorBatch12ForMutation(t, raw)
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
	return encodeBehaviorBatch12Records(t, records)
}

func decodeBehaviorBatch12ForMutation(t *testing.T, raw []byte) []behaviorBatchRecord {
	t.Helper()
	records, err := decodeBehaviorBatchRecords(raw)
	if err != nil {
		t.Fatal(err)
	}
	return records
}

func encodeBehaviorBatch12Records(t *testing.T, records []behaviorBatchRecord) []byte {
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

func mutateBehaviorBatch12RoadmapStatus(t *testing.T, raw []byte, status string) []byte {
	t.Helper()
	var roadmap map[string]any
	if err := json.Unmarshal(raw, &roadmap); err != nil {
		t.Fatal(err)
	}
	entries, ok := roadmap["capability_entries"].([]any)
	if !ok {
		t.Fatal("capability_entries ausente")
	}
	for _, value := range entries {
		entry, entryOK := value.(map[string]any)
		if !entryOK || entry["id"] != "ORC-12" {
			continue
		}
		entry["status"] = status
		changed, err := json.Marshal(roadmap)
		if err != nil {
			t.Fatal(err)
		}
		return changed
	}
	t.Fatal("ORC-12 ausente")
	return nil
}
