# orquesta-web

Responsabilidad: interfaz primaria para pedir apps, revisar progreso y operar acciones seguras.

Incluye:

- nueva app;
- panel de agentes;
- panel de fases;
- revision de propuestas;
- acciones seguras;
- progreso compacto.

La web es adaptador inbound. No decide negocio ni accede a DB directamente.

El formulario `/app-change` puede enviar refs opacas de `external_work` para
que Orquesta coordine trabajos de una app externa sin importar su nucleo ni
acoplarse a sus bases de datos, runtime, proveedor o contratos internos.
