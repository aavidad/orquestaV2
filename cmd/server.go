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
	"net"
	"os"
	"os/exec"
	"path/filepath"
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
		security, err := resolveServerSecurityOptions(cmd)
		if err != nil {
			return err
		}
		if strings.TrimSpace(security.TLSCert) != "" || strings.TrimSpace(security.TLSKey) != "" || strings.TrimSpace(security.TLSClientCA) != "" {
			return fmt.Errorf("server run no soporta TLS; usa 'orquesta serve' para exposición HTTPS/mTLS")
		}
		return arrancarServidorUnificado(listenAddr, "server", false, resolveServerDebugOptions(cmd), security)
	},
}

var serverStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Consulta el estado del servidor local",
	RunE: func(cmd *cobra.Command, args []string) error {
		info, recoveredFromHealth, err := loadServerInfoWithHealthFallback("")
		if err != nil {
			return err
		}
		fmt.Printf("Servidor activo en %s (pid=%d)\n", info.Addr, info.PID)
		if recoveredFromHealth {
			fmt.Printf("Statefile: ausente; usando healthz del daemon activo\n")
		}
		if driver := strings.TrimSpace(info.StorageDriver); driver != "" {
			fmt.Printf("Storage: %s %s\n", driver, resolveServerStorageTarget(info))
		}
		fmt.Printf("Log: %s\n", localServerLogPath())
		return nil
	},
}

var serverStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Arranca el servidor local como daemon y espera healthz",
	RunE: func(cmd *cobra.Command, args []string) error {
		addr, _ := cmd.Flags().GetString("addr")
		baseURL := rpclocal.BaseURL(addr)
		if info, _, err := loadServerInfoWithHealthFallback(baseURL); err == nil {
			fmt.Printf("Servidor ya activo en %s (pid=%d)\n", rpclocal.BaseURL(info.Addr), info.PID)
			return nil
		}
		if err := ensureLocalServer(baseURL); err != nil {
			return err
		}
		info, _, err := loadServerInfoWithHealthFallback(baseURL)
		if err != nil {
			return err
		}
		fmt.Printf("Servidor activo en %s (pid=%d)\n", rpclocal.BaseURL(info.Addr), info.PID)
		return nil
	},
}

var serverPrepararSesionCmd = &cobra.Command{
	Use:   "preparar-sesion",
	Short: "Deja el servidor listo para una nueva sesión con limpieza segura de residuos terminales",
	RunE: func(cmd *cobra.Command, args []string) error {
		addr, _ := cmd.Flags().GetString("addr")
		baseURL := rpclocal.BaseURL(addr)
		proyecto, _ := cmd.Flags().GetString("proyecto")
		olderThanMinutes, _ := cmd.Flags().GetInt("older-than-minutes")
		actor, _ := cmd.Flags().GetString("actor")
		if olderThanMinutes < 0 {
			return fmt.Errorf("--older-than-minutes no puede ser negativo")
		}
		if info, _, err := loadServerInfoWithHealthFallback(baseURL); err == nil {
			fmt.Printf("Servidor previo detectado en %s (pid=%d); reinicio limpio...\n", rpclocal.BaseURL(info.Addr), info.PID)
			if err := serverStopCmd.RunE(cmd, args); err != nil {
				return err
			}
		}
		if err := ensureLocalServer(baseURL); err != nil {
			return err
		}
		info, _, err := loadServerInfoWithHealthFallback(baseURL)
		if err != nil {
			return err
		}
		fmt.Printf("Servidor activo en %s (pid=%d)\n", rpclocal.BaseURL(info.Addr), info.PID)

		proyecto = strings.TrimSpace(proyecto)
		if proyecto == "" {
			proyecto = "orquestador"
		}
		actor = strings.TrimSpace(actor)
		if actor == "" {
			actor = "orquesta"
		}

		handlesResp, ok, err := purgarRuntimeHandlesDesdeAPI("", proyecto, []string{"cerrado", "fallido"}, actor)
		if !ok {
			return serverFirstCommandError("server preparar-sesion")
		}
		if err != nil {
			return err
		}
		ordersResp, ok, err := purgarRuntimeOrdersDesdeAPI("", proyecto, []string{"completada", "fallida", "expirada", "cancelada"}, nil, olderThanMinutes, actor)
		if !ok {
			return serverFirstCommandError("server preparar-sesion")
		}
		if err != nil {
			return err
		}

		fmt.Printf("Limpieza segura: %d handles inactivos, %d órdenes terminales\n", handlesResp.Deleted, ordersResp.Deleted)
		fmt.Printf("Proyecto: %s · corte órdenes: %d min\n", proyecto, olderThanMinutes)
		return nil
	},
}

var serverStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Detiene el servidor local",
	RunE: func(cmd *cobra.Command, args []string) error {
		info, _, err := loadServerInfoWithHealthFallback("")
		if err != nil {
			return err
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
		if err := validateHealthStorage(health); err != nil {
			return err
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
		storageDriver := db.CurrentStorageDriver()
		storageTarget := currentServerStorageTarget()
		addr := rpclocal.ResolveServerAddr()

		fmt.Printf("Modo objetivo: servidor local\n")
		fmt.Printf("Statefile: %s\n", infoPath)
		fmt.Printf("Storage driver: %s\n", storageDriver)
		fmt.Printf("Storage target: %s\n", storageTarget)
		fmt.Printf("Addr resuelta: %s\n", addr)

		info, recoveredFromHealth, err := loadServerInfoWithHealthFallback(addr)
		if err != nil {
			fmt.Printf("State: no disponible (%v)\n", err)
		} else {
			addr = rpclocal.BaseURL(info.Addr)
			fmt.Printf("State: pid=%d addr=%s scope=%s storage=%s %s started_at=%s\n",
				info.PID,
				info.Addr,
				info.ScopeID,
				strings.TrimSpace(info.StorageDriver),
				resolveServerStorageTarget(info),
				info.StartedAt.Format(time.RFC3339),
			)
			if recoveredFromHealth {
				fmt.Printf("State recovery: healthz (statefile ausente o stale)\n")
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := rpclocal.Ping(ctx, addr); err != nil {
			fmt.Printf("Health RPC: KO (%v)\n", err)
			fmt.Println("Modo local: solo con ORQUESTA_FORCE_LOCAL_DB=1 y solo para recuperación")
			return nil
		}
		fmt.Printf("Health RPC: OK\n")
		fmt.Printf("Modo local: solo por recuperación explícita (ORQUESTA_FORCE_LOCAL_DB=1)\n")
		return nil
	},
}

func init() {
	serverStartCmd.Flags().String("addr", strings.TrimPrefix(rpclocal.DefaultAddr(), "http://"), "Dirección HTTP local del servidor")
	serverPrepararSesionCmd.Flags().String("addr", strings.TrimPrefix(rpclocal.DefaultAddr(), "http://"), "Dirección HTTP local del servidor")
	serverPrepararSesionCmd.Flags().String("proyecto", "orquestador", "Proyecto sobre el que purgar residuos terminales")
	serverPrepararSesionCmd.Flags().Int("older-than-minutes", 60, "Purga solo órdenes terminales creadas hace más de N minutos")
	serverPrepararSesionCmd.Flags().String("actor", "orquesta", "Actor que solicita la preparación")
	serverRunCmd.Flags().String("addr", strings.TrimPrefix(rpclocal.DefaultAddr(), "http://"), "Dirección HTTP local del servidor")
	serverRunCmd.Flags().String("tls-cert", "", "Certificado PEM del servidor")
	serverRunCmd.Flags().String("tls-key", "", "Clave privada PEM del servidor")
	serverRunCmd.Flags().String("tls-client-ca", "", "CA PEM para exigir certificados cliente (mTLS)")
	serverRunCmd.Flags().Bool("debug", false, "Activa logging de depuración del servidor")
	serverRunCmd.Flags().Bool("debug-http", false, "Log HTTP detallado por request")
	serverRunCmd.Flags().Bool("debug-control-plane", false, "Log detallado del ciclo del control plane")
	serverCmd.AddCommand(serverStartCmd, serverPrepararSesionCmd, serverRunCmd, serverStatusCmd, serverStopCmd, serverDoctorCmd)
	rootCmd.AddCommand(serverCmd)
}

func ensureServerDBOpen() error {
	if db.IsOpen() {
		return nil
	}
	if err := db.Open(); err != nil {
		return err
	}
	if err := db.EnsureCapacidadModeloBaseCodex(); err != nil {
		return fmt.Errorf("seed capacidad/modelo base: %w", err)
	}
	return nil
}

func loadServerInfoWithHealthFallback(addr string) (*rpclocal.ServerInfo, bool, error) {
	info, err := rpclocal.LoadServerInfo()
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), rpclocal.DefaultTimeout())
		health, pingErr := rpclocal.NewClient(rpclocal.BaseURL(info.Addr), nil).Ping(ctx)
		cancel()
		if pingErr == nil {
			if strings.TrimSpace(health.ScopeID) != "" && health.ScopeID != rpclocal.CurrentScopeID() {
				return nil, false, fmt.Errorf("el servidor activo pertenece a otro scope (%s)", health.ScopeID)
			}
			if err := validateHealthStorage(health); err != nil {
				return nil, false, err
			}
			return info, false, nil
		}
	}

	if strings.TrimSpace(addr) == "" {
		addr = rpclocal.ResolveServerAddr()
	}
	resolvedAddr := strings.TrimPrefix(rpclocal.BaseURL(addr), "http://")
	ctx, cancel := context.WithTimeout(context.Background(), rpclocal.DefaultTimeout())
	defer cancel()
	health, pingErr := rpclocal.NewClient(addr, nil).Ping(ctx)
	if pingErr != nil {
		return nil, false, fmt.Errorf("no hay statefile de servidor (%v) y healthz no responde en %s: %w", err, rpclocal.BaseURL(addr), pingErr)
	}
	if !health.OK {
		return nil, false, fmt.Errorf("healthz responde pero no esta sano en %s", rpclocal.BaseURL(addr))
	}
	if strings.TrimSpace(health.ScopeID) != "" && health.ScopeID != rpclocal.CurrentScopeID() {
		return nil, false, fmt.Errorf("el servidor activo pertenece a otro scope (%s)", health.ScopeID)
	}
	if err := validateHealthStorage(health); err != nil {
		return nil, false, err
	}
	advertisedAddr := strings.TrimSpace(health.Addr)
	if advertisedAddr == "" {
		advertisedAddr = resolvedAddr
	}
	return &rpclocal.ServerInfo{
		Addr:          advertisedAddr,
		PID:           health.PID,
		Kind:          health.Kind,
		ScopeID:       health.ScopeID,
		DBPath:        legacyStorageTarget(health.StorageTarget, health.DBPath),
		StorageDriver: health.StorageDriver,
		StorageTarget: legacyStorageTarget(health.StorageTarget, health.DBPath),
		StartedAt:     health.StartedAt,
		Version:       health.Version,
	}, true, nil
}

