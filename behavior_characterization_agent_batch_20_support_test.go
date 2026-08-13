// Estas utilidades validan el lote 20 contra fuentes canónicas V2; nunca ejecutan legacy.
package orquesta_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
)

type behaviorBatch20Entry struct {
	behaviorBatchTaskEntry
	CapabilityDecision  string `json:"capability_decision"`
	Disposition         string `json:"disposition"`
	SemanticReviewState string `json:"semantic_review_state"`
}

type behaviorBatch20Group struct {
	ref, capability string
	entries         []string
}

func behaviorBatch20Groups() map[string]behaviorBatch20Group {
	return map[string]behaviorBatch20Group{
		"perimetro_de_ejecucion_cerrado_antes_del_launch":   {"BEHAVIOR-AGENT-BATCH-20-CLOSED-EXECUTION-PERIMETER", "EVD-13", []string{"TASKENTRY-d9b21766020e25cd53aa64d2", "TASKENTRY-5719f92d368df695ee4f9bf0", "TASKENTRY-81489d2bc1099ba5db3d6e07", "TASKENTRY-caaa11e98f857b6d9483c752"}},
		"documentacion_navegable_sin_contradecir_runtime":   {"BEHAVIOR-AGENT-BATCH-20-DOCS-AS-VERIFIED-PROJECTION", "STG-12", []string{"TASKENTRY-18691be78ff16dd5932c7377", "TASKENTRY-32fde759e0fff6b1249ff4f5", "TASKENTRY-c75204c61032a5e61e911356", "TASKENTRY-c5dfff876fe887bc45eb8611"}},
		"gate_ci_reproducible_de_formato_tests_y_seguridad": {"BEHAVIOR-AGENT-BATCH-20-REPRODUCIBLE-CI-GATE", "APP-12", []string{"TASKENTRY-ca772539446e0650e897750d", "TASKENTRY-bb999733c71c290c053da5da", "TASKENTRY-f9e417bb6eec8015991a0952", "TASKENTRY-05cc98b297b81eadb387c765"}},
		"web_admin_fina_sobre_proyecciones_publicas":        {"BEHAVIOR-AGENT-BATCH-20-THIN-WEB-ADMIN", "UI-04", []string{"TASKENTRY-0c1da6c3dcdee5f1a326eb83", "TASKENTRY-62cc4c1b3a96687a6896536a", "TASKENTRY-748465aa04dfafc515bfc8c6", "TASKENTRY-373b5c8cb44c9df6ebfb0515"}},
		"proyeccion_derivada_unica_sin_poder_de_cierre":     {"BEHAVIOR-AGENT-BATCH-20-DERIVED-STATE-NO-CLOSURE", "GOV-06", []string{"TASKENTRY-0fc25a7e13a3a370dde0fb3b", "TASKENTRY-139425daaf753c8c09b8b464", "TASKENTRY-2553c208306b48f2284b93b9", "TASKENTRY-f0645f095e43ffc1bf4841f5"}},
	}
}

func validateBehaviorBatch20(fixture, ledger, roadmap []byte, previous [][]byte) error {
	records, err := decodeBehaviorBatchRecords(fixture)
	if err != nil {
		return err
	}
	entries, err := decodeBehaviorBatch20Ledger(ledger)
	if err != nil {
		return err
	}
	if err := validateBehaviorBatch20Roadmap(roadmap); err != nil {
		return err
	}
	groups := behaviorBatch20Groups()
	priorEntries, priorRefs := map[string]bool{}, map[string]bool{}
	for _, raw := range previous {
		prior, e := decodeBehaviorBatchRecords(raw)
		if e != nil {
			return e
		}
		for _, record := range prior {
			priorRefs[record.Ref] = true
			for _, ref := range record.EntryRefs {
				priorEntries[ref] = true
			}
		}
	}
	seenEntries, seenRefs := map[string]bool{}, map[string]bool{}
	var evidence []behaviorBatchEvidence
	if len(records) != 5 {
		return fmt.Errorf("conductas=%d", len(records))
	}
	for _, record := range records {
		group, ok := groups[record.Behavior]
		if !ok || seenRefs[record.Ref] || priorRefs[record.Ref] || !behaviorBatchHeaderValid(record, group.capability, group.ref) || !behaviorBatchFieldsPresent(record) || len(record.EntryRefs) != 4 || len(record.Evidence) != 4 {
			return fmt.Errorf("registro inválido: %s", record.Ref)
		}
		for i, ev := range record.Evidence {
			entry, exists := entries[ev.EntryRef]
			if !exists || ev.EntryRef != group.entries[i] || record.EntryRefs[i] != group.entries[i] || seenEntries[ev.EntryRef] || priorEntries[ev.EntryRef] || entry.CapabilityDecision != "accept" || entry.SemanticReviewState != "reviewed" || (entry.Disposition != "accepted_pending_reimplementation" && entry.Disposition != "historical_superseded_by_capability") || !behaviorBatchEvidenceMatches(group.capability, ev, entry.behaviorBatchTaskEntry) {
				return fmt.Errorf("procedencia inválida: %s", ev.EntryRef)
			}
			seenEntries[ev.EntryRef] = true
			evidence = append(evidence, ev)
		}
		seenRefs[record.Ref] = true
		delete(groups, record.Behavior)
	}
	if len(groups) != 0 || len(seenEntries) != 20 || !behaviorBatch20RangesDisjoint(evidence) {
		return fmt.Errorf("cobertura o rangos inválidos")
	}
	return nil
}

