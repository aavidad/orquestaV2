# Plan de ejecución del censo físico histórico

Fecha: 2026-07-30.

Estado: plan condicionado. No autoriza todavía a abrir, enumerar ni censar
ninguna raíz física. Antes deben quedar acreditados el contrato mínimo del
censador, la vista estable, los recibos, los límites y su recuperación.

## Encargo y corte de autoridad

```text
capability IDs: GOV-16
invariante: hechos multitemporales, particiones, sujetos, intentos y lotes de observación conservan identidades distintas
autoridad que escribe: futura aplicación local de censo; este documento no escribe estado de producto
puertos afectados: ninguno en esta tarea
adaptadores afectados: futuro censador Linux local y proveedor de vista estable acreditados
write-set: este documento y su prueba contractual aislada
dependencias causales: manifiesto V3, universo expandido sellado, vista estable, censador y recibos acreditados
código antiguo que permitirá retirar: ninguno antes de revisar los censos
test de contrato: cardinalidad, membresía, recibos, repetición, límites y recuperación
negativo/mutación: colección 1:N, alias, solapamiento, cambio, montaje, disco, caída y recibo manipulado
E2E/compuerta: caso adversarial -> vista estable -> mapeo -> censo -> copia -> recuperación, sin raíz histórica
presupuesto: 382 sujetos potenciales, un sujeto activo y límites exactos por intento
```

La consulta local de lecciones `GOV-16` no devolvió patrones para
`planificar_censo_fisico_historico` ni para
`corregir_plan_censo_fisico_dictamen_final`. Es un hueco de conocimiento, no
permiso para reducir pruebas.

Base exacta:

- `product/traceability/legacy_source_roots_2026-07-30.json`;
- commit `723a0ca79fe59204801ad67375127ab8c74bace7`;
- SHA-256 de bytes
  `0feffd7e493bb643bd6dbfa393d8480688f73f7ef452dbc73fbd0e23ede02044`;
- dominio lógico `orquesta.legacy-root-ids.sorted-lf.v1`;
- resumen V3 de los 122 identificadores en ese dominio:
  `sha256:c4a93e4679a045ac68d835accb7bfd0ff0b6cb918bc2a7c9ab1248d3803291ec`.

El dominio `sorted-lf` significa: extraer `roots[].id`, ordenar bytes con
locale `C`, añadir LF después de cada identificador —también del último— y
aplicar SHA-256. El nombre del dominio no forma parte de los bytes históricos;
documenta su serialización para que no se confunda con el dominio NUL.

```bash
jq -r '.roots[].id' \
  product/traceability/legacy_source_roots_2026-07-30.json |
  LC_ALL=C sort |
  sha256sum
```

Durante esta tarea no se abrió, listó ni censó ninguna raíz física. Tampoco se
creó salida privada ni se ejecutó Orquesta.

## 1. Dos universos sellados antes de mapear

### 1.1 Referencias de planificación

El V3 contiene 122 referencias lógicas: 10 no requieren censo y 112 lo
requieren. Las 112 son únicas. Su SHA-256 en orden V3, con un byte NUL real
después de cada identificador, es:

```text
2d2b81b27bfccad5a0b3a6dfabdec299a516dc3b31400f2cb2ecd6948c1e7834
```

```bash
jq -j '.blocking_physical_census_root_ids[] | ., "\u0000"' \
  product/traceability/legacy_source_roots_2026-07-30.json |
  sha256sum
```

Partición administrativa por observación de origen:

| Observación V3 | Referencias | Particiones |
|---|---:|---|
| `base_git_023829` | 95 | `B01`–`B05`, 16 cada una; `B06`, 15 |
| `huecos_worktrees_023829_033143` | 4 | `H01` |
| `externos_metadata_023829_033143` | 11 | `E01` |
| `vec_vivo_035814_035815` | 2 | `V01` |
| **Total** | **112** | **9 particiones** |

Estas particiones solo asignan referencias de forma disjunta:

```text
95 + 4 + 11 + 2 == 112
unión_particiones == referencias_pendientes_v3
intersección_particiones == vacío
sha256_unión_ordenada == 2d2b81b27bfccad5a0b3a6dfabdec299a516dc3b31400f2cb2ecd6948c1e7834
```

