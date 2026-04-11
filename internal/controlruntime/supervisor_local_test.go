package controlruntime

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestConsultarEstadoLocalDetectaSessionIDSinSobrescribirDriver(t *testing.T) {
	resetSupervisoresLocalesForTest(t)
	tmp := t.TempDir()
	wrapper := filepath.Join(tmp, "codex-perfiles", "bin", "codex-perfil")
	if err := os.MkdirAll(filepath.Dir(wrapper), 0o755); err != nil {
		t.Fatalf("mkdir wrapper: %v", err)
	}
	startedAt := time.Now().UTC()
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	sessionDir := filepath.Join(filepath.Dir(filepath.Dir(wrapper)), "homes", "Codex3", "sessions", startedAt.In(time.Local).Format("2006"), startedAt.In(time.Local).Format("01"), startedAt.In(time.Local).Format("02"))
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatalf("mkdir session dir: %v", err)
	}
	sessionFile := filepath.Join(sessionDir, "supervisor-status.jsonl")
	sessionMeta := `{"timestamp":"` + startedAt.Add(time.Second).Format(time.RFC3339Nano) + `","type":"session_meta","payload":{"id":"sess-supervisor-123","timestamp":"` + startedAt.Add(time.Second).Format(time.RFC3339Nano) + `","cwd":"` + workingDir + `"}}` + "\n"
	if err := os.WriteFile(sessionFile, []byte(sessionMeta), 0o644); err != nil {
		t.Fatalf("write session file: %v", err)
	}

	pid := int64(os.Getpid())
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("executable: %v", err)
	}
	estado, observed, err := ConsultarEstadoLocal(ObjetivoProceso{
		PID: &pid,
		MetadataJSON: `{"driver":"process_pty_cli","rendered_command":"'` + wrapper + `' 'Codex3'","wrapped_command":"` + exe + `","working_dir":"` +
			workingDir + `","started_at":"` + startedAt.Format(time.RFC3339Nano) + `"}`,
	})
	if err != nil {
		t.Fatalf("ConsultarEstadoLocal: %v", err)
	}
	if !observed || estado == nil {
		t.Fatalf("ConsultarEstadoLocal deberia observar el proceso local: observed=%v estado=%+v", observed, estado)
	}
	if !estado.Vivo {
		t.Fatalf("el proceso actual deberia seguir vivo: %+v", estado)
	}
	meta := metadataMap(estado.MetadataJSON)
	if got := stringValueFromMetadata(meta, "external_session_id"); got != "sess-supervisor-123" {
		t.Fatalf("external_session_id inesperado: %q", got)
	}
	if got := stringValueFromMetadata(meta, "supervisor_driver"); got != "local_runtime_supervisor" {
		t.Fatalf("supervisor_driver inesperado: %q", got)
	}
	if got := stringValueFromMetadata(meta, "driver"); got != "" {
		t.Fatalf("el supervisor no deberia sobrescribir driver: %q", got)
	}
}

