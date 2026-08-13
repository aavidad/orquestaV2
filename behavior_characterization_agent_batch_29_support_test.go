// Estas utilidades validan el lote 29 contra ledger y roadmap; el legado nunca es autoridad runtime.
package orquesta_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

const behaviorBatch29ExpectedEntryCount = 16

type behaviorBatch29TaskEntry struct {
	behaviorBatchTaskEntry
	CapabilityDecision  string `json:"capability_decision"`
	SemanticReviewState string `json:"semantic_review_state"`
}

type behaviorBatch29Roadmap struct {
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

func behaviorBatch29Groups() map[string]struct {
	ref, capability, anchor string
	entries                 []string
} {
	return map[string]struct {
		ref, capability, anchor string
		entries                 []string
	}{
		"ingesta_mapper_validator_receipt_durable":            {"BEHAVIOR-AGENT-BATCH-29-DURABLE-DATA-INGESTION", "EXT-05", "receipt durable", []string{"TASKENTRY-912c4b4949d38a3f9f2d38a7"}},
		"plan_documental_expande_unidades_con_refs_unicas":    {"BEHAVIOR-AGENT-BATCH-29-DOCUMENT-PLAN-EXPANSION", "EXT-07", "refs deterministas", []string{"TASKENTRY-899ca5c66a1f32b7b54b05c2", "TASKENTRY-1c8f98a7d5a938f7df800bad"}},
		"broker_codigo_neutral_un_indexador_gobernado":        {"BEHAVIOR-AGENT-BATCH-29-CODEBASE-BROKER", "EXT-08", "indexador por agente", []string{"TASKENTRY-cf6350df17451229c4c80e74", "TASKENTRY-f43162c03f9b4d1dabf5952c"}},
		"forges_git_detras_puerto_comun":                      {"BEHAVIOR-AGENT-BATCH-29-FORGE-CONNECTORS", "EXT-11", "familia de conectores", []string{"TASKENTRY-99b75ef6de69d76717b24263"}},
		"lectura_db_externa_por_dialecto_y_solo_lectura":      {"BEHAVIOR-AGENT-BATCH-29-EXTERNAL-DB-READ", "EXT-17", "solo lectura", []string{"TASKENTRY-73c373ad29487a3be94d81ff"}},
		"intent_manifest_integro_inmutable_antes_appspec":     {"BEHAVIOR-AGENT-BATCH-29-IMMUTABLE-INTENT", "GOV-02", "negativo de hardlink", []string{"TASKENTRY-cc827a55ebabfe9e9bd686f7", "TASKENTRY-847f50973e2e0196041fd389"}},
		"control_humano_autorizado_con_efecto_causal":         {"BEHAVIOR-AGENT-BATCH-29-HUMAN-CONTROLS", "GOV-07", "cero procesos", []string{"TASKENTRY-0ad29d7aa62dceb5f8211f87", "TASKENTRY-f692ec07292973260331c2a5"}},
		"votante_derivado_launch_acreditado_no_payload":       {"BEHAVIOR-AGENT-BATCH-29-ACCREDITED-VOTER", "GOV-14", "voter_ref autocontenido", []string{"TASKENTRY-9588d379eaa1c5458545a27a"}},
		"multiproyecto_sin_db_global_y_con_contexto":          {"BEHAVIOR-AGENT-BATCH-29-MULTIPROJECT-ISOLATION", "GOV-19", "db global", []string{"TASKENTRY-301876e0441173012475abf3", "TASKENTRY-1c43c7bcf8084c87aece64e1"}},
		"jerarquia_workspace_grupo_proyecto_repo_refs_opacas": {"BEHAVIOR-AGENT-BATCH-29-OPAQUE-WORKSPACE-HIERARCHY", "GOV-22", "path como identidad", []string{"TASKENTRY-b2174c9cc4b28b4105f444a2", "TASKENTRY-85ac1ae2ff36cdb3985f217a"}},
	}
}

func validateBehaviorBatch29(fixture, ledger, roadmap []byte, previous [][]byte) error {
	if err := validateBehaviorBatch29Roadmap(roadmap); err != nil {
		return err
	}
	records, err := decodeBehaviorBatchRecords(fixture)
	if err != nil {
		return err
	}
	entries, err := decodeBehaviorBatch29Ledger(ledger)
	if err != nil {
		return err
	}
	groups := behaviorBatch29Groups()
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
	evidence := make([]behaviorBatchEvidence, 0, behaviorBatch29ExpectedEntryCount)
	if len(records) != 10 {
		return fmt.Errorf("conductas=%d", len(records))
	}
	for _, record := range records {
		group, ok := groups[record.Behavior]
		_, duplicate := seen[record.Ref]
		if !ok || duplicate || !behaviorBatch29HeaderValid(record, group.ref, group.capability) || len(record.EntryRefs) != len(group.entries) || len(record.Evidence) != len(group.entries) || !behaviorBatchFieldsPresent(record) || !strings.Contains(behaviorBatch29Text(record), group.anchor) {
			return fmt.Errorf("conducta inválida: %s", record.Ref)
		}
		for index, item := range record.Evidence {
			entry, exists := entries[item.EntryRef]
			_, reused := used[item.EntryRef]
			if item.EntryRef != group.entries[index] || record.EntryRefs[index] != item.EntryRef || reused || !exists || !behaviorBatch29EvidenceMatches(item, entry) {
				return fmt.Errorf("procedencia inválida: %s", item.EntryRef)
			}
			used[item.EntryRef] = struct{}{}
			evidence = append(evidence, item)
		}
		seen[record.Ref] = struct{}{}
		delete(groups, record.Behavior)
	}
	if len(groups) != 0 || len(evidence) != behaviorBatch29ExpectedEntryCount || !behaviorBatch29RangesDisjoint(evidence) {
		return fmt.Errorf("cobertura o rangos inválidos")
	}
	return nil
}

func decodeBehaviorBatch29Ledger(raw []byte) (map[string]behaviorBatch29TaskEntry, error) {
	result := map[string]behaviorBatch29TaskEntry{}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		var entry behaviorBatch29TaskEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, err
		}
		result[entry.EntryRef] = entry
	}
	return result, scanner.Err()
}

