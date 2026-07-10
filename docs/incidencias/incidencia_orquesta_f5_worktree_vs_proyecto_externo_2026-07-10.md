# Incidencia F5: worktree de identidad confundido con proyecto externo

Fecha: 2026-07-10  
Estado: abierta, correccion local en curso  
ID: `BUG-ORQ-20260710-208L`

## Evidencia

El intento D1 `smoke_goal_first_app_server_real.sh` con raiz retenida
`/tmp/orquesta-goal-first-app-server.IOpIBs` compilo el servidor y preparo un
proyecto externo temporal, pero no genero estado ni readiness. El proyecto
objetivo se exporta como `ORQUESTA_CODEX_PROJECT_WORKDIR=$project_dir` y no es
un repositorio Git de Orquesta por diseno.

F5 validaba ese mismo valor en
`serverIdentityPreflightFromEnvV0` mediante
`validateServerWorktreeIdentityWithRuntimeV0`. El servidor entraba en
`degraded_identity` antes de crear el estado que espera el smoke.

## Causa estructural

Se mezclaron dos responsabilidades de composicion:

1. El worktree Git canonico de Orquesta, que prueba remoto/ref/commit del
   binario vivo.
2. El directorio de una app externa sobre la que trabaja el Goal, que puede ser
   temporal y no Git.

Esto contradice la frontera generica de Orquesta: una app externa no debe
necesitar ser un checkout interno de Orquesta para recibir trabajo.

## Correccion y cierre

La correccion introduce `ORQUESTA_SERVER_WORKTREE` como identidad explicita
del servidor, conserva el proyecto objetivo en
`ORQUESTA_CODEX_PROJECT_WORKDIR` y mantiene fallback compatible solo para
composiciones antiguas. El ctl y el smoke deben exportar la nueva variable.

No cerrar hasta que el D1 local complete readiness, prepare/observe/cierre y
shutdown limpio con esta separacion. Si reaparece `backend_still_running`,
abrir incidencia separada: no ocultarla bajo 208L.
