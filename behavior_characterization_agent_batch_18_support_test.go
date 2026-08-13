// Estas utilidades validan la propuesta del lote 18 contra trazabilidad y roadmap vigentes, nunca contra el runtime legado.
package orquesta_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

const behaviorBatch18ExpectedEntryCount = 20

type behaviorBatch18Group struct {
	ref        string
	capability string
	entryRefs  []string
	anchors    []string
}

type behaviorBatch18TaskEntry struct {
	behaviorBatchTaskEntry
	CapabilityDecision  string `json:"capability_decision"`
	SemanticReviewState string `json:"semantic_review_state"`
}

type behaviorBatch18Roadmap struct {
	Capabilities []struct {
		ID                  string   `json:"id"`
		Title               string   `json:"title"`
		Decision            string   `json:"decision"`
		Kind                string   `json:"kind"`
		OwnerContext        string   `json:"owner_context"`
		AcceptanceContracts []string `json:"acceptance_contracts"`
		Status              string   `json:"status"`
		EvidenceRefs        []string `json:"evidence_refs"`
	} `json:"capability_entries"`
}

func behaviorBatch18Groups() map[string]behaviorBatch18Group {
	return map[string]behaviorBatch18Group{
		"aplicacion_generada_con_fronteras_hexagonales_verificables": {
			ref:        "BEHAVIOR-AGENT-BATCH-18-HEXAGONAL-BOUNDARIES",
			capability: "APP-02",
			entryRefs:  []string{"TASKENTRY-42d9dc522726fe09795ed722", "TASKENTRY-8e7b292cd0d448d3d0b19860", "TASKENTRY-18f0fe92cc7f0ac76b24a8ef", "TASKENTRY-4fb707da5c4fdc777c6ce315"},
			anchors:    []string{"puertos pequeños", "política todavía incrustada", "writer o lifecycle paralelo", "no son evidencia independiente"},
		},
		"dashboard_operativo_derivado_sin_autoridad": {
			ref:        "BEHAVIOR-AGENT-BATCH-18-FACTUAL-DASHBOARD",
			capability: "UI-12",
			entryRefs:  []string{"TASKENTRY-ce627aeeff1452695cc1b663", "TASKENTRY-ed37b271ace39d77aabee08d", "TASKENTRY-9178407af9974b704dc13ddc", "TASKENTRY-3ee34bbe8806643621e5a1de"},
			anchors:    []string{"proyección read-only", "builder productivo", "dashboard como lifecycle alternativo", "no son evidencia independiente"},
		},
		"director_transferible_propuestas_motor_autoritativo": {
			ref:        "BEHAVIOR-AGENT-BATCH-18-TRANSFERABLE-DIRECTOR",
			capability: "GOV-09",
			entryRefs:  []string{"TASKENTRY-8bc1f57d9a98410de4228aca", "TASKENTRY-ab5dc46433032603bbc79fb5", "TASKENTRY-2bb7962b24e43e8efa6f51a6", "TASKENTRY-90fbe8192aa8069040f7b599"},
			anchors:    []string{"fencing rechaza", "loop residente", "scheduler por proveedor", "no son evidencia independiente"},
		},
		"ledger_capacidades_unico_por_revision_y_evidencia": {
			ref:        "BEHAVIOR-AGENT-BATCH-18-CAPABILITY-LEDGER",
			capability: "GOV-16",
			entryRefs:  []string{"TASKENTRY-b3df19b350477e4f25fa5ddb", "TASKENTRY-089c7c0a6f68dd6eae4ebbc9", "TASKENTRY-bd41a40bd7f7bca1716d7b59", "TASKENTRY-e08fa76b8adc7a433c207ef9"},
			anchors:    []string{"cerrado en Markdown", "cero filas importadas", "dos estados vigentes", "no demuestran una implementación productiva"},
		},
		"persistencia_intercambiable_por_puerto_sin_politica": {
			ref:        "BEHAVIOR-AGENT-BATCH-18-PERSISTENCE-PORT",
			capability: "OPS-09",
			entryRefs:  []string{"TASKENTRY-6c01558ffb6c33351a08fc23", "TASKENTRY-573e6f553f19f7afe98cab4a", "TASKENTRY-fc0458142cfb4892ad04eb4f", "TASKENTRY-f063593f8e16a716a0a3886f"},
			anchors:    []string{"fake en memoria", "detalle de conector en DTO", "writer, outbox o lifecycle paralelo", "no son evidencia independiente"},
		},
	}
}

