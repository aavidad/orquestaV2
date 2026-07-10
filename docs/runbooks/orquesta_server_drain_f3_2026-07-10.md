# Drain gobernado F3-R2 de Orquesta

Estado: tooling offline listo para revisión del operador. Este runbook no
autoriza a Codex ni a tests a ejecutar un drain real.

## Preflight sin efectos

Dependencias: Linux con `pidfd_open(2)`, Python 3.10 o posterior que exponga
`os.pidfd_open` y `signal.pidfd_send_signal`, `flock`, `tmux` y `curl`. El
wrapper verifica todo antes de inventariar. Las raíces runtime, state, backup y
`uso-app` deben ser absolutas, canónicas, propiedad del usuario de servicio,
no escribibles por grupo/otros, sin symlinks ni solapes peligrosos.

```bash
scripts/orquesta_server_drain.sh --dry-run
python3 -m json.tool \
  /srv/orquesta-self/claude-director-20260705/state/orquesta_server_drain_receipt_v1.json
```

Un dry-run válido devuelve rc 0 y `drain_status=dry_run`. `refused` siempre
devuelve rc 3, `residual` rc 4 y un lock ocupado rc 73; esas salidas llevan
`orquesta_server_drain=not_ok`, nunca `ok`. Antes de cualquier modo real hay que
revisar `inventory_errors`, `before.drainable`, `before.protected`,
`before.ambiguous`, `owner_markers`, `tmux_sessions`, backup y
`uso_app_protected`.

El marker admitido es el real
`orquesta_codex_app_server_tmux_owner_generation.v0`: `owner_ref`,
`app_server_pid/start_ref`, `socket_owner_pid/start_ref`,
`tmux_session_id/created` y `tmux_pane_pid/start_ref`. Un campo ausente,
marker/symlink inválido, error de `/proc` o tmux, socket no materializado o URL
no causal produce `inventory_errors` y rehúsa señales.

## Secuencia real reservada al operador

```bash
bash -Eeuo pipefail -c '
  scripts/orquesta_server_drain.sh --drain --confirm-drain orquesta-server-drain
  python3 -c '\''import json; p="/srv/orquesta-self/claude-director-20260705/state/orquesta_server_drain_receipt_v1.json"; r=json.load(open(p)); assert r["drain_status"] == "clean" and r["exit_code"] == 0 and not r["inventory_errors"] and r["backup"]["status"] == "complete" and r["backup"]["source_preserved"] is True and r["uso_app_protected"]["intact"] and not r["uso_app_protected"]["signals_sent"]'\''
  ORQUESTA_DEPLOY_REF=trabajo/plataforma-agentes scripts/orquesta_server_deploy.sh
'
```

El backup es precondición fatal y se materializa como ficheros regulares con
SHA-256, sin enlaces al origen. Después: shutdown HTTP solo a una IP loopback
literal cuyo puerto LISTEN pertenece al PID inventariado; TERM mediante pidfd;
espera cooperativa; `tmux -S <socket-exacto> kill-session -t <session_id>` tras
revalidar ID/created/pane/start; nueva espera observable; y KILL final mediante
pidfd solo para la misma identidad kernel. Todo error real queda en `actions`.

## Dos pases amplios aislados

El runner ejecuta exactamente dos pases consecutivos y guarda un receipt por
lote además del receipt agregado:

```bash
ORQUESTA_TEST_BATCH_RECEIPT=/srv/orquesta-self/runtime/test-cache/f3-r2/receipt.json \
  scripts/orquesta_test_batches.sh
python3 -c 'import json; r=json.load(open("/srv/orquesta-self/runtime/test-cache/f3-r2/receipt.json")); assert r["status"] == "passed" and r["passes_required"] == 2 and r["passes_completed"] == [1, 2]'
```

`go list` se captura primero con su rc real; una lista parcial nunca inicia
lotes ni queda verde. Cada lote usa timeout con `--kill-after`, log y receipt.
El perfil común mantiene raíz privada y lease `flock` del rango de puertos; al
terminar elimina solo su env/cache efímero y conserva receipts/logs.

## Pruebas offline

```bash
ORQUESTA_DRAIN_TEST_CACHE_ROOT=/tmp/orquesta-f3-r2-drain \
  bash scripts/test_orquesta_server_drain.sh
ORQUESTA_BATCH_TEST_CACHE_ROOT=/tmp/orquesta-f3-r2-batches \
  bash scripts/test_orquesta_test_batches.sh
```

El primer test crea un único `sleep` hijo y lo termina con el helper pidfd para
probar identidad correcta y start-ref incorrecta. `/proc`, marker, tmux, curl y
el resto del drain son fixtures privadas; el sandbox actual no permite crear
sockets Unix, por lo que solo en `DRAIN_TEST_MODE=1` un fichero regular 0600
representa el inode de socket sintético. Ese hook y todos los overrides de
comandos se rechazan en modo real.

## Fronteras y rollback

El drain no borra estado ni usa `runs/control`, `pkill`, `killall` o señales por
patrón. No protege `uso-app` por texto de cmdline: exige exe/cwd bajo su raíz y
propaga protección por PPID. El deploy conserva su rollback por binario
`.bak.<timestamp>`; el backup del drain no se restaura ni limpia
automáticamente. Esta reparación no demuestra un receipt real `clean`, deploy,
API o servidor reiniciado: esas evidencias siguen reservadas al operador.
