package orquestaruntime

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const processRuntimeTestChildEnvV0 = "ORQUESTA_RUNTIME_TEST_CHILD"
const processRuntimeParentMarkerEnvV0 = "ORQUESTA_RUNTIME_PARENT_MARKER"

func TestMain(m *testing.M) {
	switch os.Getenv(processRuntimeTestChildEnvV0) {
	case "wait":
		processRuntimeTestChildWaitV0()
	case "exit":
		os.Exit(0)
	case "envdump":
		processRuntimeTestChildEnvDumpV0()
	}
	os.Exit(m.Run())
}

func TestProcessRuntimeConnectorV0LanzaProcesoRealYLoPara(t *testing.T) {
	connector := NewProcessRuntimeConnectorV0()
	req := processRuntimeLaunchRequestForTestV0(t, "wait")

	launched, err := connector.LaunchV0(context.Background(), req)
	requireNoProcessRuntimeErrorV0(t, err)
	if launched.Status != ProcessRuntimeRunningV0 {
		t.Fatalf("status launch = %q", launched.Status)
	}
	if launched.ProcessRef == "" || launched.SessionRef == "" || launched.LaunchRef == "" {
		t.Fatalf("refs publicas incompletas: %+v", launched)
	}
	assertProcessRuntimeSnapshotDoesNotLeakV0(t, launched, req)

	running, err := connector.SnapshotV0(launched.ProcessRef)
	requireNoProcessRuntimeErrorV0(t, err)
	if running.Status != ProcessRuntimeRunningV0 {
		t.Fatalf("status snapshot = %q", running.Status)
	}
	if running.SessionRef != launched.SessionRef {
		t.Fatalf("session_ref cambio: launch=%q snapshot=%q", launched.SessionRef, running.SessionRef)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	stopped, err := connector.StopV0(ctx, launched.ProcessRef)
	requireNoProcessRuntimeErrorV0(t, err)
	if stopped.Status != ProcessRuntimeStoppedV0 {
		t.Fatalf("status stop = %q", stopped.Status)
	}
	if stopped.StopRef == "" {
		t.Fatalf("stop_ref vacia: %+v", stopped)
	}
	assertProcessRuntimeSnapshotDoesNotLeakV0(t, stopped, req)
}

func TestProcessRuntimeConnectorV0StopEsIdempotente(t *testing.T) {
	connector := NewProcessRuntimeConnectorV0()
	launched, err := connector.LaunchV0(
		context.Background(),
		processRuntimeLaunchRequestForTestV0(t, "wait"),
	)
	requireNoProcessRuntimeErrorV0(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	first, err := connector.StopV0(ctx, launched.ProcessRef)
	requireNoProcessRuntimeErrorV0(t, err)
	second, err := connector.StopV0(ctx, launched.ProcessRef)
	requireNoProcessRuntimeErrorV0(t, err)

	if second.Status != ProcessRuntimeStoppedV0 {
		t.Fatalf("status segundo stop = %q", second.Status)
	}
	if second.StopRef != first.StopRef {
		t.Fatalf("stop idempotente cambio stop_ref: first=%q second=%q", first.StopRef, second.StopRef)
	}
}

func TestProcessRuntimeConnectorV0ProcesoQueSaleSoloQuedaStopped(t *testing.T) {
	connector := NewProcessRuntimeConnectorV0()
	launched, err := connector.LaunchV0(
		context.Background(),
		processRuntimeLaunchRequestForTestV0(t, "exit"),
	)
	requireNoProcessRuntimeErrorV0(t, err)

	stopped := waitForProcessRuntimeStatusV0(
		t,
		connector,
		launched.ProcessRef,
		ProcessRuntimeStoppedV0,
	)
	if stopped.StopRef != "" {
		t.Fatalf("proceso que salio solo no debe inventar stop_ref: %+v", stopped)
	}
}

func TestProcessRuntimeConnectorV0RechazaShellYEnvHeredado(t *testing.T) {
	connector := NewProcessRuntimeConnectorV0()
	req := processRuntimeLaunchRequestForTestV0(t, "wait")
	req.CommandPath = filepath.Join(string(filepath.Separator), "bin", "sh")
	_, err := connector.LaunchV0(context.Background(), req)
	requireProcessRuntimeCodeV0(t, err, ProcessRuntimeShellProhibidaV0)

	req = processRuntimeLaunchRequestForTestV0(t, "wait")
	req.Env = nil
	_, err = connector.LaunchV0(context.Background(), req)
	requireProcessRuntimeCodeV0(t, err, ProcessRuntimeEnvProhibidoV0)
}

func TestProcessRuntimeConnectorV0NoHeredaEntornoPadre(t *testing.T) {
	t.Setenv(processRuntimeParentMarkerEnvV0, "parent-value-should-not-leak")

	connector := NewProcessRuntimeConnectorV0()
	req := processRuntimeLaunchRequestForTestV0(t, "envdump")
	launched, err := connector.LaunchV0(context.Background(), req)
	requireNoProcessRuntimeErrorV0(t, err)
	waitForProcessRuntimeStatusV0(t, connector, launched.ProcessRef, ProcessRuntimeStoppedV0)

	data, err := os.ReadFile(filepath.Join(req.WorkingDir, "env.txt"))
	if err != nil {
		t.Fatalf("leer env dump: %v", err)
	}
	if strings.TrimSpace(string(data)) != "" {
		t.Fatalf("entorno padre filtrado al proceso: %q", string(data))
	}
}

func TestProcessRuntimeConnectorV0AceptaPathOperativoExplicito(t *testing.T) {
	connector := NewProcessRuntimeConnectorV0()
	req := processRuntimeLaunchRequestForTestV0(t, "exit")
	req.Env = append(req.Env, "PATH="+filepath.Join(t.TempDir(), "bin")+string(os.PathListSeparator)+"/usr/bin")

	launched, err := connector.LaunchV0(context.Background(), req)
	requireNoProcessRuntimeErrorV0(t, err)
	waitForProcessRuntimeStatusV0(t, connector, launched.ProcessRef, ProcessRuntimeStoppedV0)
}

func TestValidateProcessRuntimeLaunchRequestV0AceptaWorkspaceRealPrivado(t *testing.T) {
	req := ProcessRuntimeLaunchRequestV0{
		CommandPath: "/home/alberto/Trabajo/orquesta/.orquesta-runs/run-001/agent",
		Env:         []string{},
		WorkingDir:  "/home/alberto/Trabajo/orquesta/proyectos/run-001",
	}

	requireNoProcessRuntimeErrorV0(t, validateProcessRuntimeLaunchRequestV0(req))
}

func TestProcessRuntimeConnectorV0RechazaPathOperativoInseguro(t *testing.T) {
	for _, item := range []string{
		"PATH=",
		"PATH=bin",
		"PATH=/tmp/oauth/bin",
		"PATH=/tmp/access_token/bin",
		"MY_PATH=/usr/bin",
	} {
		t.Run(item, func(t *testing.T) {
			req := processRuntimeLaunchRequestForTestV0(t, "exit")
			req.Env = append(req.Env, item)

			_, err := NewProcessRuntimeConnectorV0().LaunchV0(context.Background(), req)
			requireProcessRuntimeCodeV0(t, err, ProcessRuntimeEnvProhibidoV0)
		})
	}
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
