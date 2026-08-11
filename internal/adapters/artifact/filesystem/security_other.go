//go:build !linux

package filesystem

import (
	"os"

	"orquesta/internal/ports"
)

func unsupportedFilesystemError() error {
	return artifactError(ports.ArtifactErrorFilesystemUnsupported, nil)
}

func openPrivateRoot(string, func(*os.File) error) (*os.Root, error) {
	return nil, unsupportedFilesystemError()
}

func (store *Store) ensureDurableDirectory(string) error {
	return unsupportedFilesystemError()
}

func validateOpenPrivateFile(*os.Root, string, *os.File, int64) error {
	return unsupportedFilesystemError()
}

func openPrivateDirectory(*os.Root, string) (*os.File, os.FileInfo, error) {
	return nil, nil, unsupportedFilesystemError()
}

func verifyOpenPrivateDirectory(*os.Root, string, *os.File, os.FileInfo) error {
	return unsupportedFilesystemError()
}

func (store *Store) readVerifiedBlob(string, string, int64) ([]byte, error) {
	return nil, unsupportedFilesystemError()
}

func (store *Store) readPrivateFileBounded(string, int64) ([]byte, error) {
	return nil, unsupportedFilesystemError()
}

func (*Store) publishNoReplace(string, string) (bool, error) {
	return false, unsupportedFilesystemError()
}
