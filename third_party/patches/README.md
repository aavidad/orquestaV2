# Parches reproducibles de dependencias

`modernc_sqlite_v1.53.0_runtime_scope.patch` es el único parche local de
dependencia del rebuild. Se aplica sobre `modernc.org/sqlite v1.53.0`, fijado en
`go.mod` y `vendor/modules.txt`.

Motivo: V14 debe validar el descriptor del fichero SQLite que cada conexión
física abrió, antes de pragmas y hooks. La API oficial de `FileControl` no
expone ese descriptor y `Driver` no ofrece un `Connector` validable. Validar
solo la ruta deja una carrera ABA; abrir `/proc/self/fd` crea otro namespace
WAL/SHM y no es equivalente. El parche añade únicamente:

- `NewConnector` y `NewValidatedConnector`;
- apertura validada sin `SQLITE_OPEN_CREATE`;
- acceso prestado al descriptor mediante `SQLITE_FCNTL_FILE_POINTER`.

Fuente oficial consultada el 2026-07-16:
<https://pkg.go.dev/modernc.org/sqlite>. La versión publicada 1.54.0 todavía no
ofrece esas APIs. Antes de actualizar la dependencia, comprobar si upstream ya
permite la misma garantía y retirar el parche si existe equivalencia probada.

Tras regenerar vendor:

```bash
go mod vendor
git apply --whitespace=nowarn -p1 \
  third_party/patches/modernc_sqlite_v1.53.0_runtime_scope.patch
go test -mod=vendor -count=1 . \
  -run '^TestModerncSQLiteRuntimeScopePatchIsReproducible$'
go test -mod=vendor -count=1 ./internal/adapters/state/sqlite
```

El test reconstruye los cinco ficheros desde el commit base V14, aplica el
parche en temporal y compara bytes contra vendor. `go mod vendor` sin reaplicar
el parche deja el gate rojo.
