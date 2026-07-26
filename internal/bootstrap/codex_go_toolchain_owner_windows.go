//go:build windows

package bootstrap

import "os"

func codexGoToolchainOwnerTrusted(os.FileInfo) bool {
	return true
}
