// Este fichero coordina entradas explícitas, JSON en stdout y diagnósticos en stderr.
package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"golang.org/x/sys/unix"
)

type contractError struct{ code string }
type commandOptions struct {
	v3Path       string
	universePath string
	verify       bool
}

func (err contractError) Error() string { return err.code }
func contractFailure(code string) error { return contractError{code: code} }

func main() {
	os.Exit(runCommand(os.Args[1:], os.Stdout, os.Stderr))
}

func runCommand(arguments []string, output, diagnostics io.Writer) int {
	err := execute(arguments, output)
	if errors.Is(err, errHelp) {
		_, _ = io.WriteString(diagnostics, spanishText("help"))
		return 0
	}
	if err != nil {
		fmt.Fprintf(diagnostics, "%s [%s]\n", spanishText("error"), errorCode(err))
		return 1
	}
	return 0
}

func execute(arguments []string, output io.Writer) error {
	options, err := parseArguments(arguments)
	if err != nil {
		return err
	}
	sourceRaw, err := readRegularInput(options.v3Path, maxSourceBytes)
	if err != nil {
		return err
	}
	encoded, err := generate(sourceRaw)
	if err != nil {
		return err
	}
	if options.verify {
		universeRaw, readErr := readRegularInput(options.universePath, maxUniverseBytes)
		if readErr != nil || !bytes.Equal(encoded, universeRaw) {
			return contractFailure("universo_no_verificado")
		}
	}
	_, err = output.Write(encoded)
	return err
}

func generate(raw []byte) ([]byte, error) {
	sourceDigest := rawSHA256(raw)
	if sourceDigest != "sha256:"+expectedSourceSHA256 {
		return nil, contractFailure("v3_no_fijado")
	}
	document, err := decodeSource(raw)
	if err != nil {
		return nil, err
	}
	expansion, err := buildExpansion(document, sourceDigest)
	if err != nil {
		return nil, err
	}
	encoded, err := encodeExpansion(expansion)
	if err != nil {
		return nil, contractFailure("salida_invalida")
	}
	return encoded, nil
}

var errHelp = errors.New("help")

func parseArguments(arguments []string) (commandOptions, error) {
	if len(arguments) == 1 && (arguments[0] == "--help" || arguments[0] == "-h") {
		return commandOptions{}, errHelp
	}
	if len(arguments) == 2 && arguments[0] == "--v3" && arguments[1] != "" {
		return commandOptions{v3Path: arguments[1]}, nil
	}
	if len(arguments) == 5 && arguments[0] == "--verify" &&
		arguments[1] == "--v3" && arguments[2] != "" &&
		arguments[3] == "--universe" && arguments[4] != "" {
		return commandOptions{v3Path: arguments[2], universePath: arguments[4], verify: true}, nil
	}
	return commandOptions{}, contractFailure("entrada_invalida")
}

func readRegularInput(path string, maximum int) ([]byte, error) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_NONBLOCK|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
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
	if err != nil || !info.Mode().IsRegular() || info.Size() > int64(maximum) {
		return nil, contractFailure("entrada_ilegible")
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

func spanishText(key string) string {
	switch key {
	case "help":
		return "Uso: legacy_physical_expansion --v3 FICHERO_V3\n" +
			"Verificar: legacy_physical_expansion --verify --v3 FICHERO_V3 --universe UNIVERSO\n" +
			"Solo un JSON verificado se escribe en stdout; ayuda y errores usan stderr.\n"
	case "error":
		return "No se pudo sellar el universo lógico"
	default:
		return "Texto no disponible"
	}
}
