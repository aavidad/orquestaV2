package lanzamientoruntime

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"orquesta/coordinacion"
	"orquesta/db"
	"orquesta/gitoperaciones"
	"orquesta/internal/bootstrapruntime"
	"orquesta/runtimeagente"
)

type Preparacion struct {
	Agente    *db.Agente
	Proyecto  *db.Proyecto
	Conector  *db.Conector
	Ultima    *db.Sesion
	Bootstrap *bootstrapruntime.State
	Plan      *runtimeagente.LaunchPlan
}

func PrepararDesdeRefs(agenteRef, proyectoRef, conectorRef, modelo, razonamiento, perfilTarea string) (*Preparacion, error) {
	agente, err := db.GetAgente(strings.TrimSpace(agenteRef))
	if err != nil {
		return nil, err
	}
	proyecto, err := db.GetProyecto(strings.TrimSpace(proyectoRef))
	if err != nil {
		return nil, err
	}
	ultima, err := db.ObtenerUltimaSesion(strings.TrimSpace(agenteRef), &proyecto.ID)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	conector, err := ResolverConector(strings.TrimSpace(agenteRef), conectorRef, ultima)
	if err != nil {
		return nil, err
	}
	return PrepararDesdeDatos(agente, proyecto, conector, ultima, modelo, razonamiento, perfilTarea)
}

func PrepararDesdeDatos(agente *db.Agente, proyecto *db.Proyecto, conector *db.Conector, ultima *db.Sesion, modelo, razonamiento, perfilTarea string) (*Preparacion, error) {
	return prepararDesdeDatosConWorkspace(agente, proyecto, conector, ultima, modelo, razonamiento, perfilTarea, gitoperaciones.WorktreeManager{})
}

