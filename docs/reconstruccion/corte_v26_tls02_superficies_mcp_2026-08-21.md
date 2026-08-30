# Corte V26 / TLS-02: separación de tools, resources y prompts

Fecha: 2026-08-21. Estado: incremento implementado y ejercitado, no cableado
ni acreditado. `TLS-02` y `AC-V26-TOOLS-SKILLS-SDK` permanecen en `declared` y
`planned`; roadmap, capabilities y evidencias de release no cambian.

## Autoridad y alcance

- capability ID: `TLS-02`;
- invariante: una tool es invocable, un resource describe contexto
  direccionable de solo lectura y un prompt describe texto gobernado; ninguna
  clase puede presentarse como otra;
- autoridad: `tooling.Registry` sigue siendo la única fuente de tools. El nuevo
  `SurfaceCatalog` es inmutable y solo conserva una referencia a ese registro;
- puertos/adaptadores/stores/procesos: ninguno;
- write-set: `internal/tooling/surfaces*`,
  `acceptance/v26_surface_separation_test.go` y este documento;
- dependencias: V08, V10, V15 y V21 acreditadas, más los cortes TLS-01/TLS-05
  locales aún no acreditados;
- presupuesto: 450 LOC productivas, 400 de tests, 120 documentales, cero
  dependencias externas y menos de 1 MiB.

Quedan fuera registro MCP productivo, handlers, lectura/render real,
paginación/suscripción TLS-03, namespaces TLS-04, SDK público de resources o
prompts, bootstrap y el gate V26.

## Preflight y caracterización

Se ejecutó antes de editar:

```text
scripts/preflight_reutilizacion_legacy.sh --capability TLS-02 \
  --path internal/tooling/surfaces.go --operation implementar \
  --task 'Separar tools invocables, resources direccionables y prompts renderizables mediante contratos tipados de descubrimiento, sin handlers, transporte MCP, store ni segundo registro de tools' \
  --function 'NewSurfaceCatalog/LookupTool/LookupResource/LookupPrompt'
```

Resultado advisory: `reimplement`. Se abrió desde el índice Git congelado la
función exacta `MCPTransportResourcesV0` y las tareas MCP-002/003/008/010. Se
caracterizó además `MCPNuevaAppPromptV0Value` para cubrir el hueco de prompts.

Valor conservado: envelopes y formas distintas, resource con URI/media type y
prompt declarativo con argumentos. Fallos descartados: registro manual que
solo incluía resources/tools, handlers dentro del descriptor, prompt fuera del
registro, literales humanos locales y presupuestos de transporte mezclados con
la taxonomía. No se copió código legacy.

## Incremento real

`SurfaceCatalog` recibe el registro TLS-01 exacto y specs independientes de
resource y prompt. La construcción es atómica, ordenada por ID/revisión y
produce hashes estables ligados también a la clase de superficie.

- `LookupTool/ListTools` delegan sin copiar otro catálogo de tools;
- `ResourceSpec` contiene identidad, URI absoluta sin query/userinfo/fragment,
  media type, permisos RBAC y máximo de bytes, pero no handler/schema de
  invocación/receipt/idempotencia;
- `PromptSpec` contiene identidad, clave i18n `prompt.*` y schema cerrado de
  argumentos, pero no handler/permisos/coste/receipt/URI;
- no existe `Lookup` genérico: cada clase tiene lookup/list tipados;
- el mismo ID/revisión puede existir en las tres clases sin colisión ni
  conversión implícita;
- todas las listas/lookups devuelven copias defensivas;
- `ValidatePromptInput` canoniza argumentos y resuelve solo versión exacta.

Este contrato no obtiene datos, no renderiza texto, no autoriza una lectura y
no ejecuta application. Por tanto no acredita todavía una superficie MCP.

## Verificación

Verde:

```text
go test -mod=vendor -count=1 ./internal/tooling ./acceptance -run '^(TestSurfaceCatalog|TestV26TLS02)'
go test -mod=vendor -race -count=1 ./internal/tooling ./acceptance -run '^(TestSurfaceCatalog|TestV26TLS02)'
go test -mod=vendor -count=1 ./internal/tooling ./internal/application ./sdk/tools ./acceptance -run '^(TestRegistry|TestSurfaceCatalog|TestInvokeTool|TestPublicCatalog|TestDecodeSpec|TestV26TLS0|TestV26ToolExecution)'
go test -mod=vendor -count=1 . -run '^(TestRebuildArchitecture|TestProductRoadmapIsExhaustiveAndCausal)$'
GOFLAGS=-mod=vendor go vet ./internal/tooling
git diff --check
```

Los negativos cubren identidad/revisión, URI, media type, permiso, máximo,
template key, schema, duplicados por clase, versión/payload de prompt y ausencia
de lookup genérico/campos cruzados. Race cubre las proyecciones inmutables.

El mínimo transversal se ejecutó. Tooling, application, SQLite, MCP y
`cmd/orquesta` quedaron verdes; `vet` también. Persisten únicamente los fallos
ajenos ya inventariados: receipt Codex obsoleto, script concurrente no
versionado, perfiles Codex no disponibles y toolchain bootstrap inválido. No
se modificaron ni se regeneraron.

## Cierre honesto

- hecho: frontera tipada e inmutable para las tres clases MCP;
- invariante restaurado: resource/prompt no son tools disfrazadas y tools no
  tienen segundo registro;
- autoridad final: registro TLS-01 para tools; catálogo read-only para
  clasificación;
- tests/E2E: unitarios, aceptación parcial, mutaciones estructurales,
  arquitectura y race verdes; sin servidor/conector E2E;
- receipts/revisión acreditada: ninguno; no existe candidato V26;
- código/legacy retirado: ninguno; cutover bloqueado;
- LOC/complexidad: 339 productivas y 280 de tests; validación aislada por clase;
- riesgos: sin P0. P1: resources aún no son paginados/suscribibles y prompts no
  tienen renderer/adaptador real;
- siguiente dependencia causal: `TLS-03` debe añadir lectura paginada y scope
  exacto detrás de un puerto consumidor sin convertirla en tool.

No se atribuye `TLS-02`, V26 ni otra capability como acreditada.

## Ratchet de aceptación de cierre rápido (2026-08-22)

La aceptación conserva además tres negativos públicos: una identidad exclusiva
de resource no puede resolverse como prompt, una versión de prompt inexistente
no selecciona otra revisión y ningún lookup devuelve memoria aliasada que
permita mutar el catálogo. El caso positivo mantiene el mismo ID/revisión en
tool, resource y prompt para demostrar separación por clase, no por convención
de nombres.

Este refuerzo no añade wiring MCP ni cambia el estado declarado de TLS-02.
El fichero de aceptación asociado queda en 95 LOC; este documento, en 121 LOC.
