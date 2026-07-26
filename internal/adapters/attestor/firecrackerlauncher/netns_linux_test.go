//go:build linux

package firecrackerlauncher

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestIPv4RouteContentAcceptsOnlyEmptyOrCanonicalHeader(t *testing.T) {
	header := strings.Join([]string{
		"Iface", "Destination", "Gateway", "Flags", "RefCnt", "Use",
		"Metric", "Mask", "MTU", "Window", "IRTT",
	}, "\t")
	route := "lo\t00000000\t00000000\t0001\t0\t0\t0\t00000000\t0\t0\t0"
	tests := map[string]struct {
		content string
		safe    bool
	}{
		"empty": {
			safe: true,
		},
		"whitespace_only": {
			content: " \n\t",
			safe:    true,
		},
		"canonical_header": {
			content: header + "\n",
			safe:    true,
		},
		"header_with_kernel_spacing": {
			content: strings.ReplaceAll(header, "\t", "\t ") + " \n",
			safe:    true,
		},
		"route_without_header": {
			content: route + "\n",
		},
		"route_after_header": {
			content: header + "\n" + route + "\n",
		},
		"duplicate_header": {
			content: header + "\n" + header + "\n",
		},
		"header_split_across_lines": {
			content: strings.Replace(header, "\tGateway", "\nGateway", 1),
		},
		"header_prefix_only": {
			content: "Iface",
		},
		"header_name_with_suffix": {
			content: strings.Replace(header, "Iface", "IfaceUnsafe", 1),
		},
		"header_missing_field": {
			content: strings.TrimSuffix(header, "\tIRTT"),
		},
		"header_extra_field": {
			content: header + "\tExtra",
		},
		"header_reordered": {
			content: strings.Replace(header, "Destination\tGateway", "Gateway\tDestination", 1),
		},
		"bounded_input": {
			content: strings.Repeat(" ", maxIPv4RouteFileBytes+1),
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if got := ipv4RouteContentEmpty([]byte(test.content)); got != test.safe {
				t.Fatalf("safe=%t, want %t", got, test.safe)
			}
		})
	}
}

func TestIPv6RouteContentAcceptsOnlyEmptyOrCanonicalRejectSentinels(t *testing.T) {
	zero := strings.Repeat("0", 32)
	route := func(
		destination, destinationPrefix, source, sourcePrefix, nextHop,
		metric, referenceCount, useCount, flags, device string,
	) string {
		return strings.Join([]string{
			destination, destinationPrefix, source, sourcePrefix, nextHop,
			metric, referenceCount, useCount, flags, device,
		}, " ")
	}
	canonical := route(
		zero, "00", zero, "00", zero,
		"ffffffff", "00000001", "00000000", "00200200", "lo",
	)
	tests := map[string]struct {
		content string
		safe    bool
	}{
		"empty": {
			safe: true,
		},
		"whitespace_only": {
			content: " \n\t",
			safe:    true,
		},
		"one_canonical_sentinel": {
			content: canonical + "\n",
			safe:    true,
		},
		"multiple_canonical_sentinels": {
			content: canonical + "\n" + canonical + "\n",
			safe:    true,
		},
		"well_formed_runtime_counters": {
			content: route(
				zero, "00", zero, "00", zero,
				"ffffffff", "abcdef01", "12345678", "00200200", "lo",
			),
			safe: true,
		},
		"destination_nonzero": {
			content: route(
				"1"+zero[1:], "00", zero, "00", zero,
				"ffffffff", "00000001", "00000000", "00200200", "lo",
			),
		},
		"destination_malformed": {
			content: route(
				"g"+zero[1:], "00", zero, "00", zero,
				"ffffffff", "00000001", "00000000", "00200200", "lo",
			),
		},
		"destination_short": {
			content: route(
				zero[1:], "00", zero, "00", zero,
				"ffffffff", "00000001", "00000000", "00200200", "lo",
			),
		},
		"destination_prefix_nonzero": {
			content: route(
				zero, "01", zero, "00", zero,
				"ffffffff", "00000001", "00000000", "00200200", "lo",
			),
		},
		"destination_prefix_malformed": {
			content: route(
				zero, "0g", zero, "00", zero,
				"ffffffff", "00000001", "00000000", "00200200", "lo",
			),
		},
		"source_nonzero": {
			content: route(
				zero, "00", "1"+zero[1:], "00", zero,
				"ffffffff", "00000001", "00000000", "00200200", "lo",
			),
		},
		"source_prefix_nonzero": {
			content: route(
				zero, "00", zero, "01", zero,
				"ffffffff", "00000001", "00000000", "00200200", "lo",
			),
		},
		"next_hop_nonzero": {
			content: route(
				zero, "00", zero, "00", "1"+zero[1:],
				"ffffffff", "00000001", "00000000", "00200200", "lo",
			),
		},
		"metric_noncanonical": {
			content: route(
				zero, "00", zero, "00", zero,
				"fffffffe", "00000001", "00000000", "00200200", "lo",
			),
		},
		"metric_malformed": {
			content: route(
				zero, "00", zero, "00", zero,
				"fffffffz", "00000001", "00000000", "00200200", "lo",
			),
		},
		"reference_count_malformed": {
			content: route(
				zero, "00", zero, "00", zero,
				"ffffffff", "0000000z", "00000000", "00200200", "lo",
			),
		},
		"use_count_wrong_width": {
			content: route(
				zero, "00", zero, "00", zero,
				"ffffffff", "00000001", "0000000", "00200200", "lo",
			),
		},
		"missing_reject_flag": {
			content: route(
				zero, "00", zero, "00", zero,
				"ffffffff", "00000001", "00000000", "00200000", "lo",
			),
		},
		"extra_flag": {
			content: route(
				zero, "00", zero, "00", zero,
				"ffffffff", "00000001", "00000000", "00200201", "lo",
			),
		},
		"flags_malformed": {
			content: route(
				zero, "00", zero, "00", zero,
				"ffffffff", "00000001", "00000000", "0020020z", "lo",
			),
		},
		"other_device": {
			content: route(
				zero, "00", zero, "00", zero,
				"ffffffff", "00000001", "00000000", "00200200", "eth0",
			),
		},
		"missing_field": {
			content: strings.TrimSuffix(canonical, " lo"),
		},
		"extra_field": {
			content: canonical + " extra",
		},
		"blank_record": {
			content: canonical + "\n\n" + canonical,
		},
		"bounded_input": {
			content: strings.Repeat(" ", maxIPv6RouteFileBytes+1),
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if got := ipv6RouteContentSafe([]byte(test.content)); got != test.safe {
				t.Fatalf("safe=%t, want %t", got, test.safe)
			}
		})
	}
}

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
