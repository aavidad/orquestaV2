# BUG-ORQ-20260711-208AH: HOME lleno al reabrir subagente

Fecha: 2026-07-11. Estado: cerrado localmente.

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
## Cierre local 2026-07-11

El commit `514b86c49` incorpora
`scripts/orquesta_session_disk_preflight.sh`: mide los mounts reales de HOME,
`/`, `/tmp`, workdir y raiz de sesion; deduplica filesystem; usa presupuesto
unico en bytes; rechaza bases/symlinks no declarados; crea caches privadas con
marker; y solo permite cleanup confirmado de rutas listadas en el receipt.
`scripts/lib/isolated_test_env.sh` lo ejecuta antes de crear caches y conserva
recibos durables de preflight y cleanup. El cleanup no borra la raiz ni la
evidencia.

La prueba real `bug-208ah-real-20260711` midio cuatro mounts, incluido HOME con
aproximadamente 15 GiB libres, aplico presupuesto de 1 GiB y elimino solo nueve
caches con marker. Evidencia retenida:
`/tmp/orquesta-sessions-208ah-evidence/bug-208ah-real-20260711/session_disk_receipt.json`
y `session_disk_cleanup_receipt.json`. El test con `df` falso cubre HOME
insuficiente, mounts grandes, unidades, symlinks, dry-run, confirmacion y marker
ausente. `test_orquesta_test_batches.sh`, `bash -n`, `git diff --check` y la
suite Go completa quedan verdes. Los scripts que no consuman el perfil aislado
deben invocar el preflight explicitamente antes de trabajo largo.

Inventario vivo: [`inventario_bugs_estado_vivo.md`](../inventario_bugs_estado_vivo.md).
Inventario histórico: [`inventario_bugs_orquesta_2026-06-30.md`](../inventario_bugs_orquesta_2026-06-30.md).
