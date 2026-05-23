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
- Comunicacion compacta: usar `$caveman full` si esta disponible, o equivalente.
- ACK valido solo con archivos reales tocados y pruebas ejecutadas.

## Aperturas de rail pendientes de revision futura

### R01 concurrency-gate-detalle-prohibido

Fecha: 2026-05-23.

Motivo: `RecordConcurrencyGate` estaba reutilizando la lista estricta de
reviews y bloqueaba metadata operativa razonable (`codex`, `runtime`, `git`,
`provider`, `model`, `adapter`, `filesystem`). Ese bloqueo impedia que el
director arrancara agentes en autoprogramacion residente.

Apertura aplicada: el gate permite refs opacas de adaptador/ejecucion/pruebas y
mantiene corte fuerte para secretos o credenciales (`secret`, `password`,
`credential`, `api_key`, tokens concretos y variantes equivalentes).

Pendiente de revision futura: cuando haya mas ejecuciones reales, estudiar si
conviene estrechar por campos concretos en vez de por palabras globales. La
revision debe conservar tolerancia a refs opacas y cortar solo seguridad,
causalidad, refs imposibles, datos sensibles o efectos externos no autorizados.
Esta revision debe entrar como tarea de automejora de baja prioridad y puede
ejecutarse en momentos de poca carga del servidor residente.

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

## Verificacion final esperada

- `git diff --check`
- `go test -count=1 ./...`
- Prueba HTTP real con servidor residente: preparar run, supervisar, revisar
  stats y shutdown limpio sin 500.
