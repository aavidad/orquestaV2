//go:build !windows

package main

import (
	"os/exec"
	"testing"
	"time"
)

func TestProcessAliveV0TreatsZombieAsStopped(t *testing.T) {
	cmd := exec.Command("/bin/sh", "-c", "exit 0")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start child: %v", err)
	}
	defer func() { _ = cmd.Wait() }()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !processAliveV0(cmd.Process.Pid) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("processAliveV0 kept reporting exited child pid=%d as alive", cmd.Process.Pid)
}

func TestWaitUntilProcessDownV0EsperaSalidaReal(t *testing.T) {
	cmd := exec.Command("/bin/sh", "-c", "sleep 0.05")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start child: %v", err)
	}
	defer func() { _ = cmd.Wait() }()

	if !waitUntilProcessDownV0(cmd.Process.Pid, time.Second) {
		t.Fatalf("waitUntilProcessDownV0 no detecto salida pid=%d", cmd.Process.Pid)
	}
}
