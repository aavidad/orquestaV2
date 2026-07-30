# Censo histórico de superficies

Este programa enumera hechos estructurales de todas las confirmaciones
alcanzables desde las referencias de un repositorio Git. No hace checkout, no
materializa árboles y no modifica el repositorio observado.

Su alcance es deliberadamente distinto del censo de funciones Go:

- conserva referencias directas a confirmaciones, árboles y blobs como un
  grafo navegable, incluso cuando no existe una confirmación intermedia;
- recorre la identidad y las rutas de árboles `vendor` o `node_modules`, pero
  no lee sus blobs salvo que el mismo objeto sea alcanzable también por una
  ruta no excluida; cada frontera conserva la obligación explícita de revisar
  parches, conexiones y licencias propias antes de descartar el material;
- identifica enlaces simbólicos y submódulos tal como están representados en
  Git;
- calcula tamaño y SHA-256 del contenido sin cargar blobs completos en memoria;
- distingue texto UTF-8 de contenido binario;
- agrega una sola vez por blob todas sus familias, tipos y modos históricos,
  usando un máximo verificable de 4096 estados de clasificación finitos por
  árbol en vez de retener cada prefijo completo; no infiere conducta, utilidad,
  calidad ni equivalencia con Orquesta V2;
- registra como hechos separados los objetos que no pueda leer.

Las familias iniciales son:

- `ordenes`;
- `http_mcp_web`;
- `configuracion`;
- `sql_migraciones`;
- `shell_python`;
- `servicios_despliegue`;
- `pruebas_datos`;
- `documentos_decisiones_incidencias_evidencias`;
- `codigo_fuente`;
- `desconocido`.

La clasificación es una ayuda para repartir la revisión humana posterior. No
constituye una decisión de adopción ni una tarea de producto.

## Ejecución

```bash
go run ./scripts/legacy_surface_inventory \
  --repo /ruta/al/repositorio \
  --jsonl /ruta/privada/superficies.jsonl \
  --manifest /ruta/privada/superficies.manifest.json
```

Las dos salidas son obligatorias, físicamente distintas y exteriores al
repositorio observado. Se rechazan enlaces simbólicos en cualquiera de sus
componentes. Se escriben con modo `0600` mediante temporales en el mismo
directorio, se sincronizan los directorios y se publican como una pareja
recuperable. El
censo vuelve a leer las referencias antes de publicar; si cambiaron durante el
recorrido, no publica resultados. Si falla el segundo renombrado, restaura la
pareja anterior o elimina el manifiesto antes de devolver el error.

El sistema de ficheros no ofrece un renombrado indivisible de dos destinos. Una
caída abrupta después de publicar el manifiesto y antes del JSONL puede dejar
temporalmente un manifiesto nuevo junto a un JSONL anterior. Esa pareja es
detectablemente inválida porque `inventory_sha256` no coincide. Todo consumidor
debe verificar la huella antes de usarla y reintentar ante discrepancia. Repetir
el censo sobre las mismas referencias publica de nuevo ambos ficheros y recupera
la pareja. No se añade un tercer marcador que compita con el manifiesto.

## Registros

El JSONL contiene siete tipos:

- `cabecera_inventario`: versión y algoritmo incompatibles con formatos
  anteriores;
- `referencia`: nombre, objeto apuntado y destino pelado con su tipo;
- `confirmacion`: árbol y padres, que permiten reconstruir la procedencia;
- `objeto_arbol`: identidad de cada árbol único alcanzable;
- `entrada_arbol`: segmento directo, modo, tipo y objeto hijo; puede contener
  la causa de una frontera vendorizada y su obligación de revisión;
- `hechos_blob`: huella, tamaño, codificación y unión ordenada de familias,
  tipos, modos y resúmenes históricos de un blob único;
- `fallo`: código estable y detalle acotado del objeto no legible.

La ruta completa no se repite por confirmación: se reconstruye desde el árbol
raíz concatenando los segmentos de `entrada_arbol`. Un segmento Git no UTF-8
no se transforma de forma silenciosa: el texto queda vacío y los bytes
originales se conservan en `path_segment_base64`.

El manifiesto fija la versión del esquema, los algoritmos de grafo,
clasificación y huella, el formato de objetos Git, los conteos, el tamaño del
JSONL y la huella de las referencias. No incluye fecha ni datos variables, por
lo que dos ejecuciones sobre el mismo estado producen exactamente los mismos
bytes. Los conteos incluyen los contextos realmente visitados y el máximo de
estados posibles por árbol.
