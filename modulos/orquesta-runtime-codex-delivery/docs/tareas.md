# Tareas: orquesta-runtime-codex-delivery

## RTDELIVERY-001 - Receipts Codex como observaciones neutras

Objetivo: leer ACKs Codex materializados por conectores externos y convertirlos
en `AgentDeliveryObservationV0` sin filtrar rutas, logs, HOME, OAuth, provider,
modelo, DB ni transcripts al nucleo.

Write-set:

- `source_v0.go`
- `source_v0_test.go`
- `memory_store_v0.go`
- `memory_store_phase_artifact_filter_v0_test.go`
- `source_phase_artifact_filter_v0_test.go`

Validacion:

- `go test -count=1 ./modulos/orquesta-runtime-codex-delivery`

## RTDELIVERY-002 - Registro externo de ACK antes del arranque

Objetivo: registrar por puerto externo el descriptor del ACK durante la
resolucion del spec, antes de confirmar `AgentStarted`.

Write-set:

- `recording_spec_resolver_v0.go`
- `recording_spec_resolver_v0_test.go`
- `agent_ack_path_resolver_v0.go`
- `agent_ack_path_resolver_v0_test.go`

Validacion:

- `go test -count=1 ./modulos/orquesta-runtime-codex-delivery`

Bloqueos:

- No reconstruir paths desde refs del nucleo.
- No escanear directorios de runtime desde el nucleo.

## RTDELIVERY-003 - Supervision conservadora de agentes reales

Objetivo: detectar agentes Codex sin ACK ni progreso observable, emitir
`stalled` y evitar falsos `loop_detected` cuando solo falta evidencia nueva de
logs/ACK.

Write-set:

- `progress_source_v0.go`
- `progress_state_v0.go`
- `progress_source_v0_test.go`
- `progress_stalled_loop_integration_v0_test.go`
- `progress_stop_integration_v0_test.go`

Validacion:

- `go test -count=1 ./modulos/orquesta-runtime-codex-delivery -run 'TestCodexProgressObservationSourceV0|TestCodexProgressObservationV0'`

Bloqueos:

- No cortar procesos por nombre, PID visible ni busqueda global.
- No parar directores u otros agentes sin `process_ref + session_ref` registrado.
- No tratar silencio temporal de un Codex real como bucle terminal.

## RTDELIVERY-004 - Smokes reales opt-in

Objetivo: demostrar que Orquesta arranca agentes Codex reales, recoge ACKs,
registra artefactos/entregas y no deja procesos vivos.

Write-set:

- `smoke_codex_real_v0_test.go`
- `smoke_codex_real_app_v0_test.go`
- `smoke_mcp_form_director_codex_real_v0_test.go`
- `smoke_mcp_form_director_team_codex_real_v0_test.go`
- `smoke_programming_team_codex_real_v0_test.go`
- `smoke_programming_team_director_v0_test.go`
- `smoke_programming_team_service_v0_test.go`
- `smoke_programming_team_workflow_v0_test.go`

Validacion:

- Los tests reales son opt-in por variables `ORQUESTA_CODEX_*`.
- Los unitarios deben seguir pasando sin credenciales ni cuota real.
- El smoke de app pequena espera 4-8 minutos y usa loop gestionado, no polling
  manual de ACK desde el test.

Regla de diseno:

- Una app completa nunca entra como una unica tarea amplia; se divide en
  microtareas con write-set acotado, ACK por corte y supervision de progreso.

## RTDELIVERY-005 - Verificacion real de write-set

Objetivo: no aceptar un ACK valido si el agente modifico ficheros fuera del
write-set en el proyecto real.

Write-set:

- `worktree_baseline_recorder_v0.go`
- `worktree_verifier_v0.go`
- `source_v0.go`
- `source_worktree_v0_test.go`
- `recording_spec_resolver_v0.go`
- `recording_spec_resolver_v0_test.go`
- `smoke_programming_team_*_test.go`
- `docs/*.md`

Validacion:

- `go test -count=1 ./modulos/orquesta-runtime-worktree ./modulos/orquesta-runtime-codex-delivery`

Bloqueos:

- El diff real pertenece al conector externo `orquesta-runtime-worktree`.
- No usar Git ni filesystem desde el nucleo.
- No filtrar `project_work_dir`, `ack_path` ni rutas absolutas en errores hacia
  el nucleo.

## RTDELIVERY-006 - Store durable de descriptors ACK

Objetivo: conservar descriptors de ACK entre reinicios para que Orquesta pueda
observar receipts pendientes sin depender solo de memoria.

Write-set:

- `file_descriptor_store_v0.go`
- `file_descriptor_store_v0_test.go`
- `docs/*.md`

Validacion:

- `go test -count=1 ./modulos/orquesta-runtime-codex-delivery -run TestFileCodexReceiptDescriptorStoreV0`

Bloqueos:

- El store filesystem es conector explicito, no default.
- No elegir SQLite/Postgres ni ningun motor concreto.
- No filtrar paths locales en errores.

## RTDELIVERY-007 - Store durable de progreso anti-bucle

Objetivo: conservar heartbeats compactos y reportes ya emitidos para que un
reinicio no reinicie la deteccion de bucles de agentes sin ACK.

Write-set:

- `file_progress_state_v0.go`
- `file_progress_state_v0_test.go`
- `docs/*.md`

Validacion:

- `go test -count=1 ./modulos/orquesta-runtime-codex-delivery -run TestFileCodexProgressStateStoreV0`

Bloqueos:

- El store filesystem es conector explicito, no default.
- No elegir SQLite/Postgres ni ningun motor concreto.
- No persistir paths, PID, HOME, OAuth, provider, modelo, prompt ni transcripts.

## RTDELIVERY-008 - Review gate de entregas Codex

Objetivo: convertir una entrega Codex ya registrada en
`ReviewGateObservationV0` para que Orquesta pueda aceptar o pedir cambios sin
intervencion manual.

Write-set:

- `source_review_gate_v0.go`
- `source_review_gate_refs_v0.go`
- `source_review_gate_issues_v0.go`
- `source_review_gate_project_files_v0.go`
- `source_review_gate_*_test.go`

Validacion:

- `go test -count=1 ./modulos/orquesta-runtime-codex-delivery`

Reglas cerradas:

- el source de revision lista descriptors aunque la delivery ya este en el run;
- la elegibilidad final se comprueba contra `run.Deliveries`;
- el conteo de lineas se obtiene desde un adaptador externo inyectado;
- el nucleo solo recibe refs compactas, estado de revision y evidencia opaca;
- ACK, tests requeridos, write-set, ficheros ausentes y tamano excesivo
  producen aceptacion o `changes_requested` por politica, no parches manuales.

## RTDELIVERY-009 - Frontera de uso Codex T209

Objetivo: mantener delivery como fuente de receipts/progreso y no convertirlo
en parser de uso, logs ni cuota.

Estado: documentado/cerrado como frontera local.

Validacion:

- `go test -count=1 ./modulos/orquesta-runtime-codex-delivery`

Criterios:

- los descriptors registrados siguen siendo la ruta autorizada para que la
  composicion encuentre el runtime del agente;
- delivery no lee stdout, stderr ni last-message para calcular tokens o cuota;
- cualquier uso visible debe venir de `codex_usage_accounting.json` redactado
  y del puerto de stats de la composicion.
