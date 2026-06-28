# Diseño: observador goal-first autónomo + estado vivo de cola

Fecha: 2026-06-27.
Para: agente programador. Cubre las tareas **T1** y **T2** del encargo
(`TAREA_OPES_ORQUESTA_ENCARGO_PROGRAMADOR_FIXES_2026-06-27.md`).

## Capacidad principal: programar de forma autónoma

Lo más importante de Orquesta **no** es procesar temarios OPES, sino
**programar de forma autónoma** (auto-mejora: construir y mejorar software por sí
mismo). Los temarios OPES son **un consumidor más** del mismo motor. La buena
noticia es que el "cerebro" de autoprogramación **ya está cableado**:

- El supervisor loop arranca **incondicionalmente** al iniciar el runtime
  (`modulos/orquesta-server/runtime_v0.go:130`), **no** depende de
  `ResidentDirectorEnabled` (solo el director residente LLM sí, en `:131`).
- En cada tick, si Orquesta está **idle**, llama a
  `maybeScheduleIdleSelfImprovementCausalV0`
  (`modulos/orquesta-server/supervisor_loop_v0.go:101`), que **lanza un goal de
  auto-mejora** goal-first (config `IdleSelfImprovementGoalFirst`, planner de
  backlog en `cmd/orquesta-server/idle_self_improvement_backlog_planner_v0.go`).

El **único hueco** es el mismo que para OPES: ese goal de auto-mejora se **lanza
pero nadie lo observa hasta cierre**. Por tanto **T1 (observador goal-first
residente) es la llave única que cierra el lazo para AMBOS caminos**:
autoprogramación autónoma y temarios OPES. Resolver T1 convierte el cerebro ya
existente en un ciclo completo "idle → programar → cerrar → siguiente".

## Principio rector del lazo

Orquesta es goal-first: `external-work/run` y la auto-mejora lanzan goals, y el
cierre depende de **observarlos**. **OPES es solo un conector**: entrega trabajo
y lee estado, pero **no forma parte del orquestador** y no debe conducir el lazo.
Por tanto, **la observación hasta cierre tiene que correr dentro de Orquesta, sin
polling externo**.

Hoy el lazo no se cierra solo:

- `ObserveActiveGoalWorksV0` (`modulos/orquesta-goal/observe_active_v0.go:26`) es
  el motor que lista goals activos y observa cada uno. **No tiene ningún llamador
  en producción** (huérfano).
- El endpoint `/api/v0/autoprogramming/goals/observe-active` existe
  (`orquesta-http-gateway/gateway_v0.go:34`, executor en
  `orquesta-app-codex-stack/stack_v0.go:128`) pero **necesita que un cliente lo
  dispare**; ese cliente acabaría siendo OPES → prohibido por el principio rector.
- El loop residente `runSupervisorLoopV0`
  (`modulos/orquesta-server/supervisor_loop_v0.go:13`) corre por timer pero solo
  drena cola (`RunGlobalSupervisorV0`); **no observa goals** y está detrás de
  `ResidentDirectorEnabled` (apagado en instancias OPES).

Resultado: goals lanzados que nadie observa → `waiting_outbox`/`running_stale`.

---

## T1 — Observador goal-first residente (en proceso, siempre activo)

### Objetivo
Que Orquesta, por sí mismo, progrese y cierre **todos** los goals activos sin
intervención externa, **independiente del director residente LLM** (que es pesado
y opcional). Esto incluye por igual:
- los goals de **auto-programación** lanzados por
  `maybeScheduleIdleSelfImprovementCausalV0` cuando está idle (capacidad
  principal), y
- los goals de **trabajo externo** lanzados por `external-work/run` (temarios
  OPES y otros consumidores).
Ambos terminan en el mismo `GoalStateStore`; el observador los trata igual.

### Diseño

1. **Nuevo ticker ligero `runGoalObservationLoopV0`** en
   `modulos/orquesta-server`, hermano de `runSupervisorLoopV0` pero separado:
   - Reutiliza la maquinaria async existente: `runAsyncWorkV0` +
     `waitAsyncWorkV0` (`runtime_async_work_v0.go`) y el patrón coalescente de
     `runSupervisorTickAsyncV0` (un solo tick en vuelo, sin solapar).
   - En cada tick llama a un puerto nuevo del runtime que ejecuta
     `orquestagoal.ObserveActiveGoalWorksV0(ctx, request, ports)` con los puertos
     **ya cableados**: `GoalStateStore`/lister, `Observer`, `ClosureValidator`
     (ver `orquesta-app-codex-stack/stack_v0.go:216-219` y
     `cmd/orquesta-server/stack.go:230-231`).
   - `request.List` por defecto = goals en estado activo (`running`); el filtro ya
     existe en `defaultActiveGoalWorkStateListRequestV0`
     (`observe_active_v0.go`).
   - Persistir transición por cada observación terminal
     (`persistStateTransitionV0`) para que el estado público refleje el cierre.

