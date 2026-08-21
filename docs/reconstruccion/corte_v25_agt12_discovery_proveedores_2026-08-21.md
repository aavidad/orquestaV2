# Corte V25 AGT-12: discovery neutral de proveedores

Fecha: 2026-08-21.

Estado canónico: `declared`. Este incremento query-only no acredita AGT-12,
no sella `AC-V25-PROVIDER-ADAPTERS` y no contiene E2E real de proveedor.

## Alcance y autoridad

- capability: `AGT-12`;
- invariante: `unknown` nunca significa `available`, y ningún hecho posterior
  al corte temporal validado entra en el catálogo;
- autoridad: ninguna escritura; `application` valida y proyecta una consulta,
  mientras `Orchestrator` conserva lifecycle y routing;
- dependencia causal: contrato neutral, catálogo y routing V25 existentes;
- retirada legacy: ninguna hasta completar composición, smokes reales y el
  gate íntegro sobre un mismo candidato;
- presupuesto: hasta 450 LOC de producción, 500 LOC de tests, 120 minutos y
  menos de 200 MiB temporales.

## Caracterización y causalidad temporal

El preflight AGT-12 ejecutado antes de editar devolvió `characterize` para la
lección histórica de cuota viva. Se conserva frescura y estados explícitos; se
rechaza scraping de stdout/stderr como autoridad y no se copia ni ejecuta
código legacy.

El Orchestrator toma un corte inicial antes del primer probe y otro corte
inmediatamente después de cada llamada. Cada observación exitosa se valida
contra su corte posterior, no contra el instante previo a la llamada. Un
`ObservedAt` posterior se conserva como fallo aislado; un reloj cero o
regresivo falla cerrado. `ProviderCatalog.observedAt` queda en el último corte,
posterior a todos los probes, incluso si una fuente falló.

`DiscoverProviderAvailability` obtiene además un corte después de completar el
catálogo y exige que no preceda al último corte del catálogo. La clasificación
ordenada mantiene la precedencia `failed`, `stale`, `unavailable`, `unknown`,
`exhausted` y solo entonces `available`. Las observaciones se clonan en
profundidad y cuota/uso/capabilities solo reflejan hechos recibidos.

## Límites y estado honesto

La consulta no selecciona modelos, decide fallback, reserva slots o
presupuesto, reintenta fuentes, lanza o detiene agentes ni muta Goal, WorkItem,
Execution, repositorio, eventos u outbox. Los tests cubren reloj que avanza
dentro del probe, observación futura, reloj regresivo, corte posterior a todas
las fuentes, seis estados, orden, copia profunda y ausencia de escrituras.

Los gates son unitarios y aceptación offline, `-race` focal, `go vet`, guard
de arquitectura, compilación de todos los paquetes y `git diff --check` cuando
el WIP concurrente lo permita.

No hay wiring de bootstrap, credenciales, lanzamiento/inferencia, E2E real,
smoke de proveedor, receipt ni cambios de roadmap/evidence/capabilities. La
siguiente dependencia causal es componer fuentes concretas y ejecutar sus
smokes reales aislados sobre el mismo candidato sellado; AGT-12 permanece
`declared`.
