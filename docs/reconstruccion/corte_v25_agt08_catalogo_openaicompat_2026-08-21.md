# Corte V25 AGT-08: catálogo local OpenAI-compatible

Fecha: 2026-08-21.

Estado canónico: `declared`. Este incremento offline no acredita AGT-08, V25,
un runtime local ni `AC-V25-PROVIDER-ADAPTERS`.

## Alcance y autoridad

El adaptador consulta únicamente `GET /v1/models` detrás de
`ProviderCatalogSource` y traduce IDs exactos al contrato neutral. Conforme al
roadmap, AGT-08 es local: el endpoint solo puede ser literal loopback `127/8`,
literal `::1` o `localhost`, fijado sin DNS a `127.0.0.1`. Se rechazan redes
privadas no loopback, CGNAT, link-local, documentación, benchmark, reservadas,
públicas y nombres arbitrarios. No existe autorización remota en este corte.

AGT-08 y AGT-07 consumen la misma política y transporte de
`internal/adapters/agent/localhttp`. Ese paquete crea desde cero dialer,
transporte y cliente; la URL efectiva contiene la IP loopback autorizada y el
dialer falla si se le solicita otra IP o puerto. No hereda `DialContext`, proxy,
cookies, redirects ni hooks TLS del caller, y no usa DNS susceptible de
rebinding.

La compatibilidad nominal y `/v1/models` no observan tools, streaming,
capabilities, reasoning efforts, cuota, uso, disponibilidad ni calidad. Esos
campos permanecen vacíos o `unknown`. `application` conserva routing, fallback
y lifecycle; el adaptador no lanza procesos, ejecuta inferencia ni persiste
estado.

## Caracterización, límites y negativos

El preflight AGT-08 ejecutado antes de editar recomendó `reimplement`: había
una pista léxica, ninguna función exacta y ninguna prueba histórica de
funcionamiento. Solo se conserva la frontera neutral y la configuración local
explícita; no se copió código legacy.

El transporte impone deadline propio, cabecera máxima de 64 KiB, cuerpo
configurable con techo duro de 4 MiB, proxy desactivado y redirects cerrados.
JSON malformado o con contenido posterior, IDs duplicados, límites excedidos,
estado no-2xx, cancelación, retroceso de reloj o ventana caducada fallan sin
observación parcial. El negativo de remapeo comprueba que la IP autorizada
gobierna el socket real.

## Write-set, gates y estado honesto

El write-set comprende el adaptador y sus tests, la aceptación AGT-08, este
corte y el paquete `localhttp` compartido con AGT-07. El presupuesto se
mantiene en hasta 450 LOC de producción y 500 LOC de tests atribuibles al
corte, 120 minutos y menos de 200 MiB temporales.

Los gates son unitarios y aceptación offline, negativos de red y límites,
`-race` focal, `go vet`, guard de arquitectura, compilación de todos los
paquetes y `git diff --check` cuando el WIP concurrente lo permita.

No hay wiring de bootstrap, credenciales, lanzamiento, inferencia, streaming,
tools, E2E real, smoke de vLLM u otro runtime, receipt ni cambios de
roadmap/evidence/capabilities. La siguiente dependencia causal es la
composición explícita y después un smoke real aislado sobre el mismo candidato
sellado; hasta entonces AGT-08 permanece `declared`.