func currentServerStorageTarget() string {
	target := strings.TrimSpace(db.CurrentStorageDisplayTarget())
	if target != "" {
		return target
	}
	return strings.TrimSpace(db.CurrentDBPath())
}

func resolveServerStorageTarget(info *rpclocal.ServerInfo) string {
	if info == nil {
		return ""
	}
	return legacyStorageTarget(info.StorageTarget, info.DBPath)
}

func legacyStorageTarget(primary, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return strings.TrimSpace(primary)
	}
	return strings.TrimSpace(fallback)
}

func validateHealthStorage(health *rpclocal.HealthResponse) error {
	if health == nil {
		return nil
	}
	if driver := strings.TrimSpace(health.StorageDriver); driver != "" && driver != db.CurrentStorageDriver() {
		return fmt.Errorf("el servidor activo usa otro driver de persistencia (%s)", driver)
	}
	if target := legacyStorageTarget(health.StorageTarget, health.DBPath); target != "" && target != currentServerStorageTarget() {
		return fmt.Errorf("el servidor activo usa otro storage target (%s)", target)
	}
	return nil
}

func resolveServerSecurityOptions(cmd *cobra.Command) (serverSecurityOptions, error) {
	tlsCert, _ := cmd.Flags().GetString("tls-cert")
	tlsKey, _ := cmd.Flags().GetString("tls-key")
	tlsClientCA, _ := cmd.Flags().GetString("tls-client-ca")

	host := ""
	if value := cmd.Flag("host"); value != nil {
		host = strings.TrimSpace(value.Value.String())
	} else if value := cmd.Flag("addr"); value != nil {
		host = strings.TrimSpace(value.Value.String())
		host = strings.TrimPrefix(strings.TrimPrefix(host, "http://"), "https://")
		if parsedHost, _, err := net.SplitHostPort(host); err == nil {
			host = parsedHost
		} else if strings.HasPrefix(host, ":") {
			host = ""
		}
	}

	if err := validateServeSecurity(host, tlsCert, tlsKey, tlsClientCA); err != nil {
		return serverSecurityOptions{}, err
	}
	return serverSecurityOptions{
		TLSCert:     strings.TrimSpace(tlsCert),
		TLSKey:      strings.TrimSpace(tlsKey),
		TLSClientCA: strings.TrimSpace(tlsClientCA),
	}, nil
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
		return commandSupportsServerMode(normalizedCommandArgs(args))
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
	cmd.Stdin = nil
	cmd.Env = buildLocalServerProcessEnv(os.Environ())
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	childPID := cmd.Process.Pid
	_ = cmd.Process.Release()

	deadline := time.Now().Add(rpclocal.DefaultStartWait())
	for time.Now().Before(deadline) {
		if !pidSigueVivo(childPID) {
			_ = rpclocal.RemoveState("")
			return fmt.Errorf("el servidor local terminó antes de publicar healthz (pid=%d). Log: %s", childPID, resumirServerLog(logPath))
		}
		ctx, cancel := context.WithTimeout(context.Background(), rpclocal.DefaultTimeout())
		pingErr := rpclocal.Ping(ctx, addr)
		cancel()
		if pingErr == nil {
			if state, stateErr := rpclocal.LoadServerInfo(); stateErr == nil {
				if strings.TrimSpace(state.Addr) != "" {
					return nil
				}
			}
		}
		time.Sleep(150 * time.Millisecond)
	}
	_ = rpclocal.RemoveState("")
	return fmt.Errorf("timeout esperando al servidor local y su statefile. Log: %s", resumirServerLog(logPath))
}

