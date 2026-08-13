// Estas utilidades validan el lote 28 contra ledger y roadmap; nunca ejecutan legacy.
package orquesta_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
)

type behaviorBatch28Entry struct {
	behaviorBatchTaskEntry
	CapabilityDecision  string `json:"capability_decision"`
	Disposition         string `json:"disposition"`
	SemanticReviewState string `json:"semantic_review_state"`
}

type behaviorBatch28Group struct {
	ref, capability string
	entries         []string
}

func behaviorBatch28Groups() map[string]behaviorBatch28Group {
	return map[string]behaviorBatch28Group{
		"bundle_documental_generado_por_tipo_y_locale":                     {"BEHAVIOR-AGENT-BATCH-28-GENERATED-APP-DOC-BUNDLE", "APP-13", []string{"TASKENTRY-2784cfd194c5069d8fab0ef9", "TASKENTRY-08ccaab251c766f33b527b79"}},
		"handoff_compacto_causal_antes_de_escalar_o_rotar":                 {"BEHAVIOR-AGENT-BATCH-28-COMPACT-CAUSAL-HANDOFF", "CTX-02", []string{"TASKENTRY-e835bdfbd3cbcdc841727b22", "TASKENTRY-aff804c77752dee62897bc16"}},
		"perfil_compacto_token_frugal_por_tipo_de_trabajo":                 {"BEHAVIOR-AGENT-BATCH-28-TOKEN-FRUGAL-COMPACT-PROFILE", "CTX-03", []string{"TASKENTRY-12ab4a46734398622a60b121"}},
		"artefactos_fuera_del_prompt_materializados_por_ref":               {"BEHAVIOR-AGENT-BATCH-28-REF-ONLY-ARTIFACT-CONTEXT", "CTX-04", []string{"TASKENTRY-7437b631e2feb75062a5ff0b"}},
		"prefijo_estable_cacheable_y_cola_dinamica_medida":                 {"BEHAVIOR-AGENT-BATCH-28-STABLE-PROMPT-CACHE-PREFIX", "CTX-05", []string{"TASKENTRY-2b6c2bb6b85b1bdf5912fa4a", "TASKENTRY-94777b218bd5513d1767d7c4"}},
		"compactacion_en_frontera_con_estado_durable_verificable":          {"BEHAVIOR-AGENT-BATCH-28-DURABLE-BOUNDARY-COMPACTION", "CTX-06", []string{"TASKENTRY-1e8f71eae9f6762d39f39de2", "TASKENTRY-88bf5be1bd8cbf41bd5a0f77"}},
		"routing_barato_para_mecanica_con_escalado_por_riesgo":             {"BEHAVIOR-AGENT-BATCH-28-RISK-AWARE-CHEAP-ROUTING", "CTX-07", []string{"TASKENTRY-5a44b8522d2fb08befb99efd", "TASKENTRY-6392da21dff8961dccc47bdb"}},
		"seleccion_explicita_de_forks_por_invariantes_y_benchmark":         {"BEHAVIOR-AGENT-BATCH-28-EXPLICIT-FORK-SELECTION", "EVD-08", []string{"TASKENTRY-0136ba4589fb578ecfeb7666"}},
		"efecto_externo_con_scope_opt_in_evidencia_y_receipt":              {"BEHAVIOR-AGENT-BATCH-28-AUTHORIZED-EXTERNAL-EFFECTS", "EVD-14", []string{"TASKENTRY-bddc1826c7bd3aa29853a620", "TASKENTRY-ec5b2bcc586f22dbb1c3cdb8"}},
		"dominio_externo_por_contrato_y_refs_opacas_sin_estado_compartido": {"BEHAVIOR-AGENT-BATCH-28-OPAQUE-EXTERNAL-DOMAIN-BOUNDARY", "EXT-00", []string{"TASKENTRY-a6294d362400b71c8d79e318", "TASKENTRY-41332a844f182687b98c7bf1"}},
	}
}

