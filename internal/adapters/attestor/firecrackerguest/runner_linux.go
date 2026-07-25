//go:build linux

package firecrackerguest

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"

	firecracker "orquesta/internal/testattestorprotocol/rawdrive"
)

type RunRequest struct {
	InputDrive       io.Reader
	InputDriveBytes  uint64
	OutputDrive      firecracker.OutputDrive
	OutputDriveBytes uint64
	ScratchRoot      string
	GoBinary         string
	Executor         Executor
	NetworkCheck     func() error
	ScratchCheck     func(string) error
	CapacityCheck    func(string, uint64) error
	PrepareIdentity  func([]string) error
	CleanupTree      func(context.Context, string, uint64) error
}

// Run validates input, materializes the authenticated snapshot, executes every
// required Go test, and commits exactly one output. It never reboots.
func Run(ctx context.Context, request RunRequest) (resultErr error) {
	if request.InputDrive == nil || request.OutputDrive == nil || request.Executor == nil ||
		request.NetworkCheck == nil || request.ScratchCheck == nil || request.CapacityCheck == nil ||
		request.PrepareIdentity == nil || request.CleanupTree == nil ||
		request.ScratchRoot == "" || !filepath.IsAbs(request.ScratchRoot) ||
		request.GoBinary == "" || !filepath.IsAbs(request.GoBinary) {
		return guestError(CodeScratchUnavailable, nil)
	}
	if err := request.ScratchCheck(request.ScratchRoot); err != nil {
		return guestError(CodeScratchUnavailable, err)
	}
	initialRequired, ok := checkedScratchBytes(request.InputDriveBytes, scratchFixedReserveBytes)
	if !ok {
		return guestError(CodeScratchCapacity, nil)
	}
	if err := request.CapacityCheck(request.ScratchRoot, initialRequired); err != nil {
		return guestError(CodeScratchCapacity, err)
	}
	snapshotFile, err := os.CreateTemp(request.ScratchRoot, "snapshot-")
	if err != nil {
		return guestError(CodeScratchUnavailable, err)
	}
	snapshotPath := snapshotFile.Name()
	workspace := ""
	workspaceCleanupLimit := maximumCleanupRecords()
	cleanupResources := func() error {
		var cleanupErr error
		if snapshotFile != nil {
			cleanupErr = errors.Join(cleanupErr, snapshotFile.Close())
			snapshotFile = nil
		}
		if snapshotPath != "" {
			if err := os.Remove(snapshotPath); err != nil && !os.IsNotExist(err) {
				cleanupErr = errors.Join(cleanupErr, err)
			}
			snapshotPath = ""
		}
		if workspace != "" {
			cleanupErr = errors.Join(
				cleanupErr,
				request.CleanupTree(context.WithoutCancel(ctx), workspace, workspaceCleanupLimit),
			)
			workspace = ""
		}
		return cleanupErr
	}
	defer func() {
		if err := cleanupResources(); err != nil {
			resultErr = errors.Join(guestError(CodeCleanupFailed, err), resultErr)
		}
	}()
	input, err := firecracker.ReadInputDrive(
		contextReader{ctx: ctx, reader: request.InputDrive}, request.InputDriveBytes, snapshotFile,
	)
	if err != nil {
		return guestError(CodeInputInvalid, err)
	}
	commitOutput := func(output firecracker.Output) error {
		cleanupErr := cleanupResources()
		if cleanupErr != nil {
			output = firecracker.Output{
				SubjectDigest: input.SubjectDigest,
				RunNonce:      input.RunNonce,
				ErrorCode:     CodeCleanupFailed,
			}
		}
		if err := firecracker.WriteOutputDrive(request.OutputDrive, request.OutputDriveBytes, output); err != nil {
			return errors.Join(
				guestError(CodeOutputWriteFailed, err),
				guestError(CodeCleanupFailed, cleanupErr),
			)
		}
		if cleanupErr != nil {
			return guestError(CodeCleanupFailed, cleanupErr)
		}
		return nil
	}
	writeError := func(code string) error {
		return commitOutput(firecracker.Output{
			SubjectDigest: input.SubjectDigest, RunNonce: input.RunNonce, ErrorCode: code,
		})
	}
	snapshotInfo, statErr := snapshotFile.Stat()
	if statErr != nil || snapshotInfo.Size() < 0 {
		return errors.Join(writeError(CodeScratchCapacity), guestError(CodeScratchCapacity, statErr))
	}
	snapshotBytes := uint64(snapshotInfo.Size())
	requiredAfterInput, ok := requiredScratchPeak(snapshotBytes, input.MaxOutputBytes)
	if !ok {
		return errors.Join(writeError(CodeScratchCapacity), guestError(CodeScratchCapacity, nil))
	}
	if err := request.CapacityCheck(request.ScratchRoot, requiredAfterInput); err != nil {
		return errors.Join(writeError(CodeScratchCapacity), guestError(CodeScratchCapacity, err))
	}
	if err := request.NetworkCheck(); err != nil {
		return errors.Join(writeError(CodeNetworkAvailable), guestError(CodeNetworkAvailable, err))
	}
	if err := snapshotFile.Chmod(0o400); err != nil {
		return errors.Join(writeError(CodeSnapshotInvalid), guestError(CodeSnapshotInvalid, err))
	}
	if _, err := snapshotFile.Seek(0, io.SeekStart); err != nil {
		return errors.Join(writeError(CodeSnapshotInvalid), guestError(CodeSnapshotInvalid, err))
	}
	workspace, err = os.MkdirTemp(request.ScratchRoot, "subject-")
	if err != nil {
		return errors.Join(writeError(CodeMaterializeFailed), guestError(CodeMaterializeFailed, err))
	}
	identity, err := MaterializeSnapshot(ctx, snapshotFile, workspace, input.SubjectDigest)
	if err != nil {
		code := ErrorCode(err)
		if ctx.Err() != nil {
			code = CodeExecutionTimeout
		}
		if code == "" {
			code = CodeSnapshotInvalid
		}
		return errors.Join(writeError(code), err)
	}
	if err := snapshotFile.Close(); err != nil {
		return errors.Join(writeError(CodeCleanupFailed), guestError(CodeCleanupFailed, err))
	}
	snapshotFile = nil
	if err := os.Remove(snapshotPath); err != nil {
		return errors.Join(writeError(CodeCleanupFailed), guestError(CodeCleanupFailed, err))
	}
	snapshotPath = ""
	workspaceCleanupLimit, ok = checkedScratchBytes(identity.Entries, identity.Directories, 1)
	if !ok || workspaceCleanupLimit > maximumCleanupRecords() {
		return errors.Join(writeError(CodeCleanupFailed), guestError(CodeCleanupFailed, nil))
	}
	outcomes := make([]firecracker.RequiredTestOutcome, 0, len(input.RequiredTests))
	verdict := firecracker.VerdictPassed
	for _, required := range input.RequiredTests {
		arguments, err := canonicalGoTestArguments(required)
		if err != nil {
			return errors.Join(writeError(CodeToolUnsupported), err)
		}
		workingDirectory, err := resolveWorkingDirectory(workspace, required.WorkingDirectory)
		if err != nil {
			return errors.Join(writeError(CodeSnapshotInvalid), err)
		}
		result, err := executeRequiredTest(
			ctx, request, input.MaxOutputBytes, workspace, workingDirectory, arguments,
		)
		if err != nil {
			code := ErrorCode(err)
			if code != CodeScratchUnavailable && code != CodeIdentityFailed && code != CodeCleanupFailed {
				code = CodeExecutionFailed
			}
			return errors.Join(writeError(code), err)
		}
		if result.TimedOut || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return writeError(CodeExecutionTimeout)
		}
		if result.OutputLimit {
			return writeError(CodeOutputLimit)
		}
		if result.OutputDigest == "" {
			return writeError(CodeExecutionFailed)
		}
		if result.ExitCode == 0 && result.TestCases == 0 {
			result.ExitCode = NoTestsExitCode
		}
		outcomes = append(outcomes, firecracker.RequiredTestOutcome{
			RequiredTestRef: required.Ref, ExitCode: result.ExitCode, OutputDigest: result.OutputDigest,
		})
		if result.ExitCode != 0 {
			verdict = firecracker.VerdictFailed
		}
	}
	return commitOutput(firecracker.Output{
		SubjectDigest: input.SubjectDigest, RunNonce: input.RunNonce,
		Verdict: verdict, Outcomes: outcomes,
	})
}