func behaviorBatch29HeaderValid(record behaviorBatchRecord, ref, capability string) bool {
	return record.SchemaVersion == 1 && record.Ref == ref && record.CapabilityID == capability && record.Authority == "proposal_fixture_not_canonical_ledger" && record.ReviewState == "bootstrap_first_review_pending_independent_counterreview" && record.Disposition == "not_evaluated" && !record.CanonicalChange && !record.CreatesWork && !record.ClosesCapability && !record.ClaimsAccreditation
}

func behaviorBatch29EvidenceMatches(e behaviorBatchEvidence, entry behaviorBatch29TaskEntry) bool {
	return entry.CapabilityDecision == "accept" && entry.SemanticReviewState == "reviewed" && behaviorBatchEvidenceMatches(entry.CapabilityID, e, entry.behaviorBatchTaskEntry)
}

func behaviorBatch29Text(record behaviorBatchRecord) string {
	values := []string{record.Problem, record.DecisionAuthority}
	groups := [][]string{record.Users, record.Inputs, record.Outputs, record.StateRead, record.StateWritten, record.Permissions.Permissions, record.Permissions.Secrets, record.Permissions.Effects, record.Recovery.Failure, record.Recovery.Retry, record.Recovery.Concurrency, record.Recovery.Restart, record.Worked, record.Failed, record.Preserve, record.Avoid, record.Uncertainties, record.Attempts}
	for _, group := range groups {
		values = append(values, group...)
	}
	return strings.Join(values, "\n")
}

func behaviorBatch29RangesDisjoint(items []behaviorBatchEvidence) bool {
	for left := range items {
		for right := left + 1; right < len(items); right++ {
			if items[left].SourceRef == items[right].SourceRef && items[left].FirstLine <= items[right].LastLine && items[right].FirstLine <= items[left].LastLine {
				return false
			}
		}
	}
	return true
}

func validateBehaviorBatch29Roadmap(raw []byte) error {
	var roadmap behaviorBatch29Roadmap
	if err := json.Unmarshal(raw, &roadmap); err != nil {
		return err
	}
	expected := map[string]struct{ status, owner, acceptance string }{
		"EXT-05": {"declared", "domain_plugins", "AC-V28-DOMAIN-PLUGINS"}, "EXT-07": {"declared", "domain_plugins", "AC-V28-DOMAIN-PLUGINS"}, "EXT-08": {"declared", "domain_plugins", "AC-V28-DOMAIN-PLUGINS"}, "EXT-11": {"declared", "domain_plugins", "AC-V28-DOMAIN-PLUGINS"}, "EXT-17": {"declared", "domain_plugins", "AC-V28-DOMAIN-PLUGINS"}, "GOV-02": {"accredited", "intent_appspec", "AC-V04-INTENT-APPSPEC"}, "GOV-07": {"accredited", "controls", "AC-V14-CONTROLS"}, "GOV-14": {"accredited", "council", "AC-V19-COUNCIL"}, "GOV-19": {"accredited", "identity_projects_rbac", "AC-V10-IDENTITY-PROJECTS-RBAC"}, "GOV-22": {"accredited", "identity_projects_rbac", "AC-V10-IDENTITY-PROJECTS-RBAC"},
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
