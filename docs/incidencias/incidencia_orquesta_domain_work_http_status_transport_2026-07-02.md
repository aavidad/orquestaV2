# Incidencia: DomainWork HTTP aceptaba estados de lifecycle como job valido

Fecha: 2026-07-02.

Inventario: `BUG-ORQ-20260702-119`.

## Sintoma

El adaptador `orquesta-domain-work-http` aceptaba respuestas HTTP 2xx con
`status=running`, `status=completed` o estados equivalentes fuera del contrato
neutral siempre que incluyeran `job_ref` o `receipt_ref`.

Esto podia convertir un estado de lifecycle de la app externa en un
`DomainWorkJobV0` o `DomainWorkArtifactReceiptV0` valido para Orquesta, creando
falsos verdes de transporte. El caso estaba parcialmente oculto por el test de
retry, que devolvia `status=pending` y solo comprobaba `job_ref`.

## Causa arquitectonica

El contrato puro de DomainWork solo publica `accepted` e `invalid` para jobs y
receipts. El cliente HTTP validaba JSON, content type, retry y presencia de refs,
pero no validaba que el estado recibido perteneciera a ese contrato.

La frontera correcta es:

- `accepted`: mutacion aceptada y ref terminal obligatoria;
- `invalid`: rechazo de dominio transportado como resultado sin exigir ref
  terminal;
- `running`, `completed`, `pending` u otros estados de lifecycle: no son estados
  del puerto de mutacion DomainWork y deben exponerse por otra superficie de
  observabilidad, records o status.

## Cierre

El cliente HTTP valida ahora el status antes de devolver jobs o receipts:

- rechaza estados fuera de `accepted|invalid` con
  `domain_work_http_response_status_invalid`;
- conserva `accepted` con requisito de `job_ref` o `receipt_ref`;
- permite `invalid` como respuesta de dominio sin ref terminal;
- el test de retry usa `status=accepted` y deja de ocultar estados ajenos al
  contrato.

Pruebas:

```bash
go test -count=1 ./modulos/orquesta-domain-work-http
go test -count=1 ./modulos/orquesta-domain-work ./modulos/orquesta-domain-work-memory ./modulos/orquesta-domain-work-file ./modulos/orquesta-domain-work-sql ./modulos/orquesta-domain-work-http
```

## Residual

Si una composicion necesita observar `running`, `completed`, timestamps,
heartbeat, artefactos o resume de la app externa, debe hacerlo por una fuente de
records/status separada. El puerto HTTP de mutacion no debe inferir cierre de
trabajo a partir de esos estados.
