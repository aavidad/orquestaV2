# ADR V17: presupuesto de tamaño sin degradar seguridad

Estado: aceptado para el sello V17.

Fecha: 2026-07-22.

## Contexto

El contrato PRE-P fijó `2700` líneas netas para adaptadores y `5200` para
producto antes de existir la implementación real. La medición provisional
omitió `config/` y clasificó de forma distinta algunos `*_test.go`; el cálculo
canónico posterior demostró que esos límites no representaban el alcance.

La parte mínima ya necesaria de Bubblewrap, Git local, SQLite y configuración
canónica suma `3857` líneas netas. Ese subtotal todavía no incluye CAS,
bootstrap, MCP ni Codex. Alcanzar `2700` requeriría retirar garantías, mover
código a una categoría artificial o dividir el mismo contrato entre versiones.

Tras dos refactorizaciones independientes y antes de P, el clasificador exacto
de `acceptance/v17_test_attestor_test.go`, aplicado a un índice temporal con
rutas tracked y untracked, mide:

| Clase | Neto real | Límite V17 |
|---|---:|---:|
| core/ports | 1804 | 2000 |
| adapters | 4442 | 4450 |
| migración | 500 | 500 |
| tests | 7458 | 7500 |
| producto total | 6746 | 6750 |

## Decisión

V17 adopta los límites de la tabla. Son una referencia reproducible del
candidato, no autorización para crecer hasta ellos. Función, fichero y paquete
siguen sujetos a los límites estructurales existentes.

Para V18–V22, el tamaño será un guardarraíl: una desviación se mide, explica y
registra como deuda, pero no bloquea una versión que tenga contratos,
arquitectura, seguridad, recuperación y E2E reales verdes. Después de V22 se
abrirá un frente de adelgazamiento sin mezclarlo con funcionalidad nueva.

Siguen siendo bloqueantes:

- núcleo hexagonal sin dependencias de producto;
- configuración y variables solo en el registro canónico;
- causalidad, idempotencia y receipts ligados a evidencia real;
- límites de recursos y cleanup fail-closed;
- regresiones históricas cuya causa siga siendo compatible;
- batería contractual y E2E real reproducibles.

Una versión antigua incompatible no aporta código, compatibilidad ni pruebas a
la nueva. Su historial solo se usa cuando existe la misma causa arquitectónica
o el mismo contrato.

## Alternativas descartadas

- Reclasificar `config/`, tests o wiring para ocultar líneas.
- Minificar JSON o comprimir código para superar una cifra.
- Mover adaptadores al core o partir V17 sin una frontera de producto real.
- Retirar pinning de ejecutables, AppArmor, cgroups, límites, CAS seguro,
  recuperación SQLite o verificación completa del árbol Git.
- Seguir refactorizando durante el cierre y reabrir fallos ya verdes.

## Consecuencias y seguimiento

El exceso inicial queda visible como deuda de mantenibilidad, no como falso
fallo funcional. El inventario de bugs conserva las incidencias del clasificador
y de las refactorizaciones. El sello P/S/E debe repetir el cálculo canónico,
race, vet y el E2E Git+SQLite+CAS+Bubblewrap; este ADR no acredita V17 por sí
solo.
