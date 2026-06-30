# Analisis DomainWork: heartbeat live

Solicitud: `req-orquesta-parallel-domain-work-20260630T143203Z`.
Write-set: `docs/orquesta_parallel_smoke_2026-06-30/domain_work.md`.

## Conclusion

El contrato neutral de `DomainWork` no deberia incorporar un heartbeat live de
ejecucion como parte de `DomainWorkJobRequestV0`,
`DomainWorkArtifactSubmissionV0` ni del store generico de jobs. Ese pulso vivo
debe quedar en el adaptador/runtime que observa el proveedor, proceso, cola o
app propietaria.

La excepcion ya cubierta por el contrato neutral es declarativa, no ejecutiva:
para trabajos que dependen de capacidades externas, como `audio_asset`, el
contrato puede exigir preflight de capacidad con `progress_heartbeat_required`,
`provider_timeout_required` y `provider_no_progress_timeout_seconds`. Esa
exigencia sirve para decidir si la composicion puede aceptar el trabajo; no
convierte a `orquesta-domain-work` en observador de liveness.

## Evidencia revisada

- `modulos/orquesta-domain-work/README.md` y `docs/contratos.md`: el modulo es
  puro, solo DTOs/puertos, sin DB, filesystem, runtime, proveedor, red ni
  conectores reales.
- `modulos/orquesta-domain-work/types_v0.go`: el contrato base modela request,
  job aceptado/invalido, record, submit de artefacto, receipt y tests por refs;
  no hay estado `running` ni puerto de heartbeat.
- `modulos/orquesta-domain-work/external_capability_v0.go`: el perfil
  `speech_synthesis` exige readiness de heartbeat/progreso y timeout, pero como
  campos de capacidad declarada/evaluada por composicion.
- `modulos/orquesta-runtime/docs/contratos.md`: `AgentProgressHeartbeatV0`
  pertenece a `orquesta-runtime`, con counters compactos y refs opacas para
  director/scheduler.
- `modulos/orquesta-external-work-run/docs/contratos.md`: `BuildExternalWorkGoalWorkSpecV0`
  compila DomainWork a GoalWorkSpec y deja lanzamiento/observacion del Goal a
  composicion opt-in.

## Recomendacion operativa

Mantener `DomainWork` con precondiciones declarativas de capacidad externa y
receipts/artefactos de dominio. Si un conector necesita detectar cuelgues vivos,
debe publicar una senal compacta desde el adaptador propietario o desde
`orquesta-runtime` y proyectarla a evidencias/refs; el contrato neutral solo
debe consumir esa evidencia para bloquear, reintentar o rework cuando falte
progreso observable.

No se uso `codebase-memory-mcp`, no se tocaron OPES productivo ni temarios, no
se arrancaron servidores ni procesos largos.
