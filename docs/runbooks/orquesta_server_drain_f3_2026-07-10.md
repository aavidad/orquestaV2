# Drain gobernado F3 de orquesta-server

Estado: artefacto operativo para `task-ref-orquesta-f3-drain-rework-20260710`.

Este procedimiento no ejecuta un drain real por defecto. Sin argumentos y con
`--dry-run`, `scripts/orquesta_server_drain.sh` inventaria procesos, prepara
backup y escribe recibo JSON, pero solo registra `dry_run_skip` y
`drain_status=refused`. Para enviar senales cooperativas hace falta
confirmacion explicita con `--drain --confirm-drain orquesta-server-drain`.

Comando unico recomendado para validacion aislada:

```bash
source scripts/lib/isolated_test_env.sh && \
orquesta_use_isolated_test_env /srv/orquesta-self/runtime/test-cache/f3-rework && \
bash scripts/test_orquesta_server_drain.sh
```

Si la raiz solicitada no es escribible por el sandbox, el helper conserva
`ORQUESTA_ISOLATED_TEST_ENV_REQUESTED_ROOT` y usa
`ORQUESTA_ISOLATED_TEST_ENV_ROOT` bajo el worktree. Cuando existe la cache local
`/srv/orquesta-self/runtime/go-mod-cache`, la siembra en la cache aislada para
evitar descargar modulos durante tests sin red.

Comando de inspeccion no destructiva:

```bash
source scripts/lib/isolated_test_env.sh && \
orquesta_use_isolated_test_env /srv/orquesta-self/runtime/test-cache/f3-rework && \
ORQUESTA_DRAIN_RECEIPT=/srv/orquesta-self/runtime/test-cache/f3-rework/orquesta_server_drain_receipt_v0.json \
ORQUESTA_DRAIN_BACKUP_DIR=/srv/orquesta-self/runtime/test-cache/f3-rework/drain-backup \
bash scripts/orquesta_server_drain.sh --dry-run
```

Contrato operativo:

- `uso-app` queda protegido por `ORQUESTA_DRAIN_PROTECTED_APP_ROOT` y nunca se
  considera target aunque coincida con patrones de runtime.
- El inventario previo, inventario posterior, acciones, backup y raices usadas
  quedan en el recibo `orquesta_server_drain_receipt.v0`.
- El backup se prepara antes de cualquier accion de parada.
- El script no depende de `/api/v0/runs/control`.
- La secuencia confirmada es shutdown HTTP contractual del servidor propio,
  `INT`, `TERM` y `tmux kill-session` solo para sesiones `orquesta-goal-*`.
- No se usa `kill -9` ni `KILL`; la decision es conservadora porque el drain F3
  se usa para romper el ciclo de deploy sin arriesgar servicios ajenos. Si un
  residual exige `KILL`, debe quedar como residual explicito en el recibo y
  decidirlo un operador o un goal posterior con evidencia.

Incidencia observada durante este corte: el goal
`goal-ref-task-autoprogramming-6dc6ba2489b7-g01` quedo `running` sin resultado
terminal tras escribir artefactos parciales. Codex aplico este ajuste como
desbloqueo acotado y debe registrarse como fallo de autonomia/reconciliacion si
el goal no se autocierra tras observar los artefactos.
