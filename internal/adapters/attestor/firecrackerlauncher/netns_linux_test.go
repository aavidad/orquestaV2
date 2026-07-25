//go:build linux

package firecrackerlauncher

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestNetNamespacePreflightRejectsRegularFilesAndMagicNamespaceLinks(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	regular := filepath.Join(root, "not-nsfs")
	if err := os.WriteFile(regular, []byte("not a namespace"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := openEmptyNetNamespace(
		regular,
		uint32(os.Geteuid()),
		time.Second,
	); ErrorCode(err) != CodeNetworkUnsafe {
		t.Fatalf("regular file accepted: %v", err)
	}
	if _, err := openEmptyNetNamespace(
		"/proc/self/ns/net",
		0,
		time.Second,
	); ErrorCode(err) != CodeNetworkUnsafe {
		t.Fatalf("host magic namespace accepted: %v", err)
	}
}

func TestNetNamespaceProbeNeverChangesParentNamespace(t *testing.T) {
	target, err := os.Open("/proc/self/ns/mnt")
	if err != nil {
		t.Fatal(err)
	}
	defer target.Close()
	before := netNamespaceIdentityForTest(t)
	if err := probeEmptyNetNamespace(target, time.Second); ErrorCode(err) != CodeNetworkUnsafe {
		t.Fatalf("non-network namespace accepted: %v", err)
	}
	after := netNamespaceIdentityForTest(t)
	if before.Dev != after.Dev || before.Ino != after.Ino {
		t.Fatalf("parent namespace changed: before=%+v after=%+v", before, after)
	}
}

func netNamespaceIdentityForTest(t *testing.T) unix.Stat_t {
	t.Helper()
	var stat unix.Stat_t
	if err := unix.Stat("/proc/self/ns/net", &stat); err != nil {
		t.Fatal(err)
	}
	return stat
}
