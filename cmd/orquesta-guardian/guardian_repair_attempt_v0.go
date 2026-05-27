package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const guardianRepairAttemptSchemaVersionV0 = "orquesta_guardian_repair_attempt.v0"

type guardianRepairPacketReceiptV0 struct {
	Path              string
	RepairAttemptRef  string
	FailurePacketHash string
}

type guardianRepairAttemptDecisionV0 struct {
	Allowed          bool
	RepairAttemptRef string
	ReasonCode       string
	EvidenceRefs     []string
}

type guardianRepairAttemptRecordV0 struct {
	SchemaVersion     string   `json:"schema_version"`
	AttemptScopeRef   string   `json:"attempt_scope_ref"`
	RepairAttemptRef  string   `json:"repair_attempt_ref"`
	FailurePacketHash string   `json:"failure_packet_hash"`
	FailurePhase      string   `json:"failure_phase"`
	Status            string   `json:"status"`
	BudgetMaxAttempts int      `json:"budget_max_attempts"`
	RetryEvidenceRefs []string `json:"retry_evidence_refs,omitempty"`
	CreatedAt         string   `json:"created_at"`
}

func finalizeGuardianRepairPacketV0(
	config guardianConfigV0,
	packet guardianRepairPacketV0,
) (guardianRepairPacketV0, guardianRepairPacketReceiptV0) {
	hash := guardianFailurePacketHashV0(packet)
	attemptRef := guardianRepairAttemptRefV0(config, hash)
	packet.FailurePacketHash = hash
	packet.RepairAttemptRef = attemptRef
	packet.RepairBudget = guardianRepairBudgetV0{MaxAgents: 1, MaxAttempts: config.RepairMaxAttempts}
	return packet, guardianRepairPacketReceiptV0{
		Path:              guardianRepairPacketPathV0(config, hash),
		RepairAttemptRef:  attemptRef,
		FailurePacketHash: hash,
	}
}

func guardianClaimRepairAttemptV0(
	config guardianConfigV0,
	phase string,
	receipt guardianRepairPacketReceiptV0,
) guardianRepairAttemptDecisionV0 {
	if receipt.FailurePacketHash == "" || receipt.RepairAttemptRef == "" {
		return guardianRepairAttemptBlockedV0("guardian_repair_attempt_invalid")
	}
	recordPath := guardianRepairAttemptRecordPathV0(config, receipt.RepairAttemptRef)
	if _, err := os.Stat(recordPath); err == nil {
		return guardianRepairAttemptBlockedV0("guardian_repair_attempt_duplicate")
	} else if !errors.Is(err, os.ErrNotExist) {
		return guardianRepairAttemptBlockedV0("guardian_repair_attempt_state_read_failed")
	}
	if guardianRepairAttemptCountV0(config) >= config.RepairMaxAttempts {
		return guardianRepairAttemptBlockedV0("guardian_repair_attempt_budget_exhausted")
	}
	record := guardianRepairAttemptRecordV0{
		SchemaVersion:     guardianRepairAttemptSchemaVersionV0,
		AttemptScopeRef:   guardianRepairAttemptScopeRefV0(config),
		RepairAttemptRef:  receipt.RepairAttemptRef,
		FailurePacketHash: receipt.FailurePacketHash,
		FailurePhase:      strings.TrimSpace(phase),
		Status:            "started",
		BudgetMaxAttempts: config.RepairMaxAttempts,
		RetryEvidenceRefs: append([]string(nil), config.RepairRetryEvidenceRefs...),
		CreatedAt:         config.OccurredAt.Format("2006-01-02T15:04:05Z"),
	}
	if err := writeJSONFileV0(recordPath, record); err != nil {
		return guardianRepairAttemptBlockedV0("guardian_repair_attempt_state_write_failed")
	}
	return guardianRepairAttemptDecisionV0{
		Allowed:          true,
		RepairAttemptRef: receipt.RepairAttemptRef,
		EvidenceRefs: []string{
			"evidence-ref-guardian-repair-attempt-claimed",
			"evidence-ref-guardian-repair-attempt-budget",
		},
	}
}

func guardianFailurePacketHashV0(packet guardianRepairPacketV0) string {
	commands := make([]guardianPublicCommandResultV0, 0, len(packet.Commands))
	for _, command := range packet.Commands {
		commands = append(commands, guardianPublicCommandResultV0{
			Phase:      command.Phase,
			CommandRef: command.CommandRef,
			ExitCode:   command.ExitCode,
			ReasonCode: command.ReasonCode,
		})
	}
	payload, _ := json.Marshal(struct {
		SchemaVersion       string                          `json:"schema_version"`
		FailurePhase        string                          `json:"failure_phase"`
		Summary             string                          `json:"summary"`
		ReasonCodes         []string                        `json:"reason_codes,omitempty"`
		RequiredCommandRefs []string                        `json:"required_command_refs,omitempty"`
		Commands            []guardianPublicCommandResultV0 `json:"commands,omitempty"`
	}{
		SchemaVersion:       packet.SchemaVersion,
		FailurePhase:        packet.FailurePhase,
		Summary:             packet.Summary,
		ReasonCodes:         packet.ReasonCodes,
		RequiredCommandRefs: packet.RequiredCommandRefs,
		Commands:            commands,
	})
	sum := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func guardianRepairAttemptRefV0(config guardianConfigV0, failureHash string) string {
	key := strings.Join([]string{
		guardianRepairAttemptScopeRefV0(config),
		strings.TrimSpace(failureHash),
		strings.Join(config.RepairRetryEvidenceRefs, ","),
	}, "|")
	return "repair-attempt-ref-" + guardianShortHashV0(key)
}

func guardianRepairAttemptScopeRefV0(config guardianConfigV0) string {
	for _, ref := range []string{config.PromotionRef, config.AttemptRef, config.RunRef} {
		if strings.TrimSpace(ref) != "" {
			return strings.TrimSpace(ref)
		}
	}
	return "guardian-repair-attempt-scope-empty"
}

func guardianRepairPacketPathV0(config guardianConfigV0, failureHash string) string {
	base := strings.TrimSuffix(config.RepairPacketPath, filepath.Ext(config.RepairPacketPath))
	return base + "-" + guardianShortHashV0(failureHash) + filepath.Ext(config.RepairPacketPath)
}

func guardianRepairAttemptRecordPathV0(config guardianConfigV0, repairAttemptRef string) string {
	return filepath.Join(
		config.StateDir,
		"repair-attempts",
		safeGuardianAttemptFilePartV0(guardianRepairAttemptScopeRefV0(config)),
		sanitizeFilenamePartV0(repairAttemptRef)+".json",
	)
}

func guardianRepairAttemptCountV0(config guardianConfigV0) int {
	dir := filepath.Join(
		config.StateDir,
		"repair-attempts",
		safeGuardianAttemptFilePartV0(guardianRepairAttemptScopeRefV0(config)),
	)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			count++
		}
	}
	return count
}

func guardianRepairAttemptBlockedV0(reason string) guardianRepairAttemptDecisionV0 {
	return guardianRepairAttemptDecisionV0{
		ReasonCode: strings.TrimSpace(reason),
		EvidenceRefs: []string{
			"evidence-ref-guardian-repair-attempt-blocked",
		},
	}
}
