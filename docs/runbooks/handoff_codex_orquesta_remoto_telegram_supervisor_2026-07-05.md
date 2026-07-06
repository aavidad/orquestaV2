# Handoff Codex remoto: Telegram, supervisor y cierre operativo - 2026-07-05

Actualizado: 2026-07-05T21:07:03Z

## Objetivo operativo

Alberto pide terminar Orquesta al 100%. En esta tanda el foco real fue comprobar si Orquesta programa de forma fiable y cerrar el frente de control movil por Telegram sin depender de un agente LLM externo.

## Regla de trabajo aplicada

- No se tocaron repos locales; el trabajo se hizo sobre el servidor `uso.dipgra.cloud`.
- Se uso Orquesta como superficie principal para lanzar tareas cuando era viable.
- Codex directo se uso para observacion, diagnostico, documentacion y verificacion acotada, porque Orquesta presento falsos cierres/proyeccion incompleta.
- No se usaron secretos en docs ni en salidas; Telegram/Hermes se trataron como adaptadores, no como nucleo.

## Acciones realizadas y motivo

1. Verificacion Hermes/Telegram remoto.
   - Motivo: Alberto necesitaba manejar Orquesta desde el movil y recibir avisos.
   - Resultado: Hermes remoto quedo autenticado (`openai-codex: logged in`), gateway activo y `hermes send` funciona contra Telegram.
   - Evidencia: `hermes send --to telegram` devolvio envio correcto al chat `39995054`.

2. Pausa de cron jobs Hermes que generaban spam.
   - Motivo: tres jobs fallaban cada 15 minutos por `Codex auth is missing access_token` y enviaban errores a Telegram.
   - Jobs pausados: `09b3751789e7`, `5052ee9bbd70`, `9def5e432f3c`.
   - Resultado: `hermes cron status` quedo en `No active jobs`.

3. Lanzamiento de goals Orquesta para canal operador-Director y Telegram.
   - Motivo: el producto debe tener comunicacion directa con el Director, no depender de Hermes como nucleo.
   - Resultados durables encontrados:
     - `goal-ref-task-autoprogramming-afceb345c0cf-g01`: canal operador-Director, estado `complete`.
     - `goal-ref-task-autoprogramming-afceb345c0cf-g01-rework-1`: rework canal operador-Director, estado `complete`.
     - `goal-ref-task-autoprogramming-8f064b280aca-g01`: Telegram/notificaciones, estado `complete`.

4. Verificacion focal de codigo generado.
   - Motivo: no aceptar `complete` documental sin compilar/probar.
   - Comando ejecutado con caches bajo `.orquesta-runtime`, no `/tmp`:
     `go test -count=1 ./modulos/orquesta-operator-telegram ./modulos/orquesta-operator-director-channel ./modulos/orquesta-operator-notifications ./modulos/orquesta-mcp ./cmd/orquesta-server ./modulos/orquesta-app-codex-stack -run 'TestTelegramOperator|TestOperatorDirector|TestOperatorNotification|TestMCPOperator|TestBuildStackOperatorDirector'`
   - Resultado: 6/6 paquetes OK.
   - Build: `/srv/orquesta-self/runtime/builds/orquesta-server-telegram-verify-20260705`, 36 MB, OK.

5. Deteccion de brecha de despliegue.
   - Motivo: compilar no prueba runtime vivo.
   - Resultado: el servidor vivo sigue ejecutando `/srv/orquesta-self/runtime/orquesta-server-claude run`, binario de 22 MB, arrancado antes del build nuevo.
   - Impacto: las piezas nuevas no estan probadas como servidor vivo.

6. Deteccion de fallo de control movil por Hermes LLM.
   - Motivo: el usuario pregunta si puede manejar la app desde movil.
   - Resultado: Telegram/Hermes puede enviar, pero el manejo conversacional por Hermes falla por `HTTP 429 usage_limit_reached` del proveedor OpenAI Codex.
   - Conclusion: el control movil debe pasar a runtime no-LLM propio de Orquesta.

7. Lanzamiento de rework Orquesta para runtime Telegram no-LLM.
   - Request: `request-ref-remoto-telegram-nollm-runtime-20260705-001`.
   - Motivo: implementar endpoint/poller/worker no-LLM para comandos Telegram y notificaciones terminales.
   - Resultado observado: `prepare-run` devolvio `accepted=true`, pero no aparece en `autoprogramming/status`; no hay resultado durable visible.
   - Evidencia runtime: existe `/srv/orquesta-self/claude-director-20260705/runtime/request-ref-remoto-telegram-nollm-runtime-20260705-001`, pero sin artefactos visibles para el usuario normal.

