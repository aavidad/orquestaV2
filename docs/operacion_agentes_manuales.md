<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# Operacion de agentes manuales

## Objetivo

Definir como deben arrancarse y cerrarse agentes manuales en consolas reales sin perder continuidad.

## Problema que se resuelve

En escenarios con varios agentes del mismo runtime:

- `resume --last` no es fiable
- varios agentes pueden compartir proyecto
- el mismo agente puede cambiar de tarea o rama
- los tokens o sesiones del runtime pueden expirar

Por eso Orquesta debe guardar estado suficiente para reanudar de forma determinista.

## Datos minimos que deben persistirse al cerrar

- `external_session_id`
- `resumen_continuidad`
- `cwd`
- `branch`
- `herramienta`
- `conector`
- `estado`

## Regla principal

Cada agente manual debe trabajar sobre un `cwd` estable y aislado.

Si varios agentes trabajan sobre el mismo proyecto:

- se crea un worktree por agente
- cada worktree actua como `cwd` canonico de esa sesion

## Politica de autonomia

- las acciones normales de lectura, compilacion, test, edicion y navegacion se ejecutan sin pedir permiso previo
- las acciones destructivas, de borrado, irreversibles o de riesgo alto no se ejecutan de forma unilateral
- antes de una accion peligrosa, el agente debe consultar a Orquesta o a otro agente del mismo proyecto

## Estrategia correcta para Codex

1. Si Orquesta ya conoce `external_session_id`, se usa directamente.
2. Si es una sesion nueva:
   - arrancar Codex con un prompt de bootstrap estructurado
   - trabajar sobre un `cwd` unico
3. Al cerrar:
   - consultar la sesion reciente compatible con ese `cwd`
   - guardar `external_session_id`
   - guardar `resumen_continuidad`

## Terminator

### Requisitos

- una ventana o pestaña por agente
- titulo visible `agente · proyecto`
- `cwd` correcto
- integracion con Orquesta antes y despues de la sesion

### Scripts actuales

- `scripts/inicio_agente.sh`
- `scripts/cargar_agentes.sh`
- `scripts/terminator_agentes.sh`
- `scripts/agente_console.sh`
- `scripts/agentes.orquestador.plan`

## Script recomendado de entrada

Hasta que la web y la app de escritorio controlen el ciclo completo, el punto de entrada recomendado para un agente manual es:

```bash
scripts/inicio_agente.sh <agente>
```

Hace en una sola llamada:

1. `orquesta sesion inicio <agente>`
2. `orquesta tarea listar <agente>`
3. inicia una tarea si:
   - se indica `--tarea <id>`, o
   - existe una única tarea `asignada`, o
   - detecta una única tarea ya `en_progreso`

Ejemplos:

```bash
scripts/inicio_agente.sh Codex2
scripts/inicio_agente.sh Codex3 --tarea 155
scripts/inicio_agente.sh antigravity --tarea 161 --proyecto orquestador
```

## Formato del fichero `.plan`

Una linea por agente:

```text
agente|rol|ruta_proyecto|titulo_tarea|prioridad|modulo|nota_asignacion|conector|comando_runtime
```

## Flujo operativo recomendado

1. Registrar agente, proyecto, asignacion y tarea desde Orquesta.
2. Crear worktree si hay concurrencia sobre el mismo proyecto.
3. Abrir la consola con el wrapper del runtime.
4. Iniciar sesion de Orquesta para ese agente.
5. Ejecutar el runtime.
6. Al salir, guardar continuidad y cerrar sesion.

## Limitaciones conocidas

- el arranque de Terminator depende del entorno grafico real del usuario
- la deteccion automatica del identificador externo depende del runtime concreto
- no debe asumirse que todos los runtimes ofrecen una API local para recuperar sesiones

## Politica futura

La operacion manual debe convivir con un modelo mas robusto:

- `orquesta agente arrancar`
- `orquesta agente pausar`
- `orquesta agente continuar`

Mientras eso no exista completo, los wrappers manuales siguen siendo una capa operativa valida.
