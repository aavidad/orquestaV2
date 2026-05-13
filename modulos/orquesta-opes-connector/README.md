# orquesta-opes-connector

Miniproyecto para el conector opt-in entre Orquesta y OPES.

Estado actual: contrato documental y cliente REST minimo para crear jobs
externos y enviar artefactos mediante endpoints publicos de OPES.

## Responsabilidad

- traducir jobs documentales OPES a contratos `orquesta-domain-work`;
- hablar con OPES solo por REST o MCP;
- conservar `correlation_id`, `idempotency_key`, `requested_by=orquesta` y
  `external_refs`;
- registrar entregas de Orquesta con `evidence_refs` hacia jobs y artefactos
  OPES;
- devolver artefactos mediante `POST /api/jobs/{id}/artifacts` o
  `submit_job_artifact`.

## Fuera de alcance

- acceder a base de datos, ficheros internos, rutas locales o workers de OPES;
- introducir conceptos de agente, sesion, lease, runtime, proveedor o modelo en
  OPES;
- asumir SQLite, Postgres u otro backend de OPES.

## Documentos

- `docs/contrato_v0.md`: frontera y contrato que debe respetar el conector.
- `docs/mapeo_rest_mcp_v0.md`: endpoints/tools OPES que el conector podria
  usar cuando exista implementacion.
- `docs/plan_integracion_v0.md`: cortes propuestos y criterios de aceptacion.

## Validacion

```sh
go test -count=1 ./modulos/orquesta-opes-connector
```
