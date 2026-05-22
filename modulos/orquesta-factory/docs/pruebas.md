# Pruebas locales: orquesta-factory

Registra pruebas obligatorias del modulo.

## AppSpec v0

```text
Caso: Sintaxis JSON de schemas y fixtures AppSpec v0
Tipo: contract
Comando: jq empty docs/schemas/app_spec_request_v0.schema.json docs/schemas/app_spec_v0.schema.json docs/fixtures/app_spec_v0/request_minima_valida.json docs/fixtures/app_spec_v0/request_i18n_invalida.json docs/fixtures/app_spec_v0/request_db_directa_invalida.json
Evidencia esperada: comando sin salida y exit code 0.
Ultima ejecucion: 2026-05-04, OK.
Riesgos: jq valida sintaxis, no semantica de JSON Schema.
```

```text
Caso: Request minima valida cumple AppSpecRequestV0
Tipo: contract
Comando: jq '.request' docs/fixtures/app_spec_v0/request_minima_valida.json > /tmp/request_minima_valida.request.json && jsonschema -i /tmp/request_minima_valida.request.json docs/schemas/app_spec_request_v0.schema.json
Evidencia esperada: comando sin salida y exit code 0.
Ultima ejecucion: 2026-05-04, OK.
Riesgos: jsonschema valida el contrato estructural; la aplicacion efectiva de defaults queda para FTY-003/FTY-004.
```

```text
Caso: Request con i18n desactivado sin justificacion falla
Tipo: contract
Comando: jq '.request' docs/fixtures/app_spec_v0/request_i18n_invalida.json > /tmp/request_i18n_invalida.request.json && jsonschema -i /tmp/request_i18n_invalida.request.json docs/schemas/app_spec_request_v0.schema.json
Evidencia esperada: falla indicando que i18n.justificacion es obligatoria.
Ultima ejecucion: 2026-05-04, OK; jsonschema reporto "'justificacion' is a required property".
Riesgos: el mapeo al error publico opcion_incompatible queda documentado en el fixture y debe implementarse en FTY-003.
```

```text
Caso: Request con proveedor DB directo falla
Tipo: contract
Comando: jq '.request' docs/fixtures/app_spec_v0/request_db_directa_invalida.json > /tmp/request_db_directa_invalida.request.json && jsonschema -i /tmp/request_db_directa_invalida.request.json docs/schemas/app_spec_request_v0.schema.json
Evidencia esperada: falla rechazando provider/db directo en datos.
Ultima ejecucion: 2026-05-04, OK; jsonschema reporto propiedad adicional provider y regla not contra provider directo.
Riesgos: el mapeo al error publico app_spec_invalida queda documentado en el fixture y debe implementarse en FTY-003.
```

```text
Caso: Instancia temporal minima cumple AppSpecV0
Tipo: contract
Comando: jq -n '{schema_version:"app_spec.v0",spec_id:"spec-fty-001-minima",request_id:"req-fty-001-minima",created_at:"2026-05-04T10:00:00Z",locale:"es-ES",app:{nombre:"Panel de reservas",slug:"panel-de-reservas",objetivo:"Permitir que un equipo gestione solicitudes de reserva desde una interfaz web.",tipo_app:"web",usuarios_objetivo:[]},scope:{objetivos:["Gestionar solicitudes de reserva"],fuera_de_alcance:[],supuestos:[],preguntas_abiertas:[]},architecture:{patron:"hexagonal",modulos_iniciales:[],fronteras:[],contratos_esperados:["SolicitarNuevaApp v0"]},i18n:{enabled:true,default_locale:"es-ES",locales:["es-ES"]},data:{persistence_required:false,needs:[]},connectors:{required:[],optional:[]},platforms:["web"],deploy:{target:"sin_preferencia",restrictions:[]},quality:{tests:"basica",accessibility:"basica",security:[],compliance:[],observability:true},docs:{user:true,development:true,systems:true,locales:["es-ES"]},agent_preferences:{human_review:true,autonomy:"media",notes:[]},defaults_applied:[{campo:"architecture.patron",valor:"hexagonal",motivo:"Default del contrato AppSpec v0"}],validation:{estado:"valida",warnings:[],errores:[]}}' > /tmp/app_spec_v0_minima.json && jsonschema -i /tmp/app_spec_v0_minima.json docs/schemas/app_spec_v0.schema.json
Evidencia esperada: comando sin salida y exit code 0.
Ultima ejecucion: 2026-05-04, OK.
Riesgos: instancia temporal no se versiona como fixture porque FTY-002 solo pidio fixtures de request.
```

```text
Caso: Unit tests del validador puro AppSpecRequestV0
Tipo: unit
Comando: go test ./modulos/orquesta-factory
Evidencia esperada: tests en verde para fixtures validos/invalidos, arquitectura no hexagonal y locale invalido.
Ultima ejecucion: 2026-05-04, OK.
Riesgos: el validador Go implementa reglas publicas minimas; no reemplaza el JSON Schema canonico completo.
```

```text
Caso: Unit tests del caso de uso SolicitarNuevaAppV0
Tipo: unit
Comando: go test ./modulos/orquesta-factory
Evidencia esperada: request minima produce AppSpecV0 valida con defaults; opciones explicitas false se respetan; persistencia se expresa como conector; deploy no soportado y proveedor directo fallan con errores publicos.
Ultima ejecucion: 2026-05-04, OK.
Riesgos: el caso de uso genera AppSpecV0 pero todavia no genera BacklogInicialPropuestoV0; eso queda en FTY-005.
```

```text
Caso: Unit tests de GenerarBacklogInicialPropuestoV0
Tipo: unit
Comando: go test ./modulos/orquesta-factory
Evidencia esperada: AppSpecV0 valida produce fases y microtareas completas con objetivo, write-set, contrato, validacion y bloqueos; specs no validas fallan; persistencia, deploy, conectores requeridos y preguntas abiertas quedan reflejados.
Ultima ejecucion: 2026-05-04, OK.
Riesgos: backlog propuesto no valida schemas JSON externos; cubre el contrato Go puro de factory.
```

```text
Caso: Adaptador REST v0 de SolicitarNuevaApp
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-factory
Evidencia esperada: httptest de POST /api/v0/apps/spec devuelve JSON canonico con app_spec y backlog, X-Correlation-ID estable, errores 400 con []ValidationIssue para JSON desconocido/request invalida, 405 para metodo no permitido y backlog con fases/microtareas/schema.
Ultima ejecucion: 2026-05-04, OK.
Riesgos: no levanta servidor real ni router externo; valida el handler inbound fino y no pruebas de red end-to-end.
```

```text
Caso: Saneamiento de appspec_usecase_v0.go
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-factory
Evidencia esperada: tests existentes siguen en verde tras mover DTOs, ensamblado, normalizacion, defaults, identidad y helpers de colecciones a ficheros separados.
Ultima ejecucion: 2026-05-04, OK.
Riesgos: no cambia cobertura funcional; verifica que la division mecanica no altera contratos publicos ni defaults.
```

```text
Caso: Adaptador REST AppSpec fuera de factory
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-factory ./modulos/orquesta-factory-http
Evidencia esperada: factory compila sin net/http; orquesta-factory-http publica handler, path, clock y envelopes REST, delegando negocio en SolicitarNuevaAppV0 y GenerarBacklogInicialPropuestoV0.
Ultima ejecucion: 2026-05-23, OK.
Riesgos: nuevos consumidores no deben volver a importar transporte HTTP desde orquesta-factory; la prueba raiz de arquitectura cubre la frontera.
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
