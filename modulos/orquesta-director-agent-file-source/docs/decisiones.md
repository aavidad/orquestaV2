# Decisiones

- El adaptador no escanea rutas. Un conector superior decide que artefactos estan pendientes.
- La lectura se abstrae con puerto para poder usar filesystem, memoria, MCP, REST o cualquier almacenamiento externo.
- El limite de lectura existe en el adaptador, no en el nucleo.
- Los artefactos pueden contener varias decisiones para permitir que un director entregue un plan compacto de fases.
- No se guarda estado de procesado aqui. La deduplicacion durable pertenece al workflow y a los conectores superiores.
- `create_microtask` normaliza `decision.phase_id` a `planificacion_microtareas`
  cuando el director externo copia por error la fase objetivo de la tarea. La
  fase canonica de ejecucion queda en `create_microtask.task.phase_id`.
- El request del proveedor de ficheros incluye `phase_artifacts` y `deliveries`
  del run. El source no decide causalidad, pero entrega esas proyecciones a
  conectores superiores para que puedan impedir que un sidecar de decisiones se
  consuma antes del ACK que lo produjo.
