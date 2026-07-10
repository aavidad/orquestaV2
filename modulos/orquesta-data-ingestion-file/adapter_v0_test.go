package orquestadataingestionfile

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ingestion "orquesta/modulos/orquesta-data-ingestion"
)

func TestAdapterV0ProfilesCSVWithHashAndProvenance(t *testing.T) {
	root := t.TempDir()
	content := "name,age,active\nAda,37,true\nBob,,false\n"
	writeFileV0(t, filepath.Join(root, "people.csv"), content)
	adapter := newTestAdapterV0(t, root, map[string]SourceRegistrationV0{"source:people": {DatasetRef: "dataset:people", RelativePath: "people.csv", SourceKind: ingestion.DataSourceKindCSVV0}})
	source, err := adapter.ResolveDataSourceV0(context.Background(), "source:people")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	sum := sha256.Sum256([]byte(content))
	wantHash := fmt.Sprintf("sha256:%x", sum)
	if source.ContentHash != wantHash || source.SnapshotRef != "snapshot:file:"+wantHash || source.ProvenanceRef != "provenance:file:"+wantHash {
		t.Fatalf("source=%+v", source)
	}
	profile, err := adapter.ProfileDataSetV0(context.Background(), source)
	if err != nil {
		t.Fatalf("profile: %v", err)
	}
	if profile.RowCount != 2 || len(profile.Columns) != 3 || profile.Columns[1].DataType != "integer" || !profile.Columns[1].Nullable || profile.Columns[2].DataType != "boolean" {
		t.Fatalf("profile=%+v", profile)
	}
}

func TestAdapterV0ProfilesJSONArrayObjects(t *testing.T) {
	root := t.TempDir()
	writeFileV0(t, filepath.Join(root, "people.json"), `[{"name":"Ada","score":1},{"name":"Bob","score":1.5,"active":true}]`)
	adapter := newTestAdapterV0(t, root, map[string]SourceRegistrationV0{"source:people": {DatasetRef: "dataset:people", RelativePath: "people.json", SourceKind: ingestion.DataSourceKindJSONV0}})
	source, err := adapter.ResolveDataSourceV0(context.Background(), "source:people")
	if err != nil {
		t.Fatal(err)
	}
	profile, err := adapter.ProfileDataSetV0(context.Background(), source)
	if err != nil {
		t.Fatal(err)
	}
	if profile.RowCount != 2 || len(profile.Columns) != 3 || profile.Columns[1].DataType != "number" || !profile.Columns[2].Nullable || profile.Columns[2].DataType != "boolean" {
		t.Fatalf("profile=%+v", profile)
	}
}

func TestAdapterV0RejectsJSONTrailingData(t *testing.T) {
	root := t.TempDir()
	writeFileV0(t, filepath.Join(root, "people.json"), `[{"name":"Ada"}] {}`)
	adapter := newTestAdapterV0(t, root, map[string]SourceRegistrationV0{"source:people": {DatasetRef: "dataset:people", RelativePath: "people.json", SourceKind: ingestion.DataSourceKindJSONV0}})
	source, err := adapter.ResolveDataSourceV0(context.Background(), "source:people")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.ProfileDataSetV0(context.Background(), source); err == nil || !strings.Contains(err.Error(), "data_file_json_trailing_data") {
		t.Fatalf("trailing err=%v", err)
	}
}

