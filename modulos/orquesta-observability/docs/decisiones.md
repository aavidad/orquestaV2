# Decisiones locales: orquesta-observability

Las decisiones de este archivo solo afectan a `orquesta-observability`. Si afectan a otro modulo, deben elevarse al director y registrarse en `../../CONTRATOS.md`.

## Plantilla

```text
Fecha:
Decision:
Motivo:
Alternativas:
Impacto:
Contratos afectados:
Estado:
```

## Decisiones iniciales desde DB v1

```text
Fecha: 2026-05-27
Decision: Usar los DTOs/validadores de observability como fuente de T198 para resources MCP read-only.
Motivo: operational-status y workspace timeline ya tienen contrato compacto; el descriptor MCP debe enlazar a esos owners y no a strings stale.
Alternativas: Duplicar shapes en `orquesta-mcp`; declarar el resource stale aunque el owner exista.
Impacto: `orquesta-mcp` puede publicar `descriptor_source` hacia observability; observability no expone stores reales ni decide negocio.
Contratos afectados: OperationalStatusQueryV0; WorkspaceTimelineQueryV0; MCPResourceDescriptorSourceV0.
Estado: aceptada_local
Revalidacion 2026-05-27: `agent-ref-task-autoprogramming-c3678e9bc306-g01`
mantiene esta fuente sin ampliar persistencia ni observabilidad productiva.
```

```text
Fecha: 2026-05-04
Decision: Usar transcripts, telemetria y auditoria de DB v1 solo como evidencia agregada.
Motivo: La DB historica contiene volumen enorme y util para aprender patrones, pero cargarlo en contexto degradaria agentes y puede mezclar estado viejo con decisiones nuevas.
Alternativas: Ignorar toda la telemetria; copiar transcripts a docs; consultar libremente la DB en cada tarea.
Impacto: `orquesta-observability` debe definir eventos y consultas compactas; cualquier uso de transcript completo requiere consulta acotada al director.
Contratos afectados: OrquestaEvent v0.
Estado: aceptada_director
```

```text
Fecha: 2026-05-04
Decision: Fijar una politica de extraccion segura para fuentes historicas V1 sin habilitar lectura productiva.
Motivo: `runtime_transcript`, `runtime_telemetry_samples` y `audit_log` contienen senales utiles para diagnostico, pero tambien material sensible, volumen excesivo y detalles internos que no deben salir a contratos read-only.
Alternativas: Permitir consultas libres a DB historica; copiar ejemplos largos a docs; exponer transcripts resumidos por el consumidor; retrasar toda regla hasta construir el adaptador.
Impacto: Un adaptador futuro solo podra emitir agregados compactos, filtros, contadores, ventanas temporales acotadas y ejemplos pequenos ya saneados. Quedan prohibidos transcripts completos, prompts, completions, SQL, secretos, rutas HOME, detalles de proveedor, nombres de tablas y cualquier lectura productiva no mediada por contrato posterior.
Contratos afectados: `OrquestaEventV0`, `OperationalStatusQueryV0`, `DiagnosticoCompactoV0`.
Estado: aceptada_local
```

```text
Fecha: 2026-06-08
Decision: Definir `DirectorAutonomousOpsSnapshotV0` como DTO neutral read-only.
Motivo: MCP/Web necesitan una proyeccion compacta para el cockpit del Director
sin meter proveedor, runtime, OPES ni logica de UI en el nucleo.
Impacto: Observability define solo el shape y las invariantes; el builder vive
en el adaptador que ya dispone de stats/cola/contexto. El snapshot no ejecuta
efectos, no filtra entregas y no sustituye el cierre causal del Director.
Contratos afectados: `DirectorAutonomousOpsSnapshotV0`.
Estado: aceptada_local
```

```text
Fecha: 2026-05-04
Decision: Arrancar `OrquestaEvent v0` como envelope local compacto, validable por JSON Schema draft7.
Motivo: Core, runtime, capacity y review necesitan un formato comun de observacion antes de que exista DB, bus o proyecciones reales.
Alternativas: Esperar a formalizar el contrato global; definir eventos separados por modulo; usar logs libres sin schema.
Impacto: Los productores publican por `PublicarOrquestaEvent v0`; el event sink queda como adaptador; MCP/web quedan como lectores futuros de proyecciones compactas.
Contratos afectados: `PublicarOrquestaEvent v0`, `OrquestaEventV0`.
Estado: aceptada_local; promovida a contrato compartido por decision del director de 2026-05-04.
```

