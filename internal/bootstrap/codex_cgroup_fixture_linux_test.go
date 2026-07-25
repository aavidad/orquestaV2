//go:build linux

package bootstrap

import (
	"crypto/rand"
	"encoding/hex"
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

const testCgroupV2Magic = 0x63677270

var errTestCgroupDelegationUnavailable = errors.New("test cgroup v2 delegation unavailable")

type testCodexCgroupFixture struct {
	original string
	root     string
	control  string
}

func configureTestCodexRuntimeCgroup(t *testing.T, configPath string) {
	t.Helper()
	fixture, err := newTestCodexCgroupFixture()
	if err != nil {
		if errors.Is(err, errTestCgroupDelegationUnavailable) {
			t.Skipf("delegated cgroup v2 unavailable: %v", err)
		}
		t.Fatalf("create delegated Codex cgroup fixture: %v", err)
	}
	t.Cleanup(func() {
		fixture.cleanup(t)
	})
	replaceTestConfigValue(t, configPath, "[runtime.codex]\n",
		"[runtime.codex]\ncgroup_root = "+strconv.Quote(fixture.root)+"\n")
}

func newTestCodexCgroupFixture() (*testCodexCgroupFixture, error) {
	var filesystem syscall.Statfs_t
	if err := syscall.Statfs("/sys/fs/cgroup", &filesystem); err != nil ||
		uint64(filesystem.Type) != testCgroupV2Magic {
		return nil, testCgroupUnavailable("unified hierarchy", err)
	}
	original, err := testCurrentCgroup()
	if err != nil {
		return nil, testCgroupUnavailable("current unified cgroup", err)
	}
	var material [16]byte
	if _, err := rand.Read(material[:]); err != nil {
		return nil, fmt.Errorf("random fixture identity: %w", err)
	}
	root := filepath.Join(original, "orquesta-bootstrap-codex-"+hex.EncodeToString(material[:]))
	control := filepath.Join(root, "control")
	fixture := &testCodexCgroupFixture{original: original, root: root, control: control}
	createdRoot := false
	createdControl := false
	moved := false
	defer func() {
		if moved {
			_ = os.WriteFile(filepath.Join(original, "cgroup.procs"), []byte(strconv.Itoa(os.Getpid())), 0o600)
		}
		if createdControl {
			_ = os.Remove(control)
		}
		if createdRoot {
			_ = os.Remove(root)
		}
	}()
	if err := os.Mkdir(root, 0o700); err != nil {
		return nil, classifyTestCgroupDelegationError("create private Codex root", err)
	}
	createdRoot = true
	if err := os.Chmod(root, 0o700); err != nil {
		return nil, classifyTestCgroupDelegationError("protect private Codex root", err)
	}
	if err := requirePrivateTestCgroup(root); err != nil {
		return nil, err
	}
	if err := os.Mkdir(control, 0o700); err != nil {
		return nil, classifyTestCgroupDelegationError("create Codex control child", err)
	}
	createdControl = true
	if err := os.Chmod(control, 0o700); err != nil {
		return nil, classifyTestCgroupDelegationError("protect Codex control child", err)
	}
	if err := requirePrivateTestCgroup(control); err != nil {
		return nil, err
	}
	for _, path := range []string{
		filepath.Join(root, "cgroup.kill"),
		filepath.Join(root, "cgroup.procs"),
		filepath.Join(control, "cgroup.procs"),
	} {
		file, err := os.OpenFile(path, os.O_WRONLY, 0)
		if err != nil {
			return nil, classifyTestCgroupDelegationError("open delegated control "+path, err)
		}
		if err := file.Close(); err != nil {
			return nil, fmt.Errorf("close delegated control %s: %w", path, err)
		}
	}
	if err := os.WriteFile(
		filepath.Join(control, "cgroup.procs"), []byte(strconv.Itoa(os.Getpid())), 0o600,
	); err != nil {
		return nil, classifyTestCgroupDelegationError("enter Codex control child", err)
	}
	moved = true
	current, err := testCurrentCgroup()
	if err != nil || current != control {
		return nil, fmt.Errorf("Codex control child identity: current=%q want=%q: %w", current, control, err)
	}
	processes, err := os.ReadFile(filepath.Join(root, "cgroup.procs"))
	if err != nil || strings.TrimSpace(string(processes)) != "" {
		return nil, fmt.Errorf("Codex root must be empty: processes=%q: %w", processes, err)
	}
	createdRoot = false
	createdControl = false
	moved = false
	return fixture, nil
}

func (fixture *testCodexCgroupFixture) cleanup(t *testing.T) {
	t.Helper()
	if fixture == nil {
		return
	}
	if err := os.WriteFile(
		filepath.Join(fixture.original, "cgroup.procs"), []byte(strconv.Itoa(os.Getpid())), 0o600,
	); err != nil {
		t.Errorf("restore original test cgroup: %v", err)
		return
	}
	if current, err := testCurrentCgroup(); err != nil || current != fixture.original {
		t.Errorf("restored test cgroup identity: current=%q want=%q err=%v", current, fixture.original, err)
		return
	}
	if populated, err := testCgroupPopulated(fixture.root); err != nil {
		t.Errorf("inspect Codex cgroup cleanup: %v", err)
	} else if populated {
		t.Errorf("Codex runtime left workers in owned cgroup %s", fixture.root)
		if err := os.WriteFile(filepath.Join(fixture.root, "cgroup.kill"), []byte("1"), 0o600); err != nil {
			t.Errorf("drain owned Codex cgroup: %v", err)
		}
		waitTestCgroupEmpty(t, fixture.root)
	}
	entries, err := os.ReadDir(fixture.root)
	if err != nil {
		t.Errorf("list owned Codex cgroup: %v", err)
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if entry.Name() != "control" && !strings.HasPrefix(entry.Name(), "execution-") {
			t.Errorf("unexpected child in owned Codex cgroup: %s", entry.Name())
			continue
		}
		if err := os.Remove(filepath.Join(fixture.root, entry.Name())); err != nil {
			t.Errorf("remove owned Codex cgroup child %s: %v", entry.Name(), err)
		}
	}
	if err := os.Remove(fixture.root); err != nil {
		t.Errorf("remove owned Codex cgroup root: %v", err)
	}
}

func testCurrentCgroup() (string, error) {
	content, err := os.ReadFile("/proc/self/cgroup")
	if err != nil || len(content) == 0 || len(content) > 4096 {
		return "", errors.Join(errors.New("invalid /proc/self/cgroup"), err)
	}
	line := strings.TrimSpace(string(content))
	if !strings.HasPrefix(line, "0::/") || strings.ContainsRune(line, '\n') {
		return "", errors.New("unified cgroup v2 entry required")
	}
	path := filepath.Join("/sys/fs/cgroup", strings.TrimPrefix(line, "0::/"))
	relative, err := filepath.Rel("/sys/fs/cgroup", path)
	if err != nil || relative == "." || relative == ".." ||
		strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("unsafe current cgroup path")
	}
	return path, nil
}

