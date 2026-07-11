# BUG-ORQ-20260711-208AH: HOME lleno al reabrir subagente

Fecha: 2026-07-11. Estado: mitigado localmente; prevención pendiente.

## Evidencia

Al reabrir un subagente falló `No space left on device`. La medición mostró
`/home` al 100% con 0 disponible, mientras `/` conservaba 134G y `/tmp` 44G.
Tras verificar que no había procesos Go vivos, `go clean -cache` liberó 17G;
`/home` quedó con 17G libres y al 99%. No se atribuye todo el uso a Orquesta:
solo se limpió la cache Go identificada y no se borraron otras caches ni
runtimes retenidos.

## Relación y prevención

Reproduce la clase de higiene de disco de
[`BUG-ORQ-20260704-172`](../inventario_bugs_orquesta_2026-06-30.md), pero añade
que `/` y `/tmp` pueden aparentar capacidad suficiente aunque el mount real de
`$HOME` esté agotado. Las herramientas y sesiones largas deben medir el
filesystem real de `$HOME`, además de `/` y `/tmp`, usar caches aisladas y
dejar cleanup gobernado con evidencia. No debe hacerse limpieza indiscriminada:
hay que identificar propietario, procesos, referencias y retención antes de
borrar.

## Criterio de cierre

Preflight documentado con `df` por mount (`$HOME`, `/`, `/tmp` y mounts de
trabajo), presupuesto de cache por sesión y recibo de creación/uso/limpieza.
El estado actual queda mitigado, pero no cerrado, hasta demostrar ese
preflight y `cache budget/receipt` en una reapertura equivalente.

Inventario vivo: [`inventario_bugs_estado_vivo.md`](../inventario_bugs_estado_vivo.md).
Inventario histórico: [`inventario_bugs_orquesta_2026-06-30.md`](../inventario_bugs_orquesta_2026-06-30.md).
