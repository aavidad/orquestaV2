// Este fichero publica una generación no sobrescribible con confirmación atómica.
package main

import (
	"encoding/json"
	"errors"
	"golang.org/x/sys/unix"
	"os"
)

const (
	journalLimit          = int64(64 << 10)
	manifestRecoveryLimit = int64(16 << 20)
)

type publication struct {
	anchor           *outputAnchor
	jsonl            *os.File
	manifest         *os.File
	jsonlStage       string
	manifestStage    string
	jsonlFinal       string
	manifestFinal    string
	journalName      string
	jsonlIdentity    string
	manifestIdentity string
	cut              func(step string) error
}
type journalRecord struct {
	Schema           string `json:"schema"`
	PairID           string `json:"pair_id"`
	JSONLStage       string `json:"jsonl_stage"`
	ManifestStage    string `json:"manifest_stage"`
	JSONLFinal       string `json:"jsonl_final"`
	ManifestFinal    string `json:"manifest_final"`
	JSONLIdentity    string `json:"jsonl_identity"`
	ManifestIdentity string `json:"manifest_identity"`
	JSONLSHA256      string `json:"jsonl_sha256"`
}

func newPublication(opts options, roots []anchoredRoot) (*publication, error) {
	anchor, err := prepareOutputAnchor(opts, roots)
	if err != nil {
		return nil, err
	}
	value := &publication{
		anchor: anchor, jsonlFinal: anchor.jsonlFinal, manifestFinal: anchor.manifestFinal,
		jsonlStage:    "." + anchor.pairID + ".jsonl.stage",
		manifestStage: "." + anchor.pairID + ".manifest.stage",
		journalName:   "." + anchor.pairID + ".journal",
	}
	if err := value.recover(); err != nil {
		return nil, errors.Join(err, anchor.close())
	}
	jsonlExists, jsonlErr := entryExistsAt(anchor.fd(), value.jsonlFinal)
	manifestExists, manifestErr := entryExistsAt(anchor.fd(), value.manifestFinal)
	if jsonlErr != nil || manifestErr != nil {
		return nil, errors.Join(jsonlErr, manifestErr, anchor.close())
	}
	if jsonlExists || manifestExists {
		return nil, errors.Join(errOutputNameExists, anchor.close())
	}
	if opts.expired() {
		return nil, errors.Join(errBudget, anchor.close())
	}
	value.jsonl, err = createAt(anchor.fd(), value.jsonlStage)
	if err == nil {
		value.manifest, err = createAt(anchor.fd(), value.manifestStage)
	}
	if err != nil {
		return nil, errors.Join(err, value.abort())
	}
	value.jsonlIdentity, err = fileIdentity(value.jsonl)
	if err == nil {
		value.manifestIdentity, err = fileIdentity(value.manifest)
	}
	if err != nil {
		return nil, errors.Join(err, value.abort())
	}
	return value, nil
}
func prepareOutputAnchor(opts options, roots []anchoredRoot) (*outputAnchor, error) {
	if opts.expired() {
		return nil, errBudget
	}
	anchor, err := openOutputAnchor(opts.jsonlPath, opts.manifestPath)
	if err != nil {
		return nil, err
	}
	reader := opts.locationReader
	if reader == nil {
		reader, err = newPhysicalLocationReader()
	}
	if err == nil {
		err = verifyPhysicalSeparation(roots, anchor, reader)
	}
	if err == nil && opts.expired() {
		err = errBudget
	}
	if err == nil {
		anchor.lock, err = anchor.openLock()
	}
	if err != nil {
		return nil, errors.Join(err, anchor.close())
	}
	return anchor, nil
}
func createAt(directoryFD int, name string) (*os.File, error) {
	fd, err := unix.Openat(
		directoryFD, name,
		unix.O_RDWR|unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600,
	)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), name), nil
}
func (publication *publication) prepareManifest(value manifest) error {
	sealed, err := sealManifest(value)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(publication.manifest)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	return encoder.Encode(sealed)
}
func (publication *publication) publish() error {
	if err := publication.jsonl.Sync(); err != nil {
		return err
	}
	if err := publication.manifest.Sync(); err != nil {
		return err
	}
	if err := publication.verifyStages(); err != nil {
		return err
	}
	jsonlDigest, err := digestOpenFile(publication.jsonl)
	if err != nil {
		return err
	}
	journal := journalRecord{
		Schema: "legacy_physical_inventory_journal/v1", PairID: publication.anchor.pairID,
		JSONLStage: publication.jsonlStage, ManifestStage: publication.manifestStage,
		JSONLFinal: publication.jsonlFinal, ManifestFinal: publication.manifestFinal,
		JSONLIdentity: publication.jsonlIdentity, ManifestIdentity: publication.manifestIdentity,
		JSONLSHA256: jsonlDigest,
	}
	if err := publication.writeJournal(journal); err != nil {
		return err
	}
	if err := publication.cutAt("journal"); err != nil {
		return publication.failAndRecover(err)
	}
	if err := publication.verifyStage(publication.jsonlStage, publication.jsonlIdentity); err != nil {
		return publication.failAndRecover(err)
	}
	if err := renameNoReplace(publication.anchor.fd(), publication.jsonlStage, publication.jsonlFinal); err != nil {
		return publication.failAndRecover(err)
	}
	if err := publication.anchor.sync(); err != nil {
		return publication.failAndRecover(err)
	}
	if err := publication.cutAt("jsonl"); err != nil {
		return publication.failAndRecover(err)
	}
	if err := publication.verifyStage(publication.manifestStage, publication.manifestIdentity); err != nil {
		return publication.failAndRecover(err)
	}
	if err := renameNoReplace(publication.anchor.fd(), publication.manifestStage, publication.manifestFinal); err != nil {
		return publication.failAndRecover(err)
	}
	if err := publication.anchor.sync(); err != nil {
		return publication.failAndRecover(err)
	}
	if err := publication.cutAt("manifest"); err != nil {
		return publication.finishCommitted(err)
	}
	return publication.finishCommitted(nil)
}
func (publication *publication) verifyStages() error {
	for _, item := range []struct {
		name, identity string
	}{
		{publication.jsonlStage, publication.jsonlIdentity},
		{publication.manifestStage, publication.manifestIdentity},
	} {
		identity, err := identityAt(publication.anchor.fd(), item.name)
		if err != nil || identity != item.identity {
			return errStageReplaced
		}
	}
	return nil
}
func (publication *publication) verifyStage(name, expected string) error {
	identity, err := identityAt(publication.anchor.fd(), name)
	if err != nil || identity != expected {
		return errStageReplaced
	}
	return nil
}
func (publication *publication) cutAt(step string) error {
	if publication.cut == nil {
		return nil
	}
	return publication.cut(step)
}
func (publication *publication) failAndRecover(cause error) error {
	if recoveryErr := publication.recover(); recoveryErr != nil {
		return &recoveryError{cause: cause, recovery: recoveryErr, journal: publication.journalName}
	}
	return cause
}
func (publication *publication) finishCommitted(cause error) error {
	if err := publication.verifyCommitted(nil); err != nil {
		return errors.Join(cause, err)
	}
	return errors.Join(cause, publication.anchor.sync())
}
func renameNoReplace(directoryFD int, oldName, newName string) error {
	return unix.Renameat2(directoryFD, oldName, directoryFD, newName, unix.RENAME_NOREPLACE)
}
func (publication *publication) abort() error {
	var failures []error
	if publication.jsonl != nil {
		failures = appendIfError(failures, publication.jsonl.Close())
		publication.jsonl = nil
	}
	if publication.manifest != nil {
		failures = appendIfError(failures, publication.manifest.Close())
		publication.manifest = nil
	}
	if publication.anchor != nil {
		failures = appendIfError(failures, publication.anchor.close())
		publication.anchor = nil
	}
	return errors.Join(failures...)
}
