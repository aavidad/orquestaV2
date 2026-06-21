# Incidencia OPES DomainWork: reintento de submit transitorio

Fecha: 2026-06-20

## Contexto

En la prueba real de `plan_temario` para `oficial-de-servicios-multiples`, el
agente de Orquesta entregó un `document_plan` válido con 17 secciones, 17
visuales y 12 pasos de revisión. OPES recibió el artefacto y llegó a crear 10
trabajos derivados, pero el envío quedó registrado en Orquesta como rechazado
con `domain_work_port_no_disponible`.

El artefacto no era inválido: al reintentar el mismo `submit_artifact` con la
misma idempotencia, OPES completó el job padre y materializó los 46 trabajos
derivados esperados.

## Causa

Orquesta trataba casi todos los registros `rejected` del ledger de
`submit_artifact` como definitivos, salvo `domain-work-submit-execute-error`.
Si el submit fallaba por timeout, puerto/transporte o saturación temporal,
el ledger impedía que una ejecución posterior reintentara la entrega
idempotente.

Además, el executor MCP de DomainWork convertía errores públicos del conector
en `domain_work_port_no_disponible`, perdiendo códigos más concretos como
`opes_http_timeout`.

## Corrección

- `domainWorkSubmissionAlreadyRecordedV0` distingue rechazos funcionales de
  rechazos transitorios.
- Se reintentan rechazos con:
  - `domain-work-submit-execute-error`
  - `domain_work_port_no_disponible`
  - `opes_http_request_failed`
  - `opes_http_timeout`
  - `opes_http_cancelled`
  - `retry_budget_exhausted`
  - `opes_http_status_429`
  - `opes_http_status_502`
  - `opes_http_status_503`
  - `opes_http_status_504`
- Los conectores pueden exponer `PublicCodeV0()` y el executor MCP conserva ese
  código público en el error.

## Validación

Pruebas ejecutadas:

```bash
go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestCodexStackV0DomainWorkSubmitAcceptedYRejectedFuncionalBloqueanReintento|TestCodexStackV0DomainWorkSubmitRegistraRejectedSiExecutorFallaYReintenta'
go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPDomainWorkExecutorV0Propaga'
go test -count=1 ./modulos/orquesta-opes-connector ./modulos/orquesta-domain-work-http
```

Comprobación OPES real:

- job padre `01421971819c7808c847386a3338b8f8`: `completed`
- trabajos derivados del plan: 46
- padres de tema materializados: 17

