# Decisiones locales

```text
Fecha: 2026-05-27
Decision: T15 no se relanza desde `orquesta-director-agent`; el DTO consume la politica comun de rails por campo y conserva vocabulario operativo opaco.
Motivo: desde 2026-06-02 los rails de detalle quedan offline por defecto. Reabrir desde el DTO duplicaria owner y volveria a bloquear decisiones validas por palabras como runtime, provider, model, db, sql o codex.
Impacto: el director puede seguir expresando refs compactas, context_refs y politicas de prompt/transcript como referencias. Los cortes por detalle solo existen con `ORQUESTA_RAILS_MODE=enforced`; en runtime normal el director/agente repara o aprovecha la evidencia.
Estado: aceptada_local
```

```text
Fecha: 2026-05-10
Decision: El DTO intercambiable del director cubre consultas, capacidad, agentes, retrabajo y replan como comandos compactos.
Motivo: una app completa no debe depender de la sesion humana para pedir informacion faltante, capacidad, agentes especializados o replanificacion tras revision.
Alternativas: dejar esos pasos solo en el director de composicion; pedir texto libre al agente externo; hacer que el director programe o repare codigo directamente.
Impacto: `DirectorAgentDecisionV0` agrega `ask_director`, `ask_user`, `request_capacity`, `request_agent`, `request_rework` y `record_replan_decision`; todos usan refs/resumen/listas pequenas y siguen sin provider, runtime, DB, HOME ni secretos.
Estado: aceptada_local
```

```text
Fecha: 2026-05-22
Decision: Separar el limite de listas de texto de las refs/evidencias compactas.
Motivo: en smoke real el director creo una microtarea vertical Go razonable con
12 entradas de `write_set`; el validador la rechazo porque `write_set` compartia
el limite de 10 pensado para `evidence_refs`. Eso hacia fragil la orquestacion
de tareas reales.
Impacto: `evidence_refs`, refs causales y contratos siguen compactos, pero las
listas textuales de microtarea (`write_set`, criterios y tests) admiten hasta
24 entradas. La validacion dura queda en refs, seguridad y datos prohibidos; la
granularidad exacta se corrige por director/review.
Estado: aceptada_local
```

```text
Fecha: 2026-05-11
Decision: Toda microtarea del director con `task.phase_id=programacion` debe
incluir `required_tests`.
Motivo: en smoke real multiagente, los agentes programaron codigo dentro de su
write-set pero no podian cerrar `go test ./...` porque el director no habia
asignado una tarea de bootstrap con `go.mod`/`cmd`. Sin tests obligatorios,
Orquesta gastaba tiempo en trabajo imposible de validar.
Alternativas: aceptar `required_tests` opcional y confiar en prompts; arreglar
la app generada manualmente; permitir cierres por notas del agente.
Impacto: el validador rechaza microtareas de programacion sin pruebas
obligatorias. Los flujos que generan decisiones deben declarar el contrato de
validacion antes de lanzar agentes.
Estado: aceptada_local
```

```text
Fecha: 2026-05-10
Decision: El director consume estadisticas compactas, no snapshots completos.
Motivo: el director debe decidir el siguiente comando con bajo contexto; los detalles de programacion, runtime y evidencias completas pertenecen a agentes especializados y stores externos.
Alternativas: pasar `OrchestrationRunV0` completo al agente externo; incluir payloads/eventos en el DTO; reconstruir contexto desde filesystem.
Impacto: se define `DirectorAgentCompactStatsV0` con contadores y refs pendientes; el puente workflow puede construirlo desde el run publico sin exponer internals ni detalles operativos.
Estado: aceptada_local
```

```text
Fecha: 2026-05-08
Decision: El director externo se expresa como DTO validado, no como import del core ni como sesion de Codex.
Motivo: el nucleo debe ser durable y reemplazable; la IA directora puede cambiar sin reescribir workflow, runtime ni persistencia.
Alternativas: meter un agente director dentro del core; dejar que la sesion principal haga orquestacion manual; acoplar a un proveedor.
Impacto: el adaptador valida `DirectorAgentDecisionV0` y lo traduce a comandos publicos. El core no sabe que IA genero la decision.
Estado: aceptada_local
```

