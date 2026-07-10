# orquesta-tool-capability

Contrato neutral para catalogar, resolver, validar y preparar capabilities/tools
reutilizables para aplicaciones generadas.

El modulo solo contiene DTOs, casos de uso y puertos. No instala bundles ni
importa runtime, orchestration core, MCP, HTTP, filesystem, red o secretos. Los
efectos reales y los receipts durables pertenecen a adaptadores opt-in.

El request solo declara refs e intencion. Binding autorizado, manifest y
snapshot verificado se resuelven por puertos independientes. Stores concretos
persisten plan+receipt, claim/lease de destino y transiciones CAS. El instalador
consume el handle opaco del snapshot y debe ser idempotente/reconciliable por
`operation_ref`. El core no promete exactly-once distribuido.

Patron y checklist: `docs/patron_sdk_tools_capabilities_orquesta_2026-07-10.md`.

Validacion:

```sh
go test -race -count=1 ./modulos/orquesta-tool-capability
go test -race -count=1 ./modulos/orquesta-tool-capability-file
```
