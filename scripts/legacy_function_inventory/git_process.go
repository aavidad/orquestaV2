// Este fichero encapsula el número acotado de procesos Git y el protocolo
// persistente de lectura por lotes. No conoce registros ni analiza fuente Go.
package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
)

var errGitObjectMissing = errors.New("objeto Git ausente")

type gitBatch struct {
	command *exec.Cmd
	input   io.WriteCloser
	output  *bufio.Reader
	stderr  bytes.Buffer
	closed  bool
}

type gitObject struct {
	oid     string
	kind    string
	content []byte
}

func gitLines(repository string, processStarted func(), arguments ...string) ([]string, error) {
	content, err := gitBytes(repository, processStarted, arguments...)
	if err != nil {
		return nil, err
	}
	var lines []string
	for _, line := range strings.Split(strings.TrimSpace(string(content)), "\n") {
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines, nil
}

func gitText(repository string, processStarted func(), arguments ...string) (string, error) {
	content, err := gitBytes(repository, processStarted, arguments...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(content)), nil
}

func gitBytes(repository string, processStarted func(), arguments ...string) ([]byte, error) {
	command := exec.Command("git", append([]string{"-C", repository}, arguments...)...)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	content, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(arguments, " "), err, strings.TrimSpace(stderr.String()))
	}
	if processStarted != nil {
		processStarted()
	}
	return content, nil
}

func newGitBatch(repository string, processStarted func()) (*gitBatch, error) {
	command := exec.Command("git", "-C", repository, "cat-file", "--batch")
	input, err := command.StdinPipe()
	if err != nil {
		return nil, err
	}
	output, err := command.StdoutPipe()
	if err != nil {
		input.Close()
		return nil, err
	}
	batch := &gitBatch{command: command, input: input, output: bufio.NewReader(output)}
	command.Stderr = &batch.stderr
	if err := command.Start(); err != nil {
		input.Close()
		return nil, fmt.Errorf("iniciar git cat-file --batch: %w", err)
	}
	if processStarted != nil {
		processStarted()
	}
	return batch, nil
}

func (batch *gitBatch) get(expression string) (gitObject, error) {
	if batch.closed {
		return gitObject{}, errors.New("lector Git por lotes cerrado")
	}
	if expression == "" || strings.ContainsAny(expression, "\r\n") {
		return gitObject{}, errors.New("expresión de objeto Git inválida")
	}
	if _, err := io.WriteString(batch.input, expression+"\n"); err != nil {
		return gitObject{}, fmt.Errorf("consultar objeto Git %s: %w", expression, err)
	}
	header, err := batch.output.ReadString('\n')
	if err != nil {
		return gitObject{}, fmt.Errorf("leer cabecera del objeto Git %s: %w", expression, err)
	}
	header = strings.TrimSuffix(header, "\n")
	if strings.HasSuffix(header, " missing") {
		return gitObject{}, fmt.Errorf("%w: %s", errGitObjectMissing, expression)
	}
	fields := strings.Fields(header)
	if len(fields) != 3 {
		return gitObject{}, fmt.Errorf("cabecera inválida del objeto Git %s: %q", expression, header)
	}
	size, err := strconv.Atoi(fields[2])
	if err != nil || size < 0 {
		return gitObject{}, fmt.Errorf("tamaño inválido del objeto Git %s: %q", expression, fields[2])
	}
	content := make([]byte, size)
	if _, err := io.ReadFull(batch.output, content); err != nil {
		return gitObject{}, fmt.Errorf("leer objeto Git %s: %w", expression, err)
	}
	trailing, err := batch.output.ReadByte()
	if err != nil {
		return gitObject{}, fmt.Errorf("leer terminador del objeto Git %s: %w", expression, err)
	}
	if trailing != '\n' {
		return gitObject{}, fmt.Errorf("terminador inválido del objeto Git %s", expression)
	}
	return gitObject{oid: fields[0], kind: fields[1], content: content}, nil
}

func (batch *gitBatch) close() error {
	if batch == nil || batch.closed {
		return nil
	}
	batch.closed = true
	inputErr := batch.input.Close()
	waitErr := batch.command.Wait()
	if inputErr != nil {
		return inputErr
	}
	if waitErr != nil {
		return fmt.Errorf("cerrar git cat-file --batch: %w: %s", waitErr, strings.TrimSpace(batch.stderr.String()))
	}
	return nil
}
