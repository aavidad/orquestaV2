//go:build linux

package firecrackerattestor

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

const (
	WorkloadBootstrapArgument = "--internal-firecracker-e2e-bootstrap"
	WorkloadChildArgument     = "--internal-firecracker-e2e-child"
	childEnvelopeSchema       = "orquesta.firecracker-attestor.workload-child.v1"
	maxChildResultBytes       = 1 << 20
)

type ChildWorkloadHandler interface {
	Run(context.Context, WorkloadRequest, json.RawMessage) ([]AttestationOutcome, error)
}

type AutoExecWorkload struct {
	SelfPath          string
	SelfSHA256        string
	UID               uint32
	GID               uint32
	Payload           json.RawMessage
	CancellationGrace time.Duration
}

type childEnvelope struct {
	Schema      string          `json:"schema"`
	ExpectedUID uint32          `json:"expected_uid"`
	ExpectedGID uint32          `json:"expected_gid"`
	Request     WorkloadRequest `json:"request"`
	Payload     json.RawMessage `json:"payload"`
}

type childResult struct {
	Schema   string               `json:"schema"`
	Outcomes []AttestationOutcome `json:"outcomes,omitempty"`
	Error    string               `json:"error,omitempty"`
}

func (adapter AutoExecWorkload) StartPhase(
	ctx context.Context,
	request WorkloadRequest,
) (WorkloadRun, error) {
	if ctx == nil || os.Geteuid() != 0 || adapter.UID == 0 || adapter.GID == 0 ||
		!validDigest(adapter.SelfSHA256) || request.Count == 0 ||
		adapter.CancellationGrace <= 0 {
		return nil, ErrInvalid
	}
	executable, err := openVerifiedRootFile(trustedFileCheck{
		path: adapter.SelfPath, digest: adapter.SelfSHA256,
		executable: true, maxBytes: 128 << 20,
	})
	if err != nil {
		return nil, err
	}
	requestFile, err := sealedChildRequest(childEnvelope{
		Schema: childEnvelopeSchema, ExpectedUID: adapter.UID,
		ExpectedGID: adapter.GID, Request: request,
		Payload: append(json.RawMessage(nil), adapter.Payload...),
	})
	if err != nil {
		executable.Close()
		return nil, err
	}
	resultReader, resultWriter, err := os.Pipe()
	if err != nil {
		executable.Close()
		requestFile.Close()
		return nil, errors.New("firecracker_attestor_e2e.child_pipe_failed")
	}
	command := exec.CommandContext(
		ctx, "/proc/self/fd/5", WorkloadBootstrapArgument,
	)
	command.Env = []string{"LANG=C", "LC_ALL=C", "PATH=/usr/bin:/bin"}
	command.ExtraFiles = []*os.File{requestFile, resultWriter, executable}
	command.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, Pdeathsig: syscall.SIGKILL,
	}
	command.Cancel = func() error { return terminateCommandProcessGroup(command) }
	command.WaitDelay = adapter.CancellationGrace
	null, err := os.OpenFile("/dev/null", os.O_RDWR|syscall.O_CLOEXEC, 0)
	if err != nil {
		executable.Close()
		requestFile.Close()
		resultReader.Close()
		resultWriter.Close()
		return nil, errors.New("firecracker_attestor_e2e.child_null_failed")
	}
	command.Stdin, command.Stdout, command.Stderr = null, null, null
	if err := command.Start(); err != nil {
		null.Close()
		executable.Close()
		requestFile.Close()
		resultReader.Close()
		resultWriter.Close()
		return nil, errors.New("firecracker_attestor_e2e.child_start_failed")
	}
	null.Close()
	executable.Close()
	requestFile.Close()
	resultWriter.Close()
	run := &autoExecRun{command: command, done: make(chan struct{})}
	go run.wait(resultReader)
	return run, nil
}

func terminateCommandProcessGroup(command *exec.Cmd) error {
	if command == nil || command.Process == nil || command.Process.Pid <= 1 {
		return os.ErrProcessDone
	}
	err := unix.Kill(-command.Process.Pid, syscall.SIGTERM)
	if errors.Is(err, unix.ESRCH) {
		return os.ErrProcessDone
	}
	return err
}

type autoExecRun struct {
	mu       sync.Mutex
	command  *exec.Cmd
	done     chan struct{}
	outcomes []AttestationOutcome
	err      error
	closed   bool
}

func (run *autoExecRun) Done() <-chan struct{} { return run.done }

func (run *autoExecRun) Result() ([]AttestationOutcome, error) {
	<-run.done
	run.mu.Lock()
	defer run.mu.Unlock()
	return append([]AttestationOutcome(nil), run.outcomes...), run.err
}

