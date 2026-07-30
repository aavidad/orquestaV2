# Censador físico de fuentes históricas

## Manifiesto de la aplicación

### Propósito y usuarios

**Qué hace:** recorre raíces físicas explícitas y produce un JSONL de hechos del
sistema de ficheros más un manifiesto sellado. Ayuda al equipo de reconstrucción a
localizar copias, ficheros operativos y material no alcanzable desde Git.

**Usuarios:** operadores y revisores técnicos de Orquesta. No es una superficie
para usuarios finales.

### Alcance y exclusiones

Es una herramienta provisional bajo `GOV-16`. No decide qué conducta se
recupera, no crea tareas, no acredita capacidades y no sustituye al inventario
semántico. Solo funciona en Linux con `openat2`, `statx` y `STATX_MNT_ID`.

No sigue enlaces simbólicos, no lee sus destinos y excluye cualquier entrada
cuyo identificador de montaje difiera del de la raíz. No atraviesa rutas
sensibles denegadas. Nunca ejecuta una raíz implícita.

Las rutas absolutas se normalizan solo de forma léxica. El anclaje por
descriptor rechaza cualquier componente simbólico sin resolver ni leer el
enlace.

### Entradas y salidas

Entradas:

- una o más raíces absolutas con alias, modo y subárboles denegados;
- dos nombres nuevos de generación en un mismo directorio absoluto privado;
- presupuestos de entradas, nombres por directorio, profundidad, bytes de ruta,
  bytes JSONL, bytes por fichero, bytes de contenido globales y tiempo.

La envolvente admite como máximo 4096 argumentos y 4 MiB, con hasta 1024
raíces y 2048 exclusiones. El plazo empieza antes de analizar esa entrada.
Los valores predeterminados de cada presupuesto son también máximos canónicos:
las opciones solo permiten reducirlos.

Salidas:

- un JSONL privado que la herramienta nunca sobrescribe por nombre;
- un manifiesto JSON privado, tampoco sobrescribible, que confirma la generación;
- un bloqueo privado y artefactos de publicación que pueden persistir como
  evidencia ante un corte o cualquier fallo previo a la confirmación.

Los nombres no UTF-8 se representan en base64. Dispositivo e inodo no se
publican: `local_identity_sha256` es un digest local no portable.

### Arquitectura y módulos

Una sola orden coordina módulos pequeños de normalización, anclaje de raíces,
recorrido por descriptores, presupuesto, codificación en flujo, anclaje de
salida y publicación. No hay base de datos, servicio residente ni otro ciclo de
vida.

La raíz se ancla con descriptor y cada acceso relativo usa resolución confinada.
El montaje se comprueba antes de clasificar cualquier objeto. Cada directorio
se observa antes y después de enumerarlo; ficheros regulares se contrastan antes
y después del resumen.

Antes del primer efecto, `/proc/self/mountinfo`, `STATX_MNT_ID`, dispositivo y
ruta interna del sistema de ficheros demuestran que raíces y salida no son
idénticas ni se solapan, aunque se presenten mediante montajes vinculados. Las
pruebas sin privilegios usan un lector falso para caracterizar ese contrato; no
afirman haber ejecutado un montaje real.

La evidencia debe vivir en un dispositivo distinto de todas las raíces. Se
rechaza incluso un directorio disjunto del mismo dispositivo: sin esa frontera,
un renombrado concurrente podría introducir la salida en una fuente entre la
comprobación y el primer efecto. Admitir el mismo dispositivo queda diferido a
una composición futura con vista estable o exclusión acreditada.

### Autoridad, datos, permisos, secretos y efectos

**Autoridad:** ninguna sobre Orquesta. Los artefactos son evidencia física para
revisión posterior.

`metadata_only` abre directorios, pero nunca ficheros regulares ni destinos de
enlaces. `source_content` abre únicamente regulares admitidos y calcula SHA-256
por bloques. En un montaje grabable se exige `O_NOATIME` sin alternativa
silenciosa; si no es posible, el censo se rechaza. Un montaje de solo lectura o
`noatime` ya satisface esa garantía.

