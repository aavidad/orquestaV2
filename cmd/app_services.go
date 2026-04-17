package cmd

import (
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
var microprogramacionService = microprogramacionapp.NewService(db.SqliteMicroprogramacionRepo{})
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
	capacidadService.SetAgentResolver(resolvedorAgentePipelineOperativo{rowsProvider: agentesService})
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

func (recoveryFlowSupport) ReactivateProjectIfNeeded(agente string, proyecto *db.Proyecto, motivo string) (bool, error) {
	return encolarReactivacionAgenteProyectoSiProcede(agente, proyecto, motivo)
}
