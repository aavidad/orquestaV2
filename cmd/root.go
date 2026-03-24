/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"orquesta/db"
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
	args := os.Args[1:]
	if localRPCEnabled(args) && !forceLocalMode(args) && !skipRemoteDelegation(args) {
		handled, exitCode, err := executeViaLocalServer(args, os.Stdout, os.Stderr)
		if err != nil {
			if allowLocalFallback(args) {
				fmt.Fprintf(os.Stderr, "orquesta: servidor local no disponible, ejecutando en modo local por politica explicita (%v)\n", err)
			} else {
				fmt.Fprintf(os.Stderr, "orquesta: servidor local no disponible (%v)\n", err)
				fmt.Fprintln(os.Stderr, "usa --local o ORQUESTA_ALLOW_LOCAL_FALLBACK=1 para modo recuperacion")
				os.Exit(1)
			}
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

func init() {
	cobra.OnInitialize(initDB)
	rootCmd.PersistentFlags().Bool("local", false, "Fuerza ejecución local sin delegar al servidor")
	_ = rootCmd.PersistentFlags().MarkHidden("local")
	rootCmd.PersistentFlags().Bool("allow-local-fallback", false, "Permite volver a modo local si el servidor no está disponible")
	rootCmd.PersistentFlags().Bool("use-localrpc", false, "Activa delegación experimental al servidor local RPC")
	_ = rootCmd.PersistentFlags().MarkHidden("use-localrpc")

	rootCmd.AddCommand(
		agenteCmd,
		sesionCmd,
		conectorCmd,
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
	)
}

func initDB() {
	if !commandNeedsDB(getCurrentExecArgs()) {
		return
	}
	if db.IsOpen() {
		return
	}
	if shouldPreferAPIClient(getCurrentExecArgs()) {
		return
	}
	if err := db.Open(); err != nil {
		fmt.Fprintf(os.Stderr, "error abriendo base de datos: %v\n", err)
		os.Exit(1)
	}
}

func executeLocalArgs(args []string, stdout, stderr io.Writer) error {
	setCurrentExecArgs(args)
	defer setCurrentExecArgs(nil)
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)
	rootCmd.SetArgs(args)
	return rootCmd.Execute()
}

func forceLocalMode(args []string) bool {
	if strings.TrimSpace(os.Getenv("ORQUESTA_FORCE_LOCAL")) == "1" {
		return true
	}
	for _, arg := range args {
		if arg == "--local" {
			return true
		}
	}
	return false
}

func allowLocalFallback(args []string) bool {
	if strings.TrimSpace(os.Getenv("ORQUESTA_ALLOW_LOCAL_FALLBACK")) == "1" {
		return true
	}
	for _, arg := range args {
		if arg == "--allow-local-fallback" {
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
	case "server":
		if len(args) < 2 {
			return false
		}
		switch args[1] {
		case "status", "stop", "doctor":
			return false
		case "run":
			return true
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
		if arg == "--local" || arg == "--allow-local-fallback" || arg == "--use-localrpc" {
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
