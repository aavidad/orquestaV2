# Incidencia: agente remoto reabre startup-lock timeout-only

Fecha: 2026-07-01

## Resumen

En el worktree remoto aislado `/srv/orquesta-self/runtime/audit-225a6752-next`,
el agente Codex/app-server genero un diff no commiteado que vuelve a cambiar la
politica de `codexSharedStartupLockShellV0`: `ORQUESTA_CODEX_STARTUP_LOCK_TIMEOUT_SECONDS`
por si sola dejaria de contar como configuracion explicita cuando el default es
cero.

Ese comportamiento contradice el cierre local `b9c5901b`, donde cualquier
variable `ORQUESTA_CODEX_STARTUP_LOCK_*` declarada por el operador debe contar
como explicita para no desactivar silenciosamente el lock.

## Evidencia

- Fichero remoto modificado: `modulos/orquesta-runtime-codex/codex_wrapper_v0.go`.
- Test remoto modificado en sentido contrario:
  `TestCodexWrapperV0TimeoutSoloNoActivaStartupLockSiDefaultEsCeroV0`.
- El diff remoto cambia una prueba que antes exigia timeout explicito aunque no
  estuviese `STALE`, y pasa a aceptar que timeout aislado no active lock.

## Clasificacion

- Area: runtime Codex / startup-lock.
- Estado: rechazado para integracion.
- Hipotesis arquitectonica: falta una guardia de contrato estable que impida a
  agentes remotos "arreglar" un fallo reescribiendo la expectativa ya cerrada.
  El problema no es solo una linea de shell; es una ruptura de autoridad entre
  evidencia local aceptada y autoprogramacion remota.

## Accion

No integrar ese bloque remoto. Si se conserva algo del trabajo remoto, apartar o
revertir los cambios de `codex_wrapper_v0.go` y `codex_wrapper_v0_test.go` antes
de reiniciar el servidor remoto con una version nueva.

La mejora pendiente es reforzar tests/contrato para que los agentes no puedan
renombrar la prueba y cambiar el criterio sin dejar evidencia explicita de
decision.
