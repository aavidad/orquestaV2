package db

import (
	"path/filepath"
	"strings"
	"testing"

	"orquesta/runtimeagente"
)

func TestBuildLaunchBootstrapPromptIncluyeRolOperativoAutonomo(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(t.TempDir(), "repo"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := UpsertProyectoAutonomia(&ProyectoAutonomia{
		ProyectoID:        proyectoID,
		Enabled:           true,
		SupervisorAgente:  "Codex1",
		ReserveSupervisor: true,
		EstadoAutonomia:   AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert autonomia: %v", err)
	}
	if err := ActivarAsignacion("Codex1", proyectoID, "supervision"); err != nil {
		t.Fatalf("activar asignacion Codex1: %v", err)
	}
	if err := ActivarAsignacion("Codex2", proyectoID, "worker"); err != nil {
		t.Fatalf("activar asignacion Codex2: %v", err)
	}

	proyecto, err := GetProyecto("orquestador")
	if err != nil {
		t.Fatalf("get proyecto: %v", err)
	}
	supervisor, err := GetAgente("Codex1")
	if err != nil {
		t.Fatalf("get agente supervisor: %v", err)
	}
	worker, err := GetAgente("Codex2")
	if err != nil {
		t.Fatalf("get agente worker: %v", err)
	}

	supervisorPrompt := BuildLaunchBootstrapPrompt(supervisor, proyecto, nil, nil, nil, nil, nil, "", "")
	if !strings.Contains(strings.ToLower(supervisorPrompt), "rol operativo en este proyecto: orquestador") {
		t.Fatalf("prompt supervisor sin rol operativo: %q", supervisorPrompt)
	}

	workerPrompt := BuildLaunchBootstrapPrompt(worker, proyecto, nil, nil, nil, nil, nil, "", "")
	if !strings.Contains(workerPrompt, "Supervisor operativo del proyecto: Codex1") {
		t.Fatalf("prompt worker sin supervisor operativo: %q", workerPrompt)
	}
}

func TestSeleccionarSupervisorAutonomiaOperativoUsaSupervisorConfigurado(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(t.TempDir(), "repo"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := UpsertProyectoAutonomia(&ProyectoAutonomia{
		ProyectoID:       proyectoID,
		Enabled:          true,
		SupervisorAgente: "Codex1",
		EstadoAutonomia:  AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert autonomia: %v", err)
	}

	supervisor, err := SeleccionarSupervisorAutonomiaOperativo(proyectoID, "")
	if err != nil {
		t.Fatalf("SeleccionarSupervisorAutonomiaOperativo: %v", err)
	}
	if supervisor == nil || supervisor.Nombre != "Codex1" {
		t.Fatalf("supervisor inesperado: %+v", supervisor)
	}
}

func TestBuildLaunchBootstrapPromptCompactaArranqueBootstrapOnly(t *testing.T) {
	prepararDBTemporal(t)

	proyecto := &Proyecto{
		ID:      1,
		Slug:    "orquestador",
		RutaAbs: filepath.Join(t.TempDir(), "repo"),
	}
	agente := &Agente{
		Nombre: "Codex7",
		Rol:    "programador",
	}
	canSendInput := false
	plan := &runtimeagente.LaunchPlan{
		Driver:              "cli",
		WorkingDir:          proyecto.RutaAbs,
		CanSendInput:        &canSendInput,
		MailboxDeliveryMode: runtimeagente.MailboxDeliveryBootstrapOnly,
	}
	tareas := []*Tarea{{ID: 483, Estado: TareaEnProgreso, Titulo: "Extraer trabajo no residente del núcleo"}}

	prompt := BuildLaunchBootstrapPrompt(agente, proyecto, plan, nil, nil, tareas, nil, "contexto muy largo que no debería entrar", "gobernanza muy larga")
	if strings.Contains(prompt, "Gobernanza efectiva:") || strings.Contains(prompt, "Reglas efectivas:") || strings.Contains(prompt, "Skills relevantes:") {
		t.Fatalf("prompt compacto no deberia incluir bloques largos:\n%s", prompt)
	}
	for _, token := range []string{"Bootstrap de Orquesta para Codex7.", "Rol: programador. Proyecto: orquestador.", "Tarea activa: #483 [en_progreso] Extraer trabajo no residente del núcleo."} {
		if !strings.Contains(prompt, token) {
			t.Fatalf("falta %q en prompt compacto:\n%s", token, prompt)
		}
	}
}

func TestBuildLaunchBootstrapPromptCompactaAgenteCLIOrquestadoAunqueNoSeaBootstrapOnly(t *testing.T) {
	prepararDBTemporal(t)

	proyecto := &Proyecto{
		ID:      1,
		Slug:    "orquestador",
		RutaAbs: filepath.Join(t.TempDir(), "repo"),
	}
	agente := &Agente{
		Nombre: "Codex3",
		Rol:    "programador",
	}
	canSendInput := true
	plan := &runtimeagente.LaunchPlan{
		Driver:              "cli",
		Comando:             "/tmp/codex-perfiles/bin/codex-perfil",
		Args:                []string{"Codex3"},
		WorkingDir:          proyecto.RutaAbs,
		CanSendInput:        &canSendInput,
		MailboxDeliveryMode: runtimeagente.MailboxDeliverySessionResume,
	}
	tareas := []*Tarea{{ID: 492, Estado: TareaEnProgreso, Titulo: "Adaptador Codex broker-first sin control por PTY"}}

	prompt := BuildLaunchBootstrapPrompt(agente, proyecto, plan, nil, nil, tareas, nil, "contexto muy largo que no debería entrar", "gobernanza muy larga")
	if strings.Contains(prompt, "Gobernanza efectiva:") || strings.Contains(prompt, "Reglas efectivas:") || strings.Contains(prompt, "Skills relevantes:") {
		t.Fatalf("prompt CLI orquestado deberia seguir siendo compacto:\n%s", prompt)
	}
	for _, token := range []string{
		"Trabaja solo dentro del alcance de la tarea activa y del mailbox actual.",
		"Si el contexto visible de la sesión no coincide con la tarea activa o el mailbox actual, ignóralo.",
		"No reabras frentes viejos ni reescribas módulos fuera del alcance inmediato.",
		"Tarea activa: #492 [en_progreso] Adaptador Codex broker-first sin control por PTY.",
	} {
		if !strings.Contains(prompt, token) {
			t.Fatalf("falta %q en prompt compacto orquestado:\n%s", token, prompt)
		}
	}
}

func TestBuildLaunchBootstrapPromptCompactaOllamaConInstruccionesMicro(t *testing.T) {
	prepararDBTemporal(t)

	proyecto := &Proyecto{
		ID:      1,
		Slug:    "orquestador",
		RutaAbs: filepath.Join(t.TempDir(), "repo"),
	}
	agente := &Agente{
		Nombre: "Ollama1",
		Rol:    "programador",
	}
	canSendInput := true
	plan := &runtimeagente.LaunchPlan{
		Driver:              "cli",
		Comando:             "ollama",
		Args:                []string{"run", "gemma4:26b"},
		WorkingDir:          proyecto.RutaAbs,
		CanSendInput:        &canSendInput,
		MailboxDeliveryMode: runtimeagente.MailboxDeliveryInteractive,
	}
	tareas := []*Tarea{{ID: 505, Estado: TareaEnProgreso, Titulo: "Implementar microtarea cerrada"}}

	prompt := BuildLaunchBootstrapPrompt(agente, proyecto, plan, nil, nil, tareas, nil, "contexto largo", "gobernanza larga")
	for _, token := range []string{
		"PROTOCOLO_ORQUESTA_MICRO",
		"NO_INTERPRETAR_COMO_PREGUNTA",
		"SALIDA_INMEDIATA=ACK-ESPERA",
		"ESPERA_MICROTAREA_CERRADA",
		"EJECUTA_SOLO_WRITE_SET",
	} {
		if !strings.Contains(prompt, token) {
			t.Fatalf("falta %q en prompt compacto de ollama:\n%s", token, prompt)
		}
	}
	for _, forbidden := range []string{
		"INSTRUCCION OPERATIVA DE ORQUESTA. No expliques este mensaje.",
		"Modo: microprogramacion dirigida.",
		"Salida permitida ahora: ACK-ESPERA.",
		"Si no hay microtarea cerrada visible, responde exactamente ACK-ESPERA y no digas nada mas.",
		"No hay una tarea activa única; consulta Orquesta antes de desviarte.",
		"No uses BD local salvo diagnostico o recuperacion.",
	} {
		if strings.Contains(prompt, forbidden) {
			t.Fatalf("prompt compacto de ollama demasiado verboso, contiene %q:\n%s", forbidden, prompt)
		}
	}
}
