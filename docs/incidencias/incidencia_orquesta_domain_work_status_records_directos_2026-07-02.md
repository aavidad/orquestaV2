# Incidencia: DomainWork status no observaba records directos

Fecha: 2026-07-02.

Relacionado con:
`external/opes/INCIDENCIA_OPES_ORQUESTA_API_ESTADO_DOMAIN_WORK_NO_OBSERVABLE_2026-06-30.md`.

## Sintoma

`GET /api/v0/domain-work/status` exponia una fachada estable, pero su fuente era
la proyeccion de cola/autoprogramacion. Si un adaptador `DomainWork` aceptaba un
job y este no aparecia aun en cola, el estado publico podia quedar `idle` aunque
existiera un `DomainWorkJobRecord` real.

## Causa arquitectonica

El estado observable de dominio estaba repartido entre cola, autoprogramacion y
store de jobs. El endpoint solo consultaba la primera proyeccion y no fusionaba
la fuente causal directa del conector (`DomainWorkJobRecordSourcePortV0`).

## Cierre

El handler de `domain-work/status` acepta ahora una fuente opcional de records.
Cuando el stack tiene un executor `DomainWork` que soporta
`ListDomainWorkJobRecordsV0`, el gateway lo conecta y fusiona esos records con
la cola sin duplicar items por `job_ref`, `run_ref` o `app_ref`.

Un `DomainWorkJobRecord` con `status=accepted` se proyecta como `queued`, no como
`completed`, porque aceptar el job no demuestra entrega ni cierre.

Pruebas:

```bash
go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./modulos/orquesta-app-codex-stack
```

## Residual

La vista aun no es una unica fuente operacional completa por `run_ref`: faltan
timestamps, ledger de artefactos, heartbeat de proveedor, stale real por edad y
endpoint de reanudacion `resume-pending`. Este cierre reduce falsos `idle`, pero
no cierra por completo la arquitectura de observabilidad DomainWork/TTS.
