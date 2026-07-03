# Tarea OPES Orquesta: Supervisor Vivo Pero No Dirige Cierre De Temarios

Fecha: 2026-06-25.

## Contexto

Durante el cierre/auditoria de temarios OPES se detecto que el trabajo volvio a
ejecutarse como reparacion directa desde Codex para rematar audio/validaciones
de Operario AP, en vez de quedar gobernado de forma clara por Orquesta.

El estado inmediato de Orquesta era:

```text
api: http://127.0.0.1:8787
servidor: running
supervisor: stalled ticks=22578 ejecuciones=43057 cola=4
autoprogramacion: ok tareas_cola=2 activas=2
procesos_locales: servidor=0 codex_padre=0 codex_vendor=0
cola_top:
  - running cron-autonomy-review-2026-05-27
  - running cron-autonomy-review-1779899631
```

## Problema

Para OPES no basta con que el servidor este vivo. El director necesita una
respuesta operativa rapida:

- que trabajos OPES estan vivos;
- que trabajos estan colgados;
- que trabajos estan realmente en cola;
- que tareas han entregado ACK util;
- que tareas deben relanzarse por Orquesta;
- que reparaciones directas deben registrarse como excepcion y volver al flujo
  Orquesta.

En esta observacion el supervisor aparece vivo pero `stalled`, con cola de
autoprogramacion activa y sin procesos locales. Eso invita a que el director
humano/Codex rodee el bloqueo con comandos directos, lo que rompe la regla OPES
de Orquesta como sistema principal.

## Resultado Esperado

Orquesta debe ofrecer un comando/API de estado OPES que devuelva en menos de
unos segundos:

- estado del servidor y supervisor;
- jobs OPES activos, terminados, colgados y en espera;
- agentes padre e hijos reales asociados;
- ultima entrega/ACK por job;
- causa de bloqueo si existe;
- accion recomendada: esperar, drenar, relanzar, cerrar, rework o escalar.

Si el supervisor esta `stalled`, Orquesta debe exponer una accion segura y
documentada para recuperar la direccion sin perder entregas utiles.

## Criterios De Aceptacion

- Un operador puede saber si OPES esta trabajando, parado o colgado sin leer
  logs manualmente.
- Las reparaciones directas quedan registradas como excepcion y no como camino
  normal de produccion.
- El estado OPES distingue trabajos productivos de cron/autoprogramacion no
  relacionados.
- El flujo de cierre de temarios a medias vuelve a lanzar o supervisar tareas
  por Orquesta en vez de depender de scripts directos.

## No Hacer

- No meter reglas editoriales OPES en el nucleo generico de Orquesta.
- No resolver con filtros por palabras ni rails estrechos.
- No descartar entregas utiles por estar fuera de formato si son recuperables.
