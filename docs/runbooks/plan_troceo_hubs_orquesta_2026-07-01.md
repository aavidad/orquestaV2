# Plan de troceo incremental de hubs Orquesta

Estado: cierre operativo de `ARCH-ORQ-20260630-001` y
`ARCH-ORQ-20260630-002`.

## Lectura actual

- `orquesta-core-workflow`: 154 ficheros `.go` no-test en un paquete plano.
- `orquesta-orchestration-core`: 144 ficheros `.go` no-test en un paquete plano.
- Metricas MEJ-106 medidas en HEAD 2026-07-03:
  `env_vars_orquesta=513`, `endpoints_status=16`,
  `interfaces_estado=65`, `modulos_director=17`.
- Frontera neutral vigente: `architecture_boundaries_test.go` ya cubre
  `orquesta-core-workflow` y `orquesta-orchestration-core`.
- Hotspots no-test actuales:
  - `modulos/orquesta-web/nueva_app_html_render_v0.go`: 1803 lineas.
  - `modulos/orquesta-app-codex-stack/goal_domain_receipt_closure_validator_v0.go`: 1147 lineas.
  - `modulos/orquesta-app-director-service/goal_first_v0.go`: 924 lineas.
  - `cmd/orquesta-server/codex_goal_app_server_v0.go`: 886 lineas.
  - `modulos/orquesta-mcp/autoprogramming_status_tool_v0.go`: 888 lineas.

## Decision

No hacer refactor masivo sin bug o feature concreto. Estos hubs son deuda
estructural, pero el repositorio esta verde y las fronteras neutral/producto
estan protegidas. La politica es trocear solo cuando se toque una zona por una
razon funcional verificable.

## Regla para cambios futuros

Cuando un cambio toque un fichero no-test de mas de 700 lineas:

1. Anadir o conservar test focal antes del cambio.
2. Extraer una responsabilidad pequena si el cambio aumenta acoplamiento.
3. No mover APIs publicas de `orquesta-core-workflow` sin migracion incremental.
4. Mantener `go test -count=1 . -run TestNeutralOrchestrationPackages` verde
   si se toca frontera de nucleo.
5. Para `nueva_app_html_render_v0.go`, priorizar extracciones por:
   catalogo/contrato de opciones, i18n/ayuda, render de secciones y validacion
   de formulario.

## Cortes sugeridos

- Primer subpaquete a extraer cuando haya cambio funcional:
  workflow tasks/microtasks de `orquesta-core-workflow`. Criterio de entrada:
  tocar `WorkflowTaskV0`, `WorkProfileV0`, `CreateMicrotask` o su validacion.
  Ficheros candidatos:
  `work_items_v0.go`, `work_items_validation_v0.go`,
  `work_items_normalize_v0.go`, `work_items_command_v0.go`,
  `work_items_command_flow_v0.go`, `work_items_command_invariant_v0_test.go`,
  `work_items_command_v0_test.go`, `work_items_v0_test.go`,
  `work_profile_v0.go`, `work_profile_task_v0.go`,
  `work_profile_validation_v0.go` y `work_profile_v0_test.go`.
  Ratchet documental: el paquete plano parte de 154 ficheros `.go` no-test y
  debe bajar tras la extraccion real; si se anade fachada temporal en el paquete
  actual, debe ser mas pequena que el grupo extraido. Ratchet focal:
  `go test -count=1 ./modulos/orquesta-core-workflow -run 'Test(NewWorkflowTask|ValidateWorkflowTask|WorkflowTaskFromWorkProfile|CreateMicrotask)'`
  y
  `go test -count=1 . -run 'TestNeutralOrchestrationPackagesDoNot(ImportProductAdapters|DependOnProductAdapters|DependOnFactoryOrHTTP)'`.
- `orquesta-orchestration-core`: mantener como composicion de puertos del
  Director V2; no introducir runtime real, Codex, OPES, HTTP ni DB.
- `orquesta-web/nueva_app_html_render_v0.go`: extraer cuando se vuelva a tocar
  Nueva App; no reescribir la UI sin captura/prueba de contrato.
- `orquesta-app-codex-stack`: dividir validadores OPES/goal-first por contrato
  de dominio cuando aparezca un bug nuevo en ese contrato.

## Ratchet MEJ-106

- `env_vars_budget_test.go` fija `env_vars_orquesta <= 513`, medido con
  `scripts/orquesta_metricas_deuda.sh --json`.
- Si el conteo baja, actualizar el maximo a la baja en el mismo commit que
  elimina o consolida variables.
- Si una variable nueva es inevitable, documentar primero por que no cabe en una
  superficie canonica existente y compensar eliminando/consolidando otra.
- Mantener `scripts/test_orquesta_metricas_deuda.sh` como test de forma del
  medidor; el ratchet de presupuesto vive en `go test`.

## Checklist retirada `legacy_director_loop`

No retirar codigo legacy por fecha blanda. Condiciones acumulativas:

1. Ventana §9 verde con nightly activo y sin reapertura de shutdown/status
   goal-first.
2. Segundo backend goal o decision explicita de que el director de escalada
   sustituye esa redundancia para produccion.
3. Smokes OPES temporales y Nueva App goal-first con cierre aceptado, refs no
   vacios y shutdown sin app-server residual.
4. Inventario sin bugs abiertos de compatibilidad que dependan de
   `legacy_director_loop`; los runbooks historicos deben quedar marcados como
   legacy explicito.
5. PR/commit de retirada con test que falle si una ruta productiva vuelve a
   seleccionar legacy de forma implicita.

## Evidencia de cierre

- `go test -count=1 . -run 'TestNeutralOrchestrationPackagesDoNot(ImportProductAdapters|DependOnProductAdapters|DependOnFactoryOrHTTP)'`.
- `go test -count=1 ./ -run TestEnvVarsBudget`.
- `go test -count=1 ./...`.
- `bash scripts/orquesta_metricas_deuda.sh`.
- Inventario vivo actualizado; los riesgos quedan cerrados como deuda gobernada,
  no como codigo refactorizado.
