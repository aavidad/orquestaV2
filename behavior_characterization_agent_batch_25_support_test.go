// Estas utilidades validan el lote 25 contra ledger y roadmap; nunca ejecutan legacy.
package orquesta_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
)

type behaviorBatch25Entry struct {
	behaviorBatchTaskEntry
	CapabilityDecision  string `json:"capability_decision"`
	Disposition         string `json:"disposition"`
	SemanticReviewState string `json:"semantic_review_state"`
}

type behaviorBatch25Group struct {
	ref, capability string
	entries         []string
}

func behaviorBatch25Groups() map[string]behaviorBatch25Group {
	return map[string]behaviorBatch25Group{
		"targets_deploy_portables_por_plan_y_dry_run":             {"BEHAVIOR-AGENT-BATCH-25-PORTABLE-DEPLOY-TARGETS", "OPS-24", []string{"TASKENTRY-a3906b0e1758f2d8edd1a592", "TASKENTRY-b55c6951e568c345278ab1c1"}},
		"rework_y_replan_causales_sin_descartar_trabajo_util":     {"BEHAVIOR-AGENT-BATCH-25-CAUSAL-REWORK-REPLAN", "STG-15", []string{"TASKENTRY-919f1f004074e3b6d637afdf", "TASKENTRY-fc2db9b65dae1420733b57b8"}},
		"promocion_de_optimizaciones_solo_tras_eval_ab":           {"BEHAVIOR-AGENT-BATCH-25-EVAL-BEFORE-OPTIMIZE", "CTX-12", []string{"TASKENTRY-edfb6cb4bc8cc731b581eb5e", "TASKENTRY-a8845d679002267cc0d8c83a"}},
		"consejo_por_politica_durable_y_skip_acreditado":          {"BEHAVIOR-AGENT-BATCH-25-DURABLE-COUNCIL-POLICY", "GOV-11", []string{"TASKENTRY-7545b9fb6626a1b3b6422ef4", "TASKENTRY-d0423706a16b764f5aeffd92"}},
		"registro_config_tipado_unico_hasta_adaptadores":          {"BEHAVIOR-AGENT-BATCH-25-TYPED-CONFIG-REGISTRY", "OPS-01", []string{"TASKENTRY-e9ea0d1a0623263dc4cdd37a", "TASKENTRY-a3c92c35c603f520bd1ca758"}},
		"snapshot_efectivo_inmutable_redactado_y_no_entrada":      {"BEHAVIOR-AGENT-BATCH-25-REDACTED-EFFECTIVE-CONFIG", "OPS-04", []string{"TASKENTRY-4cba2d84600f99cc32184880", "TASKENTRY-06b986108da77b6b6862951c"}},
		"operacion_atomica_instalar_doctor_actualizar_y_rollback": {"BEHAVIOR-AGENT-BATCH-25-ATOMIC-INSTALL-UPDATE-ROLLBACK", "OPS-15", []string{"TASKENTRY-a69a9ffa0ce8f69d11037abd", "TASKENTRY-bec4d4d0b486c6dc9cb7d5b4"}},
		"perfil_neutral_resuelto_por_ejecucion":                   {"BEHAVIOR-AGENT-BATCH-25-EXECUTION-WORK-PROFILE", "ORC-06", []string{"TASKENTRY-5f6cfbf998cb9f39ac27062a", "TASKENTRY-5ba76c81874326ea20a64452"}},
		"criticidad_seguridad_como_piso_separado_de_esfuerzo":     {"BEHAVIOR-AGENT-BATCH-25-SECURITY-CRITICALITY-FLOOR", "ORC-08", []string{"TASKENTRY-b498e7cad57342e165065aa9", "TASKENTRY-672dae0d1d1decbca4e13b9d"}},
		"runtime_local_openai_compatible_detras_de_provider_port": {"BEHAVIOR-AGENT-BATCH-25-LOCAL-OPENAI-COMPATIBLE", "AGT-08", []string{"TASKENTRY-a42be37b8ae7520a0b056e6b"}},
	}
}

