// Estas utilidades validan el lote 33 contra ledger y roadmap; el legado nunca es autoridad runtime.
package orquesta_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

const behaviorBatch33ExpectedEntryCount = 13

type behaviorBatch33TaskEntry struct {
	behaviorBatchTaskEntry
	CapabilityDecision  string `json:"capability_decision"`
	SemanticReviewState string `json:"semantic_review_state"`
}

type behaviorBatch33Roadmap struct {
	Capabilities []struct {
		ID                  string   `json:"id"`
		Title               string   `json:"title"`
		Decision            string   `json:"decision"`
		Kind                string   `json:"kind"`
		OwnerContext        string   `json:"owner_context"`
		Status              string   `json:"status"`
		AcceptanceContracts []string `json:"acceptance_contracts"`
		EvidenceRefs        []string `json:"evidence_refs"`
	} `json:"capability_entries"`
}

func behaviorBatch33Groups() map[string]struct {
	ref, capability, anchor string
	entries                 []string
} {
	return map[string]struct {
		ref, capability, anchor string
		entries                 []string
	}{
		"preparacion_workspace_preserva_continuidad_y_backup":         {"BEHAVIOR-AGENT-BATCH-33-WORKSPACE-PREPARATION", "STG-10", "worktree aislado", []string{"TASKENTRY-f4268b3e5636ef27f69805d3"}},
		"validacion_final_registrada_no_cierra_por_si_sola":           {"BEHAVIOR-AGENT-BATCH-33-FINAL-VALIDATION", "STG-17", "evento que cierra Goal", []string{"TASKENTRY-3a8a8026a6cd07097e14f8e0"}},
		"cierre_archivo_aprendizaje_con_evidencia_y_followups":        {"BEHAVIOR-AGENT-BATCH-33-CLOSURE-ARCHIVE-LEARNING", "STG-20", "propuesta de borrado", []string{"TASKENTRY-5bb9b60d4e0a02b28880c3c9", "TASKENTRY-c8457e151d9811adefdb2649"}},
		"resources_direccionables_suscribibles_y_paginados":           {"BEHAVIOR-AGENT-BATCH-33-PAGED-RESOURCES", "TLS-03", "paginación", []string{"TASKENTRY-bfa0848120382fb4c62d0dbb"}},
		"skill_revisada_hash_permisos_revocacion_rollback":            {"BEHAVIOR-AGENT-BATCH-33-SKILL-IMMUTABLE-SNAPSHOT", "TLS-09", "snapshot por sesión", []string{"TASKENTRY-18a437d76b1a5987d2aa6726"}},
		"skill_install_upgrade_disable_remove_con_cas_auditoria":      {"BEHAVIOR-AGENT-BATCH-33-GOVERNED-SKILL-LIFECYCLE", "TLS-11", "disable antes de remove", []string{"TASKENTRY-c91726522fa395be47d02faa"}},
		"catalogo_skills_curado_carga_progresiva":                     {"BEHAVIOR-AGENT-BATCH-33-CURATED-SKILL-CATALOG", "TLS-12", "instalar todo", []string{"TASKENTRY-5602452355aa1df70f535b3e"}},
		"packs_reglas_workflows_version_scope_precedencia_revocacion": {"BEHAVIOR-AGENT-BATCH-33-VERSIONED-RULE-PACKS", "TLS-13", "sin activación automática", []string{"TASKENTRY-b9404cbf512296a5c68b61a5", "TASKENTRY-ee09d5f45ee4893125c01f04"}},
		"timeline_global_causal_paginada_y_no_autoritativa":           {"BEHAVIOR-AGENT-BATCH-33-CAUSAL-TIMELINE", "UI-11", "timeline que cierra", []string{"TASKENTRY-7811fd590d10f752a956348a", "TASKENTRY-a371c716107488af17904a96"}},
		"email_telegram_chat_principal_autorizado_y_receipt":          {"BEHAVIOR-AGENT-BATCH-33-AUTHORIZED-NOTIFICATIONS", "UI-16", "chat como principal", []string{"TASKENTRY-3073efa719c2acd7be4ea8ac"}},
	}
}

func validateBehaviorBatch33(fixture, ledger, roadmap []byte, previous [][]byte) error {
	if err := validateBehaviorBatch33Roadmap(roadmap); err != nil {
		return err
	}
	records, err := decodeBehaviorBatchRecords(fixture)
	if err != nil {
		return err
	}
	entries, err := decodeBehaviorBatch33Ledger(ledger)
	if err != nil {
		return err
	}
	groups := behaviorBatch33Groups()
	used := map[string]struct{}{}
	for _, raw := range previous {
		prior, decodeErr := decodeBehaviorBatchRecords(raw)
		if decodeErr != nil {
			return decodeErr
		}
		for _, record := range prior {
			for _, ref := range record.EntryRefs {
				used[ref] = struct{}{}
			}
		}
	}
	seen := map[string]struct{}{}
	evidence := make([]behaviorBatchEvidence, 0, behaviorBatch33ExpectedEntryCount)
	if len(records) != 10 {
		return fmt.Errorf("conductas=%d", len(records))
	}
	for _, record := range records {
		group, ok := groups[record.Behavior]
		_, duplicate := seen[record.Ref]
		if !ok || duplicate || !behaviorBatch33HeaderValid(record, group.ref, group.capability) || len(record.EntryRefs) != len(group.entries) || len(record.Evidence) != len(group.entries) || !behaviorBatchFieldsPresent(record) || !strings.Contains(behaviorBatch33Text(record), group.anchor) {
			return fmt.Errorf("conducta inválida: %s", record.Ref)
		}
		for index, item := range record.Evidence {
			entry, exists := entries[item.EntryRef]
			_, reused := used[item.EntryRef]
			if item.EntryRef != group.entries[index] || record.EntryRefs[index] != item.EntryRef || reused || !exists || !behaviorBatch33EvidenceMatches(item, entry) {
				return fmt.Errorf("procedencia inválida: %s", item.EntryRef)
			}
			used[item.EntryRef] = struct{}{}
			evidence = append(evidence, item)
		}
		seen[record.Ref] = struct{}{}
		delete(groups, record.Behavior)
	}
	if len(groups) != 0 || len(evidence) != behaviorBatch33ExpectedEntryCount || !behaviorBatch33RangesDisjoint(evidence) {
		return fmt.Errorf("cobertura o rangos inválidos")
	}
	return nil
}

