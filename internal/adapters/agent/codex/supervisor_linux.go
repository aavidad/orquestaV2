//go:build linux

package codex

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

const supervisorRequiredSeals = unix.F_SEAL_WRITE | unix.F_SEAL_GROW | unix.F_SEAL_SHRINK | unix.F_SEAL_SEAL

func platformSupervisorCommand(
	runContext context.Context,
	envelope supervisorEnvelope,
) (supervisorCommand, error) {
	payload, err := json.Marshal(envelope)
	if err != nil {
		return supervisorCommand{}, err
	}
	defer clearBytes(payload)
	if len(payload) == 0 || len(payload) > maxSupervisorEnvelopeBytes {
		return supervisorCommand{}, errors.New("supervisor envelope too large")
	}
	fd, err := unix.MemfdCreate("orquesta-codex-supervisor", unix.MFD_CLOEXEC|unix.MFD_ALLOW_SEALING)
	if err != nil {
		return supervisorCommand{}, err
	}
	sealed := os.NewFile(uintptr(fd), "orquesta-codex-supervisor")
	fail := func(cause error) (supervisorCommand, error) {
		_ = sealed.Close()
		return supervisorCommand{}, cause
	}
	if count, err := sealed.Write(payload); err != nil || count != len(payload) {
		return fail(errors.New("write supervisor envelope"))
	}
	if err := unix.Fchmod(int(sealed.Fd()), 0o400); err != nil {
		return fail(err)
	}
	if _, err := sealed.Seek(0, io.SeekStart); err != nil {
		return fail(err)
	}
	if _, err := unix.FcntlInt(sealed.Fd(), unix.F_ADD_SEALS, supervisorRequiredSeals); err != nil {
		return fail(err)
	}
	got, err := unix.FcntlInt(sealed.Fd(), unix.F_GET_SEALS, 0)
	if err != nil || got&supervisorRequiredSeals != supervisorRequiredSeals {
		return fail(errors.New("seal supervisor envelope"))
	}
	gateReader, gateWriter, err := os.Pipe()
	if err != nil {
		return fail(err)
	}
	diagnosticReader, diagnosticWriter, err := os.Pipe()
	if err != nil {
		_ = gateReader.Close()
		_ = gateWriter.Close()
		return fail(err)
	}
	executable, err := os.Executable()
	if err != nil {
		_ = gateReader.Close()
		_ = gateWriter.Close()
		_ = diagnosticReader.Close()
		_ = diagnosticWriter.Close()
		return fail(err)
	}
	command := exec.CommandContext(runContext, executable, localSupervisorArgument)
	command.ExtraFiles = []*os.File{gateReader, sealed, diagnosticWriter}
	return supervisorCommand{
		command: command, gateReader: gateReader, gateWriter: gateWriter, sealed: sealed,
		diagnosticReader: diagnosticReader, diagnosticWriter: diagnosticWriter,
	}, nil
}

