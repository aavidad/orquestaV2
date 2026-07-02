# Incidencia: auditoria OPES de temario existente sin runtime Orquesta descubierto

Fecha: 2026-07-02.

## Sintoma

Una auditoria de `Grupo B Informatica` busco Orquesta en puertos historicos
`19023-19028`. Al no encontrar runtime local, la auditoria se cerro con
validadores directos y no quedo como ola dirigida por Orquesta ni como dictamen
`rework_mayor` materializable desde el conector.

## Lectura arquitectonica

El discovery generico de Orquesta ya esta cubierto por el contrato de
`/api/v0/server/readiness` y `/api/v0/server/resources.route_manifest`.
El hueco restante era de dominio OPES: no habia un `work_kind` explicito para
auditar un temario ya existente, devolver un dictamen global y preparar
`rework_task_requests` por tema.

## Mitigacion

`orquesta-opes-bridge` incorpora el contrato
`audit_existing_syllabus_quality` y alias:

- `audit_temario_existente`
- `quality_audit_existing_syllabus`
- `auditoria_calidad_temario_existente`

El bridge conserva el `work_kind`, lo puede transportar como `review_textual`,
declara `expected_artifact_type=opes_quality_audit_report`,
`audit_contract=opes_existing_syllabus_quality_audit.v0` y decision global
obligatoria `apto|revision|rework_menor|rework_mayor|bloqueado`.

## Evidencia

- `TestBuildExternalWorkRunRequestV0MapeaAuditoriaTemarioExistenteV0`
- `TestOPESBridgeArtifactContractMapConsumeOwnerNeutralV0`
- `TestOPESBridgeTransportJobTypeForWorkKindV0`
- `go test -count=1 ./modulos/orquesta-opes-bridge`

## Limite

No arranca Orquesta ni OPES productivo. El runtime activo sigue debiendo
descubrirse por readiness/resources, no por puertos memorizados ni `/healthz`.
