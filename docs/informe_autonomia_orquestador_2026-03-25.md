<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# Informe de autonomía del orquestador

Fecha: `2026-03-25`
Revisión: `Codex3`

## Objetivo

Determinar qué falta para que Orquesta pueda hacerse cargo de los agentes de forma realmente autónoma:

- arrancarlos y retomarlos
- vigilar su salud
- transferir trabajo entre ellos
- aplicar política de presupuesto y handoff
- operar sin depender de scripts manuales ni de operador humano como camino principal

## Resumen ejecutivo

El orquestador ya controla bien el plano de control, pero todavía no cierra el ciclo completo de autonomía.

Estado real actual:

- sí existe daemon con runner operativo y lazo de control
- sí existen `runtime_orders`, `runtime_mailbox`, `runtime_checkpoints` y `handoff`
- sí existe control real de proceso para conectores CLI locales
- sí existe watchdog con sondeo y handoff automático por heartbeat obsoleto
- sí existe autoarranque por carga de trabajo cuando el agente tiene tarea asignada y no hay sesión/continuidad viva
- sí existe reactivación automática tras reanimación de cuota, con `resume` si había runtime pausado y `start` si había trabajo sin runtime vivo
- sí existe handoff preventivo por presupuesto de sesión con validación de frescura del snapshot
- sí existe bootstrap de continuidad al arrancar o retomar
- sí existe consumo periódico en el daemon para todas las `runtime_orders` despachables, no solo para órdenes básicas
- sí existe estado operativo explícito de proyecto (`activo`, `esperando_humano`, `bloqueado_externo`, `cerrado`) y auto-reactivación al detectarse desbloqueo
- sí existe retirada segura de agentes con liberación automática del trabajo y sustitución replanificable por otro agente disponible
- sí existe resolución efectiva unificada del catálogo de gobernanza (reglas, skills, workflows) y conservación de su identidad en continuidad/resume
- sí existe notificación `governance_refresh` por mailbox cuando cambian reglas o workflows del rol
- `stop` ya cierra también la sesión viva, no solo runtime y handle
- el adaptador remoto ya tiene política configurable de reintentos para acciones de control seguras, sin reintentar `start` ni `send_instruction`
- `sync_status` ya puede observar estado remoto real por `status_path`, normalizarlo a estado canónico del handle y conservar el estado remoto crudo en metadata
- la unidad `systemd` oficial del daemon ya incorpora endurecimiento base de servicio

Lo que todavía falta para hablar de autonomía completa:

- cerrar el lazo autónomo de decisión del agente más allá de `pause`, `resume`, `start` y `nudge`
- retirada de scripts manuales como vía principal de operación
- seguir endureciendo observabilidad y recuperación avanzada del adaptador remoto

Conclusión:

- Orquesta ya es un control plane funcional
- todavía no es un orquestador plenamente autónomo de agentes de extremo a extremo

## Qué ya está cubierto

### 1. Daemon y lazo de control

El daemon oficial ya existe y arranca el runner del plano de control:

- `serve` y `server run` convergen en `arrancarServidorUnificado(...)` en `cmd/serve.go` y `cmd/servidor_unificado.go`
- el runner se crea en `cmd/controlplane_support.go`
- el lazo operativo ejecuta reanimación, salud, planificación, runtime orders, refinería y handoffs en `planocontrol/runner.go`

Esto ya cubre el núcleo de “single process / single writer” para el gobierno del sistema.

### 2. Control plane persistente

La base de control ya está modelada y operativa:

- `runtime_orders`
- `runtime_handles`
- `runtime_mailbox`
- `runtime_checkpoints`

La ejecución de órdenes vive en `db/controlplane_entities.go` y cubre:

- `start`
- `pause`
- `resume`
- `stop`
- `restart`
- `send_instruction`
- `sync_status`
- `checkpoint`
- `nudge`
- `discordia`
- `handoff`

Además, el batch periódico del daemon ya despacha esas órdenes compatibles desde `ProcesarRuntimeOrdersBatch()`; ya no queda limitado a `sync_status`, `checkpoint`, `nudge` y `discordia`.

### 2.b Estado operativo de proyecto y retorno automático

Ya existe una capa explícita de operación por proyecto:

- `db/proyectos_operacion.go` persiste el estado operativo del proyecto y su política básica
- `cmd/controlplane_support.go` marca `esperando_humano` cuando el agente queda bloqueado y necesita intervención
- `db/planificador.go` ya no replanifica proyectos no operables y reactiva automáticamente un proyecto pausado cuando detecta que el bloqueo ha sido resuelto