2. **Gating correcto (clave dada la aclaración OPES):** este ticker se controla
   con un flag propio **`ORQUESTA_SERVER_GOAL_OBSERVER_ENABLED` por defecto
   `true`**, NO con `ResidentDirectorEnabled`. Así una instancia con director
   residente apagado (caso OPES) **sigue cerrando goals sola**. El director
   residente LLM queda como capa opcional aparte.
   - Añadir el flag en `cmd/orquesta-server/server_env_autonomy_v0.go` siguiendo
     el patrón de `serverResidentDirectorEnabledFromEnvV0`, y arrancarlo en
     `runSupervisorLoopV0`/arranque del runtime junto al supervisor loop.
   - Intervalo configurable (reutiliza el patrón del intervalo del supervisor);
     por defecto razonable (p.ej. 2–5 s), nunca `time.Sleep` de sincronización en
     tests (usar `waitAsyncWorkV0`).

3. **El endpoint `observe-active` se mantiene** como disparo manual puntual
   (intervención de operador), pero **deja de ser el único motor**: el ticker es
   el camino autónomo. Misma función `ObserveActiveGoalWorksV0` detrás de ambos.

4. **Respuesta honesta de `external-work/run` goal-first:** si el observador está
   deshabilitado (`GOAL_OBSERVER_ENABLED=false`) y no hay director residente, el
   resultado debe avisar (`next_action=goal_observer_required` o similar) en vez
   de devolver `ok` y dejar el goal huérfano. Punto de cambio:
   `orquesta-app-codex-stack/external_work_goal_first_executor_v0.go` (resultado
   de éxito, alrededor de `externalWorkGoalFirstResultV0`).

### Verificación T1
- **Auto-programación end-to-end (capacidad principal):** con
  `ResidentDirectorEnabled=false`, `IdleSelfImprovementGoalFirst=true` y
  `GOAL_OBSERVER_ENABLED=true`, sin trabajo externo y con backlog de auto-mejora
  disponible, comprobar que Orquesta: detecta idle → lanza goal de auto-mejora →
  el observador lo lleva a terminal → puede encadenar el siguiente, **sin ninguna
  llamada externa**.
- Test de runtime: con `ResidentDirectorEnabled=false` y
  `GOAL_OBSERVER_ENABLED=true`, lanzar un goal-first vía `external-work/run`,
  dejar correr el ticker y comprobar que el `GoalWorkStateV0` llega a
  `complete`/terminal sin ninguna llamada externa. Sincronizar con
  `waitAsyncWorkV0`, no con `Sleep`.
- Test: con el observador apagado, `external-work/run` devuelve el aviso de
  `goal_observer_required`.
- `go test -race ./modulos/orquesta-server/...` verde.

### Avance Orquesta 2026-06-28 - `complete` pendiente de cierre observable

Se corrigió un hueco del observador goal-first: un goal con
`status=complete` pero sin `LastClosure.Accepted=true` ya no queda invisible ni
se etiqueta como `running_live`.

- `GoalWorkStatePendingObservationV0` define qué estados deben volver a
  observarse: `running` y `complete` sin closure aceptado/bloqueado.
- `ObserveActiveGoalWorksV0` y
  `orquesta.autoprogramming.observe_active_goals.v0` listan por defecto
  `running, complete` y filtran con ese helper.
- `autoprogramming/status` también lista esos goals pendientes por el lister y
  publica `queue_health.goal_closure_pending` en vez de inflar
  `running_live`.
- Los goals `complete` con closure aceptado o rework/bloqueo no se reobservan.

Cobertura añadida:

- `TestObserveActiveGoalWorksV0ListaYObservaPendientesDeObservacion`
- `TestMCPAutoprogrammingObserveActiveGoalsToolExecutorV0ListaYObservaActivos`
  actualizado para `running + complete pendiente`.
- `TestMCPAutoprogrammingStatusExecutorV0ListaGoalCompletePendingClosure`

Validación ejecutada:

- `go test -count=1 ./modulos/orquesta-goal ./modulos/orquesta-mcp -run 'ObserveActive|AutoprogrammingStatusExecutorV0(ListaGoalCompletePendingClosure|RecomiendaObserveGoalParaRunGoalFirst)'`
- `go test -count=1 ./modulos/orquesta-goal ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./cmd/orquesta-server`

---

## T2 — Estado vivo de cola: `running_live` vs `running_stale`

### Objetivo
Que `autoprogramming/status` distinga runs con proceso vivo de runs `running`
fantasma (incidencia `alive_percentage=0`) y proponga acción segura, sin que
OPES tenga que deducirlo.

### Diseño

