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
		var record v22ProcessRecord
		if payload, readErr := os.ReadFile(path); readErr != nil {
			return readErr
		} else if decodeErr := json.Unmarshal(payload, &record); decodeErr != nil {
			return decodeErr
		}
		record.path = path
		if record.Schema != 1 || record.Exec == "" || record.Hash == "" || record.Scope == "" ||
			record.PID <= 0 || record.PGID <= 0 || record.Boot == "" || record.Birth == "" {
			return fmt.Errorf("invalid process identity in %s", path)
		}
		if _, duplicate := found[record.Exec]; duplicate {
			return fmt.Errorf("duplicate process identity for %s", record.Exec)
		}
		found[record.Exec] = record
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return found
}

func v22WaitProcess(t *testing.T, root, execution string) v22ProcessRecord {
	t.Helper()
	until := time.Now().Add(20 * time.Second)
	for time.Now().Before(until) {
		if record, ok := v22ProcessRecords(t, root)[execution]; ok {
			if live, err := v22ProcessAlive(record); err != nil {
				t.Fatal(err)
			} else if live {
				return record
			}
		}
		time.Sleep(50 * time.Millisecond)
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
	return err == nil && pgid == record.PGID && fields[19] == record.Birth &&
		fields[0] != "Z" && fields[0] != "X", err
}

func v22AssertProcessGone(t *testing.T, record v22ProcessRecord) {
	t.Helper()
	for until := time.Now().Add(20 * time.Second); time.Now().Before(until); time.Sleep(50 * time.Millisecond) {
		if live, err := v22ProcessGroupAlive(record); err != nil {
			t.Fatal(err)
		} else if !live {
			return
		}
	}
	t.Fatalf("exact process survives: %+v", record)
}

func v22AssertStopEvidence(t *testing.T, record v22ProcessRecord) {
	t.Helper()
	dir := filepath.Dir(record.path)
	receipts, _ := filepath.Glob(filepath.Join(dir, "stop-*.receipt.json"))
	signals, _ := filepath.Glob(filepath.Join(dir, "stop-*.signal.json"))
	completions, _ := filepath.Glob(filepath.Join(dir, "stop-completion.json"))
	if len(receipts) != 1 || len(signals) != 1 || len(completions) != 1 {
		t.Fatalf("stop receipt/signal/completion=%d/%d/%d", len(receipts), len(signals), len(completions))
	}
	var receipt struct {
		Hash   string `json:"request_hash"`
		Status string `json:"status"`
		Ref    string `json:"receipt_ref"`
	}
	var signal, completion struct {
		Hash     string `json:"request_hash"`
		Key      string `json:"idempotency_key"`
		Sequence uint64 `json:"sequence"`
	}
	v22ReadStopJSON(t, receipts[0], &receipt)
	v22ReadStopJSON(t, signals[0], &signal)
	v22ReadStopJSON(t, completions[0], &completion)
	if receipt.Hash == "" || receipt.Ref == "" || (receipt.Status != "stopped" && receipt.Status != "already_stopped") ||
		signal.Hash != receipt.Hash || completion.Hash != receipt.Hash || signal.Key == "" ||
		signal.Key != completion.Key || signal.Sequence == 0 || signal.Sequence != completion.Sequence {
		t.Fatalf("invalid stop receipt/fence: %+v %+v %+v", receipt, signal, completion)
	}
}

func v22ReadStopJSON(t *testing.T, path string, target any) {
	t.Helper()
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(payload, target); err != nil {
		t.Fatal(err)
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
