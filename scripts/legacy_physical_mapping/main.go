// Este fichero confina tres entradas explícitas y separa resultado y diagnóstico.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

type contractError struct{ code string }
type commandOptions struct {
	v3Path, universePath, candidatePath string
}

var errHelp = errors.New("help")

var spanishCatalog = map[string]string{
	"help": "Uso: legacy_physical_mapping --v3 V3 --universe UNIVERSO --candidate CANDIDATO\n" +
		"Solo un veredicto coherente se escribe en stdout; ayuda y errores usan stderr.\n",
	"error":    "No se pudo validar el candidato privado de mapeo",
	"fallback": "Texto no disponible",
}

func (err contractError) Error() string { return err.code }
func contractFailure(code string) error { return contractError{code: code} }

func main() {
	os.Exit(runCommand(os.Args[1:], os.Stdout, os.Stderr))
}

func runCommand(arguments []string, output, diagnostics io.Writer) int {
	result, err := execute(arguments)
	if errors.Is(err, errHelp) {
		_, _ = io.WriteString(diagnostics, catalogText("help"))
		return 0
	}
	if err != nil {
		fmt.Fprintf(diagnostics, "%s [%s]\n", catalogText("error"), errorCode(err))
		return 1
	}
	written, writeErr := output.Write(result)
	if writeErr != nil || written != len(result) {
		fmt.Fprintf(diagnostics, "%s [%s]\n", catalogText("error"), "salida_incompleta")
		return 1
	}
	return 0
}

func execute(arguments []string) ([]byte, error) {
	options, err := parseArguments(arguments)
	if err != nil {
		return nil, err
	}
	v3Raw, err := readRegularInput(options.v3Path, maxBaseBytes, false)
	if err != nil {
		return nil, err
	}
	universeRaw, err := readRegularInput(options.universePath, maxBaseBytes, false)
	if err != nil {
		return nil, err
	}
	candidateRaw, err := readRegularInput(options.candidatePath, maxCandidateBytes, true)
	if err != nil {
		return nil, err
	}
	return validateMapping(v3Raw, universeRaw, candidateRaw)
}

func parseArguments(arguments []string) (commandOptions, error) {
	if len(arguments) == 1 && (arguments[0] == "--help" || arguments[0] == "-h") {
		return commandOptions{}, errHelp
	}
	if len(arguments) != 6 ||
		arguments[0] != "--v3" || arguments[1] == "" ||
		arguments[2] != "--universe" || arguments[3] == "" ||
		arguments[4] != "--candidate" || arguments[5] == "" {
		return commandOptions{}, contractFailure("entrada_invalida")
	}
	return commandOptions{
		v3Path: arguments[1], universePath: arguments[3], candidatePath: arguments[5],
	}, nil
}

func readRegularInput(path string, maximum int, private bool) ([]byte, error) {
	how := &unix.OpenHow{
		Flags:   unix.O_RDONLY | unix.O_NONBLOCK | unix.O_CLOEXEC | unix.O_NOFOLLOW,
		Resolve: unix.RESOLVE_NO_SYMLINKS,
	}
	fd, err := unix.Openat2(unix.AT_FDCWD, path, how)
	if err != nil {
		return nil, contractFailure("entrada_ilegible")
	}
	file := os.NewFile(uintptr(fd), "entrada_explicitamente_autorizada")
	if file == nil {
		_ = unix.Close(fd)
		return nil, contractFailure("entrada_ilegible")
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() < 0 || info.Size() > int64(maximum) {
		return nil, contractFailure("entrada_ilegible")
	}
	if private {
		stat, owned := info.Sys().(*syscall.Stat_t)
		if info.Mode().Perm() != 0o600 || !owned || stat.Uid != uint32(os.Geteuid()) {
			return nil, contractFailure("candidato_no_privado")
		}
	}
	raw, err := io.ReadAll(io.LimitReader(file, int64(maximum)+1))
	if err != nil || len(raw) > maximum {
		return nil, contractFailure("entrada_ilegible")
	}
	return raw, nil
}

func errorCode(err error) string {
	var failure contractError
	if errors.As(err, &failure) {
		return failure.code
	}
	return "fallo_interno"
}

func catalogText(key string) string {
	if text := spanishCatalog[key]; text != "" {
		return text
	}
	return spanishCatalog["fallback"]
}
