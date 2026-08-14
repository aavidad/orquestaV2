# ADR V38: regularización del journal durable de solicitudes de provider

Fecha: 2026-08-14.

## Contexto

El corte `0a562d7` añadió el journal neutral que conserva, en el mismo
`StateRepository` SQLite, los bytes exactos preparados para un provider y el
binding físico que este devuelve. La estimación prospectiva declaró
`P<=120,V<=180`, pero la medición independiente del commit fue muy superior:

| Clase | Añadidas | Retiradas | Neto | Exceso sobre el límite |
|---|---:|---:|---:|---:|
| producto (`P`) | 687 | 2 | 685 | 565 |
| pruebas y acceptance (`V`) | 457 | 6 | 451 | 271 |

Vendor y documentación quedan fuera de esta clasificación. Reducir el corte
hasta la estimación habría exigido retirar causalidad durable, defensa frente
a SQL directo o recuperación tras reopen. Eso degradaría el invariante que el
corte debe preservar y no constituye una compensación válida.

La superficie estructural del corte es acotada: un puerto neutral, una tabla
en el repositorio existente, tres operaciones del journal, tres triggers y un
validador de recovery. No añade otro writer, lifecycle, scheduler, store,
goroutine ni política de provider.

## Retirada compensatoria

La revisión posterior identificó duplicación que sí podía retirarse sin
rebajar el contrato:

1. `RecordAgentProviderRequest` repetía en un `INSERT ... SELECT` los gates de
   causalidad que el trigger de la migración 037 ya aplica a toda escritura,
   incluida SQL directa. El repository conserva la validación de entrada, el
   write serializado y la relectura exacta, y delega ese gate al trigger.
2. `BindAgentProviderLaunch` repetía en su `UPDATE` el gate receipt/state del
   trigger. Conserva el predicado compare-and-set de binding nulo, la relectura
   y la comparación exacta; el trigger sigue protegiendo causalidad.
3. `ResolveAgentProviderRequest` abría una transacción de lectura sin agrupar
   más de una consulta. La lectura usa ahora la conexión del repository sin
   cambiar su semántica.
4. Los tests repetían preparación, binding y aserciones de conflicto/recovery.
   Los helpers retiraron ese ruido sin fusionar los casos de replay, reopen,
   orden, ejecución/cerca cruzadas, carrera de binding, receipt tardío,
   corrupción o provider cruzado.

La compensación respecto de `0a562d7` es:

| Clase | Añadidas | Retiradas | Neto retirado |
|---|---:|---:|---:|
| producto (`P`) | 14 | 65 | 51 |
| pruebas (`V`) | 137 | 172 | 35 |

## Decisión

1. Para este corte ya implementado, el techo histórico se sustituye por el
   resultado exacto después de la compensación: `P=634,V=416`, sin margen.
2. La deuda visible frente al presupuesto original queda en `P=514,V=236`.
   No se diluye con vendor, documentación ni trabajo del adaptador Docker.
3. La cifra regulariza un hecho pasado y no crea una bolsa para el adaptador,
   configuración, bootstrap ni tareas posteriores. Cada uno requiere
   write-set y presupuesto propios.
4. La migración 037, sus triggers, el compare-and-set, la relectura posterior
   a cada escritura y el validador de recovery se conservan. Protegen caminos
   distintos y no son duplicación compensable.
5. `MicroVMHostLaunchAuthority` tampoco se retira: expresa concesión y
   servicios estructurados; no es equivalente al journal opaco de bytes.
6. El corte permanece `implemented_not_runtime_wired`. No acredita Docker,
   Firecracker, `ORC-28` ni V38, y no cambia la obligación Firecracker A+B+C.

## Gates de la regularización

La revisión debe cubrir de forma explícita:

- escritura SQL directa y gates de la migración 037;
- rechazo de binding posterior a receipt y compare-and-set concurrente bajo
  `-race`;
- replay, reopen y recovery de corrupción/causalidad;
- acceptance del contrato opt-in, suite focal de puertos y suite SQLite;
- `git diff --check`, clasificación `numstat` y ausencia de cambios en vendor.

Los resultados concretos se conservan en el commit y su revisión local; este
ADR no los sustituye ni promueve estado por sí solo.

## Siguiente dependencia causal

Solo después de que esta regularización quede revisada y verde puede retomarse
el adaptador Docker neutral ya iniciado. Ese adaptador deberá usar el módulo
publicado y vendorizado detrás de `AgentLauncher`, sin socket Docker directo,
DB/filesystem/secretos compartidos, fallback silencioso ni atribución de la
compuerta Firecracker.
