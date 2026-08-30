# Corte V26 / TLS-01: contrato y registro canónico de tools

Fecha: 2026-08-21. Estado: incremento implementado y ejercitado, no cableado ni acreditado.
`TLS-01` y `AC-V26-TOOLS-SKILLS-SDK` permanecen respectivamente en
`declared` y `planned` en `product/roadmap.json`.

## Autoridad y alcance

- capability IDs: `TLS-01`;
- invariante: una versión de tool se describe una sola vez mediante
  `tooling.CapabilitySpec`; ninguna proyección, adapter o provider mantiene un
  catálogo paralelo;
- autoridad que escribe: ninguna nueva autoridad durable. El registro se
  construye atómicamente por composición y queda inmutable;
- puertos afectados: ninguno;
- adaptadores afectados: ninguno;
- write-set: `internal/tooling/**`, `acceptance/v26_tool_registry_test.go` y este documento;
- dependencias causales: V08 credentials, V10 identidad/RBAC, V15
  presupuestos/efectos y V20 registro de comandos, todas acreditadas;
- código antiguo que permitirá retirar: catálogo y schemas MCP ensamblados a
  mano, únicamente cuando existan paridad y ausencia de consumidores legacy;
- presupuesto aplicado: hasta 500 LOC productivas, 250 LOC de tests, 120 LOC
  documentales, cero dependencias/adapters nuevos y menos de 1 MiB de disco.

Quedan expresamente fuera dispatcher, ejecución, autorización en application,
persistencia, Tool SDK público, bindings MCP/HTTP/CLI, resources, skills,
plugins, artifact spill, instalación y el gate completo V26.

## Preflight y caracterización

Se ejecutó antes del diseño el preflight obligatorio:

```text
scripts/preflight_reutilizacion_legacy.sh --capability TLS-01 --path internal/tools \
  --operation implementar --task 'Definir el contrato y registro único de tools con schema, versión, permisos, coste, idempotencia y metadatos de receipt, sin ejecución ni adapters' \
  --function 'NewRegistry/Register/Lookup/List'
```

El resultado advisory fue `reimplement`. Se caracterizaron, mediante
`git show` y sin materializar el sparse legacy, los símbolos exactos
`MCPTransportToolInputFieldsV0` y `newMCPTransportToolDescriptorsV0`.

Conducta útil conservada: un descriptor deriva de una fuente canónica; schema,
versión, permisos, coste, idempotencia y receipts son explícitos; ningún
`switch`, enum o transporte mantiene otro catálogo.

Fallo descartado: el legado derivaba parte de los campos desde DTO, pero
mantenía por separado nombres, required, enums y descriptores. Una acción
directa podía existir sin ser descubrible por un cliente estricto.

La raíz final se ajustó a `internal/tooling`, fijada por el plan causal V26.
No se copió código legacy.

## Incremento real

`tooling.NewRegistry` valida todas las specs antes de publicar el registro y
falla de forma atómica ante una entrada inválida o una identidad
`tool ID + version` duplicada. El registro:

- exige IDs con namespace, revisiones enteras positivas y schemas JSON cerrados, incluidos objetos anidados;
- exige al menos un permiso conocido por el RBAC V10 y rechaza duplicados;
- declara una cota de coste con el `ResourceVector` canónico V15;
- admite solo `required + application receipt` y `read_reexecute + observation receipt`;
- normaliza schemas/permisos antes del hash, ordena por ID/revisión y resuelve solo versiones exactas;
- devuelve copias profundas, no expone mutación y liga spec/catálogo a digests `sha256:` estables.

Este registro solo describe admisibilidad. No autoriza ni ejecuta una tool y
no puede mutar un Goal.

## Verificación

Verde:

```text
go test -mod=vendor -count=1 ./internal/tooling ./acceptance -run 'Test(Registry|V26TLS01)'
go test -mod=vendor -race -count=1 ./internal/tooling ./acceptance -run 'Test(Registry|V26TLS01)'
go test -mod=vendor -count=1 ./acceptance -run 'TestV26TLS01'
go test -mod=vendor -count=1 . ./acceptance -run '^(TestRebuildArchitecture|TestProductRoadmapIsExhaustiveAndCausal)$'
GOFLAGS=-mod=vendor go vet ./internal/... ./cmd/orquesta
```

Los negativos cubren identidad, permisos, coste, receipt, JSON/schema cerrado,
required/constraints/keywords, duplicados y mutación posterior del caller.

El mínimo transversal también se ejecutó. `internal/tooling` y
`cmd/orquesta` quedaron verdes. El resultado global conserva fallos
preexistentes y ajenos al write-set:

- `go test -mod=vendor -count=1 .`: receipts Codex obsoletos y el fichero ajeno
  `scripts/continuar_orquestav2_home_codex.sh`;
- `go test -mod=vendor -count=1 ./internal/... ./cmd/orquesta`: perfiles Codex
  configurados no disponibles y `bootstrap.codex_go_toolchain_invalid`;
- SQLite quedó verde en 77.321 s y `internal/tooling` en 0.004 s dentro de esa
  misma invocación.

No se regeneraron receipts ni se corrigieron ficheros fuera del write-set.

## Cierre honesto

- hecho: contrato único e inmutable y registro determinista de specs de tools;
- invariante restaurado: catálogo, schema y gobierno ya no requieren fuentes paralelas en el paquete;
- autoridad final: registro read-only compuesto; application sigue siendo la
  única autoridad futura para autorizar y persistir efectos/receipts;
- tests/negativos/mutaciones/E2E: unitarios, aceptación parcial y race verdes; sin E2E de tools;
- receipts y revisión acreditada: ninguno; no hay candidato V26;
- código o decisión retirados: ninguno;
- legacy retirado o bloqueo de retirada: bloqueado hasta paridad SDK/consumers y cutover;
- LOC netas y complejidad: 459 LOC lógicas productivas y 226 LOC lógicas de
  tests antes de esta bitácora (excluyen blancos y comentarios; 628/303 líneas
  físicas respectivamente); 31 funciones productivas pequeñas, con la ramificación mayor
  aislada en la validación del dialecto JSON Schema;
- riesgos/P0/P1: sin P0 observado. P1: todavía no existe admission RBAC en
  application, dispatcher, receipt durable, artifact spill, SDK ni dos
  consumidores reales;
- siguiente dependencia causal: `TLS-05` debe consumir este contrato y demostrar paridad interna/externa;
  `TLS-04` puede proyectar después namespaces lazy sin crear otro registro.

No se atribuye `TLS-01`, V26 ni ninguna capability adicional como acreditada.
