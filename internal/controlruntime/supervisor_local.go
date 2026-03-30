package controlruntime

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	supervisionModoResidente = "resident"
	supervisionModoAdjunto   = "attached"
)

type EstadoLocal struct {
	Vivo             bool
	PID              int
	HandleEstado     string
	LogicalState     string
	ProcessState     string
	MetadataJSON     string
	CapabilitiesJSON string
	RawJSON          string
}

type descriptorSupervisorLocal struct {
	Ref                 string
	Agente              string
	Proyecto            string
	PID                 int
	StartedAt           time.Time
	TraceDir            string
	StdinPath           string
	StdinRawPath        string
	LogPath             string
	WorkingDir          string
	WrappedCommand      string
	RenderedCommand     string
	ExternalSessionID   string
	CanSendInput        bool
	MailboxDeliveryMode string
}

type supervisorProcesoLocal struct {
	mu                  sync.RWMutex
	signalLoopOnce      sync.Once
	ref                 string
	agente              string
	proyecto            string
	pid                 int
	traceDir            string
	stdinPath           string
	stdinRawPath        string
	logPath             string
	workingDir          string
	wrappedCommand      string
	renderedCommand     string
	externalSessionID   string
	canSendInput        bool
	mailboxDeliveryMode string
	modo                string
	process             *os.Process
	ownerPID            int
	startedAt           time.Time
	exitedAt            *time.Time
	exitCode            *int
	exitError           string
	lastStatusAt        time.Time
}

type registroSupervisoresLocales struct {
	mu    sync.RWMutex
	byRef map[string]*supervisorProcesoLocal
	byPID map[int]*supervisorProcesoLocal
}

var supervisoresLocales = &registroSupervisoresLocales{
	byRef: map[string]*supervisorProcesoLocal{},
	byPID: map[int]*supervisorProcesoLocal{},
}

type SupervisorSignal struct {
	Agente            string
	Proyecto          string
	Host              string
	PID               int
	ExternalSessionID string
	Finalizado        bool
	Motivo            string
}

type SupervisorSignalHandler func(SupervisorSignal) error

var supervisorSignalRegistry = struct {
	mu      sync.RWMutex
	handler SupervisorSignalHandler
}{}

func SetSupervisorSignalHandler(handler SupervisorSignalHandler) {
	supervisorSignalRegistry.mu.Lock()
	defer supervisorSignalRegistry.mu.Unlock()
	supervisorSignalRegistry.handler = handler
}

func emitSupervisorSignal(signal SupervisorSignal) error {
	supervisorSignalRegistry.mu.RLock()
	handler := supervisorSignalRegistry.handler
	supervisorSignalRegistry.mu.RUnlock()
	if handler == nil {
		return nil
	}
	return handler(signal)
}

func registrarSupervisorLocalResidente(desc descriptorSupervisorLocal, cmd *exec.Cmd) string {
	if cmd == nil || cmd.Process == nil {
		return ""
	}
	desc.PID = cmd.Process.Pid
	if strings.TrimSpace(desc.Ref) == "" {
		desc.Ref = construirRefSupervisorLocal(desc)
	}
	supervisor := supervisoresLocales.registrar(desc, supervisionModoResidente, cmd.Process)
	go supervisor.esperar(cmd)
	go supervisor.sondearExternalSessionID()
	return supervisor.ref
}

func ConsultarEstadoLocal(obj ObjetivoProceso) (*EstadoLocal, bool, error) {
	supervisor, observed, err := supervisoresLocales.resolver(obj)
	if err != nil || !observed {
		return nil, observed, err
	}
	if supervisor == nil {
		return nil, true, nil
	}
	return supervisor.estado(), true, nil
}

func ActivarSupervisionOrquestada(obj ObjetivoProceso) error {
	supervisor, observed, err := supervisoresLocales.resolver(obj)
	if err != nil || !observed || supervisor == nil {
		return err
	}
	supervisor.activarSignalLoop()
	return nil
}

