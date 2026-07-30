// Este fichero prueba el punto de confirmación, el bloqueo y la identidad de etapas.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
	"testing"
)

func TestPublishedGenerationIsPrivateAndSealed(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "dato"), "dato", 0o600)
	opts := testOptions(t, root, modeMetadata)
	if err := run(opts); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(opts.jsonlPath)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)
	value := readManifest(t, opts.manifestPath)
	if value.JSONLSHA256 != hex.EncodeToString(sum[:]) ||
		value.JSONLBytes != int64(len(content)) ||
		value.JSONLName != filepath.Base(opts.jsonlPath) {
		t.Fatalf("sello JSONL incorrecto: %#v", value)
	}
	manifestDigest := value.ManifestSHA256
	value.ManifestSHA256 = ""
	unsigned, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	unsignedSum := sha256.Sum256(unsigned)
	if manifestDigest != hex.EncodeToString(unsignedSum[:]) {
		t.Fatalf("sello propio incorrecto: %s", manifestDigest)
	}
	jsonlFile, err := os.Open(opts.jsonlPath)
	if err != nil {
		t.Fatal(err)
	}
	manifestBytes, err := os.ReadFile(opts.manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	verifyErr := verifyPublishedPair(manifestBytes, jsonlFile, filepath.Base(opts.jsonlPath))
	closeErr := jsonlFile.Close()
	if verifyErr != nil || closeErr != nil {
		t.Fatalf("verificador contractual rechazó artefactos válidos: %v / %v", verifyErr, closeErr)
	}
	value.ManifestSHA256 = "0000000000000000000000000000000000000000000000000000000000000000"
	tampered, _ := json.Marshal(value)
	jsonlFile, _ = os.Open(opts.jsonlPath)
	if err := verifyPublishedPair(tampered, jsonlFile, filepath.Base(opts.jsonlPath)); !errors.Is(err, errInvalidManifestSeal) {
		t.Fatalf("el verificador aceptó manifiesto alterado: %v", err)
	}
	_ = jsonlFile.Close()
	for _, path := range []string{opts.jsonlPath, opts.manifestPath} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("%s tiene permisos %o", filepath.Base(path), info.Mode().Perm())
		}
	}
}
func TestVerifierRejectsResealedNonCanonicalContracts(t *testing.T) {
	opts := testOptions(t, t.TempDir(), modeMetadata)
	if err := run(opts); err != nil {
		t.Fatal(err)
	}
	original := readManifest(t, opts.manifestPath)
	mutations := []func(*manifest){
		func(value *manifest) { value.Schema = "legacy_physical_inventory/v0" },
		func(value *manifest) { value.Algorithm = "sha512" },
		func(value *manifest) { value.ManifestDigestDomain = "otro_dominio" },
	}
	for _, mutate := range mutations {
		value := original
		mutate(&value)
		value, err := sealManifest(value)
		if err != nil {
			t.Fatal(err)
		}
		content, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		jsonl, err := os.Open(opts.jsonlPath)
		if err != nil {
			t.Fatal(err)
		}
		verifyErr := verifyPublishedPair(content, jsonl, filepath.Base(opts.jsonlPath))
		if closeErr := jsonl.Close(); verifyErr == nil || closeErr != nil {
			t.Fatalf("contrato no canónico aceptado: %v / %v", verifyErr, closeErr)
		}
	}
}
func TestNonReplacingGenerationAndExclusiveLockRejectConflicts(t *testing.T) {
	root := t.TempDir()
	opts := testOptions(t, root, modeMetadata)
	first, err := newPublicationForTest(opts)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := newPublicationForTest(opts); !errors.Is(err, errOutputLocked) {
		t.Fatalf("el segundo escritor no fue bloqueado: %v", err)
	}
	if err := first.abort(); err != nil {
		t.Fatal(err)
	}
	opts = testOptions(t, t.TempDir(), modeMetadata)
	mustWrite(t, opts.jsonlPath, "ya existe", 0o600)
	mustWrite(t, opts.manifestPath, "también existe", 0o600)
	if _, err := newPublicationForTest(opts); !errors.Is(err, errRecoveryConflict) {
		t.Fatalf("los finales sin diario no se preservaron como ambiguos: %v", err)
	}
}
func TestSharedOutputDirectoryIsRejectedBeforeCreatingStages(t *testing.T) {
	root := t.TempDir()
	opts := testOptions(t, root, modeMetadata)
	if err := os.Chmod(filepath.Dir(opts.jsonlPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := newPublicationForTest(opts); !errors.Is(err, errOutputDirectory) {
		t.Fatalf("directorio compartido aceptado: %v", err)
	}
	if matches, _ := filepath.Glob(filepath.Join(filepath.Dir(opts.jsonlPath), ".*.stage")); len(matches) != 0 {
		t.Fatalf("se crearon etapas antes de validar permisos: %v", matches)
	}
}
func TestReplacedStageIsRejectedBeforeCommit(t *testing.T) {
	root := t.TempDir()
	opts := testOptions(t, root, modeMetadata)
	publication, err := newPublicationForTest(opts)
	if err != nil {
		t.Fatal(err)
	}
	defer publication.abort()
	if err := unix.Unlinkat(publication.anchor.fd(), publication.jsonlStage, 0); err != nil {
		t.Fatal(err)
	}
	replacement, err := createAt(publication.anchor.fd(), publication.jsonlStage)
	if err != nil {
		t.Fatal(err)
	}
	if err := replacement.Close(); err != nil {
		t.Fatal(err)
	}
	if err := publication.verifyStages(); !errors.Is(err, errStageReplaced) {
		t.Fatalf("etapa sustituida aceptada: %v", err)
	}
}
