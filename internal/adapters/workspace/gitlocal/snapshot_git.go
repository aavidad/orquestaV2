package gitlocal

import (
	"bufio"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"hash"
	"io"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"unicode/utf8"

	"orquesta/internal/ports"
)

type snapshotRepository struct {
	adapter   *Adapter
	workspace *os.File
}

func (adapter *Adapter) openSnapshotRepository(ctx context.Context, subject ports.TestSubject) (*snapshotRepository, error) {
	repository, _, err := adapter.repository(ctx, subject.RepositoryRef)
	if err != nil {
		return nil, err
	}
	if err = adapter.verifyWorkspaceBindingMarker(ctx, repository, subject); err != nil {
		return nil, err
	}
	workspace, _, err := adapter.verifyWorkspaceReadOnly(
		ctx, repository, adapter.workspacePath(subject.WorkspaceRef), "", subject.WorkspaceRef,
	)
	if err != nil {
		return nil, err
	}
	headRaw, err := adapter.gitRun(ctx, repository, nil,
		"rev-parse", "--verify", "refs/heads/"+workspaceBranch(subject.WorkspaceRef))
	head := trimOID(headRaw)
	if err != nil || ports.ValidateGitOID(head, subject.ObjectFormat) != nil || head != subject.HeadOID {
		_ = workspace.Close()
		return nil, &Error{Code: CodeSnapshotInvalid}
	}
	snapshot := &snapshotRepository{adapter, workspace}
	if err := verifySnapshotChangeMarker(ctx, snapshot, subject.ObjectFormat,
		snapshotChangeMarkerRef(subject.ChangeSetRef), snapshotChangeSubjectMarkerMessage(subject), CodeSnapshotInvalid); err != nil {
		_ = snapshot.Close()
		return nil, err
	}
	return snapshot, nil
}

func (repository *snapshotRepository) Close() error {
	if repository == nil || repository.workspace == nil {
		return nil
	}
	err := errors.Join(verifySnapshotDirectoryBinding(repository.workspace.Name(), repository.workspace), repository.workspace.Close())
	repository.workspace = nil
	return err
}

func (repository *snapshotRepository) run(ctx context.Context, maxBytes int64, args ...string) ([]byte, error) {
	command, cleanup, err := repository.adapter.pinnedGitCommand(ctx, []*os.File{repository.workspace}, gitArguments("/proc/self/fd/4", args...)...)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	command.Env = gitEnvironment(repository.adapter.root)
	output, err := repository.adapter.runGitCommand(ctx, command, nil)
	if err != nil {
		return nil, err
	}
	if int64(len(output)) > maxBytes {
		return nil, &Error{Code: CodeSnapshotLimit}
	}
	return output, nil
}

type gitObjectSession struct {
	format  ports.GitObjectFormat
	command *exec.Cmd
	stdin   io.WriteCloser
	stdout  *bufio.Reader
	broken  bool
}

func openGitObjectSession(ctx context.Context, repository *snapshotRepository, format ports.GitObjectFormat) (*gitObjectSession, error) {
	command, cleanup, err := repository.adapter.pinnedGitCommand(ctx, []*os.File{repository.workspace},
		gitArguments("/proc/self/fd/4", "cat-file", "--batch")...)
	if err != nil {
		return nil, err
	}
	command.Env, command.Stderr = gitEnvironment(repository.adapter.root), io.Discard
	stdin, inputErr := command.StdinPipe()
	stdout, outputErr := command.StdoutPipe()
	if inputErr != nil || outputErr != nil {
		cleanup()
		if stdin != nil {
			_ = stdin.Close()
		}
		return nil, &Error{Code: CodeGitFailed, Cause: errors.Join(inputErr, outputErr)}
	}
	if err := command.Start(); err != nil {
		cleanup()
		_ = stdin.Close()
		return nil, &Error{Code: CodeGitFailed, Cause: err}
	}
	cleanup()
	return &gitObjectSession{format, command, stdin, bufio.NewReaderSize(stdout, 256), false}, nil
}

func (session *gitObjectSession) Close() error {
	if session == nil {
		return nil
	}
	inputErr := session.stdin.Close()
	if session.broken {
		_ = killPinnedProcessGroup(session.command)
	}
	waitErr := waitPinnedCommand(session.command)
	if session.broken && waitErr != nil {
		waitErr = &Error{Code: CodeGitFailed, Cause: waitErr}
	}
	return errors.Join(inputErr, waitErr)
}

func (session *gitObjectSession) read(oid, kind string, maxBytes int64,
	consume func(int64, io.Reader) error,
) (resultErr error) {
	if maxBytes < 0 {
		return &Error{Code: CodeSnapshotInvalid}
	}
	defer func() {
		if resultErr != nil {
			session.broken = true
		}
	}()
	if _, err := io.WriteString(session.stdin, oid+"\n"); err != nil {
		return &Error{Code: CodeSnapshotStream, Cause: err}
	}
	line, err := session.stdout.ReadSlice('\n')
	fields := strings.Fields(string(line))
	if err != nil || len(line) > 256 || len(fields) != 3 {
		return &Error{Code: CodeSnapshotInvalid, Cause: err}
	}
	size, err := strconv.ParseInt(fields[2], 10, 64)
	if err != nil || fields[0] != oid || fields[1] != kind || size < 0 || size > maxBytes {
		if size > maxBytes {
			return &Error{Code: CodeSnapshotLimit}
		}
		return &Error{Code: CodeSnapshotInvalid, Cause: err}
	}
	digest := snapshotObjectHash(session.format)
	_, _ = io.WriteString(digest, kind+" "+strconv.FormatInt(size, 10)+"\x00")
	limited := &io.LimitedReader{R: session.stdout, N: size}
	if err := consume(size, io.TeeReader(limited, digest)); err != nil {
		return err
	}
	terminator, err := session.stdout.ReadByte()
	if limited.N != 0 || err != nil || terminator != '\n' {
		return &Error{Code: CodeSnapshotStream, Cause: io.ErrUnexpectedEOF}
	}
	if hex.EncodeToString(digest.Sum(nil)) != oid {
		return &Error{Code: CodeSnapshotHash}
	}
	return nil
}

func snapshotObjectHash(format ports.GitObjectFormat) hash.Hash {
	if format == ports.GitObjectFormatSHA256 {
		return sha256.New()
	}
	return sha1.New()
}

func snapshotEntryMode(mode string) (bool, error) {
	switch mode {
	case "040000":
		return true, nil
	case "100644", "100755", "120000":
		return false, nil
	case "160000":
		return false, &Error{Code: CodeSnapshotGitlink}
	default:
		return false, &Error{Code: CodeSnapshotInvalid}
	}
}

func validSnapshotPath(value string) bool {
	if value == "" || !utf8.ValidString(value) || path.IsAbs(value) || path.Clean(value) != value || strings.IndexByte(value, 0) >= 0 {
		return false
	}
	for _, component := range strings.Split(value, "/") {
		if component == "" || component == "." || component == ".." || strings.EqualFold(component, ".git") {
			return false
		}
	}
	return true
}
