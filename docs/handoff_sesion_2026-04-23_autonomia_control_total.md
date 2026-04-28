# Handoff de sesión 2026-04-23

## Estado cerrado en esta sesión

- Se documentó el estado real de autonomía y control total en:
  - `ARQUITECTURA.md`
  - `docs/BIBLIA_APP_ORQUESTA.md`
  - `docs/op_096_control_total_estado_proyecto_y_estadisticas.md`
  - `docs/db_runtime_inventory.md`
  - `docs/propuesta_adk_eventos_delta_artifacts_rewind_2026-04-23.md`
- Se sembraron tareas reales en Orquesta:
  - `#32` `AutonomyEvent y state_delta minimo en handoff y recovery`
  - `#33` `Reducir sin_clasificar y usar progreso semantico real en estado operativo`
  - `#34` `API global de control total de proyecto y estadisticas Git`
  - `#35` `Artifacts versionados para patch, tests y transcript relevante`
- Se endureció `agentesapp/service.go` para:
  - no contar `progress_update`, `tool_result_ok`, `patch_or_code_evidence` ni `ui_noise` como continuidad pendiente del supervisor
  - permitir `semantic progress` reciente sin depender solo de `LastAutonomyState=work_confirmed`
  - reinyectar transcript semántico fuerte reciente en la fila operativa antes de derivar estado

## Delta documental a 2026-04-28

La foto de `2026-04-23` ya necesita cuatro matices explícitos:

- bug de retiro bloqueante:
  - ya hay mitigaciones de control plane para no reactivar retirados, consumir mailbox residual y bloquear tarea huérfana
  - no debe seguir documentándose como hueco base de arquitectura, sino como bug operativo residual a vigilar en superficies de control

- degradación `workers_stuck`:
  - el repo ya refleja restart coordinado, continuidad local por `session_resume`, `repair-helper` y cierre del helper
  - la deuda restante es de tuning y proyección global, no de ausencia de mecanismo

- clasificación de transcript:
  - sigue abierta y debe considerarse el frente técnico con más impacto inmediato en observabilidad pasiva
  - la deuda real es `classification=''` persistente en `runtime_transcript`, no una clase de dominio `sin_clasificar`

- control total/estadísticas:
  - agente y proyecto ya están operativos
  - el cierre pendiente real sigue en la agregación global de timeline, estadísticas y coste

## Validación hecha

- `go test ./agentesapp -run 'Test(WorkerHasRecentSemanticProgressAceptaWorkQueueRecienteSinWorkConfirmedPersistido|WorkerHasRecentSemanticProgressNoAceptaSenalDebilSinHealthOperativa|ApplyRecentSemanticTranscriptSignalsPromueveLastProgressDesdeTranscript|ApplyRecentSemanticTranscriptSignalsIgnoraClasificacionDebil|EffectiveContinuityPendingNoCuentaSupervisorConTranscriptSignalDeProgresoBajoValor)$' -count=1 -timeout 180s`
- `go build -o orquesta .`

## Estado vivo observado al cerrar

- El servidor activo seguía resolviéndose por storage en `http://127.0.0.1:16543`
- `./orquesta status` mostraba:
  - `workers conectados 1`
  - `workers trabajando 1`
  - `supervisores activos 1`
  - tarea viva `#24` en `Codex4`
- No se dejó confirmado un reinicio limpio del daemon con el último binario; el siguiente arranque debe revalidarse explícitamente

## Siguiente ROI real

El siguiente corte con mejor retorno ya está aislado:

1. bajar `classification=''` real en transcript filtrando mejor ruido TUI/bootstrap y mejorando progreso semántico útil
2. convertir `semantic progress` del transcript en estado durable corto, no solo reinyectado en memoria de fila
3. usar esa señal durable en `deriveOperationalState`, `status` y control plane
4. cerrar la proyección global de control total/estadísticas sobre la base ya operativa por agente/proyecto

Archivos candidatos inmediatos:

- `db/runtime_transcript_ingest.go`
- `db/runtime_transcript_store.go`
- `agentesapp/service.go`
- `cmd/status_service.go`

## Riesgos abiertos

- árbol muy sucio; no mezclar commits grandes
- el servidor activo puede no coincidir con el último binario si no se reinicia explícitamente
- `finish_app` amplia sigue viva en la doctrina, pero el frente correcto hoy es trabajar sobre slices acotadas y la tarea `#33`
- no volver a describir `retiro bloqueante` o `workers_stuck` como si no existiera ya tratamiento en el control plane; el hueco real está en señal, visibilidad y cierre global
