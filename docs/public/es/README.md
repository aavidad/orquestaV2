# Orquesta

Orquesta coordina trabajo durable mediante proyectos y Goals. El núcleo decide
el ciclo de vida; HTTP, MCP y CLI son adaptadores del mismo registro de
comandos.

## Idioma

El español es el idioma predeterminado y de respaldo. Las respuestas conservan
sin traducir sus identificadores, referencias, hashes, códigos de error y
claves semánticas. Solo cambia el texto destinado a personas.

Las instalaciones actuales incluyen español e inglés. Una variante regional
compatible usa su idioma base; una locale válida pero no instalada usa el
respaldo español. Una etiqueta inválida produce un error explícito.

## Superficies actuales

- La CLI presenta ayuda y errores humanos desde el catálogo.
- MCP presenta instrucciones y descripciones desde el mismo catálogo.
- HTTP conserva el contrato máquina del registro de comandos.
- Esta documentación pública mantiene una pareja equivalente por idioma.

Web, Wizard, notificaciones y prompts consumirán este mismo contrato cuando sus
verticales existan. Su presencia futura no se declara como funcionalidad actual.

## Garantías

El catálogo falla ante claves ausentes, locales sin paridad, claves duplicadas o
variables de formato incompatibles. Plurales, números, moneda, fechas y zonas
horarias se comprueban por locale sin modificar datos de protocolo.
