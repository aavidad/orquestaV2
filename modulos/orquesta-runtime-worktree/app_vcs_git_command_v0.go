package orquestaruntimeworktree

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const (
	appVCSDefaultMaxOutputBytesV0  int64 = 64 * 1024
	appVCSDefaultMaxChangedPathsV0       = 2000
)

func (connector GitAppVCSConnectorV0) runGitCommandV0(
	ctx context.Context,
	repo string,
	args ...string,
) (string, *AppVCSIssueV0) {
	timeout := connector.commandTimeoutV0()
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(runCtx, "git", append([]string{"-C", repo}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GCM_INTERACTIVE=Never")
	stdoutPipe, stdoutErr := cmd.StdoutPipe()
	stderrPipe, stderrErr := cmd.StderrPipe()
	if stdoutErr != nil || stderrErr != nil {
		issue := connector.gitCommandIssueV0(AppVCSIssueGitErrorV0, args, "pipe_error=true")
		return "", &issue
	}
	if err := cmd.Start(); err != nil {
		issue := connector.gitCommandIssueV0(AppVCSIssueGitErrorV0, args, "start_error=true")
		return "", &issue
	}
	var stdout, stderr appVCSCapturedOutputV0
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		stdout = appVCSReadLimitedOutputV0(stdoutPipe, connector.maxOutputBytesV0())
	}()
	go func() {
		defer wg.Done()
		stderr = appVCSReadLimitedOutputV0(stderrPipe, connector.maxOutputBytesV0())
	}()
	wg.Wait()
	err := cmd.Wait()
	if runCtx.Err() != nil {
		issue := connector.gitCommandIssueV0(AppVCSIssueGitCommandTimeoutV0, args, "timeout_ms="+fmt.Sprint(timeout.Milliseconds()))
		return "", &issue
	}
	if stdout.Err != nil || stderr.Err != nil {
		issue := connector.gitCommandIssueV0(AppVCSIssueGitErrorV0, args, "read_error=true")
		return "", &issue
	}
	if stdout.Truncated || stderr.Truncated {
		issue := connector.gitCommandIssueV0(
			AppVCSIssueGitOutputTooLargeV0,
			args,
			"max_output_bytes="+fmt.Sprint(connector.maxOutputBytesV0()),
			fmt.Sprintf("stdout_truncated=%t", stdout.Truncated),
			fmt.Sprintf("stderr_truncated=%t", stderr.Truncated),
		)
		return "", &issue
	}
	if err != nil {
		evidence := []string{"exit_error=true"}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			evidence = []string{"exit_code=" + fmt.Sprint(exitErr.ExitCode())}
		}
		issue := connector.gitCommandIssueV0(AppVCSIssueGitErrorV0, args, evidence...)
		return "", &issue
	}
	return stdout.Text, nil
}

type appVCSCapturedOutputV0 struct {
	Text      string
	Truncated bool
	Err       error
}

func appVCSReadLimitedOutputV0(reader io.Reader, maxBytes int64) appVCSCapturedOutputV0 {
	if maxBytes <= 0 {
		maxBytes = appVCSDefaultMaxOutputBytesV0
	}
	buffer := make([]byte, 32*1024)
	capHint := int(maxBytes)
	if capHint > 4096 {
		capHint = 4096
	}
	out := make([]byte, 0, capHint)
	var total int64
	truncated := false
	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			total += int64(n)
			if int64(len(out)) < maxBytes {
				remaining := int(maxBytes - int64(len(out)))
				if n < remaining {
					remaining = n
				}
				out = append(out, buffer[:remaining]...)
			}
			if total > maxBytes {
				truncated = true
			}
		}
		if err == io.EOF {
			return appVCSCapturedOutputV0{Text: string(out), Truncated: truncated}
		}
		if err != nil {
			return appVCSCapturedOutputV0{Text: string(out), Truncated: truncated, Err: err}
		}
	}
}

func (connector GitAppVCSConnectorV0) commandTimeoutV0() time.Duration {
	if connector.CommandTimeout > 0 {
		return connector.CommandTimeout
	}
	return appVCSCommandTimeoutV0
}

func (connector GitAppVCSConnectorV0) maxOutputBytesV0() int64 {
	if connector.MaxOutputBytes > 0 {
		return connector.MaxOutputBytes
	}
	return appVCSDefaultMaxOutputBytesV0
}

func (connector GitAppVCSConnectorV0) maxChangedPathsV0() int {
	if connector.MaxChangedPaths > 0 {
		return connector.MaxChangedPaths
	}
	return appVCSDefaultMaxChangedPathsV0
}

func (connector GitAppVCSConnectorV0) gitStatusPathBudgetIssueV0(paths []string) *AppVCSIssueV0 {
	maxPaths := connector.maxChangedPathsV0()
	if len(paths) <= maxPaths {
		return nil
	}
	issue := connector.gitCommandIssueV0(
		AppVCSIssueGitStatusTooManyPathsV0,
		[]string{"status"},
		"changed_path_count="+fmt.Sprint(len(paths)),
		"max_changed_paths="+fmt.Sprint(maxPaths),
	)
	return &issue
}

func (connector GitAppVCSConnectorV0) gitCommandIssueV0(
	code AppVCSIssueCodeV0,
	args []string,
	evidence ...string,
) AppVCSIssueV0 {
	base := []string{"action=" + appVCSGitActionV0(args)}
	return appVCSIssueV0(code, gitIssueFieldV0(args), append(base, evidence...)...)
}

func appVCSGitActionV0(args []string) string {
	if len(args) == 0 {
		return "git"
	}
	action := strings.TrimSpace(args[0])
	if action == "" {
		return "git"
	}
	action = strings.ReplaceAll(action, "-", "_")
	return "git." + action
}
