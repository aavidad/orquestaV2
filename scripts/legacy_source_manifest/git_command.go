// Este fichero aísla la ejecución acotada y sin configuración privada de Git.
package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type gitFailure struct {
	exitCode int
	timeout  bool
}

func (failure gitFailure) Error() string {
	if failure.timeout {
		return "tiempo de inspección Git agotado"
	}
	return "Git terminó con código " + strconv.Itoa(failure.exitCode)
}

func gitInspectionError(path, kind string, err error) sourceRecord {
	var failure gitFailure
	if errors.As(err, &failure) && failure.timeout {
		return sealSource(sourceRecord{
			Path: path, Kind: kind, Status: "error", ErrorCode: "git_inspection_timeout",
			RequiresPhysicalInventory: true,
		})
	}
	return sealSource(sourceRecord{
		Path: path, Kind: kind, Status: "error", ErrorCode: "git_inspection_failed",
		RequiresPhysicalInventory: true,
	})
}

func runGit(timeout time.Duration, directory string, arguments ...string) ([]byte, error) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return nil, err
	}
	contextValue, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	baseArguments := []string{
		"--no-pager",
		"-c", "core.hooksPath=/dev/null",
		"-c", "safe.directory=*",
	}
	if directory != "" {
		baseArguments = append(baseArguments, "-C", directory)
	}
	command := exec.CommandContext(contextValue, gitPath, append(baseArguments, arguments...)...)
	command.Env = []string{
		"LANG=C",
		"LC_ALL=C",
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_TERMINAL_PROMPT=0",
		"GIT_NO_LAZY_FETCH=1",
		"GIT_OPTIONAL_LOCKS=0",
		"GIT_NO_REPLACE_OBJECTS=1",
	}
	var stdout bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = io.Discard
	err = command.Run()
	if contextValue.Err() != nil {
		return nil, gitFailure{exitCode: -1, timeout: true}
	}
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			return nil, gitFailure{exitCode: exitError.ExitCode()}
		}
		return nil, err
	}
	return stdout.Bytes(), nil
}

func gitText(timeout time.Duration, directory string, arguments ...string) (string, error) {
	content, err := runGit(timeout, directory, arguments...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(content)), nil
}