func TestConsultarEstadoLocalRehidrataSupervisorDesdeManifestSinPIDInicial(t *testing.T) {
	resetSupervisoresLocalesForTest(t)
	tmp := t.TempDir()
	workingDir := filepath.Join(tmp, "repo")
	runDir := filepath.Join(workingDir, ".orquesta-runtime", "codex7", "20260402-150000-000000001")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run dir: %v", err)
	}
	manifestPath := filepath.Join(runDir, "runtime.json")
	payload := map[string]any{
		"agente":              "Codex7",
		"proyecto":            "orquestador",
		"working_dir":         workingDir,
		"driver":              "process_pty_cli",
		"pid":                 os.Getpid(),
		"stdin_path":          filepath.Join(runDir, "pty.stdin"),
		"stdin_raw_path":      filepath.Join(runDir, "pty.stdin.raw"),
		"log_path":            filepath.Join(runDir, "pty.log"),
		"rendered_command":    "codex-perfil Codex7",
		"wrapped_command":     "script ...",
		"supervisor_ref":      runDir,
		"mailbox_delivery_mode": "bootstrap_only",
		"created_at":          time.Now().UTC().Format(time.RFC3339Nano),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := os.WriteFile(manifestPath, append(data, '\n'), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	estado, observed, err := ConsultarEstadoLocal(ObjetivoProceso{
		HandleKind:   "session",
		HandleRef:    "remote-session-like",
		MetadataJSON: `{"working_dir":"` + workingDir + `","agente":"Codex7","proyecto":"orquestador"}`,
	})
	if err != nil {
		t.Fatalf("ConsultarEstadoLocal: %v", err)
	}
	if !observed || estado == nil {
		t.Fatalf("deberia rehidratar un supervisor local desde manifest: observed=%v estado=%+v", observed, estado)
	}
	meta := metadataMap(estado.MetadataJSON)
	if got := stringValueFromMetadata(meta, "stdin_path"); got != filepath.Join(runDir, "pty.stdin") {
		t.Fatalf("stdin_path no rehidratado: %q meta=%+v", got, meta)
	}
	if got := stringValueFromMetadata(meta, "supervisor_driver"); got != "local_runtime_supervisor" {
		t.Fatalf("supervisor_driver inesperado: %q meta=%+v", got, meta)
	}
}

func TestSupervisorLocalEmiteHeartbeatYFinalizacionAlActivarse(t *testing.T) {
	cmd := exec.Command("sleep", "1")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start sleep: %v", err)
	}
	t.Cleanup(func() {
		if cmd.Process == nil {
			return
		}
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})

	signals := make(chan SupervisorSignal, 4)
	prevHandler := func() SupervisorSignalHandler {
		supervisorSignalRegistry.mu.RLock()
		defer supervisorSignalRegistry.mu.RUnlock()
		return supervisorSignalRegistry.handler
	}()
	SetSupervisorSignalHandler(func(signal SupervisorSignal) error {
		signals <- signal
		return nil
	})
	t.Cleanup(func() {
		SetSupervisorSignalHandler(prevHandler)
	})

	ref := registrarSupervisorLocalResidente(descriptorSupervisorLocal{
		Ref:      "test-supervisor-signals",
		Agente:   "Codex1",
		Proyecto: "orquestador",
		PID:      cmd.Process.Pid,
	}, cmd)
	if ref == "" {
		t.Fatal("faltaba ref supervisor")
	}
	obj := ObjetivoProceso{
		PID:          intPtr64(int64(cmd.Process.Pid)),
		HandleKind:   "process",
		HandleRef:    "process",
		MetadataJSON: `{"supervisor_ref":"` + ref + `","agente":"Codex1","proyecto":"orquestador"}`,
	}
	if err := ActivarSupervisionOrquestada(obj); err != nil {
		t.Fatalf("activar supervision: %v", err)
	}

	select {
	case signal := <-signals:
		if signal.Finalizado {
			t.Fatalf("la primera señal no debería ser final: %+v", signal)
		}
		if signal.Agente != "Codex1" || signal.Proyecto != "orquestador" {
			t.Fatalf("heartbeat supervisor inesperado: %+v", signal)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout esperando heartbeat inicial del supervisor")
	}

	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf("kill process: %v", err)
	}
	select {
	case signal := <-signals:
		if !signal.Finalizado {
			t.Fatalf("faltaba señal final del supervisor: %+v", signal)
		}
		if signal.Motivo != "process_exit" {
			t.Fatalf("motivo final inesperado: %+v", signal)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout esperando señal final del supervisor")
	}
}

func TestEmitSupervisorSignalRecuperaPanicDelHandler(t *testing.T) {
	prevHandler := func() SupervisorSignalHandler {
		supervisorSignalRegistry.mu.RLock()
		defer supervisorSignalRegistry.mu.RUnlock()
		return supervisorSignalRegistry.handler
	}()
	t.Cleanup(func() {
		SetSupervisorSignalHandler(prevHandler)
	})
	SetSupervisorSignalHandler(func(SupervisorSignal) error {
		panic("boom")
	})

	if err := emitSupervisorSignal(SupervisorSignal{
		Agente:   "Codex1",
		Proyecto: "orquestador",
	}); err == nil || !strings.Contains(err.Error(), "supervisor signal handler panic") {
		t.Fatalf("el handler panic deberia recuperarse como error, got=%v", err)
	}
}

func TestCommandIdentityHintsIncluyeFirmaUtil(t *testing.T) {
	hints := commandIdentityHints(
		"/usr/bin/script -qefc '/home/alberto/Trabajo/codex-perfiles/bin/codex-perfil Codex1' /tmp/trace.log",
		"/home/alberto/Trabajo/codex-perfiles/bin/codex-perfil Codex1",
	)
	joined := strings.Join(hints, "|")
	for _, want := range []string{"codex-perfil", "codex1", "script"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("faltaba hint %q en %q", want, joined)
		}
	}
}

