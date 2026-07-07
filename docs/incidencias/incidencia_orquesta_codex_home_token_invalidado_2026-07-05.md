# Incidencia: CODEX_HOME de Orquesta con token invalidado - 2026-07-05

Actualizado: 2026-07-05T21:15:15Z

## Resumen

El runtime Codex usado por Orquesta en remoto (`/srv/orquesta-self/codex-home`) figura como `Logged in using ChatGPT`, pero al arrancar un agente real falla con `401 Unauthorized`, `token_invalidated` y `refresh_token_invalidated`.

## Evidencia observada

- Agente afectado: `agent-ref-task-autoprogramming-c20a585ff5c3-g01`.
- Run/request: `request-ref-remoto-telegram-nollm-runtime-20260705-001`.
- Log: `/srv/orquesta-self/claude-director-20260705/runtime/request-ref-remoto-telegram-nollm-runtime-20260705-001/agent-ref-task-autoprogramming-c20a585ff5c3-g01/codex_stderr.log`.
- El agente arranco con `workdir=/srv/orquesta-self/worktrees/pilot-remoto-1`, modelo `gpt-5.5`, sandbox `workspace-write`, pero fallo antes de programar.
- Mensajes clave del log: `token_invalidated`, `refresh_token_invalidated`, y peticion de iniciar sesion de nuevo.
- `codex login status` con `CODEX_HOME=/srv/orquesta-self/codex-home` no detecta el problema porque solo informa estado local, no valida refresh real contra proveedor.

## Impacto

- Orquesta puede aceptar prepare-runs, pero no puede ejecutar nuevos agentes Codex mientras este home siga invalidado.
- Los resultados previos con tests siguen siendo evidencia historica, pero no prueban que el servidor este programando ahora.
- Telegram salida por Hermes no queda afectada; la parte rota es el proveedor de agentes de Orquesta.

## Accion requerida

Reautenticar en remoto el Codex CLI usando el mismo `CODEX_HOME` del servidor Orquesta y relanzar/reconciliar el request no-LLM. No registrar tokens en docs ni logs.

Comando orientativo para operador humano:

```bash
ssh -t berserk@uso.dipgra.cloud 'sudo env CODEX_HOME=/srv/orquesta-self/codex-home /usr/local/bin/codex login --device-auth'
```

Si el login requiere navegador/codigo, completarlo manualmente y despues lanzar una prueba minima o relanzar el request por Orquesta.


## Actualizacion 2026-07-05T21:19:11Z: pruebas de homes alternativos

Se probaron ejecuciones minimas `codex exec` sin leer ni copiar secretos:

- `/srv/orquesta-self/codex-home`: `401 token_invalidated` y `refresh_token_invalidated`.
- `/srv/orquesta-self/runtime/goal-srv/codex-home`: `401 token_invalidated` y `refresh_token_invalidated`.
- `/root/.codex`: `refresh_token_reused` / `token_expired`.
- `/home/dipuso/.codex`: `refresh_token_reused` / `token_expired`.

Conclusion: el bloqueo de agentes Codex no es cuota (`429`) sino autenticacion (`401`). Orquesta no puede lanzar programacion con Codex hasta reautenticar o configurar otro proveedor valido.

## Estado

Abierta hasta que un agente real escriba `agent_ack.json` y complete una tarea nueva despues de la reautenticacion.

## Actualizacion 2026-07-07: diagnostico de runtime endurecido

Durante esta sesion, dos subagentes locales fallaron con el mensaje publico
`access token could not be refreshed`, la misma clase operativa que los
`token_invalidated`/`refresh_token_invalidated` del servidor remoto. No se
leyeron ni copiaron secretos.

Cierre de codigo local:

- `orquesta-runtime-codex-appserver` clasifica ahora
  `token_invalidated`, `refresh_token_invalidated`, `refresh_token_reused`,
  `token_expired` y `access token could not be refreshed` como
  `codex_app_server_provider_unauthorized`.
- Ese issue code ya tiene evidencia y mensaje de accion en la composicion:
  revisar/reautenticar Codex/OpenAI en el `CODEX_HOME` aislado y reiniciar.

Prueba:

- `go test -count=1 ./modulos/orquesta-runtime-codex-appserver`

El estado operativo no cambia: sigue abierta hasta reautenticar en remoto y
verificar un agente real nuevo con ACK/resultado.