func controlarProcesoLocalSupervisado(obj ObjetivoProceso, sig syscall.Signal, accion string) (bool, int, bool, error) {
	supervisor, observed, err := supervisoresLocales.resolver(obj)
	if err != nil || !observed {
		return false, 0, observed, err
	}
	if supervisor == nil {
		return false, 0, true, nil
	}
	return supervisor.controlar(sig, accion)
}

func construirRefSupervisorLocal(desc descriptorSupervisorLocal) string {
	switch {
	case strings.TrimSpace(desc.TraceDir) != "":
		return strings.TrimSpace(desc.TraceDir)
	case strings.TrimSpace(desc.StdinPath) != "":
		return strings.TrimSpace(desc.StdinPath)
	case desc.PID > 0:
		return fmt.Sprintf("pid:%d", desc.PID)
	default:
		return ""
	}
}

func supervisorLocalRefDesdeMetadata(raw string) string {
	return stringValueFromMetadata(metadataMap(raw), "supervisor_ref")
}

func supervisorLocalEsAplicable(obj ObjetivoProceso) bool {
	if _, ok, _ := ResolverPID(obj); ok {
		return true
	}
	if strings.TrimSpace(obj.HandleKind) == "process" {
		return true
	}
	meta := metadataMap(obj.MetadataJSON)
	if strings.EqualFold(strings.TrimSpace(stringValueFromMetadata(meta, "driver")), "process_pty_cli") {
		return true
	}
	return strings.TrimSpace(supervisorLocalRefDesdeMetadata(obj.MetadataJSON)) != ""
}

func descriptorSupervisorLocalDesdeObjetivo(obj ObjetivoProceso, pid int) descriptorSupervisorLocal {
	meta := metadataMap(obj.MetadataJSON)
	return descriptorSupervisorLocal{
		Ref:                 strings.TrimSpace(supervisorLocalRefDesdeMetadata(obj.MetadataJSON)),
		Agente:              stringValueFromMetadata(meta, "agente"),
		Proyecto:            stringValueFromMetadata(meta, "proyecto"),
		PID:                 pid,
		StartedAt:           metadataTime(meta, "started_at"),
		TraceDir:            stringValueFromMetadata(meta, "trace_dir"),
		StdinPath:           stringValueFromMetadata(meta, "stdin_path"),
		StdinRawPath:        stringValueFromMetadata(meta, "stdin_raw_path"),
		LogPath:             stringValueFromMetadata(meta, "log_path"),
		WorkingDir:          stringValueFromMetadata(meta, "working_dir"),
		WrappedCommand:      stringValueFromMetadata(meta, "wrapped_command"),
		RenderedCommand:     stringValueFromMetadata(meta, "rendered_command"),
		ExternalSessionID:   stringValueFromMetadata(meta, "external_session_id"),
		CanSendInput:        !metaBoolDefinedAndFalse(meta, "can_send_input"),
		MailboxDeliveryMode: stringValueFromMetadata(meta, "mailbox_delivery_mode"),
	}
}

func metaBoolDefinedAndFalse(meta map[string]any, key string) bool {
	if meta == nil {
		return false
	}
	v, ok := meta[key]
	if !ok {
		return false
	}
	b, ok := v.(bool)
	return ok && !b
}

func (r *registroSupervisoresLocales) resolver(obj ObjetivoProceso) (*supervisorProcesoLocal, bool, error) {
	if !supervisorLocalEsAplicable(obj) {
		return nil, false, nil
	}
	pid, ok, err := ResolverPID(obj)
	if err != nil {
		return nil, true, err
	}
	desc := descriptorSupervisorLocal{}
	if ok && pid > 0 {
		desc = descriptorSupervisorLocalDesdeObjetivo(obj, pid)
		if strings.TrimSpace(desc.Ref) == "" {
			desc.Ref = construirRefSupervisorLocal(desc)
		}
	}

	ref := strings.TrimSpace(supervisorLocalRefDesdeMetadata(obj.MetadataJSON))
	r.mu.RLock()
	if ref != "" {
		if supervisor := r.byRef[ref]; supervisor != nil {
			r.mu.RUnlock()
			if ok && pid > 0 {
				supervisor.actualizar(desc, supervisionModoAdjunto, nil)
			}
			return supervisor, true, nil
		}
	}
	if ok && pid > 0 {
		if supervisor := r.byPID[pid]; supervisor != nil {
			r.mu.RUnlock()
			supervisor.actualizar(desc, supervisionModoAdjunto, nil)
			return supervisor, true, nil
		}
	}
	r.mu.RUnlock()

	if !ok || pid <= 0 {
		return nil, true, nil
	}
	return r.registrar(desc, supervisionModoAdjunto, nil), true, nil
}

