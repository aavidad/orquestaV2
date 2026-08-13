// Estas utilidades validan el lote 32 contra ledger y roadmap; nunca ejecutan legacy.
package orquesta_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
)

type behaviorBatch32Entry struct {
	behaviorBatchTaskEntry
	CapabilityDecision  string `json:"capability_decision"`
	Disposition         string `json:"disposition"`
	SemanticReviewState string `json:"semantic_review_state"`
}

type behaviorBatch32Group struct {
	ref, capability string
	entries         []string
}

func behaviorBatch32Groups() map[string]behaviorBatch32Group {
	return map[string]behaviorBatch32Group{
		"broker_contexto_compacto_por_ref_busqueda_y_limites":         {"BEHAVIOR-AGENT-BATCH-32-COMPACT-CONTEXT-BROKER", "ORC-19", []string{"TASKENTRY-fe0310e4473dadb530972cdb", "TASKENTRY-fc38a1e681d218f77978aa5f"}},
		"memoria_de_proyecto_con_autoridad_y_deteccion_de_drift":      {"BEHAVIOR-AGENT-BATCH-32-PROJECT-MEMORY-DRIFT", "ORC-20", []string{"TASKENTRY-71592040ac5961f1c32a5837", "TASKENTRY-9c5c018855dcfb518a5b0750"}},
		"rag_de_dominio_paginado_con_presupuesto_y_refs":              {"BEHAVIOR-AGENT-BATCH-32-BOUNDED-DOMAIN-RAG", "ORC-21", []string{"TASKENTRY-11c1e4769aef0a9138c801d8"}},
		"control_residente_dormido_con_wakeup_watchdog_y_stop_idle":   {"BEHAVIOR-AGENT-BATCH-32-EVENT-DRIVEN-IDLE-CONTROL", "ORC-25", []string{"TASKENTRY-c23c733ce7319307e37bf255", "TASKENTRY-819cc757e9ccbae51ccf7ba6"}},
		"pipeline_de_fases_causales_sin_lifecycle_paralelo":           {"BEHAVIOR-AGENT-BATCH-32-CAUSAL-PHASE-PIPELINE", "STG-00", []string{"TASKENTRY-e32249ff64e1539ac20967cc"}},
		"descubrimiento_workspace_con_fuentes_forenses_en_cuarentena": {"BEHAVIOR-AGENT-BATCH-32-FORENSIC-WORKSPACE-DISCOVERY", "STG-02", []string{"TASKENTRY-24a562270693d5493e0247b4"}},
		"brainstorm_durable_antes_de_fijar_solucion":                  {"BEHAVIOR-AGENT-BATCH-32-DURABLE-BRAINSTORM-ALTERNATIVES", "STG-06", []string{"TASKENTRY-c81bc0ace137c6837b5acfc8", "TASKENTRY-ae321a2098e7f9d31564f4f5"}},
		"corte_arquitectura_seguridad_y_gobierno_antes_de_planificar": {"BEHAVIOR-AGENT-BATCH-32-ARCH-SECURITY-GOVERNANCE-CUT", "STG-07", []string{"TASKENTRY-2da2b74ec05b571d446783d9", "TASKENTRY-a8d6b897cdc1c233ad2c2514"}},
		"rondas_consejo_convertidas_en_trabajo_neutral":               {"BEHAVIOR-AGENT-BATCH-32-COUNCIL-PLAN-TO-CANDIDATES", "STG-08", []string{"TASKENTRY-a2577fe6ae3d6988db75be57"}},
		"plan_con_write_set_cerrado_dependencias_y_presupuesto":       {"BEHAVIOR-AGENT-BATCH-32-WRITESET-PRECEDENCE-PLANNING", "STG-09", []string{"TASKENTRY-cf305697da6d74165edf2b58", "TASKENTRY-6a2ae1672dc3e81c652a9786"}},
	}
}

