//go:build linux

package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"testing"
	"time"

	executor "orquesta/internal/adapters/executor/codexwork"
	protocol "orquesta/internal/adapters/protocol/codexwork"
)

const signalHelperMarker = "--codexwork-signal-helper"

func TestRunRejectsArgumentsWithOnlyStableCode(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := run(
		context.Background(), []string{"--secret", "payload"},
		strings.NewReader("prompt muy secreto"), &stdout, &stderr,
	)
	if exit != 2 || stdout.Len() != 0 || stderr.String() != codeArgumentsInvalid+"\n" {
		t.Fatalf("exit=%d stdout=%q stderr=%q", exit, stdout.String(), stderr.String())
	}
}

func TestRunNeverPrintsMalformedPacket(t *testing.T) {
	var stdout, stderr bytes.Buffer
	secret := `{"prompt":"NO_FILTRAR"`
	exit := run(context.Background(), nil, strings.NewReader(secret), &stdout, &stderr)
	if exit != 1 || stdout.Len() != 0 || stderr.String() != string(protocol.CodePacketMalformed)+"\n" {
		t.Fatalf("exit=%d stdout=%q stderr=%q", exit, stdout.String(), stderr.String())
	}
	if strings.Contains(stderr.String(), "NO_FILTRAR") {
		t.Fatal("stderr filtro el paquete")
	}
}

func TestSafeCodeFallsBackWithoutLeakingUnknownError(t *testing.T) {
	if got := safeCode(errors.New("credential=NO_FILTRAR")); got != string(executor.CodeCommandIO) {
		t.Fatalf("code=%q", got)
	}
}

func TestSIGTERMInterruptsRealStdinPipe(t *testing.T) {
	command := exec.Command(
		os.Args[0], "-test.run=^TestCodexWorkSignalHelperProcess$", "--", signalHelperMarker,
	)
	stdin, err := command.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	var waited chan error
	waitComplete := false
	defer func() {
		_ = stdin.Close()
		if command.ProcessState == nil {
			_ = command.Process.Kill()
		}
		if waitComplete {
			return
		}
		if waited == nil {
			_ = command.Wait()
			return
		}
		select {
		case <-waited:
		case <-time.After(3 * time.Second):
		}
	}()
	ready := make(chan error, 1)
	go func() {
		line, readErr := bufio.NewReader(stdout).ReadString('\n')
		if readErr == nil && line != "ready\n" {
			readErr = errors.New("ready invalido")
		}
		ready <- readErr
	}()
	select {
	case err := <-ready:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("helper no preparado")
	}
	started := time.Now()
	if err := command.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	waited = make(chan error, 1)
	go func() { waited <- command.Wait() }()
	select {
	case waitErr := <-waited:
		waitComplete = true
		var exitErr *exec.ExitError
		if !errors.As(waitErr, &exitErr) || exitErr.ExitCode() != 1 {
			t.Fatalf("wait=%v", waitErr)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("SIGTERM no interrumpio stdin real")
	}
	if time.Since(started) > 2*time.Second {
		t.Fatal("salida tras SIGTERM demasiado lenta")
	}
	if stderr.String() != string(executor.CodeCancelled)+"\n" {
		t.Fatalf("stderr=%q", stderr.String())
	}
}

func TestCodexWorkSignalHelperProcess(t *testing.T) {
	found := false
	for _, argument := range os.Args {
		if argument == signalHelperMarker {
			found = true
			break
		}
	}
	if !found {
		return
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	_, _ = os.Stdout.WriteString("ready\n")
	os.Exit(run(ctx, nil, os.Stdin, os.Stdout, os.Stderr))
}
