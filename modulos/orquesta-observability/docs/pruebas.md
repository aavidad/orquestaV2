# Pruebas locales: orquesta-observability

Registra pruebas obligatorias del modulo.

```text
Caso: OBS-CT-012 descriptor_source MCP read-only
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-operator-mcp ./modulos/orquesta-observability ./modulos/orquesta-governance ./modulos/orquesta-core ./cmd/orquesta-server
Evidencia esperada: los resources MCP que apuntan a observability declaran
owner, fuente canonica, DTO/validador y errores publicos sin exponer DB,
runtime, transcripts, prompts, completions ni payloads completos.
Ultima ejecucion: 2026-05-27, ok en paquete
`agent-ref-task-autoprogramming-c3678e9bc306-g01`.
Riesgos: La prueba no crea un sink productivo; valida contrato y transporte existentes.
Revalidacion: `agent-ref-task-autoprogramming-c3678e9bc306-g01` conserva el
comando obligatorio como evidencia de cierre T198.
```

## `OrquestaEvent v0`

```text
Caso: Schema y fixtures son JSON validos.
Tipo: contract
Comando: jq empty docs/schemas/orquesta_event_v0.schema.json docs/fixtures/orquesta_event_v0/*.json
Evidencia esperada: salida vacia y codigo 0.
Ultima ejecucion: 2026-05-04, correcta.
Riesgos: `jq` solo valida sintaxis JSON, no invariantes de contrato.
```

```text
Caso: Fixture positivo cumple `OrquestaEventV0` draft7.
Tipo: contract
Comando: npx --yes ajv-cli@5 validate --spec=draft7 -s docs/schemas/orquesta_event_v0.schema.json -d docs/fixtures/orquesta_event_v0/core_event_minimo_valido.json
Evidencia esperada: fixture valido aceptado.
Ultima ejecucion: 2026-05-04, correcta.
Riesgos: validaciones relacionales futuras pueden requerir harness adicional.
```

```text
Caso: Fixtures negativos son rechazados por `OrquestaEventV0`.
Tipo: contract
Comando: ! npx --yes ajv-cli@5 validate --spec=draft7 -s docs/schemas/orquesta_event_v0.schema.json -d docs/fixtures/orquesta_event_v0/transcript_completo_invalido.json && ! npx --yes ajv-cli@5 validate --spec=draft7 -s docs/schemas/orquesta_event_v0.schema.json -d docs/fixtures/orquesta_event_v0/secreto_detectado_invalido.json
Evidencia esperada: ambos fixtures invalidos fallan validacion; el wrapper `!` devuelve codigo 0 para el caso esperado.
Ultima ejecucion: 2026-05-04, correcta.
Riesgos: `ajv-cli` reporta el primer fallo; no sustituye pruebas unitarias del puerto cuando exista codigo.
```

```text
Caso: DTOs y validacion pura Go de `OrquestaEventV0` y `PublishOrquestaEventRequestV0`.
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-observability
Evidencia esperada: fixture valido aceptado; fixtures invalidos rechazados; request sin sink real devuelve `stored=false`; se validan `source_area`/`event_type`, `privacy` false, payload compacto y claves prohibidas.
Ultima ejecucion: 2026-05-04, correcta.
Riesgos: no prueba adaptadores reales porque DB, bus, filesystem, memoria, servidor y sink real quedan fuera del contrato de este corte.
```

```text
Caso: Saneamiento de tamano de `orquesta_event_v0.go`.
Tipo: unit | contract
Comando: gofmt -w orquesta_event_v0.go orquesta_event_types_v0.go orquesta_event_validation_v0.go orquesta_event_payload_validation_v0.go orquesta_event_helpers_v0.go && go test -count=1 ./modulos/orquesta-observability && git diff --check -- modulos/orquesta-observability && wc -l modulos/orquesta-observability/orquesta_event_v0.go modulos/orquesta-observability/orquesta_event_types_v0.go modulos/orquesta-observability/orquesta_event_validation_v0.go modulos/orquesta-observability/orquesta_event_payload_validation_v0.go modulos/orquesta-observability/orquesta_event_helpers_v0.go
Evidencia esperada: tests existentes pasan; no hay whitespace invalido; los Go tocados quedan por debajo de 300 lineas; no cambian contratos publicos, JSON tags, errores ni fixtures.
Ultima ejecucion: 2026-05-04, correcta.
Riesgos: refactor mecanico; no anyade cobertura nueva ni prueba adaptadores reales.
```

## `OperationalStatusQueryV0` / `DiagnosticoCompactoV0`

```text
Caso: DTOs y validacion pura Go de `OperationalStatusQueryV0` y `DiagnosticoCompactoV0`.
Tipo: unit
Comando: go test -count=1 .
Evidencia esperada: query valida aceptada; JSON con campos de recuperacion activa rechazado; consumidores, scopes, secciones, limites, referencias opacas, `privacy` false y contenido prohibido se validan sin DB, sink, runtime, filesystem ni servidor.
Ultima ejecucion: 2026-05-04, correcta.
Riesgos: no prueba adaptadores reales ni disponibilidad de proyecciones; schema JSON y fixtures quedan para microtarea posterior si un consumidor los necesita.
```

