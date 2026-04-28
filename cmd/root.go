/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"orquesta/db"
	"orquesta/tareasapp/rules"
)

var rootCmd = &cobra.Command{
	Use:   "orquesta",
	Short: "CLI de orquestación multi-agente del workspace",
	Long: `Orquesta coordina proyectos, asignaciones, tareas, propuestas y sesiones
de los agentes del workspace. Los agentes deben iniciar sesión al comenzar,
obtener desde la BD sus reglas/skills/workflows y cerrar la sesión al terminar.`,
	SilenceUsage: true,
}

var (
	currentExecArgs   []string
	currentExecArgsMu sync.RWMutex
)

// Execute es el punto de entrada principal.
func Execute() {
	autoloadWorkspaceEnv()
	args := os.Args[1:]
	if localRPCEnabled(args) && !forceLocalMode(args) && !skipRemoteDelegation(args) {
		ensureRequireServerEnv()
		handled, exitCode, err := executeViaLocalServer(args, os.Stdout, os.Stderr)
		if err != nil {
			msg := fmt.Sprintf("orquesta: servidor local no disponible (%v)", err)
			fmt.Fprintln(os.Stderr, msg)
			fmt.Fprintln(os.Stderr, "usa ORQUESTA_FORCE_LOCAL_DB=1 solo para recuperacion consciente")
			os.Exit(1)
		}
		if handled {
			os.Exit(exitCode)
		}
	}
	if err := executeLocalArgs(args, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func autoloadWorkspaceEnv() {
	path := strings.TrimSpace(os.Getenv("ORQUESTA_ENV_FILE"))
	if path == "" {
		if cwd, err := os.Getwd(); err == nil && strings.TrimSpace(cwd) != "" {
			path = filepath.Join(cwd, "orquesta.env")
		}
	}
	if strings.TrimSpace(path) == "" {
		return
	}
	_ = loadEnvFileIfPresent(path)
}

func loadEnvFileIfPresent(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" || strings.TrimSpace(os.Getenv(key)) != "" {
			continue
		}
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		_ = os.Setenv(key, value)
	}
	return scanner.Err()
}

func init() {
	cobra.OnInitialize(initDB)
	rootCmd.PersistentFlags().Bool("local", false, "Fuerza ejecución local sin delegar al servidor")
	_ = rootCmd.PersistentFlags().MarkHidden("local")
	rootCmd.PersistentFlags().Bool("use-localrpc", false, "Activa delegación experimental al servidor local RPC")
	_ = rootCmd.PersistentFlags().MarkHidden("use-localrpc")

	rootCmd.AddCommand(
		agenteCmd,
		sesionCmd,
		conectorCmd,
		microprogramacionCmd,
		proyectoCmd,
		asignacionCmd,
		lockCmd,
		worktreeCmd,
		tareaCmd,
		propuestaCmd,
		votarCmd,
		memoriaCmd,
		configCmd,
		exportarCmd,
		statusCmd,
		logsCmd,
		workspaceCmd,
	)
}

func initDB() {
	args := currentOrOSArgs()
	if !commandNeedsDB(args) {
		return
	}
	if db.IsOpen() {
		return
	}
	if shouldPreferAPIClient(args) {
		return
	}
	if err := openDBForCommand(args); err != nil {
		fmt.Fprintf(os.Stderr, "error abriendo base de datos: %v\n", err)
		os.Exit(1)
	}
	rules.Install()
}

func openDBForCommand(args []string) error {
	if localRecoveryRequested(args) {
		if !localRecoveryCommandAllowed(args) {
			return localRecoveryUnsupportedError(args)
		}
		if shouldOpenRecoveryReadOnlyDB(args) {
			return db.OpenRecoveryReadOnly()
		}
		return db.Open()
	}
	return db.Open()
}

func executeLocalArgs(args []string, stdout, stderr io.Writer) error {
	setCurrentExecArgs(args)
	defer setCurrentExecArgs(nil)
	resetCommandTreeFlags(rootCmd)
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)
	rootCmd.SetArgs(args)
	return rootCmd.Execute()
}

func resetCommandTreeFlags(cmd *cobra.Command) {
	resetFlagSet(cmd.Flags())
	resetFlagSet(cmd.PersistentFlags())
	for _, child := range cmd.Commands() {
		resetCommandTreeFlags(child)
	}
}

func resetFlagSet(flags *pflag.FlagSet) {
	if flags == nil {
		return
	}
	flags.VisitAll(func(f *pflag.Flag) {
		_ = flags.Set(f.Name, normalizedFlagResetValue(f))
		f.Changed = false
	})
}

func normalizedFlagResetValue(f *pflag.Flag) string {
	if f == nil {
		return ""
	}
	valueType := strings.TrimSpace(f.Value.Type())
	switch valueType {
	case "stringSlice", "stringArray", "intSlice", "int32Slice", "int64Slice", "uintSlice", "durationSlice", "ipSlice":
		trimmed := strings.TrimSpace(f.DefValue)
		if trimmed == "[]" {
			return ""
		}
		trimmed = strings.TrimPrefix(trimmed, "[")
		trimmed = strings.TrimSuffix(trimmed, "]")
		return trimmed
	default:
		return f.DefValue
	}
}

func ensureRequireServerEnv() {
	if strings.TrimSpace(os.Getenv("ORQUESTA_REQUIRE_SERVER")) != "" {
		return
	}
	_ = os.Setenv("ORQUESTA_REQUIRE_SERVER", "1")
}

func forceLocalMode(args []string) bool {
	if strings.TrimSpace(os.Getenv("ORQUESTA_FORCE_LOCAL")) == "1" {
		return true
	}
	if strings.TrimSpace(os.Getenv("ORQUESTA_FORCE_LOCAL_DB")) == "1" {
		return true
	}
	for _, arg := range args {
		if arg == "--local" {
			return true
		}
	}
	return false
}

func commandNeedsDB(args []string) bool {
	args = normalizedCommandArgs(args)
	if len(args) == 0 {
		return false
	}
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return false
		}
	}
	switch args[0] {
	case "help", "completion", "version":
		return false
	case "persistencia":
		return false
	case "serve":
		return false
	case "server":
		if len(args) < 2 {
			return false
		}
		switch args[1] {
		case "status", "stop", "doctor":
			return false
		case "run":
			return false
		default:
			return false
		}
	case "agente":
		if len(args) > 1 && args[1] == "lanzar-plan" {
			return false
		}
		return true
	default:
		return true
	}
}

func normalizedCommandArgs(args []string) []string {
	out := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "--local" || arg == "--use-localrpc" {
			continue
		}
		out = append(out, arg)
	}
	return out
}

func localRPCEnabled(args []string) bool {
	if strings.TrimSpace(os.Getenv("ORQUESTA_USE_LOCALRPC")) == "1" {
		return true
	}
	for _, arg := range args {
		if arg == "--use-localrpc" {
			return true
		}
	}
	return false
}

func skipRemoteDelegation(args []string) bool {
	if configuredServerURL() != "" {
		return true
	}
	args = normalizedCommandArgs(args)
	return len(args) > 0 && args[0] == "persistencia"
}

func setCurrentExecArgs(args []string) {
	currentExecArgsMu.Lock()
	defer currentExecArgsMu.Unlock()
	if args == nil {
		currentExecArgs = nil
		return
	}
	currentExecArgs = append([]string(nil), args...)
}

func getCurrentExecArgs() []string {
	currentExecArgsMu.RLock()
	defer currentExecArgsMu.RUnlock()
	return append([]string(nil), currentExecArgs...)
}
