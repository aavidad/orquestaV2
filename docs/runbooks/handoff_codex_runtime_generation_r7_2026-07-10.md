# Handoff Codex runtime generation R7 — 2026-07-10

## Alcance

Reparación mecánica y acotada del fixture `TestSocketOwnerV0AceptaDescendienteDeWrapperV0`. Se conservan R4-R6; no se cambia producción, no se usa `uso-app`, ni hubo commit, push, deploy o procesos productivos.

## Causa y cambio

El wrapper hereda el FD y hace visible el listener antes de crear y escribir `owner.pid`; esperar únicamente `codexAppServerTmuxSocketListenerAliveV0` dejaba una ventana determinista de `ENOENT`. El test ahora usa el mismo timeout acotado para esperar el handshake: `owner.pid` debe existir, contener un PID completo y parseable, y ese PID debe estar vivo (`kill(pid, 0)`, aceptando `EPERM` como proceso existente). Solo después resuelve y compara el propietario causal. El timeout sigue siendo fallo fatal y se mantienen cleanup y aserciones de seguridad; no se añade `Skip` ni se oculta ningún error.

## Verificación

Se ejecutará la prueba focal en el sandbox si el listener Unix/procfs lo permite. El operador repetirá paquete, `-race` y `-count=40` fuera del sandbox.

estado_final: verification_blocked_by_sandbox_r7
