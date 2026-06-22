# Incidencia OPES Supervision Bloqueante Tractorista

Fecha: 2026-06-22.

## Contexto

Run OPES:
`run-opes-tractorista-produccion-temas-20260622`.

Objetivo: producir temas del curso `operario-tractorista-grupo-5` con Orquesta,
un padre por tema y entregas con ACK.

## Sintomas

1. La primera tanda lanzo 5 padres: temas 002, 004, 006, 008 y 010.
2. Los temas 002, 006, 008 y 010 entregaron `agent_ack.json`.
3. El tema 004 escribio artefactos y paso pruebas obligatorias, pero no llego a
   escribir `agent_ack.json`.
4. `POST /api/v0/autoprogramming/supervise` se llamo con `max_ticks=3`,
   `max_bursts=4` y `max_commands=40`, pero no devolvio respuesta durante mas
   de dos minutos mientras quedaba un agente activo sin ACK.
5. `POST /api/v0/autoprogramming/status` seguia respondiendo, con 4 tareas en
   estado `delivered`, 1 agente `in_flight` y 5 tareas pendientes.
6. Al pedir `POST /api/v0/runs/control action=stop forced=true`, la peticion de
   supervision bloqueada devolvio finalmente `stop_reason=stopped`.
7. La parada afecto a toda la run; no habia control fino para detener solo el
   agente 004 y dejar que Orquesta siguiera con 001, 003, 005, 007 y 009.

## Impacto

- Una entrega sin ACK puede bloquear la continuidad del temario aunque otros
  padres ya hayan entregado trabajo valido.
- El operador humano pierde visibilidad porque `supervise` queda esperando en
  vez de devolver estado parcial, timeout controlado o accion recomendada.
- La unica salida publica evidente fue parar la run completa, lo que obliga a
  lanzar una segunda run de continuacion y aumenta trabajo operativo.

## Evidencia

- Respuesta vacia mientras la peticion seguia abierta:
  `/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/produccion_tractorista_2026-06-22/responses/supervise_after_first_acks.json`.
- Stop solicitado:
  `/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/produccion_tractorista_2026-06-22/responses/stop_run_g04_no_ack.json`.
- ACKs presentes:
  `/tmp/orquesta-opes-tractorista-bases-20260622/runtime/run-opes-tractorista-produccion-temas-20260622/agent-ref-task-autoprogramming-e8fb7b516501-g02/agent_ack.json`.
  `/tmp/orquesta-opes-tractorista-bases-20260622/runtime/run-opes-tractorista-produccion-temas-20260622/agent-ref-task-autoprogramming-e8fb7b516501-g06/agent_ack.json`.
  `/tmp/orquesta-opes-tractorista-bases-20260622/runtime/run-opes-tractorista-produccion-temas-20260622/agent-ref-task-autoprogramming-e8fb7b516501-g08/agent_ack.json`.
  `/tmp/orquesta-opes-tractorista-bases-20260622/runtime/run-opes-tractorista-produccion-temas-20260622/agent-ref-task-autoprogramming-e8fb7b516501-g10/agent_ack.json`.
- Tema 004 con artefactos sin ACK:
  `/home/alberto/Trabajo/OPES/opes-salidas/diputacion_granada/cursos_opes_2026/administracion_especial/operario-tractorista-grupo-5/temas/tema_004/`.

## Tareas Tecnicas

- `SUP-TASK-001`: `autoprogramming/supervise` debe tener tiempo maximo efectivo
  y devolver estado parcial accionable si quedan agentes vivos sin ACK.
- `SUP-TASK-002`: si una cohorte tiene entregas con ACK y un agente sin ACK
  pero sin actividad reciente, Orquesta debe poder cerrar o aislar las entregas
  validas y lanzar capacidad pendiente, sin depender de parar toda la run.
- `SUP-TASK-003`: incorporar control fino por agente/tarea para `stop` o
  `quarantine`, con checkpoint y evidencia, sin convertirlo siempre en parada
  global de run.
- `SUP-TASK-004`: detectar artefactos creados sin ACK, marcarlos como
  `delivery_without_ack_candidate` y pedir validacion/rework causal en vez de
  dejarlos como tarea pendiente opaca.

## Criterio De Cierre

Un smoke OPES equivalente debe demostrar que, con 4 ACKs y un agente sin ACK,
`autoprogramming/supervise` responde en plazo, no bloquea la API, conserva las
entregas validas y propone una accion distinta a parar toda la run cuando haya
tareas pendientes sanas.
