# orquesta-document-extraction-tool-capability

Adaptador hexagonal que proyecta el descriptor neutral de extracción documental
al SDK `orquesta-tool-capability`. No implementa PDF, OCR, filesystem, red,
MCP, API ni proveedores.

La intención nombra solo un `bundle_ref`; una autoridad resuelve el descriptor
y su registro. El adaptador proyecta un `ToolBundleV0`, pide un snapshot
verificado por puerto y delega claim/CAS/lease al store del SDK. El efecto se
ejecuta mediante un puerto idempotente por `operation_ref` y recibe el plan con
el `snapshot.handle_ref` opaco.
