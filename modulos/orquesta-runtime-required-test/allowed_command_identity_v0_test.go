package orquestaruntimerequiredtest

import (
	"bytes"
	"context"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestAllowedCommandIdentityRegistryV0RejectsSymlinkAndReplacement(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	link := filepath.Join(dir, "link")
	if err := os.WriteFile(target, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := newAllowedCommandIdentityRegistryV0(map[string]string{"runner": link}); err == nil {
		t.Fatal("symlink admitted")
	}
	registry, err := newAllowedCommandIdentityRegistryV0(map[string]string{"runner": target})
	if err != nil {
		t.Fatal(err)
	}
	defer registry.Close()
	if err := os.WriteFile(target, []byte("#!/bin/sh\nexit 1\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.resolve("runner"); err == nil || !strings.Contains(err.Error(), "identity_changed") {
		t.Fatalf("mutation err=%v", err)
	}
}

func TestAllowedCommandIdentityRegistryV0RejectsRenameReplacement(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "runner")
	replacement := filepath.Join(dir, "replacement")
	for _, item := range []string{path, replacement} {
		if err := os.WriteFile(item, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	registry, err := newAllowedCommandIdentityRegistryV0(map[string]string{"runner": path})
	if err != nil {
		t.Fatal(err)
	}
	defer registry.Close()
	if err := os.Rename(replacement, path); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.resolve("runner"); err == nil || !strings.Contains(err.Error(), "identity_changed") {
		t.Fatalf("rename replacement err=%v", err)
	}
}

func TestAllowedCommandIdentityRegistryV0RejectsNormalizedDuplicateAlias(t *testing.T) {
	path, err := filepath.EvalSymlinks(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	if _, err := newAllowedCommandIdentityRegistryV0(map[string]string{
		"runner":   path,
		" runner ": path,
	}); err == nil || !strings.Contains(err.Error(), "alias_duplicate") {
		t.Fatalf("duplicate alias err=%v", err)
	}
}

func TestAllowedCommandIdentityRegistryV0CloseReleasesMasterDescriptor(t *testing.T) {
	path, err := filepath.EvalSymlinks(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	registry, err := newAllowedCommandIdentityRegistryV0(map[string]string{"runner": path})
	if err != nil {
		t.Fatal(err)
	}
	masterFD := int(registry.commands["runner"].file.Fd())
	sourceFD := int(registry.commands["runner"].source.Fd())
	if err := registry.Close(); err != nil {
		t.Fatal(err)
	}
	var stat syscall.Stat_t
	if err := syscall.Fstat(masterFD, &stat); err == nil {
		t.Fatalf("master fd %d remains open after Close", masterFD)
	}
	if err := syscall.Fstat(sourceFD, &stat); err == nil {
		t.Fatalf("source fd %d remains open after Close", sourceFD)
	}
}

func TestAllowedCommandIdentityRegistryV0ResolvedSnapshotIsSealedAgainstSameInodeMutation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "runner")
	original := []byte("#!/bin/sh\nprintf 'snapshot-original-v0\\n'\n")
	mutated := []byte("#!/bin/sh\nprintf 'source-mutated-v0\\n'\n")
	if err := os.WriteFile(path, original, 0o700); err != nil {
		t.Fatal(err)
	}
	registry, err := newAllowedCommandIdentityRegistryV0(map[string]string{"runner": path})
	if err != nil {
		t.Fatal(err)
	}
	defer registry.Close()
	resolved, err := registry.resolve("runner")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, mutated, 0o700); err != nil {
		_ = resolved.executionFile.Close()
		t.Fatal(err)
	}
	seals, err := allowedCommandSealsV0(int(resolved.executionFile.Fd()))
	if err != nil || seals&allowedCommandRequiredSealsV0 != allowedCommandRequiredSealsV0 {
		_ = resolved.executionFile.Close()
		t.Fatalf("snapshot seals=%#x err=%v", seals, err)
	}
	if _, err := syscall.Pwrite(int(resolved.executionFile.Fd()), []byte("x"), 0); err != syscall.EPERM {
		_ = resolved.executionFile.Close()
		t.Fatalf("sealed snapshot accepted write: %v", err)
	}
	got := make([]byte, len(original))
	n, err := syscall.Pread(int(resolved.executionFile.Fd()), got, 0)
	if err != nil || n != len(original) || !bytes.Equal(got, original) {
		_ = resolved.executionFile.Close()
		t.Fatalf("snapshot bytes n=%d err=%v got=%q", n, err, got[:n])
	}
	output, err := runLocalCommandV0(context.Background(), LocalCommandExecutorV0{
		ProjectWorkDir: dir,
		OutputDir:      t.TempDir(),
		MaxOutputBytes: 4096,
	}, resolved, nil)
	if err != nil || output.String() != "snapshot-original-v0\n" {
		t.Fatalf("snapshot execution output=%q err=%v", output.String(), err)
	}
	if _, err := registry.resolve("runner"); err == nil || !strings.Contains(err.Error(), "identity_changed") {
		t.Fatalf("mutated source revalidation err=%v", err)
	}
}

func TestAllowedCommandIdentityRegistryV0DuplicateHasCloexecAtRuntime(t *testing.T) {
	path, err := filepath.EvalSymlinks(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	registry, err := newAllowedCommandIdentityRegistryV0(map[string]string{"test-bin": path})
	if err != nil {
		t.Fatal(err)
	}
	defer registry.Close()
	resolved, err := registry.resolve("test-bin")
	if err != nil {
		t.Fatal(err)
	}
	defer resolved.executionFile.Close()
	flags, _, errno := syscall.Syscall(syscall.SYS_FCNTL, resolved.executionFile.Fd(), uintptr(syscall.F_GETFD), 0)
	if errno != 0 || flags&syscall.FD_CLOEXEC == 0 {
		t.Fatalf("F_GETFD flags=%#x errno=%v", flags, errno)
	}
}

func TestMemfdCreateSyscallNumberV0LinuxArchitectureContract(t *testing.T) {
	want := map[string]uintptr{
		"amd64":   319,
		"386":     356,
		"arm64":   279,
		"arm":     385,
		"riscv64": 279,
		"loong64": 279,
		"ppc64":   360,
		"ppc64le": 360,
		"s390x":   350,
	}
	for goarch, syscallNumber := range want {
		if got := memfdCreateSyscallNumberForGOARCHV0(goarch); got != syscallNumber {
			t.Fatalf("memfd_create %s=%d want=%d", goarch, got, syscallNumber)
		}
	}
	if got := memfdCreateSyscallNumberForGOARCHV0("unsupported-test-arch"); got != 0 {
		t.Fatalf("unsupported memfd_create syscall=%d", got)
	}
	if wantRuntime, ok := want[runtime.GOARCH]; ok && memfdCreateSyscallNumberV0() != wantRuntime {
		t.Fatalf("runtime %s memfd_create=%d want=%d", runtime.GOARCH, memfdCreateSyscallNumberV0(), wantRuntime)
	}
}

func TestAllowedCommandIdentityRegistryV0CloexecGuardUsesTypedThirdArgument(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "allowed_command_identity_v0.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Uses: map[*ast.Ident]types.Object{}}
	config := types.Config{Importer: importer.Default(), Error: func(error) {}}
	_, _ = config.Check("orquestaruntimerequiredtest", fset, []*ast.File{file}, info)
	found := false
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || len(call.Args) < 4 {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "Syscall" {
			return true
		}
		id, ok := call.Args[2].(*ast.CallExpr)
		if !ok || len(id.Args) != 1 {
			return true
		}
		flag, ok := id.Args[0].(*ast.SelectorExpr)
		if !ok || flag.Sel.Name != "F_DUPFD_CLOEXEC" {
			return true
		}
		constant, typed := info.Uses[flag.Sel].(*types.Const)
		if typed && constant.Pkg() != nil && constant.Pkg().Path() == "syscall" && formatNodeV0(fset, call.Args[3]) == "3" {
			found = true
		}
		return true
	})
	if !found {
		t.Fatal("F_DUPFD_CLOEXEC must be the fcntl flags argument Args[2]")
	}
}

func formatNodeV0(fset *token.FileSet, node ast.Node) string {
	start, end := fset.Position(node.Pos()).Offset, fset.Position(node.End()).Offset
	data, _ := os.ReadFile("allowed_command_identity_v0.go")
	if start < 0 || end < start || end > len(data) {
		return ""
	}
	return string(data[start:end])
}

func TestLocalCommandExecutorV0RunCloseRaceFailsClosed(t *testing.T) {
	executor := localCommandExecutorForTestV0(t, t.TempDir(), "pass")
	type outcomeV0 struct {
		result orquestacionnucleoapp.RequiredTestCommandExecutionResultV0
		err    error
	}
	done := make(chan outcomeV0, 1)
	go func() {
		result, err := executor.RunRequiredTestCommandV0(context.Background(), commandRequestForTestV0("orquesta-test-bin"))
		done <- outcomeV0{result: result, err: err}
	}()
	if err := executor.Close(); err != nil {
		t.Fatal(err)
	}
	first := <-done
	if first.err != nil || (first.result.Status != orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0 &&
		first.result.Status != orquestacionnucleoapp.RequiredTestEvidenceStatusFailedV0) {
		t.Fatalf("concurrent Run outcome=%+v err=%v", first.result, first.err)
	}
	result, err := executor.RunRequiredTestCommandV0(context.Background(), commandRequestForTestV0("orquesta-test-bin"))
	if err != nil || result.Status != orquestacionnucleoapp.RequiredTestEvidenceStatusFailedV0 {
		t.Fatalf("after Close result=%+v err=%v", result, err)
	}
}
