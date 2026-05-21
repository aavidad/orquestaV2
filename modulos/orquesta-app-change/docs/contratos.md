# Contratos: orquesta-app-change

## `AppChangeRequestV0`

Entrada para pedir cambios sobre una app ya creada.

Campos principales:

- `run_ref`: run existente que gobierna la app;
- `app_ref`: identificador opaco opcional de producto/app;
- `change_ref`: idempotencia de la solicitud;
- `user_intent`: peticion humana;
- `target_area`: area sugerida, como `web`, `api`, `docs` o `quality`;
- `current_state_refs`: refs compactas a estado previo;
- `acceptance_criteria`: criterios visibles de cierre;
- `allowed_write_set`: rutas relativas permitidas si el usuario/director ya
  quiere acotar el cambio.
- `required_tests`: pruebas o validaciones explicitas que deben conservarse en
  la microtarea del director; si faltan, el adaptador puede inferir pruebas
  minimas por write-set o dominio.
- `external_work`: metadata opaca opcional para trabajos de dominio de una app
  externa, por ejemplo una fabrica documental. Incluye `project_ref`,
  `job_ref`, `interface_refs`, `work_kind`, `work_refs` e `input_fields`.

`external_work` no contiene rutas reales, endpoints, DB, HOME, proveedor ni
modelo. La app propietaria conserva su dominio; Orquesta solo recibe trabajo
acotado para que el director lo convierta en microtareas si tambien hay
`allowed_write_set` y `acceptance_criteria`.

`external_work.input_fields` usa `orquesta-domain-work.DomainWorkFieldV0`.
Sirve para transportar contexto de dominio ya normalizado por el borde, como
`program_id`, `topic_id`, `level`, `language_code` o refs de fuentes. Los
nombres de campo deben ser compactos: sin espacios, barras ni saltos de linea.
Los campos vacios se eliminan y los duplicados exactos se compactan.

`external_work.job_ref` es la referencia opaca del trabajo externo si la app
propietaria ya lo ha creado. Orquesta la conserva para trazabilidad y para
devolver artefactos sin inferir IDs desde prefijos como `opes-job-*`.

## Puertos

- `AppChangeStorePortV0`: persistencia durable de la solicitud.
- `AppChangeDirectorNotifierPortV0`: entrega compacta al director.

Ambos son conectores. El modulo no trae SQLite, Postgres, filesystem, broker,
runtime ni proveedor por defecto.

## `AppChangeIntentEventV0`

Evento compacto para cambios a mitad de ejecucion procedentes de formulario,
API o MCP.

- `event_id`: idempotencia del evento externo;
- `run_ref`: run existente;
- `user_intent`: texto de la peticion humana;
- `change_ref`: opcional; si falta se deriva de `event_id`;
- el resto de campos coincide con `AppChangeRequestV0`.

`ReceiveAppChangeIntentEventV0` normaliza el evento a `AppChangeRequestV0`,
reutiliza la validacion existente, guarda por puerto y notifica al director por
puerto. La referencia de pregunta por defecto usa el prefijo
`question-ref-app-change` para que la fuente de decisiones pueda enlazarla con
la replanificacion.
