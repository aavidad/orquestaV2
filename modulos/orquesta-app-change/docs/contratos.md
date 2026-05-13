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
- `external_work`: metadata opaca opcional para trabajos de dominio de una app
  externa, por ejemplo una fabrica documental. Incluye `project_ref`,
  `interface_refs`, `work_kind` y `work_refs`.

`external_work` no contiene rutas reales, endpoints, DB, HOME, proveedor ni
modelo. La app propietaria conserva su dominio; Orquesta solo recibe trabajo
acotado para que el director lo convierta en microtareas si tambien hay
`allowed_write_set` y `acceptance_criteria`.

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
