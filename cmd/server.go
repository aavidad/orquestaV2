/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"orquesta/db"
	"orquesta/internal/rpclocal"
)

var commandExecMu sync.Mutex

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Servidor local único para Orquesta",
}

var serverRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Arranca el servidor local único de Orquesta",
	RunE: func(cmd *cobra.Command, args []string) error {
		addr, _ := cmd.Flags().GetString("addr")
		listenAddr := strings.TrimPrefix(rpclocal.BaseURL(addr), "http://")
		if !db.IsOpen() {
			if err := db.Open(); err != nil {
				return err
			}
		}

		info, err := newLocalRPCState("server", listenAddr)
		if err != nil {
			return err
		}
		if err := rpclocal.SaveServerInfo(info); err != nil {
			return err
		}
		defer rpclocal.RemoveState("")

		server := &http.Server{
			Addr:    listenAddr,
			Handler: rpclocal.NewMux(&rpclocal.Server{State: info, Executor: executeRPCRequest}),
		}

		ctx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stopSignals()
		errCh := make(chan error, 1)
		go func() {
			errCh <- server.ListenAndServe()
		}()

		fmt.Printf("Servidor local de Orquesta en %s\n", rpclocal.BaseURL(listenAddr))
		select {
		case <-ctx.Done():
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = server.Shutdown(shutdownCtx)
			err := <-errCh
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				return err
			}
			return nil
		case err := <-errCh:
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				return err
			}
			return nil
		}
	},
}

var serverStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Consulta el estado del servidor local",
	RunE: func(cmd *cobra.Command, args []string) error {
		info, infoErr := rpclocal.LoadServerInfo()
		addr := rpclocal.ResolveServerAddr()
		if infoErr == nil {
			addr = rpclocal.BaseURL(info.Addr)
		}
		ctx, cancel := context.WithTimeout(context.Background(), rpclocal.DefaultTimeout())
		defer cancel()
		err := rpclocal.Ping(ctx, addr)
		if err != nil {
			if infoErr == nil {
				fmt.Printf("Servidor no disponible en %s (pid=%d)\n", info.Addr, info.PID)
				return err
			}
			return err
		}
		if infoErr == nil {
			fmt.Printf("Servidor activo en %s (pid=%d)\n", info.Addr, info.PID)
			return nil
		}
		fmt.Printf("Servidor activo en %s\n", addr)
		return nil
	},
}

var serverStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Detiene el servidor local",
	RunE: func(cmd *cobra.Command, args []string) error {
		info, err := rpclocal.LoadServerInfo()
		if err != nil {
			return fmt.Errorf("no hay statefile de servidor: %w", err)
		}
		if !rpclocal.MatchesCurrentScope(info) {
			return fmt.Errorf("el statefile no pertenece al scope actual")
		}

		addr := rpclocal.BaseURL(info.Addr)
		ctx, cancel := context.WithTimeout(context.Background(), rpclocal.DefaultTimeout())
		health, healthErr := rpclocal.NewClient(addr, nil).Ping(ctx)
		cancel()
		if healthErr != nil {
			if removeErr := rpclocal.RemoveState(""); removeErr != nil {
				return fmt.Errorf("state obsoleto; remove state: %w", removeErr)
			}
			fmt.Printf("State obsoleto limpiado; no se ha enviado señal a ningún proceso\n")
			return nil
		}
		if strings.TrimSpace(health.ScopeID) != "" && health.ScopeID != rpclocal.CurrentScopeID() {
			return fmt.Errorf("el servidor activo pertenece a otro scope (%s)", health.ScopeID)
		}
		if strings.TrimSpace(health.DBPath) != "" && health.DBPath != db.CurrentDBPath() {
			return fmt.Errorf("el servidor activo usa otra DB (%s)", health.DBPath)
		}

		proc, err := os.FindProcess(health.PID)
		if err != nil {
			return err
		}
		if err := proc.Signal(syscall.SIGTERM); err != nil {
			return fmt.Errorf("no se pudo enviar SIGTERM al servidor %d: %w", health.PID, err)
		}

		deadline := time.Now().Add(4 * time.Second)
		for time.Now().Before(deadline) {
			ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			pingErr := rpclocal.Ping(ctx, addr)
			cancel()
			if pingErr != nil {
				_ = rpclocal.RemoveState("")
				fmt.Printf("Servidor detenido (%d)\n", health.PID)
				return nil
			}
			time.Sleep(100 * time.Millisecond)
		}

		_ = rpclocal.RemoveState("")
		fmt.Printf("Señal enviada al servidor (%d); estado limpiado\n", health.PID)
		return nil
	},
}

var serverDoctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnostica el estado del modo servidor y la persistencia objetivo",
	RunE: func(cmd *cobra.Command, args []string) error {
		infoPath := rpclocal.DefaultInfoPath()
		dbPath := db.CurrentDBPath()
		addr := rpclocal.ResolveServerAddr()

		fmt.Printf("Modo objetivo: servidor local\n")
		fmt.Printf("Statefile: %s\n", infoPath)
		fmt.Printf("DB objetivo: %s\n", dbPath)
		fmt.Printf("Addr resuelta: %s\n", addr)

		info, err := rpclocal.LoadServerInfo()
		if err != nil {
			fmt.Printf("State: no disponible (%v)\n", err)
		} else {
			addr = rpclocal.BaseURL(info.Addr)
			fmt.Printf("State: pid=%d addr=%s scope=%s db=%s started_at=%s\n", info.PID, info.Addr, info.ScopeID, info.DBPath, info.StartedAt.Format(time.RFC3339))
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := rpclocal.Ping(ctx, addr); err != nil {
			fmt.Printf("Health RPC: KO (%v)\n", err)
			fmt.Println("Fallback local: solo con --local o ORQUESTA_ALLOW_LOCAL_FALLBACK=1")
			return nil
		}
		fmt.Printf("Health RPC: OK\n")
		fmt.Printf("Fallback local: solo por politica explicita (--local / ORQUESTA_ALLOW_LOCAL_FALLBACK=1)\n")
		return nil
	},
}

func init() {
	serverRunCmd.Flags().String("addr", strings.TrimPrefix(rpclocal.DefaultAddr(), "http://"), "Dirección HTTP local del servidor")
	serverCmd.AddCommand(serverRunCmd, serverStatusCmd, serverStopCmd, serverDoctorCmd)
	rootCmd.AddCommand(serverCmd)
}

func shouldDelegateToLocalServer(args []string) bool {
	if !localRPCEnabled(args) {
		return false
	}
	if len(args) == 0 {
		return false
	}
	for _, arg := range args {
		if arg == "--help" || arg == "-h" || arg == "--local" {
			return false
		}
	}
	switch args[0] {
	case "help", "completion", "serve", "server":
		return false
	default:
		return true
	}
}

func executeViaLocalServer(args []string, stdout, stderr io.Writer) (bool, int, error) {
	if !shouldDelegateToLocalServer(args) {
		return false, 0, nil
	}

	addr := rpclocal.ResolveServerAddr()
	ctx, cancel := context.WithTimeout(context.Background(), rpclocal.DefaultTimeout())
	err := rpclocal.Ping(ctx, addr)
	cancel()
	if err != nil {
		if startErr := ensureLocalServer(addr); startErr != nil {
			return false, 0, fmt.Errorf("arranque RPC en %s: %w", addr, startErr)
		}
	}

	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	state, stateErr := rpclocal.LoadServerInfo()
	if stateErr != nil {
		return false, 0, stateErr
	}
	addr = rpclocal.BaseURL(state.Addr)
	resp, err := rpclocal.NewClient(addr, nil).WithToken(state.Token).Exec(ctx, &rpclocal.ExecuteRequest{Args: args})
	if err != nil {
		return false, 0, err
	}
	if resp.Stdout != "" {
		_, _ = io.WriteString(stdout, resp.Stdout)
	}
	if resp.Stderr != "" {
		_, _ = io.WriteString(stderr, resp.Stderr)
	}
	return true, resp.ExitCode, nil
}

