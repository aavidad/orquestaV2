# Contratos: orquesta-domain-work

## DomainWorkJobRequestV0

Entrada generica para pedir un trabajo de dominio externo.

Campos principales:

- `correlation_id`: correlacion estable entre app externa y Orquesta;
- `idempotency_key`: deduplicacion de reintentos;
- `requested_by`: origen de la peticion, por defecto `orquesta`;
- `domain_ref`: app o dominio externo, por ejemplo `opes`;
- `interface_refs`: contratos REST/MCP externos expresados como refs;
- `work_kind`: tipo de trabajo de dominio, sin semantica de programacion;
- `input_fields`: campos de dominio ya normalizados por un adaptador superior,
  por ejemplo `program_id`, `topic_id`, `level`, `language_code` o
  `source_refs`. Cada campo acepta `value` string, `values` lista de strings o
  `value_json` para objetos/arrays estructurados de dominio como
  `block_position` o `neighbor_context`;
- `work_refs`, `input_refs`, `external_refs` y `evidence_refs`: refs compactas.
  Para material largo, como temas OPES de decenas de folios, el texto completo
  no debe viajar como un unico `input_field`; debe viajar como refs/materiales
  externos mas un paquete editorial acotado.

Invariantes:

- no contiene DB, rutas locales, HOME, OAuth, proveedor, modelo ni runtime;
- no transporta payloads sin contrato; `value_json` solo se permite dentro de
  campos nombrados y validados por adaptadores de dominio;
- no decide agentes, capacidad, sesiones ni reintentos;
- no exige granularidad minima: una unidad puede ser bloque, subcapitulo o
  capitulo si el dominio aporta contexto suficiente y revisable;
- si una materializacion requerida llega truncada, el agente debe bloquear o
  justificar en ACK que resolvio por refs/materializacion externa;
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
- `DomainWorkArtifactSubmitterPortV0`

Ambos son puertos hexagonales. Un conector OPES o de otra app debe vivir en otro
modulo y consumir estos contratos sin filtrar internals al nucleo.
