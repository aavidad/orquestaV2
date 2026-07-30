// Este fichero encapsula las consultas Git de solo lectura y su transporte.
package main

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

func readRefs(repository string, started func()) ([]refInfo, error) {
	content, err := gitBytes(
		repository,
		started,
		"for-each-ref",
		"--format=%(refname)%00%(objectname)%00%(objecttype)%00%(*objectname)%00%(*objecttype)",
	)
	if err != nil {
		return nil, err
	}
	var refs []refInfo
	for _, line := range bytes.Split(bytes.TrimSpace(content), []byte{'\n'}) {
		if len(line) == 0 {
			continue
		}
		fields := bytes.Split(line, []byte{0})
		if len(fields) != 5 {
			return nil, fmt.Errorf("referencia Git con %d campos", len(fields))
		}
		item := refInfo{
			Name:       string(fields[0]),
			Object:     string(fields[1]),
			ObjectType: string(fields[2]),
		}
		item.Target, item.TargetType = item.Object, item.ObjectType
		if item.ObjectType == "tag" && len(fields[3]) > 0 && len(fields[4]) > 0 {
			item.Target, item.TargetType = string(fields[3]), string(fields[4])
		}
		refs = append(refs, item)
	}
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].Name != refs[j].Name {
			return refs[i].Name < refs[j].Name
		}
		return refs[i].Object < refs[j].Object
	})
	return refs, nil
}

func readHistory(repository string, refs []refInfo, started func()) (map[string]historyEntry, error) {
	commitSet := map[string]struct{}{}
	for _, ref := range refs {
		if ref.TargetType == "commit" {
			commitSet[ref.Target] = struct{}{}
		}
	}
	commits := make([]string, 0, len(commitSet))
	for commit := range commitSet {
		commits = append(commits, commit)
	}
	sort.Strings(commits)
	if len(commits) == 0 {
		return map[string]historyEntry{}, nil
	}
	content, err := gitBytesInput(
		repository,
		started,
		[]byte(strings.Join(commits, "\n")+"\n"),
		"log",
		"--stdin",
		"--format=%H%x00%T%x00%P",
	)
	if err != nil {
		return nil, err
	}
	result := map[string]historyEntry{}
	for _, line := range bytes.Split(bytes.TrimSpace(content), []byte{'\n'}) {
		if len(line) == 0 {
			continue
		}
		fields := bytes.Split(line, []byte{0})
		if len(fields) != 3 {
			return nil, fmt.Errorf("confirmación Git con %d campos", len(fields))
		}
		parents := strings.Fields(string(fields[2]))
		result[string(fields[0])] = historyEntry{tree: string(fields[1]), parents: parents}
	}
	return result, nil
}

func newGitBatch(repository string, started func()) (*gitBatch, error) {
	command := gitCommand(repository, "cat-file", "--batch")
	batch := &gitBatch{command: command}
	command.Stderr = &batch.stderr
	var err error
	batch.input, err = command.StdinPipe()
	if err != nil {
		return nil, err
	}
	output, err := command.StdoutPipe()
	if err != nil {
		return nil, err
	}
	batch.output = bufio.NewReaderSize(output, 64*1024)
	if started != nil {
		started()
	}
	if err := command.Start(); err != nil {
		return nil, err
	}
	return batch, nil
}

func (batch *gitBatch) header(oid string) (gitObjectHeader, error) {
	if batch.closed {
		return gitObjectHeader{}, errors.New("lector Git cerrado")
	}
	if _, err := io.WriteString(batch.input, oid+"\n"); err != nil {
		return gitObjectHeader{}, err
	}
	line, err := batch.output.ReadString('\n')
	if err != nil {
		return gitObjectHeader{}, err
	}
	fields := strings.Fields(line)
	if len(fields) == 2 && fields[1] == "missing" {
		return gitObjectHeader{oid: fields[0], missing: true}, nil
	}
	if len(fields) != 3 {
		return gitObjectHeader{}, fmt.Errorf("cabecera cat-file inválida: %q", boundedDetail(line))
	}
	size, err := strconv.ParseInt(fields[2], 10, 64)
	if err != nil || size < 0 {
		return gitObjectHeader{}, fmt.Errorf("tamaño Git inválido: %q", fields[2])
	}
	return gitObjectHeader{oid: fields[0], kind: fields[1], size: size}, nil
}

func (batch *gitBatch) readContent(target []byte) error {
	if _, err := io.ReadFull(batch.output, target); err != nil {
		return err
	}
	return batch.consumeDelimiter()
}

func (batch *gitBatch) discard(size int64) error {
	if _, err := io.CopyN(io.Discard, batch.output, size); err != nil {
		return err
	}
	return batch.consumeDelimiter()
}

func (batch *gitBatch) consumeDelimiter() error {
	delimiter, err := batch.output.ReadByte()
	if err != nil {
		return err
	}
	if delimiter != '\n' {
		return fmt.Errorf("delimitador cat-file inválido: %d", delimiter)
	}
	return nil
}

func (batch *gitBatch) close() error {
	if batch == nil || batch.closed {
		return nil
	}
	batch.closed = true
	_ = batch.input.Close()
	if err := batch.command.Wait(); err != nil {
		return fmt.Errorf("cat-file --batch: %w: %s", err, boundedDetail(batch.stderr.String()))
	}
	return nil
}

func gitText(repository string, started func(), arguments ...string) (string, error) {
	content, err := gitBytes(repository, started, arguments...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(content)), nil
}

func gitBytes(repository string, started func(), arguments ...string) ([]byte, error) {
	return gitBytesInput(repository, started, nil, arguments...)
}

func gitBytesInput(repository string, started func(), input []byte, arguments ...string) ([]byte, error) {
	command := gitCommand(repository, arguments...)
	if input != nil {
		command.Stdin = bytes.NewReader(input)
	}
	if started != nil {
		started()
	}
	content, err := command.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(arguments, " "), err, boundedDetail(string(content)))
	}
	return content, nil
}

func gitCommand(repository string, arguments ...string) *exec.Cmd {
	all := []string{
		"--no-pager",
		"--no-replace-objects",
		"-c", "core.hooksPath=/dev/null",
		"-c", "core.fsmonitor=false",
		"-c", "credential.helper=",
		"-C", repository,
	}
	all = append(all, arguments...)
	command := exec.Command("git", all...)
	command.Env = []string{
		"GCM_INTERACTIVE=Never",
		"GIT_ATTR_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_NO_LAZY_FETCH=1",
		"GIT_NO_REPLACE_OBJECTS=1",
		"GIT_OPTIONAL_LOCKS=0",
		"GIT_PAGER=cat",
		"GIT_TERMINAL_PROMPT=0",
		"LANG=C",
		"LC_ALL=C",
	}
	return command
}

func objectIDBytes(format string) (int, error) {
	switch format {
	case "sha1":
		return sha1.Size, nil
	case "sha256":
		return sha256.Size, nil
	default:
		return 0, fmt.Errorf("formato de objetos Git no soportado: %q", format)
	}
}