func ensureLocalServer(addr string) error {
	executable, err := os.Executable()
	if err != nil {
		executable = os.Args[0]
	}
	cmd := exec.Command(executable, "server", "run", "--addr", strings.TrimPrefix(rpclocal.BaseURL(addr), "http://"))
	logPath := strings.TrimSuffix(rpclocal.DefaultInfoPath(), ".json") + ".log"
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer logFile.Close()

	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Env = append(os.Environ(), "ORQUESTA_FORCE_LOCAL=1")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	_ = cmd.Process.Release()

	deadline := time.Now().Add(rpclocal.DefaultStartWait())
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), rpclocal.DefaultTimeout())
		pingErr := rpclocal.Ping(ctx, addr)
		cancel()
		if pingErr == nil {
			return nil
		}
		time.Sleep(150 * time.Millisecond)
	}
	return errors.New("timeout esperando al servidor local")
}

func newLocalRPCState(kind, addr string) (rpclocal.ServerInfo, error) {
	token, err := randomHexToken(32)
	if err != nil {
		return rpclocal.ServerInfo{}, err
	}
	return rpclocal.ServerInfo{
		Addr:      strings.TrimPrefix(rpclocal.BaseURL(addr), "http://"),
		PID:       os.Getpid(),
		Kind:      kind,
		ScopeID:   rpclocal.CurrentScopeID(),
		DBPath:    db.CurrentDBPath(),
		Token:     token,
		StartedAt: time.Now().UTC(),
		Version:   "dev",
	}, nil
}

func randomHexToken(size int) (string, error) {
	if size <= 0 {
		size = 32
	}
	raw := make([]byte, size)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generando token localrpc: %w", err)
	}
	return hex.EncodeToString(raw), nil
}

func executeRPCRequest(ctx context.Context, req *rpclocal.ExecRequest) (*rpclocal.ExecResponse, error) {
	if !shouldExecuteRemotely(req.Args) {
		return nil, fmt.Errorf("comando no permitido por RPC")
	}
	stdout, stderr, exitCode := executeLocalCommandCaptured(req.Args)
	return &rpclocal.ExecResponse{
		Stdout:   stdout,
		Stderr:   stderr,
		ExitCode: exitCode,
	}, nil
}

func shouldExecuteRemotely(args []string) bool {
	if len(args) == 0 {
		return false
	}
	for _, arg := range args {
		if arg == "--help" || arg == "-h" || arg == "--local" {
			return false
		}
	}
	if !commandSupportsServerMode(normalizedCommandArgs(args)) {
		return false
	}
	return true
}

func executeLocalCommandCaptured(args []string) (string, string, int) {
	commandExecMu.Lock()
	defer commandExecMu.Unlock()

	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		return "", err.Error(), 1
	}
	defer stdoutR.Close()
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		_ = stdoutW.Close()
		return "", err.Error(), 1
	}
	defer stderrR.Close()

	oldStdout := os.Stdout
	oldStderr := os.Stderr
	os.Stdout = stdoutW
	os.Stderr = stderrW

	stdoutDone := make(chan string, 1)
	stderrDone := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, stdoutR)
		stdoutDone <- buf.String()
	}()
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, stderrR)
		stderrDone <- buf.String()
	}()

	exitCode := 0
	err = executeLocalArgs(args, stdoutW, stderrW)
	if err != nil {
		exitCode = 1
		_, _ = io.WriteString(stderrW, err.Error()+"\n")
	}

	_ = stdoutW.Close()
	_ = stderrW.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	return <-stdoutDone, <-stderrDone, exitCode
}
