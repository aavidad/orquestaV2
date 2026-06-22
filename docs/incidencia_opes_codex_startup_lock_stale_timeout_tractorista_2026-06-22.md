# Incidencia OPES Codex Startup Lock Stale Timeout Tractorista

Fecha: 2026-06-22.

## Contexto

Run OPES:
`run-opes-tractorista-rework-tests-004-006-008-20260622`.

Tras reconciliar `g02`, Orquesta genero un agente sustituto:
`agent-ref-assessment-task-autoprogramming-53c892417bba-g02-2737767eec2568dad04a391c89cc71a9`.

## Sintoma

El wrapper `orquesta_codex_exec_v0.sh` quedo esperando el lock global:
`/home/alberto/.codex/.orquesta-codex-startup.lock`.

El directorio de lock estaba vacio y era anterior al agente sustituto, pero el
script usaba:

- `ORQUESTA_CODEX_STARTUP_LOCK_TIMEOUT_SECONDS` por defecto: `120`;
- `ORQUESTA_CODEX_STARTUP_LOCK_STALE_SECONDS` por defecto: `900`.

Resultado: un lock obsoleto podia provocar timeout antes de que el propio
reaper lo considerara stale.

## Impacto

- Un agente de replan/sustitucion puede fallar sin ejecutar `codex exec`.
- La cola avanza a nuevos reintentos o estados de atencion aunque el problema
  sea un lock local recuperable.
- Reduce la autonomia de OPES porque exige intervencion manual para limpiar un
  lock que Orquesta ya sabe identificar.

## Arreglo Esperado

El stale efectivo del lock de arranque no debe superar el timeout efectivo del
wrapper. Si `ORQUESTA_CODEX_STARTUP_LOCK_STALE_SECONDS` no esta configurado o
es invalido, se toma el timeout como maximo. Si se configura por encima del
timeout, se acota al timeout.

## Arreglo Aplicado

`modulos/orquesta-runtime-codex/codex_wrapper_v0.go` calcula
`orquesta_codex_lock_stale_seconds_v0` a partir de
`orquesta_codex_lock_timeout_v0` y lo limita al timeout.

Validacion focal:

```bash
go test -count=1 ./modulos/orquesta-runtime-codex -run 'TestCodexWrapperV0(RetiraStartupLockObsoletoVacio|StaleLockPorDefectoNoSuperaTimeout)'
```

## Criterio De Cierre

Un wrapper con lock vacio antiguo, `STALE_SECONDS` sin configurar y timeout bajo
debe limpiar el lock y arrancar el comando Codex fake sin emitir
`orquesta_codex_startup_lock_timeout`.