```text
Fecha: 2026-05-10
Decision: El director cierra microtareas con `close_task` antes de abrir `validacion_final`.
Motivo: `accept_review` solo acepta la revision; el core separa el cierre durable de tarea como `CloseTask`.
Alternativas: cerrar la tarea implicitamente desde `accept_review`; saltar directo a `register_final_validation`; fusionar cierre de tarea y validacion final.
Impacto: `close_task` exige tarea, fase `revision`, entrega, revision aceptada, resumen y evidencias compactas; el puente lo traduce a `CloseTask`.
Estado: aceptada_local
```

```text
Fecha: 2026-05-10
Decision: El director declara politica de solicitud al validar y cerrar.
Motivo: el cierre depende de si se pidio documentar, programar un modulo o crear
una app completa. Sin esa declaracion, la aplicacion no puede distinguir un
cierre parcial legitimo de un falso completo.
Alternativas: meter politica de app dentro del core; deducirla por nombres de
ficheros; permitir que cualquier cierre pase y revisarlo manualmente.
Impacto: `register_final_validation` y `close_run` incluyen `request_kind`,
`execution_mode` y `minimum_deliverables`; el DTO sigue siendo compacto y el
servicio de aplicacion decide si hay evidencia suficiente.
Estado: aceptada_local
```

```text
Fecha: 2026-05-08
Decision: El director externo debe ser de bajo consumo: decide y delega, no absorbe el contexto completo.
Motivo: v1/v2 se degradaron por sesiones con contexto grande y decisiones mezcladas con trabajo pesado. Un director con contexto minimo mantiene replay, coste y depuracion bajo control.
Alternativas: usar un director que lea toda la documentacion y resuelva todo; hacer que cada agente pregunte directamente a otros grupos; meter analisis largo en el DTO.
Impacto: `DirectorAgentDecisionV0` limita texto/evidencia, prohibe transcripts y obliga a usar refs compactas. El trabajo real queda en agentes especializados arrancados por Orquesta.
Estado: aceptada_local
```

```text
Fecha: 2026-05-08
Decision: el primer corte de v0 solo soportaba `request_brainstorm`.
Motivo: cerrar primero el ciclo real minimo: Orquesta lanza director externo, recibe decision, valida y aplica un comando observable.
Alternativas: soportar todos los comandos del workflow de golpe; aceptar texto libre; pedir al agente que aplique cambios directamente.
Impacto: el alcance queda pequeno y verificable; nuevos comandos se anaden con payloads explicitos y tests.
Estado: aceptada_local
```

```text
Fecha: 2026-05-09
Decision: El director puede proponer `open_phase`, `request_vote` y `accept_decision`.
Motivo: la autoprogramacion no puede empezar en planificacion si nadie mueve el run desde brainstorming hasta votacion y decision aceptada.
Alternativas: que la sesion principal abra fases manualmente; saltar fases; esconder transiciones en un servicio acoplado.
Impacto: el director puede avanzar el workflow mediante comandos publicos sin importar core, store, runtime ni proveedores.
Estado: aceptada_local
```

```text
Fecha: 2026-05-09
Decision: El director puede proponer `publish_function_contract` y `create_microtask`.
Motivo: para que Orquesta y el director puedan programarse sin intervencion manual necesitan descomponer el plan en contratos funcionales y microtareas durables.
Alternativas: dejar que la sesion principal cree tareas; generar tareas desde texto libre; saltar directamente a programacion.
Impacto: el director sigue siendo externo y ligero, pero ya puede alimentar la fase `planificacion_microtareas`; el puente aplica la decision con comandos publicos del core.
Estado: aceptada_local
```

```text
Fecha: 2026-05-10
Decision: El director puede proponer `propose_autonomous_plan_team` como DTO puro.
Motivo: antes de crear microtareas individuales conviene poder revisar un plan/equipo autonomo compacto sin acoplar proveedor, DB, runtime ni stores.
Alternativas: saltar directo a microtareas; delegar el plan a un servicio de aplicacion; convertir el plan en comando publico del core sin contrato previo.
Impacto: el DTO queda validado en `orquesta-director-agent`; el puente workflow no lo traduce hasta que exista un comando publico del core.
Estado: aceptada_local
```
