# Prueba real: self-observability del director

Fecha: 2026-05-13.

Objetivo: documentar la prueba real de observabilidad operativa del director
sin cambiar codigo productivo. La prueba cubrio el flujo REST, estadisticas con
progreso vivo, deteccion de agentes stalled y parada supervisada del run.

## Alcance probado

- Entrada REST real por `POST /api/v0/apps/director`.
- Consulta de estado por `POST /api/v0/director/stats` con:
  `include_process_refs`, `include_agent_progress` e `include_agent_usage`.
- Progreso vivo de agentes Codex reflejado en stats mientras el run seguia en
  ejecucion.
- Clasificacion operativa de agentes sin avance como `stalled` cuando habia
  proceso registrado pero el progreso no cambiaba.
- Control de ejecucion con stop del run desde el plano de control.
- Confirmacion posterior mediante `agents_stop_confirmed`.

## Criterios de exito observados

- `POST /api/v0/apps/director` devolvio respuesta 2xx con `run_ref` utilizable.
- `POST /api/v0/director/stats` devolvio snapshots repetibles para el mismo
  `run_ref` sin requerir acceso directo a rutas internas del runtime.
- Las stats mostraron progreso vivo por agente: proceso asociado, estado de
  avance y uso agregado cuando estaba disponible.
- Los agentes que seguian vivos pero no avanzaban quedaron visibles como
  stalled/no-signal en la superficie de observabilidad, sin ocultarse como
  completados.
- El stop por run-control pudo solicitar parada sobre agentes en vuelo.
- El ciclo posterior registro `agents_stop_confirmed`, por lo que el stop no
  quedo solo como intencion del API sino como evento confirmado por el runtime.

## Criterios de fallo usados

- Fallo inmediato si `POST /api/v0/apps/director` no devuelve 2xx o no entrega
  `run_ref`.
- Fallo si `/api/v0/director/stats` no puede consultarse de forma repetida para
  el run creado.
- Fallo si las stats no exponen progreso vivo suficiente para diferenciar:
  agente arrancado, agente sin senal, agente stalled y agente parado.
- Fallo si run-control stop devuelve exito superficial pero no aparece
  `agents_stop_confirmed` en la observabilidad posterior.
- Fallo si la prueba obliga a leer rutas internas del runtime para entender el
  estado, porque eso rompe la frontera de conectores y la operacion por API.

## Resultado

La prueba valida la frontera minima de self-observability del director:
operacion por REST, observabilidad por stats, diagnostico de progreso/stalled y
parada confirmada. El resultado es suficiente para operar un run real sin
inspeccion manual de directorios temporales ni acoplamiento a detalles internos
del runtime.

La solucion sigue el criterio hexagonal: la web y los operadores consumen
contratos REST y conectores de runtime; no se introducen dependencias directas a
proveedores concretos ni a rutas fisicas. Los nombres expuestos al usuario
deben mantenerse en la capa de i18n/documentacion, no incrustados como logica
de dominio.

## Hueco pendiente

Queda pendiente cerrar la continuidad entre `continuation` y `deliveries`.
Durante la prueba la observabilidad pudo mostrar progreso, stalled y stop
confirmado, pero aun falta una evidencia end-to-end que conecte de forma
inequivoca:

- decisiones de continuation del director;
- entregas materializadas en `deliveries`;
- reflejo posterior en stats como agentes completados por entrega;
- reanudacion limpia de un run con esa frontera ya persistida.

Hasta cerrar ese hueco, la prueba demuestra control y observabilidad del run,
pero no debe considerarse una validacion completa del contrato de continuacion y
entrega final.