func (run *autoExecRun) Close() error {
	if run == nil {
		return nil
	}
	<-run.done
	run.mu.Lock()
	defer run.mu.Unlock()
	if run.closed {
		return nil
	}
	run.closed = true
	return nil
}

func (run *autoExecRun) wait(reader *os.File) {
	defer close(run.done)
	content, readErr := io.ReadAll(io.LimitReader(reader, maxChildResultBytes+1))
	closeErr := reader.Close()
	waitErr := run.command.Wait()
	var result childResult
	decodeErr := json.Unmarshal(content, &result)
	if len(content) > maxChildResultBytes ||
		result.Schema != childEnvelopeSchema ||
		(result.Error == "") == (len(result.Outcomes) == 0) {
		decodeErr = errors.New("firecracker_attestor_e2e.child_result_invalid")
	}
	if result.Error != "" {
		decodeErr = errors.New(result.Error)
	}
	run.mu.Lock()
	run.outcomes = append([]AttestationOutcome(nil), result.Outcomes...)
	run.err = errors.Join(readErr, closeErr, waitErr, decodeErr)
	run.mu.Unlock()
}

func sealedChildRequest(envelope childEnvelope) (*os.File, error) {
	content, err := json.Marshal(envelope)
	if err != nil || len(content) > 64<<10 {
		return nil, ErrInvalid
	}
	fd, err := unix.MemfdCreate(
		"orquesta-firecracker-e2e-request",
		unix.MFD_CLOEXEC|unix.MFD_ALLOW_SEALING,
	)
	if err != nil {
		return nil, errors.New("firecracker_attestor_e2e.child_request_failed")
	}
	file := os.NewFile(uintptr(fd), "orquesta-firecracker-e2e-request")
	if _, err := file.Write(content); err != nil || file.Sync() != nil {
		file.Close()
		return nil, errors.New("firecracker_attestor_e2e.child_request_failed")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		file.Close()
		return nil, errors.New("firecracker_attestor_e2e.child_request_failed")
	}
	seals := unix.F_SEAL_SEAL | unix.F_SEAL_SHRINK |
		unix.F_SEAL_GROW | unix.F_SEAL_WRITE
	if _, err := unix.FcntlInt(file.Fd(), unix.F_ADD_SEALS, seals); err != nil {
		file.Close()
		return nil, errors.New("firecracker_attestor_e2e.child_request_failed")
	}
	return file, nil
}

// RunAutoExecMode handles only the two private, self-exec modes. The public
// command remains responsible for normal argument parsing.
func RunAutoExecMode(argument string, handler ChildWorkloadHandler) (bool, int) {
	switch argument {
	case WorkloadBootstrapArgument:
		return true, runBootstrap()
	case WorkloadChildArgument:
		return true, runWorkloadChild(handler)
	default:
		return false, 0
	}
}

func runBootstrap() int {
	if os.Geteuid() != 0 {
		return 120
	}
	envelope, err := readChildEnvelope()
	if err != nil || envelope.ExpectedUID == 0 || envelope.ExpectedGID == 0 {
		return 121
	}
	if unix.Setgroups([]int{}) != nil {
		return 122
	}
	for capability := 0; capability < 64; capability++ {
		err := unix.Prctl(unix.PR_CAPBSET_DROP, uintptr(capability), 0, 0, 0)
		if err != nil && !errors.Is(err, unix.EINVAL) {
			return 123
		}
	}
	if unix.Prctl(unix.PR_CAP_AMBIENT, unix.PR_CAP_AMBIENT_CLEAR_ALL, 0, 0, 0) != nil ||
		unix.Prctl(unix.PR_SET_KEEPCAPS, 0, 0, 0, 0) != nil ||
		unix.Setresgid(int(envelope.ExpectedGID), int(envelope.ExpectedGID), int(envelope.ExpectedGID)) != nil ||
		unix.Setresuid(int(envelope.ExpectedUID), int(envelope.ExpectedUID), int(envelope.ExpectedUID)) != nil ||
		unix.Prctl(unix.PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0) != nil {
		return 124
	}
	flags, err := unix.FcntlInt(uintptr(5), unix.F_GETFD, 0)
	if err != nil {
		return 125
	}
	if _, err := unix.FcntlInt(uintptr(5), unix.F_SETFD, flags&^unix.FD_CLOEXEC); err != nil {
		return 125
	}
	if _, err := unix.Seek(3, 0, io.SeekStart); err != nil {
		return 126
	}
	err = unix.Exec(
		"/proc/self/fd/5",
		[]string{"orquesta-firecracker-attestor-e2e", WorkloadChildArgument},
		[]string{"LANG=C", "LC_ALL=C", "PATH=/usr/bin:/bin"},
	)
	if err != nil {
		return 127
	}
	return 0
}

