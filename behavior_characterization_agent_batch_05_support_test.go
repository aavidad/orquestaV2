// Estas utilidades validan la propuesta del lote 05 contra trazabilidad congelada, nunca contra el legado.
package orquesta_test

import (
	"fmt"
	"strings"
)

const behaviorBatch05ExpectedEntryCount = 20

type behaviorBatch05Group struct {
	capability string
	ref        string
	entryRefs  []string
	anchors    []string
}

func behaviorBatch05Groups() map[string]behaviorBatch05Group {
	return map[string]behaviorBatch05Group{
		"composicion_codex_opt_in_y_compatibilidad": {
			capability: "AGT-03",
			ref:        "BEHAVIOR-AGENT-BATCH-05-CODEX-OPT-IN",
			entryRefs: []string{
				"TASKENTRY-4ca8a7b17d4cd9e364bafdd9", "TASKENTRY-f7b3dda3a806aeb79ae7059e",
				"TASKENTRY-71b7b09c07599e927c1665e1", "TASKENTRY-165a9aeb913fc234fd9b5a23",
			},
			anchors: []string{
				"Composicion Codex opt-in, closure source y supervisor",
				"Wiring opt-in, ACK/evidencia, tests requeridos o smoke fake acotado",
				"`ConfigV0` con puertos/stores/runtime/capacidad inyectados",
				"recuperacion manual opt-in",
			},
		},
		"codex_real_aislamiento_y_apagado": {
			capability: "AGT-03",
			ref:        "BEHAVIOR-AGENT-BATCH-05-CODEX-ISOLATION",
			entryRefs: []string{
				"TASKENTRY-361fe2b61e301f13dad513e0", "TASKENTRY-096a0e3edacb456535372190",
				"TASKENTRY-b4992fee8759bb4118fe4634", "TASKENTRY-ed2fa14c1fdea646467dd98b",
			},
			anchors: []string{
				"artefacto materializado",
				"app_server_tmux aislado",
				"`runtime_work_dir` contiene packet, prompt, ACK esperado, stdout, stderr",
				"máquina de estados explícita del backend",
			},
		},
		"escalado_de_modelo_por_salud_y_evidencia": {
			capability: "ORC-27",
			ref:        "BEHAVIOR-AGENT-BATCH-05-MODEL-ESCALATION",
			entryRefs: []string{
				"TASKENTRY-18aeca163df745651163d385", "TASKENTRY-b1aacc42dda80a0a200df426",
				"TASKENTRY-6e8eccb781e2e6003161e810", "TASKENTRY-f509120173342a02934d733c",
			},
			anchors: []string{
				"Autobloquear agentes degradados por estado operativo",
				"prompts interactivos de cuota/rate-limit",
				"cascada-modelos-medida",
				"ModelEscalationPolicyV0",
			},
		},
		"normalizacion_y_contrato_runtime_neutrales": {
			capability: "AGT-01",
			ref:        "BEHAVIOR-AGENT-BATCH-05-NEUTRAL-NORMALIZER",
			entryRefs: []string{
				"TASKENTRY-2c1dd6a652e1fd489c82affc", "TASKENTRY-dd1baf5a13d1e4f6693cc9b1",
				"TASKENTRY-6e939895b962f2a200adae63", "TASKENTRY-4e7082a172a0d252a92d242a",
			},
			anchors: []string{
				"normalizeClaudeGoalEvidenceRefsJSONValueV0",
				"normalizador por proveedor",
				"RuntimeAdapter",
				"No provider fork de una primitiva neutral.",
			},
		},
		"observacion_recuperacion_y_paridad_de_runtime": {
			capability: "AGT-01",
			ref:        "BEHAVIOR-AGENT-BATCH-05-RUNTIME-OBSERVATION",
			entryRefs: []string{
				"TASKENTRY-0e2f469492d8c20194fed8d8", "TASKENTRY-b7d4a16179f2420558d76498",
				"TASKENTRY-086e55a9553458a9a91042df", "TASKENTRY-ba1d022efd7fb8aeb88eaced",
			},
			anchors: []string{
				"status estructurado por perfil",
				"`sync_status` puntual por `status_path`",
				"convertir `tick` de señal informativa a parte de un ciclo autónomo",
				"Matriz E2E común",
			},
		},
	}
}

