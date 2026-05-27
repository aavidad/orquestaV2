# orquesta-web

Responsabilidad: interfaz primaria para pedir apps, revisar progreso y operar acciones seguras.

Incluye:

- nueva app;
- panel de agentes;
- panel de fases;
- panel de control de runs;
- panel de cola multiapp;
- revision de propuestas;
- acciones seguras;
- progreso compacto.
- uso agregado saneado cuando `director/stats` lo publica con opt-in.

La web es adaptador inbound. No decide negocio ni accede a DB directamente.
Para T209, la web no lee logs ni runtime: solo proyecta `usage_summary` y
`agent.usage` ya saneados. Si hay senales vivas sin reporte, muestra
`unknown`/`unavailable` con reason code publico, no `-` ni detalles privados.

Para T210, la web consume `DirectorRunStatsV0`: si stats trae progreso vivo por
entrega, agente o proceso registrado, no debe degradarlo a 0% por fallback
visual. `tasks_closed` sigue siendo cierre aceptado por el nucleo. La politica
de cache/frescura de `/ops` pertenece a T211.

El formulario `/app-change` puede enviar refs opacas de `external_work` para
que Orquesta coordine trabajos de una app externa sin importar su nucleo ni
acoplarse a sus bases de datos, runtime, proveedor o contratos internos.
