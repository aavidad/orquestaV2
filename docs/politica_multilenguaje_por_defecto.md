<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# Política multilenguaje por defecto donde aplique

## Objetivo

Dejar por escrito la política aprobada en Orquesta para decidir cuándo un proyecto debe nacer con base multilenguaje y cuándo no.

Esta política no impone i18n como ritual universal.
La regla acordada es más precisa:

- multilenguaje por defecto donde haya interfaz o consumo humano real
- excepciones explícitas para piezas técnicas sin superficie funcional equivalente

## Base de decisión

La política vigente sale de:

- `OP-065`, aprobada en consenso
- `OP-066`, rechazada como formulación duplicada y demasiado rígida
- la implementación de política y matriz en `cmd/lenguaje.go`
- el contrato técnico de esqueleto i18n en `docs/plantillas_i18n/README_es.md`
- la materialización de ese esqueleto en `i18n/project_skeleton.go`

## Regla principal

Un proyecto debe arrancar con base multilenguaje si su superficie principal contiene texto de consumo humano que vaya a vivir en producto, documentación operativa o interfaz visible.

Aplica por defecto a:

- apps con interfaz de usuario
- paneles web
- clientes desktop
- documentación que va a consumirse como parte del producto o de la operación normal
- flujos guiados, asistentes y formularios
- mensajes visibles para usuarios, operadores o administradores

No aplica por defecto a:

- scripts puntuales
- utilidades internas sin interfaz funcional real
- piezas puramente técnicas o de infraestructura
- librerías sin superficie textual de producto
- componentes operativos efímeros donde el coste de i18n no tiene retorno real

## Qué significa “por defecto”

“Por defecto” no significa “obligatorio en cualquier contexto”.
Significa esto:

1. si el proyecto encaja en una superficie con consumo humano claro, Orquesta debe asumir i18n desde el inicio
2. si el proyecto no encaja ahí, la carga de justificar i18n desaparece y puede omitirse
3. si hay una excepción, debe quedar explícita en la política o en la matriz de selección

La intención es evitar dos errores simétricos:

- crear apps visibles con texto hardcodeado y deuda de localización desde el día 1
- imponer sobrecoste artificial a herramientas técnicas donde no aporta valor

## Idioma inicial y fallback

La política actual mantiene estas reglas base:

- el idioma inicial por defecto es castellano salvo política explícita distinta
- el fallback inicial debe estar definido desde el arranque
- las apps y la documentación con i18n deben nacer con contrato estructurado, no con cadenas sueltas

En el esqueleto i18n actual:

- `default_language` sale de la política o del valor explícito elegido
- `fallback_language` cae al idioma por defecto si no se fija otro
- el pack inicial de idiomas previsto incluye `es`, `en`, `de`, `fr`, `it`, `zh`, `gl`, `eu`, `ca` y `val`

## Integración en Orquesta

Orquesta ya contempla la política multilenguaje como dato configurable, no como simple texto doctrinal.

La política global de lenguaje distingue, al menos:

- idioma por defecto global
- si la documentación es multilenguaje
- si las apps son multilenguaje
- idioma por defecto de documentación
- idioma por defecto de apps
- idiomas permitidos

Además existe una matriz de selección para fijar idioma efectivo por:

- proyecto
- tarea
- contexto

Eso permite que la política sea global sin impedir excepciones justificadas.

## Regla operativa para proyectos nuevos

Cuando Orquesta genere o prepare un proyecto:

1. identificar si el tipo de proyecto tiene interfaz o documentación de consumo humano
2. aplicar la política global
3. si procede, materializar el esqueleto i18n desde el inicio
4. si no procede, dejar constancia de la excepción y no inventar i18n artificial

La decisión correcta no es “todo traducido siempre”.
La decisión correcta es “i18n desde el inicio cuando la naturaleza del proyecto lo pide”.

## Relación con el esqueleto i18n

La política multilenguaje por defecto no se queda en un criterio abstracto.
Se aterriza en un contrato mínimo:

- directorio `i18n/`
- `config.json`
- `README.md`
- subdirectorios por idioma
- ficheros por dominio funcional
- fallback de claves al idioma por defecto

Ese contrato evita seguir creando proyectos con:

- textos hardcodeados en vistas o componentes
- localización improvisada por framework
- mezcla de mensajes de distintos módulos sin dominios claros

## Excepciones aceptables

Una excepción es válida cuando el proyecto:

- no tiene interfaz funcional real
- no expone textos de producto a usuarios u operadores
- es una utilidad efímera o muy acotada
- tendría un coste de i18n claramente desproporcionado respecto a su alcance

La excepción no debe ser implícita.
Debe quedar razonada en la definición del proyecto o en la matriz de lenguaje.

## Qué no debe hacerse

- no declarar “multilenguaje obligatorio” para todo sin distinguir tipo de proyecto
- no crear apps visibles con castellano hardcodeado “para luego traducir”
- no usar i18n como fachada si no existe contrato de claves, dominios y fallback
- no duplicar la política entre documentación, código y app sin fuente de verdad compartida

## Criterio de aceptación

La política está bien aplicada si:

- las apps y docs con interfaz o consumo humano nacen con base i18n cuando procede
- las piezas técnicas sin superficie funcional no cargan con i18n artificial
- la excepción queda explícita y trazable
- el idioma inicial y el fallback están resueltos desde el arranque
- Orquesta puede materializar el esqueleto i18n sin decisiones ad hoc de última hora
