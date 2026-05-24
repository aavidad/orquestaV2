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
