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