Con esto el flujo deja de ser solo local al agente:

1. tarea bloqueada sin trabajo real restante
2. proyecto pasa a `esperando_humano`
3. sesión/asignación se aparcan
4. al desbloquear la tarea, el planificador reactiva el proyecto y devuelve al agente al frente pausado

### 2.c Retirada segura y sustitución automática de agentes

La retirada de un agente ya no deja estados muertos:

- `db/sesiones.go` pausa sus asignaciones activas/planificadas, libera tareas `asignada` o `en_progreso` y cierra la sesión sin bloquear el scheduler
- `agentesapp/service.go` encola una `pause` al control plane si detecta un `runtime_handle` activo antes de retirar al agente
- `db/planificador.go` puede recolocar automáticamente ese trabajo en otro agente disponible porque la tarea vuelve al pool y la afinidad previa queda aparcada

Con esto el flujo ya soporta:

1. retirar o deshabilitar un agente
2. aparcar su runtime vivo cuando existe control real
3. dejar su frente en estado replanificable
4. reasignar el trabajo automáticamente a otro agente del pool

### 3. Bootstrap y continuidad

El agente puede arrancar con continuidad real:

- el bootstrap integra orden, mailbox y checkpoint en `internal/bootstrapruntime/bootstrap.go`
- el arranque de órdenes `start` prepara `ResumeContext`, `LaunchPlan` y continuidad en `db/controlplane_entities.go`
- el endpoint `/api/agente/preparar` y el bundle asociado ya soportan ese contexto

Además, la gobernanza efectiva ya no se resuelve por duplicado en varias capas:

- `db/reglas.go` expone `ResolveGovernanceCatalog(...)` como resolución única actual del catálogo efectivo
- `sesionesapp/service.go` y `agentesapp/runtime_service.go` reutilizan esa resolución para briefing, start-context y prepare
- `db/controlplane_entities.go` añade el `hash` del catálogo efectivo al `resume payload` cuando existe continuidad, de forma que `start/resume/handoff` conservan también identidad de gobernanza y no solo contexto operativo
- `db/governance_runtime_refresh.go` notifica por `runtime_mailbox` a los agentes activos del rol cuando cambia una regla o un workflow, reusando el patrón ya existente de `skills_refresh`

### 4. Watchdog e handoff por inactividad

Ya existe relevo automático por heartbeat obsoleto:

- detección y sondeo previo en `db/handoff_manager.go`
- handoff batch desde el runner en `planocontrol/runner.go`
- cobertura E2E con proceso vivo en `planocontrol/runner_e2e_test.go`

### 5. Control real de proceso y control remoto mínimo

Para conectores CLI locales, Orquesta ya puede gobernar el proceso vivo:

- arranque en `internal/controlruntime/arranque.go`
- pausa, continuar y detener en `internal/controlruntime/proceso.go`
- inyección de instrucciones vía FIFO en `internal/controlruntime/arranque.go`

Además, ya existe un adaptador remoto mínimo operativo:

- `runtimeagente/driver.go` genera configuración remota estructurada para conectores `api`, `mcp_http` y `otro`
- `internal/controlruntime/remoto.go` ejecuta `start`, `resume`, `pause`, `continue`, `stop` e `input` por HTTP
- `internal/controlruntime/remoto.go` también consulta estado remoto por HTTP cuando el conector declara `status_path`
- `db/controlplane_entities.go` persiste correctamente sesiones, runtimes y handles remotos sin depender de PID local
- `db/controlplane_entities.go` ya usa `sync_status` para observar estado remoto, actualizar runtime y enriquecer metadata del handle sin romper el enum local de estados
- `runtime_orders` ya gobierna de extremo a extremo `start`, `send_instruction`, `pause`, `resume` y `stop` para handles remotos

Límite actual:

- el adaptador remoto ya no es declarativo, pero sigue siendo una primera versión HTTP
- ya soporta autenticación por cabecera/token de entorno, timeout configurable y reintentos seguros para `pause`/`resume`/`stop`
- `start` y `send_instruction` siguen sin reintento automático por seguridad frente a duplicados
- ya existe observabilidad remota básica por conector vía `status_path`
- faltan todavía polling/health más ricos, políticas de recuperación y métricas específicas por conector

## Contraste con referencias externas