func (r *registroSupervisoresLocales) registrar(desc descriptorSupervisorLocal, modo string, proc *os.Process) *supervisorProcesoLocal {
	ref := strings.TrimSpace(desc.Ref)
	if ref == "" {
		ref = construirRefSupervisorLocal(desc)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if ref != "" {
		if existente := r.byRef[ref]; existente != nil {
			existente.actualizar(desc, modo, proc)
			if existente.pid > 0 {
				r.byPID[existente.pid] = existente
			}
			return existente
		}
	}
	if desc.PID > 0 {
		if existente := r.byPID[desc.PID]; existente != nil {
			existente.actualizar(desc, modo, proc)
			if ref != "" {
				r.byRef[ref] = existente
			}
			return existente
		}
	}

	supervisor := &supervisorProcesoLocal{
		ref:                 ref,
		agente:              strings.TrimSpace(desc.Agente),
		proyecto:            strings.TrimSpace(desc.Proyecto),
		pid:                 desc.PID,
		startedAt:           desc.StartedAt,
		traceDir:            strings.TrimSpace(desc.TraceDir),
		stdinPath:           strings.TrimSpace(desc.StdinPath),
		stdinRawPath:        strings.TrimSpace(desc.StdinRawPath),
		logPath:             strings.TrimSpace(desc.LogPath),
		workingDir:          strings.TrimSpace(desc.WorkingDir),
		wrappedCommand:      strings.TrimSpace(desc.WrappedCommand),
		renderedCommand:     strings.TrimSpace(desc.RenderedCommand),
		externalSessionID:   strings.TrimSpace(desc.ExternalSessionID),
		canSendInput:        desc.CanSendInput,
		mailboxDeliveryMode: strings.TrimSpace(desc.MailboxDeliveryMode),
		modo:                strings.TrimSpace(modo),
		process:             proc,
		ownerPID:            os.Getpid(),
	}
	if ref != "" {
		r.byRef[ref] = supervisor
	}
	if desc.PID > 0 {
		r.byPID[desc.PID] = supervisor
	}
	return supervisor
}

func (s *supervisorProcesoLocal) actualizar(desc descriptorSupervisorLocal, modo string, proc *os.Process) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(desc.Ref) != "" {
		s.ref = strings.TrimSpace(desc.Ref)
	}
	if strings.TrimSpace(desc.Agente) != "" {
		s.agente = strings.TrimSpace(desc.Agente)
	}
	if strings.TrimSpace(desc.Proyecto) != "" {
		s.proyecto = strings.TrimSpace(desc.Proyecto)
	}
	if desc.PID > 0 {
		s.pid = desc.PID
	}
	if !desc.StartedAt.IsZero() {
		s.startedAt = desc.StartedAt
	}
	if strings.TrimSpace(desc.TraceDir) != "" {
		s.traceDir = strings.TrimSpace(desc.TraceDir)
	}
	if strings.TrimSpace(desc.StdinPath) != "" {
		s.stdinPath = strings.TrimSpace(desc.StdinPath)
	}
	if strings.TrimSpace(desc.StdinRawPath) != "" {
		s.stdinRawPath = strings.TrimSpace(desc.StdinRawPath)
	}
	if strings.TrimSpace(desc.LogPath) != "" {
		s.logPath = strings.TrimSpace(desc.LogPath)
	}
	if strings.TrimSpace(desc.WorkingDir) != "" {
		s.workingDir = strings.TrimSpace(desc.WorkingDir)
	}
	if strings.TrimSpace(desc.WrappedCommand) != "" {
		s.wrappedCommand = strings.TrimSpace(desc.WrappedCommand)
	}
	if strings.TrimSpace(desc.RenderedCommand) != "" {
		s.renderedCommand = strings.TrimSpace(desc.RenderedCommand)
	}
	if strings.TrimSpace(desc.ExternalSessionID) != "" {
		s.externalSessionID = strings.TrimSpace(desc.ExternalSessionID)
	}
	if strings.TrimSpace(desc.MailboxDeliveryMode) != "" {
		s.mailboxDeliveryMode = strings.TrimSpace(desc.MailboxDeliveryMode)
	}
	s.canSendInput = desc.CanSendInput
	if strings.TrimSpace(modo) != "" {
		s.modo = strings.TrimSpace(modo)
	}
	if proc != nil {
		s.process = proc
		s.ownerPID = os.Getpid()
	}
}

