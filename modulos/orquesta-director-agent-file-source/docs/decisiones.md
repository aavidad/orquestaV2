# Decisiones

- El adaptador no escanea rutas. Un conector superior decide que artefactos estan pendientes.
- La lectura se abstrae con puerto para poder usar filesystem, memoria, MCP, REST o cualquier almacenamiento externo.
- El limite de lectura existe en el adaptador, no en el nucleo.
- Los artefactos pueden contener varias decisiones para permitir que un director entregue un plan compacto de fases.
- No se guarda estado de procesado aqui. La deduplicacion durable pertenece al workflow y a los conectores superiores.
