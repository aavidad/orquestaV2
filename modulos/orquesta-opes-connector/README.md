# orquesta-opes-connector

Miniproyecto para el conector opt-in entre Orquesta y OPES.

Estado actual: contrato documental y cliente REST minimo para consultar jobs
externos, leer bloques publicos de tema, crear jobs externos y enviar
artefactos mediante endpoints publicos de OPES. El artefacto `document_plan`
queda cubierto para `plan_tema`, `plan_temario` y `plan_documento`.

Para `plan_temario`, la frontera vigente apunta al flujo local completo
documentado en `../../docs/opes_flujo_temario_operativo_2026-06-02.md`:
investigacion de examenes, redaccion, infografias, banco de tests, revisiones,
ensamblado, audio por tema/apartado, tutor/bots y HTML local USO/TCAE antes de
produccion. El conector transporta jobs y artefactos; OPES decide producto,
marca, UI, fuentes y adaptadores.

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
- tratar un HTTP 4xx de OPES al entregar artefactos como `receipt` invalido de
  dominio con codigo `opes_http_status_<status>`, no como puerto no disponible.

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
adaptador REST local por tests y fake aislado. El smoke real temporal de
derivados/cierre quedo cerrado funcionalmente por goal-first el 2026-06-28
hasta `completed_syllabus_package`; ver
`docs/runbooks/resultado_smoke_opes_derivados_goal_first_real_2026-06-28.md` y
la fila `OPES-DER-RESTO` de la matriz vigente. Repetirlo solo ante regresion,
con OPES temporal, scope duro, confirmacion explicita de efectos y cuota/modelo
confirmados.
