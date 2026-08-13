// Estas utilidades validan el lote 35 contra ledger y roadmap; nunca ejecutan legacy.
package orquesta_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
)

type behaviorBatch35Entry struct {
	behaviorBatchTaskEntry
	CapabilityDecision  string `json:"capability_decision"`
	Disposition         string `json:"disposition"`
	SemanticReviewState string `json:"semantic_review_state"`
}

type behaviorBatch35Group struct {
	ref, capability string
	entries         []string
}

func behaviorBatch35Groups() map[string]behaviorBatch35Group {
	return map[string]behaviorBatch35Group{
		"ayuda_contextual_glosario_ejemplos_y_explicacion":      {"BEHAVIOR-AGENT-BATCH-35-CONTEXTUAL-HELP-EXPLAIN", "WIZ-17", []string{"TASKENTRY-3917736c004c97117565a75e", "TASKENTRY-7c239f35bf37bd9fcdf46e4f"}},
		"wizard_determinista_sin_llm_con_rag_opcional_grounded": {"BEHAVIOR-AGENT-BATCH-35-DETERMINISTIC-WIZARD-BOT", "WIZ-19", []string{"TASKENTRY-9a7dfcfddd17e84170cd3ea4"}},
		"reabrir_decisiones_dependientes_al_cambiar_respuesta":  {"BEHAVIOR-AGENT-BATCH-35-DEPENDENT-DECISION-REOPEN", "WIZ-20", []string{"TASKENTRY-26f912c6dfe4553d9785f5f3"}},
		"dossier_final_legible_vinculante_y_regenerable":        {"BEHAVIOR-AGENT-BATCH-35-FINAL-LAUNCH-DOSSIER", "WIZ-21", []string{"TASKENTRY-cd875a3ab313e43735884573", "TASKENTRY-7ade43d93fc60364b3d43280"}},
	}
}

func validateBehaviorBatch35(fixture, ledger, roadmap []byte, previous [][]byte) error {
	records, err := decodeBehaviorBatchRecords(fixture)
	if err != nil {
		return err
	}
	entries, err := decodeBehaviorBatch35Ledger(ledger)
	if err != nil {
		return err
	}
	if err := validateBehaviorBatch35Roadmap(roadmap); err != nil {
		return err
	}
	groups := behaviorBatch35Groups()
	priorEntry, priorRef := map[string]bool{}, map[string]bool{}
	for _, raw := range previous {
		prior, e := decodeBehaviorBatchRecords(raw)
		if e != nil {
			return e
		}
		for _, record := range prior {
			priorRef[record.Ref] = true
			for _, ref := range record.EntryRefs {
				priorEntry[ref] = true
			}
		}
	}
	seenEntry, seenRef := map[string]bool{}, map[string]bool{}
	var evidence []behaviorBatchEvidence
	if len(records) != 4 {
		return fmt.Errorf("conductas=%d", len(records))
	}
	for _, record := range records {
		group, ok := groups[record.Behavior]
		expectedCount := 2
		if group.capability == "WIZ-19" || group.capability == "WIZ-20" {
			expectedCount = 1
		}
		if !ok || seenRef[record.Ref] || priorRef[record.Ref] || !behaviorBatchHeaderValid(record, group.capability, group.ref) || !behaviorBatchFieldsPresent(record) || len(record.EntryRefs) != expectedCount || len(record.Evidence) != expectedCount {
			return fmt.Errorf("registro inválido: %s", record.Ref)
		}
		for i, ev := range record.Evidence {
			entry, exists := entries[ev.EntryRef]
			if !exists || ev.EntryRef != group.entries[i] || record.EntryRefs[i] != group.entries[i] || seenEntry[ev.EntryRef] || priorEntry[ev.EntryRef] || entry.CapabilityDecision != "accept" || entry.SemanticReviewState != "reviewed" || (entry.Disposition != "accepted_pending_reimplementation" && entry.Disposition != "historical_superseded_by_capability") || !behaviorBatchEvidenceMatches(group.capability, ev, entry.behaviorBatchTaskEntry) {
				return fmt.Errorf("procedencia inválida: %s", ev.EntryRef)
			}
			seenEntry[ev.EntryRef] = true
			evidence = append(evidence, ev)
		}
		seenRef[record.Ref] = true
		delete(groups, record.Behavior)
	}
	if len(groups) != 0 || len(seenEntry) != 6 || !behaviorBatch35RangesDisjoint(evidence) {
		return fmt.Errorf("cobertura o rangos inválidos")
	}
	return nil
}

func decodeBehaviorBatch35Ledger(raw []byte) (map[string]behaviorBatch35Entry, error) {
	result := map[string]behaviorBatch35Entry{}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		var entry behaviorBatch35Entry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, err
		}
		result[entry.EntryRef] = entry
	}
	return result, scanner.Err()
}
func behaviorBatch35RangesDisjoint(v []behaviorBatchEvidence) bool {
	for i := range v {
		for j := i + 1; j < len(v); j++ {
			if v[i].SourceRef == v[j].SourceRef && v[i].FirstLine <= v[j].LastLine && v[j].FirstLine <= v[i].LastLine {
				return false
			}
		}
	}
	return true
}

func validateBehaviorBatch35Roadmap(raw []byte) error {
	type cap struct {
		ID        string   `json:"id"`
		Title     string   `json:"title"`
		Decision  string   `json:"decision"`
		Kind      string   `json:"kind"`
		Owner     string   `json:"owner_context"`
		Contracts []string `json:"acceptance_contracts"`
		Status    string   `json:"status"`
		Evidence  []string `json:"evidence_refs"`
	}
	var root struct {
		Capabilities []cap `json:"capability_entries"`
	}
	if err := json.Unmarshal(raw, &root); err != nil {
		return err
	}
	want := map[string]cap{
		"WIZ-17": {Title: "Ayuda, glosario, ejemplos y “explícamelo todo”", Kind: "intake", Owner: "wizard", Contracts: []string{"AC-V23-WIZARD"}, Status: "declared"},
		"WIZ-19": {Title: "Modo determinista sin LLM y bot/RAG opcional", Kind: "intake", Owner: "wizard", Contracts: []string{"AC-V23-WIZARD"}, Status: "declared"},
		"WIZ-20": {Title: "Reabrir decisiones dependientes cuando cambia una respuesta", Kind: "intake", Owner: "wizard", Contracts: []string{"AC-V23-WIZARD"}, Status: "declared"},
		"WIZ-21": {Title: "Dossier final legible: producto, datos, UI, seguridad, i18n, deploy y riesgos", Kind: "intake", Owner: "wizard", Contracts: []string{"AC-V23-WIZARD"}, Status: "declared"},
	}
	for _, got := range root.Capabilities {
		expected, ok := want[got.ID]
		if !ok {
			continue
		}
		if got.Title != expected.Title || got.Decision != "accept" || got.Kind != expected.Kind || got.Owner != expected.Owner || got.Status != expected.Status || len(got.Contracts) != 1 || got.Contracts[0] != expected.Contracts[0] || len(got.Evidence) != len(expected.Evidence) {
			return fmt.Errorf("roadmap drift: %s", got.ID)
		}
		delete(want, got.ID)
	}
	if len(want) != 0 {
		return fmt.Errorf("capabilities ausentes: %v", want)
	}
	return nil
}
