<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# Plantilla base de i18n para proyectos

## Objetivo

Definir el contrato minimo que Orquesta debe dejar preparado cuando un proyecto nazca con i18n obligatorio.

No cubre renderizado final ni detalles de framework. Cubre estructura, semillas y reglas minimas para no seguir creando proyectos sin cimiento multilenguaje.

## Estructura esperada

```text
i18n/
  config.json
  README.md
  es/
    common.json
    navigation.json
    actions.json
    validation.json
    errors.json
  en/
    common.json
    navigation.json
    actions.json
    validation.json
    errors.json
```

## Reglas del contrato

- `es` es el idioma por defecto inicial salvo politica explicita distinta
- el pack inicial previsto es `es`, `en`, `de`, `fr`, `it`, `zh`, `gl`, `eu`, `ca` y `val`, salvo politica mas restrictiva
- cada idioma vive en su propia subcarpeta
- cada dominio funcional vive en un fichero distinto
- las claves deben ser estables y con namespace
- si falta una clave en un idioma secundario, el fallback inicial es el idioma por defecto

## Contrato canonico y compatibilidad

- el contrato canonico aprobado es `i18n/config.json` mas `i18n/<lang>/<domain>.json`
- `i18n/bundle.go` debe considerar ese formato como la referencia viva
- los ficheros legacy planos `i18n/<lang>.json` se mantienen solo por compatibilidad con bundles antiguos de Orquesta
- cuando convivan ambos formatos, el objetivo es preservar compatibilidad sin volver a declarar el formato plano como estructura recomendada para proyectos nuevos

## Ficheros base

- `common.json`: textos transversales de app y estados genericos
- `navigation.json`: menu, breadcrumbs y rotulos de navegacion
- `actions.json`: botones y acciones reutilizables
- `validation.json`: mensajes de validacion y formularios
- `errors.json`: errores de red, permisos, autenticacion y fallos genericos

## Semilla minima recomendada

Cada dominio debe nacer al menos con:

- un titulo o bloque base
- estados principales
- acciones comunes
- errores genericos
- validaciones de formulario frecuentes

## Soporte en Orquesta

La implementacion base de esta tarea queda materializada en el comando:

```bash
orquesta lenguaje esqueleto <ruta-proyecto>
```

Ese comando crea la estructura `i18n/`, el `config.json`, el `README.md` y las semillas iniciales por idioma y dominio.