func decodeBehaviorBatch33Ledger(raw []byte) (map[string]behaviorBatch33TaskEntry, error) {
	result := map[string]behaviorBatch33TaskEntry{}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		var entry behaviorBatch33TaskEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, err
		}
		result[entry.EntryRef] = entry
	}
	return result, scanner.Err()
}

func behaviorBatch33HeaderValid(record behaviorBatchRecord, ref, capability string) bool {
	return record.SchemaVersion == 1 && record.Ref == ref && record.CapabilityID == capability && record.Authority == "proposal_fixture_not_canonical_ledger" && record.ReviewState == "bootstrap_first_review_pending_independent_counterreview" && record.Disposition == "not_evaluated" && !record.CanonicalChange && !record.CreatesWork && !record.ClosesCapability && !record.ClaimsAccreditation
}

func behaviorBatch33EvidenceMatches(e behaviorBatchEvidence, entry behaviorBatch33TaskEntry) bool {
	return entry.CapabilityDecision == "accept" && entry.SemanticReviewState == "reviewed" && behaviorBatchEvidenceMatches(entry.CapabilityID, e, entry.behaviorBatchTaskEntry)
}

func behaviorBatch33Text(record behaviorBatchRecord) string {
	values := []string{record.Problem, record.DecisionAuthority}
	groups := [][]string{record.Users, record.Inputs, record.Outputs, record.StateRead, record.StateWritten, record.Permissions.Permissions, record.Permissions.Secrets, record.Permissions.Effects, record.Recovery.Failure, record.Recovery.Retry, record.Recovery.Concurrency, record.Recovery.Restart, record.Worked, record.Failed, record.Preserve, record.Avoid, record.Uncertainties, record.Attempts}
	for _, group := range groups {
		values = append(values, group...)
	}
	return strings.Join(values, "\n")
}

func behaviorBatch33RangesDisjoint(items []behaviorBatchEvidence) bool {
	for left := range items {
		for right := left + 1; right < len(items); right++ {
			if items[left].SourceRef == items[right].SourceRef && items[left].FirstLine <= items[right].LastLine && items[right].FirstLine <= items[left].LastLine {
				return false
			}
		}
	}
	return true
}

func validateBehaviorBatch33Roadmap(raw []byte) error {
	var roadmap behaviorBatch33Roadmap
	if err := json.Unmarshal(raw, &roadmap); err != nil {
		return err
	}
	expected := map[string]struct{ status, owner, acceptance string }{
		"STG-10": {"accredited", "workspace_git", "AC-V16-WORKSPACE-GIT"}, "STG-17": {"declared", "cutover_accreditation", "AC-V34-CUTOVER-ACCREDITATION"}, "STG-20": {"declared", "operations_telemetry", "AC-V32-OPERATIONS-TELEMETRY"}, "TLS-03": {"declared", "tools_skills_sdk", "AC-V26-TOOLS-SKILLS-SDK"}, "TLS-09": {"declared", "tools_skills_sdk", "AC-V26-TOOLS-SKILLS-SDK"}, "TLS-11": {"declared", "tools_skills_sdk", "AC-V26-TOOLS-SKILLS-SDK"}, "TLS-12": {"declared", "tools_skills_sdk", "AC-V26-TOOLS-SKILLS-SDK"}, "TLS-13": {"declared", "tools_skills_sdk", "AC-V26-TOOLS-SKILLS-SDK"}, "UI-11": {"declared", "web_admin", "AC-V24-WEB-ADMIN"}, "UI-16": {"declared", "deploy_notifications", "AC-V29-DEPLOY-NOTIFICATIONS"},
	}
	for _, capability := range roadmap.Capabilities {
		want, ok := expected[capability.ID]
		if !ok {
			continue
		}
		if capability.Decision != "accept" || capability.Status != want.status || capability.OwnerContext != want.owner || len(capability.AcceptanceContracts) != 1 || capability.AcceptanceContracts[0] != want.acceptance || (want.status == "accredited" && len(capability.EvidenceRefs) == 0) || (want.status == "declared" && len(capability.EvidenceRefs) != 0) {
			return fmt.Errorf("roadmap inesperado: %s", capability.ID)
		}
		delete(expected, capability.ID)
	}
	if len(expected) != 0 {
		return fmt.Errorf("faltan capabilities: %v", expected)
	}
	return nil
}
