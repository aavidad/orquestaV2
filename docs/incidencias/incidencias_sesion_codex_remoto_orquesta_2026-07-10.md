# Incidencias de sesion Codex/Orquesta remoto

Fecha: 2026-07-10  
Estado: abierto  
Objetivo de la sesion: dejar Orquesta trabajando en remoto, conservar los
avances de los agentes, documentar fallos estructurales y preparar commit/push
sin tocar `uso-app` ni otros servicios productivos.

Este documento agrupa todos los fallos observados en la sesion para revision de
Claude. No sustituye a las incidencias detalladas; sirve como indice operativo
para detectar patrones estructurales.

## Fallos observados

### S1 - Servidor remoto vivo con worktree obsoleto

El proceso remoto de Orquesta estaba vivo, pero arrancado con
`ORQUESTA_CTL_WORKDIR=/srv/orquesta-self/worktrees/orquesta-repair-checkpoint-20260709T215549Z`,
un worktree que habia sido retirado en una limpieza previa.

Estado: mitigado en runtime, no cerrado estructuralmente. Se reinicio solo
Orquesta apuntando a `/srv/orquesta-self/worktrees/orquesta`, sin tocar
`uso-app`. Falta que el servidor detecte y bloquee esta condicion por si mismo.

Relacion: `BUG-ORQ-20260710-208A`.

### S2 - Limpieza operativa confundida con limpieza de codigo muerto

La orden de limpiar se interpreto en parte como limpieza de worktrees/agentes
vivos. El operador aclaro que se referia a codigo muerto, historicos falsos,
duplicados y variables/envs, no a eliminar artefactos o workdirs de ejecucion
activa.

Estado: abierto como riesgo operativo. Antes de borrar cualquier worktree,
checkpoint, runtime o cache hay que verificar procesos, refs, manifests,
`agent_ack`, `director_decisions`, `outbox` y evidencias.

Relacion: regla de higiene en `AGENTS.md` y `BUG-ORQ-20260710-208A`.

### S3 - Remoto no estaba sincronizado con GitHub

El remoto tenia `origin=/tmp/orquesta-self.bundle`, no el remoto GitHub. Local
y GitHub estaban en `e715812f4`, mientras el remoto seguia en
`16224f637b` con cambios sin commitear generados por agentes.

Estado: abierto hasta commit/push remoto. No se debe asumir que `git pull` en
remoto trae GitHub mientras `origin` apunte al bundle temporal.

### S4 - Goal-first remoto acepto trabajo amplio y quedo inconsistente

El `POST /api/v0/autoprogramming/prepare-run` remoto acepto cuatro goals
paralelos. Varios solo materializaron `checkpoint_started`; algunos escribieron
`orquesta_goal_result` con `status=blocked`, pero la API seguia exponiendo
`goal_status=running`.

Estado: abierto. Incidencia detallada en
`docs/incidencias/incidencia_orquesta_goal_first_remote_checkpoint_control_2026-07-10.md`.

Relacion: `BUG-ORQ-20260710-208B`.

### S5 - `runs/control` no paro el backend goal-first

El intento de parar `goal-04` por `/api/v0/runs/control` devolvio
`control_not_propagated_to_goal_backend`. El reintento con `forced=true`
mantuvo `goal_status_after=running`.

Estado: abierto. La reparacion actual normaliza parte de la observacion y
secuencia write-sets, pero no cierra todavia la propagacion efectiva de
stop/cancel al backend.

Relacion: `BUG-ORQ-20260710-208C`.

### S6 - `observe_goal` devolvio 504 con snapshot parcial contradictorio

`POST /api/v0/autoprogramming/goal/observe` devolvio
`autoprogramming_observe_goal_timeout`; el parcial mezclaba
`goal_status=running` con `closure_status=blocked`.

Estado: mitigado por patch focal: en timeout parcial con goal running ya no se
publica cierre bloqueado ni replan falso. Falta desplegar y probar por API en
remoto.

Relacion: `BUG-ORQ-20260710-208B`.

### S7 - Batch paralelo acepto write-sets solapados

La request tenia al menos dos tareas con write-set `scripts`. Orquesta las
lanzo en paralelo en vez de secuenciarlas, rechazarlas o pedir replan.

Estado: mitigado por patch focal en `orquesta-autoprogramming`: los write-sets
declarados solapados se secuencian. Falta desplegar y demostrar con repro/API.

Relacion: `BUG-ORQ-20260710-208D`.

### S8 - Agentes remotos fallaron aplicando parches por contexto stale

El app-server registro errores de `apply_patch` porque esperaba lineas antiguas
en:

- `scripts/smoke_opes_domain_work_real.sh`
- `scripts/orquesta_server_deploy.sh`

Estado: abierto como sintoma de concurrencia/write-set/contexto stale. Parte
del codigo resultante es recuperable y ya esta en el worktree remoto, pero el
backend no cerro el goal correctamente.

Relacion: `BUG-ORQ-20260710-208`.

### S9 - Prueba amplia remota falla con procesos residentes vivos

La verificacion amplia remota fallo bajo concurrencia:

- `cmd/orquesta-server`: `claude_goal_result_invalid`
- `cmd/orquesta-server`: `gemini_goal_result_invalid`
- `cmd/orquesta-server`: child de compilacion terminado en flaky harness
- `modulos/orquesta-app-codex-stack`: umbral temporal excedido en
  `TestSimulacionDeterministaFallosGoalFirstV0`

Paquetes focales si pasaron:

- `modulos/orquesta-autoprogramming`
- `modulos/orquesta-mcp`
- `modulos/orquesta-runtime-codex-appserver`
- `modulos/orquesta-app-codex-stack` focal
- `modulos/orquesta-web`

Estado: abierto. No se puede cerrar el frente remoto hasta tener drain/cleanup
gobernado o harness aislado.

Relacion: `BUG-ORQ-20260710-208E`.

### S10 - Artefactos generados quedaron en el worktree

Quedaron un checkpoint y un resultado generated bajo `scripts/`:

- `scripts/checkpoint_started_goal-ref-task-autoprogramming-f0ad7bb152dd-g04.txt`
- `scripts/docs/orquesta_goal_result_goal-ref-task-autoprogramming-f0ad7bb152dd-g04.json`

Estado: mitigado. Se movieron a backup remoto:
`/srv/orquesta-self/backups/active-goals-20260710T-doc-before-commit/generated-artifacts/`.
No deben commitearse como fuente.

### S11 - Telegram/Hermes remoto no es fiable para control operativo aun

Durante los cortes anteriores Telegram/Hermes reporto fallos de auth de Codex
(`Codex auth is missing access_token`) y el bot no daba estado util del
programador. En esta sesion se decidio aplazar Telegram/Hermes hasta cerrar el
nucleo/control de Orquesta.

Estado: aplazado, no cerrado. Debe quedar detras de goal-first/control/observe.

### S12 - Limpieza de variables/codigo muerto sigue incompleta como corte remoto

Hay avances en consolidacion de envs y scripts, pero el remoto aun necesita un
corte gobernado para migrar perfil remoto, secretos, defaults y codigo muerto.
No debe mezclarse con parada de agentes vivos ni con OPES productivo.

Estado: abierto. Alimentado por:

- `docs/auditoria_envs_pisadas_2026-07-04.md`
- `docs/auditoria_codigo_muerto_duplicado_2026-07-04.md`
- `docs/instrucciones_director_codex_2026-07-09.md`

### S13 - Artefactos de ejecucion ya versionados en el repo

Tras el push se comprobo que el repo remoto contiene 37 ficheros ya versionados
con patrones de ejecucion:

- `checkpoint_started_goal-ref-task-autoprogramming-*.txt`
- `orquesta_goal_result_goal-ref-task-autoprogramming-*.json`

Afectan a rutas como:

- `cmd/orquesta-server/docs/`
- `modulos/orquesta-web/docs/`
- `modulos/orquesta-autoprogramming/docs/`
- `modulos/orquesta-operator-telegram/docs/`
- `scripts/docs/`

Estado: abierto. No se borran en caliente porque pueden estar citados por
incidencias, commits o evidencias de agentes. Necesitan una auditoria gobernada:
clasificar si son evidencia historica valida, moverlos a una carpeta de
evidencias/retencion o retirarlos del arbol fuente con commit explicito.

Relacion: limpieza de historicos falsos y `BUG-ORQ-20260710-208E`.

## Patrones estructurales detectados

- Varias fuentes de verdad para un goal: estado persistido, checkpoint/result
  en filesystem y proceso real backend.
- Control plane local puede escribir `stop_requested` sin detener backend.
- Observacion parcial mezcla estado vivo con cierre bloqueado.
- Paralelizacion no siempre respeta write-sets declarados.
- Limpieza operativa sin guardas puede borrar rutas que siguen referenciadas.
- Verificaciones amplias no estan aisladas de procesos residentes.
- Remoto puede quedar fuera de GitHub por remoto Git configurado a bundle.
- Artefactos de ejecucion pueden acabar versionados como si fueran fuente.

## Estado de mitigaciones ya hechas

- Reinicio gobernado solo de Orquesta apuntando al worktree canonico.
- Backup de artefactos generados antes de retirarlos del worktree.
- Documentacion detallada de `BUG-ORQ-20260710-208`.
- Patch focal para workdir inexistente en app-server.
- Patch focal para rework residente ante bloqueos operativos recuperables.
- Patch focal para secuenciar write-sets declarados solapados.
- Patch focal para no publicar cierre `blocked` desde un timeout parcial con
  `goal_status=running`.
- Tests focales de esos patches en verde.

## Pendiente inmediato

1. Commit remoto de los cambios recuperables de agentes y parches de Codex.
2. Cambiar remoto Git del servidor a GitHub o anadir remote GitHub explicito.
3. Push de la rama `trabajo/plataforma-agentes`.
4. Desplegar binario Orquesta remoto con estos cambios.
5. Hacer drain/cleanup gobernado de procesos Orquesta vivos, sin tocar
   `uso-app`.
6. Repro/API de `prepare-run`, `observe` y `runs/control` con write-sets
   solapados y stop/cancel.
7. Solo despues, reactivar Telegram/Hermes y limpieza amplia de codigo muerto.