8. Deteccion de fallo estructural del supervisor.
   - Motivo: entender por que el rework aceptado no avanza.
   - `/api/status` muestra:
     - `last_supervisor_status=error`
     - `last_supervisor_stop=tick_error`
     - `supervisor_error_ticks=4394`
     - error: `director_tick_input_build_invalido: field=scheduler_input.payload`
   - Focales de `orquesta-director-tick-input`, `orquesta-director-cycle` y `cmd/orquesta-server` pasan en el worktree actual, lo que apunta a binario vivo antiguo o datos reales no cubiertos por tests.

9. Documentacion de incidencia.
   - Incidencia creada: `docs/incidencias/incidencia_orquesta_telegram_nollm_accepted_invisible_2026-07-05.md`.
   - Inventario actualizado: `docs/inventario_bugs_orquesta_2026-06-30.md` con `BUG-ORQ-20260705-TELEGRAM-NOLLM-ACCEPTED-INVISIBLE`.

## Estado real resumido

- Orquesta programa codigo util: si.
- Focales recientes: verdes.
- Build reciente: verde.
- Runtime vivo con esos cambios: no verificado/no desplegado.
- Supervisor autonomo: no sano; hay error repetido de tick input.
- Cola: hay runs `ready` y diagnostico de supervision segura.
- Telegram salida: funciona.
- Telegram control por Hermes LLM: bloqueado por 429.
- Telegram control no-LLM propio de Orquesta: requerido, aun no cerrado.


## Actualizacion 2026-07-05T21:11:04Z: fallo real del agente no-LLM

Se reviso con permisos de servidor el log del agente lanzado por Orquesta para `request-ref-remoto-telegram-nollm-runtime-20260705-001`.

Resultado:
- El agente llego a arrancar con workdir `/srv/orquesta-self/worktrees/pilot-remoto-1` y sandbox `workspace-write`.
- Fallo antes de programar por autenticacion del Codex CLI usado por Orquesta:
  - `token_invalidated`
  - `refresh_token_invalidated`
  - mensaje operativo: hay que cerrar sesion/iniciar sesion de nuevo.
- Esto afecta al `CODEX_HOME` de Orquesta (`/srv/orquesta-self/codex-home`), no al envio Telegram de Hermes.

Por que se documenta: separa el problema en dos capas. La capa Telegram salida funciona; la capa Orquesta-agentes no puede ejecutar nuevas tareas hasta reautenticar el runtime Codex que usa el servidor. Cualquier `complete` nuevo que dependa de agentes Codex debe tratarse como no posible hasta corregir esta frontera externa.

Accion requerida antes de relanzar tareas: reautenticar Codex CLI en remoto para el home de Orquesta y relanzar/reconciliar `request-ref-remoto-telegram-nollm-runtime-20260705-001`.


## Actualizacion 2026-07-05T21:15:15Z: supervisor y auth separados como bugs

Se dejaron dos incidencias separadas para evitar falsos diagnosticos:

- `docs/incidencias/incidencia_orquesta_supervisor_scheduler_payload_2026-07-05.md`: el supervisor no progresa porque el primer run ranked (`request-ref-autoprogramming-backlog-t137-public-http-request-body-bounds-248bf945-retry-f7319198e315`) falla con `scheduler_input.payload`. Motivo de la separacion: es un bug de cola/tick-input/compactacion, no de Telegram.
- `docs/incidencias/incidencia_orquesta_codex_home_token_invalidado_2026-07-05.md`: los agentes Codex de Orquesta no arrancan por token/refresh invalidado en `/srv/orquesta-self/codex-home`. Motivo de la separacion: es frontera externa de proveedor/auth y bloquea cualquier nueva programacion con Codex.

Ambas quedaron enlazadas en `docs/inventario_bugs_orquesta_2026-06-30.md`.


## Actualizacion 2026-07-05T21:16:11Z: subagente de diagnostico cerrado

Se habia lanzado un subagente read-only para investigar el error `scheduler_input.payload`, sin tocar archivos ni usar codebase-memory. No devolvio resultado antes de varios waits y se cerro manualmente para no dejar trabajo colgado. La evidencia util de esta seccion procede de inspeccion directa del remoto: `/api/status`, `orquesta_server_audit_v0.jsonl`, cola persistida y lectura acotada de `orquesta-director-cycle`, `orquesta-director-tick-input` y `orquesta-director-scheduler`.


