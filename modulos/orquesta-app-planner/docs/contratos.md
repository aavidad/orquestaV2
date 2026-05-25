# Contratos: orquesta-app-planner

## `AppMicrotaskPlanV0`

Plan compacto de microtareas para una app.

Estado publico: compatibilidad/preview para `app-runner`. No es la ruta
operativa preferente de `AppSpecV0`; cuando se necesita juicio del Director,
plan-state, waits por ola/cohorte, review/tests/cierre o recursion, la entrada
publica debe ser `orquesta.apps.arrancar_director.v0`.

Campos principales:

- `run_ref`, `app_ref`, `schema_version`;
- `units[]`: tarea, claim, agente logico, delivery esperada, phase, rol,
  `work_profile_kind` neutral, capacidad recomendada, write-set, criterios y
  dependencias por delivery.

Invariantes:

- no contiene proveedor, modelo, HOME, OAuth, tokens ni runtime concreto;
- cada unidad tiene write-set no vacio;
- las dependencias se expresan contra delivery refs, no contra procesos;
- una app completa se divide en cortes pequenos.
- `scale=large` aumenta el numero de cortes: arquitectura, dominio,
  persistencia por puerto, API, web, i18n, deploy, integracion, docs y revision.
- Persistencia y deploy se expresan como contratos/conectores; el plan no elige
  SQLite, Postgres, Docker, Kubernetes ni proveedor concreto.
- `work_profile_kind` reutiliza el contrato neutral de `orquesta-core-workflow`:
  preparacion/arquitectura son `code_study`, implementacion/integracion/entorno
  son `implementation`, documentacion es `documentation` y revision es `review`.

## `WorkProfileForUnitV0` y `WorkflowTaskForUnitV0`

Adaptadores puros desde una unidad del planner hacia `WorkProfileV0` y
`WorkflowTaskV0`.

Reglas:

- no crean un perfil propio de programacion paralelo al nucleo;
- conservan write-set, criterios, tests obligatorios y contrato de funcion;
- traducen dependencias internas desde `delivery_ref` del plan a `task_ref` para
  que `WorkflowTaskV0.depends_on` quede causal y reutilizable por el nucleo.
- rechazan unidades ajenas al plan o dependencias que no puedan mapearse a una
  unidad del plan.

## `AppPlanCandidateProviderV0`

Adaptador que implementa `CandidateProviderPortV0`.

Reglas:

- solo emite unidades de la fase actual;
- no reemite unidades ya entregadas, arrancadas, fallidas o paradas;
- solo emite unidades cuyas dependencias ya esten en `run.deliveries`;
- cada candidato incluye todos los claims de la ola lista para que el gate de
  concurrencia vea conflictos entre workers paralelos.
- el payload de capacidad/agente se resuelve desde el perfil neutral del nucleo,
  no desde textos propios del planner.

## `EvaluateAppPlanProgressV0`

Proyeccion compacta para web/director.

Reglas:

- calcula total, entregadas, pendientes, listas y bloqueadas desde deliveries;
- no lee procesos, logs, runtime ni filesystem;
- marca `complete=true` solo cuando todas las `delivery_ref` del plan estan
  registradas.

## Resolutores de lanzamiento

`AppPlanFunctionContractResolverV0`, `AppPlanLaunchEvidenceResolverV0` y
`AppPlanContextBundleResolverV0` adaptan una unidad del plan a los puertos del
lanzador.

Reglas:

- el contrato de funcion usa `RuntimeFunctionContractV0` canonico y activo;
- el ACK esperado es el `delivery_ref` de la unidad;
- el contexto materializado apunta al modulo local y mantiene la tarea de app
  como write-set y contrato compacto;
- la capacidad final puede venir de un puerto externo de decision, sin acoplar
  el planificador a proveedor, modelo, HOME ni credenciales.

## `AppPlanRequestFromAppSpecV0`

Adaptador puro desde `orquesta-factory.AppSpecV0` validada hacia
`AppPlanRequestV0`.

Reglas:

- solo acepta `AppSpecV0` con `validation.estado=valida`;
- no lee HTTP, MCP, DB ni filesystem;
- conserva `spec_id`, slug, nombre, locale y superficies `api`/`web`;
- rechaza tipos que este planner aun no sabe dividir.
- escala a `large` si la AppSpec exige persistencia, pruebas altas o deploy no
  local/sin preferencia.
