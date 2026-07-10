package scripts_test

import (
	"os/exec"
	"path/filepath"
	"runtime"
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
	paths := []string{
		filepath.Join(root, "orquesta_server_drain.sh"),
		filepath.Join(root, "test_orquesta_server_drain.sh"),
		filepath.Join(root, "orquesta_test_batches.sh"),
		filepath.Join(root, "test_orquesta_test_batches.sh"),
		filepath.Join(root, "lib/isolated_test_env.sh"),
	}
	args := append([]string{"-n"}, paths...)
	if output, err := exec.Command("bash", args...).CombinedOutput(); err != nil {
		t.Fatalf("bash -n F3-R2: %v\n%s", err, output)
	}
}