func TestSamePathResuelveSymlink(t *testing.T) {
	tmp := t.TempDir()
	target := filepath.Join(tmp, "real")
	link := filepath.Join(tmp, "link")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	if !samePath(target, link) {
		t.Fatalf("samePath deberia considerar equivalentes %q y %q", target, link)
	}
}

func TestValidarIdentidadProcesoLocalDetectaMismatchDeCWD(t *testing.T) {
	ok, err := validarIdentidadProcesoLocal(os.Getpid(), t.TempDir(), "go test", "")
	if err != nil {
		t.Fatalf("validarIdentidadProcesoLocal: %v", err)
	}
	if ok {
		t.Fatal("un cwd incorrecto no deberia validar la identidad del proceso")
	}
}

func TestConsultarEstadoLocalRehidrataMetadataRicaDesdeRuntimeManifest(t *testing.T) {
	resetSupervisoresLocalesForTest(t)
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("executable: %v", err)
	}
	runDir := filepath.Join(workingDir, ".orquesta-runtime", "codex2", "20990101-000000-000000000")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(filepath.Join(workingDir, ".orquesta-runtime"))
	})
	manifestPath := filepath.Join(runDir, "runtime.json")
	payload := map[string]any{
		"agente":                "Codex2",
		"proyecto":              "orquestador",
		"pid":                   os.Getpid(),
		"driver":                "process_pty_cli",
		"stdin_path":            filepath.Join(runDir, "pty.stdin"),
		"stdin_raw_path":        filepath.Join(runDir, "pty.stdin.raw"),
		"log_path":              filepath.Join(runDir, "pty.log"),
		"working_dir":           workingDir,
		"rendered_command":      "codex-perfil Codex2",
		"wrapped_command":       exe,
		"mailbox_delivery_mode": "bootstrap_only",
		"can_send_input":        false,
		"supervisor_ref":        runDir,
		"created_at":            time.Now().UTC().Format(time.RFC3339Nano),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := os.WriteFile(manifestPath, append(data, '\n'), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	pid := int64(os.Getpid())
	estado, observed, err := ConsultarEstadoLocal(ObjetivoProceso{
		PID:          &pid,
		MetadataJSON: `{"cwd":"` + workingDir + `","trace_dir":"` + runDir + `","trace_manifest":"` + manifestPath + `"}`,
	})
	if err != nil {
		t.Fatalf("ConsultarEstadoLocal: %v", err)
	}
	if !observed || estado == nil || !estado.Vivo {
		t.Fatalf("ConsultarEstadoLocal deberia observar el proceso local: observed=%v estado=%+v", observed, estado)
	}
	meta := metadataMap(estado.MetadataJSON)
	if got := stringValueFromMetadata(meta, "stdin_path"); got != filepath.Join(runDir, "pty.stdin") {
		t.Fatalf("stdin_path no rehidratado: %q meta=%+v", got, meta)
	}
	if got := stringValueFromMetadata(meta, "supervisor_ref"); got != runDir {
		t.Fatalf("supervisor_ref no rehidratado: %q meta=%+v", got, meta)
	}
	if got := stringValueFromMetadata(meta, "rendered_command"); got != "codex-perfil Codex2" {
		t.Fatalf("rendered_command no rehidratado: %q meta=%+v", got, meta)
	}
}

func TestSupervisorLocalEstadoRehabilitaSalidaExternaSiPIDSigueVivo(t *testing.T) {
	now := time.Now().UTC()
	s := &supervisorProcesoLocal{
		ref:                 "rehabilita-test",
		agente:              "Codex3",
		proyecto:            "orquestador",
		pid:                 os.Getpid(),
		modo:                supervisionModoAdjunto,
		workingDir:          mustGetwdTest(t),
		renderedCommand:     "codex-perfil Codex3",
		wrappedCommand:      mustExecutableTest(t),
		traceDir:            mustGetwdTest(t),
		stdinPath:           filepath.Join(t.TempDir(), "pty.stdin"),
		mailboxDeliveryMode: "bootstrap_only",
		exitedAt:            ptrTime(now.Add(-time.Minute)),
		exitError:           "process identity mismatch",
	}

	estado := s.estado()
	if estado == nil {
		t.Fatal("faltaba estado")
	}
	if !estado.Vivo {
		t.Fatalf("el proceso actual deberia rehabilitarse como vivo: %+v", estado)
	}
	if estado.HandleEstado != "activo" {
		t.Fatalf("handle_state inesperado tras rehabilitarse: %+v", estado)
	}
	meta := metadataMap(estado.MetadataJSON)
	if got := stringValueFromMetadata(meta, "exited_at"); got != "" {
		t.Fatalf("no deberia conservar exited_at tras rehabilitarse: %q meta=%+v", got, meta)
	}
}

func mustGetwdTest(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return wd
}

func resetSupervisoresLocalesForTest(t *testing.T) {
	t.Helper()
	supervisoresLocales.mu.Lock()
	supervisoresLocales.byRef = map[string]*supervisorProcesoLocal{}
	supervisoresLocales.byPID = map[int]*supervisorProcesoLocal{}
	supervisoresLocales.mu.Unlock()
}

func mustExecutableTest(t *testing.T) string {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("executable: %v", err)
	}
	return exe
}