func (s *supervisorProcesoLocal) esperar(cmd *exec.Cmd) {
	err := cmd.Wait()
	now := time.Now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.exitedAt == nil {
		s.exitedAt = &now
	}
	s.lastStatusAt = now
	s.exitError = ""
	s.exitCode = nil
	if err != nil {
		s.exitError = strings.TrimSpace(err.Error())
		if exitErr, ok := err.(*exec.ExitError); ok {
			signalCode := exitErr.ExitCode()
			s.exitCode = &signalCode
		}
	} else {
		successCode := 0
		s.exitCode = &successCode
	}
	go s.emitirSignal(true, "process_exit")
}

func (s *supervisorProcesoLocal) sondearExternalSessionID() {
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if !s.debeSondearExternalSessionID() {
			return
		}
		_, _, _, _, _, _ = s.snapshot()
		s.mu.RLock()
		externalSessionID := strings.TrimSpace(s.externalSessionID)
		s.mu.RUnlock()
		if externalSessionID != "" {
			return
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func (s *supervisorProcesoLocal) debeSondearExternalSessionID() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.exitedAt != nil {
		return false
	}
	if strings.TrimSpace(s.externalSessionID) != "" {
		return false
	}
	return renderedCommandLooksLikeCodexCLI(s.renderedCommand) && strings.TrimSpace(s.workingDir) != ""
}

func (s *supervisorProcesoLocal) estado() *EstadoLocal {
	pid, modo, meta, caps, alive, err := s.snapshot()

	handleEstado := "activo"
	logicalState := "disponible"
	processState := "running"
	if !alive {
		handleEstado = "cerrado"
		logicalState = "degradado"
		processState = "finalizado"
		if err != nil {
			handleEstado = "fallido"
			processState = "missing"
		}
	}
	payload := map[string]any{
		"observed_by":       "local_runtime_supervisor",
		"supervisor_ref":    meta["supervisor_ref"],
		"supervision_mode":  modo,
		"pid":               pid,
		"alive":             alive,
		"handle_state":      handleEstado,
		"logical_state":     logicalState,
		"process_state":     processState,
		"metadata":          meta,
		"capabilities":      caps,
		"last_status_error": "",
	}
	if err != nil {
		payload["last_status_error"] = strings.TrimSpace(err.Error())
	}
	rawJSON, _ := json.Marshal(payload)
	metaJSON, _ := json.Marshal(meta)
	capsJSON, _ := json.Marshal(caps)
	return &EstadoLocal{
		Vivo:             alive,
		PID:              pid,
		HandleEstado:     handleEstado,
		LogicalState:     logicalState,
		ProcessState:     processState,
		MetadataJSON:     string(metaJSON),
		CapabilitiesJSON: string(capsJSON),
		RawJSON:          string(rawJSON),
	}
}

func (s *supervisorProcesoLocal) snapshot() (int, string, map[string]any, map[string]any, bool, error) {
	s.mu.RLock()
	agente := s.agente
	proyecto := s.proyecto
	pid := s.pid
	modo := s.modo
	startedAt := s.startedAt
	traceDir := s.traceDir
	stdinPath := s.stdinPath
	stdinRawPath := s.stdinRawPath
	logPath := s.logPath
	workingDir := s.workingDir
	wrappedCommand := s.wrappedCommand
	renderedCommand := s.renderedCommand
	externalSessionID := s.externalSessionID
	canSendInput := s.canSendInput
	mailboxDeliveryMode := s.mailboxDeliveryMode
	ownerPID := s.ownerPID
	exitedAt := s.exitedAt
	exitCode := s.exitCode
	exitError := s.exitError
	lastStatusAt := s.lastStatusAt
	s.mu.RUnlock()

	alive := false
	var aliveErr error
	if exitedAt == nil && pid > 0 {
		alive, aliveErr = procesoVivoPID(pid)
		if alive {
			if ok, err := validarIdentidadProcesoLocal(pid, workingDir, renderedCommand, wrappedCommand); err != nil {
				alive = false
				aliveErr = err
			} else if !ok {
				alive = false
				aliveErr = fmt.Errorf("process identity mismatch")
			}
		}
		if !alive {
			s.marcarSalidaExterna(aliveErr)
		}
	} else if exitedAt == nil && pid <= 0 {
		aliveErr = fmt.Errorf("runtime local sin pid")
	} else {
		aliveErr = nil
	}

	if externalSessionID == "" && renderedCommandLooksLikeCodexCLI(renderedCommand) {
		if detected, err := detectCodexSessionID(renderedCommand, workingDir, startedAt, time.Now().UTC()); err == nil {
			externalSessionID = strings.TrimSpace(detected)
			if externalSessionID != "" {
				s.mu.Lock()
				if s.externalSessionID == "" {
					s.externalSessionID = externalSessionID
				}
				s.mu.Unlock()
			}
		}
	}

	meta := map[string]any{
		"supervisor_driver":     "local_runtime_supervisor",
		"supervisor_ref":        strings.TrimSpace(s.ref),
		"agente":                strings.TrimSpace(agente),
		"proyecto":              strings.TrimSpace(proyecto),
		"supervision_mode":      strings.TrimSpace(modo),
		"supervisor_owner_pid":  ownerPID,
		"trace_dir":             traceDir,
		"stdin_path":            stdinPath,
		"stdin_raw_path":        stdinRawPath,
		"log_path":              logPath,
		"working_dir":           workingDir,
		"wrapped_command":       wrappedCommand,
		"rendered_command":      renderedCommand,
		"external_session_id":   externalSessionID,
		"mailbox_delivery_mode": mailboxDeliveryMode,
		"can_send_input":        canSendInput,
	}
	if !startedAt.IsZero() {
		meta["started_at"] = startedAt.Format(time.RFC3339Nano)
	}
	if !lastStatusAt.IsZero() {
		meta["supervisor_last_status_at"] = lastStatusAt.Format(time.RFC3339Nano)
	}
	if exitedAt != nil {
		meta["exited_at"] = exitedAt.UTC().Format(time.RFC3339Nano)
	}
	if exitCode != nil {
		meta["exit_code"] = *exitCode
	}
	if strings.TrimSpace(exitError) != "" {
		meta["exit_error"] = strings.TrimSpace(exitError)
	}
	caps := map[string]any{
		"can_send_input":        canSendInput,
		"can_checkpoint":        true,
		"can_resume":            true,
		"can_capture_pid":       pid > 0,
		"can_track_continuity":  true,
		"can_pause":             pid > 0,
		"can_stop":              pid > 0,
		"mailbox_delivery_mode": mailboxDeliveryMode,
	}
	return pid, modo, meta, caps, alive, aliveErr
}

func (s *supervisorProcesoLocal) activarSignalLoop() {
	s.signalLoopOnce.Do(func() {
		go s.signalLoop()
	})
}

func (s *supervisorProcesoLocal) signalLoop() {
	s.emitirSignal(false, "runtime_supervisor_start")
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		estado := s.estado()
		if estado == nil || !estado.Vivo || estado.PID <= 0 {
			return
		}
		s.emitirSignal(false, "runtime_supervisor_heartbeat")
	}
}

