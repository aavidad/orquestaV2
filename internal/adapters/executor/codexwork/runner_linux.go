//go:build linux

package codexwork

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
	protocol "orquesta/internal/adapters/protocol/codexwork"
)

const (
	SealedCommand   = "/perfil/bin/agente"
	SealedArgument  = "app-server"
	sealedHome      = "/trabajo"
	sealedCodexHome = "/credencial-codex"
	sealedPath      = "/perfil/bin:/bin"
	sealedLocale    = "C.UTF-8"

	defaultDiagnosticBytes = 64 << 10
	postKillWait           = 2 * time.Second
)

type Code string

const (
	CodeConfigInvalid Code = "codexwork_executor.config_invalid"
	CodeInputRead     Code = "codexwork_executor.input_read_failed"
	CodeCommandStart  Code = "codexwork_executor.command_start_failed"
	CodeCommandIO     Code = "codexwork_executor.command_io_failed"
	CodeTimeout       Code = "codexwork_executor.timeout"
	CodeCancelled     Code = "codexwork_executor.cancelled"
	CodeChildFailed   Code = "codexwork_executor.child_failed"
	CodeOutputMissing Code = "codexwork_executor.output_missing"
	CodeCleanupFailed Code = "codexwork_executor.cleanup_failed"
)

// Error no conserva detalles del proceso, del paquete o del protocolo remoto.
// Así puede atravesar el bootstrap sin filtrar prompt, credenciales o stderr.
type Error struct{ Code Code }

func (err *Error) Error() string {
	if err == nil {
		return ""
	}
	return string(err.Code)
}

func (err *Error) Is(target error) bool {
	var other *Error
	return errors.As(target, &other) && err != nil && err.Code == other.Code
}

func ErrorCode(err error) Code {
	var runnerErr *Error
	if errors.As(err, &runnerErr) {
		return runnerErr.Code
	}
	return ""
}

func runnerError(code Code) error { return &Error{Code: code} }

type CommandFactory func(context.Context, string, ...string) *exec.Cmd
type TimeoutFactory func(context.Context, time.Duration) (context.Context, context.CancelFunc)

type Config struct {
	Command            string
	Arguments          []string
	MaxPacketBytes     int
	MaxFrameBytes      int
	MaxDiagnosticBytes int64
	CleanupTimeout     time.Duration
	CommandFactory     CommandFactory
	WithTimeout        TimeoutFactory
}

type Runner struct{ config Config }

// NewSealed fija el unico proceso permitido por la imagen productiva. No lee
// entorno, fichero de configuracion, red ni credenciales.
func NewSealed() Runner {
	runner, err := New(Config{
		Command:            SealedCommand,
		Arguments:          []string{SealedArgument},
		MaxPacketBytes:     protocol.MaxPacketBytesV1,
		MaxFrameBytes:      protocol.MaxPacketBytesV1,
		MaxDiagnosticBytes: defaultDiagnosticBytes,
		CleanupTimeout:     postKillWait,
	})
	if err != nil {
		panic(err)
	}
	return runner
}

// New admite sustituciones solo para pruebas del ciclo de vida. Los limites no
// pueden ampliar el contrato canonico aunque el llamador los inyecte.
func New(config Config) (Runner, error) {
	if config.Command == "" || config.MaxPacketBytes <= 0 ||
		config.MaxPacketBytes > protocol.MaxPacketBytesV1 || config.MaxFrameBytes <= 0 ||
		config.MaxFrameBytes > protocol.MaxPacketBytesV1 || config.MaxDiagnosticBytes <= 0 ||
		config.CleanupTimeout <= 0 || config.CleanupTimeout > postKillWait {
		return Runner{}, runnerError(CodeConfigInvalid)
	}
	config.Arguments = append([]string(nil), config.Arguments...)
	if config.CommandFactory == nil {
		config.CommandFactory = exec.CommandContext
	}
	if config.WithTimeout == nil {
		config.WithTimeout = context.WithTimeout
	}
	return Runner{config: config}, nil
}

