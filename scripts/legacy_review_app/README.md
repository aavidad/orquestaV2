# Aplicación local de revisión del inventario

## Propósito y usuarios

Esta herramienta provisional permite que una persona autorizada revise
elementos JSONL del inventario histórico y conserve propuestas trazables. Es
una superficie de apoyo para la guía maestra de inventario, no una parte del
núcleo de Orquesta.

No decide el catálogo, no acredita capacidades, no modifica el inventario y no
crea tareas. Una propuesta debe revisarse posteriormente mediante las
autoridades canónicas.

La entrada es el inventario **semántico** ya normalizado, no los censos físicos
de millones de funciones, árboles o superficies. Debe ser un fichero regular y
privado, con identificador estable en cada elemento. La aplicación impone
límites de 32 MiB, 4 MiB por elemento y 10.000 elementos para conservar un
diagnóstico y un consumo de memoria acotados.

## Arquitectura y módulos

- `main.go` valida la configuración, limita la escucha a una dirección
  numérica de loopback y coordina la parada.
- `model.go` lee el inventario y crea revisiones inmutables por contenido.
- `store.go` es el único escritor: conserva un historial JSONL de propuestas.
- `http.go` aplica rutas fijas, autorización HTTP, origen y protección contra
  petición cruzada.
- `render.go` y `templates.go` presentan lista, filtros, detalle y formulario.
- `catalog.go` contiene todos los textos visibles, con español predeterminado y
  de reserva.

No hay base de datos, planificador, proceso en segundo plano, ciclo de vida de
dominio ni conexión con el entorno de ejecución antiguo.

## Datos, permisos, secretos y efectos

El inventario se abre en solo lectura y se comprueba de nuevo antes de cada
propuesta. El historial de propuestas se reemplaza atómicamente con modo
`0600`, revisión esperada e idempotencia. Un fichero lateral `.lock`, también
privado, serializa escritores concurrentes.

La autorización se recibe solo mediante `--authorization-file`. El fichero
debe ser regular, no simbólico, privado y contener entre 24 y 4096 bytes. La
aplicación no imprime su contenido. El actor y el proyecto son referencias
explícitas de arranque y no pueden sustituirse desde el navegador.

El único efecto es escribir el JSONL de propuestas elegido por el operador. No
hay llamadas externas, publicación, Git, despliegue ni creación de tareas.

## Arranque y parada

```bash
go run ./scripts/legacy_review_app \
  --inventory /ruta/privada/inventario.jsonl \
  --proposals /ruta/privada/propuestas.jsonl \
  --authorization-file /ruta/privada/autorizacion \
  --actor-ref actor:operador \
  --project-ref project:orquesta \
  --listen 127.0.0.1:8787
```

Se accede con autorización HTTP básica, usuario `local` y como contraseña el
contenido exacto del fichero privado. La escucha rechaza comodines, nombres de
equipo e interfaces que no sean loopback.

La parada por `Ctrl+C` o `SIGTERM` espera las peticiones en curso. Para
recuperar una interrupción basta con arrancar de nuevo sobre el mismo
inventario y propuestas: el historial se valida antes de escuchar.

## Diagnóstico, contratos y pruebas

Un conflicto indica que otra propuesta avanzó la revisión; se recarga el
detalle y se revisa de nuevo. Un cambio del inventario exige reiniciar. Un
historial malformado impide el arranque sin sobrescribirlo.

Los contratos focales se ejecutan con:

```bash
go test -mod=vendor -count=1 ./scripts/legacy_review_app
go test -mod=vendor -count=1 -race ./scripts/legacy_review_app
GOFLAGS=-mod=vendor go vet ./scripts/legacy_review_app
```

Cubren negativos de autorización, host, origen, petición cruzada y traversal;
idempotencia, revisión esperada y concurrencia; renderizado escapado,
internacionalización, navegación por teclado y señales textuales además del
color; además impiden cargar censos físicos, ficheros públicos o entradas fuera
del presupuesto declarado.
