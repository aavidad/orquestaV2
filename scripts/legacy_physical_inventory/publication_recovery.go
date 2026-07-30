// Este fichero recupera cortes de publicación usando un diario privado y acotado.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"golang.org/x/sys/unix"
	"os"
)

func (publication *publication) writeJournal(value journalRecord) error {
	tempName := publication.journalName + ".tmp"
	file, err := createAt(publication.anchor.fd(), tempName)
	if err != nil {
		return errors.Join(errRecoveryConflict, err)
	}
	encoder := json.NewEncoder(file)
	if err := encoder.Encode(value); err != nil {
		return errors.Join(err, file.Close())
	}
	if err := file.Sync(); err != nil {
		return errors.Join(err, file.Close())
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := renameNoReplace(publication.anchor.fd(), tempName, publication.journalName); err != nil {
		return err
	}
	return publication.anchor.sync()
}
func (publication *publication) recover() error {
	exists, err := entryExistsAt(publication.anchor.fd(), publication.journalName)
	if err != nil {
		return err
	}
	if !exists {
		return publication.rejectUnconfirmedArtifacts()
	}
	journal, err := publication.readJournal()
	if err != nil {
		return errors.Join(errRecoveryConflict, err)
	}
	if !publication.matchesJournal(journal) {
		return errRecoveryConflict
	}
	manifestExists, err := entryExistsAt(publication.anchor.fd(), publication.manifestFinal)
	if err != nil {
		return err
	}
	if manifestExists {
		if err := publication.verifyCommitted(&journal); err != nil {
			return errors.Join(errRecoveryConflict, err)
		}
		return nil
	}
	return errRecoveryConflict
}
func (publication *publication) readJournal() (journalRecord, error) {
	file, err := openRegularAt(publication.anchor.fd(), publication.journalName)
	if err != nil {
		return journalRecord{}, err
	}
	content, readErr := readLimited(file, journalLimit)
	closeErr := file.Close()
	if readErr != nil || closeErr != nil {
		return journalRecord{}, errors.Join(readErr, closeErr)
	}
	var value journalRecord
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return journalRecord{}, err
	}
	if err := requireJSONEOF(decoder); err != nil {
		return journalRecord{}, err
	}
	return value, nil
}
func (publication *publication) matchesJournal(value journalRecord) bool {
	return value.Schema == "legacy_physical_inventory_journal/v1" &&
		value.PairID == publication.anchor.pairID &&
		value.JSONLStage == publication.jsonlStage &&
		value.ManifestStage == publication.manifestStage &&
		value.JSONLFinal == publication.jsonlFinal &&
		value.ManifestFinal == publication.manifestFinal
}
func (publication *publication) verifyCommitted(expected *journalRecord) error {
	manifestFile, err := openRegularAt(publication.anchor.fd(), publication.manifestFinal)
	if err != nil {
		return err
	}
	if expected != nil {
		identity, identityErr := fileIdentity(manifestFile)
		if identityErr != nil || identity != expected.ManifestIdentity {
			return errors.Join(errRecoveryConflict, identityErr, manifestFile.Close())
		}
	}
	content, readErr := readLimited(manifestFile, manifestRecoveryLimit)
	closeErr := manifestFile.Close()
	if readErr != nil || closeErr != nil {
		return errors.Join(readErr, closeErr)
	}
	jsonl, err := openRegularAt(publication.anchor.fd(), publication.jsonlFinal)
	if err != nil {
		return err
	}
	if expected != nil {
		identity, identityErr := fileIdentity(jsonl)
		digest, digestErr := digestOpenFile(jsonl)
		if identityErr != nil || digestErr != nil ||
			identity != expected.JSONLIdentity || digest != expected.JSONLSHA256 {
			return errors.Join(errRecoveryConflict, identityErr, digestErr, jsonl.Close())
		}
	}
	verifyErr := verifyPublishedPair(content, jsonl, publication.jsonlFinal)
	closeErr = jsonl.Close()
	return errors.Join(verifyErr, closeErr)
}
func (publication *publication) rejectUnconfirmedArtifacts() error {
	for _, name := range []string{
		publication.jsonlStage,
		publication.manifestStage,
		publication.journalName + ".tmp",
		publication.jsonlFinal,
		publication.manifestFinal,
	} {
		exists, err := entryExistsAt(publication.anchor.fd(), name)
		if err != nil {
			return err
		}
		if exists {
			return errRecoveryConflict
		}
	}
	return nil
}
func openRegularAt(directoryFD int, name string) (*os.File, error) {
	fd, err := unix.Openat(directoryFD, name, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), name)
	var stat unix.Stat_t
	if err := unix.Fstat(fd, &stat); err != nil {
		return nil, errors.Join(err, file.Close())
	}
	if stat.Mode&unix.S_IFMT != unix.S_IFREG || stat.Uid != uint32(os.Geteuid()) {
		return nil, errors.Join(errRecoveryConflict, file.Close())
	}
	return file, nil
}
func entryExistsAt(directoryFD int, name string) (bool, error) {
	var stat unix.Stat_t
	err := unix.Fstatat(directoryFD, name, &stat, unix.AT_SYMLINK_NOFOLLOW)
	if errors.Is(err, unix.ENOENT) {
		return false, nil
	}
	return err == nil, err
}
func fileIdentity(file *os.File) (string, error) {
	var stat unix.Stat_t
	if err := unix.Fstat(int(file.Fd()), &stat); err != nil {
		return "", err
	}
	return localIdentity(&stat), nil
}
func identityAt(directoryFD int, name string) (string, error) {
	var stat unix.Stat_t
	if err := unix.Fstatat(directoryFD, name, &stat, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return "", err
	}
	return localIdentity(&stat), nil
}
func appendIfError(values []error, err error) []error {
	if err != nil {
		return append(values, err)
	}
	return values
}
