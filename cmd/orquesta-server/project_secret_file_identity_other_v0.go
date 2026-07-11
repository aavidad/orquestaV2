//go:build !unix

package main

import "os"

func serverProjectSecretFileOwnerUIDV0(info os.FileInfo) (int, bool) {
	return 0, false
}
