# Incidencia: generation_conflict en el PRIMER lanzamiento local (bloquea todo goal-first)

Fecha: 2026-07-10, ~11:50 UTC.
Detectado por: Claude (revisor), primera ejecucion local del smoke real tras el
corte del lease de generacion unica.
Estado: cerrado por `01cb27d77`. El bloqueo del primer lanzamiento queda
resuelto; se conserva debajo la evidencia historica del fallo original.
Sospechosos directos: commits `6b848faa5` (enforce single app-server generation
identity) y `4a97c37e3` (lease inicial), en
`modulos/orquesta-runtime-codex-appserver/codex_goal_app_server_tmux_generation_lease_v0.go`.

## Sintoma

`scripts/smoke_goal_first_app_server_real.sh` (doble opt-in, proveedor real) en
un entorno temporal LIMPIO:

- `POST /api/v0/apps/director` -> HTTP 500 con
  `arrancar_director_http_error: codex_app_server_tmux_generation_conflict`.
- Despues, DOS `runtime_shutdown_hook` en la auditoria fallan con el MISMO
  `codex_app_server_tmux_generation_conflict`: el conflicto bloquea tambien la
  limpieza del backend, no solo el arranque.

Evidencia retenida completa: `/tmp/orquesta-goal-first-app-server.OvWTgQ`
(`server.stderr.log`, `runtime/goal-srv/`, `state/audit/orquesta_server_audit_v0.jsonl`,
`state/orchestration-state/app_director_goal_states/cbc6c082....json` linea 193).

## Hechos verificados (no hipotesis)

1. Era el PRIMER lanzamiento en un temp-root recien creado: no habia estado
   previo, ni sesiones tmux, ni markers de otra ejecucion.
2. El app-server SI llego a arrancar: existe
   `runtime/goal-srv/orquesta-goal-4e6f58e0a36da4f7.log` con los warnings de
   arranque de codex 0.144.1, y el socket + `stdin.pipe` se crearon.
3. `g-4e6f58e0a36da4f7.sock.owner.json` quedo escrito y completo
   (`generation_ref=generation-ref-90f5f306...`, `lease_owner_pid=1765683`).
   `g-4e6f58e0a36da4f7.sock.owner.lease` quedo VACIO (0 bytes).
4. NO hubo doble lanzamiento: un solo goal state, `orquesta-idle/` vacio, la
   auditoria muestra 4 `idle_self_improvement_check` sin launch propio y solo
   3 `http_request`. El conflicto es del unico lanzamiento contra si mismo.
5. Los primitivos tmux funcionan en esta maquina (tmux 3.6): reproduje a mano
   `new-session -d -e ORQUESTA_CODEX_APP_SERVER_GENERATION_REF=... -s repro`
   + `display-message -p -t repro '#{session_id}\t#{session_created}\t#{pane_pid}'`
   (devuelve exactamente 3 campos tab) y
   `show-environment -t repro ORQUESTA_CODEX_APP_SERVER_GENERATION_REF`
   (devuelve exactamente `VAR=valor`). Ambos rc=0.
6. `go test ./modulos/orquesta-runtime-codex-appserver` esta verde: los tests
   del lease (704 lineas) pasan con fakes mientras el primer uso real falla.
   MISMO patron que la reapertura F3-R2: fixtures que no reproducen el schema
   o el flujo real.

## Donde mirar (acotado por el revisor)

Los puntos que devuelven `codexAppServerTmuxGenerationConflictV0` en el fichero
del lease: lineas ~189/193/200 (`codexAppServerTmuxSessionIdentityFromOutputV0`
y validaciones de identidad), ~212 (`verifyTmuxGenerationTokenV0`, compara
`show-environment` con `WANT=ref` exacto) y ~468. Preguntas concretas:

- Que ref compara `verifyTmuxGenerationTokenV0` contra la sesion: si la sesion
  se creo con un ref y la verificacion usa otro origen (owner.json vs memoria
  vs env del proceso), el primer arranque siempre conflicta.
- El `.lease` vacio: si algun paso posterior espera contenido (pid/starttime)
  en ese fichero y lo trata como identidad invalida -> conflicto.
- `runTmuxCommandV0` con `-t`: comprobar si el target usado en verificacion
  (session_id `$N` vs nombre) coincide con el creado, y si el output llega con
  decoraciones (prefijos, lineas extra) que rompan el `== want` exacto.
- Por que el MISMO conflicto tumba los shutdown hooks: la limpieza no puede
  depender de una verificacion que ya sabemos rota; debe degradar a limpieza
  por identidad de proceso/socket con evidencia, no fallar en silencio.

## Contexto de concurrencia (secundario)

Durante el smoke habia una sesion Codex F5-r8 ejecutando tests de
`cmd/orquesta-server` en la misma maquina, con shims de tmux propios bajo
temp-dirs privados. Los artefactos del smoke estan en directorios privados y
`tmux ls` del servidor por defecto quedo vacio, asi que el cross-talk es poco
probable, pero la reproduccion debe hacerse sin tests concurrentes para
descartarlo del todo.

## Reproduccion

```bash
ORQUESTA_KEEP_SMOKE_DIR=1 \
ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_REAL_CONFIRM=1 \
ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_CODEX_EXECUTION_CONFIRMED=1 \
bash scripts/smoke_goal_first_app_server_real.sh
```

Nota harness: el fallo NO se propago como exit code distinto de 0 observable
(el operador puede creer que paso). Verificar que el smoke devuelve rc!=0
cuando `POST /api/v0/apps/director` responde 500, y anadir al smoke la
asercion de que los shutdown hooks no fallan.

## Criterio de cierre

- Test que reproduzca el PRIMER lanzamiento contra tmux real (o un fake fiel
  al schema real de display-message/show-environment) y falle con el codigo
  actual.
- Smoke real local verde end-to-end: launch, checkpoint, cierre, shutdown
  hooks ok y cleanup sin procesos.
- Shutdown/cleanup nunca bloqueado por generation_conflict: degradacion
  documentada por identidad de proceso.

## Cierre verificable

El inventario canonico `docs/inventario_bugs_estado_vivo.md` registra esta
incidencia como cerrada por `01cb27d77`. El arreglo usa el selector tmux
`=sesion:` y verificacion triestado. Cleanup y shutdown degradan a evidencia
residual cuando corresponde, en lugar de convertir ese estado en un bloqueo.
El indice tambien registra el smoke real local verde end-to-end.
