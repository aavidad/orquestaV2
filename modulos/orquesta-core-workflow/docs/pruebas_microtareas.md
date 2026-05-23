# Pruebas de microtareas v0

```text
Caso: workflow_task_v0_contrato_compacto
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: `WorkflowTaskV0` valido normaliza refs compactas, conserva `write_set`, criterios, `context_refs` y `function_contract_refs` opacas, acepta `function_name` sin `contract_ref` en el DTO base y valida la fase contra el catalogo v0.
Ultima ejecucion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow
Riesgos: La integracion durable con eventos/comandos queda para otro corte autorizado.
```

```text
Caso: workflow_task_v0_rechaza_detalles_prohibidos
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: Fase desconocida, `write_set` vacio y criterio vacio se rechazan con errores publicos. Secretos y credenciales se rechazan; refs/texto opacos de adaptador o ejecucion se permiten para no cortar el ciclo por palabras reparables. La apertura queda pendiente de revision futura.
Ultima ejecucion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow
Riesgos: La lista negativa debe ampliarse cuando aparezcan nuevos conectores o terminos de infraestructura.
```

```text
Caso: create_microtask_handler_puro
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: `HandleCommandV0` acepta `CreateMicrotask` solo con fase `planificacion_microtareas` activa y contratos de funcion publicados, valida `WorkflowTaskV0`, rechaza refs sin `contract_ref`, emite un unico `MicrotaskCreated`, outbox vacio y error publico para payload invalido.
Ultima ejecucion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Riesgos: La ejecucion real de la tarea queda fuera del nucleo y debe mantenerse en adaptadores futuros.
```

```text
Caso: microtask_created_reducer_y_replay
Tipo: replay
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: `ApplyEventV0` proyecta `task_id` en `tasks` solo con fase `planificacion_microtareas` activa y refs ya publicadas; `ReplayDurableEventsV0` acepta `MicrotaskCreated` con sequence estricta.
Ultima ejecucion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Riesgos: El evento guarda solo refs compactas; consumidores que necesiten el detalle completo deben leerlo desde el contrato de planificacion futuro.
```

```text
Caso: idempotencia_create_microtask
Tipo: idempotency
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: Repetir `CreateMicrotask` sobre estado que ya contiene `task_id` devuelve no-op idempotente sin eventos ni outbox.
Ultima ejecucion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Riesgos: Es idempotencia logica v0; persistence futura debe imponer unicidad durable de `idempotency_key`.
```