func validateBehaviorBatch18(fixture, ledger, roadmap []byte, previous [][]byte) error {
	if err := validateBehaviorBatch18Roadmap(roadmap); err != nil {
		return err
	}
	records, err := decodeBehaviorBatchRecords(fixture)
	if err != nil {
		return err
	}
	entries, err := decodeBehaviorBatch18Ledger(ledger)
	if err != nil {
		return err
	}
	groups := behaviorBatch18Groups()
	expected := make(map[string]bool, behaviorBatch18ExpectedEntryCount)
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
	if len(records) != 5 || len(expected) != behaviorBatch18ExpectedEntryCount {
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
			return fmt.Errorf("entrada reutilizada de lotes 01-16: %s", ref)
		}
	}
	seenCharacterizations := make(map[string]struct{})
	allEvidence := make([]behaviorBatchEvidence, 0, behaviorBatch18ExpectedEntryCount)
	for _, record := range records {
		group, ok := groups[record.Behavior]
		_, duplicateCurrent := seenCharacterizations[record.Ref]
		_, duplicatePrevious := previousCharacterizationRefs[record.Ref]
		if !ok || duplicateCurrent || duplicatePrevious || !behaviorBatch18HeaderValid(record, group) {
			return fmt.Errorf("cabecera o grupo inválido: %s", record.Ref)
		}
		if len(record.EntryRefs) != 4 || len(record.Evidence) != 4 || !behaviorBatchFieldsPresent(record) {
			return fmt.Errorf("registro incompleto: %s", record.Behavior)
		}
		text := behaviorBatch18RecordText(record)
		for _, anchor := range group.anchors {
			if !strings.Contains(text, anchor) {
				return fmt.Errorf("ancla literal ausente en %s: %q", record.Behavior, anchor)
			}
		}
		for index, evidence := range record.Evidence {
			entry, exists := entries[evidence.EntryRef]
			seen, expectedRef := expected[evidence.EntryRef]
			_, reused := previousEntryRefs[evidence.EntryRef]
			if record.EntryRefs[index] != group.entryRefs[index] || evidence.EntryRef != group.entryRefs[index] || !expectedRef || seen || reused || !exists || !behaviorBatch18EvidenceMatches(evidence, entry) {
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
	if len(groups) != 0 || !behaviorBatch18RangesDisjoint(allEvidence) {
		return fmt.Errorf("faltan grupos o existen rangos solapados")
	}
	return nil
}

func decodeBehaviorBatch18Ledger(raw []byte) (map[string]behaviorBatch18TaskEntry, error) {
	result := make(map[string]behaviorBatch18TaskEntry)
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		var entry behaviorBatch18TaskEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, err
		}
		result[entry.EntryRef] = entry
	}
	return result, scanner.Err()
}

func behaviorBatch18EvidenceMatches(evidence behaviorBatchEvidence, entry behaviorBatch18TaskEntry) bool {
	return entry.CapabilityDecision == "accept" && entry.SemanticReviewState == "reviewed" && behaviorBatchEvidenceMatches(entry.CapabilityID, evidence, entry.behaviorBatchTaskEntry)
}

func behaviorBatch18HeaderValid(record behaviorBatchRecord, group behaviorBatch18Group) bool {
	return record.SchemaVersion == 1 && record.Ref == group.ref && record.CapabilityID == group.capability && record.Authority == "proposal_fixture_not_canonical_ledger" && record.ReviewState == "bootstrap_first_review_pending_independent_counterreview" && record.Disposition == "not_evaluated" && !record.CanonicalChange && !record.CreatesWork && !record.ClosesCapability && !record.ClaimsAccreditation
}