func (s *supervisorProcesoLocal) emitirSignal(finalizado bool, motivo string) {
	estado := s.estado()
	if estado == nil || estado.PID <= 0 {
		return
	}
	meta := metadataMap(estado.MetadataJSON)
	agente := strings.TrimSpace(stringValueFromMetadata(meta, "agente"))
	proyecto := strings.TrimSpace(stringValueFromMetadata(meta, "proyecto"))
	if agente == "" || proyecto == "" {
		return
	}
	host, _ := os.Hostname()
	_ = emitSupervisorSignal(SupervisorSignal{
		Agente:            agente,
		Proyecto:          proyecto,
		Host:              strings.TrimSpace(host),
		PID:               estado.PID,
		ExternalSessionID: strings.TrimSpace(stringValueFromMetadata(meta, "external_session_id")),
		Finalizado:        finalizado,
		Motivo:            strings.TrimSpace(motivo),
	})
}

func (s *supervisorProcesoLocal) marcarSalidaExterna(aliveErr error) {
	now := time.Now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.exitedAt == nil {
		s.exitedAt = &now
	}
	s.lastStatusAt = now
	if aliveErr != nil {
		s.exitError = strings.TrimSpace(aliveErr.Error())
	}
}

func (s *supervisorProcesoLocal) controlar(sig syscall.Signal, accion string) (bool, int, bool, error) {
	estado := s.estado()
	if estado == nil {
		return false, 0, true, nil
	}
	if !estado.Vivo || estado.PID <= 0 {
		return false, estado.PID, true, nil
	}
	proc := s.proceso()
	if proc == nil {
		var err error
		proc, err = os.FindProcess(estado.PID)
		if err != nil {
			return true, estado.PID, true, err
		}
	}
	if err := proc.Signal(sig); err != nil {
		return true, estado.PID, true, err
	}

	s.mu.Lock()
	s.lastStatusAt = time.Now().UTC()
	s.mu.Unlock()

	return true, estado.PID, true, nil
}

