# Web Director Stats v0

Tipo: puerto_salida web hacia puerto inbound de estadisticas del director.

Responsabilidad:

- Enviar `run_ref`, correlacion, locale y flags de inclusion al inbound de stats.
- Recibir un envelope publico `ok|error` con `stats` o `errores_publicos`.
- Proyectar un panel web compacto con `run_ref`, contadores, progreso por tareas, progreso por agentes y errores publicos.
- Preparar refresco semitiempo-real desde `/director-stats` con `include_process_refs=true`, `include_agent_progress=true` e `include_agent_usage=true`, reutilizando el mismo cliente/endpoint y sin crear backend paralelo.
- Preparar la proyeccion web para estado de checkpoint si el API/handler entrega
  `checkpoint_agents_pending`, `pending_checkpoint_agent_refs` y
  `checkpoint_evidence_refs` junto a las stats del run.

Invariantes:

- La web no carga `RunStore`, DB, runtime, procesos, proveedor, HOME ni OAuth.
- El cliente REST usa `base_url`, `endpoint` y timeout inyectados.
- Los DTOs locales son frontera JSON; no dependen de structs internos del nucleo en runtime.
- `process_ref`, rutas, transcripts, proveedor, SQL y DSN no se proyectan al viewmodel.
- Las refs de checkpoint son opacas; la web no resuelve agentes, runtime,
  ficheros de ACK ni evidencia.
- Todo texto visible del panel sale de i18n local `es-ES|en-US`.
- La pagina web puede publicar metadatos de refresco (`href`, metodo e intervalo) solo si hay `run_ref`; no abre conexiones push ni conoce stores/runtime.
- Corte actual: `/director-stats` no invoca `orquesta.server.shutdown.v0` ni
  `server-shutdown`; solo proyecta esos campos si llegan ya saneados por el
  contrato consumido.

Errores publicos:

- `error_transporte`
- `respuesta_invalida`
- issues devueltos por el inbound en `errores_publicos`
- issues de `stats.progress.issues`

Pruebas:

- `director_stats_viewmodel_v0_test.go` valida proyeccion compacta y saneada.
- `director_stats_client_v0_test.go` valida contrato HTTP, includes por defecto, correlacion, errores publicos y errores de cliente.
- `director_stats_web_v0_test.go` valida que el endpoint web pide referencias de proceso, progreso compacto y uso de agentes por defecto, y devuelve metadatos de refresco localizados.
- `director_stats_viewmodel_v0_test.go` valida que un checkpoint pendiente
  recibido por contrato se muestra como atencion, con refs deduplicadas y sin
  acoplar la web a runtime/shutdown.