func decodeBehaviorBatch20Ledger(raw []byte) (map[string]behaviorBatch20Entry, error) {
	result := map[string]behaviorBatch20Entry{}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		var entry behaviorBatch20Entry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, err
		}
		result[entry.EntryRef] = entry
	}
	return result, scanner.Err()
}

func behaviorBatch20RangesDisjoint(values []behaviorBatchEvidence) bool {
	for i := range values {
		for j := i + 1; j < len(values); j++ {
			if values[i].SourceRef == values[j].SourceRef && values[i].FirstLine <= values[j].LastLine && values[j].FirstLine <= values[i].LastLine {
				return false
			}
		}
	}
	return true
}

func validateBehaviorBatch20Roadmap(raw []byte) error {
	type cap struct {
		ID                  string   `json:"id"`
		Title               string   `json:"title"`
		Decision            string   `json:"decision"`
		Kind                string   `json:"kind"`
		OwnerContext        string   `json:"owner_context"`
		Status              string   `json:"status"`
		AcceptanceContracts []string `json:"acceptance_contracts"`
		EvidenceRefs        []string `json:"evidence_refs"`
	}
	var root struct {
		Capabilities []cap `json:"capability_entries"`
	}
	if err := json.Unmarshal(raw, &root); err != nil {
		return err
	}
	want := map[string]cap{
		"EVD-13": {ID: "EVD-13", Title: "Sandbox non-root, permisos/egress, sin socket Docker, metadata, HOME/producción ni mounts host no autorizados", Decision: "accept", Kind: "evidence", OwnerContext: "test_attestor", Status: "accredited", AcceptanceContracts: []string{"AC-V17-TEST-ATTESTOR"}, EvidenceRefs: make([]string, 3)},
		"STG-12": {ID: "STG-12", Title: "Documentación para usuario, operación y mantenimiento", Decision: "accept", Kind: "stage_template", OwnerContext: "tools_skills_sdk", Status: "declared", AcceptanceContracts: []string{"AC-V26-TOOLS-SKILLS-SDK"}},
		"APP-12": {ID: "APP-12", Title: "CI, formato, lint, seguridad y dependencias reproducibles", Decision: "accept", Kind: "generated_app_profile", OwnerContext: "generated_apps", Status: "declared", AcceptanceContracts: []string{"AC-V33-GENERATED-APPS"}},
		"UI-04":  {ID: "UI-04", Title: "Web admin: Goals, fase actual/historial, trabajo, agentes, artefactos, decisiones y config", Decision: "accept", Kind: "interface", OwnerContext: "web_admin", Status: "declared", AcceptanceContracts: []string{"AC-V24-WEB-ADMIN"}},
		"GOV-06": {ID: "GOV-06", Title: "Estado derivado/proyecciones sin poder de cierre", Decision: "accept", Kind: "governance", OwnerContext: "atomic_state_outbox", Status: "accredited", AcceptanceContracts: []string{"AC-V06-ATOMIC-STATE-OUTBOX"}, EvidenceRefs: make([]string, 3)},
	}
	for _, got := range root.Capabilities {
		expected, ok := want[got.ID]
		if !ok {
			continue
		}
		if got.Title != expected.Title || got.Decision != expected.Decision || got.Kind != expected.Kind || got.OwnerContext != expected.OwnerContext || got.Status != expected.Status || len(got.AcceptanceContracts) != 1 || got.AcceptanceContracts[0] != expected.AcceptanceContracts[0] || len(got.EvidenceRefs) != len(expected.EvidenceRefs) {
			return fmt.Errorf("roadmap drift: %s", got.ID)
		}
		delete(want, got.ID)
	}
	if len(want) != 0 {
		return fmt.Errorf("capabilities ausentes: %v", want)
	}
	return nil
}
