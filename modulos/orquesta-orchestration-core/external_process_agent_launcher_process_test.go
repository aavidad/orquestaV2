package orquestacionnucleoapp

import (
	"os"
	"os/signal"
	"path/filepath"
	"testing"
	"time"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

const externalProcessLauncherChildEnvV0 = "ORQUESTA_NUCLEO_PROCESS_CHILD"

func TestMain(m *testing.M) {
	switch os.Getenv(externalProcessLauncherChildEnvV0) {
	case "exit":
		os.Exit(0)
	case "wait":
		externalProcessLauncherChildWaitV0()
	}
	os.Exit(m.Run())
}

func externalProcessRuntimeRequestForTestV0(
	t *testing.T,
	childMode string,
) orquestaruntime.ProcessRuntimeLaunchRequestV0 {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("test executable: %v", err)
	}
	if !filepath.IsAbs(executable) {
		t.Fatalf("test executable path no absoluto: %q", executable)
	}
	return orquestaruntime.ProcessRuntimeLaunchRequestV0{
		CommandPath: executable,
		Env: []string{
			externalProcessLauncherChildEnvV0 + "=" + childMode,
		},
		WorkingDir: t.TempDir(),
	}
}

func waitExternalProcessRuntimeStatusV0(
	t *testing.T,
	connector *orquestaruntime.ProcessRuntimeConnectorV0,
	processRef string,
	status orquestaruntime.ProcessRuntimeStatusV0,
) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		snapshot, err := connector.SnapshotV0(processRef)
		if err != nil {
			t.Fatalf("snapshot process: %v", err)
		}
		if snapshot.Status == status {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timeout esperando status %s en process_ref=%s", status, processRef)
}

func externalProcessLauncherChildWaitV0() {
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
