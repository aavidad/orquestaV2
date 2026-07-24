//go:build v22_real_e2e && linux

package acceptance_test

import (
	"encoding/json"
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
	Schema int    `json:"schema_version"`
	Exec   string `json:"execution_ref"`
	Hash   string `json:"request_hash"`
	Scope  string `json:"runtime_scope"`
	PID    int    `json:"pid"`
	PGID   int    `json:"pgid"`
	Boot   string `json:"boot_id"`
	Birth  string `json:"birth_marker"`
	path   string
}

func v22ProcessRecords(t *testing.T, root string) map[string]v22ProcessRecord {
	t.Helper()
	found := map[string]v22ProcessRecord{}
	err := filepath.WalkDir(filepath.Join(root, "work"), func(path string, entry os.DirEntry, err error) error {
		if errors.Is(err, os.ErrNotExist) {
			return filepath.SkipDir
		}
		if err != nil || entry.IsDir() || entry.Name() != "process.json" {
			return err
		}
		record := v22Decode[v22ProcessRecord](path)
		record.path = path
		if record.Schema != 1 || record.Exec == "" || record.Hash == "" || record.Scope == "" || record.PID <= 0 || record.PGID <= 0 || record.Boot == "" || record.Birth == "" {
			return fmt.Errorf("invalid process identity in %s", path)
		}
		if _, duplicate := found[record.Exec]; duplicate {
			return fmt.Errorf("duplicate process identity for %s", record.Exec)
		}
		found[record.Exec] = record
		return nil
	})
	v22Require(t, err == nil, "process census: %v", err)
	return found
}

func v22WaitProcess(t *testing.T, root, execution string) v22ProcessRecord {
	t.Helper()
	for until := time.Now().Add(20 * time.Second); time.Now().Before(until); time.Sleep(50 * time.Millisecond) {
		if record, ok := v22ProcessRecords(t, root)[execution]; ok {
			live, err := v22ProcessAlive(record)
			v22Require(t, err == nil, "process liveness: %v", err)
			if live {
				return record
			}
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
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
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
		live, err := v22ProcessGroupAlive(record)
		v22Require(t, err == nil, "process group liveness: %v", err)
		if !live {
			return
		}
	}
	t.Fatalf("exact process survives: %+v", record)
}

func v22AssertCooperativeStopEvidence(t *testing.T, record v22ProcessRecord) {
	t.Helper()
	if err := v22ValidateCooperativeStopEvidence(record); err != nil {
		t.Fatalf("invalid cooperative stop evidence: %v", err)
	}
}

func v22ValidateCooperativeStopEvidence(record v22ProcessRecord) error {
	dir := filepath.Dir(record.path)
	requests, _ := filepath.Glob(filepath.Join(dir, "stop-*.request.json"))
	intents, _ := filepath.Glob(filepath.Join(dir, "stop-*.signal-intent.json"))
	signals, _ := filepath.Glob(filepath.Join(dir, "stop-*.signal.json"))
	completions, _ := filepath.Glob(filepath.Join(dir, "stop-completion.json"))
	terminals, _ := filepath.Glob(filepath.Join(dir, "terminal.json"))
	receipts, _ := filepath.Glob(filepath.Join(dir, "stop-*.receipt.json"))
	if len(requests) != 1 || len(intents) != 1 || len(signals) != 1 ||
		len(completions) != 1 || len(terminals) != 1 || len(receipts) != 0 {
		return fmt.Errorf(
			"request/intent/signal/completion/terminal/receipt=%d/%d/%d/%d/%d/%d, want 1/1/1/1/1/0",
			len(requests), len(intents), len(signals), len(completions), len(terminals), len(receipts),
		)
	}
	request, err := v22ReadEvidenceObject(requests[0])
	if err != nil {
		return err
	}
	intent, err := v22ReadEvidenceObject(intents[0])
	if err != nil {
		return err
	}
	signal, err := v22ReadEvidenceObject(signals[0])
	if err != nil {
		return err
	}
	completion, err := v22ReadEvidenceObject(completions[0])
	if err != nil {
		return err
	}
	terminal, err := v22ReadEvidenceObject(terminals[0])
	if err != nil {
		return err
	}
	hash, key, mode := request.text("request_hash"), request.text("idempotency_key"), request.text("mode")
	sequence := intent.number("sequence")
	if hash == "" || key == "" || mode != "cooperative" || sequence == 0 ||
		intent.text("request_hash") != hash || signal.text("request_hash") != hash || completion.text("request_hash") != hash ||
		intent.text("idempotency_key") != key || signal.text("idempotency_key") != key || completion.text("idempotency_key") != key ||
		intent.text("mode") != mode || signal.text("mode") != mode || completion.text("mode") != mode ||
		signal.number("sequence") != sequence || completion.number("sequence") != sequence {
		return fmt.Errorf(
			"causal stop chain mismatch: request=%+v intent=%+v signal=%+v completion=%+v",
			request, intent, signal, completion,
		)
	}
	if terminal.text("request_hash") != record.Hash ||
		terminal.text("status") != "failed" ||
		terminal.text("error_code") != "codex.execution_stopped" {
		return fmt.Errorf("terminal does not bind stopped process: process=%+v terminal=%+v", record, terminal)
	}
	return nil
}

func v22ReadEvidenceObject(path string) (v22Object, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", filepath.Base(path), err)
	}
	var object v22Object
	if err := json.Unmarshal(payload, &object); err != nil {
		return nil, fmt.Errorf("decode %s: %w", filepath.Base(path), err)
	}
	return object, nil
}