func platformRunLocalSupervisor() error {
	gate := os.NewFile(3, "orquesta-codex-supervisor-gate")
	envelopeFile := os.NewFile(4, "orquesta-codex-supervisor-envelope")
	diagnosticOutput := os.NewFile(5, "orquesta-codex-supervisor-diagnostic")
	if gate == nil || envelopeFile == nil || diagnosticOutput == nil {
		return errors.New("supervisor descriptors unavailable")
	}
	unix.CloseOnExec(int(diagnosticOutput.Fd()))
	defer gate.Close()
	defer envelopeFile.Close()
	defer diagnosticOutput.Close()
	seals, err := unix.FcntlInt(envelopeFile.Fd(), unix.F_GET_SEALS, 0)
	if err != nil || seals&supervisorRequiredSeals != supervisorRequiredSeals {
		return errors.New("supervisor envelope is not sealed")
	}
	envelope, err := decodeSupervisorEnvelope(envelopeFile)
	if err != nil {
		return err
	}
	defer clearSupervisorEnvelope(&envelope)
	var record processRecord
	decoder := json.NewDecoder(io.LimitReader(gate, 32<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&record); err != nil {
		return err
	}
	if err := requireJSONEOF(decoder); err != nil {
		return err
	}
	if record.SchemaVersion != supervisedProcessSchemaVersion ||
		record.SupervisorInstance != envelope.SupervisorInstance ||
		record.CompletionPublicKey != envelope.CompletionPublicKey ||
		record.ExecutionRef != envelope.ExecutionRef || record.RequestHash != envelope.RequestHash ||
		record.RuntimeScope != envelope.RuntimeScope || record.PID != os.Getpid() {
		return errors.New("supervisor gate identity mismatch")
	}
	pgid, bootID, birthMarker, err := platformCaptureProcess(os.Getpid())
	if err != nil || pgid != record.PGID || bootID != record.BootID || birthMarker != record.BirthMarker {
		return errors.New("supervisor process identity mismatch")
	}
	_ = gate.Close()
	_ = envelopeFile.Close()
	return superviseCodex(envelope, record, diagnosticOutput)
}

func superviseCodex(envelope supervisorEnvelope, record processRecord, diagnosticOutput *os.File) error {
	if err := validateSupervisorFilesystem(envelope); err != nil {
		return err
	}
	diagnostic := &cappedDiagnostic{maximum: envelope.MaxDiagnosticBytes}
	command := exec.Command(envelope.Command, envelope.Arguments...)
	command.Dir = envelope.WorkingDirectory
	command.Env = append([]string(nil), envelope.Environment...)
	command.Stdin = stringsReaderAndClear(envelope.Prompt)
	command.Stdout = nil
	command.Stderr = diagnostic
	command.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGKILL}
	_, pipeDelay := supervisorDurations(envelope)
	command.WaitDelay = pipeDelay
	if err := command.Start(); err != nil {
		clearEnvironment(command.Env)
		command.Env = nil
		return publishStartedFailure(envelope, record, diagnostic, diagnosticOutput, err)
	}
	clearEnvironment(command.Env)
	command.Env = nil
	workerPGID, workerBootID, workerBirth, err := platformCaptureProcess(command.Process.Pid)
	if err != nil || workerPGID != record.PGID {
		_ = killProcessGroupMembersExcept(record.PGID, os.Getpid(), syscall.SIGKILL)
		_ = command.Wait()
		return errors.New("capture supervised worker")
	}

	signalChannel := make(chan os.Signal, 2)
	signal.Notify(signalChannel, syscall.SIGTERM, syscall.SIGINT)
	defer signal.Stop(signalChannel)
	waited := make(chan error, 1)
	go func() { waited <- command.Wait() }()
	timeout, pipeDrain := supervisorDurations(envelope)
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	cause := supervisorCauseNatural
	var waitErr error
	waiting := true
	for waiting {
		select {
		case waitErr = <-waited:
			waiting = false
		case <-timer.C:
			cause = supervisorCauseTimeout
			if err := killProcessGroupMembersExcept(record.PGID, os.Getpid(), syscall.SIGKILL); err != nil {
				return err
			}
			select {
			case waitErr = <-waited:
			case <-time.After(pipeDrain):
				return errors.New("supervisor worker wait did not settle after timeout")
			}
			waiting = false
		case <-signalChannel:
			// The exact PGID signal reaches the worker too. If it elects to
			// ignore a cooperative request, keep supervising until a forced
			// stop or the independent timeout; never smuggle escalation here.
		}
	}
	if errors.Is(waitErr, exec.ErrWaitDelay) && command.ProcessState != nil && command.ProcessState.Success() {
		waitErr = nil
	}
	if err := drainProcessGroup(record.PGID, os.Getpid(), pipeDrain); err != nil {
		return err
	}
	diagnosticPayload, diagnosticTruncated := diagnostic.snapshot()
	proof, err := buildCompletionProof(
		envelope, record, command, workerPGID, workerBootID, workerBirth,
		cause, waitErr, diagnosticPayload, diagnosticTruncated,
	)
	if err != nil {
		return err
	}
	if err := signCompletionProof(&proof, envelope.CompletionPrivateKey); err != nil {
		return err
	}
	if err := publishCompletionProof(envelope.RunDirectory, proof); err != nil {
		return err
	}
	_, _ = diagnosticOutput.Write(diagnosticPayload)
	return nil
}

func publishStartedFailure(
	envelope supervisorEnvelope,
	record processRecord,
	diagnostic *cappedDiagnostic,
	diagnosticOutput *os.File,
	startErr error,
) error {
	_, _ = diagnostic.Write([]byte(startErr.Error()))
	payload, truncated := diagnostic.snapshot()
	proof := completionProof{
		SchemaVersion: completionProofSchema, SupervisorInstance: envelope.SupervisorInstance,
		ExecutionRef: envelope.ExecutionRef, RequestHash: envelope.RequestHash,
		SpecHash: envelope.SpecHash, ArtifactMediaType: envelope.ArtifactMediaType,
		RuntimeScope:  envelope.RuntimeScope,
		SupervisorPID: record.PID, SupervisorPGID: record.PGID,
		SupervisorBootID: record.BootID, SupervisorBirthMarker: record.BirthMarker,
		Cause: supervisorCauseNatural, Exited: true, ExitCode: supervisorInternalFailureExit,
		TreeGone: true, DiagnosticSize: int64(len(payload)),
		DiagnosticHash: digestBytes(payload), DiagnosticTruncated: truncated,
	}
	if err := signCompletionProof(&proof, envelope.CompletionPrivateKey); err != nil {
		return err
	}
	if err := publishCompletionProof(envelope.RunDirectory, proof); err != nil {
		return err
	}
	_, _ = diagnosticOutput.Write(payload)
	return nil
}

