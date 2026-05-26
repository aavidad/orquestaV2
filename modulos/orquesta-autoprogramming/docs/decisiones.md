# Decisiones: orquesta-autoprogramming

## 2026-05-13: autoprogramacion sale del nucleo

Decision: `AutoprogrammingRequestV0`, agrupacion por area y review gate de
codigo viven en `orquesta-autoprogramming`.

Motivo: son reglas de producto/programacion, no capacidades esenciales del
nucleo de orquestacion. El core debe poder orquestar OPES u otros dominios sin
arrastrar contratos de programacion.

Consecuencia:

- MCP y delivery Codex dependen de este modulo.
- El core mantiene puertos genericos de observacion, progreso, review y
  scheduling.
- Cualquier nuevo dominio debe crear su propio modulo de contrato, no meter
  reglas en el core.
- La capacidad de juicio no vive aqui: Orquesta usa director/agentes para
  decidir plan, fases y estrategia. Este modulo solo aporta contratos de
  programacion para que el director tenga reglas verificables.

## 2026-05-13: sin adaptadores dentro del modulo

Decision: este modulo no lee disco ni ejecuta pruebas, aunque sus contratos
hablen de `write_set` y `required_tests`.

Motivo: la ejecucion real pertenece a conectores. Asi evitamos hardcodear DB,
runtime, VCS, proveedor, HOME o modelos.

## 2026-05-23: entrega util fuera de alcance crea follow-up

Decision: el review gate distingue rechazo bloqueante de entrega aprovechable
fuera del `write_set`.

Motivo: si un agente sale del alcance para desbloquear una prueba, Orquesta no
debe aceptarlo como normal ni tirar la evidencia. El contrato marca
`preserve_output=true`, `requires_followup=true` y
`recommended_action=request_followup_review` solo cuando ACK y tests requeridos
estan verdes y el problema es el alcance.

Consecuencia: ACK ausente o tests fallidos siguen bloqueando cierre con
`recommended_action=block_closure`; el follow-up queda como reparacion dirigida
para revision o tarea posterior.

## 2026-05-26: capacidad por marcadores tokenizados

Decision: la politica de capacidad detecta OPES, planes documentales y riesgo
alto por tokens o frases normalizadas, no por subcadenas arbitrarias.

Motivo: palabras practicas de programacion como `scopes` no deben activar la
politica documental OPES ni escalar capacidad a `xhigh`.

## 2026-05-26: ola programable 10x6

Decision: `AutoprogrammingRequestV0` admite por defecto hasta 10 tareas padre,
10 areas y 10 entradas de `write_set`; cada `WorkflowTaskV0` generado transporta
profundidad recursiva 1, hasta 6 subagentes por padre y presupuesto derivado de
60 agentes hijos salvo limite explicito menor en la request.

Motivo: la ola complementaria de autoprogramacion necesita expresar capacidad
10 padres + 6 subagentes por padre sin mover runtime, proveedor ni lanzamiento
real dentro del contrato puro.