// Run consume exactamente un paquete, ejecuta una unica sesion app-server y
// publica solo el artefacto crudo. Todo proceso iniciado queda esperado antes
// de retornar. Para poder interrumpir read(2), el input productivo Linux debe
// ser un *os.File (os.Stdin lo es); otro Reader se admite unicamente cuando su
// contrato garantiza lectura finita o Close interruptible.
func (runner Runner) Run(ctx context.Context, input io.Reader, output io.Writer) error {
	if ctx == nil || input == nil || output == nil || runner.config.Command == "" ||
		runner.config.CommandFactory == nil || runner.config.WithTimeout == nil {
		return runnerError(CodeConfigInvalid)
	}
	packetBytes, err := readInitialPacket(ctx, input, runner.config.MaxPacketBytes)
	if err != nil {
		return err
	}
	packet, err := protocol.DecodeWorkPacketV1(packetBytes)
	if err != nil {
		return err
	}
	machine, err := protocol.NewMachine(packet)
	if err != nil {
		return err
	}

	runCtx, cancel := runner.config.WithTimeout(ctx, time.Duration(packet.TimeBudgetMS)*time.Millisecond)
	if runCtx == nil || cancel == nil {
		return runnerError(CodeConfigInvalid)
	}
	defer cancel()
	command := runner.config.CommandFactory(runCtx, runner.config.Command, runner.config.Arguments...)
	if command == nil {
		return runnerError(CodeCommandStart)
	}
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pdeathsig: syscall.SIGKILL}
	command.WaitDelay = postKillWait
	// El app-server no hereda proxy, token, HOME ni ninguna otra variable del
	// ejecutor. Solo se proyecta el endpoint local constante cuando el paquete
	// causal acredita que el plan firmado contiene la concesion de egreso.
	command.Env = sealedEnvironment(packet.ControlledEgressProxy)

	stdin, err := command.StdinPipe()
	if err != nil {
		return runnerError(CodeCommandIO)
	}
	stdout, err := command.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return runnerError(CodeCommandIO)
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		return runnerError(CodeCommandIO)
	}
	if err := command.Start(); err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		_ = stderr.Close()
		return runnerError(CodeCommandStart)
	}

	// CommandContext mata solo el PID por defecto. Esta cancelacion mata el
	// grupo completo y evita dejar descendientes del app-server en la microVM.
	stopCancellation := context.AfterFunc(runCtx, func() { _ = killProcessGroup(command) })
	stderrDone := make(chan struct{})
	go func() {
		bufferBytes := min(runner.config.MaxDiagnosticBytes, int64(32<<10))
		_, _ = io.CopyBuffer(io.Discard, stderr, make([]byte, int(bufferBytes)))
		close(stderrDone)
	}()

	result, bufferedStdout, exchangeErr := runner.exchange(runCtx, machine, stdin, stdout)
	_ = stdin.Close()
	var stdoutDone <-chan struct{}
	if exchangeErr == nil && result != nil {
		stdoutDone = drainOutput(bufferedStdout)
	} else {
		_ = stdout.Close()
	}
	if exchangeErr != nil {
		_ = killProcessGroup(command)
	}
	waitErr, waitExpired := waitAndReap(runCtx, command, runner.config.CleanupTimeout)
	// Wait acredita el hijo principal. Se purga el grupo tambien despues de
	// recolectarlo: asi mueren descendientes que hayan heredado un pipe aunque
	// el padre ya haya salido limpiamente.
	cleanupErr := killProcessGroup(command)
	_ = stdout.Close()
	_ = stderr.Close()
	if stdoutDone != nil {
		<-stdoutDone
	}
	<-stderrDone
	stopCancellation()

	if runCtx.Err() != nil {
		if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
			return runnerError(CodeTimeout)
		}
		return runnerError(CodeCancelled)
	}
	if exchangeErr != nil {
		return exchangeErr
	}
	if waitExpired || cleanupErr != nil {
		return runnerError(CodeCleanupFailed)
	}
	if waitErr != nil {
		return runnerError(CodeChildFailed)
	}
	if result == nil {
		return runnerError(CodeOutputMissing)
	}
	if _, err := io.WriteString(output, result.Artifact); err != nil {
		return runnerError(CodeCommandIO)
	}
	return nil
}

