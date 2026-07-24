package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestCommandGenerationIsByteDeterministicAndSourceDigestBound(t *testing.T) {
	repositoryRoot, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	registry, err := os.ReadFile(filepath.Join(repositoryRoot, "internal/commands/registry.json"))
	if err != nil {
		t.Fatal(err)
	}
	generate := func() string {
		root := t.TempDir()
		if err := createGeneratedParents(root); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(root, "internal/commands")
		if err := os.WriteFile(filepath.Join(path, "registry.json"), registry, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := run(root, "internal/commands/registry.json", modeWrite); err != nil {
			t.Fatal(err)
		}
		return root
	}
	first, second := generate(), generate()
	for _, relative := range generatedTestPaths() {
		one, err := os.ReadFile(filepath.Join(first, relative))
		if err != nil {
			t.Fatal(err)
		}
		two, err := os.ReadFile(filepath.Join(second, relative))
		if err != nil {
			t.Fatal(err)
		}
		checked, err := os.ReadFile(filepath.Join(repositoryRoot, relative))
		if err != nil {
			t.Fatal(err)
		}
		if string(one) != string(two) || string(one) != string(checked) {
			t.Errorf("generated file drift: %s", relative)
		}
	}
}

func TestGeneratorRejectsContractMutationsAndTrailingJSON(t *testing.T) {
	repositoryRoot, _ := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	body, err := os.ReadFile(filepath.Join(repositoryRoot, "internal/commands/registry.json"))
	if err != nil {
		t.Fatal(err)
	}
	decode := func(t *testing.T) document {
		t.Helper()
		decoder := json.NewDecoder(bytes.NewReader(body))
		decoder.DisallowUnknownFields()
		var registry document
		if err := decoder.Decode(&registry); err != nil {
			t.Fatal(err)
		}
		return registry
	}
	tests := []struct {
		name   string
		mutate func(*document)
	}{
		{"duplicate_id", func(value *document) { value.Commands[1].ID = value.Commands[0].ID }},
		{"duplicate_handler", func(value *document) { value.Commands[1].Handler = value.Commands[0].Handler }},
		{"unknown_permission", func(value *document) { value.Commands[0].Permission = "unknown" }},
		{"permission_handler_mismatch", func(value *document) { value.Commands[0].Permission = "goals.get" }},
		{"open_schema", func(value *document) {
			value.Commands[0].InputSchema = json.RawMessage(`{"type":"object","properties":{}}`)
		}},
		{"missing_binding", func(value *document) { value.Commands[0].MCP.Tool = "" }},
		{"missing_annotations", func(value *document) {
			value.Commands[0].MCP.Annotations = nil
		}},
		{"missing_read_only_annotation", func(value *document) {
			value.Commands[0].MCP.Annotations.ReadOnly = nil
		}},
		{"missing_destructive_annotation", func(value *document) {
			value.Commands[0].MCP.Annotations.Destructive = nil
		}},
		{"missing_idempotent_annotation", func(value *document) {
			value.Commands[0].MCP.Annotations.Idempotent = nil
		}},
		{"missing_open_world_annotation", func(value *document) {
			value.Commands[0].MCP.Annotations.OpenWorld = nil
		}},
		{"query_not_read_only", func(value *document) {
			no := false
			value.Commands[2].MCP.Annotations.ReadOnly = &no
		}},
		{"query_destructive", func(value *document) {
			yes := true
			value.Commands[2].MCP.Annotations.Destructive = &yes
		}},
		{"non_idempotent", func(value *document) {
			no := false
			value.Commands[0].MCP.Annotations.Idempotent = &no
		}},
		{"open_world_true", func(value *document) {
			yes := true
			value.Commands[0].MCP.Annotations.OpenWorld = &yes
		}},
		{"wrong_description", func(value *document) { value.Commands[0].DescriptionKey = "command.other.description" }},
		{"wrong_cli", func(value *document) { value.Commands[0].CLI.Path = []string{"other"} }},
		{"wrong_errors", func(value *document) { value.Commands[0].ErrorCodes[0] = "other" }},
		{"wrong_replay", func(value *document) { value.Commands[0].ReplayMode = "read_reexecute" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			registry := decode(t)
			test.mutate(&registry)
			if err := validate(registry.Commands); err == nil {
				t.Fatal("mutation accepted")
			}
		})
	}
	root := t.TempDir()
	path := filepath.Join(root, "internal/commands")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "registry.json"), append(body, []byte("\n{}")...), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run(root, "internal/commands/registry.json", modeWrite); err == nil {
		t.Fatal("trailing JSON accepted")
	}
}