No lee variables de entorno, credenciales ni entrada estándar. No usa red ni
crea procesos. No modifica las raíces. Sus únicos efectos son ficheros `0600`
en un directorio del operador con propietario correcto y sin permisos de grupo
o terceros.

### Arranque, diagnóstico, recuperación y parada

```bash
go run ./scripts/legacy_physical_inventory \
  --root codigo=source_content:/ruta/absoluta/codigo \
  --root estado=/ruta/absoluta/estado \
  --deny estado=credenciales \
  --jsonl /ruta/privada/censo-generacion-001.jsonl \
  --manifest /ruta/privada/censo-generacion-001.manifest.json
```

Los dos nombres deben estar ausentes: la herramienta jamás sustituye una
generación confirmada. Los ficheros quedan `0600`, por lo que su propietario
todavía puede modificarlos; no se promete inmutabilidad del sistema de
ficheros. Un bloqueo exclusivo impide dos escritores. El JSONL se hace visible
primero, pero ningún lector debe aceptarlo sin el manifiesto; el renombrado
atómico del manifiesto es el único punto de confirmación.

Un diario sincronizado permite diagnosticar una caída en cualquier paso.
La herramienta no borra automáticamente etapas, temporales, diarios ni finales. Si
falta el manifiesto o hay cualquier ambigüedad, conserva todo para estudio y
devuelve `recovery_conflict`; una ejecución posterior debe usar otros nombres
de generación. Si el par quedó confirmado, también conserva el diario como
evidencia. `--help` y los errores visibles proceden del catálogo castellano y
usan códigos máquina sin rutas físicas.

La parada normal cierra descriptores y bloqueo. Una muerte abrupta no deja
procesos. Un corte o cualquier fallo previo a la confirmación puede dejar
etapas, temporales, diarios o finales que el siguiente arranque detecta y
preserva.

### Contratos y pruebas

El manifiesto declara
`sha256_canonical_json_manifest_sha256_empty_v1`: SHA-256 del JSON compacto
producido por el contrato Go, con `manifest_sha256` vacío. Un verificador
contractual exige además esquema, algoritmo y dominio canónicos; después vuelve
a calcular el sello y contrasta nombre, tamaño y SHA-256 del JSONL.
Su promesa se limita a confirmar la publicación atómica y la integridad en
bytes del par. No valida la verdad semántica de `complete`, contadores,
presupuestos, raíces o exclusiones, ni acredita el inventario; esos campos
siguen siendo afirmaciones para revisión posterior.

El límite de entradas se comprueba antes de enumerar o tratar trabajo pesado y
antes de cada apertura o resumen. Con un máximo de directorio `N` se leen como
máximo `N+1` nombres para detectar exceso, se ordenan y solo se procesan los
primeros `N`; el adicional nunca se clasifica. Los bloques de contenido no
superan 128 KiB. El tiempo es cooperativo: una llamada bloqueada en el núcleo no
puede interrumpirse hasta que el sistema operativo la devuelva. Al agotar un
presupuesto se publica una generación parcial con `complete=false`, diagnóstico
estable y salida no mayor que el límite.

```bash
go test -mod=vendor -count=1 -race ./scripts/legacy_physical_inventory
GOFLAGS=-mod=vendor go vet ./scripts/legacy_physical_inventory
for fichero in $(rg --files scripts/legacy_physical_inventory); do
  diagnostico=$(git diff --no-index --check /dev/null "$fichero" 2>&1)
  [ -z "$diagnostico" ] || { printf '%s\n' "$diagnostico"; exit 1; }
done
```

Las pruebas incluyen sustitución de padres y etapas, mutación concurrente,
montaje simulado, `noatime`, enlaces hostiles, FIFO, socket, no-UTF-8, todos los
presupuestos, determinismo, bloqueo y cortes antes y después del punto de
confirmación.

La herramienta se retira cuando el inventario físico quede sellado y la
trazabilidad necesaria pase a la superficie administrativa canónica.
