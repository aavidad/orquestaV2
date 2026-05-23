# Autoprogramacion Orquesta: cola pendiente 2026-05-23

Este documento alimenta la cola de autoprogramacion del servidor Orquesta.
Cada agente debe leer `AGENTS.md`, el `AGENTS.md` local de su modulo y este
documento antes de editar.

## Reglas globales

- Orquesta nucleo no importa CLI, web, MCP, Codex, OPES, DB concreta, HOME,
  tokens, OAuth ni paths locales.
- CLI, web, API, MCP, Codex y OPES son adaptadores/composiciones.
- Mantener hexagonal puro: contratos, puertos y refs opacas en nucleo.
- No borrar codigo, docs ni tests sin revisar referencias y dejar evidencia.
- Cambios pequenos, con test focal antes de ampliar.
- Si un agente devuelve nombres o refs cercanos pero no exactos, preferir
  normalizar/corregir en director/adaptador antes que tirar todo el trabajo.
- Dar manga ancha a trabajos largos: no cortar por timeouts estrechos; detectar
  bucles por falta real de progreso.
- Si se abre un rail para desbloquear ejecucion real, documentarlo aqui como
  apertura intencional y pendiente de revision futura. Por defecto se abre para
  que Orquesta funcione; despues se estrecha con evidencia, no por suposicion.
- Las aperturas de rail son deuda viva de automejora: el director puede
  convertirlas en tareas de fondo cuando detecte poca carga, ejecutarlas con
  agentes y cerrarlas solo con pruebas reales.
- Si una ejecucion real queda parada por un rail de forma, vocabulario,
  normalizacion o validacion demasiado estrecha, la politica vigente es abrir
  el rail para que Orquesta funcione, registrar la apertura aqui y dejar su
  cierre fino como automejora futura. No tirar trabajos completos por fallos
  reparables de interpretacion.
- Los cortes fuertes se reservan para seguridad real demostrada, causalidad
  rota, refs imposibles, datos sensibles efectivos o efectos externos no
  autorizados; no para listas de palabras o equivalencias que el director pueda
  normalizar o reparar.
- Los errores de rail se registran como casos acumulables en
  `docs/rail_errors_observados_2026-05-23.md`. Antes de recompilar servidor o
  probar Orquesta completa, ejecutar la matriz rapida `./scripts/test_rails_fast.sh`
  y anadir ahi cada nuevo caso observado para pasar todos de golpe.
- Mejora futura de seguridad: antes de enviar contexto a agentes premium o
  remotos, una IA local o sanitizador local inyectado por puerto debe poder
  revisar y limpiar datos sensibles, claves, tokens, secretos, rutas privadas y
  material no publicable. El nucleo solo debe conocer refs opacas y politica
  neutral; proveedor/modelo local y transporte quedan en adaptador/composicion.
- Las instrucciones largas se compactan antes de `WorkflowTaskV0`; el detalle
  amplio debe viajar por `context_refs`/docs/artefactos, no como payload durable
  gigante. Si el core rechaza una forma reparable, el issue debe conservar el
  subcampo causal para que el director cree followup o normalizacion.
- Comunicacion compacta: usar `$caveman full` si esta disponible, o equivalente.
- ACK valido solo con archivos reales tocados y pruebas ejecutadas.

## Aperturas de rail pendientes de revision futura

### R01 concurrency-gate-detalle-prohibido

Fecha: 2026-05-23.

Motivo: `RecordConcurrencyGate` estaba reutilizando la lista estricta de
reviews y bloqueaba metadata operativa razonable (`codex`, `runtime`, `git`,
`provider`, `model`, `adapter`, `filesystem`). Ese bloqueo impedia que el
director arrancara agentes en autoprogramacion residente.

Apertura aplicada: el gate permite refs opacas de adaptador/ejecucion/pruebas.
El 2026-05-23 se retiro tambien el filtro global por palabras en
`RecordConcurrencyGate` y su evento `ConcurrencyGateRecorded`, porque bloqueaba
falsos positivos como `secrets_policy` antes de que el director pudiera lanzar
agentes y reparar.
Tambien se excluyen `concurrency_gates` y `command_effects` del filtro global
por palabras en `ValidateOrchestrationRunV0`: son proyecciones/efectos
estructurados con refs opacas y hashes, y seguian bloqueando refs legitimas como
`request-ref-app-completion-loop`.

Pendiente de revision futura: cuando haya mas ejecuciones reales, estudiar si
conviene reintroducir una guarda de datos sensibles por campo o por clasificador
semantico, no por lista global de palabras. La revision debe conservar
tolerancia a refs opacas y cortar solo seguridad real, causalidad, refs
imposibles, datos sensibles efectivos o efectos externos no autorizados. Esta
revision debe entrar como tarea de automejora de baja prioridad y puede
ejecutarse en momentos de poca carga del servidor residente.

### R02 workflow-task-detalle-prohibido

Fecha: 2026-05-23.

