# orquesta-config v0

`orquesta-config` es una librería pura: no abre ficheros, no consulta entorno y
no publica candidatos. La composición limita I/O y llama a sus primitivas.

`DecodeStrictBoundedJSONV0` rechaza documentos demasiado grandes, campos
desconocidos y valores JSON adicionales sin mutar el destino si falla.
`DecodeAndValidateJSONV0` conserva la misma garantía durante validación
semántica: solo copia el candidato al destino cuando todo el ciclo es válido.
`EncodeCanonicalJSONV0` devuelve JSON
compacto determinista y una revisión `sha256:` calculada sobre esos bytes.
Las composiciones deben leer como máximo `límite+1` bytes antes de llamar al
decoder; así el límite protege también la materialización desde fichero.
El loader productivo de una composición debe construir desde esa única lectura
el candidato `Config`, los bytes canónicos, su revisión y el catálogo que lo
describe; no debe mezclar esos valores entre lecturas o candidatos distintos.

`ReflectJSONLeavesV0` y `BuildCatalogV0` reflejan la misma visibilidad de
campos JSON para exported fields, embedded structs y colisiones. La presencia
es opcional para `omitempty`, punteros y descendientes de padres opcionales.
Cada leaf debe tener una política con presencia, sensibilidad, editabilidad,
motivo y comportamiento de reinicio explícitos; una política editable requiere
un `setter_ref` real y una no editable no puede anunciar uno. Arrays y mapas
son leaves.

`BuildCatalogV0` rechaza tanto hojas sin política como políticas cuya presencia
no coincide exactamente con la hoja reflejada. Las composiciones deben asignar
las hojas a tablas de política explícitas, sin un fallback que catalogue una
hoja nueva. Un `api_key_file` es una referencia a secreto y no debe marcarse
como sensible: el secreto es el contenido externo, que no pertenece al
documento de configuración ni al catálogo.
