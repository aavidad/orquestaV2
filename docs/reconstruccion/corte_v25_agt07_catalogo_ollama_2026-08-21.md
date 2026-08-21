# Corte V25 AGT-07: catálogo local Ollama

Fecha: 2026-08-21.

Estado canónico: `declared`. Este incremento offline no acredita AGT-07, V25,
disponibilidad de Ollama ni `AC-V25-PROVIDER-ADAPTERS`.

## Alcance y autoridad

El adaptador consulta únicamente `GET /api/tags` detrás de
`ProviderCatalogSource` y conserva los IDs exactos observados. El endpoint solo
puede ser loopback explícito: literal `127/8`, literal `::1` o `localhost`, que
se fija sin DNS a `127.0.0.1`. Se rechazan redes privadas no loopback, CGNAT,
link-local, documentación, benchmark, reservadas, públicas y nombres
arbitrarios.

La política y el transporte comunes viven en
`internal/adapters/agent/localhttp`: crean desde cero el `net.Dialer`, el
`http.Transport` y el `http.Client`. La URL efectiva contiene la IP loopback
autorizada y el dialer falla si el transporte solicita otro destino. No se
heredan `DialContext`, proxy, cookies, redirects ni hooks TLS del caller; no hay
resolución DNS que pueda rebindear el socket.

Una lista de modelos no prueba inferencia, salud, cuota, uso, capabilities ni
reasoning efforts. El adaptador publica disponibilidad y cuota `unknown`, uso
desconocido y listas de capabilities/efforts vacías. `application` conserva
routing, fallback y lifecycle; este corte no escribe Goals ni estado durable.

## Caracterización, límites y negativos

El preflight AGT-07 ejecutado antes de editar recomendó `reimplement`. Se
conserva la frontera neutral del adaptador y se descartan pull/serve, tmux como
autoridad, slots inferidos desde procesos y capacidades deducidas de nombres.
No se copió código legacy.

El transporte impone deadline propio, cabecera máxima de 64 KiB, cuerpo
configurable con techo duro de 4 MiB, proxy desactivado y redirects cerrados.
JSON malformado o con contenido posterior, IDs duplicados, límites excedidos,
estado no-2xx, cancelación, retroceso de reloj o ventana caducada fallan sin
observación parcial. El negativo de remapeo exige que la dirección solicitada
por el dialer coincida exactamente con la IP y puerto autorizados.

## Write-set, gates y estado honesto

El write-set del hardening comprende el adaptador y sus tests, la aceptación
AGT-07, este corte y el paquete `localhttp` compartido con AGT-08. El presupuesto
se mantiene en hasta 450 LOC de producción y 500 LOC de tests atribuibles al
corte, 120 minutos y menos de 200 MiB temporales.

Los gates son unitarios y aceptación offline, negativos de red y límites,
`-race` focal, `go vet`, guard de arquitectura, compilación de todos los
paquetes y `git diff --check` cuando el WIP concurrente lo permita.

No hay wiring de bootstrap, credenciales, lanzamiento, inferencia, gestión de
modelos, E2E real, smoke contra un daemon Ollama, receipt ni cambios de
roadmap/evidence/capabilities. La siguiente dependencia causal es la
composición explícita y después un smoke real aislado sobre el mismo candidato
sellado; hasta entonces AGT-07 permanece `declared`.
