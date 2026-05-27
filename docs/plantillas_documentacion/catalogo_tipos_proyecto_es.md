<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# Catálogo inicial de plantillas por tipo de proyecto

> Desarrollado con Orquesta de Alberto Avidad Fernandez.

## Objetivo

Definir el conjunto documental mínimo que Orquesta debería preparar desde la fase inicial según el tipo de proyecto.

Este catálogo está alineado con la política arquitectónica por tipo de proyecto y evita tratar como equivalentes un servicio core, una herramienta operativa puntual o un controlador de infraestructura.

## Piezas base del catálogo

- `README`: visión rápida, propósito, alcance y mapa documental
- `manual_usuario`: uso funcional para personal no técnico o mixto
- `manual_desarrollador`: arquitectura, desarrollo local, contratos y calidad
- `manual_sysadmin`: operación de plataforma, configuración y continuidad
- `guia_instalacion`: instalación inicial o puesta en marcha en local / preproducción
- `guia_despliegue`: despliegue controlado entre entornos y salida a producción
- `faq`: dudas recurrentes y resolución rápida
- `ayuda_contextual`: microayudas por pantalla, flujo, estado o acción
- `runbook_operativo`: respuesta operativa ante incidencias y tareas recurrentes
- `pruebas_documentales`: evidencia compacta de validaciones ejecutadas o
  revisiones documentales del proyecto generado
- `pendientes`: huecos reales, decisiones diferidas y trabajo posterior del
  proyecto generado

Alias historico: `manual_sistemas_deploy` no es una pieza raiz nueva; se
resuelve como `manual_sysadmin` mas `guia_despliegue` cuando operacion y
despliegue aplican.

## Reglas de selección

- si el proyecto tiene interfaz de usuario real, incluir `manual_usuario` y `ayuda_contextual`
- si el proyecto se despliega en más de un entorno o tiene paso a producción, incluir `guia_despliegue`
- si la instalación no es trivial o requiere dependencias previas, incluir `guia_instalacion`
- si existe operación continuada o guardias, incluir `runbook_operativo`
- si hay separación entre desarrollo y operación, incluir `manual_sysadmin`
- si terceros ampliarán o integrarán el sistema, incluir `manual_desarrollador`
- si el director exige cierre con evidencia documental, incluir
  `pruebas_documentales`
- si quedan huecos reales o decisiones diferidas, incluir `pendientes`

## Mínimos por tipo de proyecto

### 1. Servicios y APIs core

Contexto típico:

- backend principal
- APIs de negocio
- servicios persistentes con evolución funcional

Documentación mínima:

- `README`
- `manual_desarrollador`
- `guia_instalacion`
- `guia_despliegue`
- `runbook_operativo`
- `faq`
- `pruebas_documentales`

Documentación recomendada:

- `manual_sysadmin`
- `manual_usuario` si existe panel o consola de uso funcional
- `ayuda_contextual` si hay interfaz operable por personas
- `pendientes` solo si queda trabajo verificable

### 2. Scripts y herramientas operativas rápidas

Contexto típico:

- utilidades internas
- migraciones asistidas
- herramientas de soporte o mantenimiento
- automatizaciones con alcance acotado

Documentación mínima:

- `README`
- `guia_instalacion`
- `faq`
- `runbook_operativo`
- `pruebas_documentales` si hay ejecucion asistida o migracion

Documentación recomendada:

- `manual_desarrollador` si la herramienta va a evolucionar o recibir contribuciones
- `manual_usuario` si la usan perfiles no técnicos
- `guia_despliegue` si se publica fuera del entorno local del autor
- `pendientes` si quedan riesgos o tareas diferidas

### 3. Controladores de infraestructura

Contexto típico:

- integraciones con proveedores
- puentes con SDKs o APIs externas
- automatización de despliegue o aprovisionamiento

Documentación mínima:

- `README`
- `manual_desarrollador`
- `guia_despliegue`
- `manual_sysadmin`
- `runbook_operativo`
- `faq`
- `pruebas_documentales`

Documentación recomendada:

- `guia_instalacion` si puede probarse o arrancarse en local
- `manual_usuario` y `ayuda_contextual` solo si añade una interfaz real de operación humana
- `pendientes` si quedan integraciones o permisos por cerrar

## Combinaciones sugeridas

### Perfil mínimo de proyecto pequeño

- `README`
- `guia_instalacion`
- `faq`

### Perfil profesional estándar

- `README`
- `manual_desarrollador`
- `guia_instalacion`
- `guia_despliegue`
- `faq`
- `runbook_operativo`
- `pruebas_documentales`

### Perfil con interfaz funcional

- `README`
- `manual_usuario`
- `ayuda_contextual`
- `faq`
- resto de piezas técnicas según el tipo de proyecto

## Criterio de calidad

La existencia de una plantilla no obliga a rellenarla con contenido artificial. Si una pieza no aplica, debe indicarse explícitamente en el `README` del proyecto en lugar de dejar un documento vacío o engañoso.
