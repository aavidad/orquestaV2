// Este contrato conserva una propuesta trazable; no crea trabajo, cierre, admisión ni estado canónico.
package orquesta_test

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

const behaviorBatchFixture = "product/traceability/fixtures/behavior_characterization_agent_batch_01.jsonl"

var behaviorBatchExpectedRefs = []string{
	"TASKENTRY-b90a2c4bfee38ab561bda3a4", "TASKENTRY-2bb9248a8767cb86ad848325",
	"TASKENTRY-e4dc5d0881aebc7d56e6be2a", "TASKENTRY-32a3d5f8a0d152431bc2efaf",
	"TASKENTRY-368dca75bd6f12cc45332a11", "TASKENTRY-24a13adcff4cd6d7932667d4",
	"TASKENTRY-05c4c9d0fae15ddf95a958e3", "TASKENTRY-603aa5279c1153c556f41b49",
	"TASKENTRY-045b74646d7a08af374b818e", "TASKENTRY-59475e7851502336e1b3ea00",
	"TASKENTRY-e435181f20e2e1904c216850", "TASKENTRY-33250dc889f217b1eb8d4512",
	"TASKENTRY-a5fec272bc74c654b3934507", "TASKENTRY-ab0a607abb6bae3d1afc7613",
	"TASKENTRY-232e141703b6871788a9dff9", "TASKENTRY-9118aaf7b1d3bf8cc3bdb186",
	"TASKENTRY-2007a03bf6266f2980a6468f", "TASKENTRY-c3aa69f4173f36833349caa8",
	"TASKENTRY-92988ea87907c4cdca287e7f", "TASKENTRY-78ec3da94694b7a4c2202fb8",
	"TASKENTRY-38833fea671dd24844ba5eee", "TASKENTRY-453e427d13159cc7b8dea5db",
	"TASKENTRY-360026590e4664dc753b2674", "TASKENTRY-a900fc0aa5057a595a9a5e1c",
}

func TestBehaviorCharacterizationAgentBatch01(t *testing.T) {
	fixture := readBehaviorBatchFile(t, behaviorBatchFixture)
	ledger := readBehaviorBatchFile(t, "product/traceability/task_entries.jsonl")
	if err := validateBehaviorBatch(fixture, ledger); err != nil {
		t.Fatal(err)
	}
}

func TestBehaviorCharacterizationAgentBatch01RejectsAuthorityMutations(t *testing.T) {
	fixture := readBehaviorBatchFile(t, behaviorBatchFixture)
	ledger := readBehaviorBatchFile(t, "product/traceability/task_entries.jsonl")
	mutations := []struct {
		name, old, replacement string
	}{
		{"cambio canónico", `"canonical_state_change":false`, `"canonical_state_change":true`},
		{"crea trabajo", `"creates_work_item":false`, `"creates_work_item":true`},
		{"cierra capacidad", `"closes_capability":false`, `"closes_capability":true`},
		{"afirma acreditación", `"claims_accreditation":false`, `"claims_accreditation":true`},
		{"revisión aceptada", `"review_state":"bootstrap_first_review_pending_independent_counterreview"`, `"review_state":"accepted"`},
		{"disposición evaluada", `"disposition":"not_evaluated"`, `"disposition":"accepted"`},
		{"referencia de conducta ajena", `"characterization_ref":"BEHAVIOR-AGENT-BATCH-01-LIVE-QUOTA"`, `"characterization_ref":"BEHAVIOR-AGENT-BATCH-01-UNKNOWN"`},
		{"usuario vacío", `"users":["operador","director"`, `"users":["","director"`},
		{"usuario duplicado", `"users":["operador","director"`, `"users":["operador","operador"`},
		{"campo desconocido", `{"schema_version":1`, `{"unknown":true,"schema_version":1`},
		{"clave duplicada", `{"schema_version":1`, `{"schema_version":2,"schema_version":1`},
		{"valor posterior", `}` + "\n", `} {}` + "\n"},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			changed := bytes.Replace(fixture, []byte(mutation.old), []byte(mutation.replacement), 1)
			if bytes.Equal(changed, fixture) {
				t.Fatal("la mutación no cambió el fixture")
			}
			if err := validateBehaviorBatch(changed, ledger); err == nil {
				t.Fatal("la mutación autoritativa o ambigua fue aceptada")
			}
		})
	}
}

func TestBehaviorCharacterizationAgentBatch01QualifiesHistoricalClaims(t *testing.T) {
	records, err := decodeBehaviorBatchRecords(readBehaviorBatchFile(t, behaviorBatchFixture))
	if err != nil {
		t.Fatal(err)
	}
	checkpointLinked := false
	for _, record := range records {
		for _, claim := range record.Worked {
			if !strings.Contains(claim, "declar") && !strings.Contains(claim, "descri") &&
				!strings.Contains(claim, "document") && !strings.Contains(claim, "propuest") &&
				!strings.Contains(claim, "sin composición real demostrada") {
				t.Errorf("afirmación histórica sin calificar: %q", claim)
			}
		}
		if record.Behavior == "decision_durable_previa_al_lanzamiento" {
			for _, output := range record.Outputs {
				for _, legacyName := range []string{"RequestCapacity", "CapacityRequested", "CapacityDecided", "RegisterCapacityDecision"} {
					if strings.Contains(output, legacyName) {
						t.Errorf("salida acoplada al nombre histórico %q", legacyName)
					}
				}
			}
		}
		if record.Behavior == "disponibilidad_y_cuota_vivas" &&
			strings.Contains(strings.Join(record.Uncertainties, " "), "punto de control en adaptadores distintos") {
			checkpointLinked = true
		}
	}
	if !checkpointLinked {
		t.Error("se perdió el enlace pendiente al soporte de punto de control")
	}
}

func readBehaviorBatchFile(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return content
}
