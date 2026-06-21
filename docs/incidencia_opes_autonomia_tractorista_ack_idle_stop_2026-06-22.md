# Incidencia OPES Autonomia Tractorista ACK, Idle Y Stop

Fecha: 2026-06-22.

## Contexto

Prueba OPES de autonomia sobre el curso
`operario-tractorista-grupo-5` desde:

`/home/alberto/Trabajo/OPES/opes-salidas/diputacion_granada/cursos_opes_2026/administracion_especial`

Run principal:
`run-opes-autonomia-operario-tractorista-course-parent-20260622`.

Objetivo: comprobar si el temario podia pasar de andamiaje a produccion. El
agente hizo lo correcto: detecto falta de bases oficiales vigentes, creo
`11_validacion/GATE_BASES_ORQUESTA.md`, actualizo `course_manifest.json` a
`bloqueo_base_oficial_pendiente_localizar` y escribio `agent_ack.json`
`completed`.

## Sintomas

1. `prepare-run` usando la raiz completa de OPES fallo con
   `worktree_isolation_invalid / worktree_snapshot_unreadable` sin ruta ni causa
   accionable. La prueba solo pudo continuar acotando `project_workdir` al
   directorio padre del curso.

2. El ACK estructurado existia y era valido, pero
   `/api/v0/autoprogramming/status` siguio marcando la run como viva:
   `agents_in_flight=1`, `tasks_open=1`, `closure_status=blocked`. Una llamada
   manual posterior a `POST /api/v0/runs/supervise` consumio el ACK en un tick y
   cerro la tarea con `open_tasks=0`.

3. Tras cerrar la run OPES, el residente arranco una automejora no solicitada
   `request-ref-autoprogramming-backlog-scanner-7e8a6ae9` sobre documentos de
   Orquesta. Esto ocurrio en una sesion dedicada a OPES, con otro agente humano
   trabajando en el nucleo, y podia pisar cambios ajenos.

4. `POST /api/v0/runs/control action=stop forced=true` marco esa run de
   automejora como `stop_requested` con checkpoint, y
   `POST /api/v0/runs/supervise` devolvio `stop_reason=stopped`, pero el proceso
   Codex real siguio vivo hasta cortar la sesion del servidor.

## Evidencia

- ACK OPES:
  `/tmp/orquesta-opes-tractorista-20260622/runtime-course-parent/run-opes-autonomia-operario-tractorista-course-parent-20260622/agent-ref-task-autoprogramming-03f5f97b81d1-g01/agent_ack.json`.
- Status antes de supervisar:
  `opes-salidas/coordinacion_temarios/prueba_autonomia_orquesta_operario_tractorista_2026-06-22/status/autoprogramming_status_after_gate_file.json`.
- Supervision manual que cerro la run:
  `opes-salidas/coordinacion_temarios/prueba_autonomia_orquesta_operario_tractorista_2026-06-22/responses/supervise_run_course_parent.json`.
- Status despues de supervisar:
  `opes-salidas/coordinacion_temarios/prueba_autonomia_orquesta_operario_tractorista_2026-06-22/status/autoprogramming_status_after_manual_supervise.json`.
- Stop de automejora:
  `opes-salidas/coordinacion_temarios/prueba_autonomia_orquesta_operario_tractorista_2026-06-22/responses/stop_self_improvement_run.json`.
- Supervision de stop:
  `opes-salidas/coordinacion_temarios/prueba_autonomia_orquesta_operario_tractorista_2026-06-22/responses/supervise_self_improvement_stop.json`.

## Impacto

- OPES no queda aun autonomo de extremo a extremo: requiere supervisar a mano
  para consumir ACKs completados en algunos casos.
- El modo idle puede lanzar automejoras de Orquesta durante una sesion de OPES
  sin autorizacion explicita, generando riesgo de solape con otro agente.
- La parada publica puede decir `stopped` sin haber terminado el proceso real.
- El fallo de snapshot de raiz OPES obliga al operador a adivinar el workdir
  correcto.

## Tareas Tecnicas

- `RTDELIVERY-010`: consumir ACK `completed` valido aunque el proceso Codex siga
  vivo y cerrar la tarea sin esperar supervision manual.
- `SRV-TASK-027`: desactivar o aislar la automejora idle durante ejecuciones OPES
  salvo opt-in explicito del operador.
- `SRV-TASK-028`: una parada de run debe confirmar parada real de procesos o
  devolver `stop_pending`, nunca `stopped` con procesos hijos vivos.
- `RTWT-OPES-001`: `worktree_snapshot_unreadable` debe incluir evidencia
  accionable o una politica de scope/ignore para raices OPES grandes.

## Criterio De Cierre

Una prueba real equivalente debe poder:

- preparar la run OPES sin acotar manualmente el proyecto, o fallar con una
  razon accionable y ruta redactada;
- cerrar automaticamente cuando aparezca `agent_ack.json completed`;
- no lanzar automejora idle mientras la sesion OPES sigue activa salvo opt-in;
- parar agentes reales con confirmacion de proceso terminado;
- dejar status final sin `queue_live` ni `recommended_action=supervise:queue`
  cuando no quedan runs relevantes.