## Actualizacion 2026-07-05T21:19:11Z: pruebas de homes alternativos

Se probaron ejecuciones minimas `codex exec` sin leer ni copiar secretos:

- `/srv/orquesta-self/codex-home`: `401 token_invalidated` y `refresh_token_invalidated`.
- `/srv/orquesta-self/runtime/goal-srv/codex-home`: `401 token_invalidated` y `refresh_token_invalidated`.
- `/root/.codex`: `refresh_token_reused` / `token_expired`.
- `/home/dipuso/.codex`: `refresh_token_reused` / `token_expired`.

Conclusion: el bloqueo de agentes Codex no es cuota (`429`) sino autenticacion (`401`). Orquesta no puede lanzar programacion con Codex hasta reautenticar o configurar otro proveedor valido.


## Actualizacion 2026-07-05T21:25:39Z: avance manual pequeno por falta de cuota/auth

Alberto autorizo parar Orquesta y programar manualmente hasta recuperar cuota/auth. Se paro solo el proceso Orquesta del puerto `19071`; el servidor Linux y Hermes quedaron vivos.

Cambio pequeno cerrado:
- Area: `modulos/orquesta-director-tick-input`.
- Motivo: el supervisor se atascaba por `scheduler_input.payload` cuando el carril `progress` conservaba historial largo de `AgentWorkAssessed` y `DirectorQuestionRaised`.
- Implementacion: `compactTickInputProgressLaneV0` ahora compacta el snapshot de progreso y conserva solo refs causales del candidato actual: agente, tarea, entrega, assessment y pregunta.
- Test nuevo: `TestBuildDirectorSchedulerTickInputV0CompactaCarrilProgressConHistorialLargo`.

Verificacion:
- `go test -count=1 ./modulos/orquesta-director-tick-input -run TestBuildDirectorSchedulerTickInputV0CompactaCarrilProgressConHistorialLargo` OK.
- `git diff --check` OK.

Limitacion: no se relanzo Orquesta ni se verifico el supervisor vivo porque el runtime Codex remoto sigue sin auth valida y el usuario pidio subir avances pequenos sin quedar a medias.

## Pendientes prioritarios para Claude/Codex

1. Corregir el fallo `director_tick_input_build_invalido: field=scheduler_input.payload` en datos reales del supervisor.
2. Reconciliar `request-ref-remoto-telegram-nollm-runtime-20260705-001`: accepted no puede quedar invisible.
3. Implementar o completar runtime Telegram no-LLM en Orquesta: endpoint/poller opt-in, autorizacion de chat, comandos contra canal operador-Director, respuestas por Bot API/Hermes-send sin LLM.
   - Avance Codex 2026-07-06: existe `POST /api/v0/operator/telegram/update`
     en `cmd/orquesta-server`, montado solo con `telegram_operator.enabled`.
     Acepta update Telegram real o payload compacto, valida chat autorizado,
     responde JSON y puede enviar por puerto `hermesTelegramSendPortV0`; `/msg`
     usa `OperatorDirectorMessage`. Pendiente: despliegue remoto y validacion
     Telegram real/poller/Bot API.
4. Desplegar de forma controlada el build nuevo solo cuando el servidor pueda apagarse/reiniciarse con evidencia de `shutdown_ready` o procedimiento documentado.
5. Revalidar `/api/status`, `/api/v0/autoprogramming/status`, MCP `orquesta.operator.director.message.v0` y envio Telegram tras despliegue.
6. Mantener los cron jobs Hermes LLM pausados hasta que haya cuota o proveedor alternativo; no usarlos como control plane principal.

## Evidencias clave

- Build verificado: `/srv/orquesta-self/runtime/builds/orquesta-server-telegram-verify-20260705`.
- Respuesta prepare-run no-LLM: `/srv/orquesta-self/operator-requests/20260705/prepare_run_telegram_nollm_runtime_20260705.response.json`.
- Incidencia accepted invisible: `docs/incidencias/incidencia_orquesta_telegram_nollm_accepted_invisible_2026-07-05.md`.
- Runbook Telegram: `docs/runbooks/telegram_operador_orquesta_remoto_2026-07-05.md`.