No son lotes temporales, unidades de ejecución ni cotas de descriptores,
entradas, miembros, tiempo o disco.

Las decisiones dentro de las 112 son 36 `incluir`, 49 `excluir` y 27
`pendiente`. `excluir` exige una observación mínima con motivo; `pendiente`
prohíbe leer contenido. Una decisión posterior crea un hecho causal nuevo.

### 1.2 Sujetos potenciales y fotografía histórica

Las 112 referencias expanden así en la observación histórica V3:

| Alcance | Referencias | Sujetos potenciales | Presentes en V3 | Ausentes en V3 |
|---|---:|---:|---:|---:|
| `single` | 97 | 97 | 96 | 1 |
| `collection` | 15 | 285 miembros | 268 | 17 |
| **Total expandido** | **112** | **382** | **364** | **18** |

El universo potencial queda fijado en 97 simples y 285 miembros pertenecientes
a 15 colecciones: 382 sujetos en total. La fotografía V3 contiene 96 simples
presentes y 1 ausente, 268 miembros presentes y 17 ausentes; en total, 364
presentes y 18 ausentes.

Esas cardinalidades de presencia caracterizan únicamente la observación
histórica. No son una compuerta de presencia futura, no preasignan ausencias y
no autorizan a fabricar recibos posteriores. La vista estable de la ejecución
debe reobservar los 382 sujetos y conservar dos hechos distintos:

- `exists_at_v3_observation`, copiado sin reinterpretar desde V3;
- `present_in_stable_view`, observado bajo la vista y el cercado vigentes.

Una diferencia entre ambos hechos es información temporal válida. Produce una
observación nueva del mismo sujeto; no cambia por sí sola el universo ni la
generación. Dentro de esta base V3 sellada, solo una modificación acreditada de
la lista de 285 miembros crea otro universo de sujetos y otra generación de
planificación. `members_sha256` se deriva de esa lista: cambiar solo el digest
es corrupción, no una generación.

Cada colección se acepta únicamente si:

- tiene exactamente un registro `collections[]`;
- su `root_id` coincide con la referencia;
- `members_sha256` coincide con el JSON canónico de `members` seguido de LF;
- aliases y miembros son únicos dentro de ella;
- la suma vuelve a dar 15 colecciones, 285 miembros, 268 presentes en V3 y 17
  ausentes en V3.

Primero se sellan el universo de 112 referencias y los 15
`members_sha256`. Después se deriva y sella la lista ordenada de 382 sujetos.
Mapeo, deduplicación y propiedad física consumen esos sellos; no pueden alterar
la membresía.

## 2. Modelo temporal sin identidades solapadas

Los conceptos canónicos son:

- `partition_ref`: reparto inmutable y disjunto de referencias V3;
- `subject_ref`: referencia simple o pareja colección/miembro;
- `unit_ref`: contrato de un sujeto, modo y generación de política;
- `attempt_ref`: una ejecución de la unidad; puede haber varios intentos;
- `observation_batch_ref`: conjunto de intentos aceptados bajo la misma vista
  estable y ventana temporal;
- `receipt_ref`: evidencia externa de sujeto, intento o lote.

Un reintento crea `attempt_ref`, no otra partición. Si cambia la vista estable,
nacen otro intento y otro lote de observación. Si cambia la política, nace otra
generación de unidad. Solo cambiar la lista sellada de miembros crea otro
universo y otra generación de sujetos. Un cambio de presencia no cambia la
identidad ni la generación. Solo un intento puede quedar aceptado por unidad y
generación; los intentos fallidos permanecen causales, no vuelven falsa la
disjunción de particiones.

El lote de observación conserva inicio y fin reales, vista, cercado y lista de
intentos aceptados. Nunca rellena fechas ni `raw_census_sha256: null` de los
siete lotes históricos V3. Los tres lotes V3 sin referencias en esta partición
siguen pendientes.

Una versión sucesora del registro añadirá hechos validados por el esquema
canónico. No se crea otro registro ni se reescriben hechos anteriores.

## 3. Vista estable antes del mapeo

No existe censo completo sobre una fuente móvil. Antes de abrir el primer
descriptor debe existir una de estas garantías:

1. instantánea o montaje inmutable de solo lectura con garantía de no actualizar
   fecha de acceso; o
2. arrendamiento cooperativo de exclusión de escritores, con cercado comprobado en
   cada fase, más una lectura que no actualice fecha de acceso.

`O_NOATIME` es una implementación válida, no la única. También sirve un montaje
acreditado de solo lectura y `noatime`. Si ninguna composición garantiza el
mismo resultado, se aborta sin lectura ordinaria alternativa.

La garantía permanece vigente desde antes del mapeo hasta sellar el censo. Si
ese censo debe alimentar una copia, permanece hasta sellar también la copia.
El cercado se valida antes de cada intento, publicación y copia. Perder el arrendamiento,
montaje o identidad aborta.

La utilidad autónoma no puede estabilizar por sí sola los directorios padre:
comprobar una vez que dos rutas no se solapan no impide mover después la salida
dentro de una fuente del mismo dispositivo. Mientras no exista una vista o un
arrendamiento externo acreditado que mantenga estable esa relación, el
directorio de salida debe estar en un dispositivo físico distinto de todas las
fuentes. Esta separación es una compuerta de seguridad de la utilidad; no
sustituye la vista estable exigida para un censo histórico real.

Cadena causal correcta:

```text
V3 -> vista_estable_y_cercado -> mapeo -> censo -> copia_opcional -> liberar_vista -> censo_Git_sobre_copia
```

Un censo sellado tras el que se libera la vista sigue siendo evidencia física,
pero no puede alimentar una copia futura de esa fuente. Para copiar habrá que
adquirir otra vista y repetir mapeo y censo, o usar la misma instantánea
inmutable todavía acreditada.

## 4. Mapeo simple 1:1 y colección 1:N

El mapeo vive fuera de Git, en salida privada `0700`, fichero `0600`, creación
exclusiva y propietario comprobado.

- Cada referencia `single` se reobserva bajo la vista estable. Si está presente,
  mapea 1:1 a descriptor e identidad comprobada; si está ausente, produce un
  recibo de ausencia de esa vista, con independencia de su presencia en V3.
- Cada referencia `collection` mapea 1:N: envolvente de colección más una
  reobservación por cada miembro sellado. La presencia actual decide descriptor
  o recibo de ausencia sin modificar la lista de miembros.
- El sello privado conserva base V3, vista/cercado, `members_sha256`, identidad
  de reapertura, alias, límites y ambos hechos de presencia sin confundirlos.

El número de descriptor solo vale en el proceso vivo. La recuperación reabre y
compara identidad; no busca por nombre. Rutas absolutas, perfiles, dispositivo,
inodo, usuario y destino no se versionan.

Solo tras sellar 112 referencias y 382 sujetos se resuelven alias,
anidamientos y límites de recorrido. Una raíz más específica posee su subárbol
y el antecesor lo poda. Deduplicar aquí significa evitar doble atribución de
subárboles sin perder procedencia; no promete leer una sola vez cada enlace físico.
Repetir el hash de un enlace físico es válido y cuenta contra el presupuesto.

El mapeo falla cerrado si:

```text
referencias_selladas != 112
simples != 97
colecciones != 15
miembros != 285
sujetos_potenciales != 382
members_sha256_invalidos != 0
sujetos_repetidos != 0
entradas_con_propiedad_ambigua != 0
podas_no_reconciliadas != 0
sujetos_reobservados_en_vista_estable != 382
presencias_actuales_inferidas_desde_v3 != 0
```

## 5. Contrato mínimo y brechas del censador

`metadata_only` necesita enumeración exacta, ruta relativa privada, tipo, modo,
tamaño de fichero regular, errores y totales. No exige por defecto tiempos,
`nlink`, propietario/grupo, atributos extendidos, ACL o capacidades.

`source_content` añade SHA-256 de ficheros regulares y resumen ordenado del
árbol. Requiere `metadata_only` completo sobre la misma vista. Un archivo se
resume como bytes; no se expande sin contrato aparte.

Propietario/grupo, enlaces físicos, atributos y tiempos son perfiles
condicionales. Solo se hacen obligatorios si una copia posterior promete
preservarlos. Si no se activan, el recibo debe declarar explícitamente
`not_observed`; nunca se infieren.

