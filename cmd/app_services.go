package cmd

import (
	"database/sql"
	"errors"
	"strings"

	"orquesta/agentesapp"
	"orquesta/capacidadapp"
	"orquesta/conectoresapp"
	"orquesta/configuracionapp"
	"orquesta/db"
	"orquesta/gitaplicacion"
	"orquesta/gobernanzaapp"
	"orquesta/lenguajeapp"
	"orquesta/microprogramacionapp"
	"orquesta/orquestacionagentesapp"
	"orquesta/progresoapp"
	"orquesta/reviewapp"
	"orquesta/supervisionapp"
)

var capacidadService = capacidadapp.NewService(capacidadapp.Repository{})
var agentesService = agentesapp.NewService(agentesapp.Repository{}, capacidadService)
var configService = configuracionapp.NewService(configuracionapp.Repository{})
var conectoresService = conectoresapp.NewService(conectoresapp.Repository{})
var gitService = gitaplicacion.NewService(db.GitGovRepository{})
var gobernanzaService = gobernanzaapp.NewService(gobernanzaapp.Repository{})
var lenguajeService = lenguajeapp.NewService(lenguajeapp.Repository{})
var microprogramacionService = microprogramacionapp.NewService(db.MicroprogramacionRepository{})
var orquestacionAgentesService = orquestacionagentesapp.NewService(agentesService, runtimesService)
var progresoService = progresoapp.NewService(progresoapp.Repository{})
var reviewService = reviewapp.NewService(reviewapp.NewRepository())
var supervisionService = supervisionapp.NewService(supervisionapp.NewRepository())

func init() {
	orquestacionAgentesService.SetAutonomyStore(orquestacionagentesapp.Repository{})
	orquestacionAgentesService.SetStartableWorkChecker(startableWorkChecker{})
	orquestacionAgentesService.SetRemoteRecoverySupport(orquestacionagentesapp.Repository{})
	orquestacionAgentesService.SetRecoveryFlowSupport(recoveryFlowSupport{})
	capacidadService.SetPhaseProvider(progresoService)
	capacidadService.SetReviewGateProvider(reviewService)
	capacidadService.SetReviewGateManager(reviewService)
	capacidadService.SetRuntimeModelManager(nuevoGestorRuntimeModelosOllama(endpointOllamaLocal(), clienteHTTPOllamaLocal()))
	capacidadService.SetTaskProvider(capacidadapp.Repository{})
	capacidadService.SetTaskActionProvider(capacidadapp.Repository{})
	capacidadService.SetPhaseControlProvider(capacidadapp.Repository{})
	capacidadService.SetAgentResolver(resolvedorAgentePipelineOperativo{
		rowsProvider:  agentesService,
		scoreProvider: proveedorScoreAgentePipelineDB{},
	})
	capacidadService.SetPipelineDispatcher(despachadorPipelineOperativo{})
	microprogramacionService.SetEscritorArchivos(microprogramacionapp.EscritorArchivosDisco{})
	microprogramacionService.SetRecolectorEntregaGit(microprogramacionRecolectorGit{})
	microprogramacionService.SetIntegradorGit(microprogramacionIntegradorGit{})
	runtimesService.SetRegistradorEntregaGit(microprogramacionService)
	runtimesService.SetRegistradorEntregaGitPremium(premiumRuntimeGitService{git: gitService})
	runtimesService.SetMaterializadorEntregaMicroprogramacion(microprogramacionService)
	runtimesService.SetResolvedorWorktreeActiva(worktreeRuntimeService{})
	runtimesService.SetAseguradorWorktreeActiva(worktreeRuntimeService{})
	runtimesService.SetTaskCompleter(tareasService)
}

type startableWorkChecker struct{}

func (startableWorkChecker) HasStartableAgentWork(agente string, proyectoID int64) (bool, error) {
	return dbAgenteTieneTrabajoArrancable(agente, proyectoID)
}

type recoveryFlowSupport struct{}

func (recoveryFlowSupport) SharedAccountAvailable(agente string) (bool, string, error) {
	return autonomiaCuentaCompartidaDisponible(agente)
}

func (recoveryFlowSupport) UsesSharedLocalPoolForReactivation(agente string, proyecto *db.Proyecto) bool {
	return agenteUsaPoolLocalCompartidoParaReactivacion(agente, proyecto)
}

func (recoveryFlowSupport) PoolLocalActivationAllowed(agente string, proyectoSlug string) (bool, string, error) {
	return db.PoolLocalCompartidoPermiteActivacionAgenteProyecto(agente, proyectoSlug)
}

func (recoveryFlowSupport) ProjectHasReactivableBacklog(proyectoID int64) (bool, error) {
	filtro := db.FiltroTareas{ProyectoID: &proyectoID}
	tareas, err := tareasService.List(filtro)
	if err != nil {
		return false, err
	}
	for _, tarea := range tareas {
		if tarea == nil {
			continue
		}
		switch tarea.Estado {
		case db.TareaLibre:
			return true, nil
		case db.TareaBloqueada:
			motivo, err := db.MotivoBloqueoActivoTarea(tarea.ID)
			if err != nil {
				return false, err
			}
			texto := strings.TrimSpace(motivo)
			if texto == "" {
				continue
			}
			if esBloqueoSobrecargaOperativa(texto) ||
				strings.HasPrefix(texto, "Agente ") ||
				strings.HasPrefix(texto, "Agente degradado:") {
				return true, nil
			}
		}
	}
	return false, nil
}

func (recoveryFlowSupport) ProjectIDFromAssignedWork(agente string) (int64, error) {
	filtro := db.FiltroTareas{Agente: &agente}
	tareas, err := tareasService.List(filtro)
	if err != nil {
		return 0, err
	}
	bestProjectID := int64(0)
	bestRank := 99
	bestID := int64(0)
	rankForState := func(estado db.EstadoTarea) int {
		switch estado {
		case db.TareaEnProgreso:
			return 0
		case db.TareaAsignada:
			return 1
		case db.TareaBloqueada:
			return 2
		default:
			return 99
		}
	}
	for _, tarea := range tareas {
		if tarea == nil || tarea.ProyectoID == nil || *tarea.ProyectoID <= 0 {
			continue
		}
		rank := rankForState(tarea.Estado)
		if rank > 2 {
			continue
		}
		if bestProjectID == 0 || rank < bestRank || (rank == bestRank && tarea.ID > bestID) {
			bestProjectID = *tarea.ProyectoID
			bestRank = rank
			bestID = tarea.ID
		}
	}
	return bestProjectID, nil
}

func (recoveryFlowSupport) ActiveProjectID(agente string) (int64, error) {
	return db.ObtenerProyectoActivoAgente(agente)
}

func (recoveryFlowSupport) OpenSessionProjectID(agente string) (int64, error) {
	sesion, err := db.GetSesionAbierta(agente, nil)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	if sesion == nil || sesion.ProyectoID == nil || *sesion.ProyectoID <= 0 {
		return 0, nil
	}
	return *sesion.ProyectoID, nil
}

func (recoveryFlowSupport) LatestSessionProjectID(agente string) (int64, error) {
	sesion, err := db.ObtenerUltimaSesion(agente, nil)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	if sesion == nil || sesion.ProyectoID == nil || *sesion.ProyectoID <= 0 {
		return 0, nil
	}
	return *sesion.ProyectoID, nil
}