func prepararDesdeDatosConWorkspace(agente *db.Agente, proyecto *db.Proyecto, conector *db.Conector, ultima *db.Sesion, modelo, razonamiento, perfilTarea string, workspace coordinacion.WorkspaceManager) (*Preparacion, error) {
	if agente == nil || proyecto == nil || conector == nil {
		return nil, fmt.Errorf("agente, proyecto y conector son obligatorios")
	}
	modeloSolicitado := strings.TrimSpace(modelo)
	razonamientoSolicitado := strings.TrimSpace(razonamiento)
	perfilSolicitado := strings.TrimSpace(perfilTarea)
	start := time.Now()
	prepareRuntimeDebugf("PrepararDesdeDatos start agente=%s proyecto=%s conector=%s", strings.TrimSpace(agente.Nombre), strings.TrimSpace(proyecto.Slug), strings.TrimSpace(conector.Slug))
	defer func() {
		prepareRuntimeDebugf("PrepararDesdeDatos done agente=%s proyecto=%s duration=%s", strings.TrimSpace(agente.Nombre), strings.TrimSpace(proyecto.Slug), time.Since(start).Round(time.Millisecond))
	}()
	var err error
	proyecto = db.ProyectoConRutaEfectiva(proyecto, "")
	stepStart := time.Now()
	perfilTarea, modelo, razonamiento, err = db.ResolverPerfilEjecucionLanzamiento(
		&agente.Nombre,
		strings.TrimSpace(proyecto.Slug),
		perfilTarea,
		modelo,
		razonamiento,
	)
	if err != nil {
		return nil, err
	}
	perfilTarea, modelo, razonamiento, err = runtimeagente.AplicarDefaultsConector(
		runtimeagente.ConnectorConfig{
			Slug:         strings.TrimSpace(conector.Slug),
			Nombre:       strings.TrimSpace(conector.Nombre),
			Transporte:   strings.TrimSpace(conector.Transporte),
			Comando:      strings.TrimSpace(conector.Comando),
			ArgsJSON:     strings.TrimSpace(conector.ArgsJSON),
			EnvJSON:      strings.TrimSpace(conector.EnvJSON),
			MetadataJSON: strings.TrimSpace(conector.MetadataJSON),
			Activo:       conector.Activo,
		},
		perfilSolicitado,
		modeloSolicitado,
		razonamientoSolicitado,
		perfilTarea,
		modelo,
		razonamiento,
	)
	if err != nil {
		return nil, fmt.Errorf("metadata_json inválido para '%s': %w", strings.TrimSpace(conector.Slug), err)
	}
	prepareRuntimeDebugf("PrepararDesdeDatos step=resolver_perfil duration=%s", time.Since(stepStart).Round(time.Millisecond))

	stepStart = time.Now()
	worktree, err := asegurarWorktreeOperativa(strings.TrimSpace(agente.Nombre), proyecto, workspace)
	if err != nil {
		return nil, err
	}
	prepareRuntimeDebugf("PrepararDesdeDatos step=asegurar_worktree duration=%s", time.Since(stepStart).Round(time.Millisecond))
	stepStart = time.Now()
	resume, bootstrap, err := bootstrapruntime.Preparar(agente.Nombre, proyecto, ultima)
	if err != nil {
		return nil, err
	}
	prepareRuntimeDebugf("PrepararDesdeDatos step=bootstrap duration=%s", time.Since(stepStart).Round(time.Millisecond))
	resume = db.SanitizeResumeContextForProject(resume, proyecto)
	resume.CWD = db.RutaTrabajoPreferidaAgenteProyecto(strings.TrimSpace(agente.Nombre), proyecto, strings.TrimSpace(resume.CWD))
	if strings.TrimSpace(resume.CWD) == "" {
		resume.CWD = strings.TrimSpace(proyecto.RutaAbs)
	}
	if worktree != nil {
		if strings.TrimSpace(resume.Branch) == "" {
			resume.Branch = strings.TrimSpace(worktree.Branch)
		}
	}
	resume = runtimeagente.SanitizarResumeParaConector(runtimeagente.ConnectorConfig{
		Slug:         strings.TrimSpace(conector.Slug),
		Nombre:       strings.TrimSpace(conector.Nombre),
		Transporte:   strings.TrimSpace(conector.Transporte),
		Comando:      strings.TrimSpace(conector.Comando),
		ArgsJSON:     strings.TrimSpace(conector.ArgsJSON),
		EnvJSON:      strings.TrimSpace(conector.EnvJSON),
		MetadataJSON: strings.TrimSpace(conector.MetadataJSON),
		Activo:       conector.Activo,
	}, resume)
	req := runtimeagente.LaunchRequest{
		Agente:       strings.TrimSpace(agente.Nombre),
		Rol:          strings.TrimSpace(agente.Rol),
		ProyectoSlug: strings.TrimSpace(proyecto.Slug),
		ProyectoRuta: strings.TrimSpace(proyecto.RutaAbs),
		Modelo:       strings.TrimSpace(modelo),
		Razonamiento: strings.TrimSpace(razonamiento),
		PerfilTarea:  strings.TrimSpace(perfilTarea),
		Conector: runtimeagente.ConnectorConfig{
			Slug:         strings.TrimSpace(conector.Slug),
			Nombre:       strings.TrimSpace(conector.Nombre),
			Transporte:   strings.TrimSpace(conector.Transporte),
			Comando:      strings.TrimSpace(conector.Comando),
			ArgsJSON:     strings.TrimSpace(conector.ArgsJSON),
			EnvJSON:      strings.TrimSpace(conector.EnvJSON),
			MetadataJSON: strings.TrimSpace(conector.MetadataJSON),
			Activo:       conector.Activo,
		},
		Resume: resume,
	}
	stepStart = time.Now()
	plan, err := runtimeagente.DefaultRegistry().Prepare(req)
	if err != nil {
		return nil, err
	}
	prepareRuntimeDebugf("PrepararDesdeDatos step=registry_prepare duration=%s", time.Since(stepStart).Round(time.Millisecond))

	return &Preparacion{
		Agente:    agente,
		Proyecto:  proyecto,
		Conector:  conector,
		Ultima:    ultima,
		Bootstrap: bootstrap,
		Plan:      plan,
	}, nil
}