func (s *supervisorProcesoLocal) proceso() *os.Process {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.process
}

func procesoVivoPID(pid int) (bool, error) {
	if pid <= 0 {
		return false, nil
	}
	err := syscall.Kill(pid, 0)
	switch {
	case err == nil:
		return true, nil
	case err == syscall.EPERM:
		return true, nil
	case err == syscall.ESRCH:
		return false, nil
	default:
		return false, err
	}
}

func validarIdentidadProcesoLocal(pid int, workingDir, renderedCommand, wrappedCommand string) (bool, error) {
	if pid <= 0 {
		return false, nil
	}
	cwd := strings.TrimSpace(workingDir)
	if cwd != "" {
		actual, err := os.Readlink(fmt.Sprintf("/proc/%d/cwd", pid))
		if err != nil {
			return false, err
		}
		if !samePath(actual, cwd) {
			return false, nil
		}
	}
	expected := commandIdentityHints(renderedCommand, wrappedCommand)
	if len(expected) == 0 {
		return true, nil
	}
	cmdlineRaw, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return false, err
	}
	cmdline := strings.ToLower(strings.TrimSpace(strings.ReplaceAll(string(cmdlineRaw), "\x00", " ")))
	if cmdline == "" {
		return false, nil
	}
	for _, hint := range expected {
		if strings.Contains(cmdline, hint) {
			return true, nil
		}
	}
	return false, nil
}

func commandIdentityHints(renderedCommand, wrappedCommand string) []string {
	seen := map[string]struct{}{}
	add := func(raw string) {
		raw = strings.ToLower(strings.TrimSpace(raw))
		if raw == "" {
			return
		}
		seen[raw] = struct{}{}
	}
	addCommand := func(raw string) {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return
		}
		add(raw)
		for _, token := range strings.Fields(raw) {
			base := strings.ToLower(strings.TrimSpace(filepath.Base(token)))
			if base != "" {
				add(base)
			}
		}
	}
	addCommand(renderedCommand)
	addCommand(wrappedCommand)
	out := make([]string, 0, len(seen))
	for hint := range seen {
		out = append(out, hint)
	}
	return out
}

func samePath(a, b string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a == "" || b == "" {
		return a == b
	}
	if a == b {
		return true
	}
	if resolvedA, err := filepath.EvalSymlinks(a); err == nil {
		a = resolvedA
	}
	if resolvedB, err := filepath.EvalSymlinks(b); err == nil {
		b = resolvedB
	}
	return a == b
}