func executeRequiredTest(
	ctx context.Context,
	request RunRequest,
	maxOutputBytes uint64,
	workspace, workingDirectory string,
	arguments []string,
) (result ExecutionResult, resultErr error) {
	processScratch, err := os.MkdirTemp(request.ScratchRoot, "process-")
	if err != nil {
		return ExecutionResult{}, guestError(CodeScratchUnavailable, err)
	}
	defer func() {
		if err := request.CleanupTree(
			context.WithoutCancel(ctx), processScratch, maximumCleanupRecords(),
		); err != nil {
			result = ExecutionResult{}
			resultErr = errors.Join(guestError(CodeCleanupFailed, err), resultErr)
		}
	}()
	identityPaths, err := prepareProcessScratch(processScratch)
	if err != nil {
		return ExecutionResult{}, guestError(CodeScratchUnavailable, err)
	}
	if err := request.PrepareIdentity(identityPaths); err != nil {
		return ExecutionResult{}, guestError(CodeIdentityFailed, err)
	}
	return request.Executor.Execute(ctx, Execution{
		GoBinary:  request.GoBinary,
		Workspace: workspace, WorkingDirectory: workingDirectory,
		Arguments: arguments, Environment: minimalEnvironment(processScratch),
		OutputDirectory: processScratch, MaxOutputBytes: maxOutputBytes,
		UID: nobodyID, GID: nobodyID, ClearSupplementaryGroups: true,
	})
}