func requirePrivateTestCgroup(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("test cgroup stat unavailable: %s", path)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 ||
		info.Mode().Perm() != 0o700 || int(stat.Uid) != os.Geteuid() {
		return fmt.Errorf("unsafe private test cgroup %s: mode=%v owner=%d", path, info.Mode(), stat.Uid)
	}
	return nil
}

func testCgroupPopulated(path string) (bool, error) {
	content, err := os.ReadFile(filepath.Join(path, "cgroup.events"))
	if err != nil {
		return false, err
	}
	fields := strings.Fields(string(content))
	for index := 0; index+1 < len(fields); index += 2 {
		if fields[index] == "populated" {
			switch fields[index+1] {
			case "0":
				return false, nil
			case "1":
				return true, nil
			default:
				return false, fmt.Errorf("invalid populated event %q", fields[index+1])
			}
		}
	}
	return false, errors.New("missing populated event")
}

func waitTestCgroupEmpty(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		populated, err := testCgroupPopulated(path)
		if err != nil || !populated {
			if err != nil {
				t.Errorf("wait owned Codex cgroup empty: %v", err)
			}
			return
		}
		if !time.Now().Before(deadline) {
			t.Errorf("owned Codex cgroup remained populated: %s", path)
			return
		}
		time.Sleep(time.Millisecond)
	}
}

func classifyTestCgroupDelegationError(operation string, err error) error {
	switch {
	case errors.Is(err, syscall.EACCES),
		errors.Is(err, syscall.EPERM),
		errors.Is(err, syscall.EROFS),
		errors.Is(err, syscall.EOPNOTSUPP),
		errors.Is(err, syscall.ENOENT):
		return testCgroupUnavailable(operation, err)
	default:
		return fmt.Errorf("%s: %w", operation, err)
	}
}

func testCgroupUnavailable(operation string, err error) error {
	return fmt.Errorf("%w: %s: %v", errTestCgroupDelegationUnavailable, operation, err)
}
