//go:build !unix

package sqlite

import "os"

func openLocalStateFile(path string) (*os.File, error) {
	return os.Open(path)
}

func localFileIdentity(os.FileInfo) (uint64, uint64, bool) {
	return 0, 0, false
}
