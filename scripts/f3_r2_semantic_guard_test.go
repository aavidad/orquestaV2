package scripts_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func scriptsRootF3R2(t *testing.T) string {
	t.Helper()
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller no disponible")
	}
	return filepath.Dir(current)
}

func TestF3R2PidfdHelperAST(t *testing.T) {
	root := scriptsRootF3R2(t)
	program := `
import ast, pathlib, sys
tree=ast.parse(pathlib.Path(sys.argv[1]).read_text(encoding="utf-8"))
calls=[]
for node in ast.walk(tree):
    if not isinstance(node, ast.Call): continue
    if isinstance(node.func, ast.Attribute): name=node.func.attr
    elif isinstance(node.func, ast.Name): name=node.func.id
    else: name=""
    calls.append((name,node.lineno))
names=[name for name,_ in calls]
assert "pidfd_open" in names and "pidfd_send_signal" in names
assert min(line for name,line in calls if name=="pidfd_open") < min(line for name,line in calls if name=="pidfd_send_signal")
assert not ({"kill","system","popen"} & set(names))
`
	command := exec.Command("python3", "-c", program, filepath.Join(root, "lib/pidfd_signal.py"))
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("AST pidfd inválido: %v\n%s", err, output)
	}
}

func TestF3R2ShellSyntax(t *testing.T) {
	root := scriptsRootF3R2(t)
	relatives := []string{
		"orquesta_server_drain.sh",
		"test_orquesta_server_drain.sh",
		"orquesta_test_batches.sh",
		"test_orquesta_test_batches.sh",
		"lib/isolated_test_env.sh",
	}
	paths := make([]string, 0, len(relatives))
	for _, relative := range relatives {
		paths = append(paths, f3R2ShellSyntaxPath(t, root, relative))
	}
	args := append([]string{"-n"}, paths...)
	if output, err := exec.Command("bash", args...).CombinedOutput(); err != nil {
		t.Fatalf("bash -n F3-R2: %v\n%s", err, output)
	}
}

func f3R2ShellSyntaxPath(t *testing.T, scriptsRoot, relative string) string {
	t.Helper()
	path := filepath.Join(scriptsRoot, filepath.FromSlash(relative))
	if _, err := os.Stat(path); err == nil {
		return path
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat F3-R2 shell %s: %v", relative, err)
	}

	repositoryRoot := filepath.Dir(scriptsRoot)
	repositoryPath := filepath.ToSlash(filepath.Join("scripts", relative))
	index, err := exec.Command("git", "-C", repositoryRoot, "ls-files", "-v", "--", repositoryPath).CombinedOutput()
	if err != nil || !strings.HasPrefix(string(index), "S ") {
		t.Fatalf("missing non-sparse F3-R2 shell %s: index=%q err=%v", relative, index, err)
	}
	payload, err := exec.Command("git", "-C", repositoryRoot, "show", "HEAD:"+repositoryPath).CombinedOutput()
	if err != nil {
		t.Fatalf("read sparse F3-R2 shell %s: %v\n%s", relative, err, payload)
	}
	materialized := filepath.Join(t.TempDir(), filepath.Base(relative))
	if err := os.WriteFile(materialized, payload, 0o600); err != nil {
		t.Fatalf("materialize sparse F3-R2 shell %s: %v", relative, err)
	}
	return materialized
}
