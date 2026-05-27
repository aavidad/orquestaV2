package orquestaruntimecodex

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCodexControlFileWriteV0CreaReemplazaEIdempotenteConReceipt(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, CodexAgentPromptFileNameV0)

	created, err := WriteCodexControlFileBytesV0(CodexControlFileWriteRequestV0{
		RootDir:     root,
		Path:        path,
		FileName:    CodexAgentPromptFileNameV0,
		ControlKind: "agent_prompt",
		Data:        []byte("prompt v1"),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.Operation != "created" || created.SizeBytes != int64(len("prompt v1")) ||
		created.SHA256 == "" || strings.Contains(created.ControlFileRef, root) {
		t.Fatalf("receipt create=%+v", created)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("perm=%o", info.Mode().Perm())
	}

	idempotent, err := WriteCodexControlFileBytesV0(CodexControlFileWriteRequestV0{
		RootDir:     root,
		Path:        path,
		FileName:    CodexAgentPromptFileNameV0,
		ControlKind: "agent_prompt",
		Data:        []byte("prompt v1"),
	})
	if err != nil {
		t.Fatalf("idempotent: %v", err)
	}
	if idempotent.Operation != "idempotent" || idempotent.SHA256 != created.SHA256 {
		t.Fatalf("receipt idempotent=%+v create=%+v", idempotent, created)
	}

	replaced, err := WriteCodexControlFileBytesV0(CodexControlFileWriteRequestV0{
		RootDir:     root,
		Path:        path,
		FileName:    CodexAgentPromptFileNameV0,
		ControlKind: "agent_prompt",
		Data:        []byte("prompt v2"),
	})
	if err != nil {
		t.Fatalf("replace: %v", err)
	}
	if replaced.Operation != "replaced" || replaced.SHA256 == created.SHA256 {
		t.Fatalf("receipt replace=%+v create=%+v", replaced, created)
	}
}

func TestCodexControlFileWriteV0BloqueaSymlinkYConflicto(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target.txt")
	if err := os.WriteFile(target, []byte("target"), 0o600); err != nil {
		t.Fatalf("target: %v", err)
	}
	link := filepath.Join(root, CodexAgentPromptFileNameV0)
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	_, err := WriteCodexControlFileBytesV0(CodexControlFileWriteRequestV0{
		RootDir:  root,
		Path:     link,
		FileName: CodexAgentPromptFileNameV0,
		Data:     []byte("prompt"),
	})
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("err=%v want symlink", err)
	}

	conflictPath := filepath.Join(root, CodexShutdownRequestFileNameV0)
	if err := os.WriteFile(conflictPath, []byte("old"), 0o600); err != nil {
		t.Fatalf("conflict seed: %v", err)
	}
	_, err = WriteCodexControlFileBytesV0(CodexControlFileWriteRequestV0{
		RootDir:  root,
		Path:     conflictPath,
		FileName: CodexShutdownRequestFileNameV0,
		Data:     []byte("new"),
		Mode:     CodexControlFileWriteCreateIfAbsentV0,
	})
	if err == nil || !strings.Contains(err.Error(), "payload_conflict") {
		t.Fatalf("err=%v want payload_conflict", err)
	}
}
