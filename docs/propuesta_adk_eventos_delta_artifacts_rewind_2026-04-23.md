# Propuesta ADK para Orquesta

## Objetivo

Tomar de Google ADK solo los contratos que sí mejoran la autonomía real de Orquesta sin reescribir su núcleo ni cambiar su filosofía.

Esto no propone adoptar ADK ni replatformar Orquesta.
Propone reforzar el control plane actual con:

- eventos canónicos
- `state_delta` por decisión y por tool
- `artifacts` versionados
- `rewind/replay` parcial de tareas autónomas

## Fuentes de inspiración verificadas

- `Session / State / Memory`
- `Callbacks`
- `state_delta` asociado a eventos
- `Artifacts`
- `rewind` de sesión

Referencias primarias:

- https://github.com/google/adk-python
- https://google.github.io/adk-docs/sessions/
- https://google.github.io/adk-docs/sessions/state/
- https://google.github.io/adk-docs/callbacks/
- https://google.github.io/adk-docs/artifacts/
- https://google.github.io/adk-docs/sessions/rewind/

## Encaje con Orquesta

Orquesta ya tiene casi todas las piezas, pero hoy están repartidas:

- `runtime orders`
- `runtime mailbox`
- auditoría
- transcript
- checkpoints
- supervisor residente
- control plane `event-driven`

Lo que falta no es otro framework. Falta formalizar mejor el contrato entre esas piezas.

La adaptación correcta es:

1. Cada decisión autónoma relevante produce un `AutonomyEvent`.
2. Cada `AutonomyEvent` lleva `state_delta` explícito.
3. Los outputs pesados no viven en el estado corto; viven como `artifacts`.
4. El supervisor puede hacer `rewind/replay` parcial cuando una continuidad o un handoff deriva mal.

## Qué copiar

### 1. `AutonomyEvent` canónico

Cada decisión del supervisor o del control plane debe persistirse como evento estructurado:

- `kind`
- `actor`
- `project_id`
- `task_id`
- `runtime_id`
- `handle_id`
- `source`
- `reason`
- `state_delta`
- `artifacts_ref`
- `created_at`

Ejemplos:

- `worker_recovered`
- `handoff_requested`
- `repair_helper_opened`
- `repair_helper_closed`
- `finish_app_slice_derived`
- `runtime_restarted`
- `prime_escalation_requested`

### 2. `state_delta` obligatorio

El cambio útil no es solo guardar el evento, sino guardar el delta:

- `last_autonomy_state`
- `last_autonomy_action`
- `last_autonomy_source`
- `mailbox_pending`
- `work_confirmed`
- `continuity_pending`
- `task_state`
- `runtime_state`
- `assignment_state`

Regla:

- nada de mutación operativa importante sin delta explícito

Eso hace el loop:

- auditable
- reproducible
- reversible

### 3. `Artifacts` versionados

El estado corto no debe cargar blobs ni evidencias grandes.

Artifacts propuestos:

- diff o patch de trabajo
- salida de test
- transcript relevante recortado
- manifiesto de sesión
- resumen de review
- evidencia de bloqueo

Un artifact debe tener:

- `artifact_id`
- `scope`: `task`, `runtime`, `project`, `session`
- `kind`
- `version`
- `content_type`
- `path` o `blob_ref`
- `created_at`

### 4. `rewind/replay` parcial

No hace falta time travel total.

Sí hace falta:

- `rewind_task_to_event(event_id)`
- `replay_task_from_event(event_id)`

Aplicación inicial:

- `finish_app`
- `handoff`
- `repair-helper`
- `worker_recovery`

Uso:

- una continuidad mala se revierte al último evento sano
- el supervisor vuelve a lanzar desde ahí
- sin dejar residuos semánticos mezclados

## Qué no copiar

- el framework ADK completo
- builders genéricos de agentes
- capas acopladas a Gemini o Vertex
- otra pila paralela dentro de Orquesta

## No-objetivos

- cambiar el modelo local-first
- reemplazar `runtime orders` o `mailbox`
- sustituir supervisor residente por otro runtime
- rehacer la arquitectura hexagonal ya existente

## Beneficio esperado

Esto no arregla por sí solo:

- runtimes rotos
- GPU saturada
- handoffs mal diseñados
- clasificación pobre de transcript

Sí mejora directamente:

- trazabilidad real del loop autónomo
- coherencia de estado entre supervisor, worker y resumen global
- rollback limpio de decisiones malas
- rehidratación tras reinicio
- depuración de autonomía larga

## Riesgo principal

El riesgo no es técnico, es de sobrealcance.

Si se intenta “meter ADK” dentro de Orquesta, será regresivo.
Si se toma solo el contrato útil, encaja bien con el núcleo actual.

## Roadmap mínimo recomendado

### Fase 1

- introducir `AutonomyEvent` en el control plane
- persistir `state_delta` en handoff, recovery, repair-helper y `finish_app_slice_derived`

### Fase 2

- sacar `artifacts` versionados para:
  - patches
  - logs
  - tests
  - manifests

### Fase 3

- añadir `rewind_task_to_event(event_id)` para tareas autónomas

### Fase 4

- hacer que el supervisor use `event + delta + artifacts` como contrato canónico de decisión

## Criterio de aceptación

Esta línea de trabajo merece la pena solo si consigue que Orquesta:

- se autocorrija mejor
- sea más depurable
- deje menos residuos históricos
- y mantenga la filosofía actual de control plane durable, supervisor residente y workers acotados

Si obliga a rehacer el núcleo o a meter otro framework entero, no merece la pena.
