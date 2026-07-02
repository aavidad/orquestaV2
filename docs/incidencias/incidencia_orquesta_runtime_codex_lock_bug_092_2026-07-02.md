# Incidencia: automejora remota reabre contrato startup-lock Codex

Fecha: 2026-07-02

## Resumen

BUG-ORQ-20260701-092 queda cerrado con contrato y pruebas focales en
`modulos/orquesta-runtime-codex`: cualquier variable
`ORQUESTA_CODEX_STARTUP_LOCK_*` presente cuenta como configuracion explicita del
startup-lock aunque el default calculado del perfil sea `0`.

El fallo observado no era un caso aislado del shell: la automejora remota
propuso parches contradictorios que estrechaban el contrato de tres formas
distintas (`default=0` desactiva todo, solo `SECONDS` cuenta, o solo
`TIMEOUT+STALE` juntos cuentan). El contrato queda documentado para que esa
familia de cambios no vuelva a entrar sin tests.

## Contrato cerrado

- `ORQUESTA_CODEX_STARTUP_LOCK_SECONDS` por si sola activa el lock.
- `ORQUESTA_CODEX_STARTUP_LOCK_TIMEOUT_SECONDS` por si sola activa el lock y
  puede terminar con `orquesta_codex_startup_lock_timeout`.
- `ORQUESTA_CODEX_STARTUP_LOCK_STALE_SECONDS` por si sola activa el lock y
  puede reaper un lock obsoleto.
- Una variable presente con valor vacio sigue contando como explicita; el valor
  vacio cae al default efectivo, pero no apaga el lock.
- `TIMEOUT+STALE` juntos siguen cubiertos, pero no sustituyen las garantias de
  `TIMEOUT` solo y `STALE` solo.

## Evidencia

- Codigo: `modulos/orquesta-runtime-codex/codex_wrapper_v0.go`.
- Contrato: `modulos/orquesta-runtime-codex/docs/contratos.md`.
- Tests:
  `TestCodexWrapperV0RespetaCadaVariableStartupLockExplicitaAunqueDefaultSeaCeroV0`,
  `TestCodexWrapperV0TimeoutSoloYTimeoutStaleExplicitosNoSeSustituyenV0`,
  `TestCodexWrapperV0RespetaStartupLockTimeoutSoloAunqueDefaultSeaCeroV0`,
  `TestCodexWrapperV0RespetaStartupLockTimeoutExplicitoAunqueDefaultSeaCeroV0`.

## Regla para automejora

No integrar cambios de automejora remota sobre `codexSharedStartupLockShellV0`,
`codexStartupLockDefaultSecondsV0` o tests asociados si no ejecutan y conservan
verde la bateria focal documentada en
`modulos/orquesta-runtime-codex/docs/pruebas.md`.
