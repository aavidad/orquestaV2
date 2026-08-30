# Corte V26 / TLS-05: SDK público del catálogo de tools

Fecha: 2026-08-21. Estado: incremento implementado y ejercitado, no cableado
ni acreditado. `TLS-05`, `EXT-09` y `AC-V26-TOOLS-SKILLS-SDK` permanecen en
`declared`, `declared` y `planned`.

## Autoridad y write-set

- capability: `TLS-05`; no se atribuye `EXT-09`;
- invariante: el SDK proyecta el registro TLS-01 y nunca valida contra una
  segunda tabla ni acepta handlers/lifecycle;
- autoridad que escribe: ninguna; `tooling.Registry` sigue siendo inmutable;
- puertos/adaptadores: ninguno;
- write-set: `sdk/tools/**`, `internal/tooling/**`,
  `acceptance/v26_tool_sdk_test.go` y este documento;
- dependencias: V08, V10, V15, V20 y el corte TLS-01 local;
- presupuesto: 450 LOC productivas, 300 de tests, 120 documentales, cero
  dependencias/adapters y menos de 1 MiB.

Quedan fuera invocación, autorización application, efectos, receipts durables,
artifact spill, transports y conectores reales.

## Preflight y caracterización del hueco

Se ejecutó antes del diseño:

```text
scripts/preflight_reutilizacion_legacy.sh --capability TLS-05 --path sdk/tools \
  --operation implementar --task 'Exponer un Tool SDK público y neutral que proyecte el registro TLS-01 sin duplicar validación, autoridad, handlers ni lifecycle' \
  --function 'NewCatalog/Lookup/List/Digest/DecodeSpec'
```

Resultado advisory: `characterize`, sin candidato semántico, ledger lead ni
función legacy. Se registró el hueco y no se abrió código antiguo. Como patrón
V2 se caracterizó `sdk/commands` acreditado por V20: DTO público deliberado,
JSON estricto, errores máquina, copias defensivas y cero autoridad oculta. El
contrato de tools permanece distinto del de commands.

## Incremento real

`sdk/tools` añade DTOs públicos sin exponer tipos internos y un `Catalog`
read-only que convierte cada manifest y delega validación atómica a
`tooling.NewRegistry`. Ofrece:

- `NewCatalog`, `Lookup`, `List` y `Digest` exactos;
- `DecodeSpec` para un único JSON estricto, sin campos desconocidos ni valores
  concatenados;
- tipos neutrales para schemas, permisos, coste entero, idempotencia y receipt;
- errores estables `toolsdk.spec_encoding_invalid`, `spec_invalid` y
  `spec_duplicate` compatibles con `errors.Is`;
- normalización y digests idénticos a TLS-01;
- copias profundas: mutar el manifest, lookup o list no modifica el catálogo;
- resolución de versión exacta, sin latest implícito.

El paquete no contiene proceso, red, filesystem, credencial, provider,
handler, retry ni regla de lifecycle.

## Verificación

Verde:

```text
go test -mod=vendor -count=1 ./internal/tooling ./sdk/tools ./acceptance -run 'Test(PublicCatalog|DecodeSpec|V26TLS0|Registry)'
go test -mod=vendor -race -count=1 ./internal/tooling ./sdk/tools ./acceptance -run 'Test(PublicCatalog|DecodeSpec|V26TLS0|Registry)'
go test -mod=vendor -count=1 ./sdk/...
go test -mod=vendor -count=1 . ./acceptance -run '^(TestRebuildArchitecture|TestProductRoadmapIsExhaustiveAndCausal|TestV26TLS0)'
GOFLAGS=-mod=vendor go vet ./internal/... ./cmd/orquesta ./sdk/...
```

La aceptación compara una spec pública y una interna construidas de forma
independiente: coinciden los digests de spec y catálogo. El negativo sin
permisos no llega al catálogo público.

El mínimo transversal reprodujo fallos ajenos: receipts Codex obsoletos,
`scripts/continuar_orquestav2_home_codex.sh`, perfiles Codex no disponibles y
toolchain configurado inválido. SQLite pasó en 76.354 s; `internal/tooling`,
`sdk/commands`, `sdk/tools` y `cmd/orquesta` quedaron verdes. Un primer
`TestTerminateCommandProcessGroupReapsChildGroup` falló y su repetición exacta
5× pasó; no se modificó ni atribuyó ese módulo.

## Cierre honesto

- hecho: frontera pública neutral y estricta sobre el registro TLS-01;
- invariante restaurado: manifest interno/externo comparte validación y hash;
- autoridad final: registro inmutable; application conserva autorización y
  efectos futuros;
- tests/negativos/mutaciones/E2E: unitarios, aceptación parcial y race verdes;
  no hay E2E de ejecución;
- receipts/revisión: ninguno; no existe candidato V26;
- código/legacy retirado: ninguno; retirada bloqueada hasta consumidores y
  cutover acreditados;
- LOC/complexidad: 230 LOC productivas y 204 de tests; 10 funciones
  productivas, conversión explícita sin reflection;
- riesgos: sin P0. P1: el SDK solo cubre catálogo; faltan admission RBAC,
  invocación, receipt, artifact spill y dos consumidores reales;
- siguiente dependencia causal: admission/ejecución TLS-01/TLS-05 en
  application con negativo de cero llamadas no autorizadas; después `EXT-09`
  debe probar un conector interno y otro externo sobre el mismo SDK.

No se promueve ninguna capability ni se fabrica evidencia.
