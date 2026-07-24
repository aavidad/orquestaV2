//go:build v22_real_e2e && linux

package acceptance_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

type v22ProcessRecord struct {
	Schema int `json:"schema_version"`; Exec string `json:"execution_ref"`; Hash string `json:"request_hash"`; Scope string `json:"runtime_scope"`
	PID int `json:"pid"`; PGID int `json:"pgid"`; Boot string `json:"boot_id"`; Birth string `json:"birth_marker"`; path string
}

func v22ProcessRecords(t *testing.T, root string) map[string]v22ProcessRecord {
	t.Helper(); found := map[string]v22ProcessRecord{}
	err := filepath.WalkDir(filepath.Join(root, "work"), func(path string, entry os.DirEntry, err error) error {
		if errors.Is(err, os.ErrNotExist) { return filepath.SkipDir }
		if err != nil || entry.IsDir() || entry.Name() != "process.json" { return err }
		record := v22Decode[v22ProcessRecord](path); record.path = path
		if record.Schema != 1 || record.Exec == "" || record.Hash == "" || record.Scope == "" || record.PID <= 0 || record.PGID <= 0 || record.Boot == "" || record.Birth == "" {
			return fmt.Errorf("invalid process identity in %s", path)
		}
		if _, duplicate := found[record.Exec]; duplicate {
			return fmt.Errorf("duplicate process identity for %s", record.Exec)
		}
		found[record.Exec] = record; return nil
	})
	v22Require(t, err == nil, "process census: %v", err)
	return found
}

func v22WaitProcess(t *testing.T, root, execution string) v22ProcessRecord {
	t.Helper()
	for until := time.Now().Add(20 * time.Second); time.Now().Before(until); time.Sleep(50 * time.Millisecond) {
		if record, ok := v22ProcessRecords(t, root)[execution]; ok {
			live, err := v22ProcessAlive(record); v22Require(t, err == nil, "process liveness: %v", err)
			if live { return record }
		}
	}
	t.Fatalf("no live persisted process identity for %s", execution)
	return v22ProcessRecord{}
}

func v22ProcessAlive(record v22ProcessRecord) (bool, error) {
	boot, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	if err != nil || strings.TrimSpace(string(boot)) != record.Boot {
		return false, err
	}
	payload, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", record.PID))
	if errors.Is(err, os.ErrNotExist) { return false, nil }
	if err != nil { return false, err }
	end := strings.LastIndexByte(string(payload), ')')
	if end < 0 {
		return false, errors.New("invalid /proc stat")
	}
	fields := strings.Fields(string(payload[end+1:]))
	if len(fields) <= 19 {
		return false, errors.New("short /proc stat")
	}
	pgid, err := strconv.Atoi(fields[2])
	return err == nil && pgid == record.PGID && fields[19] == record.Birth && fields[0] != "Z" && fields[0] != "X", err
}

func v22AssertProcessGone(t *testing.T, record v22ProcessRecord) {
	t.Helper()
	for until := time.Now().Add(20 * time.Second); time.Now().Before(until); time.Sleep(50 * time.Millisecond) {
		live, err := v22ProcessGroupAlive(record); v22Require(t, err == nil, "process group liveness: %v", err)
		if !live { return }
	}
	t.Fatalf("exact process survives: %+v", record)
}

func v22AssertStopEvidence(t *testing.T, record v22ProcessRecord) {
	t.Helper()
	dir := filepath.Dir(record.path)
	receipts, _ := filepath.Glob(filepath.Join(dir, "stop-*.receipt.json")); signals, _ := filepath.Glob(filepath.Join(dir, "stop-*.signal.json")); completions, _ := filepath.Glob(filepath.Join(dir, "stop-completion.json"))
	v22Require(t, len(receipts) == 1 && len(signals) == 1 && len(completions) == 1, "stop receipt/signal/completion=%d/%d/%d", len(receipts), len(signals), len(completions))
	receipt, signal, completion := v22Decode[v22Object](receipts[0]), v22Decode[v22Object](signals[0]), v22Decode[v22Object](completions[0])
	hash, key, sequence := receipt.text("request_hash"), signal.text("idempotency_key"), signal.number("sequence")
	status := receipt.text("status")
	v22Require(t, hash != "" && receipt.text("receipt_ref") != "" && (status == "stopped" || status == "already_stopped") && signal.text("request_hash") == hash && completion.text("request_hash") == hash && key != "" && key == completion.text("idempotency_key") && sequence > 0 && sequence == completion.number("sequence"),
		"invalid stop receipt/fence: %+v %+v %+v", receipt, signal, completion)
}

func v22KillExact(record v22ProcessRecord) {
	if live, _ := v22ProcessGroupAlive(record); live { _ = syscall.Kill(-record.PGID, syscall.SIGKILL) }
}

func v22ProcessGroupAlive(record v22ProcessRecord) (bool, error) {
	boot, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	if err != nil || strings.TrimSpace(string(boot)) != record.Boot {
		return false, err
	}
	err = syscall.Kill(-record.PGID, 0)
	if err == nil || errors.Is(err, syscall.EPERM) { return true, nil }
	if errors.Is(err, syscall.ESRCH) { return false, nil }
	return false, err
}