El censador existente no se declara compatible ni aceptado. La cuarta auditoría
independiente lo rechazó; solo una quinta auditoría independiente, ejecutada
sobre el candidato corregido y congelado, podrá cambiar ese dictamen. Hasta
entonces todas las brechas siguientes son compuertas:

| Área | Contrato necesario | Brecha observada | Compuerta antes de fuente histórica |
|---|---|---|---|
| Expansión | 97 simples y 15 colecciones/285 miembros sellados | no acreditada | caso 1:1, 1:N, ausencia y `members_sha256` |
| Vista | garantía estable sin actualizar fecha de acceso desde mapeo hasta fin | no ofrece congelación, arrendamiento ni cercado | proveedor externo o composición integrada con pérdida de cercado |
| Separación de efectos | la salida autónoma no puede entrar en una fuente después de comprobarla | acepta ubicaciones disjuntas del mismo dispositivo y no estabiliza sus padres | mientras no exista vista o arrendamiento externo acreditado, salida en dispositivo físico distinto de todas las fuentes |
| Mapeo | descriptor por simple o miembro, reapertura y poda | no acreditado | alias, anidamiento, sustitución y montaje hostil |
| Metadatos mínimos | ruta, tipo, modo, tamaño y totales | faltan otros metadatos, pero no son mínimos | contrato focal del perfil mínimo; perfiles opcionales separados |
| Propiedad ampliada | UID/GID, tiempos, `nlink`, xattrs solo si se prometen | UID/GID sí se emiten; faltan tiempos, `nlink`, xattrs, ACL y capacidades | bloquear únicamente el perfil que solicite un dato todavía no observado |
| Contenido | resúmenes criptográficos y árbol en la misma vista | los enlaces físicos se resumen repetidamente | repetición permitida y presupuestada; cero promesa de eliminar lecturas duplicadas |
| Recibos | sujeto, intento, idempotencia y lote externos | no acreditados | repetición, recibo manipulado y doble intento aceptado |
| Límites/disco | todos los máximos aplicados y observados | mecanismo no acreditado | agotamiento individual y recibo de reserva real |
| Publicación | confirmación válida distingue completo de etapa | usa manifiesto de confirmación y más de un renombrado | caída alrededor de cada paso; no exigir renombrado único |
| Recuperación | repetición devuelve confirmado o incompleto seguro sin destruir material ambiguo | borra etapas por nombres predecibles y separa comprobación de identidad y borrado | recuperación conservadora: preservar artefactos ambiguos, fallar cerrado y operar sin borrado automático |

Una etapa sin manifiesto de confirmación válido nunca es resultado completo.
Una etapa o huérfano de propiedad no demostrada se conserva y devuelve
`recovery_conflict`; no se elimina automáticamente. Una retirada posterior
requiere recibo externo previo, identidad estable y una operación distinta,
auditable e idempotente. El bloqueo cooperativo no sustituye ese contrato.

## 6. Presupuestos por sujeto expandido

Las nueve particiones no presupuestan recursos. Una colección de una referencia
puede expandir decenas de miembros. Se ejecuta un único sujeto cada vez y cada
intento recibe estos máximos explícitos:

| Campo | `metadata_only` | `source_content` |
|---|---:|---:|
| `max_entries` | 2.000.000 | exacto del metadato aceptado |
| `max_entries_per_directory` | 100.000 | exacto del metadato aceptado |
| `max_depth` | 128 | 128 |
| `max_relative_path_bytes` | 4.096 | 4.096 |
| `max_regular_file_bytes` | no aplica a lectura; tamaño solo observado | 2 GiB |
| `max_hashed_bytes` | 0 | 64 GiB |
| `max_output_bytes` | 256 MiB | 1 GiB |
| `max_wall_time` | 30 minutos | 2 horas |
| `max_memory_bytes` | 1 GiB | 1 GiB |
| `max_open_fds` | 160 | 160 |
| `max_scratch_bytes` | 512 MiB | 2 GiB |
| `max_total_attempt_bytes` | 1 GiB | 4 GiB |

La envolvente de colección admite como máximo 1.024 miembros y 382 sujetos
totales para este V3. Los miembros se procesan secuencialmente; nunca se abren
285 descriptores a la vez.