func validateBehaviorBatch25(fixture, ledger, roadmap []byte, previous [][]byte) error {
	records, err := decodeBehaviorBatchRecords(fixture)
	if err != nil {
		return err
	}
	entries, err := decodeBehaviorBatch25Ledger(ledger)
	if err != nil {
		return err
	}
	if err := validateBehaviorBatch25Roadmap(roadmap); err != nil {
		return err
	}
	groups := behaviorBatch25Groups()
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
		if group.capability == "AGT-08" {
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
	if len(groups) != 0 || len(seenEntry) != 19 || !behaviorBatch25RangesDisjoint(evidence) {
		return fmt.Errorf("cobertura o rangos inválidos")
	}
	return nil
}

func decodeBehaviorBatch25Ledger(raw []byte) (map[string]behaviorBatch25Entry, error) {
	result := map[string]behaviorBatch25Entry{}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		var entry behaviorBatch25Entry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, err
		}
		result[entry.EntryRef] = entry
	}
	return result, scanner.Err()
}
func behaviorBatch25RangesDisjoint(v []behaviorBatchEvidence) bool {
	for i := range v {
		for j := i + 1; j < len(v); j++ {
			if v[i].SourceRef == v[j].SourceRef && v[i].FirstLine <= v[j].LastLine && v[j].FirstLine <= v[i].LastLine {
				return false
			}
		}
	}
	return true
}

func validateBehaviorBatch25Roadmap(raw []byte) error {
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
		"OPS-24": {Title: "Docker remoto/SSH, Kubernetes, PaaS, serverless y Dokploy", Kind: "operations", Owner: "deploy_notifications", Contracts: []string{"AC-V29-DEPLOY-NOTIFICATIONS"}, Status: "declared"},
		"STG-15": {Title: "Rework/replan causal", Kind: "stage_template", Owner: "controls", Contracts: []string{"AC-V14-CONTROLS"}, Status: "accredited", Evidence: make([]string, 3)},
		"CTX-12": {Title: "Evals de calidad, coste, latencia y recall antes de promover optimización", Kind: "context", Owner: "context_rag_evals", Contracts: []string{"AC-V27-CONTEXT-RAG-EVALS"}, Status: "declared"},
		"GOV-11": {Title: "Consejo por política durable `auto|required|skip_by_operator`; un skip exige operador, motivo, fecha, `spec_hash` y receipt", Kind: "governance", Owner: "council", Contracts: []string{"AC-V19-COUNCIL"}, Status: "accredited", Evidence: make([]string, 3)},
		"OPS-01": {Title: "Registro tipado único de configuración", Kind: "operations", Owner: "config", Contracts: []string{"AC-V07-CONFIG"}, Status: "accredited", Evidence: make([]string, 3)},
		"OPS-04": {Title: "Snapshot efectivo generado, inmutable y redactado", Kind: "operations", Owner: "config", Contracts: []string{"AC-V07-CONFIG"}, Status: "accredited", Evidence: make([]string, 3)},
		"OPS-15": {Title: "Instalación, doctor, actualización y rollback", Kind: "operations", Owner: "operations_telemetry", Contracts: []string{"AC-V32-OPERATIONS-TELEMETRY"}, Status: "declared"},
		"ORC-06": {Title: "Roles, habilidades, herramientas y capacidades por ejecución", Kind: "orchestration", Owner: "goal_dag_phases", Contracts: []string{"AC-V05-GOAL-DAG-PHASES"}, Status: "accredited", Evidence: make([]string, 3)},
		"ORC-08": {Title: "Criticidad de seguridad separada del esfuerzo del modelo", Kind: "orchestration", Owner: "budgets_effects", Contracts: []string{"AC-V15-BUDGETS-EFFECTS"}, Status: "accredited", Evidence: make([]string, 3)},
		"AGT-08": {Title: "Runtime local OpenAI-compatible/vLLM", Kind: "agent_protocol", Owner: "provider_adapters", Contracts: []string{"AC-V25-PROVIDER-ADAPTERS"}, Status: "declared"},
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
