// Estas pruebas reproducen HEAD no nacido/simbólico y referencias con nombres
// no UTF-8 para fijar su representación V4 y su digest byte a byte.
package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHeadPreservesUnbornSymbolicTarget(t *testing.T) {
	repository := t.TempDir()
	gitTest(t, repository, "init", "-q")
	target := []byte(strings.TrimSpace(gitTest(t, repository, "symbolic-ref", "HEAD")))
	refs, err := readRefs(repository, nil)
	if err != nil {
		t.Fatal(err)
	}
	head := findRawRef(t, refs, []byte("HEAD"))
	if head.Mode != "symbolic" || !bytes.Equal(head.Target, target) ||
		head.Object != "" || head.Commit != "" {
		t.Fatalf("HEAD no nacido perdió su estado: %#v", head)
	}
	emitted := referenceRecord(head)
	if emitted.RefMode != "symbolic" ||
		emitted.RefTargetEncoding != "utf8" ||
		emitted.RefTarget != string(target) {
		t.Fatalf("registro HEAD simbólico incompleto: %#v", emitted)
	}
}

func TestHeadSymbolicAndDetachedDifferWithSameCommit(t *testing.T) {
	repository := newCommittedRepository(t)
	symbolicRefs, err := readRefs(repository, nil)
	if err != nil {
		t.Fatal(err)
	}
	symbolic := findRawRef(t, symbolicRefs, []byte("HEAD"))
	gitTest(t, repository, "checkout", "--detach", "-q", "HEAD")
	detachedRefs, err := readRefs(repository, nil)
	if err != nil {
		t.Fatal(err)
	}
	detached := findRawRef(t, detachedRefs, []byte("HEAD"))
	if symbolic.Object == "" || symbolic.Object != detached.Object ||
		symbolic.Commit != detached.Commit {
		t.Fatalf("HEAD no resolvió el mismo OID: simbólico=%#v separado=%#v", symbolic, detached)
	}
	if symbolic.Mode != "symbolic" || len(symbolic.Target) == 0 ||
		detached.Mode != "direct" || len(detached.Target) != 0 {
		t.Fatalf("los estados de HEAD no se distinguen: simbólico=%#v separado=%#v", symbolic, detached)
	}
	if digestJSON(referenceHashDomain, symbolicRefs) ==
		digestJSON(referenceHashDomain, detachedRefs) {
		t.Fatal("el digest confunde HEAD simbólico y separado con el mismo OID")
	}
}

func TestNonUTF8ReferenceNameIsExactDeterministicAndAffectsDigest(t *testing.T) {
	repository := newCommittedRepository(t)
	commit := strings.TrimSpace(gitTest(t, repository, "rev-parse", "HEAD"))
	firstName := []byte("refs/heads/bin-\xff")
	firstPath := looseReferencePath(repository, firstName)
	if err := os.WriteFile(firstPath, []byte(commit+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	firstJSONL := filepath.Join(t.TempDir(), "inventory.jsonl")
	firstManifest := filepath.Join(t.TempDir(), "manifest.json")
	secondJSONL := filepath.Join(t.TempDir(), "inventory.jsonl")
	secondManifest := filepath.Join(t.TempDir(), "manifest.json")
	for _, output := range [][2]string{
		{firstJSONL, firstManifest},
		{secondJSONL, secondManifest},
	} {
		if err := run(options{
			repository: repository, jsonl: output[0], manifest: output[1],
		}); err != nil {
			t.Fatal(err)
		}
	}
	assertSameFile(t, firstJSONL, secondJSONL)
	assertSameFile(t, firstManifest, secondManifest)
	assertEncodedReference(t, firstJSONL, firstName)
	firstDigest := readManifestForReferenceTest(t, firstManifest).RefSnapshotSHA256

	secondName := []byte("refs/heads/bin-\xfe")
	secondPath := looseReferencePath(repository, secondName)
	if err := os.Rename(firstPath, secondPath); err != nil {
		t.Fatal(err)
	}
	thirdManifest := filepath.Join(t.TempDir(), "manifest.json")
	thirdJSONL := filepath.Join(t.TempDir(), "inventory.jsonl")
	if err := run(options{
		repository: repository, jsonl: thirdJSONL, manifest: thirdManifest,
	}); err != nil {
		t.Fatal(err)
	}
	assertEncodedReference(t, thirdJSONL, secondName)
	secondDigest := readManifestForReferenceTest(t, thirdManifest).RefSnapshotSHA256
	if firstDigest == secondDigest {
		t.Fatal("dos nombres binarios distintos produjeron el mismo digest de referencias")
	}
}

func findRawRef(t *testing.T, refs []refInfo, name []byte) refInfo {
	t.Helper()
	for _, ref := range refs {
		if bytes.Equal(ref.Name, name) {
			return ref
		}
	}
	t.Fatalf("no se encontró la referencia %q", name)
	return refInfo{}
}

func looseReferencePath(repository string, name []byte) string {
	relative := bytes.TrimPrefix(name, []byte("refs/"))
	return filepath.Join(repository, ".git", "refs", string(relative))
}

func assertEncodedReference(t *testing.T, jsonl string, expected []byte) {
	t.Helper()
	for _, item := range readTestRecords(t, jsonl) {
		if item.RecordKind != "reference" || item.RefNameEncoding != "base64" {
			continue
		}
		decoded, err := base64.StdEncoding.DecodeString(item.RefNameBase64)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Equal(decoded, expected) && item.RefName == "" {
			return
		}
	}
	t.Fatalf("el inventario no conservó la referencia binaria %x", expected)
}

func readManifestForReferenceTest(t *testing.T, path string) manifest {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var result manifest
	if err := json.Unmarshal(content, &result); err != nil {
		t.Fatal(err)
	}
	return result
}
