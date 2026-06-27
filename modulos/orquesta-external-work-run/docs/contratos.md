# Contratos: orquesta-external-work-run

## `StartExternalWorkRunV0`

Entrada:

- `StartExternalWorkRunRequestV0`.
- `StartExternalWorkRunPortsV0`.
- `StartExternalWorkRunConfigV0`.

Salida:

- `StartExternalWorkRunResultV0`.

Responsabilidad:

- normalizar una solicitud de trabajo externo ya definida;
- crear el run si no existe;
- abrir directamente `programacion`;
- registrar el `AppChangeRequestV0`;
- encolar el run para el supervisor global.

Invariantes:

- no arranca director LLM inicial;
- no conoce OPES, DB, runtime, proveedor, modelo ni filesystem;
- todos los efectos salen por puertos;
- exige `external_work` porque este flujo no sustituye a crear app completa;
- mantiene idempotencia por refs compactas de run, comandos y cola.

## `BuildExternalWorkGoalWorkSpecV0`

Entrada:

- `StartExternalWorkRunRequestV0`.
- `StartExternalWorkRunConfigV0`.

Salida:

- `GoalWorkSpecV0`.
- incidencias `ExternalWorkRunIssueV0`.

Responsabilidad:

- normalizar y validar la solicitud external-work sin exigir puertos;
- compilar el contrato neutral `DomainWork` a `GoalWorkSpecV0`;
- declarar objetivo, refs de contexto, reglas, write-set, tests, artefacto
  requerido y cierre por receipt de dominio;
- dejar el lanzamiento/observacion del Goal a la composicion opt-in.

Invariantes:

- no arranca runtime ni supervisor;
- no conoce OPES, DB, proveedor, modelo ni filesystem;
- conserva nombres de `input_fields` y, cuando el campo no parece sensible ni
  supera el presupuesto de contexto, incluye un resumen inline acotado en
  `context_refs` con `kind=input_field_value`;
- los campos sensibles, privados o demasiado grandes no se inlinean: quedan como
  ref durable al payload `AppChange`/`DomainWork` y el Goal debe bloquear con
  rework si necesita un valor omitido;
- declara `director_kind=runtime_goal`; el adaptador de composicion puede
  especializarlo, por ejemplo a Codex Goal, al lanzar;
- conserva `AllowedWriteSet` si viene declarado y si no usa un write-set logico
  `domain-work/<dominio>/<tipo>/<token>`.
