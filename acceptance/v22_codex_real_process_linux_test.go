//go:build v22_real_e2e && linux

package acceptance_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
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

type v22ProcStat struct {
	PID, PPID, PGID int
	State, Birth    string
}

type v22SleepWitness struct {
	PID, PGID  int
	Birth      string
	Executable string
	Argv       []string
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
	stat, err := v22ReadProcStat(record.PID)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return stat.PID == record.PID && stat.PGID == record.PGID && stat.Birth == record.Birth &&
		stat.State != "Z" && stat.State != "X", nil
}

func v22WaitExactSleep(t *testing.T, record v22ProcessRecord, seconds int) v22SleepWitness {
	t.Helper()
	for until := time.Now().Add(45 * time.Second); time.Now().Before(until); time.Sleep(25 * time.Millisecond) {
		witness, found, err := v22FindExactSleep(record, seconds)
		v22Require(t, err == nil, "inspect exact sleep %d for %s: %v", seconds, record.Exec, err)
		if found {
			return witness
		}
	}
	t.Fatalf("no exact /usr/bin/sleep %d descendant for %+v", seconds, record)
	return v22SleepWitness{}
}

func v22RequireExactSleepAlive(t *testing.T, record v22ProcessRecord, witness v22SleepWitness, seconds int) {
	t.Helper()
	leaderAlive, err := v22ProcessAlive(record)
	v22Require(t, err == nil && leaderAlive, "exact leader did not survive server SIGKILL: execution=%s alive=%v err=%v", record.Exec, leaderAlive, err)
	alive, err := v22ExactSleepAlive(record, witness, seconds)
	v22Require(t, err == nil && alive, "exact sleep descendant did not survive server SIGKILL: execution=%s witness=%+v alive=%v err=%v", record.Exec, witness, alive, err)
}

func v22FindExactSleep(record v22ProcessRecord, seconds int) (v22SleepWitness, bool, error) {
	leaderAlive, err := v22ProcessAlive(record)
	if err != nil || !leaderAlive {
		return v22SleepWitness{}, false, err
	}
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return v22SleepWitness{}, false, err
	}
	for _, entry := range entries {
		pid, parseErr := strconv.Atoi(entry.Name())
		if parseErr != nil || pid == record.PID {
			continue
		}
		stat, statErr := v22ReadProcStat(pid)
		if statErr != nil || stat.State == "Z" || stat.State == "X" || stat.PGID != record.PGID {
			continue
		}
		witness, exact, inspectErr := v22InspectExactSleep(record, stat, seconds)
		if inspectErr != nil {
			if errors.Is(inspectErr, os.ErrNotExist) {
				continue
			}
			return v22SleepWitness{}, false, inspectErr
		}
		if exact {
			return witness, true, nil
		}
	}
	return v22SleepWitness{}, false, nil
}

func v22ExactSleepAlive(record v22ProcessRecord, witness v22SleepWitness, seconds int) (bool, error) {
	stat, err := v22ReadProcStat(witness.PID)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if stat.PID != witness.PID || stat.PGID != witness.PGID || stat.Birth != witness.Birth ||
		stat.State == "Z" || stat.State == "X" {
		return false, nil
	}
	got, exact, err := v22InspectExactSleep(record, stat, seconds)
	return exact && got.Executable == witness.Executable &&
		v22EqualStrings(got.Argv, witness.Argv), err
}

func v22InspectExactSleep(record v22ProcessRecord, stat v22ProcStat, seconds int) (v22SleepWitness, bool, error) {
	if stat.PGID != record.PGID {
		return v22SleepWitness{}, false, nil
	}
	descendant, err := v22ProcessDescendsFrom(stat.PID, record.PID)
	if err != nil || !descendant {
		return v22SleepWitness{}, false, err
	}
	executable, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", stat.PID))
	if err != nil {
		return v22SleepWitness{}, false, err
	}
	expectedExecutable, err := filepath.EvalSymlinks("/usr/bin/sleep")
	if err != nil {
		return v22SleepWitness{}, false, err
	}
	if filepath.Clean(executable) != filepath.Clean(expectedExecutable) {
		return v22SleepWitness{}, false, nil
	}
	payload, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", stat.PID))
	if err != nil {
		return v22SleepWitness{}, false, err
	}
	argv, err := v22ParseProcCmdline(payload)
	if err != nil {
		return v22SleepWitness{}, false, nil
	}
	if !v22ExactSleepArgv(expectedExecutable, executable, argv, seconds) {
		return v22SleepWitness{}, false, nil
	}
	return v22SleepWitness{
		PID: stat.PID, PGID: stat.PGID, Birth: stat.Birth,
		Executable: executable, Argv: argv,
	}, true, nil
}

