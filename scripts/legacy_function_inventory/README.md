# Censo histórico de funciones

## Propósito y usuarios

Esta aplicación local permite al equipo de reconstrucción enumerar de forma
reproducible las declaraciones Go conservadas en todas las referencias de un
repositorio Git histórico. Produce hechos estructurales para la revisión
semántica posterior; no decide qué debe adoptar Orquesta.

## Alcance y exclusiones

Recorre el grafo de objetos sin cambiar la rama activa ni materializar árboles.
Conserva referencias, confirmaciones, árboles, blobs Go regulares,
declaraciones válidas, variantes y fallos parciales de análisis.

No ejecuta el código encontrado, no interpreta su utilidad, no crea tareas, no
acredita capacidades y no modifica el repositorio. Un resultado completo solo
demuestra cobertura estructural del corte Git observado, no equivalencia con el
producto nuevo.

## Entradas y salidas

La entrada es un repositorio Git explícito. Las salidas obligatorias son:

- un JSONL progresivo con el grafo y las declaraciones;
- un manifiesto JSON con versión, algoritmos, conteos, huella del JSONL y
  huella de las referencias.

Las salidas deben ser distintas y quedar físicamente fuera del repositorio.
Pueden contener nombres históricos, firmas, documentación resumida y la ruta
local del repositorio; por eso se crean privadas y no se incorporan al producto
sin una normalización posterior.

## Arquitectura y módulos

- `main.go` valida la orden y delega la ejecución.
- `git_references.go`, `git_repository.go` y `git_process.go` capturan el corte
  Git sin descarga diferida.
- `object_graph.go` recorre objetos inmutables.
- `go_source.go` analiza blobs regulares y conserva resultados parciales.
- `records.go` define el contrato JSONL y sus dominios de huella.
- `inventory.go` coordina una única ejecución.
- `inventory_stream.go` escribe, sella y publica el par de artefactos.

No existe base de datos, servidor, planificador ni proceso residente.

## Autoridad, datos, permisos, secretos y efectos

La aplicación no tiene autoridad sobre Goals, capacidades ni tareas. Su único
efecto autorizado es crear o sustituir las dos salidas indicadas. No hace
`checkout`, no escribe referencias y ejecuta Git con carga diferida desactivada
para no mutar la fuente por una descarga implícita.

No necesita credenciales. Si el repositorio requiere objetos que no están
presentes, registra o devuelve el fallo en vez de consultar la red. Las salidas
se publican con modo `0600`; el consumidor debe verificar siempre manifiesto,
huella y esquema antes de leer el JSONL.

## Arranque, diagnóstico, recuperación y parada

```bash
go run ./scripts/legacy_function_inventory \
  --repo /ruta/al/repositorio \
  --jsonl /ruta/privada/funciones.jsonl \
  --manifest /ruta/privada/funciones.manifest.json
```

Una referencia que cambie durante el recorrido impide la publicación. Un blob
Go parcialmente válido conserva las declaraciones recuperables y un hecho de
fallo; un objeto ausente nunca se interpreta como ausencia de conducta.

La parada es la terminación de la orden. Ante interrupción o pareja incoherente,
se conservan los artefactos para diagnóstico y se repite el censo desde un
corte estable en un destino controlado. No se borra ningún resultado anterior
hasta estudiarlo.

## Contratos y pruebas

```bash
go test -mod=vendor -count=1 ./scripts/legacy_function_inventory
go test -mod=vendor -count=1 -race ./scripts/legacy_function_inventory
GOFLAGS=-mod=vendor go vet ./scripts/legacy_function_inventory
```

Las pruebas cubren referencias normales, simbólicas, directas, huérfanas y no
UTF-8; objetos ausentes; blobs Go parciales; enlaces y destinos de salida;
publicación recuperable; determinismo; escala y ausencia de descargas
implícitas.