func TestCommandRegistryRejectsDuplicateIDsAndBindingsUnknownPermissionsHandlersAndCatalogKeys(t *testing.T) {
	repositoryRoot, _ := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	body, err := os.ReadFile(filepath.Join(repositoryRoot, "internal/commands/registry.json"))
	if err != nil {
		t.Fatal(err)
	}
	decode := func(t *testing.T) document {
		t.Helper()
		decoder := json.NewDecoder(bytes.NewReader(body))
		decoder.DisallowUnknownFields()
		var registry document
		if err := decoder.Decode(&registry); err != nil {
			t.Fatal(err)
		}
		return registry
	}
	tests := []struct {
		name   string
		mutate func(*document)
	}{
		{"duplicate_id", func(value *document) {
			value.Commands[1] = value.Commands[0]
		}},
		{"duplicate_http_binding", func(value *document) {
			value.Commands[1].HTTP = value.Commands[0].HTTP
		}},
		{"duplicate_mcp_binding", func(value *document) {
			value.Commands[1].MCP = value.Commands[0].MCP
		}},
		{"duplicate_cli_binding", func(value *document) {
			value.Commands[1].CLI.Path = append([]string(nil), value.Commands[0].CLI.Path...)
		}},
		{"unknown_permission", func(value *document) {
			value.Commands[0].Permission = "goals.unknown"
		}},
		{"unknown_handler", func(value *document) {
			value.Commands[0].Handler = "UnknownUseCase"
		}},
		{"wrong_catalog_key", func(value *document) {
			value.Commands[0].DescriptionKey = "command.unrelated.description"
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			registry := decode(t)
			test.mutate(&registry)
			if err := validate(registry.Commands); err == nil {
				t.Fatal("invalid registry mutation accepted")
			}
		})
	}
}

func TestCheckDetectsDriftWithoutMutation(t *testing.T) {
	root := t.TempDir()
	repositoryRoot, _ := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	registry, err := os.ReadFile(filepath.Join(repositoryRoot, "internal/commands/registry.json"))
	if err != nil {
		t.Fatal(err)
	}
	registryPath := filepath.Join(root, "internal/commands")
	if err := createGeneratedParents(root); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(registryPath, "registry.json"), registry, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run(root, "internal/commands/registry.json", modeWrite); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "internal/commands/definitions_generated.go")
	before := []byte("drift")
	if err := os.WriteFile(target, before, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run(root, "internal/commands/registry.json", modeCheck); err == nil {
		t.Fatal("check accepted drift")
	}
	after, _ := os.ReadFile(target)
	if string(after) != string(before) {
		t.Fatal("check mutated generated file")
	}
}

