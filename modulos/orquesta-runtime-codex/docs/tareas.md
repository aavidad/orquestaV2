# Tareas: orquesta-runtime-codex

## RTCODEX-001 - Resolver Codex opt-in

Objetivo: construir `ProcessRuntimeLaunchRequestV0` para Codex real desde `ExternalAgentLaunchSpecV0`.

Write-set:

- `codex_profile_v0.go`
- `codex_resolver_v0.go`
- `codex_prompt_v0.go`
- `codex_wrapper_v0.go`
- `codex_ack_v0.go`
- `codex_resolver_v0_test.go`

Validacion:

- `go test -count=1 ./modulos/orquesta-runtime-codex`

Bloqueos:

- No ejecuta Codex real sin opt-in externo.
- No resuelve cuotas ni OAuth; solo usa rutas/configuracion explicita.

## RTCODEX-002 - E2E agenda no controlada

Objetivo: consumir este resolver desde `orquesta-e2e` para sustituir procesos controlados por agente externo real.

Bloqueos:

- Requiere variables opt-in del operador.
- Puede consumir cuota real.

## RTCODEX-003 - Control files por agente

Objetivo: permitir ejecucion paralela de varios Codex sobre un mismo
`project_work_dir` sin pisar `agent_packet.json`, `agent_ack.json` ni logs.

Decision:

- `runtime_work_dir` contiene packet, prompt, ACK esperado, stdout, stderr,
  last-message y wrapper.
- `project_work_dir` queda reservado para el codigo/documentacion de la app.
- El prompt usa rutas absolutas de control.

Validacion:

- `go test -count=1 ./modulos/orquesta-runtime-codex`

## RTCODEX-004 - ACK/receipt Codex endurecido

Objetivo: validar el ACK opt-in antes de aceptarlo como receipt operacional.

Write-set:

- `codex_ack_v0.go`
- `codex_ack_validation_v0.go`
- `codex_ack_validation_v0_test.go`
- `codex_prompt_v0.go`
- `docs/contratos.md`
- `docs/pruebas.md`
- `docs/tareas.md`

Validacion:

- `go test -count=1 ./modulos/orquesta-runtime-codex`

Bloqueos:

- ACK `completed` sin artifacts exigidos.
- ACK con HOME real, secretos, OAuth, prompt/completion o transcript completo.
- ACK sin correlacion entre outbox/agent: `request_id`, `correlation_id`,
  `target_module`, `task_ref` y `ack_ref`.

Review posterior:

- Artifacts fuera del write-set se conservan como rail blando si la identidad,
  las rutas seguras y los tests requeridos estan bien; el review gate genera el
  follow-up no bloqueante.

## RTCODEX-005 - decision_path condicionado por contrato

Objetivo: evitar que el prompt base contradiga tareas de director que deben
emitir decisiones ejecutables.

Write-set:

- `codex_prompt_v0.go`
- `codex_resolver_v0_test.go`
- `docs/decisiones.md`
- `docs/pruebas.md`
- `docs/tareas.md`

Validacion:

- `go test -count=1 ./modulos/orquesta-runtime-codex`

Bloqueos:

- `director_decisions.json` sigue fuera del write-set.
- El conector no interpreta decisiones; solo materializa el archivo de control.

## RTCODEX-006 - Reporte de uso redactado

Objetivo: generar evidencia local de uso Codex sin filtrar proveedor, HOME,
OAuth, prompts, transcripts, completions, coste, cuenta real ni rutas privadas.

Estado: hecho para T209.

Validacion:

- `go test -count=1 ./modulos/orquesta-runtime-codex`

Criterios cerrados:

- el wrapper genera `codex_usage_accounting.json` en `runtime_work_dir`;
- los contadores permitidos son `usage.input_tokens`,
  `usage.output_tokens` y `usage.total_tokens`;
- la cuota solo cruza como `quota.status` redactado;
- si no hay cuota real fiable, el status queda `unknown` con
  `quota_observed_unavailable`;
- stdout, stderr y last-message no se convierten en fuente publica de uso.

## RTCODEX-006 - T207 como consumidor de handoff

Objetivo: reconciliar que Codex no implementa rotacion automatica; solo aporta
ACK/handoff/checkpoint que una composicion opt-in puede convertir en
`RuntimeSessionRotationHandoffV0`.

Validacion:

- `go test -count=1 ./modulos/orquesta-runtime-codex`

Bloqueos:

- No relanza Codex real, no mata sesiones y no lee HOME/OAuth/proveedor.
- El smoke real de relevo pertenece a composicion opt-in futura.

## RTCODEX-006 - Reconciliacion T208

Objetivo: conservar la frontera del reparador Codex del guardian como opt-in
break-glass ya gobernado por owners especificos, sin reabrir el umbrella T208.

Validacion:

- `go test -count=1 ./modulos/orquesta-runtime-codex`
- bateria T208 cruzada del paquete OrquestaV2.

Estado: documentado el 2026-05-27; nuevos huecos requieren owner focal.

## RTCODEX-007 - Startup lock obsoleto no supera timeout

Objetivo: evitar que un lock global de arranque Codex vacio y obsoleto bloquee
agentes OPES hasta agotar timeout cuando Orquesta ya puede reaperlo.

Estado: hecho local.

Trabajo aplicado:

- el wrapper calcula `ORQUESTA_CODEX_STARTUP_LOCK_STALE_SECONDS` por defecto a
  partir de `ORQUESTA_CODEX_STARTUP_LOCK_TIMEOUT_SECONDS`;
- si el stale configurado es mayor que el timeout efectivo, se acota al
  timeout;
- un lock vacio antiguo se reapea antes de emitir
  `orquesta_codex_startup_lock_timeout`.

Validacion:

- `go test -count=1 ./modulos/orquesta-runtime-codex -run 'TestCodexWrapperV0(RetiraStartupLockObsoletoVacio|StaleLockPorDefectoNoSuperaTimeout)'`

Criterios cerrados:

- no toca `orquesta-core-workflow`;
- no interpreta salida de Codex;
- no elimina locks recientes;
- corrige autonomia de lanzamiento sin cambiar la politica de serializacion de
  arranque compartido.
