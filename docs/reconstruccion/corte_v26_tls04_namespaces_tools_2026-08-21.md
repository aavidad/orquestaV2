# Corte V26 / TLS-04: descubrimiento diferido por namespaces

Fecha: 2026-08-21. Estado: incremento implementado y ejercitado, no cableado
ni acreditado. `TLS-04` y `AC-V26-TOOLS-SKILLS-SDK` permanecen en `declared` y
`planned`; no cambian roadmap, capabilities ni evidence.

## Autoridad y write-set

- capability ID: `TLS-04`;
- invariante: el descubrimiento inicial publica solo namespaces compactos; la
  spec se carga después desde el registro TLS-01 exacto;
- autoridad: `tooling.Registry` continúa como única fuente. `ToolDiscovery` es
  un read-model derivado que guarda solo ID/revisión;
- puertos/adaptadores/stores/procesos: ninguno;
- write-set: `internal/tooling/namespaces*`,
  `acceptance/v26_tool_namespaces_test.go` y este documento;
- dependencias: V08, V10, V15 y V21 acreditadas; corte TLS-01 local no
  acreditado;
- presupuesto: 260 LOC productivas, 320 de tests, 110 documentales, cero
  dependencias y menos de 1 MiB.

Quedan fuera proyección MCP/HTTP/CLI, SDK público, búsqueda, selección por
modelo, configuración, bootstrap y E2E real.

## Preflight y hueco caracterizado

Se ejecutó antes de editar:

```text
scripts/preflight_reutilizacion_legacy.sh --capability TLS-04 \
  --path internal/tooling/namespaces.go --operation implementar \
  --task 'Derivar del registro TLS-01 un índice de namespaces pequeños que publique solo nombre, cantidad y digest antes de cargar specs exactas, sin catálogo paralelo ni transporte' \
  --function 'NewToolDiscovery/ListNamespaces/ListNamespace'
```

Resultado advisory: `characterize`; no existe candidato semántico, función
legacy, ledger lead ni ficha profunda. El hueco queda registrado y no se abrió
ni copió código antiguo.

## Incremento real

`NewToolDiscovery` deriva el namespace del primer segmento del ID canónico,
ordena nombres e identidades y produce digests SHA-256 estables ligados al
registro exacto. Conserva solo identidades privadas; cada `ListNamespace`
resuelve de nuevo las registrations mediante `Registry.Lookup`.

`ListNamespaces` devuelve exclusivamente `name`, `tool_count` y `digest`, sin
schema, permisos, coste, handler ni payload. `ListNamespace` devuelve copias
defensivas en orden ID/revisión y nunca elige latest implícito.

El límite contractual `MaxToolsPerNamespace=32` hace ejecutable la palabra
“pequeño”. Un namespace mayor falla durante la construcción completa y debe
dividirse; no se trunca ni se publica parcialmente. El límite es protocolo, no
configuración de despliegue ni clave ad hoc.

Este read-model no obliga aún a los transports a usar descubrimiento lazy, por
lo que no acredita TLS-04.

## Verificación

Verde:

```text
go test -mod=vendor -count=1 ./internal/tooling ./acceptance -run '^(TestToolDiscovery|TestV26TLS04)'
go test -mod=vendor -race -count=1 ./internal/tooling ./acceptance -run '^(TestToolDiscovery|TestV26TLS04)'
go test -mod=vendor -count=1 ./internal/tooling ./internal/application ./sdk/tools ./acceptance -run '^(TestRegistry|TestSurfaceCatalog|TestResource|TestToolDiscovery|TestInvokeTool|TestPublicCatalog|TestDecodeSpec|TestV26TLS0|TestV26ToolExecution)'
go test -mod=vendor -count=1 . -run '^(TestRebuildArchitecture|TestProductRoadmapIsExhaustiveAndCausal)$'
GOFLAGS=-mod=vendor go vet ./internal/tooling
git diff --check
```

Los negativos cubren registry nil, namespace ausente, 33 entradas, orden de
construcción, cambio de revisión y mutación del caller. Aceptación comprueba que
el índice serializado no contiene schemas ni permisos.

El mínimo transversal dejó verdes tooling, application, SQLite, MCP,
`cmd/orquesta` y `vet`. Persisten solo los fallos ajenos inventariados: receipt
Codex obsoleto, script concurrente no versionado, perfiles Codex no disponibles
y toolchain bootstrap inválido; no se tocaron ni regeneraron.

## Cierre honesto

- hecho: índice compacto y carga diferida por namespace;
- invariante restaurado: no aparece un segundo catálogo de specs;
- autoridad final: registro TLS-01 inmutable;
- tests/E2E: unitarios, aceptación parcial, arquitectura y race verdes; sin
  binding público ni E2E;
- receipts/revisión acreditada: ninguno; no existe candidato V26;
- código/legacy retirado: ninguno; cutover bloqueado;
- LOC: 138 productivas y 173 de tests;
- riesgos: sin P0. P1: SDK y transports todavía pueden listar el catálogo
  completo porque no consumen este read-model;
- siguiente dependencia causal: proyectar el mismo índice en TLS-05 y luego en
  un binding fino, sin duplicar límite, orden o digest.

No se atribuye `TLS-04`, V26 ni otra capability como acreditada.

## Ratchet de aceptación de cierre rápido (2026-08-22)

La aceptación comprueba ahora que un namespace ausente no resuelve, que las
registrations devueltas son copias defensivas y que cambiar una revisión cambia
el digest del namespace afectado. Un segundo caso construye exactamente
`MaxToolsPerNamespace + 1` IDs canónicos bajo el mismo prefijo y exige rechazo
atómico con `tooling.tool_namespace_too_large`, sin truncamiento publicable.

El incremento continúa siendo un read-model interno: no acredita por sí solo
que MCP, HTTP, CLI o el SDK adopten descubrimiento diferido.
El fichero de aceptación asociado queda en 103 LOC; este documento, en 108 LOC.
