// Estas utilidades validan la propuesta del lote 06 contra trazabilidad congelada, nunca contra el legado.
package orquesta_test

import (
	"fmt"
	"strings"
)

const behaviorBatch06ExpectedEntryCount = 20

type behaviorBatch06Group struct {
	capability string
	ref        string
	entryRefs  []string
	anchors    []string
}

func behaviorBatch06Groups() map[string]behaviorBatch06Group {
	return map[string]behaviorBatch06Group{
		"lease_y_timeout_sin_reloj_oculto": {
			capability: "ORC-12",
			ref:        "BEHAVIOR-AGENT-BATCH-06-LEASE-POLICY",
			entryRefs: []string{
				"TASKENTRY-d563352c3719db45f14a8ce5", "TASKENTRY-819d8930ce3eb1e865bc1754",
				"TASKENTRY-dd0a140b653eeaeaf7e256ca", "TASKENTRY-c9deea061d72d8d0eb09d0b4",
			},
			anchors: []string{
				"Crear microproyecto y contratos candidatos de leases/timeouts.",
				"AgentLeasePolicyV0, AgentHeartbeatReportV0",
				"tiempos llegan como input; no usar time.Now",
				"AgentTimeoutAssessmentV0 -> AgentLeaseExpiredV0 candidato",
			},
		},
		"identidad_fuerte_para_replay_y_cas": {
			capability: "ORC-12",
			ref:        "BEHAVIOR-AGENT-BATCH-06-STRONG-IDENTITY",
			entryRefs: []string{
				"TASKENTRY-e66b6339ef372cb5fd7f1423", "TASKENTRY-52f013f9990a82901bb28bd2",
				"TASKENTRY-73a81aae1f75e45faddabd33", "TASKENTRY-3308121c98cca467de2d0664",
			},
			anchors: []string{
				"identidad de `PhaseOpened` es por `event_id`, no por `phase_id`",
				"impedir lifecycle ambiguo por refs ya reflejadas",
				"gate de concurrencia, expiracion de lease y respuesta del director",
				"AgentStopConfirmed, DeliveryRegistered",
			},
		},
		"tick_y_supervision_idempotentes": {
			capability: "ORC-12",
			ref:        "BEHAVIOR-AGENT-BATCH-06-IDEMPOTENT-TICK",
			entryRefs: []string{
				"TASKENTRY-984c11b3240a9aca7072541b", "TASKENTRY-88ec53b5a2bd577fad8c69f3",
				"TASKENTRY-17f560d238228ea406b0b994", "TASKENTRY-d2d14f0a04e668d059e06f1f",
			},
			anchors: []string{
				"Fijar prioridad base del tick y dedupe por refs compactas.",
				"refs compactas para dedupe de progreso, preguntas y replan.",
				"`runs/supervise` idempotente sobre runs ya entregados",
				"Anti-churn idempotente",
			},
		},
		"control_activo_por_handle_y_orden": {
			capability: "ORC-16",
			ref:        "BEHAVIOR-AGENT-BATCH-06-ACTIVE-CONTROL",
			entryRefs: []string{
				"TASKENTRY-9223123582ed2769b583c445", "TASKENTRY-0ed8bbc316f3d8ec1e85983a",
				"TASKENTRY-02876d8ca78f7410262c5db4", "TASKENTRY-8565dbb60e878fc05d6a8744",
			},
			anchors: []string{
				"`Autogestión supervisada de agentes y resolución autónoma de bloqueos`",
				"gobernar agentes vivos sin que el usuario haga de puente manual",
				"CREATE TABLE IF NOT EXISTS runtime_handles (",
				"CREATE TABLE IF NOT EXISTS runtime_orders (",
			},
		},
		"stop_exacto_pendiente_hasta_confirmacion": {
			capability: "ORC-16",
			ref:        "BEHAVIOR-AGENT-BATCH-06-EXACT-STOP",
			entryRefs: []string{
				"TASKENTRY-8b849dc26a359ccf5c4fb31f", "TASKENTRY-b78702d6e52888dbd376f505",
				"TASKENTRY-19648d55a55b1e9cf8f41949", "TASKENTRY-e813dbe494914675ccae2fdb",
			},
			anchors: []string{
				"control_not_propagated_to_goal_backend",
				"APP-CODEX-STACK-045",
				"StopRuntimeAgentRequestV0, sin implementar parada real ni procesos.",
				"Alinear ACK tardio con `confirmed_stopped_agents`.",
			},
		},
	}
}

func validateBehaviorBatch06(fixture, ledger []byte, previous [][]byte) error {
	records, err := decodeBehaviorBatchRecords(fixture)
	if err != nil {
		return err
	}
	entries, err := decodeBehaviorBatchLedger(ledger)
	if err != nil {
		return err
	}
	groups := behaviorBatch06Groups()
	expected := make(map[string]bool, behaviorBatch06ExpectedEntryCount)
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
	if len(records) != 5 || len(groups) != 5 || len(expected) != behaviorBatch06ExpectedEntryCount {
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
			return fmt.Errorf("entrada reutilizada de los lotes 01/02/03/04/05: %s", ref)
		}
	}
	seenCharacterizations := make(map[string]struct{}, len(records))
	allEvidence := make([]behaviorBatchEvidence, 0, behaviorBatch06ExpectedEntryCount)
	for _, record := range records {
		group, ok := groups[record.Behavior]
		_, duplicateCurrent := seenCharacterizations[record.Ref]
		_, duplicatePrevious := previousCharacterizationRefs[record.Ref]
		if !ok || duplicateCurrent || duplicatePrevious || !behaviorBatch06HeaderValid(record, group) {
			return fmt.Errorf("cabecera o grupo inválido: %s", record.Ref)
		}
		if len(record.EntryRefs) != 4 || len(record.Evidence) != 4 ||
			!behaviorBatchFieldsPresent(record) {
			return fmt.Errorf("registro incompleto: %s", record.Behavior)
		}
		text := behaviorBatch06RecordText(record)
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
	if !behaviorBatch06RangesDisjoint(allEvidence) {
		return fmt.Errorf("el lote contiene rangos de procedencia solapados")
	}
	return nil
}

func behaviorBatch06HeaderValid(record behaviorBatchRecord, group behaviorBatch06Group) bool {
	return record.SchemaVersion == 1 && record.Ref == group.ref &&
		record.CapabilityID == group.capability &&
		record.Authority == "proposal_fixture_not_canonical_ledger" &&
		record.ReviewState == "bootstrap_first_review_pending_independent_counterreview" &&
		record.Disposition == "not_evaluated" && !record.CanonicalChange && !record.CreatesWork &&
		!record.ClosesCapability && !record.ClaimsAccreditation
}

func behaviorBatch06RecordText(record behaviorBatchRecord) string {
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

func behaviorBatch06RangesDisjoint(evidence []behaviorBatchEvidence) bool {
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
