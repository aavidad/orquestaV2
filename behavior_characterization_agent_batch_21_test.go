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

const behaviorBatch21Fixture = "product/traceability/fixtures/behavior_characterization_agent_batch_21.jsonl"

func TestBehaviorCharacterizationAgentBatch21(t *testing.T) {
	fixture := readBehaviorBatch21File(t, behaviorBatch21Fixture)
	ledger := readBehaviorBatch21File(t, "product/traceability/task_entries.jsonl")
	roadmap := readBehaviorBatch21File(t, "product/roadmap.json")
	if err := validateBehaviorBatch21(fixture, ledger, roadmap, behaviorBatch21PreviousFixtures(t)); err != nil {
		t.Fatal(err)
	}
}

func TestBehaviorCharacterizationAgentBatch21RejectsMutations(t *testing.T) {
	fixture := readBehaviorBatch21File(t, behaviorBatch21Fixture)
	ledger := readBehaviorBatch21File(t, "product/traceability/task_entries.jsonl")
	roadmap := readBehaviorBatch21File(t, "product/roadmap.json")
	previous := behaviorBatch21PreviousFixtures(t)
	mutations := []struct{ name, old, replacement string }{
		{"autoridad canónica", `"authority":"proposal_fixture_not_canonical_ledger"`, `"authority":"canonical_ledger"`},
		{"cambio canónico", `"canonical_state_change":false`, `"canonical_state_change":true`},
		{"crea trabajo", `"creates_work_item":false`, `"creates_work_item":true`},
		{"cierra capacidad", `"closes_capability":false`, `"closes_capability":true`},
		{"afirma acreditación", `"claims_accreditation":false`, `"claims_accreditation":true`},
		{"revisión aceptada", `"review_state":"bootstrap_first_review_pending_independent_counterreview"`, `"review_state":"accepted"`},
		{"capacidad incorrecta", `"capability_id":"STG-11"`, `"capability_id":"STG-10"`},
		{"referencia reutilizada", `"TASKENTRY-5276168c9504f2c4618ffe01"`, `"TASKENTRY-42d9dc522726fe09795ed722"`},
		{"fuentes solapadas independientes", `no son evidencia independiente`, `son evidencia independiente`},
		{"ancla semántica", `os.Open tras validar string`, `os.Open sin validar`},
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
			if err := validateBehaviorBatch21(changed, ledger, roadmap, previous); err == nil {
				t.Fatal("la mutación autoritativa, reutilizada o ambigua fue aceptada")
			}
		})
	}
	t.Run("LF final ausente", func(t *testing.T) {
		if err := validateBehaviorBatch21(bytes.TrimSuffix(fixture, []byte("\n")), ledger, roadmap, previous); err == nil {
			t.Fatal("el fixture sin LF final fue aceptado")
		}
	})
	t.Run("roadmap acredita UI-14 sin evidencia", func(t *testing.T) {
		changed := mutateBehaviorBatch21RoadmapStatus(t, roadmap, "accredited")
		if err := validateBehaviorBatch21(fixture, ledger, changed, previous); err == nil {
			t.Fatal("el estado roadmap mutado fue aceptado")
		}
	})
}

func TestBehaviorCharacterizationAgentBatch21QualifiesHistoricalClaims(t *testing.T) {
	records, err := decodeBehaviorBatchRecords(readBehaviorBatch21File(t, behaviorBatch21Fixture))
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

func behaviorBatch21PreviousFixtures(t *testing.T) [][]byte {
	t.Helper()
	result := make([][]byte, 0, 18)
	for index := 1; index <= 18; index++ {
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

func readBehaviorBatch21File(t *testing.T, path string) []byte {
	t.Helper()
	if path == behaviorBatch21Fixture {
		if override := strings.TrimSpace(os.Getenv("ORQUESTA_BATCH21_FIXTURE")); override != "" {
			path = override
		}
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func mutateBehaviorBatch21RoadmapStatus(t *testing.T, raw []byte, status string) []byte {
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
		if !entryOK || entry["id"] != "UI-14" {
			continue
		}
		entry["status"] = status
		changed, err := json.Marshal(roadmap)
		if err != nil {
			t.Fatal(err)
		}
		return changed
	}
	t.Fatal("UI-14 ausente")
	return nil
}
