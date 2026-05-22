# Contratos: orquesta-domain-work

## DomainWorkJobRequestV0

Entrada generica para pedir un trabajo de dominio externo.

Campos principales:

- `correlation_id`: correlacion estable entre app externa y Orquesta;
- `idempotency_key`: deduplicacion de reintentos;
- `requested_by`: origen de la peticion, por defecto `orquesta`;
- `domain_ref`: app o dominio externo;
- `interface_refs`: contratos REST/MCP externos expresados como refs;
- `work_kind`: tipo de trabajo de dominio, sin semantica de programacion;
- `input_fields`: campos de dominio ya normalizados por un adaptador superior,
  por ejemplo `program_ref`, `document_ref`, `level`, `language_code` o
  `source_refs`. Cada campo acepta `value` string, `values` lista de strings o
  `value_json` para objetos/arrays estructurados de dominio como
  `block_position` o `neighbor_context`;
- `work_refs`, `input_refs`, `external_refs` y `evidence_refs`: refs compactas.
  Para material largo, el texto completo no debe viajar como un unico
  `input_field`; debe viajar como refs/materiales externos mas un paquete
  editorial acotado.
- `acceptance_criteria` y `required_tests`: criterios y pruebas exigidas por el
  dominio propietario. Las pruebas se expresan como `test_ref`, refs de
  criterios/materiales/evidencias y `external_refs`; si hace falta resolverlas
  dinamicamente, el adaptador debe hacerlo por
  `DomainWorkRequiredTestPolicyPortV0`.

Invariantes:

- no contiene DB, rutas locales, HOME, OAuth, proveedor, modelo ni runtime;
- no transporta payloads sin contrato; `value_json` solo se permite dentro de
  campos nombrados y validados por adaptadores de dominio;
- no decide agentes, capacidad, sesiones ni reintentos;
- no decide estrategia con juicio propio: si el dominio necesita planificar,
  priorizar, dividir, revisar o elegir agentes, debe solicitar un `work_kind`
  de planificacion para que Orquesta lo resuelva mediante director;
- no exige granularidad minima: una unidad puede ser bloque, subcapitulo o
  capitulo si el dominio aporta contexto suficiente y revisable;
- si una materializacion requerida llega truncada, el agente debe bloquear o
  justificar en ACK que resolvio por refs/materializacion externa;
- no hay banco comun de tests de dominio en Orquesta: una app no-OPES, OPES u
  otra composicion aportan su politica por contrato y puerto propio, sin strings
  de producto dentro de `orquesta-domain-work`;
- todo conector real queda fuera de este modulo.

## DomainWorkArtifactSubmissionV0

Salida generica para devolver artefactos al dominio externo.

Campos principales:

- `job_ref`: job externo aceptado por el dominio propietario;
- `artifact_ref`: artefacto producido por Orquesta;
- `artifact_type`: tipo de artefacto de dominio;
- `payload_fields`: campos de resultado normalizados por el adaptador, por
  ejemplo `title`, `body`, `source_refs` o `stable_id`; tambien aceptan
  `value_json` cuando el dominio externo requiere estructura. Para
  `visual_asset`, los campos esperados son `asset_type`, `format`, `title`,
  `caption`, `alt_text`, `body`, `placement`, `language_code` y `source_refs`
  si aplica;
- `payload_refs`: referencias a payloads/materializaciones externas;
- `complete_job`: senal opcional para indicar cierre del job externo.

## Puertos

- `DomainWorkJobCreatorPortV0`
- `DomainWorkJobRecordSourcePortV0`
- `DomainWorkArtifactSubmitterPortV0`
- `DomainWorkRequiredTestPolicyPortV0`

Son puertos hexagonales. Un conector de DB, cola, REST, app propietaria o
runtime debe vivir en otro modulo y consumir estos contratos sin filtrar
internals al nucleo.

Contrato minimo para un creador de jobs:

- normaliza y valida `DomainWorkJobRequestV0` antes de materializar nada;
- materializa el request como job aceptado del dominio propietario y devuelve
  `DomainWorkJobV0{status=accepted, job_ref=...}`;
- usa `idempotency_key` como identidad de retry dentro del `domain_ref`;
- `request_id` no forma parte de la identidad idempotente; puede cambiar entre
  reintentos equivalentes;
- ante replay equivalente devuelve el mismo `job_ref`;
- ante misma clave con contrato distinto rechaza sin sobrescribir;
- no decide plan, runtime, agentes, modelos, cuotas ni estrategia;
- si usa DB, cola, REST o filesystem, esos detalles pertenecen al adaptador
  externo, no a este modulo.