func v22ProcessDescendsFrom(pid, ancestor int) (bool, error) {
	seen := map[int]bool{}
	for current, depth := pid, 0; current > 0 && depth < 64; depth++ {
		if current == ancestor {
			return true, nil
		}
		if seen[current] {
			return false, errors.New("cycle in /proc ancestry")
		}
		seen[current] = true
		stat, err := v22ReadProcStat(current)
		if err != nil {
			return false, err
		}
		if stat.PPID <= 0 || stat.PPID == current {
			return false, nil
		}
		current = stat.PPID
	}
	return false, nil
}

func v22ReadProcStat(pid int) (v22ProcStat, error) {
	payload, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return v22ProcStat{}, err
	}
	return v22ParseProcStat(payload)
}

func v22ParseProcStat(payload []byte) (v22ProcStat, error) {
	raw := string(payload)
	open, end := strings.IndexByte(raw, '('), strings.LastIndexByte(raw, ')')
	if open <= 0 || end <= open {
		return v22ProcStat{}, errors.New("invalid /proc stat")
	}
	pid, err := strconv.Atoi(strings.TrimSpace(raw[:open]))
	if err != nil || pid <= 0 {
		return v22ProcStat{}, errors.New("invalid /proc pid")
	}
	fields := strings.Fields(raw[end+1:])
	if len(fields) <= 19 {
		return v22ProcStat{}, errors.New("short /proc stat")
	}
	ppid, ppidErr := strconv.Atoi(fields[1])
	pgid, pgidErr := strconv.Atoi(fields[2])
	if ppidErr != nil || pgidErr != nil || ppid < 0 || pgid <= 0 || fields[0] == "" || fields[19] == "" {
		return v22ProcStat{}, errors.New("invalid /proc identity")
	}
	return v22ProcStat{PID: pid, PPID: ppid, PGID: pgid, State: fields[0], Birth: fields[19]}, nil
}

func v22ParseProcCmdline(payload []byte) ([]string, error) {
	if len(payload) < 2 || payload[len(payload)-1] != 0 {
		return nil, errors.New("invalid /proc cmdline")
	}
	raw := bytes.Split(payload[:len(payload)-1], []byte{0})
	argv := make([]string, len(raw))
	for index, value := range raw {
		if len(value) == 0 {
			return nil, errors.New("empty /proc argument")
		}
		argv[index] = string(value)
	}
	return argv, nil
}

func v22ExactSleepArgv(expectedExecutable, executable string, argv []string, seconds int) bool {
	return filepath.Clean(executable) == filepath.Clean(expectedExecutable) && len(argv) == 2 &&
		(argv[0] == "sleep" || argv[0] == "/usr/bin/sleep" || argv[0] == executable) &&
		argv[1] == strconv.Itoa(seconds)
}

