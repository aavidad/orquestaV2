# Contratos: orquesta-http-gateway

## `NewAppGatewayMuxV0`

Constructor publico:

```go
func NewAppGatewayMuxV0(handlers RouteHandlersV0) http.Handler
```

Reglas:

- devuelve siempre un `http.Handler`;
- registra una ruta solo si el handler correspondiente no es `nil`;
- no envuelve ni modifica la request;
- no restringe metodo HTTP;
- no inyecta dependencias de negocio.

## `RouteHandlersV0`

Contenedor de handlers `net/http` ya construidos por otro borde de composicion.

Campos:

- `Home`;
- `NuevaApp`;
- `AutoprogrammingPage`;
- `AppChangePage`;
- `DirectorStatsPage`;
- `RunControlPage`;
- `RunQueuePage`;
- `AppSpec`;
- `AppDirector`;
- `AppIntakeGuidedTurn`;
- `AppChange`;
- `DirectorStats`;
- `RunControl`;
- `RunQueuePriority`;
- `RunSupervisor`;
- `ServerShutdown`;
- `AutoprogrammingValidateRequest`.
- `AutoprogrammingSelfImprovement`.
- `AutoprogrammingPrepareRun`.
- `AutoprogrammingStatus`.
- `AutoprogrammingSupervise`.
- `GovernanceCatalogQuery`.
- `HumanDirectorWorkReviewPlan`.
- `DomainWork`.
- `ExternalWorkRun`.

## Rutas estables

- `RouteHomeV0`: `/`;
- `RouteNuevaAppV0`: `/nueva-app`;
- `RouteAutoprogrammingPageV0`: `/autoprogramming`;
- `RouteAppChangePageV0`: `/app-change`;
- `RouteDirectorStatsPageV0`: `/director-stats`;
- `RouteRunControlPageV0`: `/run-control`;
- `RouteRunQueuePageV0`: `/run-queue`;
- `RouteAppSpecV0`: `/api/v0/apps/spec`;
- `RouteAppDirectorV0`: `/api/v0/apps/director`;
- `RouteAppDirectorGoalObserveV0`: `/api/v0/apps/director/goal/observe`;
- `RouteAppIntakeGuidedTurnV0`: `/api/v0/apps/intake/guided-turn`;
- `RouteAppChangeV0`: `/api/v0/apps/`;
- `RouteDirectorStatsV0`: `/api/v0/director/stats`.
- `RouteRunControlV0`: `/api/v0/runs/control`;
- `RouteRuntimeModelsV0`: `/api/v0/runtime/models`;
- `RouteRunQueuePriorityV0`: `/api/v0/runs/queue/priority`;
- `RouteRunSupervisorV0`: `/api/v0/runs/supervise`;
- `RouteAutoprogrammingValidateRequestV0`: `/api/v0/autoprogramming/validate-request`;
- `RouteAutoprogrammingSelfImprovementV0`: `/api/v0/autoprogramming/self-improvement`;
- `RouteAutoprogrammingPrepareRunV0`: `/api/v0/autoprogramming/prepare-run`;
- `RouteAutoprogrammingStatusV0`: `/api/v0/autoprogramming/status`;
- `RouteAutoprogrammingSuperviseV0`: `/api/v0/autoprogramming/supervise`;
- `RouteGovernanceCatalogQueryV0`: `/api/v0/governance/catalog/query`;
- `RouteHumanDirectorWorkReviewPlanV0`: `/api/v0/director/human-work/review-plan`;
- `RouteServerShutdownV0`: `/api/v0/server/shutdown`.
- `RouteDomainWorkV0`: `/api/v0/domain-work`.
- `RouteExternalWorkRunV0`: `/api/v0/external-work/run`.

## Manifiesto de rutas

`PublicRouteManifestV0` publica el inventario canonico de rutas exactas,
prefijos y overlays externos con `ref`, `owner`, metodo esperado y perfil de
seguridad. Las rutas exactas bajo un prefijo, como `/api/v0/apps/director`,
`/api/v0/apps/intake/guided-turn` o el overlay AppVCS `/api/v0/apps/vcs`, deben
declarar explicitamente que preceden al prefijo `/api/v0/apps/`.

`ValidateRouteManifestV0` detecta duplicados exactos, prefijos duplicados y
shadows no declarados. El builder del mux valida que las rutas que registra
existan en el manifiesto y que su patron coincida con la entrada canonica.