func TestWritePreparesEveryTemporaryBeforeRenamingAnyTarget(t *testing.T) {
	root := t.TempDir()
	paths := []string{"generated/a.go", "generated/b.go", "generated/c.go"}
	outputs := map[string][]byte{}
	for _, relative := range paths {
		outputs[relative] = []byte("new:" + relative)
		target := filepath.Join(root, relative)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, []byte("old:"+relative), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	before := snapshotGeneratedTree(t, root)
	prepared := 0
	err := writeGeneratedWith(root, paths, outputs, func(path string, body []byte) (string, error) {
		prepared++
		if prepared == len(paths) {
			return "", errors.New("injected late preparation failure")
		}
		return prepareGeneratedTemp(path, body)
	})
	if err == nil || prepared != len(paths) {
		t.Fatalf("err=%v prepared=%d", err, prepared)
	}
	after := snapshotGeneratedTree(t, root)
	if !bytes.Equal(before, after) {
		t.Fatalf("tree changed after preparation failure:\nbefore=%s\nafter=%s", before, after)
	}
}

func TestWritePreflightsEveryDestinationBeforePreparing(t *testing.T) {
	root := t.TempDir()
	paths := []string{"generated/a.go", "generated/b.go"}
	outputs := map[string][]byte{"generated/a.go": []byte("new a"), "generated/b.go": []byte("new b")}
	first := filepath.Join(root, paths[0])
	if err := os.MkdirAll(filepath.Dir(first), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(first, []byte("old a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, paths[1]), 0o755); err != nil {
		t.Fatal(err)
	}
	before := snapshotGeneratedTree(t, root)
	prepared := 0
	err := writeGeneratedWith(root, paths, outputs, func(string, []byte) (string, error) {
		prepared++
		return "", errors.New("must not prepare")
	})
	if err == nil || prepared != 0 {
		t.Fatalf("err=%v prepared=%d", err, prepared)
	}
	after := snapshotGeneratedTree(t, root)
	if !bytes.Equal(before, after) {
		t.Fatalf("tree changed during preflight:\nbefore=%s\nafter=%s", before, after)
	}
}

func TestWriteRejectsDuplicateAndSymlinkedPathsWithoutMutation(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "generated"), 0o755); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "generated", "escape")); err != nil {
		t.Fatal(err)
	}
	before := snapshotGeneratedTree(t, root)
	prepareCalls := 0
	prepare := func(string, []byte) (string, error) { prepareCalls++; return "", errors.New("unexpected") }
	if err := writeGeneratedWith(root, []string{"generated/a.go", "generated/a.go"}, map[string][]byte{"generated/a.go": []byte("a")}, prepare); err == nil {
		t.Fatal("duplicate path accepted")
	}
	if err := writeGeneratedWith(root, []string{"generated/escape/a.go"}, map[string][]byte{"generated/escape/a.go": []byte("a")}, prepare); err == nil {
		t.Fatal("symlinked parent accepted")
	}
	if prepareCalls != 0 || !bytes.Equal(before, snapshotGeneratedTree(t, root)) {
		t.Fatalf("parent preflight mutated tree or prepared files: calls=%d", prepareCalls)
	}
	if err := os.Symlink(filepath.Join(outside, "target.go"), filepath.Join(root, "generated", "target.go")); err != nil {
		t.Fatal(err)
	}
	withTargetSymlink := snapshotGeneratedTree(t, root)
	if err := writeGeneratedWith(root, []string{"generated/target.go"}, map[string][]byte{"generated/target.go": []byte("a")}, prepare); err == nil {
		t.Fatal("symlinked target accepted")
	}
	if prepareCalls != 0 || !bytes.Equal(withTargetSymlink, snapshotGeneratedTree(t, root)) {
		t.Fatalf("target preflight mutated tree or prepared files: calls=%d", prepareCalls)
	}
}

func createGeneratedParents(root string) error {
	for _, relative := range generatedTestPaths() {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, relative)), 0o755); err != nil {
			return err
		}
	}
	return nil
}

func generatedTestPaths() []string {
	return []string{"internal/commands/definitions_generated.go"}
}

func snapshotGeneratedTree(t *testing.T, root string) []byte {
	t.Helper()
	var snapshot bytes.Buffer
	err := filepath.WalkDir(root, func(path string, _ os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		fmt.Fprintf(&snapshot, "%s|%s|%04o|", relative, info.Mode().Type(), info.Mode().Perm())
		if info.Mode().IsRegular() {
			body, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			snapshot.Write(body)
		} else if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			snapshot.WriteString(target)
		}
		snapshot.WriteByte('\n')
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return snapshot.Bytes()
}