func behaviorBatch18RecordText(record behaviorBatchRecord) string {
	values := []string{record.Problem, record.DecisionAuthority}
	groups := [][]string{record.Users, record.Inputs, record.Outputs, record.StateRead, record.StateWritten, record.Permissions.Permissions, record.Permissions.Secrets, record.Permissions.Effects, record.Recovery.Failure, record.Recovery.Retry, record.Recovery.Concurrency, record.Recovery.Restart, record.Worked, record.Failed, record.Preserve, record.Avoid, record.Uncertainties, record.Attempts}
	for _, group := range groups {
		values = append(values, group...)
	}
	return strings.Join(values, "\n")
}

func behaviorBatch18RangesDisjoint(evidence []behaviorBatchEvidence) bool {
	for left := 0; left < len(evidence); left++ {
		for right := left + 1; right < len(evidence); right++ {
			if evidence[left].SourceRef == evidence[right].SourceRef && evidence[left].FirstLine <= evidence[right].LastLine && evidence[right].FirstLine <= evidence[left].LastLine {
				return false
			}
		}
	}
	return true
}

func validateBehaviorBatch18Roadmap(raw []byte) error {
	var roadmap behaviorBatch18Roadmap
	if err := json.Unmarshal(raw, &roadmap); err != nil {
		return err
	}
	type expectation struct {
		title, kind, owner, status, acceptance string
		evidence                               []string
	}
	expected := map[string]expectation{
		"APP-02": {"Arquitectura hexagonal pura: dominio/aplicación/puertos/adaptadores/composición", "generated_app_profile", "generated_apps", "declared", "AC-V33-GENERATED-APPS", nil},
		"UI-12":  {"Dashboard de Goals, fase actual/historial, work, agentes, cuotas, reviews y efectos", "interface", "web_admin", "declared", "AC-V24-WEB-ADMIN", nil},
		"GOV-09": {"Rol `director` transferible entre Hermes, Codex, otro agente u operador", "governance", "director_lease", "accredited", "AC-V12-DIRECTOR-LEASE", []string{"acceptance/v12_director_lease_test.go", "acceptance/fixtures/v12_director_lease.json", "product/evidence/v12_director_lease.json"}},
		"GOV-16": {"Ledger de capacidades: declarada, implementada, cableada, ejercida, acreditada", "governance", "canonical_ledgers", "accredited", "AC-V03-CANONICAL-LEDGERS", []string{"acceptance/v03_canonical_ledgers_test.go", "acceptance/fixtures/v03_canonical_ledgers.json", "product/evidence/v03_canonical_ledgers.json"}},
		"OPS-09": {"Persistencia elegida mediante puerto", "operations", "atomic_state_outbox", "accredited", "AC-V06-ATOMIC-STATE-OUTBOX", []string{"acceptance/v06_atomic_state_outbox_test.go", "acceptance/fixtures/v06_atomic_state_outbox.json", "product/evidence/v06_atomic_state_outbox.json"}},
	}
	for _, capability := range roadmap.Capabilities {
		want, ok := expected[capability.ID]
		if !ok {
			continue
		}
		if capability.Title != want.title || capability.Decision != "accept" || capability.Kind != want.kind || capability.OwnerContext != want.owner || capability.Status != want.status || len(capability.AcceptanceContracts) != 1 || capability.AcceptanceContracts[0] != want.acceptance || !equalBehaviorBatch18Strings(capability.EvidenceRefs, want.evidence) {
			return fmt.Errorf("estado roadmap %s inesperado", capability.ID)
		}
		delete(expected, capability.ID)
	}
	if len(expected) != 0 {
		return fmt.Errorf("capabilities ausentes del roadmap: %v", expected)
	}
	return nil
}

func equalBehaviorBatch18Strings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
