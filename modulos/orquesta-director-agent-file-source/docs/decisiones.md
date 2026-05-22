# Decisiones

- El adaptador no escanea rutas. Un conector superior decide que artefactos estan pendientes.
- La lectura se abstrae con puerto para poder usar filesystem, memoria, MCP, REST o cualquier almacenamiento externo.
- El limite de lectura existe en el adaptador, no en el nucleo.
- Los artefactos pueden contener varias decisiones para permitir que un director entregue un plan compacto de fases.
- No se guarda estado de procesado aqui. La deduplicacion durable pertenece al workflow y a los conectores superiores.
- `create_microtask` normaliza `decision.phase_id` a `planificacion_microtareas`
  cuando el director externo copia por error la fase objetivo de la tarea. La
  fase canonica de ejecucion queda en `create_microtask.task.phase_id`.
- `create_microtask.v0` se normaliza a `create_microtask` cuando el payload
  `create_microtask` esta presente. Es una tolerancia del adaptador de entrada
  para agentes externos; el contrato validado que sale del modulo sigue usando
  el command type canonico.
- La lectura tolera comas finales antes de `}` o `]` en JSON emitido por
  agentes. Es una relajacion de borde para no perder planes completos por un
  error mecanico de serializacion; las decisiones siguen pasando por validacion
  de contrato despues de decodificar.
- Para `request_vote`, `accept_decision` y `publish_function_contract` el
  adaptador puede hidratar el payload canonico desde campos planos del mismo
  objeto si el agente omitio el contenedor anidado. Es compatibilidad de entrada;
  la salida validada conserva los payloads canonicos.
- El request del proveedor de ficheros incluye `phase_artifacts` y `deliveries`
  del run. El source no decide causalidad, pero entrega esas proyecciones a
  conectores superiores para que puedan impedir que un sidecar de decisiones se
  consuma antes del ACK que lo produjo.
- Aunque el descriptor no declare `run_id`, las decisiones leidas se filtran
  por `decision.run_id` cuando la request trae run. Asi un artefacto descubierto
  por referencia opaca no puede inyectar decisiones de otro run.
