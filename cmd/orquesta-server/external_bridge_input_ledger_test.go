package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExternalBridgeInputLedgerSubmittedEntryIsScopedBySystemV0(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	name := "external-bridge-input-ledger.json"
	ledger, err := newFileExternalBridgeInputLedgerV0(
		filepath.Join(dir, name),
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
	assertCommandDurableFilePolicyV0(t, dir, name)
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

func TestExternalBridgeInputLedgerLimitaLecturaSnapshotV0(t *testing.T) {
	path := filepath.Join(t.TempDir(), "external-bridge-input-ledger.json")
	if err := os.WriteFile(path, []byte(strings.Repeat("x", int(externalBridgeInputLedgerMaxBytesV0)+1)), 0o600); err != nil {
		t.Fatalf("write oversized ledger: %v", err)
	}
	ledger, err := newFileExternalBridgeInputLedgerV0(path)
	if err != nil {
		t.Fatalf("ledger: %v", err)
	}
	_, _, err = ledger.LookupExternalBridgeInputV0(context.Background(), "opes:job-ref-001")
	if err == nil || err.Error() != "external_bridge_input_ledger_size_limit_exceeded" {
		t.Fatalf("err=%v", err)
	}
}

func TestExternalBridgeInputLedgerLimitaRecordsSnapshotV0(t *testing.T) {
	path := filepath.Join(t.TempDir(), "external-bridge-input-ledger.json")
	entries := make([]string, 0, externalBridgeInputLedgerMaxRecordsV0+1)
	for index := 0; index <= externalBridgeInputLedgerMaxRecordsV0; index++ {
		entries = append(entries, `{"key":"opes:job-ref-`+string(rune('a'+index%26))+`"}`)
	}
	body := `{"entries":[` + strings.Join(entries, ",") + `]}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write ledger: %v", err)
	}
	ledger, err := newFileExternalBridgeInputLedgerV0(path)
	if err != nil {
		t.Fatalf("ledger: %v", err)
	}
	_, _, err = ledger.LookupExternalBridgeInputV0(context.Background(), "opes:job-ref-a")
	if err == nil || err.Error() != "external_bridge_input_ledger_records_limit_exceeded" {
		t.Fatalf("err=%v", err)
	}
}

func assertCommandDurableFilePolicyV0(t *testing.T, dir string, name string) {
	t.Helper()
	info, err := os.Stat(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("stat durable file: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("durable file mode=%#o", got)
	}
	leftovers, err := filepath.Glob(filepath.Join(dir, "."+name+".*.tmp"))
	if err != nil {
		t.Fatalf("glob temp: %v", err)
	}
	if len(leftovers) != 0 {
		t.Fatalf("temps persistidos=%v", leftovers)
	}
	if _, err := os.Stat(filepath.Join(dir, name+".tmp")); !os.IsNotExist(err) {
		t.Fatalf("temp fijo presente err=%v", err)
	}
}
