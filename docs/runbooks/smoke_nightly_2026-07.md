# Smoke nightly Orquesta

Fecha: 2026-07-03.

## Objetivo

Ejecutar cada noche el smoke de Nueva App goal-first sin operador y dejar un
artefacto JSON fechable. Por defecto solo corre preflight: no arranca Codex
real, no consume cuota, no toca OPES y no hace commits.

Script: `scripts/orquesta_smoke_nightly.sh`.

Resultado por defecto:

```text
~/.orquesta-nightly/resultado_YYYYMMDD.json
```

El JSON incluye `exit_code`, `status`, `phase_reached`, `phases`, refs
observadas, ref git, estado dirty/clean, notificacion terminal, duracion y ruta
del log. La retencion borra resultados y logs con mas de 30 dias; ajustar con
`ORQUESTA_NIGHTLY_RETENTION_DAYS`.

## Preflight

Comando manual barato:

```bash
scripts/orquesta_smoke_nightly.sh
```

Equivale a:

```bash
ORQUESTA_GOAL_FIRST_SMOKE_PREFLIGHT_ONLY=1 \
scripts/smoke_goal_first_app_server_real.sh
```

Debe validar la CLI `codex app-server`/`tmux` sin generar app ni gastar cuota.
Si `OPES_BASE_URL`, `ORQUESTA_OPES_BASE_URL` o confirmaciones productivas OPES
estan presentes, el wrapper bloquea antes de llamar al smoke base.

## Modo real opt-in

El modo real solo se activa si la unidad o cron exporta:

```bash
ORQUESTA_NIGHTLY_REAL_CONFIRM=1
```

Con esa confirmacion, el wrapper exporta las confirmaciones requeridas por
`scripts/smoke_goal_first_app_server_real.sh` y ejecuta Nueva App goal-first en
un proyecto temporal. OPES queda vacio. No hay commits automaticos ni push.

Variables utiles:

- `ORQUESTA_CODEX_COMMAND`: ruta de `codex`.
- `ORQUESTA_CODEX_CODE_HOME`: fuente de auth/config para Codex real.
- `ORQUESTA_CODEX_MODEL`, `ORQUESTA_CODEX_REASONING_EFFORT`.
- `ORQUESTA_GOAL_FIRST_SMOKE_POLLS`,
  `ORQUESTA_GOAL_FIRST_SMOKE_SLEEP_SECONDS`.
- `ORQUESTA_KEEP_SMOKE_DIR=1` para conservar temporales del smoke base.
- `ORQUESTA_NIGHTLY_RESULTS_DIR` para pruebas o rutas no estandar.

## systemd de usuario

Servicio preflight por defecto:

```ini
# ~/.config/systemd/user/orquesta-smoke-nightly.service
[Unit]
Description=Orquesta nightly smoke preflight

[Service]
Type=oneshot
WorkingDirectory=/home/alberto/Trabajo/orquesta
ExecStart=/home/alberto/Trabajo/orquesta/scripts/orquesta_smoke_nightly.sh
```

Timer diario:

```ini
# ~/.config/systemd/user/orquesta-smoke-nightly.timer
[Unit]
Description=Run Orquesta nightly smoke

[Timer]
OnCalendar=*-*-* 03:30:00
Persistent=true

[Install]
WantedBy=timers.target
```

Activacion:

```bash
systemctl --user daemon-reload
systemctl --user enable --now orquesta-smoke-nightly.timer
```

Para modo real, anadir al servicio una linea de entorno decidida por el
operador:

```ini
Environment=ORQUESTA_NIGHTLY_REAL_CONFIRM=1
```

No poner variables OPES en esta unidad.

## Notificacion Telegram

El nightly lee la configuracion canonica de Telegram desde:

1. `ORQUESTA_CTL_CONFIG`, si existe.
2. `$ORQUESTA_CTL_WORKDIR/orquesta.config.json`, si existe.
3. `./orquesta.config.json`, si existe.

Si `telegram_operator.enabled=true`, el script envia un mensaje terminal por
Bot API con `status`, `mode`, `phase`, `run_id`, ref git, dirty/clean y ruta del
resultado. El token y el target salen solo de `telegram_operator.*`; no hay
variables de entorno para token/chat. Si Telegram esta activado pero incompleto
o la API rechaza el envio, un smoke que iba verde pasa a `failed` con
`phase_reached=notification_failed`.

Para tests locales con Bot API falso se puede inyectar
`ORQUESTA_NIGHTLY_TELEGRAM_BOT_API_BASE_URL`; no usarla como configuracion
productiva.

## cron de usuario

Preflight diario:

```cron
30 3 * * * cd /home/alberto/Trabajo/orquesta && scripts/orquesta_smoke_nightly.sh
```

Real opt-in:

```cron
30 3 * * * cd /home/alberto/Trabajo/orquesta && ORQUESTA_NIGHTLY_REAL_CONFIRM=1 scripts/orquesta_smoke_nightly.sh
```

## Fallos

Si el resultado corriente no es `ok`, el script imprime un diff de fases contra
el ultimo `resultado_*.json` verde disponible. Esa salida debe bastar para abrir
fila en `docs/inventario_bugs_orquesta_2026-06-30.md` con fase, refs y log.

Estados esperados:

- `ok`: preflight o smoke real completado.
- `blocked`: falta configuracion local o hay entorno OPES/productivo.
- `failed`: el smoke base arranco y fallo.

Nota: `ok` en modo `preflight` no demuestra ejecucion real de proveedor. Para
cerrar un nightly real de campo debe constar `mode=real`,
`ORQUESTA_NIGHTLY_REAL_CONFIRM=1`, `git.ref` y notificacion Telegram recibida.

## Verificacion local

Sin cuota:

```bash
bash -n scripts/orquesta_smoke_nightly.sh scripts/test_orquesta_smoke_nightly.sh
scripts/test_orquesta_smoke_nightly.sh
```

El guard usa un smoke falso, Bot API falso y escribe en un directorio temporal.