```text
Fecha: 2026-05-04
Decision: Prohibir transcripts completos, secretos y detalles de DB/proveedor en `OrquestaEventV0`.
Motivo: Observability debe generar contexto compacto y enlazable sin convertirse en persistencia paralela ni exponer material sensible.
Alternativas: Permitir payload libre; almacenar transcript resumido y completo juntos; delegar filtrado a consumidores MCP/web.
Impacto: El schema limita tamanos, obliga `privacy.contains_secret=false`, `privacy.contains_transcript=false` y rechaza claves de payload asociadas a secretos, transcripts, SQL, tablas, DSN, conexiones, proveedores, prompts, completions y texto bruto.
Contratos afectados: `OrquestaEventV0`.
Estado: aceptada_local
```

```text
Fecha: 2026-05-04
Decision: Mantener `OperationalStatusQueryV0` y `DiagnosticoCompactoV0` como candidato local documental de observability para diagnostico/progreso compacto.
Motivo: La consulta CLI necesita una respuesta contractual que evite acceso a internals, DB, runtime o sink, pero la superficie afecta a CLI/MCP/Web y por tanto no debe activarse globalmente desde este modulo.
Alternativas: Cambiar `modulos/CONTRATOS.md` en este corte; dejar CLI bloqueado sin candidato; permitir diagnostico CLI por lectura directa de DB/runtime.
Impacto: Observability define el shape read-only y las invariantes de privacidad; CLI/MCP/Web siguen bloqueados para consumo real hasta decision del director.
Contratos afectados: `OperationalStatusQueryV0`, `DiagnosticoCompactoV0`.
Estado: aceptada_local; superada_por_promocion_global
```

```text
Fecha: 2026-05-04
Decision: Implementar `OperationalStatusQueryV0` y `DiagnosticoCompactoV0` como DTOs con validacion pura Go.
Motivo: El contrato ya esta promovido globalmente y necesita una superficie validable antes de conectar CLI/MCP/Web a proyecciones reales.
Alternativas: Mantener solo documentacion; crear schema JSON primero; implementar un adaptador con lectura real de proyecciones.
Impacto: Observability valida shape, read-only por JSON estricto, consumidores autorizados, scopes/secciones acotadas, referencias opacas, `privacy` false y ausencia de secretos/transcripts/prompts/completions/SQL/DSN/HOME sin tocar DB, sink, filesystem, servidor, runtime ni recuperacion activa.
Contratos afectados: `OperationalStatusQueryV0`, `DiagnosticoCompactoV0`.
Estado: aceptada_local
```

```text
Fecha: 2026-05-04
Decision: Implementar un adaptador puro en memoria para `OperationalStatusQueryV0` orientado a contract tests de consumidores.
Motivo: CLI, MCP y Web necesitan probar consumo del contrato read-only sin esperar una fuente real de proyecciones ni cruzar internals.
Alternativas: Conectar a DB/event sink/runtime; crear un servidor fake; mantener solo DTOs sin query handler.
Impacto: El adaptador recibe `DiagnosticoCompactoV0` en el constructor, valida query y diagnosticos con validadores existentes, busca por `scope`, `subject_ref` y `correlation_id` como referencias opacas, aplica secciones/limite y devuelve errores publicos de disponibilidad/frescura. No lee DB, event sink, runtime, filesystem, bus, procesos ni red y no es backend productivo.
Contratos afectados: `OperationalStatusQueryV0`, `DiagnosticoCompactoV0`.
Estado: aceptada_local
```

```text
Fecha: 2026-05-04
Decision: Dividir `operational_status_v0.go` por responsabilidad sin cambiar comportamiento.
Motivo: El fichero estaba en zona roja de tamano y la regla global recomienda separar DTOs, validacion de query, validacion de diagnostico y helpers antes de nuevos cambios.
Alternativas: Mantener el fichero grande; extraer solo tipos; mezclar el saneamiento con cambios funcionales.
Impacto: `operational_status_v0.go` queda como API publica/decode; los DTOs y constantes viven en `operational_status_types_v0.go`; la validacion se separa en query, diagnostico y helpers compartidos. No cambian contratos publicos, tags JSON, errores ni adaptadores.
Contratos afectados: `OperationalStatusQueryV0`, `DiagnosticoCompactoV0`.
Estado: aceptada_local
```

