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
  `POST /api/v0/apps/director`, observar agentes paralelos desde
  `POST /api/v0/director/stats` con `include_process_refs`,
  `include_agent_progress` e `include_agent_usage`, y parar limpio mediante el
  `trap` del operador.

Validacion real 2026-05-13:

- La prueba de self-observability documento el flujo REST real, stats con
  progreso vivo, deteccion stalled, stop por run-control y confirmacion
  `agents_stop_confirmed`.
- Queda pendiente cerrar la evidencia end-to-end entre continuation y
  deliveries antes de considerar completa la frontera de finalizacion del run.