func canonicalGoTestArguments(required firecracker.RequiredTest) ([]string, error) {
	if required.ToolRef != "tool:go" || len(required.Arguments) == 0 || required.Arguments[0] != "test" {
		return nil, guestError(CodeToolUnsupported, nil)
	}
	arguments := []string{"test", "-json"}
	for index, argument := range required.Arguments[1:] {
		if argument == "-json" || argument == "--json" ||
			strings.HasPrefix(argument, "-json=") || strings.HasPrefix(argument, "--json=") {
			return nil, guestError(CodeToolUnsupported, nil)
		}
		arguments = append(arguments, argument)
		if argument == "-args" {
			arguments = append(arguments, required.Arguments[index+2:]...)
			break
		}
	}
	return arguments, nil
}

func resolveWorkingDirectory(root, relative string) (string, error) {
	components := strings.Split(relative, "/")
	fd, err := unix.Open(root, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return "", guestError(CodeSnapshotInvalid, err)
	}
	defer unix.Close(fd)
	for _, component := range components {
		if component == "." {
			continue
		}
		next, err := unix.Openat(fd, component, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		if err != nil {
			return "", guestError(CodeSnapshotInvalid, err)
		}
		_ = unix.Close(fd)
		fd = next
	}
	return filepath.Join(root, filepath.FromSlash(relative)), nil
}

func prepareProcessScratch(root string) ([]string, error) {
	names := []string{"gocache", "gopath", "gotmp", "home", "tmp"}
	paths := make([]string, 0, len(names)+1)
	paths = append(paths, root)
	for _, name := range names {
		value := filepath.Join(root, name)
		if err := os.Mkdir(value, 0o700); err != nil {
			return nil, err
		}
		paths = append(paths, value)
	}
	return paths, nil
}

func PrepareNobody(paths []string) error {
	for _, name := range paths {
		if err := os.Chown(name, nobodyID, nobodyID); err != nil {
			return err
		}
		if err := os.Chmod(name, 0o700); err != nil {
			return err
		}
	}
	return nil
}

func RequireNoNetwork() error {
	interfaces, err := net.Interfaces()
	if err != nil {
		return err
	}
	for _, networkInterface := range interfaces {
		if networkInterface.Name != "lo" || networkInterface.Flags&net.FlagUp != 0 {
			return guestError(CodeNetworkAvailable, nil)
		}
	}
	return nil
}

func RequirePrivateTmpfs(root string) error {
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0o711 {
		return guestError(CodeScratchUnavailable, err)
	}
	ownerStat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || ownerStat.Uid != 0 || ownerStat.Gid != 0 {
		return guestError(CodeScratchUnavailable, nil)
	}
	var filesystemStat unix.Statfs_t
	if err := unix.Statfs(root, &filesystemStat); err != nil {
		return err
	}
	if filesystemStat.Type != unix.TMPFS_MAGIC {
		return guestError(CodeScratchUnavailable, nil)
	}
	requiredFlags := int64(unix.ST_NODEV | unix.ST_NOSUID)
	if filesystemStat.Flags&requiredFlags != requiredFlags ||
		filesystemStat.Flags&int64(unix.ST_NOEXEC) != 0 {
		return guestError(CodeScratchUnavailable, nil)
	}
	distinct, err := distinctMountFromParent(root)
	if err != nil || !distinct {
		return guestError(CodeScratchUnavailable, err)
	}
	return nil
}

func distinctMountFromParent(root string) (bool, error) {
	cleanRoot := filepath.Clean(root)
	parent := filepath.Dir(cleanRoot)
	if parent == cleanRoot {
		return false, nil
	}
	rootMount, rootAvailable, rootErr := mountID(cleanRoot)
	parentMount, parentAvailable, parentErr := mountID(parent)
	if rootErr != nil || parentErr != nil {
		return false, errors.Join(rootErr, parentErr)
	}
	if rootAvailable && parentAvailable {
		return rootMount != parentMount, nil
	}
	var rootStat unix.Stat_t
	var parentStat unix.Stat_t
	if err := unix.Lstat(cleanRoot, &rootStat); err != nil {
		return false, err
	}
	if err := unix.Lstat(parent, &parentStat); err != nil {
		return false, err
	}
	return rootStat.Dev != parentStat.Dev, nil
}

func mountID(path string) (uint64, bool, error) {
	var status unix.Statx_t
	err := unix.Statx(
		unix.AT_FDCWD,
		path,
		unix.AT_NO_AUTOMOUNT|unix.AT_SYMLINK_NOFOLLOW,
		unix.STATX_MNT_ID,
		&status,
	)
	if err == nil {
		if status.Mask&unix.STATX_MNT_ID == 0 {
			return 0, false, nil
		}
		return status.Mnt_id, true, nil
	}
	if errors.Is(err, unix.ENOSYS) || errors.Is(err, unix.EINVAL) ||
		errors.Is(err, unix.EOPNOTSUPP) {
		return 0, false, nil
	}
	return 0, false, err
}

const (
	scratchFixedReserveBytes = 64 << 20
	scratchCacheReserveBytes = 256 << 20
)

func requiredScratchPeak(snapshotBytes, maxOutputBytes uint64) (uint64, bool) {
	materializationPeak, ok := checkedScratchBytes(
		snapshotBytes, snapshotBytes, scratchFixedReserveBytes,
	)
	if !ok {
		return 0, false
	}
	executionPeak, ok := checkedScratchBytes(
		snapshotBytes, scratchCacheReserveBytes, maxOutputBytes, scratchFixedReserveBytes,
	)
	if !ok {
		return 0, false
	}
	return max(materializationPeak, executionPeak), true
}

func RequireScratchCapacity(root string, required uint64) error {
	var filesystemStat unix.Statfs_t
	if err := unix.Statfs(root, &filesystemStat); err != nil {
		return err
	}
	available, ok := checkedScratchProduct(uint64(filesystemStat.Bavail), uint64(filesystemStat.Bsize))
	if !ok || available < required {
		return guestError(CodeScratchCapacity, nil)
	}
	totalFilesystem, ok := checkedScratchProduct(uint64(filesystemStat.Blocks), uint64(filesystemStat.Bsize))
	if !ok {
		return guestError(CodeScratchCapacity, nil)
	}
	var memory unix.Sysinfo_t
	if err := unix.Sysinfo(&memory); err != nil {
		return err
	}
	memoryUnit := uint64(memory.Unit)
	if memoryUnit == 0 {
		memoryUnit = 1
	}
	totalRAM, ok := checkedScratchProduct(uint64(memory.Totalram), memoryUnit)
	if !ok {
		return guestError(CodeScratchCapacity, nil)
	}
	maximum := totalRAM - totalRAM/4
	maximumWithRounding, ok := checkedScratchBytes(maximum, uint64(os.Getpagesize()))
	if !ok || totalFilesystem > maximumWithRounding {
		return guestError(CodeScratchCapacity, nil)
	}
	return nil
}

func checkedScratchBytes(values ...uint64) (uint64, bool) {
	var total uint64
	for _, value := range values {
		if value > ^uint64(0)-total {
			return 0, false
		}
		total += value
	}
	return total, true
}

func checkedScratchProduct(left, right uint64) (uint64, bool) {
	if left != 0 && right > ^uint64(0)/left {
		return 0, false
	}
	return left * right, true
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (reader contextReader) Read(value []byte) (int, error) {
	if err := reader.ctx.Err(); err != nil {
		return 0, err
	}
	return reader.reader.Read(value)
}

func maximumCleanupRecords() uint64 {
	return uint64(maxSnapshotEntries) + 1
}

type cleanupDirectoryFrame struct {
	path       string
	directory  *os.File
	entries    []os.DirEntry
	entryIndex int
}

func cleanupPrivateTree(ctx context.Context, root string, maximumRecords uint64) error {
	if maximumRecords == 0 {
		return unix.EFBIG
	}
	info, err := os.Lstat(root)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		if err := ctx.Err(); err != nil {
			return err
		}
		return os.Remove(root)
	}
	if err := os.Chmod(root, 0o700); err != nil {
		return err
	}
	directory, err := os.Open(root)
	if err != nil {
		return err
	}
	stack := []cleanupDirectoryFrame{{path: root, directory: directory}}
	records := uint64(1)
	closeStack := func() error {
		var closeErr error
		for index := len(stack) - 1; index >= 0; index-- {
			closeErr = errors.Join(closeErr, stack[index].directory.Close())
		}
		stack = nil
		return closeErr
	}
	for len(stack) > 0 {
		if err := ctx.Err(); err != nil {
			return errors.Join(err, closeStack())
		}
		frame := &stack[len(stack)-1]
		if frame.entryIndex >= len(frame.entries) {
			frame.entries, err = frame.directory.ReadDir(256)
			frame.entryIndex = 0
			if errors.Is(err, io.EOF) {
				closeErr := frame.directory.Close()
				removeErr := os.Remove(frame.path)
				stack = stack[:len(stack)-1]
				if joined := errors.Join(closeErr, removeErr); joined != nil {
					return errors.Join(joined, closeStack())
				}
				continue
			}
			if err != nil {
				return errors.Join(err, closeStack())
			}
			continue
		}
		entry := frame.entries[frame.entryIndex]
		frame.entryIndex++
		if records >= maximumRecords {
			return errors.Join(unix.EFBIG, closeStack())
		}
		records++
		name := filepath.Join(frame.path, entry.Name())
		if !entry.IsDir() {
			if err := os.Remove(name); err != nil {
				return errors.Join(err, closeStack())
			}
			continue
		}
		if err := os.Chmod(name, 0o700); err != nil {
			return errors.Join(err, closeStack())
		}
		child, err := os.Open(name)
		if err != nil {
			return errors.Join(err, closeStack())
		}
		stack = append(stack, cleanupDirectoryFrame{path: name, directory: child})
	}
	return nil
}
