# Plan de troceo incremental de hubs Orquesta

Estado: cierre operativo de `ARCH-ORQ-20260630-001` y
`ARCH-ORQ-20260630-002`.

## Lectura actual

- `orquesta-core-workflow`: 154 ficheros `.go` no-test en un paquete plano.
- `orquesta-orchestration-core`: 144 ficheros `.go` no-test en un paquete plano.
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

- `orquesta-core-workflow`: separar por invariantes, no por estetica:
  comandos/eventos, reducer/replay, waits/outbox, tasks/closure y tests
  contractuales.
- `orquesta-orchestration-core`: mantener como composicion de puertos del
  Director V2; no introducir runtime real, Codex, OPES, HTTP ni DB.
- `orquesta-web/nueva_app_html_render_v0.go`: extraer cuando se vuelva a tocar
  Nueva App; no reescribir la UI sin captura/prueba de contrato.
- `orquesta-app-codex-stack`: dividir validadores OPES/goal-first por contrato
  de dominio cuando aparezca un bug nuevo en ese contrato.

## Evidencia de cierre

- `go test -count=1 . -run 'TestNeutralOrchestrationPackagesDoNot(ImportProductAdapters|DependOnProductAdapters|DependOnFactoryOrHTTP)'`.
- `go test -count=1 ./...`.
- Inventario vivo actualizado; los riesgos quedan cerrados como deuda gobernada,
  no como codigo refactorizado.
