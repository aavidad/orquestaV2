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

## Script manual de compatibilidad

Aunque el control plane ya cubre el runtime vivo por agente/proyecto, el punto de entrada manual de compatibilidad sigue siendo:

```bash
scripts/inicio_agente.sh <agente>
```

Este script no debe considerarse la vía principal de operación de Orquesta. Su papel es de compatibilidad, recuperación y operación manual controlada.
Contrato T121: estos wrappers son recuperacion/operacion asistida. No pueden
arrancar flotas por seed, mutar DB/stores/worktrees ni lanzar runtime directo
por fuera del servidor residente. Si el servidor no responde con readiness
versionada, deben bloquear con un error publico recuperable.

El resto de scripts del directorio `scripts/` cumplen funciones de:

- backend de lanzamiento
- integración con terminal concreta
- utilidades de soporte

No deben usarse como sustituto directo del arranque manual base salvo que una tarea concreta lo exija.

Hace en una sola llamada:

1. `orquesta sesion inicio <agente>`
2. `orquesta tarea listar <agente>`
3. inicia una tarea solo si se indica `--tarea <id>`; si no, bloquea el
   autoarranque o exige `--no-auto`

No debe autoasignar una tarea por heuristica local ni inferir estado desde DB,
tmux, Terminator, ficheros de runtime o rutas privadas.

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
3. Arrancar o verificar primero el servicio de Orquesta.
4. Abrir la consola con el wrapper del runtime solo si el flujo requiere intervención manual.
5. Iniciar sesion de Orquesta para ese agente.
6. Ejecutar el runtime.
7. Al salir, guardar continuidad y cerrar sesion.

## Correlacion operativa

La continuidad se correlaciona por refs opacas:

- `agent_ref` o nombre publico de agente para iniciar sesion.
- `task_ref` y `run_ref` si la consola se abre para una tarea viva.
- `external_session_id` solo como identificador de sesion del runtime concreto,
  registrado por el adaptador correspondiente.
- `worktree_ref` como referencia opaca del espacio de trabajo; no se documenta
  ni se valida una ruta local como contrato.
- cierre de sesion por CLI/API del servidor, con resumen de continuidad
  redactado y evidencia compacta.

Terminator, tmux u otra terminal grafica solo son contenedores de consola. No
son fuente de verdad del nucleo ni autorizan saltarse `readiness`, run/task refs
o shutdown/checkpoint gobernados.

## Limitaciones conocidas

- el arranque de Terminator depende del entorno grafico real del usuario
- la deteccion automatica del identificador externo depende del runtime concreto
- no debe asumirse que todos los runtimes ofrecen una API local para recuperar sesiones

## Politica futura

La operacion manual debe convivir con un modelo mas robusto:

- `orquesta agente arrancar`
- `orquesta agente pausar`
- `orquesta agente continuar`

Ya existe además una operación administrativa conservadora para sanear duplicados históricos de identidad:

- `orquesta agente fusionar <origen> <destino>`

Debe usarse solo cuando:

- ya existe un nombre canónico claro
- el origen no tiene sesión activa
- no hay locks, worktrees ni runtime vivo asociados al origen
- interesa preservar trazabilidad histórica sin tocar la BD a mano

Mientras eso no exista completo, los wrappers manuales siguen siendo una capa
operativa valida solo para recuperacion asistida y siempre subordinada al
servidor residente.
