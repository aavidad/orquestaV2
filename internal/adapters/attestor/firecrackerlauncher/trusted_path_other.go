//go:build !linux

package firecrackerlauncher

import "os"

func OpenTrustedConfigFile(string, uint32, int64) (*os.File, error) {
	return nil, launcherError(CodeUnavailable)
}