Revisión contrastada el `2026-03-25` con repos públicos ya usados como referencia en el proyecto:

- OpenClaw (`github.com/openclaw/openclaw`): refuerza la idea de un gateway único como control plane de sesiones, presencia, UI de control y A2UI. Orquesta ya está bien orientada en `serve`/daemon único, pero todavía tiene menos adaptadores de runtime y menos canalización operativa remota.
- LangGraph (`github.com/langchain-ai/langgraph`): destaca por ejecución duradera y reanudación tras fallo. Confirma que el camino correcto para Orquesta es seguir profundizando en checkpoint, resume y estado persistente, no en bucles efímeros sin memoria.
- Swarm (`github.com/openai/swarm`): es útil como modelo lógico de handoff y routing ligero, pero al ser esencialmente cliente y sin estado persistido entre llamadas no sirve como referencia suficiente para el runtime autónomo que queremos en Orquesta.

Conclusión comparativa:

- Orquesta ya está más cerca de un control plane persistente tipo gateway que de un framework ligero tipo Swarm.
- El siguiente salto no es “más handoff conceptual”, sino mejor ejecución duradera, conectores remotos y retirada del camino manual.

## Lo que falta

## Crítico

### 1. Arranque automático por demanda de trabajo

Esto ya ha quedado cubierto en `db/planificador.go`.

El planificador ahora:

1. detecta trabajo arrancable del agente en su proyecto activo
2. deduplica continuidad ya pendiente (`start`, `resume`, `handoff`)
3. no invade sesiones ni handles activos de otros proyectos del mismo agente
4. encola `start` automáticamente con auditoría

Además, el ciclo ya no aborta la pasada completa si falla un agente concreto; audita el error y sigue con los demás.

### 2. Lazo autónomo del agente ya existe de forma mínima, pero no está completo

El `tick` ya no es solo observación pasiva.

Evidencia:

- `apiHandlerAgenteTick` en `cmd/api.go` actualiza heartbeat y devuelve `agenteTickOutput`
- `construirAgenteTickOutput` en `cmd/agente.go` calcula `AccionRecomendada`
- `cmd/controlplane_support.go` ya ejecuta un batch autónomo del daemon que evalúa sesiones activas
- ese batch traduce recomendaciones duras en acciones automáticas trazables:
  - `pause` si la recomendación es `pausar_por_cuota`
  - `pause` si la recomendación es `pausar_y_reasignar`
  - `nudge` si la recomendación es `votar_propuestas_pendientes`
  - `nudge` si la recomendación es `pedir_intervencion`
  - `nudge` si la recomendación es `continuar_trabajo`
  - `nudge` si la recomendación es `esperar_o_pedir_tarea`
- el ciclo de reanimación del runner ya no se limita a limpiar el enfriamiento:
  - si el agente tenía handle pausado, encola `resume`
  - si tenía trabajo arrancable pero no runtime vivo, encola `start`

Límite real:

- el daemon ya decide y ejecuta más acciones por sí solo, pero no ejecuta todavía un loop integral de trabajo del agente
- `continuar_trabajo` y `esperar_o_pedir_tarea` siguen siendo semánticas de coordinación apoyadas en `nudge`, no automatismo completo del runtime
- la telemetría del agente sigue entrando principalmente por cliente externo que hace `tick`

Para cerrar esto hace falta:

1. ampliar el batch autónomo a más decisiones seguras
2. separar mejor “telemetría del agente” de “decisión del orquestador”
3. definir un loop oficial de runtime gestionado por el daemon para el caso no local

### 3. Presupuesto de sesión integrado en el handoff automático

Esto ya no está pendiente.

`db/handoff_manager.go` ya evalúa:

- heartbeat obsoleto como disparador `watchdog`
- presupuesto crítico como disparador `presupuesto`

El handoff preventivo por presupuesto:

1. usa el último snapshot de `presupuestos_sesion`
2. exige frescura mínima del snapshot para no disparar con telemetría vieja
3. no requiere sondeo previo `sync_status`/`nudge`
4. entra de forma natural en `ProcesarHandoffsBatch()` y, por tanto, en el `Runner`
5. selecciona de forma determinista la tarea viva del agente, priorizando el proyecto de la sesión activa si existen varias `en_progreso`

La cobertura ya existe tanto a nivel `db` como en `planocontrol/runner_e2e_test.go`.

### 4. El adaptador remoto ya existe, pero necesita endurecimiento

