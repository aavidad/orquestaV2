# orquesta-opes-connector

Miniproyecto para el conector opt-in entre Orquesta y OPES.

Estado actual: contrato documental y cliente REST minimo para consultar jobs
externos, leer bloques publicos de tema, crear jobs externos y enviar
artefactos mediante endpoints publicos de OPES. El artefacto `document_plan`
queda cubierto para `plan_tema`, `plan_temario` y `plan_documento`.

MCP de Orquesta ya existe como adaptador generico `orquesta.domain_work.v0` y
puede delegar en este conector cuando el stack lo inyecta. Lo que no hay aqui
es un cliente MCP especifico contra tools OPES; REST es la ruta funcional actual
para OPES.

## Responsabilidad

- traducir jobs documentales OPES a contratos `orquesta-domain-work`;
- permitir que OPES pida planificacion documental a Orquesta cuando haga falta
  juicio de director;
- hablar con OPES solo por REST o MCP;
- consultar ventanas pequenas de `GET /api/jobs` cuando un bridge externo debe
  drenar cola OPES;
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
- planificar temarios con logica local: el conector solo transporta la peticion
  para que Orquesta piense mediante director;
- asumir SQLite, Postgres u otro backend de OPES.

## Documentos

- `docs/contrato_v0.md`: frontera y contrato que debe respetar el conector.
- `docs/mapeo_rest_mcp_v0.md`: endpoints REST actuales, MCP generico de
  Orquesta y posible cliente MCP especifico OPES futuro.
- `docs/plan_integracion_v0.md`: cortes cerrados, pendientes y criterios de
  aceptacion.

## Validacion

```sh
go test -count=1 ./modulos/orquesta-opes-connector
```

## Estado T12

Para `T12 opes-consumer-smoke-real-opt-in`, el conector queda validado como
adaptador REST local por tests y fake aislado, pero el smoke real de
derivados/cierre permanece bloqueado hasta disponer de OPES temporal, Orquesta
temporal, confirmacion explicita de efectos y cuota/modelo confirmados. No
convertir esa falta de entorno en nueva implementacion del conector.
