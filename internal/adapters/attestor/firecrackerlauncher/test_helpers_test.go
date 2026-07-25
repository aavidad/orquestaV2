package firecrackerlauncher

import (
	"os"
	"path/filepath"
	"time"
)

func validConfigForTest(root string) Config {
	uid, gid := uint32(os.Geteuid()), uint32(os.Getegid())
	return Config{
		SocketPath: filepath.Join(root, "launcher.sock"), RuntimeRoot: root,
		FirecrackerCommand: "/usr/local/bin/firecracker", JailerCommand: "/usr/local/bin/jailer",
		KernelImage: "/usr/local/lib/orquesta/firecracker/vmlinux",
		GuestImage:  "/usr/local/lib/orquesta/firecracker/guest.cpio.gz",
		CgroupRoot:  "/sys/fs/cgroup", ParentCgroup: "orquesta-firecracker",
		AllowedUID: uid, AllowedGID: gid, JailUID: 65534, JailGID: 65534,
		MaxInputBytes: 1 << 20, MaxOutputDriveBytes: 256 << 10,
		MaxCapturedOutputBytes: 64 << 10,
		MaxMemoryBytes:         1 << 30, MaxPIDs: 256,
		MaxCPUQuotaMicros: 200_000, MaxConcurrentRuns: 2,
		MaxTimeout: 3 * time.Second, CleanupTimeout: time.Second, MaxDiagnosticBytes: 64 << 10,
	}
}
