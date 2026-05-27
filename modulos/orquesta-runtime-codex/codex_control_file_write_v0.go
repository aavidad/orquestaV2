package orquestaruntimecodex

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const CodexControlFileWriteReceiptSchemaVersionV0 = "codex_control_file_write_receipt.v0"

type CodexControlFileWriteModeV0 string

const (
	CodexControlFileWriteCreateOrReplaceV0 CodexControlFileWriteModeV0 = "create_or_replace"
	CodexControlFileWriteCreateIfAbsentV0  CodexControlFileWriteModeV0 = "create_if_absent"
)

type CodexControlFileWriteRequestV0 struct {
	RootDir     string
	Path        string
	FileName    string
	ControlKind string
	Data        []byte
	Mode        CodexControlFileWriteModeV0
	Perm        os.FileMode
}

type CodexControlFileWriteReceiptV0 struct {
	SchemaVersion  string   `json:"schema_version"`
	ControlFileRef string   `json:"control_file_ref"`
	ControlKind    string   `json:"control_kind"`
	FileName       string   `json:"file_name"`
	Operation      string   `json:"operation"`
	Mode           string   `json:"mode"`
	SizeBytes      int64    `json:"size_bytes"`
	SHA256         string   `json:"sha256"`
	EvidenceRefs   []string `json:"evidence_refs,omitempty"`
}

func WriteCodexControlFileBytesV0(
	request CodexControlFileWriteRequestV0,
) (CodexControlFileWriteReceiptV0, error) {
	request = normalizeCodexControlFileWriteRequestV0(request)
	if err := validateCodexControlFileWriteRequestV0(request); err != nil {
		return CodexControlFileWriteReceiptV0{}, err
	}
	if err := os.MkdirAll(request.RootDir, 0o700); err != nil {
		return CodexControlFileWriteReceiptV0{}, fmt.Errorf("codex_control_file_write: root_unavailable")
	}
	if err := validateCodexControlFileWriteDirV0(request.RootDir); err != nil {
		return CodexControlFileWriteReceiptV0{}, err
	}
	if err := os.MkdirAll(filepath.Dir(request.Path), 0o700); err != nil {
		return CodexControlFileWriteReceiptV0{}, fmt.Errorf("codex_control_file_write: parent_unavailable")
	}
	if err := validateCodexControlFileWriteDirV0(filepath.Dir(request.Path)); err != nil {
		return CodexControlFileWriteReceiptV0{}, err
	}
	operation, err := codexControlFileWriteOperationV0(request)
	if err != nil {
		return CodexControlFileWriteReceiptV0{}, err
	}
	receipt := codexControlFileWriteReceiptV0(request, operation)
	if operation == "idempotent" {
		_ = os.Chmod(request.Path, request.Perm)
		return receipt, nil
	}
	if err := writeCodexControlFileDurableV0(request); err != nil {
		return CodexControlFileWriteReceiptV0{}, err
	}
	return receipt, nil
}

func normalizeCodexControlFileWriteRequestV0(
	request CodexControlFileWriteRequestV0,
) CodexControlFileWriteRequestV0 {
	request.RootDir = filepath.Clean(strings.TrimSpace(request.RootDir))
	request.Path = filepath.Clean(strings.TrimSpace(request.Path))
	request.FileName = codexControlFileNameV0(request.FileName)
	request.ControlKind = strings.TrimSpace(request.ControlKind)
	if request.ControlKind == "" {
		request.ControlKind = request.FileName
	}
	if request.Mode == "" {
		request.Mode = CodexControlFileWriteCreateOrReplaceV0
	}
	if request.Perm == 0 {
		request.Perm = 0o600
	}
	return request
}

func validateCodexControlFileWriteRequestV0(request CodexControlFileWriteRequestV0) error {
	if request.RootDir == "." || request.RootDir == "" || !filepath.IsAbs(request.RootDir) {
		return fmt.Errorf("codex_control_file_write: root_invalid")
	}
	if request.Path == "." || request.Path == "" || !filepath.IsAbs(request.Path) {
		return fmt.Errorf("codex_control_file_write: path_invalid")
	}
	if request.FileName == "" || filepath.Base(request.Path) != request.FileName {
		return fmt.Errorf("codex_control_file_write: file_name_mismatch")
	}
	if !codexControlFilePathInsideRootV0(request.Path, request.RootDir) {
		return fmt.Errorf("codex_control_file_write: path_outside_root")
	}
	if request.Mode != CodexControlFileWriteCreateOrReplaceV0 &&
		request.Mode != CodexControlFileWriteCreateIfAbsentV0 {
		return fmt.Errorf("codex_control_file_write: mode_invalid")
	}
	if request.Perm != 0o600 && request.Perm != 0o700 {
		return fmt.Errorf("codex_control_file_write: mode_perm_invalid")
	}
	return nil
}