Motivo: `WorkflowTaskV0` bloqueaba instrucciones y refs operativas por palabras
como `runtime`, `provider`, `db`, `sql`, `oauth`, `docker`, `tmux` o `token`
aunque fueran contexto opaco o presupuesto, impidiendo crear tareas reales para
autoprogramacion.

Apertura aplicada: `WorkflowTaskV0` permite refs/texto opacos de adaptador,
ejecucion o pruebas y mantiene corte fuerte para secretos o credenciales
(`secret`, `password`, `credential`, `api_key`, tokens concretos y variantes
equivalentes). La validacion estructural de refs, phase, write-set, criterios,
linaje, payload compacto y causalidad se conserva.

Pendiente de revision futura: estudiar validacion por campo y clasificacion de
refs sensibles en vez de bloqueo por palabras globales. Debe entrar como tarea
de automejora de baja prioridad cuando haya ejecuciones reales suficientes.

## Cierres aplicados 2026-05-23

- Autoprogramacion residente no se da por terminada tras una pasada global con
  ejecucion real; exige una pasada posterior sin ejecuciones para cerrar la
  llamada.
- `PrepareAutoprogrammingRunV0` marca las `WorkflowTaskV0` con
  `operational_director.task_source:autoprogramming` como `context_ref` opaca.
- `orquesta-app-director-service` acepta ese marker en `context_refs` y puede
  sembrar `OperationalDirectorPlanState` desde tareas persistidas sin meter
  Codex ni producto en el nucleo.
- El bridge de autoprogramacion devuelve `operational_director_plan_ref` en el
  `Continue` cuando hay store/writer inyectados; asi el ciclo puede reentrar por
  wait, review, tests requeridos, replan/cierre.
- La cola no marca una run autoprogramming entregada como terminal si quedan
  tareas abiertas pendientes de revision/cierre formal.
- La decision source de composicion abre `revision` cuando todas las entregas
  autoprogramming estan listas y no hay agentes externos pendientes.
- Verificado con:
  `go test -count=1 ./modulos/orquesta-app-codex-stack`,
  `go test -count=1 ./modulos/orquesta-app-director-service` y focales
  `TestAutoprogrammingDirectorDecisionSourceV0`,
  `TestStackDrainQueueStatus...`,
  `TestPrepareAutoprogrammingRunV0Persiste...`,
  `TestAutoprogrammingResidentModeV0`.
- Smoke temporal verificado:
  `ORQUESTA_AUTOPROGRAMMING_SUPERVISED_SMOKE_CONFIRM=1 SMOKE_ID=post-planstate-20260523 ORQUESTA_KEEP_SMOKE_DIR=0 ./scripts/smoke_autoprogramming_supervised.sh`.
  Resultado: servidor temporal, Codex fake, un agente arrancado por supervisor
  residente, replay idempotente OK, `external-work/run` OK,
  `codex_real_executed=false`, `opes_touched=false`.

## T01 server-autonomy