func v22EqualStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func TestV22ExactSleepIdentityParsersAreStrict(t *testing.T) {
	fields := make([]string, 20)
	for index := range fields {
		fields[index] = "0"
	}
	fields[0], fields[1], fields[2], fields[19] = "S", "4000", "4242", "123456"
	stat, err := v22ParseProcStat([]byte("4242 (sleep (fixture)) " + strings.Join(fields, " ")))
	if err != nil || stat != (v22ProcStat{PID: 4242, PPID: 4000, PGID: 4242, State: "S", Birth: "123456"}) {
		t.Fatalf("exact stat parse=%+v err=%v", stat, err)
	}
	for _, invalid := range [][]byte{
		[]byte("4242 sleep S 4000 4242"),
		[]byte("4242 (sleep) S 4000"),
		[]byte("not-a-pid (sleep) " + strings.Join(fields, " ")),
	} {
		if _, err := v22ParseProcStat(invalid); err == nil {
			t.Fatalf("invalid stat accepted: %q", invalid)
		}
	}

	argv, err := v22ParseProcCmdline([]byte("sleep\x00180\x00"))
	if err != nil || !v22EqualStrings(argv, []string{"sleep", "180"}) {
		t.Fatalf("exact cmdline parse=%q err=%v", argv, err)
	}
	for _, invalid := range [][]byte{
		[]byte("sleep\x00180"),
		[]byte("sleep\x00\x00180\x00"),
	} {
		if _, err := v22ParseProcCmdline(invalid); err == nil {
			t.Fatalf("invalid cmdline accepted: %q", invalid)
		}
	}
	for _, candidate := range []struct {
		expected   string
		executable string
		argv       []string
		seconds    int
	}{
		{"/resolved/sleep", "/resolved/sleep", []string{"sleep 180"}, 180},
		{"/resolved/sleep", "/resolved/sleep", []string{"sleep", "60"}, 180},
		{"/resolved/sleep", "/tmp/sleep", []string{"sleep", "180"}, 180},
		{"/resolved/sleep", "/resolved/sleep", []string{"wrapper", "180"}, 180},
	} {
		if v22ExactSleepArgv(candidate.expected, candidate.executable, candidate.argv, candidate.seconds) {
			t.Fatalf("inexact sleep identity accepted: %+v", candidate)
		}
	}
	if !v22ExactSleepArgv("/resolved/sleep", "/resolved/sleep", []string{"sleep", "180"}, 180) ||
		!v22ExactSleepArgv("/resolved/sleep", "/resolved/sleep", []string{"/usr/bin/sleep", "60"}, 60) {
		t.Fatal("exact sleep identity rejected")
	}
}

