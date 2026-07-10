# Handoff Codex F3-R2 - 2026-07-10

Alcance: reparación crítica offline de TAREA-F3/BUG-208E después de que la
revisión adversarial invalidara el verde F3 previo. No se ejecutaron drain,
deploy, restart, API, `runs/control`, OPES, `uso-app` ni otras aplicaciones. No
se observó ni señaló ningún proceso ajeno; la única señal real de las pruebas
fue a un `sleep` hijo efímero creado por el propio test para validar pidfd.

## Resultado técnico

- El inventario consume el marker real
  `orquesta_codex_app_server_tmux_owner_generation.v0` con `owner_ref`,
  identidades app-server/socket-owner/pane y sesión tmux completas. Errores de
  `/proc`, markers, campos, tmux, sockets, PID file o listener se materializan
  en `inventory_errors` y rehúsan toda mutación.
- `uso-app` se protege por raíz canónica + exe/cwd y descendencia PPID. Una
  mera mención textual no protege ni oculta un residuo Orquesta ambiguo.
- La URL usa `urllib.parse.urlsplit`: solo `http`, IP loopback literal, sin
  userinfo/query/fragment, puerto válido y asociado al inode LISTEN del único
  servidor anclado por `server.pid`. El bypass `127.0.0.1:x@host` queda
  rechazado antes de curl.
- La señal real abre pidfd, relee start-ref y usa `pidfd_send_signal` sobre el
  mismo objeto kernel. Resultado `sent` solo se registra si el helper devuelve
  rc 0 y payload `sent`; no existe check Python seguido de `kill` shell.
- Tmux se revalida por socket exacto, session ID, created, pane PID y start-ref;
  mata por session ID inmutable, espera salida observable tras `kill-session` y
  solo después escala residuos por pidfd.
- Backup es precondición fatal: raíces/owner/symlink seguros, contenido
  materializado en ficheros regulares, SHA-256, verificación antes/después y
  `source_preserved`. Error, enlace externo o ref no regular produce rc 3 sin
  curl ni señales.
- Dry-run válido: `drain_status=dry_run`, rc 0. Drain real limpio: `clean`, rc
  0. Todo `refused`: rc 3; todo `residual`: rc 4; lock ocupado: rc 73. Los tres
  fallos imprimen `orquesta_server_drain=not_ok`.
- Drain tiene `flock` exclusivo. `receipt_id`, `backup_id` y request ID llevan
  nanosegundos y nonce.
- Los hooks de comandos, proc y sockets sintéticos requieren simultáneamente
  `DRAIN_TEST_MODE=1` y `/proc` sintético. En modo real se rehúsan.
- El perfil aislado exige raíz privada/canónica/no-symlink, adquiere lease
  `flock` del rango de puertos y exporta el rango efectivo. Deploy conserva el
  mismo env/lease desde build hasta postverify; nightly y batch limpian solo sus
  env efímeros y conservan receipts/logs.
- El runner captura rc de `go list` antes de `mapfile`, nunca usa lista parcial,
  ejecuta exactamente dos pases, aplica `timeout --kill-after` y deja receipt
  por lote y agregado.

## Pruebas ejecutadas

Verdes:

- `bash -n` de drain, batch, perfil aislado, deploy, nightly y consumidores.
- `python3 -m py_compile scripts/lib/pidfd_signal.py scripts/lib/orquesta_drain_runtime.py`.
- `ORQUESTA_DRAIN_TEST_CACHE_ROOT=/tmp/... bash scripts/test_orquesta_server_drain.sh`:
  verde. Cubre proceso pidfd real propio, start-ref incorrecto, marker real,
  dry-run/clean/refused/residual y rc, error real de señal, identidad tmux
  sustituida, socket/ID exactos, backup/symlink fatal, `/proc` y tmux fallidos,
  URL injection, texto `uso-app`, overrides prohibidos y drain concurrente.
- `ORQUESTA_BATCH_TEST_CACHE_ROOT=/tmp/... bash scripts/test_orquesta_test_batches.sh`:
  verde. Cubre dos pases 2/2/1, receipts por lote, `--kill-after`, lote fallido,
  `go list` parcial rc 9, raíz symlink, lease de puerto concurrente, retención y
  ejecución sin `rg`.
- `TMPDIR=/tmp/... ORQUESTA_TEST_CACHE_ROOT=/tmp/... bash scripts/test_orquesta_server_deploy.sh`:
  verde offline; no hizo deploy real.
- `go test -count=1 ./scripts -v` con GOCACHE/GOTMPDIR en `/tmp` y
  GOMODCACHE local: verde (`TestF3R2PidfdHelperAST`, `TestF3R2ShellSyntax`).
- `git diff --check`: verde sobre el worktree concurrente.

Bloqueos honestos, no verdes:

- El primer intento del guard focal de `./cmd/orquesta-server` quedó bloqueado
  por un cambio concurrente fuera de F3-R2: referencias entonces inválidas a
  `ClaudeRuntimeConfigV0.Model` y `.Effort`. Ese owner cambió durante la sesión;
  el reintento posterior compiló paquetes y llegó al link del enorme binario de
  test, pero el sandbox terminó la ejecución sin resultado terminal. No se
  declara verde. El guard semántico equivalente sí queda verde en `./scripts`.
- `scripts/test_orquesta_smoke_nightly.sh` se bloquea antes de probar nightly:
  el sandbox devuelve `EPERM` al crear el socket AF_INET del servidor Telegram
  falso. No se añadió skip ni se declaró verde.
- No se ejecutó la tanda Go amplia real de dos pases: el operador pidió no tocar
  procesos/apps vivos y el paquete `cmd/orquesta-server` ya está bloqueado por
  la compilación concurrente anterior.

## Operación pendiente

El runbook vigente es
`docs/runbooks/orquesta_server_drain_f3_2026-07-10.md`. El operador debe obtener
primero resultado terminal del guard `cmd/orquesta-server`, revisar un dry-run,
y solo en una ventana autorizada producir receipt real `clean` más receipt batch con
`passes_completed=[1,2]`. Esta sesión no aporta ni simula esas evidencias.

No se creó commit: `.git` es de solo lectura para esta sesión y además el
worktree contiene cambios concurrentes que deben preservarse.

estado_final: ready_for_operator_f3_r2