Objetivo: reforzar el servidor residente para autoprogramacion desatendida.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-server`

Criterios:

- El loop de supervisor debe seguir funcionando aunque un tick falle; registrar
  error durable pero permitir ticks posteriores.
- Exponer/registrar suficientes numeros para saber cola, ejecuciones, skips,
  ultimo error y ultimo tick sin depender del operador.
- No meter conocimiento de Codex ni de tareas de producto en `orquesta-server`.
- Tests: `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server`.

## T02 director-self-repair

Objetivo: mejorar la capacidad del director/stack para corregir desviaciones
pequenas de agentes sin descartar trabajos completos.

Alcance:

- `modulos/orquesta-app-codex-stack`

Criterios:

- Normalizar pequenos desajustes de entregas: aliases de rutas, refs derivables,
  nombres equivalentes y tests declarados con diferencias inocuas.
- Si falta algo real, pedir rework acotado; no rehacer todo salvo que sea mas
  barato y quede justificado.
- Limitar reintentos por task para evitar bucles, pero con margen suficiente.
- Tests: `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'Review|Rework|Repair|Autoprogramming|Drain'`.

## T03 replay-state

Objetivo: cerrar huecos de replay/idempotencia de metadata viva del Director.

Alcance:

- `modulos/orquesta-state-file`
- `modulos/orquesta-orchestration-core`

Criterios:

- Restaurar o rematerializar de forma verificable `WorkflowTaskStore`,
  `WorkflowTaskWaitStateV0` y `OperationalDirectorPlanStateV0`.
- No reconstruir desde eventos compactos si faltan metadatos que solo viven en
  stores; documentar y probar la frontera.
- Tests: `go test -count=1 ./modulos/orquesta-state-file ./modulos/orquesta-orchestration-core`.

## T04 api-mcp-autoprogramming

Objetivo: completar la superficie API/MCP de gestion de autoprogramacion.

Alcance:

- `modulos/orquesta-mcp`
- `modulos/orquesta-app-gateway`
- `modulos/orquesta-http-gateway`

Criterios:

- Endpoints/herramientas finas para estado de cola/run, supervision puntual y
  diagnostico de autoprogramacion.
- No ejecutar logica de negocio en HTTP/MCP; solo adaptar a puertos/casos de uso.
- i18n/errores publicos coherentes con los modulos existentes.
- Tests: `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./modulos/orquesta-http-gateway`.

## T05 web-autoprogramming

Objetivo: mejorar la web como cliente fino de autoprogramacion.

Alcance:

- `modulos/orquesta-web`

Criterios:

- Vista/modelos para cola, runs, agentes, errores publicos y progreso.
- La web no accede a stores ni a runtime; consume endpoints/cliente inyectado.
- No meter textos de instrucciones visibles en la UI; i18n ES/EN si hay textos.
- Tests: `go test -count=1 ./modulos/orquesta-web`.

## T06 cli-thin-client

Objetivo: completar CLI como cliente fino del servidor, sin entrar en nucleo.

Alcance:

- `cmd/orquesta-cli`
- `modulos/orquesta-cli`

Criterios:

- Comandos para ver estado de servidor, cola y runs de autoprogramacion usando
  HTTP/API cuando proceda.
- Sin acceso directo a stores internos del nucleo ni imports de producto en core.
- Help ES/EN y rechazo de argumentos sobrantes.
- Tests: `go test -count=1 ./cmd/orquesta-cli ./modulos/orquesta-cli`.

## T07 opes-consumer-smoke

Objetivo: cerrar el smoke OPES aislado pendiente como consumidor, no como nucleo.

Alcance:

- `scripts`
- `docs/runbooks`
- `modulos/orquesta-opes-bridge`
- `modulos/orquesta-opes-connector`

Criterios:

- Smoke temporal aislado para derivados/cierre OPES sin tocar OPES productivo.
- Guardas de confirmacion y filtro por tipo de job.
- Documentar evidencia y limites; no drenar colas amplias.
- Tests: `go test -count=1 ./modulos/orquesta-opes-bridge ./modulos/orquesta-opes-connector`.

## T08 runtime-neutral-e2e

Objetivo: probar runtime externo no-Codex a nivel contrato neutral.

Alcance:

- `modulos/orquesta-runtime`
- `modulos/orquesta-runtime-worktree`

Criterios:

- E2E fake/contractual para launch/progress/stop sin provider concreto.
- El nucleo no debe saber de Codex ni CLI real; usar refs opacas y puertos.
- Tests: `go test -count=1 ./modulos/orquesta-runtime ./modulos/orquesta-runtime-worktree`.

## T09 local-sensitive-data-sanitizer

Objetivo: anadir una frontera de saneamiento local antes de mandar contexto a
agentes premium/remotos.

Alcance:

- `modulos/orquesta-context`
- `modulos/orquesta-runtime`
- `modulos/orquesta-app-codex-stack`
- `docs/runbooks`

Criterios:

- Definir puerto neutral para sanitizar contexto saliente sin meter IA,
  proveedor, modelo ni transporte en el nucleo.
- Implementar adaptador opt-in para IA local o sanitizador local que detecte y
  sustituya claves, tokens, secretos, rutas privadas y material no publicable
  por refs opacas.
- Conservar evidencia durable de saneamiento sin persistir el dato sensible.
- Si el sanitizador duda, pedir revision humana/director o enviar contexto
  minimo por refs, no bloquear toda la ejecucion salvo riesgo efectivo.
- Tests: `go test -count=1 ./modulos/orquesta-context ./modulos/orquesta-runtime ./modulos/orquesta-app-codex-stack`.

## T10 flaky-tests-observability

Objetivo: tratar fallos intermitentes como prioridad de fiabilidad, no como
ruido.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-runtime`
- `modulos/orquesta-app-codex-stack`
- `docs/runbooks`

Criterios:

- Registrar tests intermitentes con fecha, comando, fallo exacto y reintento
  posterior.
- Aislar fuentes de no determinismo: orden de mapas, tiempos, concurrencia,
  ficheros temporales, salida stdout/stderr y estado compartido.
- Anadir harness de repeticion acotado para tests criticos antes de marcarlos
  como fiables.
- Caso observado 2026-05-23: `go test -count=1 ./...` fallo una vez en
  `cmd/orquesta-server` con
  `TestCodexLaunchDirectorWaveCommandV0RecursiveFakeRuntimeEjecutableConLinaje`
  por stdout sin prompt esperado; el test focal y `go test -count=1
  ./cmd/orquesta-server` pasaron al reintentar. Debe investigarse como flake.
- Tests: `go test -count=1 ./cmd/orquesta-server` y repeticion focal del caso
  observado.

## Verificacion final esperada

- `git diff --check`
- `go test -count=1 ./...`
- Prueba HTTP real con servidor residente: preparar run, supervisar, revisar
  stats y shutdown limpio sin 500.