func TestV22ExactSleepIdentityBindsOwnedProcessTree(t *testing.T) {
	command := exec.Command("/bin/sh", "-c", "sleep 3 & wait")
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
		_ = command.Wait()
	}()
	stat, err := v22ReadProcStat(command.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	boot, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	if err != nil {
		t.Fatal(err)
	}
	record := v22ProcessRecord{
		Exec: "execution:v22-sleep-identity", PID: stat.PID, PGID: stat.PGID,
		Boot: strings.TrimSpace(string(boot)), Birth: stat.Birth,
	}
	var witness v22SleepWitness
	var found bool
	for until := time.Now().Add(2 * time.Second); time.Now().Before(until); time.Sleep(10 * time.Millisecond) {
		witness, found, err = v22FindExactSleep(record, 3)
		if err != nil {
			t.Fatal(err)
		}
		if found {
			break
		}
	}
	if !found {
		t.Fatal("owned exact sleep descendant not found")
	}
	alive, err := v22ExactSleepAlive(record, witness, 3)
	if err != nil || !alive {
		t.Fatalf("owned exact sleep witness not alive: witness=%+v alive=%v err=%v", witness, alive, err)
	}
	tampered := witness
	tampered.Birth += "0"
	if alive, err := v22ExactSleepAlive(record, tampered, 3); err != nil || alive {
		t.Fatalf("tampered birth accepted: alive=%v err=%v", alive, err)
	}
	wrongGroup := record
	wrongGroup.PGID++
	if _, found, err := v22FindExactSleep(wrongGroup, 3); err != nil || found {
		t.Fatalf("wrong process group accepted: found=%v err=%v", found, err)
	}
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

type v22StopRequestEvidence struct {
	SchemaVersion  int       `json:"schema_version"`
	RequestHash    string    `json:"request_hash"`
	IdempotencyKey string    `json:"idempotency_key"`
	Mode           string    `json:"mode"`
	RequestedAt    time.Time `json:"requested_at"`
}

type v22StopIntentEvidence struct {
	SchemaVersion  int       `json:"schema_version"`
	RequestHash    string    `json:"request_hash"`
	IdempotencyKey string    `json:"idempotency_key"`
	Mode           string    `json:"mode"`
	Sequence       uint64    `json:"sequence"`
	PreparedAt     time.Time `json:"prepared_at"`
}

type v22StopSignalEvidence struct {
	SchemaVersion  int       `json:"schema_version"`
	RequestHash    string    `json:"request_hash"`
	IdempotencyKey string    `json:"idempotency_key"`
	Mode           string    `json:"mode"`
	Sequence       uint64    `json:"sequence"`
	SignaledAt     time.Time `json:"signaled_at"`
}

type v22StopCompletionEvidence struct {
	SchemaVersion  int       `json:"schema_version"`
	RequestHash    string    `json:"request_hash"`
	IdempotencyKey string    `json:"idempotency_key"`
	Mode           string    `json:"mode"`
	Sequence       uint64    `json:"sequence"`
	ObservedAt     time.Time `json:"observed_at"`
}

type v22StoppedTerminalEvidence struct {
	SchemaVersion       int       `json:"schema_version"`
	RequestHash         string    `json:"request_hash"`
	Status              string    `json:"status"`
	MediaType           string    `json:"media_type,omitempty"`
	Artifact            string    `json:"artifact,omitempty"`
	ErrorCode           string    `json:"error_code,omitempty"`
	ObservedAt          time.Time `json:"observed_at"`
	Diagnostic          []byte    `json:"diagnostic,omitempty"`
	DiagnosticTruncated bool      `json:"diagnostic_truncated,omitempty"`
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
	request, err := v22ReadStrictEvidence[v22StopRequestEvidence](requests[0])
	if err != nil {
		return err
	}
	intent, err := v22ReadStrictEvidence[v22StopIntentEvidence](intents[0])
	if err != nil {
		return err
	}
	signal, err := v22ReadStrictEvidence[v22StopSignalEvidence](signals[0])
	if err != nil {
		return err
	}
	completion, err := v22ReadStrictEvidence[v22StopCompletionEvidence](completions[0])
	if err != nil {
		return err
	}
	terminal, err := v22ReadStrictEvidence[v22StoppedTerminalEvidence](terminals[0])
	if err != nil {
		return err
	}
	stem := v22StopEvidenceStem(request.IdempotencyKey)
	if request.SchemaVersion != 1 || intent.SchemaVersion != 1 ||
		signal.SchemaVersion != 1 || completion.SchemaVersion != 1 ||
		request.RequestHash == "" || request.IdempotencyKey == "" || request.Mode != "cooperative" ||
		request.RequestedAt.IsZero() || intent.PreparedAt.IsZero() ||
		signal.SignaledAt.IsZero() || completion.ObservedAt.IsZero() ||
		filepath.Base(requests[0]) != stem+".request.json" ||
		filepath.Base(intents[0]) != stem+".signal-intent.json" ||
		filepath.Base(signals[0]) != stem+".signal.json" {
		return fmt.Errorf(
			"causal stop chain mismatch: request=%+v intent=%+v signal=%+v completion=%+v",
			request, intent, signal, completion,
		)
	}
	if intent.RequestHash != request.RequestHash || signal.RequestHash != request.RequestHash ||
		completion.RequestHash != request.RequestHash ||
		intent.IdempotencyKey != request.IdempotencyKey || signal.IdempotencyKey != request.IdempotencyKey ||
		completion.IdempotencyKey != request.IdempotencyKey ||
		intent.Mode != request.Mode || signal.Mode != request.Mode || completion.Mode != request.Mode ||
		intent.Sequence != 1 || signal.Sequence != 1 || completion.Sequence != 1 {
		return fmt.Errorf(
			"causal stop fence mismatch: request=%+v intent=%+v signal=%+v completion=%+v",
			request, intent, signal, completion,
		)
	}
	if record.Hash == "" || terminal.SchemaVersion != 5 || terminal.ObservedAt.IsZero() ||
		terminal.RequestHash != record.Hash ||
		terminal.Status != "failed" ||
		terminal.ErrorCode != "codex.execution_stopped" {
		return fmt.Errorf("terminal does not bind stopped process: process=%+v terminal=%+v", record, terminal)
	}
	return nil
}

func v22StopEvidenceStem(idempotencyKey string) string {
	digest := sha256.Sum256([]byte(idempotencyKey))
	return "stop-" + hex.EncodeToString(digest[:])
}

func v22ReadStrictEvidence[T any](path string) (T, error) {
	var value T
	payload, err := os.ReadFile(path)
	if err != nil {
		return value, fmt.Errorf("read %s: %w", filepath.Base(path), err)
	}
	if len(payload) == 0 || len(payload) > 64<<10 {
		return value, fmt.Errorf("invalid evidence size in %s: %d", filepath.Base(path), len(payload))
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return value, fmt.Errorf("decode %s: %w", filepath.Base(path), err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return value, fmt.Errorf("trailing JSON in %s", filepath.Base(path))
	}
	return value, nil
}

func TestV22CooperativeStopEvidenceRequiresReceiptlessCausalChain(t *testing.T) {
	dir := t.TempDir()
	record := v22ProcessRecord{Hash: "sha256:launch", path: filepath.Join(dir, "process.json")}
	now := time.Date(2026, 7, 25, 0, 0, 0, 0, time.UTC)
	request := v22StopRequestEvidence{
		SchemaVersion: 1, RequestHash: "sha256:stop", IdempotencyKey: "stop-key",
		Mode: "cooperative", RequestedAt: now,
	}
	intent := v22StopIntentEvidence{
		SchemaVersion: 1, RequestHash: request.RequestHash, IdempotencyKey: request.IdempotencyKey,
		Mode: request.Mode, Sequence: 1, PreparedAt: now.Add(time.Second),
	}
	signal := v22StopSignalEvidence{
		SchemaVersion: 1, RequestHash: request.RequestHash, IdempotencyKey: request.IdempotencyKey,
		Mode: request.Mode, Sequence: 1, SignaledAt: now.Add(2 * time.Second),
	}
	completion := v22StopCompletionEvidence{
		SchemaVersion: 1, RequestHash: request.RequestHash, IdempotencyKey: request.IdempotencyKey,
		Mode: request.Mode, Sequence: 1, ObservedAt: now.Add(3 * time.Second),
	}
	terminal := v22StoppedTerminalEvidence{
		SchemaVersion: 5, RequestHash: record.Hash, Status: "failed",
		ErrorCode: "codex.execution_stopped", ObservedAt: now.Add(4 * time.Second),
	}
	stem := v22StopEvidenceStem(request.IdempotencyKey)
	requestPath := filepath.Join(dir, stem+".request.json")
	v22WriteEvidenceFixture(t, requestPath, request)
	v22WriteEvidenceFixture(t, filepath.Join(dir, stem+".signal-intent.json"), intent)
	v22WriteEvidenceFixture(t, filepath.Join(dir, stem+".signal.json"), signal)
	v22WriteEvidenceFixture(t, filepath.Join(dir, "stop-completion.json"), completion)
	v22WriteEvidenceFixture(t, filepath.Join(dir, "terminal.json"), terminal)
	if err := v22ValidateCooperativeStopEvidence(record); err != nil {
		t.Fatalf("valid receiptless chain rejected: %v", err)
	}
	receiptPath := filepath.Join(dir, stem+".receipt.json")
	v22WriteEvidenceFixture(t, receiptPath, struct{}{})
	if err := v22ValidateCooperativeStopEvidence(record); err == nil {
		t.Fatal("physical stop receipt accepted for live cooperative chain")
	}
	if err := os.Remove(receiptPath); err != nil {
		t.Fatal(err)
	}
	completion.Sequence = 2
	v22WriteEvidenceFixture(t, filepath.Join(dir, "stop-completion.json"), completion)
	if err := v22ValidateCooperativeStopEvidence(record); err == nil {
		t.Fatal("mismatched stop fence accepted")
	}
	completion.Sequence = 1
	v22WriteEvidenceFixture(t, filepath.Join(dir, "stop-completion.json"), completion)
	v22WriteEvidenceFixture(t, requestPath, map[string]any{
		"schema_version": 1, "request_hash": request.RequestHash,
		"idempotency_key": request.IdempotencyKey, "mode": request.Mode,
		"requested_at": request.RequestedAt, "unknown": true,
	})
	if err := v22ValidateCooperativeStopEvidence(record); err == nil {
		t.Fatal("unknown stop request field accepted")
	}
	v22WriteEvidenceFixture(t, requestPath, request)
	wrongPath := filepath.Join(dir, "stop-wrong.request.json")
	if err := os.Rename(requestPath, wrongPath); err != nil {
		t.Fatal(err)
	}
	if err := v22ValidateCooperativeStopEvidence(record); err == nil {
		t.Fatal("stop request filename not bound to idempotency key")
	}
	if err := os.Rename(wrongPath, requestPath); err != nil {
		t.Fatal(err)
	}
	terminal.SchemaVersion = 4
	v22WriteEvidenceFixture(t, filepath.Join(dir, "terminal.json"), terminal)
	if err := v22ValidateCooperativeStopEvidence(record); err == nil {
		t.Fatal("non-V22 terminal schema accepted")
	}
}

func v22WriteEvidenceFixture(t *testing.T, path string, value any) {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, payload, 0o600); err != nil {
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