func prepareRuntimeDebugf(format string, args ...any) {
	if !prepareRuntimeDebugEnabled() {
		return
	}
	log.Printf("orquesta[prepare-runtime] "+format, args...)
}

func prepareRuntimeDebugEnabled() bool {
	for _, key := range []string{"ORQUESTA_DEBUG_PREPARE", "ORQUESTA_DEBUG"} {
		value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
		switch value {
		case "1", "true", "yes", "on", "si", "sí":
			return true
		}
	}
	return false
}

func asegurarWorktreeOperativa(agente string, proyecto *db.Proyecto, workspace coordinacion.WorkspaceManager) (*coordinacion.Worktree, error) {
	if proyecto == nil || proyecto.ID == 0 || strings.TrimSpace(agente) == "" {
		return nil, nil
	}
	if proyecto.Tipo != db.ProyectoRepo {
		return nil, nil
	}
	state := coordinacion.WorktreeActive
	filter := coordinacion.WorktreeFilter{
		ProjectID: &proyecto.ID,
		Agent:     strPtr(strings.TrimSpace(agente)),
		State:     &state,
	}
	repo := db.CoordinationWorktreeRepository()
	activa, err := repo.List(filter)
	if err != nil {
		return nil, err
	}
	if len(activa) > 0 {
		return activa[0], nil
	}
	tarea, err := resolverTareaActivaAgenteProyecto(strings.TrimSpace(agente), proyecto.ID)
	if err != nil || tarea == nil {
		return nil, err
	}
	svc := &coordinacion.Service{
		Locks:     db.CoordinationLockRepository(),
		Worktrees: repo,
		Projects:  db.CoordinationProjectRepository(),
		Sessions:  db.CoordinationSessionRepository(),
		Config:    db.CoordinationConfigRepository(),
		Workspace: workspace,
	}
	return svc.PrepareWorktree(coordinacion.PrepareWorktreeInput{
		ProjectRef: strings.TrimSpace(proyecto.Slug),
		Agent:      strings.TrimSpace(agente),
		TaskID:     &tarea.ID,
		Reason:     "autonomia_runtime",
	})
}

func resolverTareaActivaAgenteProyecto(agente string, proyectoID int64) (*db.Tarea, error) {
	tareas, err := db.ListarTareas(db.FiltroTareas{
		Agente:     strPtr(strings.TrimSpace(agente)),
		ProyectoID: &proyectoID,
	})
	if err != nil {
		return nil, err
	}
	var asignada *db.Tarea
	for _, tarea := range tareas {
		if tarea == nil {
			continue
		}
		switch tarea.Estado {
		case db.TareaEnProgreso:
			return tarea, nil
		case db.TareaAsignada:
			if asignada == nil {
				asignada = tarea
			}
		}
	}
	return asignada, nil
}

func strPtr(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}

func ResolverConector(agente, conectorRef string, ultima *db.Sesion) (*db.Conector, error) {
	ref := strings.TrimSpace(conectorRef)
	if ref == "" && ultima != nil {
		if strings.TrimSpace(ultima.ConectorSlug) != "" {
			ref = strings.TrimSpace(ultima.ConectorSlug)
		} else if ultima.ConectorID != nil && *ultima.ConectorID > 0 {
			ref = fmt.Sprintf("%d", *ultima.ConectorID)
		}
	}
	if ref == "" {
		ref = runtimeagente.ConectorPorDefectoAgente(agente)
	}
	conector, err := db.GetConector(ref)
	if err != nil {
		if ref != runtimeagente.ConectorPorDefectoAgente(agente) {
			return nil, fmt.Errorf("no se pudo resolver el conector para %s: %w", strings.TrimSpace(agente), err)
		}
		return nil, err
	}
	return conector, nil
}

func ResumeContextNoVacio(resume runtimeagente.ResumeContext) bool {
	return strings.TrimSpace(resume.ExternalSessionID) != "" ||
		strings.TrimSpace(resume.ResumePayloadJSON) != "" ||
		strings.TrimSpace(resume.ResumenContinuidad) != ""
}
