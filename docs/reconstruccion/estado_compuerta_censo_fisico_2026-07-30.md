# Estado de la compuerta del censo físico histórico

Fecha de observación: 2026-07-30.

Estado de ejecución real: **NO-GO**.

Este registro sucede de forma factual al
`plan_ejecucion_censo_fisico_historico_2026-07-30.md`. No modifica ni reescribe
ese plan sellado: conserva lo que entonces estaba pendiente y registra qué
compuertas cambiaron después. No acredita `GOV-16`, no cierra el inventario
histórico y no autoriza una ejecución.

La base documental queda ligada por bytes:

- plan sellado:
  `sha256:2a863bb5d2f648ea1844a0653484b060e295a8fd4372f3c48ddb46317ff7dc96`;
- prueba contractual del plan:
  `sha256:87d8a010dc82d49ec823100fa91522abda28498681f90ed308fc7260af276de4`.

## Encargo y autoridad

```text
capability IDs: GOV-16
invariante: aceptar el censador no se confunde con acreditar expansión, vista, mapeo, recibos o ejecución
autoridad que escribe: ninguna de producto; este documento es un estado de compuerta
puertos afectados: ninguno
adaptadores afectados: futuro proveedor de vista estable y futura salida POSIX separada
write-set: este documento y su prueba contractual
dependencias causales: plan sellado -> censador aceptado -> expansión -> vista -> mapeo -> recibos -> ejecución
código antiguo que permitirá retirar: ninguno hasta revisar censos acreditados
test de contrato: conserva sujeto, revisiones, cierres y bloqueos sin consultar infraestructura viva
negativo/mutación: quitar un bloqueo, cambiar una huella o afirmar cierre hace fallar la prueba
E2E/gate: no ejecutado; ninguna raíz física histórica fue abierta, enumerada ni censada
presupuesto: documentación de 250 líneas como máximo; prueba de 220 líneas como máximo
```

La consulta local de lecciones `GOV-16` para
`documentar_compuerta_censo_fisico` no devolvió patrones. Se registra como
hueco de conocimiento y no como permiso para rebajar la compuerta.

## Cambio acreditado desde el plan

El censador mínimo dejó de ser una brecha:

- implementación comprometida en
  `02c4ad641d83b656a3cd58fbde090faed66ca3e0`;
- sujeto congelado con SHA-256 agregado
  `33898168cdf41b6ece7702a47d4bd7a073a0652f5b1e0f7a395c91f6d434901f`;
- dominio del agregado: nombres de
  `scripts/legacy_physical_inventory/**` ordenados por bytes, una línea por
  fichero con `sha256sum`, y SHA-256 de la concatenación exacta de esas líneas;
- 41 pruebas superadas tanto en ejecución normal como con detector de carreras,
  además del análisis estático y la comprobación de formato;
- dos revisores independientes aceptaron el mismo sujeto congelado con
  `P0 = 0`, `P1 = 0` y `P2 = 0`.

El commit `d154d3e2b53b1eea786dd043f956af994d23826a` cerró, contra ese
mismo sujeto y esas dos revisiones, los defectos
`BUG-REBUILD-20260730-004`, `BUG-REBUILD-20260730-005`,
`BUG-REBUILD-20260730-008`, `BUG-REBUILD-20260730-009`,
`BUG-REBUILD-20260730-010` y `BUG-REBUILD-20260730-011`.

Estos cierres acreditan el censador confinado: anclaje sin seguir enlaces,
separación física, límites, publicación y recuperación conservadora. No
acreditan la verdad semántica de un censo, la aplicación de expansión, una
vista estable, el mapeo físico ni los recibos externos.

### Trazabilidad de las dos auditorías del censador

Las referencias durables disponibles son el código congelado en `02c4ad64`, su
SHA-256 agregado exacto y las seis filas del registro
`product/traceability/rebuild_bugs.jsonl` comprometidas en `d154d3e2`. Cada
fila enlaza el mismo sujeto y conserva la descripción consolidada
`reviews:two_independent_accepts`.

La primera y la segunda auditoría revisaron ese mismo sujeto y ambas devolvieron
aceptación con `P0 = 0`, `P1 = 0` y `P2 = 0`. No existe en el árbol un
artefacto durable individual por revisor que conserve por separado identidad,
fecha y dictamen íntegro. Esta limitación de procedencia se declara
expresamente: el ledger acredita la conclusión consolidada usada para cerrar
los seis defectos, pero no permite reconstruir dos recibos individuales.