func TestAdapterV0RejectsTraversalAndSymlinks(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.csv")
	writeFileV0(t, outside, "name\nAda\n")
	if _, err := NewAdapterV0(ConfigurationV0{AllowedRoot: root, Sources: map[string]SourceRegistrationV0{"source:bad": {DatasetRef: "dataset:bad", RelativePath: "../outside.csv", SourceKind: ingestion.DataSourceKindCSVV0}}}); err == nil || !strings.Contains(err.Error(), "data_file_path_outside_allowed_root") {
		t.Fatalf("traversal err=%v", err)
	}
	if _, err := NewAdapterV0(ConfigurationV0{AllowedRoot: root, Sources: map[string]SourceRegistrationV0{"source:also-bad": {DatasetRef: "dataset:bad", RelativePath: "nested/../outside.csv", SourceKind: ingestion.DataSourceKindCSVV0}}}); err == nil || !strings.Contains(err.Error(), "data_file_path_outside_allowed_root") {
		t.Fatalf("normalized traversal err=%v", err)
	}
	link := filepath.Join(root, "linked.csv")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	if _, err := NewAdapterV0(ConfigurationV0{AllowedRoot: root, Sources: map[string]SourceRegistrationV0{"source:link": {DatasetRef: "dataset:link", RelativePath: "linked.csv", SourceKind: ingestion.DataSourceKindCSVV0}}}); err == nil || !strings.Contains(err.Error(), "data_file_symlink_not_allowed") {
		t.Fatalf("symlink err=%v", err)
	}
}

func TestAdapterV0RejectsSymlinkCreatedAfterConstruction(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "people.csv")
	writeFileV0(t, path, "name\nAda\n")
	adapter := newTestAdapterV0(t, root, map[string]SourceRegistrationV0{"source:people": {DatasetRef: "dataset:people", RelativePath: "people.csv", SourceKind: ingestion.DataSourceKindCSVV0}})
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside.csv")
	writeFileV0(t, outside, "name\nBob\n")
	if err := os.Symlink(outside, path); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	if _, err := adapter.ResolveDataSourceV0(context.Background(), "source:people"); err == nil || !strings.Contains(err.Error(), "data_file_symlink_not_allowed") {
		t.Fatalf("symlink err=%v", err)
	}
}

func TestAdapterV0EnforcesSizeAndRowLimits(t *testing.T) {
	root := t.TempDir()
	writeFileV0(t, filepath.Join(root, "people.csv"), "name\nAda\nBob\n")
	config := ConfigurationV0{AllowedRoot: root, MaxBytes: 8, MaxRows: 1, Sources: map[string]SourceRegistrationV0{"source:people": {DatasetRef: "dataset:people", RelativePath: "people.csv", SourceKind: ingestion.DataSourceKindCSVV0}}}
	adapter, err := NewAdapterV0(config)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.ResolveDataSourceV0(context.Background(), "source:people"); err == nil || !strings.Contains(err.Error(), "data_file_size_limit_exceeded") {
		t.Fatalf("size err=%v", err)
	}
	config.MaxBytes = 1024
	adapter, err = NewAdapterV0(config)
	if err != nil {
		t.Fatal(err)
	}
	source, err := adapter.ResolveDataSourceV0(context.Background(), "source:people")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.ProfileDataSetV0(context.Background(), source); err == nil || !strings.Contains(err.Error(), "data_file_row_limit_exceeded") {
		t.Fatalf("row err=%v", err)
	}
}

func TestAdapterV0RejectsChangedSnapshot(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "people.csv")
	writeFileV0(t, path, "name\nAda\n")
	adapter := newTestAdapterV0(t, root, map[string]SourceRegistrationV0{"source:people": {DatasetRef: "dataset:people", RelativePath: "people.csv", SourceKind: ingestion.DataSourceKindCSVV0}})
	source, err := adapter.ResolveDataSourceV0(context.Background(), "source:people")
	if err != nil {
		t.Fatal(err)
	}
	writeFileV0(t, path, "name\nBob\n")
	if _, err := adapter.ProfileDataSetV0(context.Background(), source); err == nil || !strings.Contains(err.Error(), "data_file_source_snapshot_mismatch") {
		t.Fatalf("snapshot err=%v", err)
	}
}

func newTestAdapterV0(t *testing.T, root string, sources map[string]SourceRegistrationV0) *AdapterV0 {
	t.Helper()
	adapter, err := NewAdapterV0(ConfigurationV0{AllowedRoot: root, Sources: sources})
	if err != nil {
		t.Fatal(err)
	}
	return adapter
}

func writeFileV0(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
