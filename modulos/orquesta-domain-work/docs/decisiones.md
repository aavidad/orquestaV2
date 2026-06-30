# Decisiones: orquesta-domain-work

```text
Fecha: 2026-06-28
Decision: Anadir contrato neutral de capacidades externas para jobs de dominio.
Motivo: los trabajos que producen `audio_asset` necesitan una frontera clara
para TTS/sintesis de voz sin asumir runner local, credenciales, modelo, host de
edge ni proveedor dentro del nucleo.
Alternativas: meter TTS en OPES; usar strings en `input_fields`; bloquear por
heuristicas de work_kind en adaptadores; implementar un runner real dentro del
contrato puro.
Impacto: `audio_asset` deriva requisito `speech_synthesis`; la composicion
declara capacidades por `DomainWorkExternalCapabilitySourcePortV0`;
`EvaluateDomainWorkExternalCapabilitiesV0` bloquea con razon operativa si falta
la capacidad y conserva refs/evidencias. El modulo sigue sin red, DB,
filesystem, runtime, OPES, Codex ni proveedor.
Estado: aceptada_local
```

```text
Fecha: 2026-06-30
Decision: Ampliar `speech_synthesis` con heartbeat/progreso y timeout de
proveedor antes de aceptar trabajos `audio_asset`.
Motivo: un conector externo puede dejar una ola en estado `running` sin avance
si el proveedor TTS queda colgado y Orquesta solo observa el proceso global.
Alternativas: confiar en logs del adaptador; aceptar el job y reparar despues;
meter un proveedor TTS concreto en el contrato neutral.
Impacto: el contrato puro exige flags declarativos de progreso observable,
timeout recuperable y ventana maxima sin avance. El proveedor, el sidecar de
heartbeat y la reparacion concreta siguen fuera de `orquesta-domain-work`.
Estado: aceptada_local
```

```text
Fecha: 2026-05-17
Decision: Reconocer `orquesta-domain-work-sql` como adaptador SQL externo de
`DomainWorkJobRecordStorePortV0`.
Motivo: el nucleo necesita una ruta hacia conectores DB sin convertir
`orquesta-domain-work` en infraestructura ni elegir un motor canonico.
Alternativas: meter `database/sql` en el contrato puro; crear primero un driver
SQLite/Postgres concreto; reutilizar `orquesta-persistence`; seguir solo con
file-based.
Impacto: el adaptador SQL vive fuera del nucleo, recibe `*sql.DB` ya abierto,
ejecuta `contracttest` y queda prohibido como import desde paquetes neutrales.
No cambia el Director ni el servidor.
Estado: aceptada_local
```

```text
Fecha: 2026-05-17
Decision: Crear `orquesta-domain-work/contracttest` como suite de conformidad
para `DomainWorkJobRecordStorePortV0`.
Motivo: `memory`, `file` y futuros conectores DB/cola/REST deben compartir la
misma semantica de idempotencia, filtros, `limit`, contexto cancelado y copias
defensivas sin importar adaptadores concretos desde el contrato puro.
Alternativas: duplicar tests en cada adaptador; extraer helpers de produccion;
esperar al primer conector DB.
Impacto: los adaptadores invocan el harness desde sus `_test.go`. El paquete de
contract tests solo importa stdlib y `orquesta-domain-work`; no introduce DB,
filesystem productivo, OPES, Codex, runtime ni servidor.
Estado: aceptada_local
```

```text
Fecha: 2026-05-17
Decision: Anadir `DomainWorkJobRecordSourcePortV0` como puerto neutral de lectura
filtrada, separado de `DomainWorkJobCreatorPortV0`.
Motivo: los conectores durables necesitan exponer records `Request+Job` para
inspeccion, operacion local y futuros bundles DB sin obligar al puerto de
comando a listar estado ni filtrar desde capas con filesystem.
Alternativas: ampliar `DomainWorkJobCreatorPortV0`; dejar lectura solo como API
del adaptador file; hacer que servidor/director lean snapshots directamente.
Impacto: `DomainWorkJobRecordStorePortV0` compone creador + fuente, pero el
comando de creacion sigue puro. Los adaptadores `memory` y `file` implementan el
puerto; `orquesta-domain-work` sigue sin DB, filesystem, OPES, Codex ni runtime.
Estado: aceptada_local
```

