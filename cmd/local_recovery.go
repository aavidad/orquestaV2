package cmd

import (
	"fmt"
	"os"
	"strings"
)

func localRecoveryEnabled() bool {
	return localRecoveryRequested(currentOrOSArgs())
}

func runtimeModoRecuperacionLocalExplicito() bool {
	return localRecoveryEnabled()
}

func localRecoveryRequested(args []string) bool {
	if envBoolLocalRecovery("ORQUESTA_FORCE_LOCAL_DB") || envBoolLocalRecovery("ORQUESTA_FORCE_LOCAL") {
		return true
	}
	for _, arg := range args {
		if arg == "--local" {
			return true
		}
	}
	return false
}

func shouldOpenRecoveryReadOnlyDB(args []string) bool {
	if !localRecoveryRequested(args) {
		return false
	}
	tokens := commandPathTokens(normalizedCommandArgs(args))
	if len(tokens) == 0 {
		return false
	}
	switch tokens[0] {
	case "status":
		return true
	case "runtime":
		if len(tokens) <= 1 {
			return false
		}
		switch tokens[1] {
		case "listar", "ver", "handles", "transcript", "ordenes", "checkpoints", "checkpoint-ver", "mailbox", "diagnostico", "traza":
			return true
		default:
			return false
		}
	default:
		return false
	}
}

func localRecoveryCommandAllowed(args []string) bool {
	if !localRecoveryRequested(args) {
		return false
	}
	tokens := commandPathTokens(normalizedCommandArgs(args))
	if len(tokens) == 0 {
		return false
	}
	// Bloqueo explícito de comandos de escritura en modo local de recuperación.
	switch tokens[0] {
	case "tarea":
		if len(tokens) > 1 {
			switch tokens[1] {
			case "nueva", "completar", "iniciar", "cancelar", "reasignar":
				return false
			}
		}
	case "runtime":
		if len(tokens) > 1 {
			switch tokens[1] {
			case "orden-nueva", "mailbox-enviar", "nudge":
				return false
			}
		}
	case "agente":
		if len(tokens) > 1 {
			switch tokens[1] {
			case "control", "ejecutar", "lanzar-plan":
				return false
			}
		}
	}
	return shouldOpenRecoveryReadOnlyDB(args)
}

func localRecoveryUnsupportedError(args []string) error {
	name := strings.Join(commandPathTokens(normalizedCommandArgs(args)), " ")
	if strings.TrimSpace(name) == "" {
		name = "este comando"
	}
	return fmt.Errorf("el modo local de recuperacion es solo de lectura y solo cubre status/runtime de diagnostico; %s requiere servidor", name)
}

func envBoolLocalRecovery(key string) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	return value == "1" || value == "true" || value == "yes" || value == "si" || value == "on"
}

func currentOrOSArgs() []string {
	args := getCurrentExecArgs()
	if len(args) == 0 && len(os.Args) > 1 {
		args = append([]string(nil), os.Args[1:]...)
	}
	return args
}
