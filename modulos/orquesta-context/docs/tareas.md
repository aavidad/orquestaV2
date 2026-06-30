# Tareas locales: orquesta-context

Cada tarea debe ser pequena y cerrada.

## CTX-007

Estado: completada 2026-06-30.

Objetivo: centralizar consultas de codigo para agentes y cerrar el riesgo de
`codebase-memory-mcp` duplicado o sin TTL.

Write-set:

- `code_context_broker_v0.go`
- `code_context_tool_lease_v0.go`
- tests y docs locales.

Contrato: `CodeContextQueryPortV0`, `CodeContextToolLeaseV0`.

Validacion:

- `go test -count=1 ./modulos/orquesta-context`

Resultado:

- cache central con fingerprint/dirty-worktree;
- dedupe in-flight de consultas concurrentes;
- `codebase-memory-mcp` exige opt-in y lease central;
- evaluador puro de lease TTL/CPU para que servidor/composicion pueda parar o
  alertar sin que los agentes arranquen indexadores propios.
- contrato de completion `stopped` para que un watchdog de composicion pueda
  dejar evidencia durable de parada cooperativa.

Bloqueos:

- adaptador real `codebase-memory-mcp`, owner marker de proceso real, observador
  PID/CPU y smoke opt-in quedan en servidor/composicion.

## CTX-006

Estado: pendiente futuro.

Objetivo: evaluar y cablear un adaptador opt-in tipo Graphify/Obsidian para
alimentar un indice contextual compacto antes de lanzar agentes.

Write-set previsto:

- adaptador de composicion o MCP, no nucleo puro;
- docs de runbook y smoke real acotado.

Contrato futuro: el adaptador debe producir documentos compactos desde refs
opacas y evidencias. No debe leer contexto global sin scope, no debe copiar
documentos completos en prompts y no debe introducir rails por palabras.

Validacion futura:

- smoke con servidor temporal limpio;
- prueba de que consultas con palabras ambiguas generan resultados/advisories y
  no paran agentes.

Bloqueos: aplazado hasta cerrar los problemas previos de Orquesta: rails stale,
redaccion no bloqueante, tests transversales y flujo autonomo residente.

## CTX-004

Estado: completada documental.

Objetivo: reconciliar T15 para contexto materializado: required `ref_only`
debe resolverse por accion explicita, y rails de detalle solo bloquean campos
activados con evidencia de valor sensible o material crudo.

Write-set:

- `docs/tareas.md`
- `docs/decisiones.md`
- `docs/pruebas.md`
- `README.md`

Contrato: `ContextMaterializedBundleV0` conserva refs opacas y clasifica
required `ref_only` con `required_ref_action`; no transporta prompts,
transcripts, HOME, OAuth, tokens ni contenido crudo.

Validacion: `go test -count=1 ./modulos/orquesta-rails ./modulos/orquesta-core-workflow ./modulos/orquesta-context ./modulos/orquesta-director-agent ./cmd/orquesta-server`.

Bloqueos: ampliar scopes de contexto requiere matriz externa propia; T15 no se
reabre por docs historicas.

## CTX-000

Estado: completada inicial.

Objetivo: crear miniproyecto `orquesta-context` con contexto local y contrato inicial.

Write-set:

- `AGENTS.md`
- `README.md`
- `arrancar_codex.sh`
- `docs/contratos.md`
- `docs/tareas.md`
- `docs/pruebas.md`
- `docs/decisiones.md`

Validacion: `git diff --check -- modulos/orquesta-context`.

## CTX-001

Estado: completada ejecutable.

Objetivo: implementar builder puro `BuildContextBundleV0` para preparar contexto pequeno por modulo/fase/tarea.

Write-set:

- `context_bundle_types_v0.go`
- `context_bundle_validation_v0.go`
- `context_bundle_builder_v0.go`
- `context_bundle_helpers_v0.go`
- `context_bundle_v0_test.go`
- docs locales

Contrato: `ContextBundleV0`.

Validacion:

- `gofmt -w modulos/orquesta-context/*.go`
- `go test -count=1 ./modulos/orquesta-context`
- `git diff --check -- modulos/orquesta-context`

Bloqueos: no materializa refs por filesystem/MCP; eso debe ser adaptador futuro.

## CTX-002

Estado: completada ejecutable.

Objetivo: implementar materializacion explicita de ContextBundleV0 mediante puerto y conector filesystem seguro.

Write-set:

- `context_materialization_types_v0.go`
- `context_materialization_helpers_v0.go`
- `context_materialization_v0.go`
- `context_materialization_file_v0.go`
- `context_materialization_v0_test.go`
- `context_materialization_file_v0_test.go`
- docs locales

Contrato: `ContextMaterializedBundleV0`, `ContextRefReaderV0`, `FileContextRefStoreV0`.

Validacion:

- `gofmt -w modulos/orquesta-context/context_materialization*.go`
- `go test -count=1 ./modulos/orquesta-context`
- `git diff --check -- modulos/orquesta-context`

Bloqueos: no transforma el contexto en prompt de proveedor; runtime/Codex/Ollama/vLLM deben consumir el materializado por adaptador futuro.

## CTX-003

Estado: completada ejecutable.

Objetivo: definir puerto neutral de sanitizacion de contexto saliente y evidencia
durable sin dato sensible.

Write-set:

- `context_sanitizer_types_v0.go`
- `context_sanitizer_v0.go`
- `context_materialization_v0.go`
- `context_materialization_types_v0.go`
- `context_materialization_v0_test.go`
- docs locales

Contrato: `ContextSanitizerPortV0`, `ContextSanitizationEvidenceV0`.

Validacion:

- `go test -count=1 ./modulos/orquesta-context`

Bloqueos: no implementa IA local, proveedor ni transporte; los adaptadores
opt-in viven en composicion.