func TestV22CooperativeStopEvidenceRequiresReceiptlessCausalChain(t *testing.T) {
	dir := t.TempDir()
	record := v22ProcessRecord{Hash: "sha256:launch", path: filepath.Join(dir, "process.json")}
	fixtures := map[string]v22Object{
		"stop-fence.request.json":       {"request_hash": "sha256:stop", "idempotency_key": "stop-key", "mode": "cooperative"},
		"stop-fence.signal-intent.json": {"request_hash": "sha256:stop", "idempotency_key": "stop-key", "mode": "cooperative", "sequence": float64(7)},
		"stop-fence.signal.json":        {"request_hash": "sha256:stop", "idempotency_key": "stop-key", "mode": "cooperative", "sequence": float64(7)},
		"stop-completion.json":          {"request_hash": "sha256:stop", "idempotency_key": "stop-key", "mode": "cooperative", "sequence": float64(7)},
		"terminal.json":                 {"request_hash": record.Hash, "status": "failed", "error_code": "codex.execution_stopped"},
	}
	for name, fixture := range fixtures {
		payload, err := json.Marshal(fixture)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), payload, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := v22ValidateCooperativeStopEvidence(record); err != nil {
		t.Fatalf("valid receiptless chain rejected: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "stop-fence.receipt.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := v22ValidateCooperativeStopEvidence(record); err == nil {
		t.Fatal("physical stop receipt accepted for live cooperative chain")
	}
	if err := os.Remove(filepath.Join(dir, "stop-fence.receipt.json")); err != nil {
		t.Fatal(err)
	}
	fixtures["stop-completion.json"]["sequence"] = float64(8)
	payload, err := json.Marshal(fixtures["stop-completion.json"])
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "stop-completion.json"), payload, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := v22ValidateCooperativeStopEvidence(record); err == nil {
		t.Fatal("mismatched stop fence accepted")
	}
}

func v22KillExact(record v22ProcessRecord) {
	if live, _ := v22ProcessGroupAlive(record); live {
		_ = syscall.Kill(-record.PGID, syscall.SIGKILL)
	}
}

func v22ProcessGroupAlive(record v22ProcessRecord) (bool, error) {
	boot, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	if err != nil || strings.TrimSpace(string(boot)) != record.Boot {
		return false, err
	}
	err = syscall.Kill(-record.PGID, 0)
	if err == nil || errors.Is(err, syscall.EPERM) {
		return true, nil
	}
	if errors.Is(err, syscall.ESRCH) {
		return false, nil
	}
	return false, err
}
