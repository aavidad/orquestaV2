//go:build linux

package firecrackerlauncher

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestNetNamespaceMetadataAcceptsSecureNSFSRepresentations(t *testing.T) {
	for _, mode := range []uint32{
		0o400, 0o404, 0o440, 0o444,
		0o600, 0o604, 0o640, 0o644,
	} {
		stat := unix.Stat_t{
			Mode:  unix.S_IFREG | mode,
			Uid:   0,
			Gid:   0,
			Nlink: 1,
		}
		filesystem := unix.Statfs_t{Type: unix.NSFS_MAGIC}
		if !netNamespaceMetadataIsSecure(stat, filesystem, 0) {
			t.Errorf("secure nsfs mode rejected: %04o", mode)
		}
	}
}

func TestNetNamespaceMetadataRejectsOutsideSecureNSFSContract(t *testing.T) {
	baseStat := unix.Stat_t{
		Mode:  unix.S_IFREG | 0o444,
		Uid:   0,
		Gid:   0,
		Nlink: 1,
	}
	baseFilesystem := unix.Statfs_t{Type: unix.NSFS_MAGIC}
	tests := map[string]struct {
		stat       unix.Stat_t
		filesystem unix.Statfs_t
		owner      uint32
	}{
		"filesystem": {
			stat:       baseStat,
			filesystem: unix.Statfs_t{Type: unix.TMPFS_MAGIC},
		},
		"type": {
			stat:       withNetNamespaceMode(baseStat, unix.S_IFDIR|0o444),
			filesystem: baseFilesystem,
		},
		"uid": {
			stat:       withNetNamespaceOwner(baseStat, 1, 0),
			filesystem: baseFilesystem,
		},
		"gid": {
			stat:       withNetNamespaceOwner(baseStat, 0, 1),
			filesystem: baseFilesystem,
		},
		"links": {
			stat:       withAdditionalNetNamespaceLink(baseStat),
			filesystem: baseFilesystem,
		},
		"owner_not_readable": {
			stat:       withNetNamespaceMode(baseStat, unix.S_IFREG|0o200),
			filesystem: baseFilesystem,
		},
		"group_writable": {
			stat:       withNetNamespaceMode(baseStat, unix.S_IFREG|0o460),
			filesystem: baseFilesystem,
		},
		"other_executable": {
			stat:       withNetNamespaceMode(baseStat, unix.S_IFREG|0o445),
			filesystem: baseFilesystem,
		},
		"special_bit": {
			stat:       withNetNamespaceMode(baseStat, unix.S_IFREG|unix.S_ISUID|0o444),
			filesystem: baseFilesystem,
		},
		"unexpected_owner": {
			stat:       baseStat,
			filesystem: baseFilesystem,
			owner:      1,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if netNamespaceMetadataIsSecure(
				test.stat,
				test.filesystem,
				test.owner,
			) {
				t.Fatal("unsafe netns metadata accepted")
			}
		})
	}
}

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

func withNetNamespaceMode(stat unix.Stat_t, mode uint32) unix.Stat_t {
	stat.Mode = mode
	return stat
}

func withNetNamespaceOwner(stat unix.Stat_t, uid, gid uint32) unix.Stat_t {
	stat.Uid = uid
	stat.Gid = gid
	return stat
}

func withAdditionalNetNamespaceLink(stat unix.Stat_t) unix.Stat_t {
	stat.Nlink++
	return stat
}