1. **Reusar la detección de vivo que YA existe en el supervisor** en vez de la
   copia muerta. En `modulos/orquesta-server/supervisor_loop_v0.go` ya hay:
   `supervisorResultHasRunningLiveV0:198`,
   `supervisorResultHasUnverifiedRunningV0:164`,
   `supervisorDiagnosticsHaveLiveProcessV0:382`,
   `supervisorDiagnosticHasLiveProcessV0:366`.
   - Extraer esa lógica a una función pura reutilizable (p.ej. en
     `orquesta-run-coordinator` o `orquesta-run-supervisor`) que, dada la
     diagnosis/snapshot de una run, devuelva `live | stale | unverified`.

2. **Cablear esa función en la proyección de status**, eliminando la copia muerta
   `hasMCPAutoprogrammingLiveProcessSignalV0` /
   `hasMCPAutoprogrammingLiveAgentSignalV0`
   (`modulos/orquesta-mcp/autoprogramming_status_health_v0.go:230,258`). El
   `status` debe exponer por run uno de:
   `running_live`, `running_stale`, `waiting_outbox`, `waiting_external`,
   `ready`, `dispatching`, `blocked_by_capacity`, `blocked_by_stale_run`,
   `delivered`, `done`, `failed_to_materialize`.

3. **Reconciliación automática:** cuando una run esté `running` sin proceso vivo
   ni ACK pendiente real, degradarla a `running_stale` y, si procede, cerrarla o
   reencolarla. Conecta con la fuente de verdad de procesos
   (`ProcessRegistry.ResolveAgentProcessV0`, ya usado en
   `orquesta-app-codex-stack/run_coordinator_launch_outbox_recovery_v0.go:237`).
   Una run con `stop_requested` + checkpoint no debe reaparecer al frente de la
   cola sin causa explícita.

4. **`status` siempre acotado y diagnóstico:** mantener el timeout ya presente en
   los handlers MCP (`*_http_v0.go`, 2 s → 202) y garantizar cuerpo JSON con
   `recommended_action` aunque no pueda despachar.

### Verificación T2
- Test: dada una run `running` cuyo proceso no existe en `ProcessRegistry`,
  `status` la reporta `running_stale` con `recommended_action` segura.
- Test: run con proceso vivo → `running_live`; `alive_percentage` coherente.
- Test: `status` responde con cuerpo y `recommended_action` aun bajo presión de
  cola (no cuelga, no responde sin cuerpo).

---

## Orden sugerido e impacto

1. T1 ticker + gating propio (cierra el lazo autónomo sin OPES) — **lo que más
   sube la autonomía real**.
2. T2 estado vivo + reconciliación (observabilidad accionable y fin de los
   `running` fantasma).

Ambas usan piezas que **ya existen** (`ObserveActiveGoalWorksV0`, puertos goal
cableados, detección de vivo del supervisor, maquinaria async coalescente); el
trabajo es **cablear y separar gating**, no construir de cero.

## Apéndice: firmas exactas (para no decidir nombres)

### Estado actual relevante (verificado en código)
- Los goals de **auto-mejora** ya se observan, pero solo **uno pendiente por
  tick**, vía `observePendingIdleSelfImprovementGoalV0`
  (`modulos/orquesta-server/supervisor_idle_goal_observation_v0.go:22`), leyendo
  el ref del **tracker**, no de la lista de estados.
- Los goals de **external-work** se guardan en `GoalStateStore`
  (`external_work_goal_first_executor_v0.go`, `SaveGoalWorkStateV0`) pero **nadie
  los observa**.
- El motor multi-goal `ObserveActiveGoalWorksV0` (lista activa desde el store)
  cubre **ambos** y hoy está huérfano. Es el que hay que cablear.

### Prerrequisito (IMPORTANTE)
`ObserveActiveGoalWorksV0` hace `ports.StateStore.(GoalWorkStateListPortV0)`
(`observe_active_v0.go:40`). Para que funcione:
1. El `GoalWorkStateStorePortV0` de producción (file-backed, `cmd/orquesta-server/stack.go:207`)
   **debe implementar también** `GoalWorkStateListPortV0`
   (`ListGoalWorkStatesV0(ctx, GoalWorkStateListRequestV0) ([]GoalWorkStateV0, error)`).
   Verificar/añadir.
2. Los goals de auto-mejora deben **persistirse en el mismo `GoalStateStore`**
   (no solo en el tracker) para que el observador activo los vea. Si hoy el lanzamiento
   idle no llama `SaveGoalWorkStateV0`, añadirlo en `launchIdleSelfImprovementGoalsV0`.
   Con esto, el ticker único reemplaza a `observePendingIdleSelfImprovementGoalV0`.