```text
Fecha: 2026-05-17
Decision: Reconocer `orquesta-domain-work-file` como adaptador durable externo
del puerto `DomainWorkJobCreatorPortV0`.
Motivo: el contrato puro necesita implementaciones ejecutables para validar que
otros dominios pueden crear jobs sin filtrar internals ni depender de OPES,
Codex o runtime. La durabilidad debe quedar fuera del modulo de contrato.
Alternativas: meter filesystem en `orquesta-domain-work`; usar
`orquesta-persistence`; crear directamente un conector DB.
Impacto: `orquesta-domain-work` sigue siendo puro. El adaptador file-based
sirve como referencia durable hasta que exista un conector DB/cola/REST real.
Estado: aceptada_local
```

```text
Fecha: 2026-05-17
Decision: Documentar el contrato esperado de los creadores genericos de jobs de
dominio.
Motivo: el nucleo ya estaba abierto a apps externas, pero faltaba una regla
operativa clara para que conectores de DB, cola, REST o app propietaria creen
jobs sin filtrar internals ni semantica de producto.
Alternativas: dejar cada conector con idempotencia propia; meter la
implementacion dentro de `orquesta-domain-work`; acoplar el contrato a un
producto concreto.
Impacto: `DomainWorkJobCreatorPortV0` queda definido como puerto puro. Los
adaptadores deben normalizar, validar, aplicar replay idempotente y rechazar
conflictos sin decidir plan, runtime, agentes, DB concreta ni estrategia.
Estado: aceptada_local
```

```text
Fecha: 2026-05-14
Decision: Crear `DomainDocumentPlanV0` como contrato ejecutable de
planificacion documental.
Motivo: existian work kinds parciales (`draft_topic_outline`,
`create_exam_outline`, `summarize_topic`, `expand_topic_from_summary`), pero
no un contrato completo para que el director devuelva un plan de tema/temario
validable por OPES u otro dominio.
Alternativas: dejarlo como docs; meter `PlanTemaV0` dentro del conector OPES;
usar JSON libre en `DomainWorkFieldV0`.
Impacto: el modulo define `DomainDocumentPlanV0`, aliases `PlanTemaV0` y
`PlanTemarioV0`, normalizacion y validacion. El contrato exige secciones,
entregables, work_kind de planificacion y refs compactas. No incorpora OPES,
DB, runtime, proveedor ni rutas.
Estado: aceptada_local
```

```text
Fecha: 2026-05-13
Decision: Crear `orquesta-domain-work` como contrato neutro para apps externas.
Motivo: Orquesta debe ser nucleo de orquestacion de agentes. Programacion, OPES
u otras apps deben usar la orquestacion por contratos sin integrar su nucleo ni
gestionar agentes.
Alternativas: mover codigo actual de programacion; crear conectores OPES
directos sin contrato comun; hacer que cada app gestione sus agentes.
Impacto: se anade un contrato puro de jobs y artefactos de dominio. No cambia
el flujo actual de programacion ni se introduce REST/MCP/DB.
Estado: aceptada_local
```

```text
Fecha: 2026-05-13
Decision: Preparar OPES en un modulo documental separado llamado
`orquesta-opes-connector`.
Motivo: OPES necesita un conector REST/MCP opt-in, pero el contrato generico no
debe absorber semantica editorial ni detalles de transporte.
Alternativas: meter OPES en `orquesta-domain-work`; crear cliente real ya;
dejar solo una nota global en `docs/`.
Impacto: se documenta la frontera del conector futuro sin tocar codigo
productivo ni OPES.
Estado: aceptada_local
```

```text
Fecha: 2026-05-13
Decision: El modulo transporta refs compactas, metadatos de correlacion y
campos de dominio normalizados, pero no JSON libre sin contrato.
Motivo: OPES necesita campos como `program_id`, `topic_id`, `level`,
`language_code`, `title` o `body` para crear jobs y artefactos. Pasarlos como
campos normalizados permite funcionar sin meter tipos OPES ni rutas internas en
el nucleo.
Alternativas: incluir `payload_json` arbitrario; pasar solo refs y obligar al
conector a leer internals; usar tipos propios de OPES.
Impacto: `input_fields` y `payload_fields` son datos de dominio declarados por
adaptadores superiores. Los conectores materializan JSON REST/MCP fuera de este
modulo y mantienen evidencia por refs.
Estado: aceptada_local
```
