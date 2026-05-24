# Tareas locales: orquesta-context

Cada tarea debe ser pequena y cerrada.

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
