# OP-121 F1 - Catalogo base de skills, versionado y anti-duplicado

## Estado

Fase 1 implementada para `OP-121`.

Objetivo ya cubierto:

- indice ordenado de skills por rol
- metadata minima normalizada
- versionado y auditoria de cambios
- estrategia de equivalencia para evitar duplicados funcionales

Queda fuera de este documento:

- deteccion automatica de skill faltante y llamada al creador de skills (`#377`)
- descubrimiento en caliente y refresh dinamico (`#376`)
- seguridad fina para herramientas externas (`#375`)

## Contrato base del catalogo de skills

Cada skill persistida en Orquesta usa este minimo comun:

- `tipo_agente`
  Rol objetivo. Ej.: `programador`, `documentador`, `admin`.
- `nombre`
  Identificador visible de la skill.
- `descripcion`
  Resumen corto de la capacidad.
- `cuando_usar`
  Criterio funcional de activacion.
- `escenario`
  Etiqueta breve del escenario principal de uso.
- `prioridad`
  Orden de descubrimiento. Menor valor = antes.
- `aliases_json`
  Alias funcionales normalizados.
- `herramientas_json`
  Herramientas externas asociadas o requeridas.
- `activa`
  Estado operativo.

## Normalizacion canonica

Antes de crear o actualizar una skill, Orquesta normaliza:

- espacios laterales en todos los campos de texto
- `escenario` a minusculas
- `prioridad` a `100` si viene vacia o no positiva
- `aliases_json` y `herramientas_json` a JSON canonico

La normalizacion de listas aplica estas reglas:

- elimina entradas vacias
- baja a minusculas
- elimina duplicados
- ordena alfabeticamente
- serializa siempre como JSON array

Ejemplo:

- entrada: `[" RG ", "ripgrep", "rg", ""]`
- salida canonica: `["rg","ripgrep"]`

## Orden del indice

Las lecturas del catalogo ya salen ordenadas por:

1. `prioridad ASC`
2. `escenario ASC`
3. `nombre ASC`

Ese es el orden base para briefing, API y CLI.

## Equivalencia y anti-duplicado

La fase 1 no se limita al `UNIQUE(tipo_agente, nombre)`.
Tambien bloquea skills funcionalmente equivalentes.

Dos skills se consideran equivalentes si se cumple alguno de estos casos:

1. comparten una clave canonica de identidad entre `nombre` y `aliases`
2. comparten `escenario`, la misma lista canonica de `herramientas_json` y el mismo `cuando_usar` normalizado

La clave canonica:

- pasa a minusculas
- elimina separadores comunes: espacios, guiones, `_`, `/`, `\`, `.`

Ejemplos equivalentes:

- `ripgrep`, `rip-grep`, `rip_grep`
- `sql lint`, `sql-lint`

Consecuencia operativa:

- si ya existe una skill equivalente, Orquesta rechaza la nueva alta
- si al editar una skill esta pasa a colisionar con otra, Orquesta rechaza la actualizacion
- si la equivalencia detectada es la misma fila natural (`tipo_agente + nombre`), la operacion se trata como actualizacion legitima

## Versionado y auditoria

Toda mutacion relevante de skills registra version en `skills_versiones` con:

- snapshot de campos funcionales
- actor
- accion
- numero de version

Hoy se versiona al menos en:

- crear
- actualizar
- activar/desactivar

Y ademas queda traza en auditoria general.

## Superficie ya disponible

CLI:

```bash
orquesta skills listar --rol programador
orquesta skills crear --rol programador --nombre rg --cuando-usar "buscar texto rapido" --escenario busqueda --prioridad 10 --alias ripgrep --herramienta rg
orquesta skills editar 12 --escenario analisis --prioridad 20 --alias ripgrep --alias rg
orquesta skills versiones 12
```

API:

- `GET /api/skills`
- `POST /api/skills`
- `GET /api/skills/{id}`
- `POST /api/skills/{id}`
- `POST /api/skills/{id}/activa`
- `GET /api/skills/{id}/versiones`

Campos ya soportados por API para crear/editar:

- `tipo_agente`
- `nombre`
- `descripcion`
- `cuando_usar`
- `escenario`
- `prioridad`
- `aliases_json`
- `herramientas_json`
- `activa`

## Garantias para F2

`#377` puede apoyarse ya en estas garantias:

- existe un indice estable y ordenado
- la metadata minima ya esta definida
- las equivalencias ya tienen canonica comun
- el orquestador ya evita crear la misma skill varias veces
- las mutaciones de catalogo ya dejan historial

## Archivos base

- `db/skills_catalog.go`
- `db/reglas.go`
- `db/catalogo_versionado.go`
- `db/schema.go`
- `db/post_migraciones_compat.go`
- `cmd/catalogo_briefing.go`

## Semillas de skills por composicion

Regla general: si un contrato se repite como procedimiento para agentes, se
plasma tambien como skill. El contrato mantiene la forma tecnica; la skill
explica al agente como trabajar sin cargar todo el contexto.

### Orquesta transversal

Estas skills viven en este repo y valen para cualquier consumidor:

- `skills/orquesta-director-agentes/SKILL.md`
  Direccion de agentes/subagentes, revisores, votacion, rework y cierre.
- `skills/orquesta-ordenacion-trabajo/SKILL.md`
  Ordenacion transversal de objetivo, backlog, plan, olas, tareas, agentes,
  artefactos, revisiones, pruebas y cierre.
- `skills/orquesta-programacion-autonoma/SKILL.md`
  Programacion, arreglos, pruebas, limpieza de worktree y cierre con evidencia.
- `skills/orquesta-programacion-minima/SKILL.md`
  Programacion con diff minimo, sin helpers, capas, ficheros, dependencias ni
  abstracciones no necesarias, conservando seguridad, pruebas y contratos.
  Estado: opt-in hasta comparar A/B en golden tasks.
- `skills/orquesta-programacion-integracion/SKILL.md`
  Integraciones hexagonales, puertos, adaptadores opt-in y configuracion.
- `skills/orquesta-programacion-tests/SKILL.md`
  Pruebas focales, suite completa, fallos reparables y evidencia.
- `skills/orquesta-programacion-revision/SKILL.md`
  Revision tecnica, arquitectura, producto/UI, tests y rework causal.
- `skills/orquesta-programacion-web-local/SKILL.md`
  Validacion web local con i18n, responsive, capturas y assets.
- `skills/orquesta-programacion-web-administrativa/SKILL.md`
  Webs administrativas densas para portales de empleado, Bolsa/VEC, expedientes,
  autobaremacion, listados, alegaciones, auditoria y muchos datos visibles.
- `skills/orquesta-programacion-release/SKILL.md`
  Worktree limpio, archivado, commit, push y despliegue seguro.
- `skills/orquesta-revision-consejo-votacion/SKILL.md`
  Debate, critica, voto y decision del Director sobre candidatos.
- `skills/orquesta-artefacto-modular/SKILL.md`
  Artefactos enchufables para cualquier app: juegos, tutor, paneles, ayudas,
  web, assets o paquetes.
- `skills/orquesta-runtime-modelos/SKILL.md`
  Gestion de modelos locales/cloud por puerto runtime opt-in.
- `skills/orquesta-supuestos-practicos-opes/SKILL.md`
  Coordinación de agentes Orquesta para crear, reutilizar, revisar y cerrar
  supuestos prácticos OPES sin meter canon editorial OPES en el núcleo.

### OPES como consumidor independiente

Las skills de cursos OPES no viven dentro del nucleo Orquesta. Estan en:

`/home/alberto/Trabajo/OPES/skills/`

Familias vigentes:

- director Orquesta para OPES;
- inventario y reutilizacion;
- investigacion de bases;
- redaccion de temarios;
- corrector ortografico;
- tests de 4 respuestas;
- supuestos prácticos y supuestos tipo test;
- infografias y revision visual;
- audio edge-tts + Whisper;
- RAG, tutor y bots;
- juegos y retos;
- HTML web USO;
- QA final y paquete/API de produccion.

### Programacion como consumidor

Para programacion no se usan reglas OPES. El agente debe usar
`orquesta-director-agentes`, `orquesta-ordenacion-trabajo`,
`orquesta-programacion-autonoma`, `orquesta-programacion-integracion`,
`orquesta-programacion-tests`, `orquesta-programacion-web-local`,
`orquesta-programacion-release`, `orquesta-revision-consejo-votacion`,
`orquesta-artefacto-modular` y las reglas locales del repo objetivo. Puede
anadir `orquesta-programacion-minima` como opt-in medido para comparar diff y
coste. Cualquier skill especifica de un repo de programacion debe vivir en ese
repo o en una composicion propia, no en OPES ni en el nucleo generico.