Contrato minimo para una fuente de records:

- devuelve `DomainWorkJobRecordV0{request, job}` con el request normalizado y el
  job aceptado;
- filtra con `DomainWorkJobRecordFilterV0` por `domain_ref`, `work_kind`,
  `job_ref`, `correlation_id`, `idempotency_key`, `status` y pares exactos de
  `external_refs`;
- combina filtros con AND;
- aplica `limit` despues del filtro y del orden estable definido por el
  adaptador; `limit <= 0` significa sin limite;
- devuelve copias defensivas;
- no decide ejecucion, plan, agentes, DB concreta ni entrega de artefactos.

Contrato minimo para politica de tests requeridos de dominio:

- recibe un `DomainWorkJobRequestV0` normalizado;
- devuelve `DomainWorkRequiredTestPlanV0` con `domain_ref`, `work_kind`,
  criterios de aceptacion y `required_tests` expresados por refs compactas;
- puede usar `acceptance_criteria` como texto de dominio, pero toda identidad
  ejecutable o evidencia debe viajar por `test_ref`, `*_refs`,
  `external_refs` o `evidence_refs`;
- no ejecuta tests, no elige runtime/proveedor y no traduce nombres de producto
  a comandos dentro del contrato puro;
- cualquier mapeo productivo a runner, API o validador real pertenece al
  adaptador/composicion que implementa el puerto.

Referencia ejecutable:

- `modulos/orquesta-domain-work/contracttest` contiene la suite de conformidad
  reusable para cualquier `DomainWorkJobRecordStorePortV0`.
- `modulos/orquesta-domain-work-memory` implementa
  `DomainWorkJobRecordStorePortV0` en memoria para pruebas offline y como guia
  de comportamiento para conectores durables.
- `modulos/orquesta-domain-work-file` implementa
  `DomainWorkJobRecordStorePortV0` con snapshot JSON atomico, replay tras
  reinicio y lectura filtrada de records persistidos.
- `modulos/orquesta-domain-work-sql` implementa
  `DomainWorkJobRecordStorePortV0` sobre `database/sql` con `*sql.DB` inyectado,
  sin registrar drivers ni decidir dialecto desde el contrato puro.

Reglas minimas de adaptadores durables:

- validan antes de escribir;
- persisten request normalizado, job aceptado y huella idempotente;
- replay equivalente devuelve el mismo `job_ref`;
- conflicto de idempotencia no sobrescribe;
- corrupcion de estado debe ser visible, no reparada en silencio;
- no introducen plan, runtime, agente, modelo, cuota, red ni dominio concreto.

## DomainDocumentPlanV0

Contrato generico de plan documental producido por Orquesta mediante director y
agentes. Mantiene nombres neutros para que pueda servir a cualquier dominio
externo con un conector propio.

Aliases publicos:

- `PlanTemaV0`
- `PlanTemarioV0`

Work kinds previstos:

- `plan_documento`
- `plan_tema`
- `plan_temario`

Campos principales:

- `plan_ref`: ref compacta del plan;
- `domain_ref`: dominio propietario;
- `work_kind`: tipo de plan, siempre con prefijo `plan_`;
- `document_kind`: tipo de documento de dominio, por ejemplo `topic`,
  `syllabus` o `manual`;
- `scope_ref`: ref externa del tema, temario o documento si ya existe;
- `language_code`, `title`, `objective` y `target_audience`;
- `estimated_pages_min` y `estimated_pages_max`;
- `sections`: secciones/capitulos/bloques que deberan convertirse en trabajos
  posteriores como `draft_content_block`;
- `visuals`: recursos visuales planificados, normalmente
  `generate_visual_asset`;
- `review_steps`: revisiones necesarias, por ejemplo `validate_topic`;
- `deliverables`: artefactos finales obligatorios o opcionales, como
  `topic_expansion_package`, `visual_asset`, `markdown`, `html` o `pdf`;
- `quality_criteria`, `constraints`, `source_refs` y `evidence_refs`.

Invariantes:

- no contiene DB, rutas locales, HOME, proveedor, modelo ni runtime;
- no ejecuta el plan, solo lo describe;
- no sustituye al dominio propietario en validacion o ensamblado final;
- puede expandirse a jobs derivados con
  `modulos/orquesta-document-plan-expander`, pero ese expander tampoco ejecuta
  jobs ni toca DB/colas;
- debe tener al menos una seccion y un entregable;
- `work_kind` debe ser de planificacion, no de redaccion o exportacion;
- las secciones pueden ser amplias si el dominio aporta contexto suficiente;
- visuales, revisiones y entregables se citan por refs compactas y work kinds.
