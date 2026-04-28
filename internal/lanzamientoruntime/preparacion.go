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

const resolverPerfilPrepareTimeout = 250 * time.Millisecond

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
	proyecto = db.ProyectoPrepareLiteConRutaEfectiva(proyecto, strings.TrimSpace(agente.Nombre))
	stepStart := time.Now()
	perfilTarea, modelo, razonamiento, err = resolverPerfilEjecucionLanzamientoBestEffort(
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
	conectorRuntime := runtimeagente.ConnectorConfig{
		Slug:         strings.TrimSpace(conector.Slug),
		Nombre:       strings.TrimSpace(conector.Nombre),
		Transporte:   strings.TrimSpace(conector.Transporte),
		Comando:      strings.TrimSpace(conector.Comando),
		ArgsJSON:     strings.TrimSpace(conector.ArgsJSON),
		EnvJSON:      strings.TrimSpace(conector.EnvJSON),
		MetadataJSON: strings.TrimSpace(conector.MetadataJSON),
		Activo:       conector.Activo,
	}
	if !runtimeagente.ModeloCompatibleConConector(conectorRuntime, modelo) {
		modelo = ""
		if modeloSolicitado == "" {
			perfilTarea, modelo, razonamiento, err = runtimeagente.AplicarDefaultsConector(
				conectorRuntime,
				perfilSolicitado,
				"",
				razonamientoSolicitado,
				perfilTarea,
				"",
				razonamiento,
			)
			if err != nil {
				return nil, fmt.Errorf("metadata_json inválido para '%s': %w", strings.TrimSpace(conector.Slug), err)
			}
		}
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
	stepStart = time.Now()
	resume = db.SanitizeResumeContextForProject(resume, proyecto)
	prepareRuntimeDebugf("PrepararDesdeDatos step=sanitize_resume duration=%s", time.Since(stepStart).Round(time.Millisecond))
	stepStart = time.Now()
	resume.CWD = db.RutaTrabajoPreferidaAgenteProyectoPrepareLite(strings.TrimSpace(agente.Nombre), proyecto, strings.TrimSpace(resume.CWD))
	prepareRuntimeDebugf("PrepararDesdeDatos step=ruta_trabajo_preferida duration=%s", time.Since(stepStart).Round(time.Millisecond))
	if worktree != nil && coordinacion.WorktreePathUsable(strings.TrimSpace(worktree.Path)) {
		cwdNormalizado := strings.TrimSpace(resume.CWD)
		rutaProyecto := strings.TrimSpace(proyecto.RutaAbs)
		if cwdNormalizado == "" || cwdNormalizado == rutaProyecto {
			resume.CWD = strings.TrimSpace(worktree.Path)
		}
	}
	if strings.TrimSpace(resume.CWD) == "" {
		resume.CWD = strings.TrimSpace(proyecto.RutaAbs)
	}
	if worktree != nil {
		if strings.TrimSpace(resume.Branch) == "" {
			resume.Branch = strings.TrimSpace(worktree.Branch)
		}
	}
	stepStart = time.Now()
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
	prepareRuntimeDebugf("PrepararDesdeDatos step=sanitizar_resume_conector duration=%s", time.Since(stepStart).Round(time.Millisecond))
	req := runtimeagente.LaunchRequest{
		Agente:       strings.TrimSpace(agente.Nombre),
		Rol:          strings.TrimSpace(agente.Rol),
		ProyectoSlug: strings.TrimSpace(proyecto.Slug),
		ProyectoRuta: strings.TrimSpace(proyecto.RutaAbs),
		Modelo:       strings.TrimSpace(modelo),
		Razonamiento: strings.TrimSpace(razonamiento),
		PerfilTarea:  strings.TrimSpace(perfilTarea),
		Conector:     conectorRuntime,
		Resume:       resume,
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
	inicio := time.Now()
	state := coordinacion.WorktreeActive
	filter := coordinacion.WorktreeFilter{
		ProjectID: &proyecto.ID,
		Agent:     strPtr(strings.TrimSpace(agente)),
		State:     &state,
	}
	repo := db.CoordinationWorktreeRepository()
	activa, err := db.ListarWorktreesCoordPrepareLite(filter)
	if err != nil {
		return nil, err
	}
	prepareRuntimeDebugf("PrepararDesdeDatos step=asegurar_worktree_lookup duration=%s agente=%s proyecto=%s activas=%d", time.Since(inicio).Round(time.Millisecond), strings.TrimSpace(agente), strings.TrimSpace(proyecto.Slug), len(activa))
	if len(activa) > 0 {
		return activa[0], nil
	}
	inicioTarea := time.Now()
	tarea, err := resolverTareaActivaAgenteProyecto(strings.TrimSpace(agente), proyecto.ID)
	if err != nil || tarea == nil {
		return nil, err
	}
	prepareRuntimeDebugf("PrepararDesdeDatos step=resolver_tarea_activa duration=%s agente=%s proyecto=%s tarea=%d", time.Since(inicioTarea).Round(time.Millisecond), strings.TrimSpace(agente), strings.TrimSpace(proyecto.Slug), tarea.ID)
	inicioPrepare := time.Now()
	svc := &coordinacion.Service{
		Locks:     db.CoordinationLockRepository(),
		Worktrees: repo,
		Projects:  db.CoordinationProjectRepository(),
		Sessions:  db.CoordinationSessionRepository(),
		Config:    db.CoordinationConfigRepository(),
		Workspace: workspace,
	}
	worktree, err := svc.PrepareWorktree(coordinacion.PrepareWorktreeInput{
		ProjectRef: strings.TrimSpace(proyecto.Slug),
		Agent:      strings.TrimSpace(agente),
		TaskID:     &tarea.ID,
		Reason:     "autonomia_runtime",
	})
	prepareRuntimeDebugf("PrepararDesdeDatos step=prepare_worktree duration=%s agente=%s proyecto=%s err=%v", time.Since(inicioPrepare).Round(time.Millisecond), strings.TrimSpace(agente), strings.TrimSpace(proyecto.Slug), err)
	return worktree, err
}

func resolverTareaActivaAgenteProyecto(agente string, proyectoID int64) (*db.Tarea, error) {
	inicio := time.Now()
	tarea, err := db.GetTareaActivaPrepareLite(strings.TrimSpace(agente), proyectoID)
	if err != nil {
		return nil, err
	}
	tareaID := int64(0)
	if tarea != nil {
		tareaID = tarea.ID
	}
	prepareRuntimeDebugf("PrepararDesdeDatos step=resolver_tarea_agente_prepare_lite duration=%s agente=%s proyecto_id=%d tarea=%d", time.Since(inicio).Round(time.Millisecond), strings.TrimSpace(agente), proyectoID, tareaID)
	return tarea, nil
}

func strPtr(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}

func resolverPerfilEjecucionLanzamientoBestEffort(agenteNombre *string, proyectoSlug, perfilTarea, modelo, razonamiento string) (string, string, string, error) {
	if strings.TrimSpace(perfilTarea) != "" && strings.TrimSpace(modelo) != "" && strings.TrimSpace(razonamiento) != "" {
		return perfilTarea, modelo, razonamiento, nil
	}
	type result struct {
		perfil       string
		modelo       string
		razonamiento string
		err          error
	}
	done := make(chan result, 1)
	go func() {
		perfilRes, modeloRes, razonamientoRes, err := db.ResolverPerfilEjecucionLanzamiento(
			agenteNombre,
			proyectoSlug,
			perfilTarea,
			modelo,
			razonamiento,
		)
		done <- result{
			perfil:       perfilRes,
			modelo:       modeloRes,
			razonamiento: razonamientoRes,
			err:          err,
		}
	}()
	select {
	case res := <-done:
		return res.perfil, res.modelo, res.razonamiento, res.err
	case <-time.After(resolverPerfilPrepareTimeout):
		prepareRuntimeDebugf("PrepararDesdeDatos step=resolver_perfil_timeout proyecto=%s timeout=%s", strings.TrimSpace(proyectoSlug), resolverPerfilPrepareTimeout)
		return strings.TrimSpace(perfilTarea), strings.TrimSpace(modelo), strings.TrimSpace(razonamiento), nil
	}
}

func ResolverConector(agente, conectorRef string, ultima *db.Sesion) (*db.Conector, error) {
	ref := strings.TrimSpace(conectorRef)
	if ref == "" && ultima != nil {
		if strings.TrimSpace(ultima.ConectorSlug) != "" &&
			runtimeagente.ConectorCompatibleConAgente(agente, strings.TrimSpace(ultima.ConectorSlug), "") {
			ref = strings.TrimSpace(ultima.ConectorSlug)
		} else if ultima.ConectorID != nil && *ultima.ConectorID > 0 &&
			runtimeagente.ConectorCompatibleConAgente(agente, "", strings.TrimSpace(ultima.Herramienta)) {
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