func ptrTime(v time.Time) *time.Time {
	return &v
}

func TestProcesoWrapperPTYSinHijoDetectaWrapperVacio(t *testing.T) {
	vacio, err := procesoWrapperPTYSinHijo(4242, "codex-perfil Codex1", "script -q -e -f -c 'codex-perfil Codex1' /tmp/pty.log", func(int64) (int, error) {
		return 0, nil
	})
	if err != nil {
		t.Fatalf("procesoWrapperPTYSinHijo: %v", err)
	}
	if !vacio {
		t.Fatal("wrapper PTY sin hijos deberia marcarse como vacio")
	}
}

func TestProcesoWrapperPTYSinHijoRespetaRuntimeConHijo(t *testing.T) {
	vacio, err := procesoWrapperPTYSinHijo(4242, "codex-perfil Codex1", "script -q -e -f -c 'codex-perfil Codex1' /tmp/pty.log", func(int64) (int, error) {
		return 1, nil
	})
	if err != nil {
		t.Fatalf("procesoWrapperPTYSinHijo: %v", err)
	}
	if vacio {
		t.Fatal("wrapper PTY con hijo real no deberia marcarse como vacio")
	}
}

func TestProcesoWrapperPTYSinHijoNoAfectaProcesosNormales(t *testing.T) {
	llamado := false
	vacio, err := procesoWrapperPTYSinHijo(4242, "sleep 30", "/usr/bin/sleep 30", func(int64) (int, error) {
		llamado = true
		return 0, nil
	})
	if err != nil {
		t.Fatalf("procesoWrapperPTYSinHijo: %v", err)
	}
	if vacio {
		t.Fatal("un proceso normal no deberia tratarse como wrapper PTY vacio")
	}
	if llamado {
		t.Fatal("no deberia consultar hijos para procesos que no son wrapper PTY")
	}
}
