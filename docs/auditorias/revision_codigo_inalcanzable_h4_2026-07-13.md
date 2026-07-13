# Revisión ejecutable de código inalcanzable H4 — 2026-07-13

Base revisada: `ff125d7cb5` (los commits documentales posteriores no alteran el
código). Fuente original:
`docs/auditorias/clasificacion_codigo_inalcanzable_2026-07-12.md`.

## Evidencia fresca

`deadcode -test ./...` devuelve 81 raíces. Frente a las 99 del inventario
original desaparecieron 19 y apareció una raíz nueva (`sortedZipNamesV0`). Las
39 entradas clasificadas `CONSERVAR` siguen siendo contratos, métodos de puerto
o falsos positivos de dispatch por interfaz y no requieren mutación.

## Cerradas con evidencia

- Retiradas: 5, 6, 8, 18, 34, 73, 78–82 y 98–99.
- Conectadas/reachables: 15, 29, 62 y 67–70.
- Las filas 15, 29 y 67 estaban clasificadas inicialmente para borrar, pero hoy
  tienen caller productivo; su cierre correcto es `CONECTADA`.

## Garantías que todavía deben conectarse

- Core safety: 7, 13–14 y 17.
- Exporter/registries documentales: 21, 27 y 30.
- Observabilidad inbound: 37–38.
- Lease real del outbox: 44–46.
- Tool de presentaciones: 47.
- Lifecycle durable de tools: 64–66.
- Clasificación de errores web: 72, 76 y 77.

La fila 21 se corrige a `CONECTAR`: `CSVDocumentExporterV0.AdapterIdentityV0`
satisface un puerto real y CSV es una capability distinta, no código muerto.

## Solapes que no deben conectarse de forma cosmética

- Retirada privada segura: 1–4, 28 y `sortedZipNamesV0`.
- Requieren retirada o reclasificación con compatibilidad documentada: 16, 35,
  51–61, 63, 71 y 74.
- 52–54 y 59–61 son overloads de compatibilidad; los resolvers reales ya llaman
  a `WithLocaleAndControlFiles`.
- 57 ya está absorbida por el lifecycle interno de ownership de tmux.
- 58 ya está absorbida por el builder privado que alimenta el packet.
- 63, 71 y 74 tienen variantes/caminos productivos más completos; añadir un
  caller artificial empeoraría la composición.

## Decisiones de integración

Se rechazaron y no deben integrarse tal cual:

- `db8762a68b`: outbox lease test-only y replay duplicable.
- `f3cd63b065`: status web sin identidad causal completa ni redacción segura.
- `a3d0f22475`: borraba API del director y dejaba el validador de stats muerto.
- `afa3f2c92d`: borraba filas `CONECTAR` y rompía API core V0.

Cada una tiene rework causal activo o planificado. Un símbolo solo se considera
cerrado cuando la frontera productiva, la causalidad y el test negativo quedan
acreditados; desaparecer de la salida de `deadcode` no basta.
