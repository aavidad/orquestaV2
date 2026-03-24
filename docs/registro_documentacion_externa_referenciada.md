<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# Registro de documentación externa referenciada

## Objetivo

Definir el criterio y el uso correcto del registro de documentación externa referenciada por proyecto en Orquesta.

La finalidad del registro no es copiar documentos externos dentro de la base de datos.
La finalidad es dejar trazabilidad mínima sobre qué fuente externa se consultó, por qué importa y con qué tarea o propuesta queda relacionada.

## Base funcional

La capacidad ya existe en Orquesta y se apoya en:

- la entidad `DocumentoExterno` en `db/project_memory.go`
- la persistencia en la tabla `documentos_externos`
- la vista web de proyecto en `cmd/proyectos_web.go`
- la API `GET/POST /api/proyectos/{slug}/documentacion`
- la memoria de proyecto agregada por `memoriaproyecto/service.go`

## Qué guarda el registro

Cada entrada de documentación externa guarda, como mínimo:

- proyecto
- tipo de documento
- título
- ruta o referencia externa
- resumen
- estado
- fuente
- propuesta relacionada opcional
- tarea relacionada opcional
- metadatos JSON opcionales

Campos obligatorios reales:

- `proyecto_id`
- `titulo`
- `ruta_ref`
- `resumen`

Valores por defecto si no se indican:

- `tipo_documento = markdown`
- `estado = vigente`
- `fuente = manual`

## Qué no guarda

El registro no debe usarse para:

- duplicar el contenido completo del documento externo dentro de Orquesta
- convertir la BD en almacén de PDFs, markdowns o webs copiadas
- sustituir una decisión o una propuesta por una simple referencia
- esconder la justificación real en un enlace sin resumen

La regla práctica es:

- Orquesta guarda la referencia y el contexto
- el contenido externo sigue viviendo fuera, en su sistema de origen

## Cuándo registrar una fuente externa

Conviene registrar una referencia cuando:

- una decisión de proyecto se apoya en documentación externa concreta
- una propuesta cita una norma, ADR, guía o diseño fuera de Orquesta
- una tarea depende de un documento técnico, legal u operativo externo
- hace falta dejar trazabilidad de qué se consultó durante análisis o implementación

No conviene registrarla cuando:

- la referencia es trivial y no afecta a decisiones ni a trabajo real
- no se puede resumir por qué importa
- la fuente no está mínimamente identificada

## Criterio de calidad de la entrada

Una entrada es útil si responde al menos a estas preguntas:

1. qué documento es
2. dónde está
3. por qué importa para este proyecto
4. si sigue vigente o ya quedó archivado
5. con qué tarea o propuesta se relaciona

Por eso `ruta_ref` sin `resumen` no basta.

## Estados y fuentes permitidos

Estados válidos actuales:

- `vigente`
- `borrador`
- `archivado`

Fuentes válidas actuales:

- `manual`
- `propuesta`
- `tarea`
- `externo`

Interpretación recomendada:

- `manual`
  entrada añadida manualmente por operador o agente
- `propuesta`
  referencia nacida al hilo de una propuesta
- `tarea`
  referencia ligada al trabajo de una tarea concreta
- `externo`
  fuente externa pura no originada dentro de Orquesta

## Regla de deduplicación

La implementación actual resuelve unicidad por:

- `proyecto_id`
- `ruta_ref`

Eso significa:

- dos proyectos distintos pueden registrar la misma fuente
- dentro del mismo proyecto, la misma `ruta_ref` se actualiza en lugar de duplicarse

Consecuencia operativa:

- usar `ruta_ref` estable y canónica
- no registrar la misma fuente con rutas ligeramente distintas si en realidad es el mismo documento

## Uso web

En la vista de detalle de proyecto existe un bloque de “Documentación externa”.

Desde ahí se puede:

- listar referencias ya registradas
- crear una nueva referencia
- asociarla opcionalmente a propuesta o tarea

Campos visibles en la UI:

- título
- tipo de documento
- ruta o referencia
- resumen
- estado
- fuente
- propuesta relacionada
- tarea relacionada

## Uso por API

Endpoints disponibles:

- `GET /api/proyectos/{slug}/documentacion`
- `POST /api/proyectos/{slug}/documentacion`

Payload mínimo de creación:

```json
{
  "titulo": "ADR-001",
  "ruta_ref": "/ruta/o/referencia",
  "resumen": "Decision base del proyecto"
}
```

Payload completo posible:

```json
{
  "tipo_documento": "markdown",
  "titulo": "ADR-001",
  "ruta_ref": "/docs/adr/001.md",
  "resumen": "Define la frontera entre API y persistencia",
  "estado": "vigente",
  "fuente": "propuesta",
  "propuesta_id": 114,
  "tarea_id": 340,
  "metadata_json": "{\"origen\":\"workspace\"}"
}
```

## Uso recomendado por agentes

Flujo recomendado:

1. detectar que una fuente externa influye en el proyecto
2. registrar referencia y resumen
3. enlazarla a tarea o propuesta si aplica
4. usar decisión o propuesta para explicar la conclusión, no la referencia sola

La referencia externa complementa:

- el historial de votaciones
- el registro de decisiones
- la trazabilidad del proyecto

No los sustituye.

## Buenas prácticas

- resumir en una o dos frases por qué la fuente importa
- usar una ruta o URL estable
- marcar como `archivado` lo que ya no rige
- relacionar con `propuesta_id` o `tarea_id` cuando exista vínculo real
- usar `metadata_json` solo para contexto adicional, no para esconder información esencial

## Qué no debe hacerse

- no meter el contenido entero del documento en `resumen`
- no usar el registro como cajón de enlaces sin criterio
- no registrar una fuente sin explicar su utilidad
- no reemplazar una decisión del proyecto por “ver enlace”
- no duplicar la misma referencia en el mismo proyecto con rutas inconsistentes

## Criterio de aceptación

El registro está bien usado si:

- la fuente queda identificada de forma estable
- el proyecto conserva trazabilidad sin copiar el contenido externo
- la referencia puede relacionarse con una tarea, propuesta o decisión
- un tercero entiende por qué esa fuente fue relevante sin abrir primero el documento
