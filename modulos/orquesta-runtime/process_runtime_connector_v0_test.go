package orquestaruntime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

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
	assertProcessRuntimeLaunchReceiptV0(t, launched, req)
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
	if stopped.StopReasonCode != ProcessRuntimeStopCooperativeSignalSentV0 {
		t.Fatalf("stop_reason_code = %q", stopped.StopReasonCode)
	}
	if stopped.StopGraceDeadline == "" {
		t.Fatalf("stop_grace_deadline vacio: %+v", stopped)
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

func TestProcessRuntimeConnectorV0AdoptaProcesoVivoYLoPara(t *testing.T) {
	original := NewProcessRuntimeConnectorV0()
	launched, err := original.LaunchV0(
		context.Background(),
		processRuntimeLaunchRequestForTestV0(t, "wait"),
	)
	requireNoProcessRuntimeErrorV0(t, err)
	if launched.PID <= 0 {
		t.Fatalf("pid no registrado: %+v", launched)
	}

	adopter := NewProcessRuntimeConnectorV0()
	adopted, err := adopter.AdoptProcessV0(context.Background(), ProcessRuntimeSnapshotV0{
		SchemaVersion: ProcessRuntimeConnectorVersionV0,
		ProcessRef:    launched.ProcessRef,
		SessionRef:    launched.SessionRef,
		LaunchRef:     launched.LaunchRef,
		PID:           launched.PID,
		Status:        ProcessRuntimeRunningV0,
	})
	requireNoProcessRuntimeErrorV0(t, err)
	if adopted.Status != ProcessRuntimeRunningV0 || adopted.ProcessRef != launched.ProcessRef {
		t.Fatalf("adopted=%+v launched=%+v", adopted, launched)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	stopped, err := adopter.StopV0(ctx, launched.ProcessRef)
	requireNoProcessRuntimeErrorV0(t, err)
	if stopped.Status != ProcessRuntimeStoppedV0 {
		t.Fatalf("status stop adoptado=%q", stopped.Status)
	}
	waitForProcessRuntimeStatusV0(t, original, launched.ProcessRef, ProcessRuntimeStoppedV0)
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

func TestValidateProcessRuntimeLaunchRequestV0PermiteShellYEnvAmplioConRailsDetalleOff(t *testing.T) {
	t.Setenv("ORQUESTA_DETAIL_PROHIBITED_RAILS", "off")
	req := processRuntimeLaunchRequestForTestV0(t, "wait")
	req.CommandPath = filepath.Join(string(filepath.Separator), "bin", "sh")
	req.Env = []string{
		"HOME=/home/alberto",
		"OPENAI_TOKEN=token-real-proyectado-por-operador",
		"PATH=/usr/bin",
	}

	requireNoProcessRuntimeErrorV0(t, validateProcessRuntimeLaunchRequestV0(req))
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