Esto ya no está pendiente como ausencia funcional.

La situación real ahora es:

- los conectores remotos ya no se quedan en `LaunchPlan` declarativo
- existe un adaptador HTTP mínimo en `internal/controlruntime/remoto.go`
- el control plane ya puede arrancar y controlar runtimes remotos sin inventar PIDs locales

Lo que sí sigue pendiente:

- definir recuperación aún más rica del adaptador
- ampliar el sondeo de salud remoto más allá del `sync_status` puntual por `status_path`

## Importante

### 5. Los scripts siguen siendo compatibilidad operativa, pero aún no han desaparecido del todo

Siguen existiendo scripts claramente manuales:

- `scripts/inicio_agente.sh`
- `scripts/agente_console.sh`
- `scripts/runtime_connector.sh`

Situación actual:

- `runtime_connector.sh` ya está bastante más limpio
- `agente_console.sh` ya no inspecciona estados locales laterales
- el modelo de esos scripts sigue siendo “abrir consola del agente y guardar sesión al salir”
- la vía oficial del sistema ya es daemon + API + `runtime_orders`; los scripts quedan como operación manual y rescate

Y `scripts/inicio_agente.sh` sigue describiéndose como:

- arranque manual base de un agente
- decidir tarea clara
- iniciar tarea si procede

Eso es útil como compatibilidad, pero no encaja con una operación plenamente autónoma.

Objetivo correcto:

- los scripts deben quedar como herramienta humana de rescate o bootstrap
- la vía normal debe ser daemon/API/control plane

### 6. El watchdog es bueno, pero aún reacciona más que dirige

El watchdog ya:

- detecta stale heartbeat
- hace sondeo
- nudgea
- dispara handoff si procede

Pero le falta cerrar políticas de gobierno más ricas:

- reinicio automático controlado según tipo de error
- escalado por repetición de fallos
- circuit breaker por agente/conector
- política de “cuántos intentos antes de relevo”

Hoy el sistema recupera bien algunos casos, pero todavía no tiene estrategia completa de operación continua.

### 7. Falta formalizar el modo oficial de producción

El daemon ya está unificado en código, pero para autonomía real aún conviene cerrar operación:

- unidad `systemd`
- política de restart del servicio
- healthcheck operativo oficial
- runbook de recuperación de runtime colgado
- política de limpieza de FIFOs/logs temporales de `internal/controlruntime`

No es el hueco más conceptual, pero sí afecta a la autonomía en entorno real.

## Menor, pero conviene cerrar

### 8. Persisten caminos de fallback local

Existen rutas de fallback/local opt-in en CLI:

- `--local`
- `--local`
- variables de entorno equivalentes

Esto es razonable para recuperación, pero para operación autónoma final conviene:

- reservarlo solo a soporte/doctor
- no dejar ambigüedad sobre cuál es el camino de verdad

## Orden recomendado

### Corte 1

- autoarranque por demanda de trabajo
- deduplicación de `start`
- auditoría de arranque automático

### Corte 2

- integrar presupuesto de sesión en `ProcesarHandoffsBatch()`
- handoff preventivo real por cuota/tiempo restante

### Corte 3

- definir worker oficial del agente gobernado por Orquesta
- convertir `tick` de señal informativa a parte de un ciclo autónomo

### Corte 4

- endurecer el adaptador remoto mínimo
- no limitar la autonomía a un contrato HTTP básico

### Corte 5

- rebajar scripts a compatibilidad/rescate
- formalizar operación de producción del daemon

## Criterio de cierre

Podremos decir que el orquestador se hace cargo de los agentes de forma autónoma cuando se cumplan estas cinco condiciones:

1. una tarea asignada puede hacer arrancar automáticamente al agente adecuado sin intervención humana
2. el orquestador decide y ejecuta pausas, reanudaciones, handoffs y reinicios según política, no solo por orden manual
3. el presupuesto de sesión participa de verdad en la automatización del relevo
4. el sistema funciona igual con conectores locales y remotos soportados
5. los scripts dejan de ser camino operativo principal

## Veredicto

Estado actual:

- autonomía del control plane: alta
- autonomía del runtime local CLI: alta
- autonomía del runtime remoto soportado: media
- autonomía integral del sistema de agentes: media

Lo que falta ya no es “crear la base”, sino cerrar el ciclo de gobierno:

- arrancar solo
- decidir solo
- relevar solo
- funcionar igual sin operador y sin scripts como pieza central
