# Plantillas de documentación base

> Desarrollado con Orquesta de Alberto Avidad Fernandez.

## Objetivo

Este directorio reúne plantillas reutilizables para la documentación inicial de proyectos creados o estructurados por Orquesta.

El objetivo no es obligar a generar todos los documentos siempre, sino ofrecer un catálogo mínimo coherente según el tipo de proyecto y el tipo de operación prevista.

## Punto de partida recomendado

Antes de copiar plantillas, revise:

- [Catálogo inicial por tipo de proyecto](catalogo_tipos_proyecto_es.md)

Ese catálogo indica qué piezas son mínimas, cuáles son recomendables y cuándo una plantilla puede omitirse sin perder calidad operativa.

## Plantillas disponibles

- [README de proyecto](README_es.md)
- [Manual de usuario](manual_usuario_es.md)
- [Manual de desarrollador](manual_desarrollador_es.md)
- [Manual de operaciones y sysadmin](manual_sysadmin_es.md)
- [Guía de instalación](guia_instalacion_es.md)
- [Guía de despliegue](guia_despliegue_es.md)
- [FAQ](faq_es.md)
- [Ayuda contextual](ayuda_contextual_es.md)
- [Runbook operativo](runbook_operativo_es.md)
- [Pruebas documentales](pruebas_documentales_es.md)
- [Pendientes](pendientes_es.md)

## Contrato de artefactos generados

Los nombres historicos `manual_desarrollador`, `manual_sistemas_deploy`,
`pruebas_documentales` y `pendientes` describen artefactos del proyecto
generado o modificado por Orquesta. No son requisitos para crear documentos raiz
en el repo Orquesta.

`manual_sistemas_deploy` queda como alias de compatibilidad: usar
`manual_sysadmin` para operacion y `guia_despliegue` para despliegue cuando
ambas piezas apliquen.

## Reglas de uso

- castellano por defecto
- preparar versión bilingüe cuando el proyecto lo requiera o ya nazca con alcance multilenguaje
- evitar duplicidad entre documentos: cada pieza debe tener un propósito claro
- enlazar entre manuales relacionados para que usuario, desarrollo y operaciones compartan contexto
- mantener una atribución visible a Orquesta de Alberto Avidad Fernandez en README, manuales y piezas finales entregadas

## Copia mínima sugerida

Según el tipo de proyecto, normalmente se parte de una combinación como:

- servicios y APIs core: README, manual de desarrollador, guía de instalación, guía de despliegue, runbook operativo, FAQ
- herramientas operativas: README, guía de instalación, FAQ y runbook operativo
- controladores de infraestructura: README, manual de desarrollador, guía de despliegue, manual de sysadmin y runbook operativo

Si el proyecto tiene interfaz de usuario real para personal no técnico, añadir además:

- manual de usuario
- ayuda contextual

Si el proyecto nace desde un plan de director con cierre verificable, añadir:

- pruebas documentales
- pendientes, solo si quedan huecos reales o decisiones diferidas