func (runner Runner) exchange(
	ctx context.Context,
	machine *protocol.Machine,
	stdin io.WriteCloser,
	stdout io.Reader,
) (*protocol.WorkResultV1, *bufio.Reader, error) {
	initial, err := machine.Start()
	if err != nil {
		return nil, nil, err
	}
	if err := writeAll(stdin, initial); err != nil {
		return nil, nil, runnerError(CodeCommandIO)
	}

	reader := bufio.NewReaderSize(stdout, min(runner.config.MaxFrameBytes+1, 64<<10))
	for {
		frame, readErr := readFrame(reader, runner.config.MaxFrameBytes)
		if len(frame) != 0 {
			transition, acceptErr := machine.AcceptJSONL(frame)
			if acceptErr != nil {
				return nil, reader, acceptErr
			}
			for _, outbound := range transition.Outbound {
				if err := writeAll(stdin, outbound); err != nil {
					return nil, reader, runnerError(CodeCommandIO)
				}
			}
			if transition.Result != nil {
				copyResult := *transition.Result
				if err := stdin.Close(); err != nil {
					return nil, reader, runnerError(CodeCleanupFailed)
				}
				// No se espera EOF: un descendiente puede heredar stdout. Run drena
				// en paralelo, espera al padre con plazo y purga el grupo despues.
				return &copyResult, reader, nil
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return nil, reader, nil
			}
			if ctx.Err() != nil {
				return nil, reader, contextFailure(ctx)
			}
			return nil, reader, readErr
		}
	}
}

func waitAndReap(ctx context.Context, command *exec.Cmd, limit time.Duration) (error, bool) {
	waited := make(chan error, 1)
	go func() { waited <- command.Wait() }()
	cleanupCtx, cancel := context.WithTimeout(ctx, limit)
	defer cancel()
	select {
	case err := <-waited:
		return err, false
	case <-cleanupCtx.Done():
		_ = killProcessGroup(command)
		return <-waited, true
	}
}

func drainOutput(input io.Reader) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		_, _ = io.CopyBuffer(io.Discard, input, make([]byte, 32<<10))
		close(done)
	}()
	return done
}

func readInitialPacket(ctx context.Context, input io.Reader, maximum int) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, contextFailure(ctx)
	}
	if file, ok := input.(*os.File); ok {
		return readInitialPacketFile(ctx, file, maximum)
	}
	var stopClose func() bool
	var closeDone chan struct{}
	if closer, ok := input.(io.ReadCloser); ok {
		closeDone = make(chan struct{})
		stopClose = context.AfterFunc(ctx, func() {
			_ = closer.Close()
			close(closeDone)
		})
	}
	if stopClose != nil {
		defer func() {
			if !stopClose() {
				<-closeDone
			}
		}()
	}
	content, err := io.ReadAll(io.LimitReader(input, int64(maximum)+1))
	if ctx.Err() != nil {
		return nil, contextFailure(ctx)
	}
	if err != nil {
		return nil, runnerError(CodeInputRead)
	}
	if len(content) > maximum {
		return nil, &protocol.Error{Code: protocol.CodePacketTooLarge}
	}
	return content, nil
}

