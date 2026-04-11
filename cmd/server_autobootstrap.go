package cmd

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"orquesta/db"
	"orquesta/supervisionapp"
)

type serverAutobootstrapConfig struct {
	Enabled              bool
	ProjectSlug          string
	ProjectName          string
	ProjectPath          string
	SupervisorAgent      string
	WorkerAgents         []string
	ObjetivoGeneral      string
	DefinitionOfDoneJSON string
}

func bootstrapServerAutonomy() error {
	cfg := loadServerAutobootstrapConfig()
	if !cfg.Enabled {
		return nil
	}
	proyecto, err := ensureAutobootstrapProject(cfg)
	if err != nil || proyecto == nil {
		return err
	}
	if err := ensureAutobootstrapAgents(proyecto.ID, cfg); err != nil {
		return err
	}
	policy, err := supervisionService.UpsertProjectPolicy(proyecto.Slug, supervisionapp.PolicyInput{
		Enabled:              true,
		ObjetivoGeneral:      cfg.ObjetivoGeneral,
		DefinitionOfDoneJSON: cfg.DefinitionOfDoneJSON,
		MaxWorkers:           len(cfg.WorkerAgents),
		SupervisorAgente:     cfg.SupervisorAgent,
		ReserveSupervisor:    true,
		ReviewRequired:       false,
		AutoCreateTasks:      true,
		AutoCloseProject:     true,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	})
	if err != nil {
		return err
	}
	supervisor, _, err := resolverSupervisorAutonomiaOperativo(proyecto.ID, policy, "server_autobootstrap_supervisor", "")
	if err != nil {
		return err
	}
	if supervisor != nil {
		if _, err := asegurarTrabajoAutonomia(policy, proyecto, supervisor); err != nil {
			return err
		}
	}
	if err := db.PlanificarTareasAutomaticamente(); err != nil {
		return err
	}
	if supervisor != nil {
		if listo, err := agenteYaBootstrappeadoServidor(supervisor.Nombre, proyecto.ID); err != nil {
			return err
		} else if !listo {
			if _, err := encolarNudgeAutonomiaDetallado(supervisor.Nombre, proyecto, "supervisar_proyecto", "arranque_autonomo_servidor", construirInstruccionBootstrapSupervisor(policy, proyecto, supervisor.Nombre, cfg.WorkerAgents), map[string]any{
				"bootstrap":               true,
				"bootstrap_kind":          "server_autobootstrap",
				"objetivo_general":        strings.TrimSpace(policy.ObjetivoGeneral),
				"definition_of_done_json": strings.TrimSpace(policy.DefinitionOfDoneJSON),
			}); err != nil {
				return err
			}
		}
	}
	for _, worker := range cfg.WorkerAgents {
		worker = strings.TrimSpace(worker)
		if worker == "" {
			continue
		}
		if _, _, err := asegurarAgenteAutonomiaOperativo(proyecto.ID, worker, "server_autobootstrap_worker"); err != nil {
			return err
		}
		if supervisor != nil && strings.EqualFold(worker, strings.TrimSpace(supervisor.Nombre)) {
			continue
		}
		if listo, err := agenteYaBootstrappeadoServidor(worker, proyecto.ID); err != nil {
			return err
		} else if !listo {
			if _, err := encolarNudgeAutonomiaDetallado(worker, proyecto, "esperar_o_pedir_tarea", "arranque_autonomo_servidor", construirInstruccionBootstrapWorker(policy, proyecto, supervisorNameOrDefault(supervisor, cfg.SupervisorAgent)), map[string]any{
				"bootstrap":      true,
				"bootstrap_kind": "server_autobootstrap",
			}); err != nil {
				return err
			}
		}
	}
	db.Audit("orquesta", "server_autobootstrap", "proyecto", proyecto.ID, fmt.Sprintf("slug=%s supervisor=%s workers=%d", proyecto.Slug, supervisorNameOrDefault(supervisor, cfg.SupervisorAgent), len(cfg.WorkerAgents)))
	return nil
}