### T1 — puerto que expone el stack al server (patrón ya usado: type-assert sobre `runtime.supervisor`)
```go
// modulos/orquesta-server/ports_v0.go
type GoalActiveObservationPortV0 interface {
    ObserveActiveGoalWorksV0(
        context.Context,
        orquestagoal.GoalWorkObserveActiveRequestV0,
    ) (orquestagoal.GoalWorkObserveActiveResultV0, error)
}
```
El supervisor del codex-stack lo implementa delegando en el motor con sus puertos
ya cableados (`stack.go:230-231`, `stack_v0.go:216-219`):
```go
// modulos/orquesta-app-codex-stack (sobre el tipo que ya es SupervisorPortV0)
func (s StackV0) ObserveActiveGoalWorksV0(ctx context.Context, req orquestagoal.GoalWorkObserveActiveRequestV0) (orquestagoal.GoalWorkObserveActiveResultV0, error) {
    return orquestagoal.ObserveActiveGoalWorksV0(ctx, req, orquestagoal.GoalWorkLifecyclePortsV0{
        Launcher:         s.Ports.GoalLauncher,
        Observer:         s.Ports.GoalObserver,
        ClosureValidator: s.Ports.GoalClosureValidator, // si existe; si no, nil
        StateStore:       s.Ports.GoalStateStore,
    })
}
```

### T1 — ticker en el runtime (espejo del supervisor, coalescente)
```go
// modulos/orquesta-server: campos nuevos en RuntimeV0
goalObserverTickActive  int32
goalObserverTickPending int32

// arranque, junto a runtime_v0.go:130
runtime.runAsyncWorkV0("goal_observer_loop", func() { runtime.runGoalObservationLoopV0(runCtx) })

// firmas (espejo de runSupervisorLoopV0 / runSupervisorTickAsyncV0 / runSupervisorTickV0)
func (runtime *RuntimeV0) runGoalObservationLoopV0(ctx context.Context)
func (runtime *RuntimeV0) runGoalObservationTickAsyncV0(ctx context.Context) bool
func (runtime *RuntimeV0) runGoalObservationTickV0(ctx context.Context)
```
Dentro del tick:
```go
observer, ok := runtime.supervisor.(GoalActiveObservationPortV0)
if !ok || observer == nil || !runtime.config.GoalObserverEnabled { return }
res, err := observer.ObserveActiveGoalWorksV0(ctx, orquestagoal.GoalWorkObserveActiveRequestV0{
    List: orquestagoal.GoalWorkStateListRequestV0{ActiveOnly: true},
})
// persistir transición por cada observación terminal (persistStateTransitionV0)
```

### T1 — config y flag
```go
// modulos/orquesta-server/config_v0.go (ConfigV0)
GoalObserverEnabled  bool          // default true en NormalizeConfigV0
GoalObserverInterval time.Duration // default p.ej. 3s; validar > 0

// cmd/orquesta-server/server_env_autonomy_v0.go (patrón de serverResidentDirectorEnabledFromEnvV0)
const envServerGoalObserverEnabledV0 = "ORQUESTA_SERVER_GOAL_OBSERVER_ENABLED" // default true
```
Gating: **NO** usar `ResidentDirectorEnabled`. El loop arranca como el supervisor
(incondicional) y solo lo apaga `GoalObserverEnabled=false`.

### T2 — función pura de liveness (donde vive `RunDrainDiagnosticV0`)
```go
// modulos/orquesta-run-coordinator (junto a RunDrainDiagnosticV0, contracts_v0.go:78)
type RunLivenessV0 string
const (
    RunLivenessLiveV0       RunLivenessV0 = "running_live"
    RunLivenessStaleV0      RunLivenessV0 = "running_stale"
    RunLivenessUnverifiedV0 RunLivenessV0 = "running_unverified"
)
// processObserved = el ProcessRegistry resolvió un proceso vivo para la run
func ClassifyRunLivenessV0(status string, diagnostics []RunDrainDiagnosticV0, processObserved bool) RunLivenessV0
```
Reusar la lógica que ya existe en `supervisor_loop_v0.go`
(`supervisorDiagnosticsHaveLiveProcessV0:382`,
`supervisorResultHasUnverifiedRunningV0:164`) extrayéndola a esta función pura, y
**eliminar** la copia muerta `hasMCPAutoprogrammingLiveProcessSignalV0`
(`autoprogramming_status_health_v0.go:230`).

### T2 — fuente de verdad de proceso vivo (ya existe)
```go
// interfaz del registry (modulos/orquesta-orchestration-core), ya usada en
// run_coordinator_launch_outbox_recovery_v0.go:237 y process_agent_stopper.go:41
ResolveAgentProcessV0(ctx context.Context, runRef string, agentRef string) (AgentProcessRecordV0, error)
```
`status` debe poblar `processObserved` consultando este registry y mapear a
`ClassifyRunLivenessV0`.

## Comprobación final
```bash
go build ./... && go vet ./...
go test ./... && go test -race ./modulos/orquesta-server/...
```
