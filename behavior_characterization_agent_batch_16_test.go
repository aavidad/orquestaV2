// Este contrato conserva una propuesta trazable; no crea trabajo, cierre, admisión ni estado canónico.
package orquesta_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

const behaviorBatch16Fixture = "product/traceability/fixtures/behavior_characterization_agent_batch_16.jsonl"

func TestBehaviorCharacterizationAgentBatch16(t *testing.T) {
	fixture := readBehaviorBatch16File(t, behaviorBatch16Fixture)
	ledger := readBehaviorBatch16File(t, "product/traceability/task_entries.jsonl")
	roadmap := readBehaviorBatch16File(t, "product/roadmap.json")
	if err := validateBehaviorBatch16(fixture, ledger, roadmap, behaviorBatch16PreviousFixtures(t)); err != nil {
		t.Fatal(err)
	}
}

func TestBehaviorCharacterizationAgentBatch16RejectsMutations(t *testing.T) {
	fixture := readBehaviorBatch16File(t, behaviorBatch16Fixture)
	ledger := readBehaviorBatch16File(t, "product/traceability/task_entries.jsonl")
	roadmap := readBehaviorBatch16File(t, "product/roadmap.json")
	previous := behaviorBatch16PreviousFixtures(t)
	mutations := []struct{ name, old, replacement string }{
		{"autoridad canónica", `"authority":"proposal_fixture_not_canonical_ledger"`, `"authority":"canonical_ledger"`},
		{"cambio canónico", `"canonical_state_change":false`, `"canonical_state_change":true`},
		{"crea trabajo", `"creates_work_item":false`, `"creates_work_item":true`},
		{"cierra capacidad", `"closes_capability":false`, `"closes_capability":true`},
		{"afirma acreditación", `"claims_accreditation":false`, `"claims_accreditation":true`},
		{"revisión aceptada", `"review_state":"bootstrap_first_review_pending_independent_counterreview"`, `"review_state":"accepted"`},
		{"capacidad incorrecta", `"capability_id":"EVD-04"`, `"capability_id":"EVD-05"`},
		{"referencia reutilizada", `"TASKENTRY-5946b8840ca9db706e194260"`, `"TASKENTRY-ad9fc8e67198a91192637027"`},
		{"fuentes solapadas independientes", `no son evidencia independiente`, `son evidencia independiente`},
		{"ancla semántica", `TestAttestor independiente`, `runner no independiente`},
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
			if err := validateBehaviorBatch16(changed, ledger, roadmap, previous); err == nil {
				t.Fatal("la mutación autoritativa, reutilizada o ambigua fue aceptada")
			}
		})
	}
	t.Run("LF final ausente", func(t *testing.T) {
		if err := validateBehaviorBatch16(bytes.TrimSuffix(fixture, []byte("\n")), ledger, roadmap, previous); err == nil {
			t.Fatal("el fixture sin LF final fue aceptado")
		}
	})
	t.Run("roadmap acredita UI-03 sin evidencia", func(t *testing.T) {
		changed := mutateBehaviorBatch16RoadmapStatus(t, roadmap, "accredited")
		if err := validateBehaviorBatch16(fixture, ledger, changed, previous); err == nil {
			t.Fatal("el estado roadmap mutado fue aceptado")
		}
	})
}

func TestBehaviorCharacterizationAgentBatch16QualifiesHistoricalClaims(t *testing.T) {
	records, err := decodeBehaviorBatchRecords(readBehaviorBatch16File(t, behaviorBatch16Fixture))
	if err != nil {
		t.Fatal(err)
	}
	qualifiers := []string{"declar", "document", "registr", "inform", "históric", "no consta", "código legacy"}
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

func behaviorBatch16PreviousFixtures(t *testing.T) [][]byte {
	t.Helper()
	result := make([][]byte, 0, 14)
	for index := 1; index <= 14; index++ {
		path := fmt.Sprintf("product/traceability/fixtures/behavior_characterization_agent_batch_%02d.jsonl", index)
		content, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		result = append(result, content)
	}
	return result
}

func readBehaviorBatch16File(t *testing.T, path string) []byte {
	t.Helper()
	if path == behaviorBatch16Fixture {
		if override := strings.TrimSpace(os.Getenv("ORQUESTA_BATCH16_FIXTURE")); override != "" {
			path = override
		}
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func mutateBehaviorBatch16RoadmapStatus(t *testing.T, raw []byte, status string) []byte {
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
		if !entryOK || entry["id"] != "UI-03" {
			continue
		}
		entry["status"] = status
		changed, err := json.Marshal(roadmap)
		if err != nil {
			t.Fatal(err)
		}
		return changed
	}
	t.Fatal("UI-03 ausente")
	return nil
}