func runWorkloadChild(handler ChildWorkloadHandler) int {
	envelope, err := readChildEnvelope()
	if err != nil || verifyChildIdentity(envelope.ExpectedUID, envelope.ExpectedGID) != nil {
		writeChildResult(childResult{Schema: childEnvelopeSchema, Error: "firecracker_attestor_e2e.child_identity_unsafe"})
		return 128
	}
	if handler == nil {
		writeChildResult(childResult{Schema: childEnvelopeSchema, Error: "firecracker_attestor_e2e.workload_unwired"})
		return 129
	}
	ctx, cancel := signal.NotifyContext(
		context.Background(), syscall.SIGTERM, syscall.SIGINT,
	)
	defer cancel()
	outcomes, err := handler.Run(
		ctx, envelope.Request,
		append(json.RawMessage(nil), envelope.Payload...),
	)
	if err != nil {
		writeChildResult(childResult{Schema: childEnvelopeSchema, Error: "firecracker_attestor_e2e.workload_failed"})
		return 130
	}
	if err := writeChildResult(childResult{
		Schema: childEnvelopeSchema, Outcomes: outcomes,
	}); err != nil {
		return 131
	}
	return 0
}

func readChildEnvelope() (childEnvelope, error) {
	file := os.NewFile(3, "orquesta-firecracker-e2e-request")
	if file == nil {
		return childEnvelope{}, ErrInvalid
	}
	content, err := io.ReadAll(io.LimitReader(file, (64<<10)+1))
	if err != nil || len(content) == 0 || len(content) > 64<<10 {
		return childEnvelope{}, ErrInvalid
	}
	var envelope childEnvelope
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&envelope) != nil || envelope.Schema != childEnvelopeSchema ||
		decoder.Decode(&struct{}{}) != io.EOF {
		return childEnvelope{}, ErrInvalid
	}
	if len(envelope.Payload) == 0 || len(envelope.Payload) > 64<<10 ||
		!json.Valid(envelope.Payload) {
		return childEnvelope{}, ErrInvalid
	}
	return envelope, nil
}

func verifyChildIdentity(uid, gid uint32) error {
	if uint32(os.Geteuid()) != uid || uint32(os.Getegid()) != gid ||
		uint32(os.Getuid()) != uid || uint32(os.Getgid()) != gid {
		return ErrInvalid
	}
	groups, err := os.Getgroups()
	if err != nil || len(groups) != 0 {
		return ErrInvalid
	}
	noNewPrivileges, err := unix.PrctlRetInt(unix.PR_GET_NO_NEW_PRIVS, 0, 0, 0, 0)
	if err != nil || noNewPrivileges != 1 {
		return ErrInvalid
	}
	status, err := os.ReadFile("/proc/self/status")
	if err != nil || !emptyCapabilityStatus(status) {
		return ErrInvalid
	}
	return nil
}

func emptyCapabilityStatus(status []byte) bool {
	required := map[string]bool{
		"CapInh:": false, "CapPrm:": false, "CapEff:": false,
		"CapBnd:": false, "CapAmb:": false,
	}
	for _, line := range bytes.Split(status, []byte{'\n'}) {
		fields := bytes.Fields(line)
		if len(fields) != 2 {
			continue
		}
		name := string(fields[0])
		if _, ok := required[name]; !ok {
			continue
		}
		raw := strings.TrimPrefix(string(fields[1]), "0x")
		value, err := hex.DecodeString(padHex(raw))
		if err != nil || !allZero(value) {
			return false
		}
		required[name] = true
	}
	for _, present := range required {
		if !present {
			return false
		}
	}
	return true
}

func padHex(value string) string {
	if len(value)%2 != 0 {
		return "0" + value
	}
	return value
}

func allZero(value []byte) bool {
	var aggregate byte
	for _, item := range value {
		aggregate |= item
	}
	return aggregate == 0
}

func writeChildResult(result childResult) error {
	file := os.NewFile(4, "orquesta-firecracker-e2e-result")
	if file == nil {
		return ErrInvalid
	}
	content, err := json.Marshal(result)
	if err != nil || len(content) > maxChildResultBytes {
		return ErrInvalid
	}
	content = append(content, '\n')
	for len(content) > 0 {
		written, err := file.Write(content)
		if err != nil || written <= 0 {
			return ErrInvalid
		}
		content = content[written:]
	}
	return nil
}
