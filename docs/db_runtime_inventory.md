# Inventario DB Runtime

Este documento resume el carril `db/runtime*` y los ficheros cercanos que hoy sostienen bootstrap, mailbox, control plane, snapshots calientes y lecturas `prepare-lite`. El objetivo no es listar todo el paquete `db/`; es dejar claro por donde entrar cuando falla la autonomía o el runtime.

## Mapa rapido

- `start/bootstrap`
  - preparar contexto, resolver lease/orden inicial, construir prompt/resume y registrar handle/runtime
- `send_instruction`
  - decidir delivery, escribir ledger/mailbox, reconciliar receipts y cerrar estados
- `runtime_orders`
  - seleccionar, reclamar, reintentar, backoff y reconciliar orders vivas
- `handles/sesiones`
  - decidir el handle canonico/operativo y su objetivo de proceso real
- `transcript/progreso`
  - ingerir texto, clasificar señales y estimar trabajo confirmado/progreso
- `prepare-lite/read-only`
  - servir bootstrap y contexto minimo aunque el servidor principal no deba abrir toda la persistencia

## Nucleo

- `db/controlplane_entities.go`
  - punto de entrada historico del control plane runtime; hoy concentra el cableado restante y las estructuras base que aun no se han movido a modulos mas finos

- `db/runtime_controlplane_helpers.go`
  - helpers comunes de SQL, placeholders, conversiones ligeras y utilidades compartidas por varios carriles

- `db/runtimes.go`
  - modelo base de `runtime_instances`, eventos y telemetria; es la capa de consulta/persistencia del runtime como entidad durable

## Start y Bootstrap

- `db/runtime_start_control.go`
  - reintentos, control de sesiones ajenas, reencolado y reuso de sesion para arranque

- `db/runtime_start_context.go`
  - carga inicial de agente, proyecto, conector, ultima sesion y resume saneado

- `db/runtime_start_profile.go`
  - resolucion de perfil de ejecucion, modelo y razonamiento

- `db/runtime_start_prepare_helpers.go`
  - helpers de `ConnectorConfig`, resume final y `LaunchPlan`

- `db/runtime_start_prompt.go`
  - inyeccion de prompt de arranque y fallback a mailbox cuando no conviene delivery interactivo directo

- `db/runtime_start_bootstrap.go`
  - construccion del bootstrap/resume y cierre final antes del lanzamiento real

- `db/runtime_start_handle_update.go`
  - sincronizacion de metadata, capabilities y estado del handle tras arrancar el proceso

- `db/runtime_start_helpers.go`
  - helpers auxiliares de payload, transcript y objetivo usados por el carril de `start`

- `db/runtime_bootstrap_controlplane.go`
  - claim de orders de bootstrap para `start`, composicion de resume/checkpoint/mailbox y evidencia de reanudacion por order

- `db/runtime_bootstrap_lease_selection.go`
  - seleccion, resolucion y marcado de leases de bootstrap

- `db/runtime_bootstrap_lease_observation.go`
  - observacion y ack por evidencia de bootstrap lease

- `db/runtime_bootstrap_prompt.go`
  - construccion del prompt compacto de bootstrap y guardas doctrinales para el primer ciclo del worker

- `db/runtime_sync_status.go`
  - `sync_status`, supervision remota/local y observacion de proceso vivo

## Prepare-lite, contexto y lecturas de rescate

- `db/bootstrap_prepare_lite.go`
  - lecturas `read-only` del carril bootstrap: siguiente order inicial, mailbox, checkpoints y orders de `nudge/discordia` sin abrir el camino pesado

- `db/project_context.go`
  - resumen de contexto de proyecto para bootstrap: operacion, worktree activa, tareas del agente y propuestas abiertas

- `db/project_context_prepare_lite.go`
  - variante `prepare-lite` de operacion/tareas/propuestas para no depender del acceso completo en bootstrap

- `db/shared_context.go`
  - items de contexto compartido por proyecto/agente y resumen serializable para inyectarlo en `resume_payload`

- `db/status_readonly.go`
  - snapshot de solo lectura para estado ligero de agentes y tareas cuando hace falta degradar el camino de lectura

- `db/local_legacy_open.go`
  - resolucion de ruta local legacy y guardas de apertura de rescate (`ORQUESTA_FORCE_LOCAL_DB` / `ORQUESTA_FORCE_LOCAL`)

## Send Instruction

- `db/runtime_send_instruction_delivery.go`
  - decisiones de entrega inmediata y tracking del `delivery path`

- `db/runtime_send_instruction_dispatch.go`
  - ledger, mailbox, requeue y degradacion de `send_instruction`

- `db/runtime_send_instruction_state.go`
  - estados finales, completado, diferido y fallos controlados

- `db/runtime_send_instruction_receipt.go`
  - receipts y reconciliacion por receipt

- `db/runtime_send_instruction_reconcile.go`
  - reconciliacion con mailbox actual, bootstrap y sesion obsoleta

- `db/runtime_send_instruction_policy.go`
  - politica de delivery: `tmux`, `session_resume`, `mailbox_only` y evidencia minima exigida

- `db/runtime_send_instruction_session_resume.go`
  - reconciliacion del carril `session_resume` antes de usarlo

- `db/runtime_send_instruction_evidence.go`
  - evidencia premium, transcript, TMUX, patch y `blocked-response`

