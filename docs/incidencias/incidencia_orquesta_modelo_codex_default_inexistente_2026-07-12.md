# Incidencia: modelo Codex por defecto inexistente

Fecha: 2026-07-12.

## Sintoma

Dos goals reales aceptaron `thread/start`, `goal/set` y `turn/start`, pero el
turn termino en menos de un segundo con `last_agent_message=null`, sin comandos,
cambios ni error publico. Orquesta proyecto despues `goal_status=blocked`.

## Causa

El routing estricto por defecto resolvia codigo normal a `gpt-5.6-terra` y
documentacion a `gpt-5.6-luna`. Esos identificadores no figuraban en el catalogo
del proveedor disponible en el runner aislado. El app-server aceptaba el ajuste
del thread, pero cerraba el turn vacio.

## Correccion

Los refs opacos de politica se conservan. Solo cambia su resolucion por defecto:

- `luna` -> `gpt-5.4-mini`, esfuerzo `low`;
- `terra` -> `gpt-5.5`, esfuerzo `medium` o `high` segun la decision;
- `sol` -> `gpt-5.5`, esfuerzo `high`.

Una configuracion explicita de proyecto sigue prevaleciendo. No se anaden envs,
permisos ni relajaciones de sandbox.

## Cierre requerido

Reejecutar suites focales, guard raiz, reconstruir el runner con el commit
canonico y relanzar el goal real que detecto la incidencia. El cierre exige un
turn con trabajo material, tests atestiguados y review independiente; aceptar
solo `turn/start` no basta.
