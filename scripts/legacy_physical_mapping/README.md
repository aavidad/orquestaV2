# Validación semántica del candidato privado de mapeo

## Manifiesto de la aplicación

### Propósito y usuarios

Esta aplicación auxiliar comprueba que un candidato privado de mapeo conserva
el V3 y el universo lógico exactos, declara una sola vista y un solo intento y
mantiene coherentes presencia, identidad, propiedad y podas. La usan operadores
y revisores de la reconstrucción antes de estudiar una composición física.

### Alcance y exclusiones

Es un validador local provisional bajo `GOV-16`.
Solo demuestra «candidato bien formado, ligado por bytes y coherente internamente». No abre ni enumera fuentes,
no acredita la estabilidad o vigencia de la vista y el cercado, no observa
presencia o ausencia real, no reabre identidades físicas y no demuestra que se
hayan declarado todos los solapamientos.

No emite un recibo productivo, no acredita `mapeo_1_1_y_1_n_acreditado`, no
cierra la compuerta del censo y no escribe V3, roadmap ni estado de Orquesta.
Los campos de evidencia son referencias y huellas declaradas; validarlas no
prueba autenticidad física. La aplicación no usa Orquesta.

Tampoco sustituye ni mezcla los cuatro artefactos futuros del censo:
`orquesta.physical-census-subject.v1`,
`orquesta.physical-census-attempt-receipt.v1`,
`orquesta.physical-census-batch-receipt.v1` y
`orquesta.physical-census-confirmation.v1`.

La apertura segura es específica de Linux. Rechaza cualquier componente
simbólico mediante `openat2`, además de FIFO, socket, directorio y entrada no
regular. No usa una alternativa menos segura si `openat2` no está disponible.

### Entradas y salidas

La orden recibe exactamente tres ficheros explícitos:

```text
legacy_physical_mapping --v3 V3 --universe UNIVERSO --candidate CANDIDATO
```

V3 y universo deben coincidir con sus bytes sellados. El candidato debe ser
JSON canónico terminado en LF, pertenecer al usuario efectivo y tener modo
`0600`. Los máximos son 1 MiB para cada base y 2 MiB para el candidato; el JSON
admite como máximo 12 niveles, 50.000 elementos léxicos, 1.024 elementos por
matriz y textos de 256 bytes.

Con código cero se escribe en `stdout` un único veredicto JSON canónico. Solo
contiene huellas, conteos y códigos máquina: nunca rutas, descriptores,
dispositivo, inodo, UID, GID, usuario, token de cercado ni secreto. Ayuda y
errores van a `stderr` sin reproducir los nombres de entrada.

### Arquitectura y módulos

- `main.go` interpreta la orden, confina las tres aperturas y reserva los
  canales de salida.
- `json.go` impone JSON estricto, canónico y acotado, y calcula huellas.
- `model.go` contiene los contratos tipados sin datos físicos.
- `base.go` liga V3 y universo y deriva las 382 presencias históricas.
- `candidate.go` valida contexto y observaciones.
- `ownership.go` reconcilia alias, propiedad, solapamientos y podas.

No hay servidor, base de datos, red, entorno, entrada estándar, procesos hijos
ni escritura de ficheros. Esta utilidad no define puerto, adaptador ni
integración productiva. Esa propiedad sigue pendiente del roadmap. Si una tarea
futura autoriza reutilizar la lógica, deberá componerla en el mismo proceso;
queda prohibido encadenar estas órdenes mediante subprocesos.

### Autoridad, datos, permisos, secretos y efectos

**Autoridad:** ninguna sobre Orquesta ni sobre hechos físicos. La autoridad
productiva sigue sin definirse en este contrato auxiliar.

El candidato no admite rutas ni valores libres: referencias de vista, cercado,
política, intento, identidad, vinculación y ausencia son opacas y acotadas. Un
presente exige identidad y evidencia de reapertura declaradas. Un ausente exige
evidencia opaca de `not_found` y prohíbe identidad o vinculación. Un error de
permiso, E/S o cercado no puede representarse como ausencia.

El V3 conserva `exists_at_v3_observation`; el candidato declara por separado
`present_in_stable_view`. Que ambos coincidan no acredita observación. Que
difieran es válido si la declaración mantiene su evidencia y su tipo.
Las referencias y huellas de política y configuración no permiten validar sus
bytes, presupuestos ni límites de ejecución; solo quedan ligadas al candidato.

### Arranque, diagnóstico, recuperación y parada

```bash
go run ./scripts/legacy_physical_mapping \
  --v3 /datos-sellados/v3.json \
  --universe /datos-sellados/universo.json \
  --candidate /datos-privados/candidato.json
```

El ejemplo usa nombres ficticios y no autoriza esas rutas. La aplicación lee
todo y valida antes de emitir. Un fallo anterior a la escritura deja `stdout`
vacío. Un fallo de E/S en `stdout` puede dejar un prefijo, pero devuelve código
distinto de cero y nunca es éxito. No hay recuperación ni limpieza porque no
se crea estado. La parada normal o abrupta no deja procesos propios.

### Contratos y pruebas

El V3 fija 112 referencias en orden: 97 simples y 15 colecciones con 285
miembros, que producen 382 sujetos. Conserva 364 presentes y 18 ausentes solo
en la observación histórica, 36/49/27 decisiones y los 15 sellos de membresía.
El universo fija las identidades tipadas y su orden.

Cada candidato declara una ventana, una vista, una generación de cercado, una
política, una configuración y un intento. El contexto y el cuerpo tienen
huellas con dominio; el cuerpo se resume con `semantic_integrity_sha256` vacío.
Es una comprobación de integridad, no un registro durable de idempotencia.
Repetir los mismos tres bytes produce el mismo veredicto; comparar dos
candidatos o impedir la reutilización de una clave queda fuera del alcance.

El JSON portable es UTF-8 sin BOM, un único objeto compacto sin espacios
insignificantes y con LF final. Conserva el orden declarado de campos y
matrices, no escapa HTML y omite solo los campos `omitempty` vacíos. La huella
se calcula como `dominio UTF-8 || NUL || JSON canónico con la huella vacía`,
incluido el LF. `TestPortableJSONEncodingAndDigestFrame` fija un ejemplo
pequeño; no existe aún un esquema multilenguaje productivo, por lo que cualquier
productor físico sigue bloqueado hasta acreditar equivalencia con ese formato.

Una identidad física repetida comparte vinculación y tiene exactamente un
dueño, sin perder los sujetos lógicos que la originan. Cada solapamiento
declarado tiene un dueño y una poda idéntica; se rechazan dobles dueños,
aristas colgantes y ciclos. Esta reconciliación no descubre solapamientos
omitidos ni decide cuál raíz es físicamente más específica.

```bash
go test -mod=vendor -count=1 ./scripts/legacy_physical_mapping
go test -mod=vendor -count=1 -race ./scripts/legacy_physical_mapping
GOFLAGS=-mod=vendor go vet ./scripts/legacy_physical_mapping
```

APP-13 productivo sigue pendiente: este manifiesto y sus pruebas locales no
acreditan el control universal de aplicaciones.
