# Incidencia OPES modo automatico desactivado - 2026-06-13

## Contexto

Durante la tanda `a2_informatica_2026-06-13`, el servidor Orquesta aislado en
`127.0.0.1:8792` estaba arrancado con contexto OPES:

```text
ORQUESTA_CODEX_PROJECT_WORKDIR=/home/alberto/Trabajo/OPES
ORQUESTA_OPES_PROJECT_WORKDIR=/home/alberto/Trabajo/OPES
ORQUESTA_CODEX_RUNTIME_WORKDIR=/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/a2_informatica_2026-06-13/runtime
ORQUESTA_SERVER_STATE_DIR=/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/a2_informatica_2026-06-13/state
```

pero tenia:

```text
ORQUESTA_SERVER_RESIDENT_DIRECTOR_ENABLED=false
```

El endpoint publico confirmaba `ORQUESTA_SERVER_RESIDENT_DIRECTOR_ENABLED=false`
en `effective_config`. En otro servidor vivo (`127.0.0.1:8787`) el estado
publico conservaba `resident_director_status=ok` de un tick historico mientras
la configuracion efectiva vigente tambien indicaba `false`.

## Impacto

- OPES podia dejar trabajos aceptados, rework o app-changes pendientes hasta una
  supervision manual.
- El operador podia leer un `resident_director_status=ok` historico como si el
  modo automatico siguiera activo.
- `orquesta-server start` podia arrancar el daemon sin proyectar explicitamente
  el modo autonomo efectivo al proceso hijo.

## Causas

1. El contexto OPES no activaba por si mismo el Director residente.
2. `ORQUESTA_SERVER_RESIDENT_DIRECTOR_ENABLED=false` era aceptado incluso con
   contexto OPES activo.
3. El entorno filtrado de `start` no derivaba
   `ORQUESTA_SERVER_AUTONOMY_ENABLED` ni
   `ORQUESTA_SERVER_RESIDENT_DIRECTOR_ENABLED` desde la configuracion efectiva.
4. El estado publico reutilizaba campos historicos `resident_director_*` aunque
   el loop residente estuviera desactivado en `effective_config`.
5. `AppChangeStore` persistia solicitudes, pero no despertaba al supervisor ni
   al Director residente; la materializacion podia depender de
   `/api/v0/runs/supervise`.

## Correccion aplicada

- El contexto OPES activa autonomia efectiva si no hay override especifico.
- Con OPES activo, `ORQUESTA_SERVER_RESIDENT_DIRECTOR_ENABLED=false` bloquea el
  arranque con error accionable.
- `start` proyecta al daemon:
  - `ORQUESTA_SERVER_AUTONOMY_ENABLED`;
  - `ORQUESTA_SERVER_RESIDENT_DIRECTOR_ENABLED`;
  - `ORQUESTA_SERVER_RESIDENT_DIRECTOR_MAX_ACTIONS`.
- El status publico muestra `resident_director_status=disabled` si la
  configuracion efectiva desactiva el Director residente.
- `AppChangeStore` queda decorado en el composition root del servidor para
  despertar supervisor y Director residente con causa `app_change_saved`.

## Pruebas focales

```bash
go test -count=1 ./cmd/orquesta-server -run 'TestServerConfigFromEnvV0(ContextoOPES|BloqueaOPES|PerfilAutonomia|DirectorResidenteExplicito|PublicaConfiguracionEfectivaCanonica)|TestServerDaemonStartEnvironmentV0(ProyectaDirector|UsaAllowlist)|TestServerWakeupAppChangeStoreV0|TestServerSupervisorWakeupDecoratorsV0Disparan'
go test -count=1 ./modulos/orquesta-server -run 'TestResidentDirectorV0(StatusPublico|Visible)|TestRuntimeV0RestauraEstadoDurable'
```

Resultado local: pasadas.

## Tarea separada

Queda pendiente el Director OPES causal completo: productor opt-in que convierta
`document_plan`, `director_review_matrix`, `completed_syllabus_package` con
`pendiente_continuar`, receipts rechazados y `followup_refs` en nuevos
`DomainWorkJobRequestV0` deduplicados. Esta pieza debe vivir como adaptador OPES
en Orquesta, apoyada en puertos `domain-work`, no dentro del nucleo puro.
