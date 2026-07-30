# Expansión lógica del universo físico histórico

## Propósito y usuarios

Esta aplicación local convierte las 112 referencias pendientes del manifiesto
V3 exacto en un conjunto sellado de 382 identidades lógicas. La usan operadores
y revisores de la reconstrucción antes del mapeo físico.

## Alcance y exclusiones

Solo lee el fichero V3 y, al verificar, el universo indicados expresamente.
Escribe JSON canónico en `stdout`. No abre, enumera ni infiere el estado de ninguna raíz física. No
observa presencia actual, no emite recibos de censo, no decide utilidad y no
acredita el inventario.

La apertura segura actual es específica de Linux: usa las banderas Unix
`O_NONBLOCK`, `O_CLOEXEC` y `O_NOFOLLOW`. No se afirma portabilidad a otros
sistemas operativos.

El V3 queda fijado por su SHA-256 exacto, clase y versión. Un campo añadido,
omitido o alterado impide producir salida. `path_alias` es un identificador
opaco dentro de una colección: nunca se interpreta como ruta. Dos colecciones
pueden repetir el mismo alias porque la identidad incluye la colección.

## Entradas y salidas

La generación recibe `--v3 FICHERO_V3`. La verificación recibe además
`--verify --universe UNIVERSO`. Ambas entradas deben ser ficheros regulares de
hasta 1 MiB: se abren sin seguir el enlace final, en modo no bloqueante y con
cierre al ejecutar. Un FIFO, socket, directorio o enlace simbólico se rechaza
antes de leer. El V3 debe coincidir byte a byte con la huella fijada. La salida
contiene:

- huella de bytes y contrato del V3;
- 112 referencias, 97 simples y 15 colecciones;
- 285 miembros y 382 sujetos tipados;
- 364 presencias y 18 ausencias únicamente históricas;
- decisiones históricas 36/49/27;
- las 15 huellas de membresía verificadas;
- la lista ordenada de identidades y su huella propia.

No publica raíces lógicas, rutas relativas o absolutas, perfiles locales,
dispositivos, inodos ni `present_in_stable_view`.

## Arquitectura y módulos

- `main.go` valida la orden, confina las lecturas y coordina generación o
  verificación finitas.
- `model.go` declara los contratos JSON.
- `json_contract.go` rechaza JSON ambiguo y calcula huellas.
- `expand.go` valida cardinalidades y deriva identidades.

No hay servidor, base de datos, cola, red, variables de entorno ni procesos
hijos.

## Autoridad, datos, permisos, secretos y efectos

La aplicación no escribe estado de Orquesta y no tiene autoridad sobre el
censo. No necesita secretos ni credenciales. Su único efecto de datos es
`stdout`; solo un JSON generado o verificado con éxito llega a ese canal. La
ayuda y los errores van a `stderr`. Las únicas lecturas son el V3 y, en modo de
verificación, el universo explícito.

La identidad simple es `{kind:"single", root_id}`. La identidad de miembro es
`{kind:"member", collection_root_id, path_alias}`. El orden conserva el orden de
las 112 referencias V3 y, dentro de cada colección, el orden de `members`.

## Arranque, diagnóstico, recuperación y parada

```bash
go run ./scripts/legacy_physical_expansion \
  --v3 product/traceability/legacy_source_roots_2026-07-30.json

go run ./scripts/legacy_physical_expansion \
  --verify \
  --v3 product/traceability/legacy_source_roots_2026-07-30.json \
  --universe product/traceability/legacy_physical_subject_universe_2026-07-30.json
```

El verificador regenera desde el V3 exacto y compara todos los bytes del
universo recibido; por ello rechaza un cambio de identidad, alias u orden aunque
el documento manipulado conserve su digest declarado. El éxito devuelve un
único documento JSON compacto terminado en LF. Los fallos anteriores a la
emisión dejan `stdout` vacío y devuelven en `stderr` un código máquina y texto
castellano, sin reproducir la ruta recibida. Un fallo de E/S durante la emisión
puede dejar un prefijo parcial en `stdout`; nunca cuenta como resultado. Todo
consumidor debe exigir código de salida cero y, si consume un artefacto,
verificar sus bytes completos mediante `--verify`. No existe recuperación: no
hay estado parcial ni ficheros de salida. La aplicación termina tras escribir
la respuesta y no deja procesos.

## Contratos y pruebas

La huella de referencias conserva el contrato histórico: identificadores en
orden V3 y NUL tras cada uno. Cada `members_sha256` es el JSON compacto del
array tipado seguido de LF. La huella de sujetos usa
`orquesta.legacy-physical-subjects.ordered-canonical-json-lf.v1`, NUL, el array
JSON compacto sin escape HTML y LF. El dominio sí entra en esta última huella;
el documento de salida nunca entra en su propia huella.

No se normaliza Unicode, no se usan flotantes ni BOM y las claves duplicadas
se rechazan. Cambiar escapes o espaciado del V3 también cambia su huella de
bytes y se rechaza.

El esquema común de trazabilidad valida forma, constantes y tipos, pero no
recalcula huellas ni vincula semánticamente los 382 sujetos con el V3. Todo
consumidor debe ejecutar el modo `--verify`; validar solo el esquema o conservar
el texto de `subject_set_sha256` no acredita la equivalencia.

```bash
go test -mod=vendor -count=1 ./scripts/legacy_physical_expansion
go test -mod=vendor -count=1 -race ./scripts/legacy_physical_expansion
GOFLAGS=-mod=vendor go vet ./scripts/legacy_physical_expansion
```
