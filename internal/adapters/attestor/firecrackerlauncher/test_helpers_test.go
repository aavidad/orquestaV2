package firecrackerlauncher

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

func validConfigForTest(root string) Config {
	uid, gid := uint32(os.Geteuid()), uint32(os.Getegid())
	return Config{
		SocketPath: filepath.Join(root, "launcher.sock"), RuntimeRoot: root,
		FirecrackerCommand: "/usr/local/bin/firecracker", JailerCommand: "/usr/local/bin/jailer",
		FirecrackerSHA256: strings.Repeat("1", 64), JailerSHA256: strings.Repeat("2", 64),
		KernelImage:         "/usr/local/lib/orquesta/firecracker/vmlinux",
		KernelSHA256:        strings.Repeat("3", 64),
		GuestImage:          "/usr/local/lib/orquesta/firecracker/guest.cpio.gz",
		GuestSHA256:         strings.Repeat("4", 64),
		GuestManifest:       "/usr/local/lib/orquesta/firecracker/guest.manifest.json",
		GuestManifestSHA256: strings.Repeat("5", 64),
		NetNSPath:           "/run/netns/orquesta-firecracker-empty",
		CgroupRoot:          "/sys/fs/cgroup", ParentCgroup: "orquesta-firecracker",
		AllowedUID: uid, AllowedGID: gid, JailUID: 65534, JailGID: 65534,
		MaxInputBytes: 1 << 20, MaxOutputDriveBytes: 256 << 10,
		MaxCapturedOutputBytes: 64 << 10,
		MaxMemoryBytes:         1 << 30, MaxPIDs: 256,
		MaxCPUQuotaMicros: 200_000, MaxConcurrentRuns: 2,
		MaxTimeout: 3 * time.Second, CleanupTimeout: time.Second, MaxDiagnosticBytes: 64 << 10,
		MaxCleanupEntries: 1024, MaxCleanupDepth: 16,
	}
}
