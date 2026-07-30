// Este contrato conserva una propuesta trazable; no crea trabajo, cierre, admisión ni estado canónico.
package orquesta_test

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

const behaviorBatch02Fixture = "product/traceability/fixtures/behavior_characterization_agent_batch_02.jsonl"

func TestBehaviorCharacterizationAgentBatch02(t *testing.T) {
	fixture := readBehaviorBatch02File(t, behaviorBatch02Fixture)
	ledger := readBehaviorBatch02File(t, "product/traceability/task_entries.jsonl")
	previous := readBehaviorBatch02File(t, behaviorBatchFixture)
	if err := validateBehaviorBatch02(fixture, ledger, previous); err != nil {
		t.Fatal(err)
	}
}

func TestBehaviorCharacterizationAgentBatch02RejectsMutations(t *testing.T) {
	fixture := readBehaviorBatch02File(t, behaviorBatch02Fixture)
	ledger := readBehaviorBatch02File(t, "product/traceability/task_entries.jsonl")
	previous := readBehaviorBatch02File(t, behaviorBatchFixture)
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
		{"capacidad ajena", `"capability_id":"AGT-01"`, `"capability_id":"AGT-12"`},
		{"referencia duplicada", `"TASKENTRY-b775431b02ac42e6e6abe7ae"`, `"TASKENTRY-d1e54d14ccd52a1241c38235"`},
		{"conducta duplicada", `"characterization_ref":"BEHAVIOR-AGENT-BATCH-02-CODEX-GOAL"`, `"characterization_ref":"BEHAVIOR-AGENT-BATCH-02-NEUTRAL-LAUNCH"`},
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
			if err := validateBehaviorBatch02(changed, ledger, previous); err == nil {
				t.Fatal("la mutación autoritativa, duplicada o ambigua fue aceptada")
			}
		})
	}
	semanticSwaps := []struct {
		name, left, right string
	}{
		{
			"mezcla grupos AGT-03",
			"ciclo_goal_codex_lanzar_y_observar",
			"supervision_y_recuperacion_codex",
		},
		{
			"mezcla grupos ORC-16",
			"parada_exacta_solicitada_y_confirmada",
			"aislamiento_concurrente_y_recuperacion_de_parada",
		},
	}
	for _, swap := range semanticSwaps {
		t.Run(swap.name, func(t *testing.T) {
			changed := swapBehaviorBatch02Evidence(t, fixture, swap.left, swap.right)
			if err := validateBehaviorBatch02(changed, ledger, previous); err == nil {
				t.Fatal("el intercambio de evidencias entre conductas fue aceptado")
			}
		})
	}
	t.Run("LF final ausente", func(t *testing.T) {
		changed := bytes.TrimSuffix(fixture, []byte("\n"))
		if err := validateBehaviorBatch02(changed, ledger, previous); err == nil {
			t.Fatal("el fixture sin LF final fue aceptado")
		}
	})
}

func TestBehaviorCharacterizationAgentBatch02QualifiesHistoricalClaims(t *testing.T) {
	records, err := decodeBehaviorBatchRecords(readBehaviorBatch02File(t, behaviorBatch02Fixture))
	if err != nil {
		t.Fatal(err)
	}
	for _, record := range records {
		for _, claim := range record.Worked {
			if !strings.Contains(claim, "declar") && !strings.Contains(claim, "document") &&
				!strings.Contains(claim, "propuest") {
				t.Errorf("afirmación histórica sin calificar: %q", claim)
			}
		}
	}
}

func readBehaviorBatch02File(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func swapBehaviorBatch02Evidence(t *testing.T, raw []byte, leftBehavior, rightBehavior string) []byte {
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
