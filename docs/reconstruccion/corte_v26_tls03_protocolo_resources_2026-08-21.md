# Corte V26 / TLS-03: protocolo causal de resources paginados

Fecha: 2026-08-21. Estado: incremento implementado y ejercitado, sin lectura
real ni acreditación. `TLS-03` y `AC-V26-TOOLS-SKILLS-SDK` permanecen en
`declared` y `planned`; no cambian roadmap, capabilities ni evidence.

## Autoridad y write-set

- capability ID: `TLS-03`;
- invariante: una página pertenece a un resource/spec/snapshot exactos y solo
  puede continuarse con el par opaco cursor + snapshot;
- autoridad: `SurfaceCatalog` valida contratos inmutables; no lee datos ni
  escribe lifecycle/estado;
- puertos/adaptadores/stores/procesos: ninguno;
- write-set: `internal/tooling/{surfaces,resource_pages}*`, ajustes a fixtures
  TLS-02, `acceptance/v26_resource_pages_test.go` y este documento;
- dependencias: V08, V10, V15 y V21 acreditadas; cortes TLS-01/02 locales no
  acreditados;
- presupuesto: 420 LOC productivas, 420 de tests, 120 documentales, cero
  dependencias externas y menos de 1 MiB.

Quedan fuera reader/provider, autorización application, persistencia,
notificaciones de suscripción, transporte MCP, SDK público, bootstrap y E2E.

## Preflight y caracterización

Se ejecutó antes de editar:

```text
scripts/preflight_reutilizacion_legacy.sh --capability TLS-03 \
  --path internal/tooling/resource_pages.go --operation implementar \
  --task 'Definir requests y páginas de resources exactos, paginados por cursor y ligados a snapshot/spec, con límites y declaración explícita de suscripción, sin handler, store ni transporte' \
  --function 'NewResourcePageRequest/ValidateResourcePage'
```

Resultado advisory: `reimplement`, sin función exacta mapeada. Se abrió T198
desde el índice Git congelado. Su valor útil es descriptor con owner/freshness,
presupuesto y ausencia de HOME/tokens/prompts/payloads crudos. No contiene
prueba de paginación o suscripción: el cierre legacy `documented_green` no se
acepta como equivalencia. No se copió código.

## Incremento real

`ResourceSpec` añade schema de item, límites default/máximo/bytes y política de
suscripción explícita `none|snapshot_changed`. Todo forma parte del digest
inmutable del descriptor y del catálogo.

`NewResourcePageRequest`:

- resuelve solo ID/revisión exactos y deriva URI/spec digest del catálogo;
- normaliza límite cero al default y rechaza superar el máximo;
- trata cursor y snapshot como refs opacas compactas;
- exige ambos vacíos para la primera página y ambos presentes para reanudar.

`ValidateResourcePage` exige identidad, URI, spec digest y cursor solicitante
exactos. La primera respuesta fija un snapshot; las siguientes deben repetirlo.
Rechaza cursor sin progreso, página incompleta vacía, cursor en página completa,
exceso de items/bytes y cada item fuera del schema. Devuelve JSON canónico con
copias defensivas.

Durante el focal se observó y corrigió un bug de copia: el slice se reemplazaba
antes de copiar sus elementos y devolvía items vacíos. El test conserva la
mutación y verifica contenido canónico y ausencia de alias.

`snapshot_changed` solo declara que el resource admite esa señal. No existe
aún suscripción/notification real ni receipt durable, y no se atribuye como tal.

## Verificación

Verde:

```text
go test -mod=vendor -count=1 ./internal/tooling ./acceptance -run '^(TestSurfaceCatalog|TestResource|TestV26TLS0[23])'
go test -mod=vendor -race -count=1 ./internal/tooling ./acceptance -run '^(TestSurfaceCatalog|TestResource|TestV26TLS0[23])'
go test -mod=vendor -count=1 ./internal/tooling ./internal/application ./sdk/tools ./acceptance -run '^(TestRegistry|TestSurfaceCatalog|TestResource|TestInvokeTool|TestPublicCatalog|TestDecodeSpec|TestV26TLS0|TestV26ToolExecution)'
go test -mod=vendor -count=1 . -run '^(TestRebuildArchitecture|TestProductRoadmapIsExhaustiveAndCausal)$'
GOFLAGS=-mod=vendor go vet ./internal/tooling
git diff --check
```

Los negativos cubren spec inválida, page/subscription policy, request forjado,
refs con control/espacios, deriva causal, snapshot cambiado, cursor estancado,
schema/null, límites y mutación del caller.

El mínimo transversal dejó verdes tooling, application, SQLite, MCP,
`cmd/orquesta` y `vet`. Persisten únicamente los fallos ajenos ya
inventariados: receipt Codex obsoleto, script concurrente no versionado,
perfiles Codex no disponibles y toolchain bootstrap inválido.

El mínimo transversal dejó verdes tooling, application, SQLite, MCP,
`cmd/orquesta` y `vet`. Persisten solo los fallos ajenos ya inventariados:
receipt Codex obsoleto, script concurrente no versionado, perfiles Codex no
disponibles y toolchain bootstrap inválido; no se tocaron ni regeneraron.

## Cierre honesto

- hecho: protocolo determinista y causal de descriptor/request/page;
- invariante restaurado: paginar no mezcla snapshots ni convierte el resource
  en tool;
- autoridad final: catálogo inmutable, sin writer nuevo;
- tests/E2E: unitarios, aceptación parcial, mutación, arquitectura y race
  verdes; sin E2E de lectura/suscripción;
- receipts/revisión acreditada: ninguno; no existe candidato V26;
- código/legacy retirado: ninguno; cutover bloqueado;
- LOC: 173 productivas y 259 de tests;
- riesgos: sin P0. P1: falta autorización, provider real y señal durable de
  cambio de snapshot;
- siguiente dependencia causal: admission de lectura en application con
  principal/proyecto explícitos y cero provider calls ante denegación; después
  TLS-04 puede proyectar namespaces lazy sin otro catálogo.

No se atribuye `TLS-03`, V26 ni otra capability como acreditada.

## Ratchet de aceptación de cierre rápido (2026-08-22)

La continuación paginada se prueba ahora contra trasplantes independientes de
ID, revisión, URI, cursor solicitante y snapshot. También se rechazan una
página completa que anuncie otro cursor y un límite solicitado superior al
máximo del descriptor. Cada mutación parte de una segunda página válida para
evitar que un fallo distinto o una fixture incompleta produzcan un falso verde.

El test sigue siendo contractual: no atribuye reader, suscripción durable,
transporte o autorización de application todavía inexistentes.
El fichero de aceptación asociado queda en 92 LOC; este documento, en 124 LOC.
