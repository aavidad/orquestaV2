# Incidencia OPES Capacity Decided No Proyectado Tractorista

Fecha: 2026-06-22.

## Contexto

Run OPES:
`run-opes-tractorista-cierre-texto-rag-tests-post-rework-20260622`.

Objetivo: cierre de texto, tests normalizados y RAG del curso
`operario-tractorista-grupo-5` tras el rework de tests.

## Síntoma

Orquesta preparo tres tareas (`g01`, `g02`, `g03`), pero solo lanzo dos agentes.
La tarea `g01` quedo pendiente aunque existia evidencia duradera de decision de
capacidad:

- el run proyectado tenia `capacity_requests=3`;
- el run proyectado solo mostraba decisiones de capacidad para `g02` y `g03`;
- el event store contenia el evento durable `CapacityDecided` de `g01`;
- el outbox `RequestCapacityDecision` de `g01` estaba despachado y ACKeado.

El supervisor no podia avanzar porque la decision duradera no estaba reflejada
en la proyeccion del run y, por tanto, no se materializaba el
`LaunchRuntimeAgent` correspondiente.

## Impacto

- Un tema o tarea OPES podia quedar sin agente aunque la decision ya estuviera
  escrita en eventos.
- La cola parecia viva pero incompleta: habia capacidad decidida sin lanzamiento.
- La autonomia exigia intervencion manual o reinicio con parche para recuperar
  el agente faltante.

## Arreglo Aplicado

`DrainRunV0` reconcilia ahora decisiones de capacidad duraderas antes de intentar
recuperar launches. Si encuentra un `CapacityDecided` en el event store que no
aparece en la proyeccion del run y la solicitud de capacidad sigue existiendo en
el estado actual:

1. aplica el `CapacityDecided` durable sobre la proyeccion actual;
2. guarda el run actualizado respetando `LastEventID` y `LastSequence`;
3. devuelve el control al ciclo normal para que se cree/lance el agente.

No interpreta contenido OPES, logs ni texto libre: usa eventos estructurados y
refs de capacidad.

## Validacion

Validacion focal:

```bash
go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestReconcileDurableCapacityDecisionsForRunV0ProyectaDecisionPerdida|TestReconcile(OrphanCapacity|ClaimedLaunch)'
```

Smoke real observado tras aplicar el arreglo:

- `capacity_decisions=3`;
- `agents_requested=3`;
- `agents_started=3`;
- `g01` paso de pendiente a `running`;
- `g02` y `g03` permanecieron como entregados.

## Tarea Técnica Relacionada

- `CAPACITY-PROJECTION-TASK-001`: mantener un smoke de recuperacion con
  `CapacityDecided` durable no proyectado y comprobar que el siguiente
  `supervise` materializa el `LaunchRuntimeAgent` sin intervencion manual.

## Criterio De Cierre

Una run con `RequestCapacityDecision` ACKeado y `CapacityDecided` durable, pero
sin proyeccion de decision en `RunStore` para una solicitud de capacidad que
sigue viva, debe recuperar la decision y lanzar el agente faltante en el
siguiente ciclo de `DrainRunV0` o `runs/supervise`.
