//go:build unix

package main

import (
	"os"
	"syscall"
)

func serverProjectSecretFileOwnerUIDV0(info os.FileInfo) (int, bool) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, false
	}
	return int(stat.Uid), true
}