`RouteDirectorStatsPageV0` y `RouteDirectorStatsV0` son rutas separadas: la
primera apunta al handler web inyectado y la segunda al contrato REST que
transporta `DirectorRunStatsV0` completo. El gateway no inspecciona campos de
tareas, agentes, rework, replan, progreso, cierre ni `decision_context`.

`RouteRunQueuePageV0` y `RouteRunQueuePriorityV0` son rutas separadas: la
primera apunta al panel web inyectado y la segunda al contrato REST/MCP de
cola. El gateway no conoce ranking, prioridad, aging, fairness ni stores.

`RouteAppDirectorGoalObserveV0` apunta al bridge REST/MCP que observa un goal
ya lanzado por el Director de nueva app. Es ruta exacta bajo `/api/v0/apps/`,
precede al catch-all de app-change y se clasifica como mutacion de control
plane porque puede persistir resultado observado, cerrar o bloquear la run.

`RouteAppIntakeGuidedTurnV0` apunta al endpoint JSON de intake guiado de nueva
app. Es ruta exacta bajo `/api/v0/apps/`, precede al catch-all de app-change y
se clasifica como lectura de control plane porque no persiste estado ni dispara
trabajo externo.

`RouteRunControlPageV0` y `RouteRunControlV0` son rutas separadas: la primera
apunta al panel web inyectado y la segunda al contrato REST/MCP de control. El
gateway no conoce pausa, parada, checkpoint, procesos ni runtime.

`RouteRuntimeModelsV0` apunta al contrato REST/MCP de gestion opt-in de modelos
runtime. El gateway solo monta el handler inyectado; no conoce Ollama, base URL,
token, HOME, proveedor, scheduler ni politica de seleccion de modelo.

`RouteRunSupervisorV0` apunta al contrato REST/MCP que ejecuta una pasada
acotada de supervision sobre una run o sobre la cola inyectada. El gateway solo
monta el handler; no conoce Codex, OPES, scheduler, runtime, DB ni reglas de
dominio.

`RouteAutoprogrammingPrepareRunV0` apunta al contrato REST/MCP que prepara una
run de autoprogramacion continuable por executor inyectado. El gateway no crea
runs ni arranca agentes; solo monta el handler y no recorta `goal?` ni
`goal_specs?` cuando el handler inyectado los publica.

`RouteAutoprogrammingObserveGoalV0` apunta al contrato REST/MCP que observa un
goal de autoprogramacion por `run_ref`. El gateway no observa goals, no valida
cierre, no sincroniza cola y no conoce Codex, OPES, scheduler, runtime, DB ni
reglas de dominio.

`RouteAutoprogrammingObserveActiveGoalsV0` apunta al contrato REST/MCP que
observa en lote goals activos de autoprogramacion. El gateway solo monta el
handler inyectado; no lista estados goal, no ejecuta `supervise` legacy y no
arranca proveedor.

`RouteAutoprogrammingStatusV0` y `RouteAutoprogrammingSuperviseV0` apuntan a
contratos REST/MCP finos para estado/diagnostico y supervision puntual de
autoprogramacion. El gateway solo registra handlers inyectados.

`RouteAutoprogrammingSelfImprovementV0` apunta al contrato REST/MCP que convierte
evidencia de fallo en automejora de baja prioridad. El gateway no prepara runs
ni decide cola; solo registra el handler inyectado.

`RouteHumanDirectorWorkReviewPlanV0` apunta al contrato REST/MCP de revision de
trabajo humano. El gateway no conoce operadores concretos ni transporte MCP
real; solo registra el handler inyectado.

`RouteDomainWorkV0` apunta al contrato REST/MCP de trabajo de dominio externo.
El gateway solo registra el handler inyectado; no conoce OPES, contratos de
dominio, conectores REST, DB, runtime ni proveedores.

`RouteExternalWorkRunV0` apunta al contrato REST/MCP que crea un run operativo
para un trabajo externo ya definido. El gateway no crea runs, no abre fases y no
encola por si mismo; solo monta el handler inyectado.

## Invariantes

- el codigo productivo solo depende de libreria estandar;
- el gateway no importa paquetes de `cmd`, `db`, `codex`, `runtime`,
  `orquesta-web`, `orquesta-mcp` ni `orquesta-factory`;
- una ruta sin handler inyectado responde como no configurada.
