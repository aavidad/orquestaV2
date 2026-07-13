package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServerProjectConfigLoaderV0BoundsBeforeDecodeAndPublishesOnlyValidCandidate(t *testing.T) {
	path := filepath.Join(t.TempDir(), serverProjectConfigFileNameV0)
	if err := os.WriteFile(path, []byte(`{"schema_version":"orquesta_config.v0"} {}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := loadServerProjectConfigPathV0(path); err == nil || ok {
		t.Fatalf("trailing candidate published ok=%v err=%v", ok, err)
	}
	if err := os.WriteFile(path, []byte(`{"schema_version":"wrong"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := loadServerProjectConfigPathV0(path); err == nil || ok || !strings.Contains(err.Error(), configFileUnsupportedSchemaCodeV0) {
		t.Fatalf("semantic candidate published ok=%v err=%v", ok, err)
	}
	if err := os.WriteFile(path, make([]byte, serverProjectConfigMaxBytesV0+1), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := loadServerProjectConfigPathV0(path); err == nil || ok {
		t.Fatalf("oversize candidate published ok=%v err=%v", ok, err)
	}
}

func TestServerProjectConfigProductLoaderV0UsesIsolatedFixturesUnderConcurrency(t *testing.T) {
	const fixtures = 16
	errs := make(chan error, fixtures)
	for i := 0; i < fixtures; i++ {
		root := t.TempDir()
		path := filepath.Join(root, serverProjectConfigFileNameV0)
		if err := os.WriteFile(path, []byte(`{"schema_version":"orquesta_config.v0"}`), 0o600); err != nil {
			t.Fatal(err)
		}
		go func(path string) {
			loaded, ok, err := loadServerProjectConfigProductPathV0(path)
			if err != nil || !ok || loaded.Revision == "" || len(loaded.Catalog) == 0 {
				errs <- fmt.Errorf("load err=%v ok=%v revision=%q catalog=%d", err, ok, loaded.Revision, len(loaded.Catalog))
				return
			}
			errs <- nil
		}(path)
	}
	for i := 0; i < fixtures; i++ {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}
}

func TestServerProjectConfigCanonicalRevisionChangesOnlyAfterValidMutationV0(t *testing.T) {
	config := serverProjectConfigFileV0{SchemaVersion: serverProjectConfigSchemaVersionV0}
	before, err := canonicalServerProjectConfigV0(config)
	if err != nil {
		t.Fatal(err)
	}
	concurrency := 2
	config.CodexRuntime.MaxConcurrency = &concurrency
	after, err := canonicalServerProjectConfigV0(config)
	if err != nil {
		t.Fatal(err)
	}
	if before.Revision == after.Revision || string(before.Bytes) == string(after.Bytes) {
		t.Fatalf("mutation did not produce a new canonical revision: before=%s after=%s", before.Revision, after.Revision)
	}
	config.SchemaVersion = "wrong"
	if _, err := canonicalServerProjectConfigV0(config); err == nil {
		t.Fatal("invalid mutation produced a canonical document")
	}
}

func TestServerProjectConfigProductLoaderReturnsOneValidatedSnapshotV0(t *testing.T) {
	path := filepath.Join(t.TempDir(), serverProjectConfigFileNameV0)
	if err := os.WriteFile(path, []byte(`{"schema_version":"orquesta_config.v0"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, ok, err := loadServerProjectConfigProductPathV0(path)
	if err != nil || !ok {
		t.Fatalf("load err=%v ok=%v", err, ok)
	}
	if loaded.Config.SchemaVersion != serverProjectConfigSchemaVersionV0 || len(loaded.CanonicalBytes) == 0 || loaded.Revision == "" || len(loaded.Catalog) == 0 {
		t.Fatalf("incomplete product load: %#v", loaded)
	}
}
