# Decision: estadisticas de agentes completados y frontera de programacion

Fecha: 2026-05-12.

Contexto: una prueba real con `orquesta-server` y agentes Codex genero
brainstorming, documentacion y un bootstrap Go, pero la app quedo incompleta.
Las estadisticas marcaban como `running/no_signal` agentes que ya habian escrito
`agent_ack.json`, y el servidor residente quedo con error
`director_supervised_burst_step` al intentar avanzar desde el bootstrap a varias
microtareas paralelas.

Decision:

- El nucleo de estadisticas considera `completed` a un agente arrancado cuando
  su trabajo ya quedo reflejado en `phase_artifacts` o `deliveries`.
- `DeliveryRegistered` proyecta `agent_ref` en `delivered_agents`; stats debe
  usar esa proyeccion para marcar agentes completados, no inferencias por nombre
  de ACK o ruta de fichero.
- `no_signal_agent_refs` solo incluye agentes arrancados que no tienen progreso,
  no han fallado, no han sido parados y tampoco tienen entrega reflejada.
- La web pide por defecto `include_process_refs`, `include_agent_progress` e
  `include_agent_usage` al consultar estadisticas de un `run_ref`.
- El servidor residente usa por defecto el presupuesto maximo de comandos del
  ciclo del director para drenar un run. El valor anterior era demasiado bajo
  para una frontera paralela normal: cuatro microtareas pueden producir hasta
  doce comandos de scheduling antes de despachar agentes.

Razonamiento:

- Un agente que ya entrego no debe figurar como perdido ni inflar
  `agents_in_flight`; eso lleva al director y a la web a diagnosticos falsos.
- La prueba real mostro que inferir el agente desde `delivery_ref` es fragil:
  algunos ACK incluyen el agent id y otros no. El dato correcto ya existe en el
  payload de `DeliveryRegistered`, por lo que debe persistirse en el estado.
- La prueba no fallo por falta de ideas del director, sino por una diferencia de
  defaults entre la frontera paralela de microtareas y el presupuesto del
  servidor residente. Subir el presupuesto al maximo validado por el core evita
  bloquear paralelismo legitimo sin saltarse las validaciones del ciclo.
- La solucion queda en contratos y adaptadores existentes: no se introduce
  runtime, DB ni proveedor hardcodeado.

Validacion esperada:

- Tests unitarios del nucleo verifican que agentes con artefacto/entrega no
  aparecen como sin senal.
- Tests del servidor verifican que el default de `MaxCommands` soporta frontera
  paralela.
- Smoke REST real opt-in debe arrancar `orquesta-server`, crear run por
  `POST /api/v0/apps/director`, observar el agente director arrancado desde
  `POST /api/v0/director/stats` con `include_process_refs`,
  `include_agent_progress` e `include_agent_usage`, y parar limpio mediante el
  `trap` del operador. La observacion de olas paralelas pertenece al smoke del
  supervisor/drain, no al endpoint REST directo `start-only`.

Validacion real 2026-05-13:

- La prueba de self-observability documento el flujo REST real, stats con
  progreso vivo, deteccion stalled, stop por run-control y confirmacion
  `agents_stop_confirmed`.
- Una prueba real posterior valido `director -> microtareas -> worker Codex real
  -> agent_ack.json -> DeliveryRegistered` para la primera microtarea de
  programacion.
- Queda pendiente repetirlo hasta completar una app entera con review,
  validacion y cierre.

Ampliacion 2026-05-14:

- En trabajos externos de OPES, `delivered` ya significa que Orquesta recibio
  ACK valido, registro `DeliveryRegistered` y entrego el artefacto al conector
  externo. Aunque la tarea no este formalmente cerrada por `CloseTask`, el
  porcentaje operativo debe contar esa tarea como resuelta.
- `progress.percent_complete` se calcula con tareas resueltas:
  `delivered_tasks union closed_tasks`. `progress.tasks_closed` sigue contando
  solo cierre formal para no mezclar estados contractuales.
- Validacion: prueba real OPES `summarize_topic` con Codex `gpt-5.5 xhigh`
  genero `topic_summary`, Orquesta lo entrego a OPES y el nucleo ya tiene test
  que exige `percent_complete=100` cuando una unica tarea esta `delivered`.