func codexControlFilePathInsideRootV0(path string, root string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." || filepath.IsAbs(rel) {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

func validateCodexControlFileWriteDirV0(dir string) error {
	info, err := os.Lstat(dir)
	if err != nil {
		return fmt.Errorf("codex_control_file_write: dir_stat_failed")
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("codex_control_file_write: dir_symlink")
	}
	if !info.IsDir() {
		return fmt.Errorf("codex_control_file_write: dir_invalid")
	}
	return nil
}

func codexControlFileWriteOperationV0(request CodexControlFileWriteRequestV0) (string, error) {
	info, err := os.Lstat(request.Path)
	if os.IsNotExist(err) {
		return "created", nil
	}
	if err != nil {
		return "", fmt.Errorf("codex_control_file_write: stat_failed")
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("codex_control_file_write: symlink")
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("codex_control_file_write: not_regular")
	}
	if codexControlFileHasMultipleLinksV0(info) {
		return "", fmt.Errorf("codex_control_file_write: hardlink")
	}
	existing, err := os.ReadFile(request.Path)
	if err != nil {
		return "", fmt.Errorf("codex_control_file_write: read_existing_failed")
	}
	if bytes.Equal(existing, request.Data) {
		return "idempotent", nil
	}
	if request.Mode == CodexControlFileWriteCreateIfAbsentV0 {
		return "", fmt.Errorf("codex_control_file_write: payload_conflict")
	}
	return "replaced", nil
}

func writeCodexControlFileDurableV0(request CodexControlFileWriteRequestV0) error {
	tmp, err := os.CreateTemp(filepath.Dir(request.Path), "."+request.FileName+".*.tmp")
	if err != nil {
		return fmt.Errorf("codex_control_file_write: temp_failed")
	}
	tmpName := tmp.Name()
	keepTemp := false
	defer func() {
		if !keepTemp {
			_ = os.Remove(tmpName)
		}
	}()
	if err := writeCodexControlFileTempV0(tmp, request); err != nil {
		return err
	}
	if err := os.Rename(tmpName, request.Path); err != nil {
		return fmt.Errorf("codex_control_file_write: rename_failed")
	}
	keepTemp = true
	return syncCodexControlFileDirV0(filepath.Dir(request.Path))
}

func writeCodexControlFileTempV0(tmp *os.File, request CodexControlFileWriteRequestV0) error {
	if err := tmp.Chmod(request.Perm); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("codex_control_file_write: chmod_failed")
	}
	if _, err := tmp.Write(request.Data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("codex_control_file_write: write_failed")
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("codex_control_file_write: sync_failed")
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("codex_control_file_write: close_failed")
	}
	return nil
}

func syncCodexControlFileDirV0(dir string) error {
	handle, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("codex_control_file_write: dir_sync_failed")
	}
	defer handle.Close()
	if err := handle.Sync(); err != nil {
		return fmt.Errorf("codex_control_file_write: dir_sync_failed")
	}
	return nil
}

func codexControlFileWriteReceiptV0(
	request CodexControlFileWriteRequestV0,
	operation string,
) CodexControlFileWriteReceiptV0 {
	sum := sha256.Sum256(request.Data)
	hash := hex.EncodeToString(sum[:])
	ref := "control-file-ref-" + safeCodexControlFileReceiptPartV0(request.ControlKind) + "-" + hash[:12]
	return CodexControlFileWriteReceiptV0{
		SchemaVersion:  CodexControlFileWriteReceiptSchemaVersionV0,
		ControlFileRef: ref,
		ControlKind:    request.ControlKind,
		FileName:       request.FileName,
		Operation:      operation,
		Mode:           string(request.Mode),
		SizeBytes:      int64(len(request.Data)),
		SHA256:         hash,
		EvidenceRefs:   []string{ref, "control-file-write-" + operation},
	}
}

func safeCodexControlFileReceiptPartV0(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "_", "-")
	value = strings.ReplaceAll(value, ".", "-")
	if value == "" {
		return "unknown"
	}
	return value
}
