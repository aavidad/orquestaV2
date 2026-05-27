package orquestaruntimecodex

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

const CodexControlFileMaxBytesV0 int64 = 256 * 1024

type CodexControlFileReadErrorV0 struct {
	FileName  string
	Reason    string
	Retryable bool
}

func (err CodexControlFileReadErrorV0) Error() string {
	reason := strings.TrimSpace(err.Reason)
	if reason == "" {
		reason = "read_failed"
	}
	return "codex_control_file_read_failed:" + reason
}

func ReadCodexControlFileBytesV0(path string, fileName string) ([]byte, error) {
	fileName = codexControlFileNameV0(fileName)
	path = strings.TrimSpace(path)
	info, err := validateCodexControlFilePathBeforeOpenV0(path, fileName)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, codexControlFileReadErrorV0(fileName, codexControlFileOpenReasonV0(err))
	}
	defer file.Close()
	openedInfo, err := file.Stat()
	if err != nil {
		return nil, codexControlFileReadErrorV0(fileName, "stat_failed")
	}
	if !os.SameFile(info, openedInfo) {
		return nil, codexControlFileReadErrorV0(fileName, "control_file_changed")
	}
	data, err := io.ReadAll(io.LimitReader(file, CodexControlFileMaxBytesV0+1))
	if err != nil {
		return nil, codexControlFileReadErrorV0(fileName, "read_failed")
	}
	if int64(len(data)) > CodexControlFileMaxBytesV0 {
		return nil, codexControlFileReadErrorV0(fileName, "control_file_too_large")
	}
	return data, nil
}

func validateCodexControlFilePathBeforeOpenV0(path string, fileName string) (os.FileInfo, error) {
	if path == "" || filepath.Base(path) != fileName || strings.Contains(filepath.Base(path), string(os.PathSeparator)) {
		return nil, codexControlFileReadErrorV0(fileName, "file_name_mismatch")
	}
	dir := filepath.Clean(filepath.Dir(path))
	if dir == "." || dir == string(os.PathSeparator) {
		return nil, codexControlFileReadErrorV0(fileName, "control_file_root_invalid")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return nil, codexControlFileReadErrorV0(fileName, codexControlFileOpenReasonV0(err))
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, codexControlFileReadErrorV0(fileName, "symlink")
	}
	if !info.Mode().IsRegular() {
		return nil, codexControlFileReadErrorV0(fileName, "not_regular")
	}
	if codexControlFileHasMultipleLinksV0(info) {
		return nil, codexControlFileReadErrorV0(fileName, "hardlink")
	}
	return info, nil
}

func codexControlFileHasMultipleLinksV0(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Nlink > 1
}

func CodexControlFileReadIssueV0(
	err error,
	fileName string,
	correlationID string,
	notReadyEvidence string,
) orquestaruntime.ExternalAgentConnectorErrorV0 {
	reason, retryable := CodexControlFileReadErrorReasonV0(err)
	if reason == "" {
		reason = "read_failed"
	}
	evidence := reason
	if retryable && strings.TrimSpace(notReadyEvidence) != "" {
		evidence = strings.TrimSpace(notReadyEvidence)
	}
	issue := codexIssueV0(CodexConnectorAckInvalidV0, codexControlFileNameV0(fileName), correlationID, evidence)
	issue.Retryable = retryable
	return issue
}

func CodexControlFileReadErrorReasonV0(err error) (string, bool) {
	var readErr CodexControlFileReadErrorV0
	if errors.As(err, &readErr) {
		return strings.TrimSpace(readErr.Reason), readErr.Retryable
	}
	return "read_failed", false
}

func codexControlFileReadErrorV0(fileName string, reason string) CodexControlFileReadErrorV0 {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "read_failed"
	}
	return CodexControlFileReadErrorV0{
		FileName:  codexControlFileNameV0(fileName),
		Reason:    reason,
		Retryable: reason == "not_found",
	}
}

func codexControlFileOpenReasonV0(err error) string {
	switch {
	case errors.Is(err, os.ErrNotExist):
		return "not_found"
	case errors.Is(err, os.ErrPermission):
		return "permission_denied"
	default:
		return "read_failed"
	}
}

func codexControlFileNameV0(fileName string) string {
	fileName = strings.TrimSpace(fileName)
	if fileName == "" {
		return "control_file"
	}
	return fileName
}