func validateBehaviorBatch05(fixture, ledger []byte, previous [][]byte) error {
	records, err := decodeBehaviorBatchRecords(fixture)
	if err != nil {
		return err
	}
	entries, err := decodeBehaviorBatchLedger(ledger)
	if err != nil {
		return err
	}
	groups := behaviorBatch05Groups()
	expected := make(map[string]bool, behaviorBatch05ExpectedEntryCount)
	for _, group := range groups {
		if len(group.entryRefs) != 4 || len(group.anchors) != 4 {
			return fmt.Errorf("grupo esperado sin cuatro entradas y anclas: %s", group.ref)
		}
		for _, ref := range group.entryRefs {
			if _, duplicate := expected[ref]; duplicate {
				return fmt.Errorf("entrada esperada duplicada: %s", ref)
			}
			expected[ref] = false
		}
	}
	if len(records) != 5 || len(groups) != 5 || len(expected) != behaviorBatch05ExpectedEntryCount {
		return fmt.Errorf("tamaño de lote inválido: conductas=%d entradas=%d", len(records), len(expected))
	}
	previousEntryRefs := make(map[string]struct{})
	previousCharacterizationRefs := make(map[string]struct{})
	for _, raw := range previous {
		previousRecords, decodeErr := decodeBehaviorBatchRecords(raw)
		if decodeErr != nil {
			return decodeErr
		}
		for _, record := range previousRecords {
			previousCharacterizationRefs[record.Ref] = struct{}{}
			for _, ref := range record.EntryRefs {
				previousEntryRefs[ref] = struct{}{}
			}
		}
	}
	for ref := range expected {
		if _, reused := previousEntryRefs[ref]; reused {
			return fmt.Errorf("entrada reutilizada de los lotes 01/02/03/04: %s", ref)
		}
	}
	seenCharacterizations := make(map[string]struct{}, len(records))
	allEvidence := make([]behaviorBatchEvidence, 0, behaviorBatch05ExpectedEntryCount)
	for _, record := range records {
		group, ok := groups[record.Behavior]
		_, duplicateCurrent := seenCharacterizations[record.Ref]
		_, duplicatePrevious := previousCharacterizationRefs[record.Ref]
		if !ok || duplicateCurrent || duplicatePrevious || !behaviorBatch05HeaderValid(record, group) {
			return fmt.Errorf("cabecera o grupo inválido: %s", record.Ref)
		}
		if len(record.EntryRefs) != 4 || len(record.Evidence) != 4 ||
			!behaviorBatchFieldsPresent(record) {
			return fmt.Errorf("registro incompleto: %s", record.Behavior)
		}
		text := behaviorBatch05RecordText(record)
		for _, anchor := range group.anchors {
			if !strings.Contains(text, anchor) {
				return fmt.Errorf("ancla literal ausente en %s: %q", record.Behavior, anchor)
			}
		}
		for index, evidence := range record.Evidence {
			entry, exists := entries[evidence.EntryRef]
			_, reused := previousEntryRefs[evidence.EntryRef]
			seen, expectedRef := expected[evidence.EntryRef]
			if record.EntryRefs[index] != group.entryRefs[index] ||
				evidence.EntryRef != group.entryRefs[index] || !expectedRef || seen || reused ||
				!exists || !behaviorBatchEvidenceMatches(record.CapabilityID, evidence, entry) {
				return fmt.Errorf("procedencia incoherente: %s", evidence.EntryRef)
			}
			expected[evidence.EntryRef] = true
			allEvidence = append(allEvidence, evidence)
		}
		seenCharacterizations[record.Ref] = struct{}{}
		delete(groups, record.Behavior)
	}
	for ref, seen := range expected {
		if !seen {
			return fmt.Errorf("falta %s", ref)
		}
	}
	if len(groups) != 0 {
		return fmt.Errorf("faltan grupos: %v", groups)
	}
	if !behaviorBatch05RangesDisjoint(allEvidence) {
		return fmt.Errorf("el lote contiene rangos de procedencia solapados")
	}
	return nil
}

func behaviorBatch05HeaderValid(record behaviorBatchRecord, group behaviorBatch05Group) bool {
	return record.SchemaVersion == 1 && record.Ref == group.ref &&
		record.CapabilityID == group.capability &&
		record.Authority == "proposal_fixture_not_canonical_ledger" &&
		record.ReviewState == "bootstrap_first_review_pending_independent_counterreview" &&
		record.Disposition == "not_evaluated" && !record.CanonicalChange && !record.CreatesWork &&
		!record.ClosesCapability && !record.ClaimsAccreditation
}

func behaviorBatch05RecordText(record behaviorBatchRecord) string {
	values := []string{record.Problem, record.DecisionAuthority}
	groups := [][]string{
		record.Users, record.Inputs, record.Outputs, record.StateRead, record.StateWritten,
		record.Permissions.Permissions, record.Permissions.Secrets, record.Permissions.Effects,
		record.Recovery.Failure, record.Recovery.Retry, record.Recovery.Concurrency, record.Recovery.Restart,
		record.Worked, record.Failed, record.Preserve, record.Avoid, record.Uncertainties, record.Attempts,
	}
	for _, group := range groups {
		values = append(values, group...)
	}
	return strings.Join(values, "\n")
}

func behaviorBatch05RangesDisjoint(evidence []behaviorBatchEvidence) bool {
	for left := 0; left < len(evidence); left++ {
		for right := left + 1; right < len(evidence); right++ {
			if evidence[left].SourceRef == evidence[right].SourceRef &&
				evidence[left].FirstLine <= evidence[right].LastLine &&
				evidence[right].FirstLine <= evidence[left].LastLine {
				return false
			}
		}
	}
	return true
}
