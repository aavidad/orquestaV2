# V23: gate candidato rechazado

Fecha: 2026-08-11. Estado: **rework_required**. No existe candidato V23 PASS,
receipt de sello ni promoción de roadmap.

## Sujeto congelado

- rama: `integracion/v23-intake-durable`;
- commit publicado local/origin: `4811ab38a469d7940e5e0c19ca6390deb1e90c6b`;
- tree OID: `f54ffbecac7ee8162ea5d546801df66d6b6892bd`;
- SHA-256 de `git archive --format=tar HEAD`:
  `a8c0254e319c8ebd4f25c62aeaa1e689475b535dac2c9ddef5458a0f5dc77736`;
- fixture V23 SHA-256:
  `d4231f371464b871f2bde535710424ac842541a91814c6e78ace64a91c34529f`.

Esta evidencia queda fuera del sujeto. El registro estructurado está en
`product/evidence/candidates/v23_wizard_candidate.json` y declara
`candidate=false`.

## Gates ejecutados una vez

1. `go test -mod=vendor -count=1 ./acceptance -run '^Test.*V23.*$'`
   pasó: paquete 0.628 s, pared 1.06 s, RSS máximo 291004 KiB.
2. `go test -mod=vendor -race -count=1 ./internal/application ./internal/adapters/state/sqlite`
   falló: aplicación pasó en 5.016 s; SQLite agotó el timeout Go de 10 min
   mientras ejecutaba `TestClaimLeaseUsesSelectedActionKind/normal_action` en
   `applyMigrationSteps`. SQLite terminó a 600.023 s; proceso 608.19 s, exit 1,
   RSS máximo 608644 KiB. Diagnóstico exacto: `panic: test timed out after
   10m0s`.
3. `go test -mod=vendor -count=1 . -run '^TestRebuildArchitecture$'`
   pasó: paquete 1.038 s, pared 1.23 s, RSS máximo 130968 KiB.

No se reintentó el race. Dos verdes parciales no convierten el gate en PASS.

## Incidencia causal

Capability afectada: `OPS-02`, con frontera observada en el adaptador SQLite.
El invariante restaurable es que el comando obligatorio del candidato termine
completo bajo su presupuesto declarado; hoy la suma de migraciones y tests con
race supera el timeout implícito de 10 minutos. No hay evidencia de fallo de
aserción ni licencia para llamarlo flaky.

Resolver exige un write-set separado y una decisión explícita entre:

- reducir de forma acreditada el coste agregado de bootstrap/migraciones bajo
  race; o
- declarar en la autoridad de la microtarea un timeout mayor, con presupuesto
  y justificación reproducibles.

Después debe repetirse V23-10 completo sobre un nuevo sujeto. V23-11 y V23-12
no pueden comenzar; V23-13 no puede sellar.

```text
hecho: sujeto congelado y tres gates ejecutados; race rechazó el candidato
invariante restaurado: ningún fallo se presentó como PASS ni como sello
autoridad final: product/evidence/candidates/v23_wizard_candidate.json, status rework_required
tests/negativos/mutaciones/E2E: acceptance PASS; race FAIL por timeout; arquitectura PASS
receipts y revisión acreditada: ninguno; reviews no admitidas
código o decisión retirados: ninguna
legacy retirado o bloqueo de retirada: V34 sigue bloqueada; legado intacto
LOC netas y complejidad: solo evidencia; 0 writers, stores o loops nuevos
riesgos/P0/P1: P1 de presupuesto del gate race SQLite
siguiente dependencia causal: decisión operador sobre presupuesto frente a optimización
```