```text
Caso: Adaptador puro en memoria para `OperationalStatusQueryV0`.
Tipo: unit | contract
Comando: go test -count=1 ./modulos/orquesta-observability
Evidencia esperada: constructor valida diagnosticos compactos; handler valida query; lookup por scope+subject_ref+correlation_id devuelve copia filtrada por secciones/limite; errores `proyeccion_no_disponible`, `diagnostico_no_disponible` y `frescura_no_garantizada` se exponen cuando aplican; no hay DB, sink, runtime, filesystem, bus, procesos ni contenido prohibido.
Ultima ejecucion: 2026-05-04, correcta.
Riesgos: sirve para contract tests de CLI/MCP/Web/Core autorizado; no sustituye un adaptador productivo de proyecciones compactas reales.
```

```text
Caso: Saneamiento de tamano de `operational_status_v0.go`.
Tipo: unit | contract
Comando: gofmt -w operational_status_v0.go operational_status_types_v0.go operational_status_query_validation_v0.go operational_status_diagnostic_validation_v0.go operational_status_helpers_v0.go && go test -count=1 ./modulos/orquesta-observability && git diff --check -- modulos/orquesta-observability && wc -l modulos/orquesta-observability/operational_status_v0.go modulos/orquesta-observability/operational_status_types_v0.go modulos/orquesta-observability/operational_status_query_validation_v0.go modulos/orquesta-observability/operational_status_diagnostic_validation_v0.go modulos/orquesta-observability/operational_status_helpers_v0.go
Evidencia esperada: tests existentes pasan; no hay whitespace invalido; los Go tocados quedan por debajo de 300-350 lineas; no cambian contratos publicos, JSON tags ni errores.
Ultima ejecucion: 2026-05-04, correcta.
Riesgos: refactor mecanico; no anyade cobertura nueva ni prueba adaptadores reales.
```

```text
Caso: Division de `operational_status_v0_test.go` por contrato o escenario.
Tipo: unit | contract
Comando: gofmt -w modulos/orquesta-observability/operational_status_v0_test.go modulos/orquesta-observability/operational_status_diagnostic_v0_test.go modulos/orquesta-observability/operational_status_helpers_v0_test.go && go test -count=1 ./modulos/orquesta-observability && git diff --check -- modulos/orquesta-observability && wc -l modulos/orquesta-observability/operational_status_v0_test.go modulos/orquesta-observability/operational_status_diagnostic_v0_test.go modulos/orquesta-observability/operational_status_helpers_v0_test.go
Evidencia esperada: tests existentes pasan; no hay whitespace invalido; `operational_status_v0_test.go` queda por escenario de query, diagnostico compacto queda en su propio test y los helpers compartidos quedan separados.
Ultima ejecucion: 2026-05-04, correcta.
Riesgos: refactor mecanico de tests; no anyade cobertura nueva ni modifica contrato o codigo productivo.
```

## `DirectorDecisionContextV0`

```text
Caso: DTO y validacion pura Go de `DirectorDecisionContextV0`.
Tipo: unit | contract
Comando: go test -count=1 ./modulos/orquesta-observability
Evidencia esperada: la proyeccion acepta progreso por fase/tarea/agente,
procesos y sesiones opacas, actividad reciente por claves i18n, agentes vivos,
parados y fallidos, bloqueos, cierre bloqueado, rework/replan, duraciones y
quietud; rechaza refs no opacas, secretos, transcripts, texto visible no i18n y
listas no compactas.
Ultima ejecucion: 2026-05-10; pasa.
Riesgos: valida el DTO compacto, no implementa adaptador productivo ni consulta
DB/runtime/procesos.
```

## `DirectorAutonomousOpsSnapshotV0`

```text
Caso: DTO neutral del snapshot operativo del Director.
Tipo: unit | contract
Comando: go test -count=1 ./modulos/orquesta-observability
Evidencia esperada: el modulo compila el DTO read-only con cola, runs, agentes,
decision compacta y privacy metadata-only. La validacion integrada del contrato
se cierra desde MCP/Web al publicar y consumir `ops_snapshot`.
Ultima ejecucion: 2026-06-08; pasa en el corte que publica el DTO.
Riesgos: El DTO no rellena por si solo waits, olas, cohortes ni modelos; esas
fuentes deben entrar por puertos publicos posteriores.
```

## `OBS-001` politica documental de extraccion segura

```text
Caso: La politica documental de extraccion segura limita las fuentes V1 a agregados compactos y ejemplos pequenos saneados.
Tipo: contract
Comando: git diff --check -- modulos/orquesta-observability
Evidencia esperada: `docs/contratos.md`, `docs/decisiones.md`, `docs/tareas.md` y `docs/pruebas.md` dejan explicito que `runtime_transcript`, `runtime_telemetry_samples` y `audit_log` solo podran usarse via un adaptador futuro para contadores, filtros, agregados y evidencia minima saneada; no habilitan DB real ni lectura productiva.
Ultima ejecucion: 2026-05-04, pendiente en este corte documental hasta cerrar la edicion.
Riesgos: es una validacion textual; no sustituye pruebas del adaptador futuro ni garantiza acceso seguro a una fuente real.
```

## Plantilla

```text
Caso:
Tipo: unit | contract | integration | smoke
Comando:
Evidencia esperada:
Ultima ejecucion:
Riesgos:
```
