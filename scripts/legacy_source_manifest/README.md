# Manifiesto reproducible de fuentes históricas

## Propósito y usuarios

Esta aplicación local caracteriza repositorios, espacios de trabajo y paquetes
Git dentro de instantáneas desechables declaradas. Permite al equipo de
reconstrucción saber qué fuentes existen y qué censos físicos siguen pendientes
antes de analizar funciones o superficies.

## Alcance y exclusiones

Solo recorre las raíces indicadas por el operador. No sigue enlaces simbólicos,
no entra en directorios `.git` durante la enumeración y no lee los ficheros de
los árboles de trabajo. Inspecciona metadatos y referencias Git; los paquetes
se verifican como paquetes sin extraerlos.

No está autorizada sobre una fuente viva. `filepath.WalkDir`, la lectura de
paquetes y los procesos Git pueden actualizar la fecha de acceso aunque no
escriban contenido ni referencias. Hasta disponer de una instantánea acreditada
creada con lectura confinada y sin fecha de acceso, esta aplicación solo se usa
en pruebas o copias desechables. El censador físico es la única enumeración
autorizada de raíces reales.

No decide si una fuente es útil, no deduplica conductas, no crea tareas y no
acredita el cierre del inventario. Una exclusión se registra como hecho y no
equivale a borrar el material.

## Entradas y salidas

Las entradas son una o más raíces de instantánea explícitas, exclusiones exactas
opcionales y un límite temporal por consulta Git. La salida es un único
manifiesto JSON privado con:

- raíces recorridas y errores;
- fuentes incluidas, excluidas o no evaluables;
- referencias, cabecera y estado Git;
- huellas por fuente y una huella del manifiesto;
- recuentos y censos físicos todavía debidos.

Un estado Git limpio solo significa que Git no mostró cambios seguidos o no
seguidos. Los ficheros ignorados quedan fuera de esa observación. Las
referencias de un repositorio desnudo tampoco cubren configuración, hooks o
reflogs. Por ello todo repositorio existente conserva siempre la obligación de
un censo físico.

El formato conserva rutas físicas porque su función es localizar las fuentes.
Por ello la salida cruda permanece fuera del repositorio. El catálogo
normalizado publica después únicamente referencias lógicas y redactadas.

## Arquitectura y módulos

- `main.go` valida la orden y coordina una ejecución finita.
- `scan.go` recorre las raíces sin seguir enlaces.
- `repository.go` y `git_command.go` inspeccionan repositorios y espacios de
  trabajo con tiempo máximo.
- `bundle.go` verifica paquetes Git sin extraerlos.
- `model.go` define hechos, resúmenes y huellas deterministas.
- `output.go` publica el manifiesto privado de forma atómica.

No hay servidor, base de datos, cola, planificador ni proceso residente.

## Autoridad, datos, permisos, secretos y efectos

La aplicación no escribe estado de Orquesta. Su único efecto es leer metadatos
de las raíces autorizadas y sustituir el manifiesto de destino con modo `0600`.
No necesita secretos ni credenciales de red; una consulta Git agotada o un
objeto inaccesible se conserva como error.

La inspección desactiva las descargas diferidas y los bloqueos opcionales de
Git. Un objeto ausente produce un error y nunca autoriza una consulta de red ni
un refresco del índice histórico.

El operador no debe pasar un directorio personal completo, producción, sesiones
ajenas ni rutas con secretos. Las exclusiones son exactas y deben quedar dentro
de una raíz declarada. La salida siempre queda fuera de esas raíces.

## Arranque, diagnóstico, recuperación y parada

```bash
go run ./scripts/legacy_source_manifest \
  --root /ruta/instantanea-desechable \
  --exclude /ruta/instantanea-desechable/subarbol-no-autorizado \
  --manifest /ruta/privada/fuentes.json \
  --git-timeout 30s
```

Los estados y códigos del manifiesto distinguen fuente incluida, exclusión y
error. Una raíz ilegible o consulta agotada no se convierte en una ausencia.
Ante interrupción se conserva cualquier manifiesto anterior y se repite la
orden sobre el mismo conjunto explícito. La parada normal ocurre al publicar el
resultado; no quedan procesos propios.

## Contratos y pruebas

```bash
go test -mod=vendor -count=1 ./scripts/legacy_source_manifest
go test -mod=vendor -count=1 -race ./scripts/legacy_source_manifest
GOFLAGS=-mod=vendor go vet ./scripts/legacy_source_manifest
```

Las pruebas cubren raíces y exclusiones, enlaces simbólicos, espacios de
trabajo limpios y sucios, repositorios desnudos, paquetes, errores, rutas de
salida, entorno Git acotado, determinismo y publicación privada.