// readInitialPacketFile usa poll sobre el descriptor real y un wake pipe de
// cancelacion. No crea una goroutine lectora y no depende de que Close consiga
// interrumpir read(2), comportamiento que FIFO y pipes no garantizan.
func readInitialPacketFile(ctx context.Context, input *os.File, maximum int) ([]byte, error) {
	fd := int(input.Fd())
	if fd < 0 {
		return nil, runnerError(CodeInputRead)
	}
	flags, err := unix.FcntlInt(uintptr(fd), unix.F_GETFL, 0)
	if err != nil {
		return nil, runnerError(CodeInputRead)
	}
	if _, err := unix.FcntlInt(uintptr(fd), unix.F_SETFL, flags|unix.O_NONBLOCK); err != nil {
		return nil, runnerError(CodeInputRead)
	}
	defer func() { _, _ = unix.FcntlInt(uintptr(fd), unix.F_SETFL, flags) }()

	wake := []int{-1, -1}
	if err := unix.Pipe2(wake, unix.O_CLOEXEC|unix.O_NONBLOCK); err != nil {
		return nil, runnerError(CodeInputRead)
	}
	defer func() {
		_ = unix.Close(wake[0])
		_ = unix.Close(wake[1])
	}()
	wakeDone := make(chan struct{})
	stopWake := context.AfterFunc(ctx, func() {
		_, _ = unix.Write(wake[1], []byte{1})
		close(wakeDone)
	})
	defer func() {
		if !stopWake() {
			<-wakeDone
		}
	}()

	pollFDs := []unix.PollFd{
		{Fd: int32(fd), Events: unix.POLLIN | unix.POLLHUP | unix.POLLERR},
		{Fd: int32(wake[0]), Events: unix.POLLIN | unix.POLLHUP | unix.POLLERR},
	}
	content := make([]byte, 0, min(maximum, 64<<10))
	buffer := make([]byte, min(maximum+1, 64<<10))
	for {
		if ctx.Err() != nil {
			return nil, contextFailure(ctx)
		}
		pollFDs[0].Revents, pollFDs[1].Revents = 0, 0
		if _, err := unix.Poll(pollFDs, -1); err != nil {
			if errors.Is(err, unix.EINTR) {
				continue
			}
			return nil, runnerError(CodeInputRead)
		}
		if ctx.Err() != nil || pollFDs[1].Revents != 0 {
			return nil, contextFailure(ctx)
		}
		if pollFDs[0].Revents == 0 {
			continue
		}
		for {
			if ctx.Err() != nil {
				return nil, contextFailure(ctx)
			}
			remaining := maximum + 1 - len(content)
			if remaining <= 0 {
				return nil, &protocol.Error{Code: protocol.CodePacketTooLarge}
			}
			readBuffer := buffer[:min(len(buffer), remaining)]
			read, readErr := unix.Read(fd, readBuffer)
			if read > 0 {
				content = append(content, readBuffer[:read]...)
				if len(content) > maximum {
					return nil, &protocol.Error{Code: protocol.CodePacketTooLarge}
				}
			}
			switch {
			case readErr == nil && read == 0:
				return content, nil
			case readErr == nil:
				continue
			case errors.Is(readErr, unix.EINTR):
				continue
			case errors.Is(readErr, unix.EAGAIN), errors.Is(readErr, unix.EWOULDBLOCK):
				break
			default:
				return nil, runnerError(CodeInputRead)
			}
			break
		}
	}
}

func sealedEnvironment(controlledEgressProxy string) []string {
	environment := []string{
		"HOME=" + sealedHome,
		"CODEX_HOME=" + sealedCodexHome,
		"PATH=" + sealedPath,
		"LANG=" + sealedLocale,
		"LC_ALL=" + sealedLocale,
	}
	if controlledEgressProxy == protocol.ControlledEgressProxyURLV1 {
		environment = append(environment,
			"HTTP_PROXY="+protocol.ControlledEgressProxyURLV1,
			"HTTPS_PROXY="+protocol.ControlledEgressProxyURLV1,
		)
	}
	return environment
}

func contextFailure(ctx context.Context) error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return runnerError(CodeTimeout)
	}
	return runnerError(CodeCancelled)
}

func readFrame(reader *bufio.Reader, maximum int) ([]byte, error) {
	frame := make([]byte, 0, min(maximum, 64<<10))
	for {
		part, err := reader.ReadSlice('\n')
		if len(frame)+len(part) > maximum {
			return nil, &protocol.Error{Code: protocol.CodeFrameTooLarge}
		}
		frame = append(frame, part...)
		if err == nil {
			return frame, nil
		}
		if errors.Is(err, bufio.ErrBufferFull) {
			continue
		}
		if errors.Is(err, io.EOF) && len(frame) != 0 {
			return frame, io.EOF
		}
		return frame, err
	}
}

func writeAll(writer io.Writer, content []byte) error {
	for len(content) != 0 {
		written, err := writer.Write(content)
		if err != nil {
			return err
		}
		if written <= 0 {
			return io.ErrShortWrite
		}
		content = content[written:]
	}
	return nil
}

func killProcessGroup(command *exec.Cmd) error {
	if command == nil || command.Process == nil || command.Process.Pid <= 0 {
		return nil
	}
	err := syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
	if errors.Is(err, syscall.ESRCH) {
		return nil
	}
	return err
}