func agenteYaBootstrappeadoServidor(agente string, proyectoID int64) (bool, error) {
	agente = strings.TrimSpace(agente)
	if agente == "" || proyectoID <= 0 {
		return false, nil
	}
	if sesion, err := db.GetSesionActiva(agente, &proyectoID); err != nil && err != sql.ErrNoRows {
		return false, err
	} else if sesion != nil {
		return true, nil
	}
	if handle, err := db.GetRuntimeHandleOperativoRecienteAgenteProyecto(agente, &proyectoID); err != nil {
		return false, err
	} else if handle != nil {
		return true, nil
	}
	estadoPendiente := "pendiente"
	mailbox, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{
		ToAgente:   &agente,
		ProyectoID: &proyectoID,
		Estado:     &estadoPendiente,
	})
	if err != nil {
		return false, err
	}
	for _, msg := range mailbox {
		if runtimeMailboxEsBootstrapServidor(msg) {
			return true, nil
		}
	}
	return false, nil
}

func runtimeMailboxEsBootstrapServidor(msg *db.RuntimeMailboxMessage) bool {
	if msg == nil || strings.TrimSpace(msg.Kind) != "autonomia" {
		return false
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(msg.PayloadJSON)), &payload); err != nil || payload == nil {
		return false
	}
	kind := strings.TrimSpace(stringMapValue(payload, "bootstrap_kind"))
	if kind != "server_autobootstrap" {
		return false
	}
	bootstrap, _ := payload["bootstrap"].(bool)
	return bootstrap
}

func loadServerAutobootstrapConfig() serverAutobootstrapConfig {
	cfg := serverAutobootstrapConfig{
		Enabled:              controlPlaneConfigBoolOrDefault("server_autobootstrap_enabled", true),
		ProjectSlug:          strings.TrimSpace(configOrDefault("server_autobootstrap_project_slug", "orquestador")),
		ProjectName:          strings.TrimSpace(configOrDefault("server_autobootstrap_project_name", "Orquestador")),
		ProjectPath:          strings.TrimSpace(configOrDefault("server_autobootstrap_project_path", "")),
		SupervisorAgent:      strings.TrimSpace(configOrDefault("server_autobootstrap_supervisor_agent", "Codex1")),
		WorkerAgents:         splitServerAutobootstrapAgents(configOrDefault("server_autobootstrap_worker_agents", "Codex2,Codex3,Codex4,Codex5")),
		ObjetivoGeneral:      strings.TrimSpace(configOrDefault("server_autobootstrap_objective_general", "Terminar la app al completo, revisando el codigo real, reparando fallos de raiz y validando con pruebas reales.")),
		DefinitionOfDoneJSON: strings.TrimSpace(configOrDefault("server_autobootstrap_definition_of_done_json", `{"estado":"app_completa","criterios":["codigo_real_y_funcional","sin_humo","pruebas_reales_en_verde","frentes_cerrados"]}`)),
	}
	if cfg.ProjectSlug == "" {
		cfg.ProjectSlug = "orquestador"
	}
	if cfg.ProjectName == "" {
		cfg.ProjectName = "Orquestador"
	}
	if cfg.ProjectPath == "" {
		if cwd, err := os.Getwd(); err == nil {
			cfg.ProjectPath = cwd
		}
	}
	if cfg.SupervisorAgent == "" {
		cfg.SupervisorAgent = "Codex1"
	}
	if len(cfg.WorkerAgents) == 0 {
		cfg.WorkerAgents = []string{"Codex2", "Codex3", "Codex4", "Codex5"}
	}
	return cfg
}

func splitServerAutobootstrapAgents(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n'
	})
	seen := map[string]struct{}{}
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key := strings.ToLower(part)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, part)
	}
	return out
}