func validateBehaviorBatch28(fixture, ledger, roadmap []byte, previous [][]byte) error {
	records, err := decodeBehaviorBatchRecords(fixture)
	if err != nil {
		return err
	}
	entries, err := decodeBehaviorBatch28Ledger(ledger)
	if err != nil {
		return err
	}
	if err := validateBehaviorBatch28Roadmap(roadmap); err != nil {
		return err
	}
	groups := behaviorBatch28Groups()
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
	if len(records) != 10 {
		return fmt.Errorf("conductas=%d", len(records))
	}
	for _, record := range records {
		group, ok := groups[record.Behavior]
		expectedCount := 2
		if group.capability == "CTX-03" || group.capability == "CTX-04" || group.capability == "EVD-08" {
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
	if len(groups) != 0 || len(seenEntry) != 17 || !behaviorBatch28RangesDisjoint(evidence) {
		return fmt.Errorf("cobertura o rangos inválidos")
	}
	return nil
}

func decodeBehaviorBatch28Ledger(raw []byte) (map[string]behaviorBatch28Entry, error) {
	result := map[string]behaviorBatch28Entry{}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		var entry behaviorBatch28Entry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, err
		}
		result[entry.EntryRef] = entry
	}
	return result, scanner.Err()
}
func behaviorBatch28RangesDisjoint(v []behaviorBatchEvidence) bool {
	for i := range v {
		for j := i + 1; j < len(v); j++ {
			if v[i].SourceRef == v[j].SourceRef && v[i].FirstLine <= v[j].LastLine && v[j].FirstLine <= v[i].LastLine {
				return false
			}
		}
	}
	return true
}

func validateBehaviorBatch28Roadmap(raw []byte) error {
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
		"APP-13": {Title: "Documentación de usuario, operador, desarrollo y arquitectura", Kind: "generated_app_profile", Owner: "generated_apps", Contracts: []string{"AC-V33-GENERATED-APPS"}, Status: "declared"},
		"CTX-02": {Title: "Handoffs compactos con hechos, refs, decisiones, tests y bloqueos", Kind: "context", Owner: "context_rag_evals", Contracts: []string{"AC-V27-CONTEXT-RAG-EVALS"}, Status: "declared"},
		"CTX-03": {Title: "Perfil caveman/compacto para inventarios, búsquedas y subagentes", Kind: "context", Owner: "context_rag_evals", Contracts: []string{"AC-V27-CONTEXT-RAG-EVALS"}, Status: "declared"},
		"CTX-04": {Title: "Artefactos fuera del prompt; resumen + hash/ref", Kind: "context", Owner: "context_rag_evals", Contracts: []string{"AC-V27-CONTEXT-RAG-EVALS"}, Status: "declared"},
		"CTX-05": {Title: "Prompt caching por prefijos estables, medido por provider", Kind: "context", Owner: "context_rag_evals", Contracts: []string{"AC-V27-CONTEXT-RAG-EVALS"}, Status: "declared"},
		"CTX-06": {Title: "Compactación en límites de fase/sesión con estado verificable fuera del resumen", Kind: "context", Owner: "context_rag_evals", Contracts: []string{"AC-V27-CONTEXT-RAG-EVALS"}, Status: "declared"},
		"CTX-07": {Title: "Routing barato para trabajo mecánico y escalado por riesgo/fallo", Kind: "context", Owner: "context_rag_evals", Contracts: []string{"AC-V27-CONTEXT-RAG-EVALS"}, Status: "declared"},
		"EVD-08": {Title: "Integración/refinery con conflictos y selección explícita", Kind: "evidence", Owner: "cutover_accreditation", Contracts: []string{"AC-V34-CUTOVER-ACCREDITATION"}, Status: "declared"},
		"EVD-14": {Title: "Efectos externos requieren autoridad explícita", Kind: "evidence", Owner: "budgets_effects", Contracts: []string{"AC-V15-BUDGETS-EFFECTS"}, Status: "accredited", Evidence: make([]string, 3)},
		"EXT-00": {Title: "Toda app/dominio externo entra por contrato y refs opacas; nunca comparte DB o filesystem interno de Orquesta", Kind: "extension", Owner: "domain_plugins", Contracts: []string{"AC-V28-DOMAIN-PLUGINS"}, Status: "declared"},
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