El límite efectivo de descriptores es
`min(160, max_depth_solicitada + 32)`. Con profundidad 128 resulta 160:
un descriptor por nivel y 32 para raíz, salida, diario, sellos y margen
comprobado. El censador aplica además el límite del proceso y publica el máximo
observado. Una profundidad menor reduce la concesión; las colecciones no la
multiplican. Si el algoritmo necesita más, el intento no empieza hasta revisar
profundidad, fórmula y negativos.

Existe un techo global de 32 GiB de salida privada y deben quedar libres al
menos 8 GiB y dos veces el temporal activo después de reservar. El límite
`max_scratch_bytes` cubre solo espacio temporal: 512 MiB en `metadata_only` y
2 GiB en `source_content`. `max_total_attempt_bytes` cubre conjuntamente
salida, temporal, manifiesto, diario y confirmación: 1 GiB y 4 GiB,
respectivamente. Ambos límites se aplican; uno no sustituye ni duplica al otro.

La reserva solo cuenta si un mecanismo acreditado de cuota o preasignación real
emite recibo; un fichero disperso o `df` aislado no acreditan espacio.

El peor caso teórico por unidad no demuestra que 32 GiB basten para los 382
sujetos potenciales si todos están presentes en la vista futura. La preparación
calcula salida acumulada; si no cabe, no se inicia ninguna fuente hasta aprobar
otra reserva. Agotar cualquier máximo produce intento incompleto; ampliarlo
exige nueva política e intento.

## 7. Recibos sin autorreferencia

Versiones y dominios iniciales:

```text
orquesta.physical-census-subject.v1
orquesta.physical-census-attempt-receipt.v1
orquesta.physical-census-batch-receipt.v1
orquesta.physical-census-confirmation.v1
```

El manifiesto de sujeto contiene base V3, sellos de referencias y miembros,
`subject_ref`, unidad, modo, límites, vista/cercado y resúmenes de herramienta,
política y configuración. Todos los digests de esta sección usan este encuadre
exacto:

```text
bytes_a_resumir =
  dominio_utf8 || 0x00 || json_canonico_sin_digest || 0x0a
digest = "sha256:" || hex_minusculas(SHA-256(bytes_a_resumir))
```

`json_canonico_sin_digest` es un único valor JSON UTF-8 sin BOM ni espacios
insignificantes, con claves únicas de objeto ordenadas por el valor UTF-8 sin
escapar, orden de arrays conservado, enteros decimales sin signo positivo ni
ceros iniciales y sin números de coma flotante. Los textos conservan sus puntos
de código, sin normalización Unicode. Solo se escapan comillas, barra inversa y
U+0000–U+001F; se usan `\b`, `\t`, `\n`, `\f`, `\r` cuando correspondan y
`\u00xx` minúsculo para los demás controles. No se escapan `/`, caracteres no
ASCII, `<`, `>` ni `&`. El campo `digest` no está presente en esos bytes: no se
representa como cadena vacía ni `null`. El byte LF final sí forma parte del
dominio.

Cada objeto usa como `dominio_utf8` una de las cuatro versiones declaradas
arriba. Así, el manifiesto de sujeto usa
`orquesta.physical-census-subject.v1`; intento, lote y confirmación usan su
dominio homónimo. El encuadre no depende de la serialización indentada que
pueda almacenarse para lectura humana.

El recibo de intento se almacena fuera del sujeto y referencia su digest,
resultado bruto, tiempos, consumo, error y clave de idempotencia. El bruto no
incluye su propio recibo ni la confirmación. La confirmación se crea después y
enlaza sujeto, bruto y recibo. Un índice posterior puede enlazar los tres; no
se exige que un árbol contenga el digest del mismo árbol que lo contiene.

El recibo de lote enlaza vista estable, ventana y exactamente un intento
aceptado por unidad/generación. Códigos máquina son estables; mensajes humanos
usan catálogo.

La aceptación observable, compatible con una publicación de varios pasos, es:

1. manifiesto de confirmación presente y válido;
2. sujeto, bruto y recibo coinciden por digest;
3. una repetición con la misma idempotencia devuelve el mismo confirmado;
4. sin confirmación se devuelve `incompleto`, nunca éxito;
5. recuperación y limpieza siguen la política acreditada.