func buildCompletionProof(
	envelope supervisorEnvelope,
	record processRecord,
	command *exec.Cmd,
	workerPGID int,
	workerBootID, workerBirth, cause string,
	waitErr error,
	diagnostic []byte,
	diagnosticTruncated bool,
) (completionProof, error) {
	if command == nil || command.Process == nil || command.ProcessState == nil {
		return completionProof{}, errors.New("supervisor worker state unavailable")
	}
	status, ok := command.ProcessState.Sys().(syscall.WaitStatus)
	if !ok {
		return completionProof{}, errors.New("supervisor worker status unavailable")
	}
	resultSize, resultHash, resultFound, resultTooLarge, resultErr := digestFile(envelope.ResultPath, envelope.MaxOutputBytes)
	if resultErr != nil {
		return completionProof{}, resultErr
	}
	proof := completionProof{
		SchemaVersion: completionProofSchema, SupervisorInstance: envelope.SupervisorInstance,
		ExecutionRef: envelope.ExecutionRef, RequestHash: envelope.RequestHash,
		SpecHash: envelope.SpecHash, ArtifactMediaType: envelope.ArtifactMediaType,
		RuntimeScope:  envelope.RuntimeScope,
		SupervisorPID: record.PID, SupervisorPGID: record.PGID,
		SupervisorBootID: record.BootID, SupervisorBirthMarker: record.BirthMarker,
		WorkerPID: command.Process.Pid, WorkerPGID: workerPGID,
		WorkerBootID: workerBootID, WorkerBirthMarker: workerBirth,
		Cause: cause, TreeGone: true, ResultFound: resultFound,
		ResultSize: resultSize, ResultHash: resultHash, ResultTooLarge: resultTooLarge,
		DiagnosticSize: int64(len(diagnostic)),
		DiagnosticHash: digestBytes(diagnostic), DiagnosticTruncated: diagnosticTruncated,
	}
	if status.Exited() {
		proof.Exited, proof.ExitCode = true, status.ExitStatus()
	} else if status.Signaled() {
		proof.Signal = int(status.Signal())
	} else {
		return completionProof{}, errors.New("supervisor worker status incomplete")
	}
	if waitErr == nil && (!proof.Exited || proof.ExitCode != 0) {
		return completionProof{}, errors.New("supervisor wait status mismatch")
	}
	if waitErr != nil && proof.Exited && proof.ExitCode == 0 {
		return completionProof{}, errors.New("supervisor wait error mismatch")
	}
	return proof, nil
}

func validateSupervisorFilesystem(envelope supervisorEnvelope) error {
	runInfo, err := os.Lstat(envelope.RunDirectory)
	if err != nil || runInfo.Mode()&os.ModeSymlink != 0 || !runInfo.IsDir() ||
		runInfo.Mode().Perm()&0o077 != 0 {
		return errors.New("invalid supervisor run directory")
	}
	workInfo, err := os.Lstat(envelope.WorkingDirectory)
	if err != nil || workInfo.Mode()&os.ModeSymlink != 0 || !workInfo.IsDir() {
		return errors.New("invalid supervisor working directory")
	}
	if filepath.Clean(envelope.ResultPath) != envelope.ResultPath ||
		filepath.Clean(envelope.RunDirectory) != envelope.RunDirectory {
		return errors.New("invalid supervisor path")
	}
	if _, err := os.Lstat(completionProofPath(envelope.RunDirectory)); !errors.Is(err, os.ErrNotExist) {
		return errors.New("completion proof already exists")
	}
	return nil
}

func killProcessGroupMembersExcept(pgid, except int, signal syscall.Signal) error {
	members, err := linuxProcessGroupMembers(pgid)
	if err != nil {
		return err
	}
	var result error
	for _, pid := range members {
		if pid == except {
			continue
		}
		state, memberPGID, _, readErr := readLinuxProcess(pid)
		if errors.Is(readErr, os.ErrNotExist) || state == "Z" || state == "X" {
			continue
		}
		if readErr != nil || memberPGID != pgid {
			result = errors.Join(result, readErr)
			continue
		}
		if err := syscall.Kill(pid, signal); err != nil && !errors.Is(err, syscall.ESRCH) {
			result = errors.Join(result, err)
		}
	}
	return result
}

func linuxProcessGroupMembers(pgid int) ([]int, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	members := make([]int, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		state, memberPGID, _, err := readLinuxProcess(pid)
		if err == nil && memberPGID == pgid && state != "Z" && state != "X" {
			members = append(members, pid)
		}
	}
	return members, nil
}

func drainProcessGroup(pgid, except int, delay time.Duration) error {
	deadline := time.Now().Add(delay)
	for {
		members, err := linuxProcessGroupMembers(pgid)
		if err != nil {
			return err
		}
		remaining := 0
		for _, pid := range members {
			if pid != except {
				remaining++
			}
		}
		if remaining == 0 {
			return nil
		}
		if err := killProcessGroupMembersExcept(pgid, except, syscall.SIGKILL); err != nil {
			return err
		}
		if !time.Now().Before(deadline) {
			return errors.New("supervisor process group did not drain")
		}
		time.Sleep(time.Millisecond)
	}
}

func stringsReaderAndClear(value string) io.Reader {
	payload := []byte(value)
	return &clearingReader{reader: bytes.NewReader(payload), payload: payload}
}

type clearingReader struct {
	reader  *bytes.Reader
	payload []byte
}

func (reader *clearingReader) Read(payload []byte) (int, error) {
	count, err := reader.reader.Read(payload)
	if errors.Is(err, io.EOF) {
		clearBytes(reader.payload)
		reader.payload = nil
	}
	return count, err
}
