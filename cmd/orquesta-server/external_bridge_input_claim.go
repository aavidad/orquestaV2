package main

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func externalBridgeClaimInputV0(
	ctx context.Context,
	ledger externalBridgeInputLedgerV0,
	entry externalBridgeInputLedgerEntryV0,
) (externalBridgeInputLedgerEntryV0, bool, error) {
	if ledger == nil {
		return entry, true, nil
	}
	return ledger.ClaimExternalBridgeInputV0(ctx, entry)
}

func (ledger *fileExternalBridgeInputLedgerV0) ClaimExternalBridgeInputV0(
	ctx context.Context,
	entry externalBridgeInputLedgerEntryV0,
) (externalBridgeInputLedgerEntryV0, bool, error) {
	if err := ctx.Err(); err != nil {
		return externalBridgeInputLedgerEntryV0{}, false, err
	}
	entry = normalizeExternalBridgeClaimEntryV0(entry)
	if entry.Key == "" {
		return externalBridgeInputLedgerEntryV0{}, false, fmt.Errorf("external_bridge_input_key_required")
	}
	ledger.mu.Lock()
	defer ledger.mu.Unlock()
	entries, err := ledger.loadV0()
	if err != nil {
		return externalBridgeInputLedgerEntryV0{}, false, err
	}
	if previous, ok := entries[entry.Key]; ok {
		if previous.Status == externalBridgeInputStatusSubmittedV0 ||
			previous.Status == externalBridgeInputStatusClaimedV0 {
			return previous, false, nil
		}
		if previous.Status == externalBridgeInputStatusSubmitFailedV0 {
			if !externalBridgeInputSameIntentV0(previous, entry) {
				return previous, false, fmt.Errorf("external_bridge_claim_conflict")
			}
			entry.Attempts = previous.Attempts + 1
			entry.UpdatedAt = time.Now().UTC()
			entries[entry.Key] = entry
			if err := ledger.saveV0(entries); err != nil {
				return externalBridgeInputLedgerEntryV0{}, false, err
			}
			return entry, true, nil
		}
		return previous, false, fmt.Errorf("external_bridge_claim_conflict")
	}
	entry.Attempts = 1
	entry.UpdatedAt = time.Now().UTC()
	entries[entry.Key] = entry
	if err := ledger.saveV0(entries); err != nil {
		return externalBridgeInputLedgerEntryV0{}, false, err
	}
	return entry, true, nil
}

func (ledger *fileExternalBridgeInputLedgerV0) RecordExternalBridgeInputSubmittedV0(
	ctx context.Context,
	entry externalBridgeInputLedgerEntryV0,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	entry = normalizeExternalBridgeSubmittedEntryV0(entry)
	if entry.Key == "" {
		return fmt.Errorf("external_bridge_input_key_required")
	}
	if entry.RunRef == "" {
		return fmt.Errorf("external_bridge_run_ref_required")
	}
	ledger.mu.Lock()
	defer ledger.mu.Unlock()
	entries, err := ledger.loadV0()
	if err != nil {
		return err
	}
	if previous, ok := entries[entry.Key]; ok {
		if previous.Status == externalBridgeInputStatusSubmittedV0 &&
			previous.RunRef != entry.RunRef {
			return fmt.Errorf("external_bridge_submitted_conflict")
		}
		if !externalBridgeInputSameIntentV0(previous, entry) {
			return fmt.Errorf("external_bridge_claim_conflict")
		}
		entry = mergeExternalBridgeInputMetadataV0(previous, entry)
	}
	if entry.Attempts <= 0 {
		entry.Attempts = 1
	}
	entry.UpdatedAt = time.Now().UTC()
	entries[entry.Key] = entry
	return ledger.saveV0(entries)
}

func normalizeExternalBridgeClaimEntryV0(
	entry externalBridgeInputLedgerEntryV0,
) externalBridgeInputLedgerEntryV0 {
	entry.Key = strings.TrimSpace(entry.Key)
	entry.ExternalSystem = strings.TrimSpace(entry.ExternalSystem)
	entry.ExternalJobRef = strings.TrimSpace(entry.ExternalJobRef)
	entry.Status = externalBridgeInputStatusClaimedV0
	entry.ClaimRef = strings.TrimSpace(entry.ClaimRef)
	entry.CorrelationID = strings.TrimSpace(entry.CorrelationID)
	entry.IdempotencyKey = strings.TrimSpace(entry.IdempotencyKey)
	entry.RunRef = strings.TrimSpace(entry.RunRef)
	entry.ChangeRef = strings.TrimSpace(entry.ChangeRef)
	entry.LastError = strings.TrimSpace(entry.LastError)
	return entry
}

func normalizeExternalBridgeSubmittedEntryV0(
	entry externalBridgeInputLedgerEntryV0,
) externalBridgeInputLedgerEntryV0 {
	entry = normalizeExternalBridgeClaimEntryV0(entry)
	entry.Status = externalBridgeInputStatusSubmittedV0
	return entry
}

func externalBridgeInputSameIntentV0(
	previous externalBridgeInputLedgerEntryV0,
	next externalBridgeInputLedgerEntryV0,
) bool {
	return matchingExternalBridgeInputValueV0(previous.CorrelationID, next.CorrelationID) &&
		matchingExternalBridgeInputValueV0(previous.IdempotencyKey, next.IdempotencyKey) &&
		matchingExternalBridgeInputValueV0(previous.ChangeRef, next.ChangeRef)
}

func matchingExternalBridgeInputValueV0(previous string, next string) bool {
	previous = strings.TrimSpace(previous)
	next = strings.TrimSpace(next)
	return previous == "" || next == "" || previous == next
}

func mergeExternalBridgeInputMetadataV0(
	previous externalBridgeInputLedgerEntryV0,
	next externalBridgeInputLedgerEntryV0,
) externalBridgeInputLedgerEntryV0 {
	next.ClaimRef = firstExternalBridgeInputValueV0(next.ClaimRef, previous.ClaimRef)
	next.CorrelationID = firstExternalBridgeInputValueV0(next.CorrelationID, previous.CorrelationID)
	next.IdempotencyKey = firstExternalBridgeInputValueV0(next.IdempotencyKey, previous.IdempotencyKey)
	next.ChangeRef = firstExternalBridgeInputValueV0(next.ChangeRef, previous.ChangeRef)
	next.Attempts = previous.Attempts + 1
	return next
}

func firstExternalBridgeInputValueV0(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
