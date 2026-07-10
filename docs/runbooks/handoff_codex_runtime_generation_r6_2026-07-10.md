# Handoff Codex runtime generation R6 — 2026-07-10

## Alcance

Reparación acotada de los dos fallos de verificación independiente posteriores
a R4/R5. Se preservan sus cambios y el trabajo concurrente del árbol. No se
operaron, observaron, mataron ni reiniciaron procesos Orquesta, `uso-app`, tmux
productivo ni otras aplicaciones. No hubo commit, push o deploy.

## Causa causal

1. `codexAppServerTmuxSocketInodeAtV0` suponía una única fila por pathname en
   `/proc/net/unix`. Un probe Unix puede dejar simultáneamente la fila
   `SO_ACCEPTCON` del listener y filas conectadas con el mismo pathname pero
   otro inode. Se contaban todas y se devolvía
   `codex_app_server_tmux_socket_inode_ambiguous`. Esto rompía el caso wrapper
   real y también hacía que `appServerAliveV0` devolviera falso durante la
   adopción; el segundo fallo era la manifestación generacional de la misma
   observación incorrecta.
2. La adopción tenía además dos huecos conservadores: ignoraba errores de
   `has-session`, tratándolos como ausencia, y cuando la sesión existía no
   revalidaba siempre identidad inmutable más token generacional antes del
   verde. Sin tmux tampoco exigía identidad tmux completa ni coherencia exacta
   entre app-server, owner del socket y pane PID+starttime.

## Cambio R6

- La resolución del inode considera solo sockets Unix stream con
  `SO_ACCEPTCON`, ignora peers conectados del mismo pathname, deduplica filas
  idénticas del mismo inode y conserva conflicto ante dos inodes listener
  distintos.
- Si varios descendientes conservan el mismo FD durante un fork/exec, se elige
  la única hoja causal. Un holder fuera del árbol del pane o varias hojas
  hermanas siguen siendo conflicto; no se acepta un listener ajeno.
- `appServerAliveV0` exige igualdad exacta PID/starttime entre app-server y
  socket owner, pane PID+starttime vivo y descendencia actual.
- Una generación con tmux presente revalida session ID, creación, pane
  PID+starttime y token antes de adoptarse. Una generación sin sesión tmux solo
  se adopta si el marker conserva identidad tmux completa y el listener/owner
  vivo coincide causalmente. Un fallo al observar tmux se propaga y nunca se
  convierte en ausencia ni takeover.

## Cobertura adversarial añadida

- listener más peer conectado con el mismo pathname;
- fila duplicada del mismo inode frente a dos listeners/inodes distintos;
- wrapper que retiene el FD junto con su descendiente;
- holder ajeno y dos hojas hermanas rechazados;
- adopción exacta tras desaparecer tmux con marker completo;
- marker tmux incompleto, app-server/owner desigual y pane PID reutilizado;
- sesión viva con token de otra generación;
- error de observación tmux que no se interpreta como ausencia.

## Verificación de esta sesión

Pasaron:

```text
gofmt sobre los dos ficheros Go modificados
git diff --check -- modulos/orquesta-runtime-codex-appserver
sin salida
```

La verificación Go funcional no se pudo ejecutar en esta sesión: el sandbox
monta `/srv/orquesta-self/runtime` y `/dev/shm` en solo lectura. La creación de
`/srv/orquesta-self/runtime/operator-agent-cache/r6-20260710` y
`/dev/shm/oq-r6` devolvió `Read-only file system`. No se usó `/tmp`, no se
degradó ningún fallo a `SKIP` y no se declara el runtime listo sin esa evidencia.

La batería pendiente debe ejecutarse fuera del sandbox, con base privada corta
en `/dev/shm`, y exige cero `SKIP` funcional, `EDQUOT` o `EINVAL`:

```bash
umask 077
cache=/srv/orquesta-self/runtime/operator-agent-cache/r6-20260710
alias_root=/dev/shm/oq-r6
mkdir -p "$cache"/{go-cache,go-tmp,tmp,flaky} "$alias_root"
chmod 700 "$cache" "$cache"/{go-cache,go-tmp,tmp,flaky} "$alias_root"

env \
  GOCACHE="$cache/go-cache" \
  GOTMPDIR="$cache/go-tmp" \
  TMPDIR="$cache/tmp" \
  ORQUESTA_FLAKY_HARNESS_CACHE_ROOT="$cache/flaky" \
  GOMODCACHE=/srv/orquesta-self/runtime/server-latest/go-mod-cache \
  ORQUESTA_TEST_UNIX_SOCKET_ALIAS_ROOT="$alias_root" \
  go test -count=1 ./modulos/orquesta-runtime-codex-appserver -v

env \
  GOCACHE="$cache/go-cache" \
  GOTMPDIR="$cache/go-tmp" \
  TMPDIR="$cache/tmp" \
  ORQUESTA_FLAKY_HARNESS_CACHE_ROOT="$cache/flaky" \
  GOMODCACHE=/srv/orquesta-self/runtime/server-latest/go-mod-cache \
  ORQUESTA_TEST_UNIX_SOCKET_ALIAS_ROOT="$alias_root" \
  go test -race -count=1 ./modulos/orquesta-runtime-codex-appserver \
    -run 'Test(SocketOwnerV0AceptaDescendienteDeWrapperV0|SocketOwnerProcV0.*|EnsureV0AdoptaGeneracionExactaCuandoTmuxDesapareceV0|EnsureV0SinTmux.*|EnsureV0ConTmuxRechazaOtraGeneracionV0|EnsureV0NoAdoptaSiAusenciaTmuxNoEsObservableV0)$' -v

env \
  GOCACHE="$cache/go-cache" \
  GOTMPDIR="$cache/go-tmp" \
  TMPDIR="$cache/tmp" \
  ORQUESTA_FLAKY_HARNESS_CACHE_ROOT="$cache/flaky" \
  GOMODCACHE=/srv/orquesta-self/runtime/server-latest/go-mod-cache \
  ORQUESTA_TEST_UNIX_SOCKET_ALIAS_ROOT="$alias_root" \
  go test -count=40 ./modulos/orquesta-runtime-codex-appserver \
    -run 'Test(SocketOwnerV0AceptaDescendienteDeWrapperV0|EnsureV0AdoptaGeneracionExactaCuandoTmuxDesapareceV0)$' -v

find "$alias_root" -mindepth 1 -print -quit
rmdir "$alias_root"
```

## Riesgos residuales

- La prueba funcional depende de Linux procfs y visibilidad de los FDs del
  mismo usuario; un procfs endurecido debe fallar conservadoramente, no adoptar.
- La selección de hoja única admite el solape real de FD durante fork/exec,
  pero conserva ambigüedad cuando no hay un único descendiente causal.
- Hasta ejecutar paquete, focal `-race` y estrés `-count=40` fuera del sandbox,
  R6 no está listo para operación.

estado_final: verification_blocked_by_readonly_sandbox_r6