```text
Fecha: 2026-05-04
Decision: Dividir `orquesta_event_v0.go` por responsabilidad sin cambiar comportamiento.
Motivo: El fichero estaba en zona roja de tamano y conviene mantener `OrquestaEvent v0` como superficie publica pequena antes de nuevas integraciones.
Alternativas: Mantener el fichero grande; extraer solo tipos; mezclar el saneamiento con cambios funcionales o nuevos adaptadores.
Impacto: `orquesta_event_v0.go` queda como API publica/decode/accept; constantes y DTOs viven en `orquesta_event_types_v0.go`; validacion estructural, payload compacto y helpers JSON/opacos quedan separados. No cambian contratos publicos, JSON tags, errores, fixtures, DB, runtime, sink ni proveedor real.
Contratos afectados: `PublicarOrquestaEvent v0`, `OrquestaEventV0`.
Estado: aceptada_local
```

```text
Fecha: 2026-05-04
Decision: Dividir `operational_status_v0_test.go` por escenario sin cambiar expectations.
Motivo: El fichero de tests habia crecido hasta una zona dificil de mantener tras DTOs, validadores y adaptador en memoria; conviene separar query, diagnostico compacto y helpers compartidos.
Alternativas: Mantener todos los tests juntos; renombrar tests; mezclar la division con nueva cobertura o cambios productivos.
Impacto: `operational_status_v0_test.go` queda centrado en `OperationalStatusQueryV0`; `operational_status_diagnostic_v0_test.go` cubre `DiagnosticoCompactoV0`; `operational_status_helpers_v0_test.go` aloja fixtures/helpers usados tambien por el adaptador en memoria. No cambian contratos, expectativas, nombres de tests ni codigo productivo.
Contratos afectados: `OperationalStatusQueryV0`, `DiagnosticoCompactoV0`.
Estado: aceptada_local
```

## Consulta al director cerrada

```text
Modulo origen: orquesta-observability
Modulo afectado: modulos/CONTRATOS.md
Bloqueo: resuelto; `OrquestaEvent v0` fue promovido a contrato compartido en `modulos/CONTRATOS.md`.
Pregunta concreta: Promovemos `OrquestaEvent v0` al contrato global como contrato compartido minimo para `orquesta-core`, `orquesta-runtime`, `orquesta-capacity`, `orquesta-review`, `orquesta-mcp` y `orquesta-web`?
Opcion recomendada: Si, anotar en `modulos/CONTRATOS.md` un resumen global que enlace a `orquesta-observability/docs/contratos.md`, manteniendo el detalle y schemas en este modulo.
Impacto: Habilita integraciones por puerto sin exponer internals ni DB real; cualquier cambio incompatible posterior requeriria v1 o nueva consulta.
Decision del director: aceptada el 2026-05-04. Productores autorizados: `orquesta-core`, `orquesta-runtime`, `orquesta-capacity` y `orquesta-review` futuro. Consumidores `orquesta-mcp` y `orquesta-web`: solo lectores futuros por proyecciones compactas. Event sink, DB, bus, fichero o memoria: adaptadores fuera del contrato de dominio.
Estado: cerrada_director
```

## CONSULTA AL DIRECTOR

```text
Modulo origen: orquesta-observability
Modulo afectado: modulos/CONTRATOS.md; orquesta-cli; orquesta-mcp; orquesta-web; orquesta-core como posible lector agregado.
Bloqueo: `OperationalStatusQueryV0` / `DiagnosticoCompactoV0` estaban definidos solo como candidato local; CLI no podia consumirlos como contrato compartido sin promocion global.
Pregunta concreta: Promovemos un contrato read-only de diagnostico/progreso compacto, propiedad de `orquesta-observability`, consumible por CLI/MCP/Web sin acceso directo a DB, runtime, event sink ni internals?
Opcion recomendada: Si, promover un resumen global minimo que enlace a `orquesta-observability/docs/contratos.md`, dejando schemas, fixtures y adaptadores para microtareas posteriores.
Impacto: Desbloquea diagnostico compacto para CLI/MCP/Web con referencias opacas, sin secretos y sin persistencia paralela; cualquier recuperacion activa o fuente real exigiria contrato/adaptador separado.
Decision del director: aceptada. `OperationalStatusQuery v0` queda promovido a `../../CONTRATOS.md` como contrato read-only minimo; schemas, fixtures, harness y adaptadores quedan para microtareas posteriores.
Estado: cerrada_director
```
