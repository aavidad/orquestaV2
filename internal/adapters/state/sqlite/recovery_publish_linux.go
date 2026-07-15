//go:build linux

package sqlite

import (
	"errors"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

// publishNoReplace atomically moves one entry through the persistently locked
// dirfd. A renamed/replaced pathname cannot redirect publication.
func (locks *recoveryRootLocks) publishNoReplace(rootPath, stageName, targetName string) error {
	if !validRecoveryEntryName(stageName) || !validRecoveryEntryName(targetName) {
		return errors.New("sqlite.recovery_publish_name_invalid")
	}
	if err := locks.verify(rootPath); err != nil {
		return err
	}
	root := locks.roots[rootPath]
	descriptor := int(root.file.Fd())
	return unix.Renameat2(descriptor, stageName, descriptor, targetName, unix.RENAME_NOREPLACE)
}

func validRecoveryEntryName(name string) bool {
	return name != "" && name != "." && name != ".." && filepath.Base(name) == name &&
		!strings.ContainsAny(name, "/\\\x00")
}
