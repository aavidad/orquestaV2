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

## 2026-05-27: limites ampliados por contrato explicito

Decision: `AutoprogrammingRequestV0` permite ampliar `max_task_refs`,
`max_areas` y `max_write_set_entries` hasta 40 cuando la request lo declara de
forma estructurada.

Motivo: OPES u otras apps grandes pueden necesitar olas de mas de 10 padres sin
microfragmentar contenido. La ampliacion debe ser verificable por contrato, no
por hints en texto libre ni por deteccion de dominio.

Consecuencia: el valor por defecto sigue siendo 10. Valores negativos o mayores
de 40 quedan rechazados con issues publicos de contrato; la delegacion por padre
sigue gobernada por su limite propio.

## 2026-05-27: contratos v1 para perfiles por tipo de app

Decision: `AutoprogrammingRequestV1` envuelve la solicitud `v0` y agrega
`app_kind` + `work_profiles` para seleccionar `profile_kind` por `task_ref`,
area o app completa. `BuildAutoprogrammingProgrammableWorkV1` produce una
envoltura versionada y mantiene un `base` compatible con `v0`.

Motivo: APG-002 pedia versionar el contrato solo cuando existan perfiles de
trabajo por tipo de app, sin cambiar el comportamiento historico.

Consecuencia: las composiciones con perfiles propios pueden transportar esa
decision como contrato puro; las que siguen usando `v0` reciben
`implementation` como antes.

## 2026-05-27: T208 queda como umbrella reconciliado

Decision: el item T208 `autoprogramming-guardian-breakglass` deja de tratarse
como pendiente generico dentro de `orquesta-autoprogramming`.

Motivo: los intentos posteriores cerraron las fronteras concretas del guardian
en owners especificos: salida publica, redaccion, entorno, artefactos,
readiness, resultado estructurado, shutdown, reparador, refs, lectura acotada y
lease. Reabrir el umbrella duplicaria trabajo y mezclaria contratos puros con
runtime/servidor.

Consecuencia: este modulo solo mantiene validacion y politicas puras de
autoprogramacion. Nuevos huecos del guardian deben documentarse como Txx
focales con evidencia propia y write-set de su owner real.

## 2026-07-01: contrato puro para automejora idle del servidor

Decision: el modulo declara un contrato `v0` para decidir si una composicion
de servidor debe preparar automejora idle, planificar entradas de backlog y
proyectar estado externo, pero no lee entorno ni arranca trabajo.

Motivo: los criterios de APG-004/T208 implican variables y estado de servidor
(`ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS`,
`ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_TARGET_QUEUE`, cola visible, outbox y
proceso externo). Esa semantica debe ser verificable para composiciones sin
meter servidor, runtime ni filesystem en `orquesta-autoprogramming`.

Consecuencia: `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS` tiene
default contractual 60 y `0` desactiva solo el disparo por reloj idle, no el
relleno por capacidad libre bajo `target_queue`. El planner conserva tareas ya
visibles en cola como skipped, filtra narrativas y puede emitir una tarea
scanner con refs de scanner/hash. La proyeccion publica distingue
`outbox_pending`, `wait_external` y `external_process_verified`; la composicion
real sigue siendo responsable de observar outbox/procesos y ejecutar tests.

## 2026-07-02: guards de contrato por refs explicitas

Decision: `BuildAutoprogrammingProgrammableWorkV0` puede inyectar tests de
regresion, criterios y refs de contrato cuando una tarea declara una ref
explicita conocida en `context_refs`.

Motivo: los agentes remotos goal-first reabrieron bugs ya cerrados de
startup-lock/codex_wrapper al no recibir los invariantes completos antes de
editar. El contrato debe viajar en el spec, no depender de memoria local del
agente ni de inferencia por nombres de fichero.

Consecuencia: el primer guard soportado es
`contract:codex-startup-lock:v0`/`bug:BUG-ORQ-20260701-092`. Inyecta tests
focales de `codex_wrapper` y criterios para enumerar las variables
`ORQUESTA_CODEX_STARTUP_LOCK_SECONDS`,
`ORQUESTA_CODEX_STARTUP_LOCK_TIMEOUT_SECONDS` y
`ORQUESTA_CODEX_STARTUP_LOCK_STALE_SECONDS`. No se activan guards por
subcadenas libres: solo por refs explicitas versionables.