### Expansión lógica acreditada posteriormente

La expansión quedó confirmada en
`adacfef1ecd3cd5c43e5eacbec43eba77f6bcd7c`, sobre el sujeto final
`sha256:593bcb5069241f937c0d10b64b8e183645fd78c66001a72832e1dac01de3336a`.
El artefacto durable
`product/traceability/legacy_physical_subject_universe_2026-07-30.json` tiene
SHA-256 de bytes
`b81478cef265fbb3970925cd09d450b23cd42196858e271c7b26a229fa106f1e`
y sella el conjunto ordenado de 382 sujetos como
`sha256:405f5b68f0e886a40bb69751fbb62f60567a15d6879393e9f42c4d1b5a458cc4`.

Dos auditorías independientes aceptaron el mismo sujeto de expansión con
`P0 = 0`, `P1 = 0` y `P2 = 0`. El commit
`8234e64eb8a48ff6ca9cf86896eabff1fb8fc0e3` conserva en el ledger sus
conclusiones consolidadas y los cierres de `BUG-REBUILD-20260730-012`,
`BUG-REBUILD-20260730-013`, `BUG-REBUILD-20260730-014` y
`BUG-REBUILD-20260730-015`.

### Validador semántico privado aceptado posteriormente

El commit `08063318512682b7c6e5212affc1986afa363da3` incorpora la aplicación
auxiliar `scripts/legacy_physical_mapping`, con sujeto agregado
`sha256:7e85ac617f396023d76c1751f4122151bf45926d059972b4d67fe2bd00bb2a71`.
Dos revisores independientes aceptaron esos mismos bytes con `P0 = 0`,
`P1 = 0` y `P2 = 0`. El registro durable vive en
`docs/reconstruccion/revision_validador_mapeo_historico_2026-07-30.md`, tiene
SHA-256
`128f335bafd0faee4eb6a2a7d962739021b1941c2f741316a33dc951c66f4d39` y quedó
comprometido en `ab318e74946f15001fa1ce429f46dd716b15787b`.

El validador solo comprueba forma, ligadura por bytes e integridad interna de
un candidato privado. No acredita el mapeo físico, la vista estable, la
reobservación, la presencia o ausencia real ni los recibos. No se ha producido
ningún candidato físico real. Por tanto,
`mapeo_1_1_y_1_n_acreditado == true` permanece pendiente.

## Estado de las compuertas

| Compuerta | Estado factual | Condición para abrirla |
|---|---|---|
| Expansión lógica | acreditada | conservar el sujeto sellado y consumirlo sin reinterpretación en el mapeo |
| Mapeo físico | bloqueado | producir y acreditar sobre la vista estable el candidato real 1:1 y 1:N que el validador aceptado solo sabe comprobar |
| Vista estable | bloqueada | mantener fuente de solo lectura y sin fecha de acceso, o exclusión cercada equivalente, desde antes del mapeo |
| Recibos | bloqueados | acreditar sujeto, intento, lote, confirmación, idempotencia y recuperación |
| Salida física | bloqueada | disponer de soporte POSIX separado, privado y con reserva demostrable |
| Ejecución real | bloqueada | abrir todas las compuertas anteriores y asignar revisión independiente |

Las claves exactas de la sección 10 del plan continúan siendo la compuerta. Su
estado en este corte es:

| Condición exacta del plan | Estado del corte |
|---|---|
| `censador_minimo_acreditado == true` | satisfecha |
| `expansion_97_15_285_382_sellada == true` | satisfecha |
| `members_sha256_invalidos == 0` | satisfecha por la expansión |
| `vista_estable_y_cercado_acreditados == true` | pendiente |
| `mapeo_1_1_y_1_n_acreditado == true` | pendiente |
| `reobservacion_382_en_vista_estable_acreditada == true` | pendiente |
| `presencias_actuales_inferidas_desde_v3 == 0` | satisfecha en la expansión; pendiente de reobservación en el mapeo |
| `limites_y_reserva_acreditados == true` | pendiente |
| `recibos_e_idempotencia_acreditados == true` | pendiente |
| `publicacion_y_recuperacion_acreditadas == true` | satisfecha en el censador |
| `salida_fuera_de_fuentes == true` | pendiente |
| `salida_en_dispositivo_distinto_o_vista_externa_cercada == true` | pendiente |
| `recuperacion_conservadora_sin_borrado_automatico == true` | satisfecha en el censador |
| `quinta_auditoria_independiente_superada == true` | satisfecha para el censador |
| `revisor_independiente_asignado == true` | pendiente para la ejecución real |

