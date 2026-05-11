# Decisiones locales: orquesta-core-concurrency

```text
Fecha: 2026-05-06
Decision: La concurrencia se decide por scopes declarados, no por inspeccion de Git/filesystem desde el core.
Motivo: El nucleo debe poder replayar decisiones y evitar acoplarse a sistema operativo, rutas reales o comandos externos.
Alternativas: Resolver conflictos con Git status; dejar que los agentes se pisen y arreglar despues; bloquear todo paralelismo.
Impacto: Se crean claims compactos y un detector puro antes de promover gates a core-workflow.
Estado: aceptada inicial
```

```text
Fecha: 2026-05-06
Decision: El claim de concurrencia se construye desde `WorkflowTaskV0` y `ContextBundleV0` publicos, nunca inspeccionando ficheros reales.
Motivo: El scheduler/director necesitan claims reproducibles a partir de contratos durables y contexto pequeno; usar Git/filesystem volveria no determinista el replay.
Alternativas: leer write-set desde el arbol de trabajo; duplicar DTO de tareas en concurrency; pedir al agente que declare scopes sin contrato.
Impacto: `BuildWorksetClaimFromWorkflowTaskV0` queda como mapper puro entre modulos, con issues publicos si tarea/contexto no son validos.
Estado: aceptada en CCY-004
```

```text
Fecha: 2026-05-06
Decision: El gate de concurrencia local se expresa como evaluacion pura sobre `ParallelGroupPlanV0`, no como comando durable del workflow.
Motivo: La promocion aun debe hacerse en core-workflow por corte separado; este modulo solo estabiliza el contrato y la semantica de allow/block/ask_director.
Alternativas: Crear eventos de workflow desde este modulo; bloquear todo el run ante cualquier conflicto; permitir sujetos desconocidos por defecto.
Impacto: `EvaluateConcurrencyGateV0` puede alimentar un futuro `EvaluateConcurrencyGate -> ConcurrencyGateEvaluated` sin importar workflow ni materializar agentes.
Estado: aceptada candidata
```

```text
Fecha: 2026-05-06
Decision: La promocion minima usa `RecordConcurrencyGate -> ConcurrencyGateRecorded`; el calculo sigue en `orquesta-core-concurrency`.
Motivo: Evita importar la politica de scopes al workflow durable y mantiene replay determinista sin filesystem/Git/runtime.
Alternativas: Importar este modulo al workflow; hacer obligatorio el gate dentro de `RequestAgent`; crear scheduler en el core.
Impacto: NCW-047 deja el gate durable disponible; hacer obligatorio el gate antes de `RequestAgent` queda como decision futura del director.
Estado: aceptada tras promocion NCW-047
```

```text
Fecha: 2026-05-06
Decision: La unidad de concurrencia es tarea/grupo/write-set, no proceso ni agente real.
Motivo: Dos procesos pueden ser seguros si no pisan scopes; un mismo agente puede ser inseguro si cambia scopes ajenos.
Alternativas: Bloquear por agente; bloquear por modulo entero; confiar en Git al final.
Impacto: El detector emite `WorksetConflictV0` y `ParallelGroupPlanV0`; director decide como materializar.
Estado: aceptada inicial
```
