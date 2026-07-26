//go:build aix || android || darwin || dragonfly || freebsd || illumos || ios || linux || netbsd || openbsd || solaris

package bootstrap

import (
	"os"
	"syscall"
)

func codexGoToolchainOwnerTrusted(info os.FileInfo) bool {
	metadata, ok := info.Sys().(*syscall.Stat_t)
	return ok && metadata.Uid == 0 && metadata.Gid == 0
}