- `db/runtime_send_instruction_autostop.go`
  - autostop local vinculado a `send_instruction`

## Runtime Orders

- `db/runtime_orders_controlplane.go`
  - operaciones principales sobre `runtime_orders`, resolucion de destino y helpers centrales del dispatcher

- `db/runtime_orders_batch_controlplane.go`
  - seleccion y procesamiento por lotes

- `db/runtime_orders_reconciliation_controlplane.go`
  - reconciliacion de orders abiertas, satisfaccion y continuidad

- `db/runtime_order_stale.go`
  - deteccion de orders stale y follow-up

- `db/runtime_order_provider_backoff.go`
  - backoff por proveedor y errores recuperables

- `db/runtime_order_canonical_target.go`
  - resolucion del runtime/handle/sesion canonicos y refresco de destino

- `db/runtime_order_simple.go`
  - orders pequenas: `checkpoint`, `nudge`, `discordia`, mailbox simple

- `db/runtime_order_types.go`
  - tipos de order y parseo de bootstrap lease desde `resultado_json`

- `db/runtime_orders_hot_index.go`
  - indice caliente de orders vivas por `id`, `estado`, `agente` y `agente+proyecto` para batches frecuentes

## Handles, sesiones, objetivo de proceso y observacion

- `db/runtime_handles_controlplane.go`
  - gestion principal de `runtime_handles`

- `db/runtime_handle_sync.go`
  - sincronizacion de handle, TMUX canonico y cierre de superseded

- `db/runtime_handle_delivery_mode.go`
  - modos de delivery y reglas de capacidad interactiva

- `db/runtime_handle_reconcile.go`
  - reconciliacion entre runtime handle y sesion

- `db/runtime_handles_hot.go`
  - snapshot caliente de handles canonicos/operativos por agente, proyecto y sesion

- `db/runtime_handle_operativo_hot.go`
  - helpers publicos para pedir el ultimo handle canonico/operativo reciente con fallback a DB fuerte

- `db/runtime_process_target.go`
  - traduce order/handle/runtime al `ObjetivoProceso` real; aqui se decide si un fallback por PID es valido o si TMUX/session debe bloquearlo

- `db/runtime_observation.go`
  - observacion de salud y degradacion de runtime desde supervision

- `db/runtime_session_inference.go`
  - inferencia de transporte, handle kind/ref, metadata y capabilities desde sesion

- `db/runtime_session_state.go`
  - estado de sesion runtime y aparcado de sesion activa

- `db/runtime_worker_state.go`
  - normalizacion de estados de worker a vocabulario canonico (`starting`, `working`, `waiting_input`, `blocked_runtime`, etc.)

## Transcript y Progreso

- `db/runtime_transcript.go`
  - fachada minima del carril transcript

- `db/runtime_transcript_store.go`
  - persistencia y listado de transcript

- `db/runtime_transcript_ingest.go`
  - ingestion por handle y resolucion de runtime

- `db/runtime_transcript_text.go`
  - limpieza, clasificacion y normalizacion del texto transcript

- `db/runtime_transcript_hot_idle.go`
  - `hot idle` y resolucion de rutas de log

- `db/progreso_runtime.go`
  - progreso estimado desde señales del runtime

## Planificador, mailbox y soporte operativo

- `db/planificador.go`
  - fachada del planificador y saneado previo del estado autonomo

- `db/planificador_pool_local.go`
  - pool local compartido y reglas de ocupacion

- `db/planificador_free_tasks.go`
  - seleccion de tarea libre por agente y scoring

- `db/runtime_mailbox_controlplane.go`
  - operaciones principales de mailbox

- `db/runtime_supervision_batches.go`
  - lotes de supervision frecuentes

- `db/runtime_reconciliation_hygiene.go`
  - higiene, reconciliacion y reparaciones operativas

- `db/runtime_purge.go`
  - purga de handles, orders e historico con guardas de seguridad

## Como depurar rapido

- Si falla un `start`
  - mira `runtime_start_control.go`, `runtime_bootstrap_controlplane.go`, `runtime_start_bootstrap.go` y `runtime_start_handle_update.go`

- Si el bootstrap no ve order, mailbox o checkpoint esperados
  - mira `bootstrap_prepare_lite.go`, `runtime_bootstrap_lease_selection.go`, `project_context.go` y `project_context_prepare_lite.go`

- Si falla `send_instruction`
  - mira primero `runtime_send_instruction_policy.go`, luego `runtime_send_instruction_dispatch.go`, despues `runtime_send_instruction_state.go` y `runtime_send_instruction_receipt.go`

- Si el problema es de runtime/handle canonico
  - mira `runtime_order_canonical_target.go`, `runtime_handle_sync.go`, `runtime_handles_hot.go` y `runtime_handle_operativo_hot.go`

- Si el problema es de objetivo de proceso/TMUX/PID
  - mira `runtime_process_target.go` y `runtime_session_inference.go`

- Si el problema es de transcript, progreso o `work_confirmed`
  - mira `runtime_transcript_ingest.go`, `runtime_transcript_text.go`, `runtime_transcript_store.go` y `progreso_runtime.go`

- Si el problema es de scheduler o autoasignacion
  - mira `planificador.go`, `planificador_pool_local.go` y `planificador_free_tasks.go`

- Si el problema aparece solo en degradacion/read-only
  - mira `status_readonly.go`, `bootstrap_prepare_lite.go` y `local_legacy_open.go`