La expansión acreditada no anticipa presencia actual. El futuro mapeo debe
reobservar los 382 sujetos bajo la misma vista estable. Hasta entonces no
existe lote físico aceptable, no se modifica el manifiesto V3 y
`closed: false` permanece.

Ninguna raíz real fue abierta, enumerada o censada para aceptar el censador ni
para producir este registro. Tampoco se creó una instantánea, una salida de
censo o un recibo de ejecución.

## Observación de infraestructura y motivo del NO-GO

La consulta de montajes del 2026-07-30 registró:

- `/home/alberto/Trabajo` sobre `/dev/sdb1`, `ext4`,
  `rw,nosuid,nodev,relatime`: es una fuente móvil y las lecturas ordinarias
  pueden actualizar fecha de acceso; no es la vista estable requerida;
- `/dev/sda1`, montado como USB, usa `vfat` con `dmask=0022` y `fmask=0022`:
  representa directorios como `0755` y ficheros como `0644`, por lo que no
  puede cumplir el directorio privado `0700` y los artefactos `0600`;
- las demás ubicaciones visibles no fueron acreditadas como salida dedicada,
  separada de todas las fuentes, privada, estable y reservada. Que tengan
  espacio libre o un sistema de ficheros POSIX no permite inferir aptitud.

Es una observación fechada, no una garantía futura. Antes de cualquier
preparación o ejecución se deben reobservar fuente, dispositivo, sistema de
ficheros, opciones de montaje, permisos efectivos, separación física, espacio
y reserva. Cualquier diferencia mantiene el **NO-GO** hasta una evaluación
nueva y acreditada.

La reserva mínima previa es **44 GiB** demostrables. Combina el techo privado
de 32 GiB con el margen libre y el peor temporal activo exigidos por el plan.
No se inicia ninguna fuente con una consulta aislada de espacio ni con un
fichero disperso; hace falta una reserva o cuota con recibo.

## Vía mínima conocida para abrir la infraestructura

La vía más corta y con menos piezas nuevas es:

1. acordar una ventana de mantenimiento y detener de forma acreditada todos los
   escritores de la fuente;
2. desde otro soporte operativo, desmontar la fuente y remontarla
   `ro,noload,noatime`, conservando la misma identidad durante mapeo y censo;
3. aportar un disco físico dedicado, distinto de las fuentes, con sistema de
   ficheros POSIX, capacidad mínima de **64 GiB** y prueba efectiva de
   directorio `0700`, ficheros `0600` y reserva de al menos 44 GiB;
4. sellar vista, identidad, salida y reserva mediante recibos antes de mapear;
5. consumir la expansión acreditada y acreditar mapeo y recibos con casos
   adversariales antes de abrir la primera raíz histórica.

Este documento **no autoriza** detener procesos, desmontar, montar, reformatear,
reparticionar, copiar, borrar ni ejecutar el censador. Esas operaciones
requieren una orden posterior del operador, comprobaciones previas y un
procedimiento reversible con responsables y recibos.

```text
prohibido_sin_orden: detener|desmontar|montar|reformatear|reparticionar|copiar|borrar|ejecutar
```

## Cierre documental

```text
hecho: censador, expansión y validador semántico aceptados; NO-GO de las compuertas restantes documentado
invariante restaurado: aceptar utilidades no equivale a acreditar mapeo, censo ni GOV-16
autoridad final: futuros recibos acreditados sobre la misma vista y sujetos
tests/negativos/mutaciones/E2E: contratos del censador, expansión, validador y estado; E2E real no ejecutado
recibos y revisión acreditada: aceptaciones bootstrap de tres utilidades; ningún recibo productivo de censo
código o decisión retirados: brechas «censador no aceptado», «expansión no acreditada» y «validador semántico ausente»
legacy retirado o bloqueo de retirada: toda retirada sigue bloqueada
LOC netas y complejidad: documentación y una prueba sin acceso a raíces reales
riesgos/P0/P1: P0 operativo si se ejecuta sin vista o salida válidas; contenido por NO-GO
siguiente dependencia causal: decidir la autoridad de vista y producir el mapeo real; preparar infraestructura solo con autorización
```
