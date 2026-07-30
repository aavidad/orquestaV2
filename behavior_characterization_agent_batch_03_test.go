// Este contrato conserva una propuesta trazable; no crea trabajo, cierre, admisión ni estado canónico.
package orquesta_test

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

const behaviorBatch03Fixture = "product/traceability/fixtures/behavior_characterization_agent_batch_03.jsonl"

func TestBehaviorCharacterizationAgentBatch03(t *testing.T) {
	fixture := readBehaviorBatch03File(t, behaviorBatch03Fixture)
	ledger := readBehaviorBatch03File(t, "product/traceability/task_entries.jsonl")
	previous := [][]byte{
		readBehaviorBatch03File(t, behaviorBatchFixture),
		readBehaviorBatch03File(t, behaviorBatch02Fixture),
	}
	if err := validateBehaviorBatch03(fixture, ledger, previous); err != nil {
		t.Fatal(err)
	}
}

func TestBehaviorCharacterizationAgentBatch03RejectsMutations(t *testing.T) {
	fixture := readBehaviorBatch03File(t, behaviorBatch03Fixture)
	ledger := readBehaviorBatch03File(t, "product/traceability/task_entries.jsonl")
	previous := [][]byte{
		readBehaviorBatch03File(t, behaviorBatchFixture),
		readBehaviorBatch03File(t, behaviorBatch02Fixture),
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
		{"capacidad ajena", `"capability_id":"AGT-02"`, `"capability_id":"AGT-12"`},
		{"revisión atribuida a AGT-10", `"capability_id":"EVD-06"`, `"capability_id":"AGT-10"`},
		{"referencia de lote anterior", `"TASKENTRY-0e134e32b7e58e4891044af3"`, `"TASKENTRY-d1e54d14ccd52a1241c38235"`},
		{"quinta referencia en un grupo", `"task_entry_refs":["TASKENTRY-0e134e32b7e58e4891044af3"`, `"task_entry_refs":["TASKENTRY-0e134e32b7e58e4891044af3","TASKENTRY-0e134e32b7e58e4891044af3"`},
		{"solo tres referencias en un grupo", `"TASKENTRY-f8e0fbd36baf3b4959c0e2ce","TASKENTRY-7aa4321139f572605c05ee5b"],"problem"`, `"TASKENTRY-7aa4321139f572605c05ee5b"],"problem"`},
		{"contrato vacío", `"users":["operador"`, `"users":[""`},
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
			if err := validateBehaviorBatch03(changed, ledger, previous); err == nil {
				t.Fatal("la mutación autoritativa, duplicada o ambigua fue aceptada")
			}
		})
	}
	t.Run("intercambio entre grupos AGT-07", func(t *testing.T) {
		changed := swapBehaviorBatch03Evidence(t, fixture,
			"conector_ollama_y_gestion_de_modelos",
			"modelos_ollama_por_perfil_de_tarea")
		if err := validateBehaviorBatch03(changed, ledger, previous); err == nil {
			t.Fatal("el intercambio de evidencias entre conductas de la misma capacidad fue aceptado")
		}
	})
	t.Run("LF final ausente", func(t *testing.T) {
		changed := bytes.TrimSuffix(fixture, []byte("\n"))
		if err := validateBehaviorBatch03(changed, ledger, previous); err == nil {
			t.Fatal("el fixture sin LF final fue aceptado")
		}
	})
}

func TestBehaviorCharacterizationAgentBatch03QualifiesHistoricalClaims(t *testing.T) {
	records, err := decodeBehaviorBatchRecords(readBehaviorBatch03File(t, behaviorBatch03Fixture))
	if err != nil {
		t.Fatal(err)
	}
	qualifiers := []string{"declar", "document", "propuest", "descrit", "registr", "inform",
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

func readBehaviorBatch03File(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func swapBehaviorBatch03Evidence(t *testing.T, raw []byte, leftBehavior, rightBehavior string) []byte {
	t.Helper()
	records, err := decodeBehaviorBatchRecords(raw)
	if err != nil {
		t.Fatal(err)
	}
	left, right := -1, -1
	for index := range records {
		switch records[index].Behavior {
		case leftBehavior:
			left = index
		case rightBehavior:
			right = index
		}
	}
	if left < 0 || right < 0 || len(records[left].EntryRefs) == 0 || len(records[right].EntryRefs) == 0 ||
		len(records[left].Evidence) == 0 || len(records[right].Evidence) == 0 {
		t.Fatal("no se encontraron dos evidencias intercambiables")
	}
	records[left].EntryRefs[0], records[right].EntryRefs[0] =
		records[right].EntryRefs[0], records[left].EntryRefs[0]
	records[left].Evidence[0], records[right].Evidence[0] =
		records[right].Evidence[0], records[left].Evidence[0]

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
