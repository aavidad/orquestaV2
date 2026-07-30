# Comprobación previa de los censos Git históricos

Fecha de observación: 2026-07-30.

Estado: comprobación previa terminada; censos completos pendientes. Este
documento conserva medidas para presupuestar el trabajo, pero no cierra el
universo de fuentes, no caracteriza funciones y no acredita capacidades.

## Encargo y límites

```text
capability IDs: GOV-16
invariante: ninguna ausencia se deduce de una copia o rama incompleta
autoridad que escribe: ninguna; este documento es advisory
puertos afectados: ninguno
adaptadores afectados: herramientas locales de inventario Git V4
write-set: este documento
dependencias causales: manifiesto de fuentes y herramientas V4 verificadas
código antiguo que permitirá retirar: ninguno todavía
test de contrato: conectividad, estabilidad de referencias y sellado de salida
negativo/mutación: detectar repositorio parcial, objeto ausente o fuente mutable
E2E/gate: repetir estado Git antes y después del censo
presupuesto: 32 GiB de disco; límites por ejecución descritos abajo
```

La consulta local de lecciones para `GOV-16`, esta ruta y la operación
`documentar_preflight_censo_git` no devolvió patrones. Se conserva como hueco de
conocimiento, no como permiso para omitir pruebas.

No se ejecutó ningún censo grande, no se modificó una fuente histórica y no se
arrancó Orquesta. Las rutas físicas de salida propuestas aún no se han creado.

## Observaciones del corte

| Raíz lógica | Commits observados | Objetos alcanzables | Código Go único | Diferencia frente a V2 | Exclusivos en la unión |
|---|---:|---:|---:|---:|---:|
| Producto V2 | 4.433 | 56.923 | 1.005,9 MB | 0 | 142 commits |
| Legado principal | 4.204 | 56.480 | 1.005,4 MB | 296 commits | 284 commits |
| Copia Berserk sin máquinas virtuales | 511 | 5.753 | 87,1 MB | 6 commits | 3 commits |
| Autonomía limpia | 1.337 | 13.356 | 648,4 MB | 12 commits | 0 commits |
| Copia de consulta V2 | 4.291 | 55.925 | 1.001,6 MB | 0 | 0 commits |
| Orquestador municipal | 60 | 774 | 4,0 MB | 0 | 0 commits |

Los números usan la revisión visible durante la comprobación. V2 seguía recibiendo
cambios locales: pasó de 14 a 20 ficheros no seguidos mientras se observaba.
Sus referencias no cambiaron, pero sus recuentos no deben tratarse como un
sello estable.

La unión mínima que cubre los commits confirmados hasta este corte es:

```text
producto V2 + legado principal + copia Berserk
```

Esa unión contiene 4.732 commits y 60.325 objetos únicos. Es una deduplicación
de contenido Git, no una deduplicación de procedencia. Autonomía, la copia de
consulta y el orquestador municipal siguen necesitando cabecera, referencias y
manifiesto propios. Cada espacio de trabajo necesita además un censo físico,
porque sus cambios sin confirmar no viven necesariamente en los objetos Git.

## Comprobaciones realizadas

- Las seis raíces tienen directorios Git físicos distintos.
- Ninguna es superficial, parcial, `promisor` ni usa almacenes de objetos
  alternativos.
- La comprobación de conectividad terminó sin objetos ausentes en las seis.
- Referencias y cabeceras permanecieron estables entre dos lecturas.
- Los rótulos `v1` a `v22` aparecen en el legado principal, V2 y la copia de
  consulta.
- Todas las cabeceras de los espacios de trabajo son alcanzables.
- No quedó vivo ningún proceso de los censadores.
- El piloto municipal V4 ya produjo dos salidas idénticas; sigue conservado
  fuera del repositorio hasta su estudio.

Git informó de 153 espacios de trabajo asociados al directorio común del
legado y 47 asociados a V2. Los 153 del legado quedan reconciliados así: 31
hechos Git individuales y 122 miembros de colecciones. Estos 122 son 83
espacios de Goal, 21 miembros Git de la reconstrucción, 11 espacios operativos
y los 7 miembros que omitía el primer manifiesto. Los otros dos miembros de la
colección de reconstrucción son artefactos físicos, no espacios Git.

Los 4 espacios de autonomía, 13 de la copia temprana, 2 municipales y 88 de
VEC pertenecen a directorios Git comunes distintos; por eso no forman parte de
los 153 anteriores. El manifiesto estructurado fija esta partición mediante
prueba. Un miembro registrado no se omite por estar limpio ni se borra por
estar marcado como retirable.

## Orden de ejecución autorizado por este preflight

1. Terminar y contrarrevisar el manifiesto de las seis raíces y de todos sus
   espacios de trabajo.
2. Ejecutar superficies y funciones V4 de la copia Berserk.
3. Ejecutar superficies y funciones V4 del legado principal.
4. Esperar a que V2 quede sin escrituras antes de censarlo.
5. Deduplicar el contenido de autonomía, copia de consulta y municipal mediante
   sus manifiestos de referencias, sin perder procedencia.
6. Censar por separado los estados físicos no confirmados.

El destino de cada tanda se creará con `mktemp`, modo `0700`, fuera de todas las
fuentes y con un subdirectorio por raíz. Los artefactos serán privados `0600`.
El patrón previsto es:

```text
/home/alberto/Trabajo/.inventario-orquesta-v4-XXXXXXXX/
```

## Presupuestos conservadores

| Trabajo | Estimación | Límite inicial |
|---|---:|---:|
| Funciones del legado principal | 1,58 GiB; 9–15 min | 4 GiB; 30 min; 1 GiB de memoria |
| Funciones de V2 | 1,58 GiB; 9–15 min | 4 GiB; 30 min; 1 GiB de memoria |
| Funciones de Berserk | 0,14 GiB; cerca de 1 min | 512 MiB; 10 min; 512 MiB de memoria |
| Superficies de cada una | total estimado de 174 MiB | 512 MiB por raíz; 15 min |

Había 956.498.542.592 bytes libres durante la observación. Se reservan
conceptualmente 32 GiB; estas cifras proceden del piloto municipal y son
aproximaciones, no promesas de duración.

## Gates antes y después de cada censo

- herramienta y esquema V4 fijados por revisión;
- referencias y cabecera repetidas y estables;
- conectividad verde y cero objetos ausentes;
- ningún censador anterior vivo;
- destino privado, fuera de fuentes y con espacio presupuestado;
- salida, resumen y manifiesto coherentes;
- mismo estado Git antes y después;
- todos los recuentos explicados, incluidas exclusiones y errores;
- cero procesos residuales propios.

El único bloqueo inmediato para los censos Git es que V2 sigue cambiando. No
impide censar primero las raíces legacy estables. Los censos físicos requieren
además terminar y auditar el censador específico; no se autorizan por este
documento.

## Cierre de esta tarea documental

```text
hecho: comprobación previa reproducible resumida y presupuestos conservados
invariante restaurado: ningún censo grande se inicia a ciegas
autoridad final: manifiestos y artefactos futuros, no este informe
tests/negativos/mutaciones/E2E: conectividad y estabilidad; censo aún pendiente
receipts y revisión acreditada: no aplica; documento advisory
código o decisión retirados: ninguno
legacy retirado o bloqueo de retirada: ningún material se retira antes del censo
LOC netas y complejidad: solo documentación; sin runtime
riesgos/P0/P1: mutabilidad de V2 y estados físicos todavía no censados
siguiente dependencia causal: cerrar manifiesto de fuentes y auditar censador físico
```
