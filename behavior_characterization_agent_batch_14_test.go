// Este contrato conserva una propuesta trazable; no crea trabajo, cierre, admisión ni estado canónico.
package orquesta_test

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

const behaviorBatch14Fixture = "product/traceability/fixtures/behavior_characterization_agent_batch_14.jsonl"

func TestBehaviorCharacterizationAgentBatch14(t *testing.T) {
	fixture := readBehaviorBatch14File(t, behaviorBatch14Fixture)
	ledger := readBehaviorBatch14File(t, "product/traceability/task_entries.jsonl")
	roadmap := readBehaviorBatch14File(t, "product/roadmap.json")
	if err := validateBehaviorBatch14(fixture, ledger, roadmap, behaviorBatch14PreviousFixtures(t)); err != nil {
		t.Fatal(err)
	}
}

func TestBehaviorCharacterizationAgentBatch14RejectsMutations(t *testing.T) {
	fixture := readBehaviorBatch14File(t, behaviorBatch14Fixture)
	ledger := readBehaviorBatch14File(t, "product/traceability/task_entries.jsonl")
	roadmap := readBehaviorBatch14File(t, "product/roadmap.json")
	previous := behaviorBatch14PreviousFixtures(t)
	mutations := []struct{ name, old, replacement string }{
		{"autoridad canónica", `"authority":"proposal_fixture_not_canonical_ledger"`, `"authority":"canonical_ledger"`},
		{"cambio canónico", `"canonical_state_change":false`, `"canonical_state_change":true`},
		{"crea trabajo", `"creates_work_item":false`, `"creates_work_item":true`},
		{"cierra capacidad", `"closes_capability":false`, `"closes_capability":true`},
		{"afirma acreditación", `"claims_accreditation":false`, `"claims_accreditation":true`},
		{"revisión aceptada", `"review_state":"bootstrap_first_review_pending_independent_counterreview"`, `"review_state":"accepted"`},
		{"capacidad incorrecta", `"capability_id":"GOV-18"`, `"capability_id":"GOV-17"`},
		{"referencia reutilizada", `"TASKENTRY-0b97e8cf27e312bbe5a462ff"`, `"TASKENTRY-76c5cb177737ae2b21014d61"`},
		{"fuentes solapadas independientes", `no son evidencia independiente`, `son evidencia independiente`},
		{"campo desconocido", `{"schema_version":1`, `{"unknown":true,"schema_version":1`},
		{"valor posterior", `}` + "\n", `} {}` + "\n"},
		{"representación no canónica", `{"schema_version":1`, ` {"schema_version":1`},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			changed := bytes.Replace(fixture, []byte(mutation.old), []byte(mutation.replacement), 1)
			if bytes.Equal(changed, fixture) {
				t.Fatal("la mutación no cambió el fixture")
			}
			if err := validateBehaviorBatch14(changed, ledger, roadmap, previous); err == nil {
				t.Fatal("la mutación autoritativa, reutilizada o ambigua fue aceptada")
			}
		})
	}
	t.Run("LF final ausente", func(t *testing.T) {
		if err := validateBehaviorBatch14(bytes.TrimSuffix(fixture, []byte("\n")), ledger, roadmap, previous); err == nil {
			t.Fatal("el fixture sin LF final fue aceptado")
		}
	})
	t.Run("roadmap acredita GOV-18 sin evidencia", func(t *testing.T) {
		changed := mutateBehaviorBatch14RoadmapStatus(t, roadmap, "accredited")
		if err := validateBehaviorBatch14(fixture, ledger, changed, previous); err == nil {
			t.Fatal("el estado roadmap mutado fue aceptado")
		}
	})
}

func TestBehaviorCharacterizationAgentBatch14QualifiesHistoricalClaims(t *testing.T) {
	records, err := decodeBehaviorBatchRecords(readBehaviorBatch14File(t, behaviorBatch14Fixture))
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

func behaviorBatch14PreviousFixtures(t *testing.T) [][]byte {
	t.Helper()
	result := make([][]byte, 0, 11)
	for index := 1; index <= 11; index++ {
		path := "product/traceability/fixtures/behavior_characterization_agent_batch_" + []string{"", "01", "02", "03", "04", "05", "06", "07", "08", "09", "10", "11"}[index] + ".jsonl"
		result = append(result, readBehaviorBatch14File(t, path))
	}
	return result
}

func readBehaviorBatch14File(t *testing.T, path string) []byte {
	t.Helper()
	if path == behaviorBatch14Fixture {
		if override := strings.TrimSpace(os.Getenv("ORQUESTA_BATCH14_FIXTURE")); override != "" {
			path = override
		}
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func mutateBehaviorBatch14RoadmapStatus(t *testing.T, raw []byte, status string) []byte {
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
		if !entryOK || entry["id"] != "GOV-18" {
			continue
		}
		entry["status"] = status
		changed, err := json.Marshal(roadmap)
		if err != nil {
			t.Fatal(err)
		}
		return changed
	}
	t.Fatal("GOV-18 ausente")
	return nil
}
