# Incidencia local 2026-07-11: preflight, consumo y scope en autoprogramacion

Estado: `BUG-ORQ-20260711-221/225` cerrados localmente; `222/223/224` abiertos.
Alcance: solo Orquesta local. No se toco remoto, OPES ni produccion.

## Contexto

Se uso el binario local de Orquesta en `91c3dbff0` para encargar por
`POST /api/v0/autoprogramming/prepare-run` una tarea real de limpieza de
configuracion. El estado y runtime aislados quedan retenidos bajo
`/tmp/orquesta-self-hermes-20260711` mientras esta incidencia sea util.

## BUG-ORQ-20260711-221: `run` publicaba readiness sin backend Codex valido

El primer servidor arranco con
`ORQUESTA_CODEX_COMMAND=/usr/local/bin/codex`, ruta absoluta inexistente.
Publico readiness verde y acepto `run-hermes-config-20260711-001`, pero el
backend termino `invalid` con `codex_app_server_tmux_socket_timeout`. El log
causal contiene `No such file or directory`.

Causa: `startServerCommandV0` ejecutaba `validateCodexCommandAvailableV0`,
pero `runServerCommandV0` no. El timeout de socket ocultaba un error de
configuracion demostrable antes de abrir el listener.

Cierre local: `run` usa el mismo preflight despues de resolver identidad. El
orden conserva `degraded_identity`, que no necesita construir runtime Codex.
Pruebas focales: `go test -count=1 ./cmd/orquesta-server -run CodexCommand`.

## BUG-ORQ-20260711-222: goal estrecho consumio mas de 1,1 M tokens sin cierre

El run `run-run-codex-preflight-20260711-001`, goal externo
`019f4fa2-e2d1-7e62-9f4c-b3be100a7833`, materializo un diff de 65 lineas y
paso el test focal, pero no escribio result durable. El contador vivo alcanzo
al menos `1.194.800` tokens. El observador publico
`checkpoint_only_high_consumption` y
`goal-observer-high-consumption-stop-requested`, pero el app-server/tmux
continuo vivo hasta un `POST /api/v0/runs/control` forzado. Ese control si
confirmo `goal_status_after=blocked`, `goal_control_signal_confirmed=true` y
elimino todos los procesos del goal.

Pendiente estructural: demostrar que el limite de consumo ejecuta y confirma
el actuador de proceso, no solo persiste una peticion cooperativa; separar
progreso real de repeticion interna y exigir result/checkpoint enriquecido
antes de seguir consumiendo. Debe haber test real que pruebe proceso ausente
tras el corte automatico.

## BUG-ORQ-20260711-223: `observe` manual dio 500 durante observacion residente

Mientras el goal anterior estaba `running`, el observador residente produjo
ticks `ok` con una observacion. En paralelo, un
`POST /api/v0/autoprogramming/goal/observe` para el mismo `run_ref` devolvio
HTTP 500 con `autoprogramming_observe_goal_error`. No se perdio el goal, pero
la superficie publica no distinguio contencion/reintento de un fallo interno.

Pendiente: reproducir concurrencia entre observer residente y llamada manual;
si existe lease ocupado, devolver estado recuperable y accion `retry/observe`,
no 500 generico. Conservar causa publica sin filtrar detalles sensibles.

## BUG-ORQ-20260711-224: rework escribio fuera del write-set

El rework `run-run-preflight-rework-20260711-001` declaro exclusivamente
`commands.go` y `commands_test.go`, pero modifico tambien `daemon.go`. La
integracion detecto el desvio y retiro ese cambio antes de aceptar el diff.

Pendiente estructural: atestar el snapshot final contra el write-set congelado
y bloquear cierre/promocion si aparece una ruta no autorizada. El Director
puede abrir un rework causal con write-set ampliado, pero el adaptador no debe
aceptar silenciosamente la escritura.

## BUG-ORQ-20260711-225: el perfil aislado filtraba `umask 077` a los tests

La suite amplia bajo `scripts/lib/isolated_test_env.sh` dio dos falsos rojos
de seguridad: un secreto bajo un padre deliberadamente escribible por grupo y
un store con raiz permisiva parecian seguros. Los mismos focales pasaron fuera
del perfil. La causa reproducida es que `orquesta_use_isolated_test_env` cambia
el umask del shell llamador a `077` y no lo restaura; por ello `os.Mkdir(...,
0770/0777)` dentro de los tests materializa `0700` y nunca construye el caso
inseguro que pretende verificar.

Cierre local: el perfil guarda y restaura exactamente el umask del llamador en
exito y error, sin relajar los modos `0700/0600` del setup. La prueba del
script valida ambos caminos; los dos focales de permisos pasan de nuevo dentro
del perfil aislado.

## Evidencia y cierre del corte

- Runtime fallido: `/tmp/orquesta-self-hermes-20260711/runtime`.
- Runtime del goal de implementacion: `runtime-2`; estado: `state-2`.
- Runtime del rework: `runtime-3`; estado: `state-3`.
- El control forzado de ambos goals confirmo parada y no quedaron procesos
  `orquesta-server`, `codex app-server`, `orquesta-goal-*` ni tmux del corte.
- El diff integrado queda limitado a `cmd/orquesta-server/commands.go` y
  `cmd/orquesta-server/commands_test.go`.
- `bash scripts/test_orquesta_test_batches.sh`, los focales de permisos y
  `go test -count=1 ./...` bajo el perfil aislado quedan verdes tras cerrar
  `221/225`.

No declarar `222/223/224` cerrados por los arreglos de `221/225`: comparten la ruta
de autoprogramacion, pero tienen criterios de cierre independientes.
