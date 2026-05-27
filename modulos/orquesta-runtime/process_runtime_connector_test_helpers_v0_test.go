package orquestaruntime

import (
	"encoding/json"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

const processRuntimeTestChildEnvV0 = "ORQUESTA_RUNTIME_TEST_CHILD"
const processRuntimeParentMarkerEnvV0 = "ORQUESTA_RUNTIME_PARENT_MARKER"

func TestMain(m *testing.M) {
	switch os.Getenv(processRuntimeTestChildEnvV0) {
	case "wait":
		processRuntimeTestChildWaitV0()
	case "ignore":
		processRuntimeTestChildIgnoreInterruptV0()
	case "exit":
		os.Exit(0)
	case "envdump":
		processRuntimeTestChildEnvDumpV0()
	}
	_ = os.Setenv("ORQUESTA_DETAIL_PROHIBITED_RAILS", "on")
	os.Exit(m.Run())
}

func processRuntimeTestChildWaitV0() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	defer signal.Stop(signals)

	select {
	case <-signals:
		os.Exit(0)
	case <-time.After(30 * time.Second):
		os.Exit(3)
	}
}

func processRuntimeTestChildIgnoreInterruptV0() {
	_ = os.WriteFile("ignore-started.txt", []byte("1"), 0o600)
	signal.Ignore(os.Interrupt, syscall.SIGINT, syscall.SIGUSR1)
	time.Sleep(30 * time.Second)
	os.Exit(3)
}

func processRuntimeTestChildEnvDumpV0() {
	_ = os.WriteFile("env.txt", []byte(os.Getenv(processRuntimeParentMarkerEnvV0)), 0o600)
	os.Exit(0)
}

func processRuntimeLaunchRequestForTestV0(t *testing.T, childMode string) ProcessRuntimeLaunchRequestV0 {
	t.Helper()
	if !filepath.IsAbs(os.Args[0]) {
		t.Fatalf("test binary path no es absoluto: %q", os.Args[0])
	}
	return ProcessRuntimeLaunchRequestV0{
		CommandPath: os.Args[0],
		Env: []string{
			processRuntimeTestChildEnvV0 + "=" + childMode,
		},
		WorkingDir: t.TempDir(),
	}
}

func waitForProcessRuntimeStatusV0(
	t *testing.T,
	connector *ProcessRuntimeConnectorV0,
	processRef string,
	status ProcessRuntimeStatusV0,
) ProcessRuntimeSnapshotV0 {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		snapshot, err := connector.SnapshotV0(processRef)
		requireNoProcessRuntimeErrorV0(t, err)
		if snapshot.Status == status {
			return snapshot
		}
		time.Sleep(10 * time.Millisecond)
	}
	snapshot, err := connector.SnapshotV0(processRef)
	requireNoProcessRuntimeErrorV0(t, err)
	t.Fatalf("status final = %q, se esperaba %q", snapshot.Status, status)
	return ProcessRuntimeSnapshotV0{}
}

func assertProcessRuntimeSnapshotDoesNotLeakV0(
	t *testing.T,
	snapshot ProcessRuntimeSnapshotV0,
	req ProcessRuntimeLaunchRequestV0,
) {
	t.Helper()
	raw, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}
	text := string(raw)
	for _, forbidden := range []string{
		req.CommandPath,
		req.WorkingDir,
		processRuntimeTestChildEnvV0,
		"pid",
	} {
		if forbidden != "" && strings.Contains(strings.ToLower(text), strings.ToLower(forbidden)) {
			t.Fatalf("snapshot filtra detalle prohibido %q: %s", forbidden, text)
		}
	}
}

func requireNoProcessRuntimeErrorV0(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
}

func requireProcessRuntimeCodeV0(
	t *testing.T,
	err error,
	code ProcessRuntimeErrorCodeV0,
) {
	t.Helper()
	if err == nil {
		t.Fatalf("error nil, se esperaba codigo %q", code)
	}
	runtimeErr, ok := err.(ProcessRuntimeErrorV0)
	if !ok {
		t.Fatalf("tipo de error = %T, se esperaba ProcessRuntimeErrorV0", err)
	}
	if runtimeErr.Code != code {
		t.Fatalf("codigo = %q, se esperaba %q", runtimeErr.Code, code)
	}
}