## 8. Orden operativo y aborto

1. Verificar V3, dominios, cardinalidades y particiones sin fuentes.
2. Ejercitar censador, recibos, límites, publicación y recuperación solo con
   casos adversariales; congelar el candidato y superar la quinta auditoría
   independiente.
3. Obtener y sellar vista estable o arrendamiento/cercado.
4. Sellar 112 referencias, 15 membresías y 382 sujetos; reobservar los 382
   sujetos sin inferir su presencia actual desde V3.
5. Mapear simples 1:1 y colecciones 1:N; resolver podas y sellar por separado
   `exists_at_v3_observation` y `present_in_stable_view`.
6. Ejecutar `metadata_only` por sujeto e intento.
7. Ejecutar `source_content` solo para `incluir`.
8. Si se necesita instantánea, copiar y sellar sin liberar la vista.
9. Liberar vista; ejecutar Git solo sobre la copia.
10. Contrarrevisar recibos y publicar únicamente el resumen redactado.

Abortan la generación activa: deriva de base o membresía, cercado caducado,
vista mutable, identidad sustituida, propiedad ambigua, salida solapada,
garantía no-atime perdida, presupuesto agotado, recibo incoherente, fallo de
confirmación o pérdida de autoridad.

Un intento fallido no altera partición ni unidad. La recuperación acepta un
resultado solo mediante confirmación válida; de lo contrario reinicia con otro
`attempt_ref` o deja bloqueo. No se modifica V3, no se borra una fuente y no se
atribuye cierre.

## 9. Alimentación posterior de la instantánea Git

El censo físico no es una instantánea ni acredita `GOV-16`. Para alimentar una
copia, la vista y el cercado deben seguir vigentes desde antes del mapeo hasta después
del sello de la copia. El copiador consume sujeto, mapeo, censo y presupuesto
exactos; no redescubre rutas ni ejecuta Git sobre la fuente.

La cadena acreditable es:

```text
V3 -> vista_estable_y_cercado -> mapeo -> censo -> copia -> liberar_vista -> censo_Git_sobre_copia
```

Si la vista se liberó tras el censo, se repite desde una vista nueva. Cada
flecha tiene recibo externo. La salida privada no entra en Git.

`closed: false` permanece mientras existan lotes históricos sin recibo,
decisiones pendientes, sujetos incompletos o censo Git sin copia. Una unión de
lotes nunca se presenta como instante global.

## 10. Compuerta previa a una ejecución real

```text
censador_minimo_acreditado == true
expansion_97_15_285_382_sellada == true
members_sha256_invalidos == 0
vista_estable_y_cercado_acreditados == true
mapeo_1_1_y_1_n_acreditado == true
reobservacion_382_en_vista_estable_acreditada == true
presencias_actuales_inferidas_desde_v3 == 0
limites_y_reserva_acreditados == true
recibos_e_idempotencia_acreditados == true
publicacion_y_recuperacion_acreditadas == true
salida_fuera_de_fuentes == true
salida_en_dispositivo_distinto_o_vista_externa_cercada == true
recuperacion_conservadora_sin_borrado_automatico == true
quinta_auditoria_independiente_superada == true
revisor_independiente_asignado == true
```

## Cierre de esta tarea documental

```text
hecho: universos, presencia temporal, vista, límites, recibos y brechas corregidos
invariante restaurado: referencias, sujetos, intentos y lotes no se confunden
autoridad final: futuros censos y recibos acreditados, no este documento
tests/negativos/mutaciones/E2E: definidos; ninguna raíz histórica ejercida
recibos y revisión acreditada: ninguno; quedan como compuertas
código o decisión retirados: presencia V3 como compuerta futura, borrado automático, límites ambiguos y promesas falsas de 1:1 o renombrado único
legacy retirado o bloqueo de retirada: toda retirada sigue bloqueada
LOC netas y complejidad: solo documentación y contrato focal
riesgos/P0/P1: salida móvil y recuperación destructiva bloquean ejecución real
siguiente dependencia causal: corregir el censador, congelarlo y someterlo a quinta auditoría independiente
```
