# Contratos: orquesta-app-director-intake

## `PrepareAppDirectorIntakeV0`

Entrada: `PrepareAppDirectorIntakeRequestV0`.

- `app_spec`: `AppSpecV0` validada por factory;
- `run_ref`, `project_ref`, `occurred_at`, `correlation_id`, `requested_by`;
- si faltan refs, se derivan de `spec_id` y slug.

Salida: `AppDirectorIntakePreparedV0`.

- `run`: run activo en `brainstorming_arquitectura`;
- `director_task`: tarea compacta del director;
- `director_tasks`: equipo de directores cuando la autonomia de agentes es
  `alta`; si no, contiene solo el director principal;
- `candidate_provider`: proveedor que emite candidatos de director.

Invariantes:

- el estado inicial usa comandos/eventos del workflow;
- la solicitud registra un `BrainstormRequested` por director preparado;
- el provider emite como maximo dos candidatos de trabajo por tick;
- no se decide proveedor, cuenta, credenciales, runtime, HOME ni DB;
- REST/MCP/web no seleccionan numero de agentes ni fases internas.

## `PrepareAppDirectorInputV0`

Entrada: `PrepareAppDirectorInputRequestV0`.

- `app_spec`: `AppDirectorInputSpecV0`, DTO neutral del intake;
- `run_ref`, `project_ref`, `occurred_at`, `correlation_id`, `requested_by`;
- si faltan refs, se derivan de `spec_id` y slug.

Salida: `AppDirectorIntakePreparedV0`, igual que el wrapper legacy
`PrepareAppDirectorIntakeV0`.

Invariantes:

- el calculo interno de tareas de director no necesita `orquesta-factory`;
- `PrepareAppDirectorIntakeV0` queda como adaptador de compatibilidad desde
  `AppSpecV0`;
- `AppDirectorInputSpecV0` transporta solo contexto funcional, politica
  `request_kind`/`execution_mode`, validacion y preferencias de agentes
  necesarias para preparar el run inicial.

## `AdvanceAppDirectorIntakeWizardV0`

Entrada: `AppDirectorIntakeWizardRequestV0`.

- `request_ref`, `source`, `locale`, `occurred_at`, refs opcionales y un
  borrador `AppSpecRequestV0`;
- `answers` permite aplicar respuestas puntuales por campo para formularios,
  API o MCP sin acoplar adaptadores;
- los campos de app se validan delegando en `orquesta-factory`.

Salida: `AppDirectorIntakeWizardResultV0`.

- `needs_input`: devuelve `next_question` como `DirectorQuestionV0` compacta
  con `summary` en clave i18n;
- `ready`: devuelve `app_spec` validada y `prepared` con el run inicial del
  director;
- `invalid`: devuelve issues compactos y no prepara run.

Invariantes:

- no persiste estado ni decide runtime, proveedor, modelo, HOME o DB;
- las preguntas son contrato de entrada, no UI localizada;
- al completar los datos reutiliza `SolicitarNuevaAppV0` y
  `PrepareAppDirectorIntakeV0`.

## `BuildHumanDirectorReviewablePlanV0`

Entrada: `HumanDirectorWorkIntakeRequestV0`.

- `request_ref`, `project_ref`, `worktree_ref` y `branch_ref` son refs opacas;
- `request` transporta objetivo humano, contexto, criterios, tests y reglas;
- `limits` fija limites operativos sin elegir proveedor ni runtime;
- `hints` permite areas, write-set tentativo, refs opacas, reparaciones seguras
  y motivos de revision.

Salida: `HumanDirectorReviewablePlanResultV0`.

- `plan` queda `ready_for_review` cuando las refs obligatorias son validas;
- `steps` distingue `execute_now`, `study_before`, `postpone_overlap` y
  `request_review`;
- los solapes no invalidan toda la entrada: el plan crea estudio previo y
  pospone el alcance solapado;
- el write-set amplio o ausente produce fase de estudio antes de preparar
  trabajo de codigo.

Invariantes:

- no crea `WorkflowTaskV0`, no ejecuta agentes y no prepara trabajo de codigo;
- no importa adaptadores, runtime, proveedor, HOME, DB, web ni MCP;
- normaliza alias y globs seguros, pero pide revision ante rutas absolutas,
  `..`, HOME o variables.
