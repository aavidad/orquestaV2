---
name: orquesta-programacion-minima
description: Programar con diff minimo y sin overengineering en Orquesta. Usar en tareas de codigo, bugfix, refactor, app generada o revision cuando haya riesgo de crear helpers, capas, interfaces, archivos, tests, logs, validaciones, docs, dependencias o abstracciones no necesarias. Prioriza reutilizar codigo existente, stdlib, plataforma y el menor cambio verificable.
---

# Orquesta Programacion Minima

Usa esta skill junto a `orquesta-programacion-autonoma` en cualquier tarea de
programacion.

## Invariante

Minimo no significa negligente. No recortes:

- seguridad;
- validacion en fronteras de confianza;
- manejo de errores que evita perdida de datos;
- accesibilidad e i18n si hay UI;
- arquitectura hexagonal cuando el contrato la exige;
- requisitos pedidos explicitamente por el operador.

## Escalera

Antes de escribir codigo, para en el primer punto que resuelva el trabajo:

1. Si no hace falta construirlo, no lo construyas.
2. Si ya existe en el repo, reutilizalo.
3. Si lo cubre la stdlib, usa la stdlib.
4. Si lo cubre la plataforma nativa, usa la plataforma.
5. Si lo cubre una dependencia ya instalada, usala.
6. Si puede ser una linea o un cambio local, haz eso.
7. Solo entonces escribe el minimo codigo nuevo que funciona.

La escalera se aplica despues de entender el flujo real tocado. Un diff pequeno
en el sitio equivocado no es minimo: es otro bug.

## Reglas

- No anadas helpers, interfaces, capas, factories, config, logs, validaciones,
  docs, tests, ficheros o dependencias si no hay necesidad demostrada.
- Necesidad demostrada: requisito explicito, contrato publico, test que falla,
  patron local obligatorio, duplicacion real que ya complica el cambio, frontera
  de seguridad o riesgo de perdida de datos.
- Prefiere editar el fichero existente. Crea fichero nuevo solo si el patron
  local lo exige o evita mezclar responsabilidades.
- No hagas refactors oportunistas ni formateos fuera de las lineas tocadas.
- En bugfixes, busca los callers del punto que vas a tocar y arregla la causa
  comun si existe. Parchar solo el sintoma no es minimo.
- Pruebas: deja la prueba mas pequena que fallaria antes y pasaria despues. No
  montes fixtures, frameworks ni suites nuevas salvo patron local o riesgo real.
- Docs: actualiza solo si cambia contrato, incidencia, inventario de bugs,
  operativa o una instruccion del operador.

## ACK

El cierre del agente debe declarar:

- `changed_files`;
- `new_files_count`;
- `helpers_added`;
- `abstractions_added`;
- `tests`;
- `scope_expansion_reason` si algun contador anterior no es cero.

Si no puede justificar un helper, abstraccion o fichero nuevo, debe retirarlo.

## Medicion

No declares que esta skill mejora el trabajo solo porque el diff parezca mas
pequeno. Para activar la regla como default amplio, compara una muestra A/B con
y sin `skill-ref-orquesta-programacion-minima-v0` y mide:

- tests pasados;
- ficheros tocados y ficheros nuevos;
- lineas anadidas/eliminadas;
- tool calls o pasos de agente si estan disponibles;
- bytes/tokens de prompt anadidos por la skill.

Si baja codigo pero suben fallos, rework o tiempo total, la medida queda como
skill opt-in y no como default.

## Revision

Al revisar, etiqueta complejidad eliminable con:

- `delete`: no aporta al requisito;
- `stdlib`: reemplazable por biblioteca estandar;
- `native`: reemplazable por plataforma;
- `yagni`: flexibilidad especulativa;
- `shrink`: mismo comportamiento con menos codigo.
