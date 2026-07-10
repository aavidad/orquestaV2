# orquesta-presentation-extraction-tool-capability

Adaptador hexagonal que proyecta el descriptor neutral de extracción de
presentaciones PPTX/ODP al SDK `orquesta-tool-capability`. No implementa
parsers, drivers, filesystem, red, MCP, API ni proveedores.

La intención solo nombra un `presentation_bundle_ref`. Una autoridad resuelve
descriptor y registro; el adaptador obtiene un snapshot verificado por puerto y
delega claim/CAS/lease al SDK. Los efectos son idempotentes por `operation_ref`
y consumen el `snapshot.handle_ref` opaco del plan.
