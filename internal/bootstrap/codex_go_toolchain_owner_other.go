//go:build !aix && !android && !darwin && !dragonfly && !freebsd && !illumos && !ios && !linux && !netbsd && !openbsd && !solaris && !windows

package bootstrap

import "os"

func codexGoToolchainOwnerTrusted(os.FileInfo) bool {
	return false
}
