# Corte V26 / TLS-05: discovery lazy en el SDK de tools

Fecha: 2026-08-21. Estado: incremento implementado y ejercitado, no cableado
ni acreditado. `TLS-04`, `TLS-05` y `AC-V26-TOOLS-SKILLS-SDK` permanecen en
`declared`, `declared` y `planned`.

## Autoridad y write-set

- capability ID: `TLS-05`, consumiendo el corte local `TLS-04`;
- invariante: límite, orden, clasificación y digest tienen un único owner
  interno; el SDK solo convierte DTOs públicos;
- autoridad: `tooling.Registry` y su `ToolDiscovery` derivado;
- puertos/adaptadores/stores/procesos: ninguno;
- write-set: `sdk/tools/discovery*`, ajuste acotado del catálogo público de
  errores en `sdk/tools/catalog.go`,
  `acceptance/v26_tool_sdk_discovery_test.go` y este documento;
- presupuesto: 180 LOC productivas, 240 de tests, 100 documentales, cero
  dependencias externas y menos de 1 MiB.

Quedan fuera bindings MCP/HTTP/CLI, bootstrap, provider y E2E público real.

## Preflight y hueco

Se ejecutó antes de editar:

```text
scripts/preflight_reutilizacion_legacy.sh --capability TLS-05 \
  --path sdk/tools/discovery.go --operation ampliar \
  --task 'Exponer en el SDK público el índice lazy TLS-04 y la carga de un namespace sin duplicar specs, límite, orden, validación ni digests' \
  --function 'NewDiscovery/ListNamespaces/ListNamespace'
```

Resultado advisory: `characterize`, sin candidato, ledger lead ni función
legacy. El hueco se implementó sobre los owners V2 ya ejercitados; no se abrió
ni copió código antiguo.

## Incremento real

`tools.NewDiscovery` recibe el `Catalog` público existente y delega en
`tooling.NewToolDiscovery` usando el mismo registro inmutable. Expone:

- `Namespace{name, tool_count, digest}` sin schema/permisos/handler;
- `ListNamespaces`, `ListNamespace` y `Digest` read-only;
- el mismo límite contractual 32, sin constante o configuración divergente;
- registrations públicas convertidas desde el lookup interno exacto;
- copias defensivas y orden ID/revisión estable;
- errores máquina `toolsdk.discovery_invalid` y
  `toolsdk.namespace_too_large` mediante el único `tools.ErrorCode`.

La aceptación construye manifest internos y públicos independientemente y
demuestra paridad de catálogo, namespaces y tool digests. El orden de entrada
distinto no altera el resultado.

## Verificación

Verde:

```text
go test -mod=vendor -count=1 ./sdk/tools ./acceptance -run '^(TestPublicDiscovery|TestV26TLS05PublicDiscovery)'
go test -mod=vendor -race -count=1 ./sdk/tools ./acceptance -run '^(TestPublicDiscovery|TestV26TLS05PublicDiscovery)'
go test -mod=vendor -count=1 ./internal/tooling ./internal/application ./sdk/... ./acceptance -run '^(TestRegistry|TestSurfaceCatalog|TestResource|TestToolDiscovery|TestInvokeTool|TestPublicCatalog|TestPublicDiscovery|TestDecodeSpec|TestV26TLS0|TestV26ToolExecution)'
go test -mod=vendor -count=1 . -run '^(TestRebuildArchitecture|TestProductRoadmapIsExhaustiveAndCausal)$'
GOFLAGS=-mod=vendor go vet ./internal/tooling ./sdk/tools
git diff --check
```

Los negativos cubren catálogo nil, namespace ausente, 33 registros, catálogo
vacío y mutación de DTO/spec devueltos.

El mínimo transversal dejó verdes tooling, application, SQLite, MCP,
`cmd/orquesta`, SDK y `vet`. Persisten solo receipt Codex obsoleto, el script
concurrente no versionado, perfiles Codex no disponibles y toolchain bootstrap
inválido; no se tocaron ni regeneraron.

## Cierre honesto

- hecho: discovery TLS-04 proyectado sin duplicar semántica en TLS-05;
- invariante restaurado: SDK e interno comparten límite/orden/digests;
- autoridad final: registro/read-model interno inmutable;
- tests/E2E: unitarios, aceptación parcial y race verdes; sin binding/E2E;
- receipts/revisión acreditada: ninguno; no existe candidato V26;
- código/legacy retirado: ninguno;
- LOC: 73 productivas y 146 de tests;
- riesgos: sin P0. P1: ninguna superficie productiva consume aún discovery;
- siguiente dependencia causal: binding fino sobre este SDK/índice, con paridad
  y RBAC, o comenzar `TLS-06` sin mezclar el registro de skills con tools.

No se atribuye `TLS-04`, `TLS-05`, V26 ni otra capability como acreditada.
