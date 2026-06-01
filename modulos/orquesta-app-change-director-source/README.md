# orquesta-app-change-director-source

Fuente hexagonal que lee `AppChangeRecordV0` aceptados y, cuando la solicitud
ya tiene write-set y criterios verificables, emite decisiones del director para
replanificar trabajo sin conocer la implementacion interna del stack.
El origen del record puede ser una solicitud directa o un evento de cambio ya
normalizado por `orquesta-app-change`.
Si el record trae `external_work`, proyecta el trabajo como contrato de dominio
externo sin conocer la app propietaria.
En trabajos documentales externos, como un `draft_content_block` de OPES,
la microtarea declara que el agente debe trabajar con un paquete de dominio
suficiente y no con contexto minimo: temario, esquema, objetivo, posicion,
vecinos, fuentes, criterios y longitud si llegan por contrato.
Para OPES, esa microtarea es una unidad durable de workflow, no necesariamente
un trozo pequeno: la redaccion de temarios largos debe mantenerse como bloque,
subcapitulo o capitulo coherente. Los trabajos `summarize_*` si pueden ser
pequenos porque son derivados y trazables.
Los trabajos de audio como `generate_audio_asset` se proyectan como trabajo
documental externo con `audio_asset`, manifest y refs opacas; el motor de audio
queda en la app propietaria.
Cuando el cambio ya tiene microtarea entregada y las entregas de programacion
cubren las tareas abiertas, emite la decision compacta para abrir `revision`;
el review gate generico valida la entrega despues.

No ejecuta agentes ni toca ficheros de app. Solo genera decisiones compactas.