func ensureAutobootstrapProject(cfg serverAutobootstrapConfig) (*db.Proyecto, error) {
	if proyecto, err := db.GetProyecto(cfg.ProjectSlug); err == nil && proyecto != nil {
		return proyecto, nil
	}
	ruta := strings.TrimSpace(cfg.ProjectPath)
	if ruta == "" {
		if cwd, err := os.Getwd(); err == nil {
			ruta = cwd
		}
	}
	if ruta != "" {
		ruta = filepath.Clean(ruta)
	}
	id, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    cfg.ProjectSlug,
		Nombre:  cfg.ProjectName,
		RutaAbs: ruta,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		return nil, err
	}
	return db.GetProyecto(fmt.Sprintf("%d", id))
}

func ensureAutobootstrapAgents(proyectoID int64, cfg serverAutobootstrapConfig) error {
	nombres := append([]string{cfg.SupervisorAgent}, cfg.WorkerAgents...)
	for _, nombre := range nombres {
		nombre = strings.TrimSpace(nombre)
		if nombre == "" {
			continue
		}
		if err := db.RegistrarAgente(nombre, "programador"); err != nil {
			return err
		}
		if err := db.ActivarAsignacion(nombre, proyectoID, "server_autobootstrap"); err != nil {
			return err
		}
	}
	return nil
}

func construirInstruccionBootstrapSupervisor(policy *db.ProyectoAutonomia, proyecto *db.Proyecto, supervisor string, workers []string) string {
	parts := []string{
		"Orquesta: has sido arrancado como orquestador autónomo del proyecto.",
		"Coordina a los demás Codex desde dentro de Orquesta, reparte trabajo real, revisa pruebas, deduplica frentes y no pidas intervención humana salvo que falten credenciales, secretos o un recurso externo real.",
		"Tu responsabilidad es terminar la app completa con código real, funcional y verificado.",
	}
	if proyecto != nil {
		parts = append(parts, "Proyecto: "+strings.TrimSpace(proyecto.Slug)+".")
	}
	if strings.TrimSpace(supervisor) != "" {
		parts = append(parts, "Supervisor operativo actual: "+strings.TrimSpace(supervisor)+".")
	}
	if len(workers) > 0 {
		parts = append(parts, "Workers disponibles: "+strings.Join(workers, ", ")+".")
	}
	if policy != nil && strings.TrimSpace(policy.ObjetivoGeneral) != "" {
		parts = append(parts, "Objetivo general: "+strings.TrimSpace(policy.ObjetivoGeneral)+".")
	}
	if policy != nil && strings.TrimSpace(policy.DefinitionOfDoneJSON) != "" && strings.TrimSpace(policy.DefinitionOfDoneJSON) != "{}" {
		parts = append(parts, "Definition of done JSON: "+strings.TrimSpace(policy.DefinitionOfDoneJSON)+".")
	}
	return strings.Join(parts, " ")
}

func construirInstruccionBootstrapWorker(policy *db.ProyectoAutonomia, proyecto *db.Proyecto, supervisor string) string {
	parts := []string{
		"Orquesta: has sido arrancado como programador del proyecto.",
		"No te quedes esperando: revisa Orquesta, coge el siguiente frente útil real, programa, prueba y deja evidencia.",
	}
	if proyecto != nil {
		parts = append(parts, "Proyecto: "+strings.TrimSpace(proyecto.Slug)+".")
	}
	if strings.TrimSpace(supervisor) != "" {
		parts = append(parts, "Orquestador operativo: "+strings.TrimSpace(supervisor)+".")
	}
	if policy != nil && strings.TrimSpace(policy.ObjetivoGeneral) != "" {
		parts = append(parts, "Objetivo general: "+strings.TrimSpace(policy.ObjetivoGeneral)+".")
	}
	return strings.Join(parts, " ")
}

func supervisorNameOrDefault(supervisor *db.Agente, fallback string) string {
	if supervisor != nil && strings.TrimSpace(supervisor.Nombre) != "" {
		return strings.TrimSpace(supervisor.Nombre)
	}
	return strings.TrimSpace(fallback)
}
