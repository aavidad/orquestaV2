# Handoff Codex runtime generation R5 — 2026-07-10

## Alcance

Reparación del bloqueo del harness Unix posterior a R4. El cambio queda
limitado a tests/helpers de `modulos/orquesta-runtime-codex-appserver` y a este
handoff. Se preservan los cambios R4 y los cambios concurrentes del árbol.

No se operaron procesos o runtimes Orquesta vivos, deploy, `uso-app` ni otras
apps. Las pruebas del paquete solo usaron binarios/fixtures efímeros propios del
harness. No se usó `codebase-memory-mcp`.

## Causa R5

La suite externa R4 no alcanzó sus aserciones funcionales porque
`shortUnixSocketTestRootV0` reservaba siempre el alias con
`os.CreateTemp("/tmp", "oq-gl-")`. El host devolvió `EDQUOT`; por tanto, aquel
resultado no contenía un `SKIP` funcional evaluable ni validaba R4.

## Contrato implementado

- `ORQUESTA_TEST_UNIX_SOCKET_ALIAS_ROOT` permite fijar una raíz explícita. Debe
  ser absoluta, directorio real sin componentes symlink, propiedad del EUID,
  privada y escribible/atravesable por el owner. Una raíz explícita inválida
  falla: no se degrada silenciosamente a otra ubicación.
- Sin configuración explícita se intenta, en orden:
  1. `XDG_RUNTIME_DIR` con las mismas garantías privadas;
  2. `/dev/shm` como base compartida para crear un subdirectorio corto privado;
  3. `/tmp` solo como último fallback validado.
- Las bases y todos sus componentes se inspeccionan con `Lstat`; un symlink se
  rechaza. El root efectivo del alias siempre es un subdirectorio nuevo modo
  privado, propiedad del EUID y revalidado contra sustitución. Una base
  compartida ajena nunca se usa directamente como root del alias.
- El alias `r` sigue apuntando al `t.TempDir` real. El cleanup compara
  dispositivo, inode, modo y UID antes de retirar el alias y su subdirectorio;
  usa eliminaciones no recursivas y no toca sustitutos ni contenido ajeno.
- La longitud en bytes se valida contra la capacidad real de
  `syscall.RawSockaddrUnix.Path` antes de `syscall.Socket`/`Bind` y antes de
  cada `net.Listen("unix", ...)` del harness afectado. Una ruta larga devuelve
  el error propio `errUnixSocketTestPathTooLongV0`, no el `EINVAL` tardío del
  kernel.
- Cualquier error de creación, incluido `EDQUOT`, llega a `Fatalf`; no existe
  una rama que lo convierta en `SKIP`. Los `SKIP` heredados siguen limitados a
  la frontera `EPERM`/`operation not permitted` del sandbox para listeners.

## Cobertura añadida

- raíz configurable autoritativa con `TMPDIR` y `XDG_RUNTIME_DIR` inutilizables;
- preferencia por un `XDG_RUNTIME_DIR` privado sin consultar `/tmp`;
- rechazo de permisos amplios, falta de write/execute del owner, UID ajeno y
  raíces symlink;
- alias apuntando al `t.TempDir`, cleanup completo y conservación de un alias
  sustituido;
- preflight determinista de exceso de `sockaddr_un` antes de abrir un socket;
- 100 repeticiones del selector, cleanup y preflight.

## Verificación ejecutada

Las cachés de esta continuación se aislaron bajo
`/tmp/orquesta-r5-real.Tofckt`; no se usó el repo como caché.

```text
go test -count=1 ./modulos/orquesta-runtime-codex-appserver
ok orquesta/modulos/orquesta-runtime-codex-appserver

go test -race -count=1 ./modulos/orquesta-runtime-codex-appserver \
  -run 'Test(ShortUnixSocketTestRootV0|UnixSocketAlias|ListenUnixFDForGenerationTestV0|MarkerCASV0ConcurrenteMismaGeneracionTieneUnSoloGanadorV0|QuarantineMarkerV0DetectaSustitucionAntesDeRenameV0)'
ok orquesta/modulos/orquesta-runtime-codex-appserver

go test -count=100 ./modulos/orquesta-runtime-codex-appserver \
  -run 'Test(ShortUnixSocketTestRootV0AcotaSockaddrV0|UnixSocketAliasRootV0ConfiguradaNoRequiereTmpYSeLimpiaV0|UnixSocketAliasCleanupV0NoEliminaSustitutoV0|ListenUnixFDForGenerationTestV0RechazaLongitudAntesDeSocketV0)$'
ok orquesta/modulos/orquesta-runtime-codex-appserver
```

También pasó un focal con `ORQUESTA_TEST_UNIX_SOCKET_ALIAS_ROOT` apuntando a un
subdirectorio privado corto efímero de `/dev/shm`; al terminar no quedaron
entradas bajo esa raíz.

## Frontera de la prueba Unix en este sandbox

La suite focal de listeners llegó a rutas cortas como
`/dev/shm/oq-*/r/runtime/goal-srv/s.sock`: no apareció `EDQUOT`, no se consultó
`/tmp` para los alias y desapareció `bind: invalid argument`. Sin embargo, los
siete casos con listener quedaron en `SKIP` porque el sandbox devuelve
`setsockopt: operation not permitted` al crear el listener Unix. No se cuentan
como verde funcional de R4/R5.

El operador debe repetir fuera de este sandbox, donde `listen(AF_UNIX)` esté
permitido:

```bash
mkdir -p \
  "$ORQUESTA_FLAKY_HARNESS_CACHE_ROOT/go-cache-r5" \
  "$ORQUESTA_FLAKY_HARNESS_CACHE_ROOT/go-tmp-r5" \
  "$ORQUESTA_FLAKY_HARNESS_CACHE_ROOT/tmp-r5"

GOCACHE="$ORQUESTA_FLAKY_HARNESS_CACHE_ROOT/go-cache-r5" \
GOTMPDIR="$ORQUESTA_FLAKY_HARNESS_CACHE_ROOT/go-tmp-r5" \
TMPDIR="$ORQUESTA_FLAKY_HARNESS_CACHE_ROOT/tmp-r5" \
go test -count=1 ./modulos/orquesta-runtime-codex-appserver -v
```

La aceptación del operador exige cero `SKIP` en los casos Unix, ausencia de
`EDQUOT`/`EINVAL`, y paso de las aserciones generation/lease/cleanup R4. Si la
raíz de cache es larga, puede fijarse una raíz corta privada con
`ORQUESTA_TEST_UNIX_SOCKET_ALIAS_ROOT`; el contenido de los tests seguirá bajo
`t.TempDir`.

## Higiene

La caché R5 nueva de `/tmp` se retira al cierre de esta sesión. La caché previa
`/srv/orquesta-self/runtime/tmp-deploy/orquesta-r5-test.SUxZBQ`, creada antes de
la continuación, quedó remontada read-only para este sandbox (`EROFS`): no se
forzó ni se borró contenido fuera de una ruta propia. Debe retirarse cuando el
mount vuelva a ser escribible si aún existe.

No se creó commit ni se tocó `.git`.

estado_final: ready_for_operator_runtime_r5