func buildLocalServerProcessEnv(base []string) []string {
	if len(base) == 0 {
		return nil
	}
	out := make([]string, 0, len(base))
	for _, entry := range base {
		key, _, found := strings.Cut(entry, "=")
		if !found {
			continue
		}
		switch key {
		case "ORQUESTA_FORCE_LOCAL", "ORQUESTA_FORCE_LOCAL_DB":
			continue
		default:
			out = append(out, entry)
		}
	}
	return out
}

func newLocalRPCState(kind, addr string) (rpclocal.ServerInfo, error) {
	token, err := randomHexToken(32)
	if err != nil {
		return rpclocal.ServerInfo{}, err
	}
	return rpclocal.ServerInfo{
		Addr:          strings.TrimPrefix(rpclocal.BaseURL(addr), "http://"),
		PID:           os.Getpid(),
		Kind:          kind,
		ScopeID:       rpclocal.CurrentScopeID(),
		DBPath:        currentServerStorageTarget(),
		StorageDriver: db.CurrentStorageDriver(),
		StorageTarget: currentServerStorageTarget(),
		Token:         token,
		StartedAt:     time.Now().UTC(),
		Version:       "dev",
	}, nil
}

func pidSigueVivo(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

func resumirServerLog(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "sin log"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("sin log legible (%v)", err)
	}
	text := strings.TrimSpace(string(data))
	if text == "" {
		return fmt.Sprintf("%s (vacío)", filepath.Base(path))
	}
	lines := strings.Split(text, "\n")
	if len(lines) > 8 {
		lines = lines[len(lines)-8:]
	}
	return fmt.Sprintf("%s :: %s", filepath.Base(path), strings.Join(lines, " | "))
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
