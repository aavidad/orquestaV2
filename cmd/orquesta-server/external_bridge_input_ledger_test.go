package main

import (
	"context"
	"path/filepath"
	"testing"
)

func TestExternalBridgeInputLedgerSubmittedEntryIsScopedBySystemV0(t *testing.T) {
	ctx := context.Background()
	ledger, err := newFileExternalBridgeInputLedgerV0(
		filepath.Join(t.TempDir(), "external-bridge-input-ledger.json"),
	)
	if err != nil {
		t.Fatalf("ledger: %v", err)
	}

	if err := externalBridgeRecordSubmittedInputV0(
		ctx,
		ledger,
		"opes",
		"job-ref-001",
		"run-ref-001",
		"change-ref-001",
	); err != nil {
		t.Fatalf("record: %v", err)
	}

	_, foundOtherSystem, err := externalBridgeSubmittedInputLedgerEntryV0(
		ctx,
		ledger,
		"another-system",
		"job-ref-001",
	)
	if err != nil {
		t.Fatalf("lookup other system: %v", err)
	}
	if foundOtherSystem {
		t.Fatalf("entry leaked across external systems")
	}

	entry, foundOPES, err := externalBridgeSubmittedInputLedgerEntryV0(
		ctx,
		ledger,
		"opes",
		"job-ref-001",
	)
	if err != nil {
		t.Fatalf("lookup opes: %v", err)
	}
	if !foundOPES ||
		entry.ExternalSystem != "opes" ||
		entry.ExternalJobRef != "job-ref-001" ||
		entry.RunRef != "run-ref-001" ||
		entry.ChangeRef != "change-ref-001" {
		t.Fatalf("entry=%+v found=%v", entry, foundOPES)
	}
}

func TestExternalBridgeInputLedgerRejectsEmptySystemOrJobRefV0(t *testing.T) {
	ctx := context.Background()
	ledger, err := newFileExternalBridgeInputLedgerV0(
		filepath.Join(t.TempDir(), "external-bridge-input-ledger.json"),
	)
	if err != nil {
		t.Fatalf("ledger: %v", err)
	}

	if err := externalBridgeRecordSubmittedInputV0(
		ctx,
		ledger,
		"",
		"job-ref-001",
		"run-ref-001",
		"change-ref-001",
	); err == nil || err.Error() != "external_bridge_input_key_required" {
		t.Fatalf("record empty system err=%v", err)
	}
	if err := externalBridgeRecordSubmittedInputV0(
		ctx,
		ledger,
		"opes",
		"",
		"run-ref-001",
		"change-ref-001",
	); err == nil || err.Error() != "external_bridge_input_key_required" {
		t.Fatalf("record empty job err=%v", err)
	}
}