func validateBehaviorBatch32(fixture, ledger, roadmap []byte, previous [][]byte) error {
	records, err := decodeBehaviorBatchRecords(fixture)
	if err != nil {
		return err
	}
	entries, err := decodeBehaviorBatch32Ledger(ledger)
	if err != nil {
		return err
	}
	if err := validateBehaviorBatch32Roadmap(roadmap); err != nil {
		return err
	}
	groups := behaviorBatch32Groups()
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
		if group.capability == "ORC-21" || group.capability == "STG-00" || group.capability == "STG-02" || group.capability == "STG-08" {
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
	if len(groups) != 0 || len(seenEntry) != 16 || !behaviorBatch32RangesDisjoint(evidence) {
		return fmt.Errorf("cobertura o rangos inválidos")
	}
	return nil
}

func decodeBehaviorBatch32Ledger(raw []byte) (map[string]behaviorBatch32Entry, error) {
	result := map[string]behaviorBatch32Entry{}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		var entry behaviorBatch32Entry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, err
		}
		result[entry.EntryRef] = entry
	}
	return result, scanner.Err()
}
func behaviorBatch32RangesDisjoint(v []behaviorBatchEvidence) bool {
	for i := range v {
		for j := i + 1; j < len(v); j++ {
			if v[i].SourceRef == v[j].SourceRef && v[i].FirstLine <= v[j].LastLine && v[j].FirstLine <= v[i].LastLine {
				return false
			}
		}
	}
	return true
}

func validateBehaviorBatch32Roadmap(raw []byte) error {
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
		"ORC-19": {Title: "Context broker con refs compactas, búsqueda y límites", Kind: "orchestration", Owner: "context_rag_evals", Contracts: []string{"AC-V27-CONTEXT-RAG-EVALS"}, Status: "declared"},
		"ORC-20": {Title: "Memoria/conocimiento por proyecto y detección de drift", Kind: "orchestration", Owner: "context_rag_evals", Contracts: []string{"AC-V27-CONTEXT-RAG-EVALS"}, Status: "declared"},
		"ORC-21": {Title: "RAG documental/dominio sin cargar corpus completos", Kind: "orchestration", Owner: "context_rag_evals", Contracts: []string{"AC-V27-CONTEXT-RAG-EVALS"}, Status: "declared"},
		"ORC-25": {Title: "Autocontrol de CPU ociosa, procesos y temporales", Kind: "orchestration", Owner: "operations_telemetry", Contracts: []string{"AC-V32-OPERATIONS-TELEMETRY"}, Status: "declared"},
		"STG-00": {Title: "`PhaseInstance` causal e inmutable: identidad, plantilla, entrada y criterios; progreso derivado de WorkItems/receipts, sin lifecycle propio", Kind: "stage_template", Owner: "goal_dag_phases", Contracts: []string{"AC-V05-GOAL-DAG-PHASES"}, Status: "accredited", Evidence: make([]string, 3)},
		"STG-02": {Title: "Descubrimiento de workspace, inventario y reutilización", Kind: "stage_template", Owner: "workspace_git", Contracts: []string{"AC-V16-WORKSPACE-GIT"}, Status: "accredited", Evidence: make([]string, 3)},
		"STG-06": {Title: "Brainstorming y alternativas", Kind: "stage_template", Owner: "council", Contracts: []string{"AC-V19-COUNCIL"}, Status: "accredited", Evidence: make([]string, 3)},
		"STG-07": {Title: "Arquitectura, seguridad y gobierno", Kind: "stage_template", Owner: "wizard", Contracts: []string{"AC-V23-WIZARD"}, Status: "declared"},
		"STG-08": {Title: "Consejo/votación cuando la política lo exige", Kind: "stage_template", Owner: "council", Contracts: []string{"AC-V19-COUNCIL"}, Status: "accredited", Evidence: make([]string, 3)},
		"STG-09": {Title: "Planificación, descomposición, write-sets y presupuesto", Kind: "stage_template", Owner: "budgets_effects", Contracts: []string{"AC-V15-BUDGETS-EFFECTS"}, Status: "accredited", Evidence: make([]string, 3)},
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
