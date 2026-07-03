# Piloto de mutation testing Orquesta

Fecha del plan: 2026-07-04.

## Objetivo

`scripts/orquesta_mutation_pilot.sh` ejecuta un piloto opt-in de mutation
testing solo sobre:

- `./modulos/orquesta-estado-vivo`
- `./modulos/orquesta-goal`

No entra en CI por defecto, no modifica codigo de produccion y no anade
dependencias a `go.mod`. El operador debe tener `go-mutesting` disponible en
`PATH` o declarar una herramienta equivalente con
`ORQUESTA_MUTATION_PILOT_TOOL`.

## Ejecucion manual/nightly extendida

Comando minimo:

```bash
ORQUESTA_MUTATION_PILOT_CONFIRM=MUTATION_PILOT_OPT_IN \
scripts/orquesta_mutation_pilot.sh
```

Resultado por defecto:

```text
~/.orquesta-nightly/mutation-testing-pilot/resultado_mutation_YYYYMMDD.json
```

Variables utiles:

- `ORQUESTA_NIGHTLY_RESULTS_DIR`: raiz alternativa del directorio nightly.
- `ORQUESTA_MUTATION_PILOT_DATE_ID`: fecha reproducible para pruebas.
- `ORQUESTA_MUTATION_PILOT_TOOL`: ruta o nombre de la herramienta.
- `ORQUESTA_MUTATION_PILOT_ACCEPTABLE_SCORE`: umbral propuesto, por defecto
  `0.70`.
- `ORQUESTA_MUTATION_PILOT_TOP_SURVIVORS`: numero de mutantes supervivientes
  destacados, por defecto `5`.

El script no acepta argumentos de paquetes. El alcance queda cerrado en el
propio script para evitar ejecutar mutation testing sobre otros modulos.

## JSON publicado

El JSON usa `schema_version=orquesta_mutation_pilot.v0` e incluye:

- `date_utc`, `run_id`, timestamps y duracion.
- `target_packages` con los dos paquetes cerrados.
- `packages[]` con `score`, `score_percent`, contadores parseados,
  `tool_exit_code` y log por paquete.
- `surviving_mutants_top` como lista corta de candidatos a tests nuevos.
- `policy.ci_default=false` y `policy.go_mod_dependency_required=false`.

## Umbral propuesto

Umbral inicial aceptable: `0.70` por paquete. Es deliberadamente diagnostico:
si un paquete queda por debajo del umbral, el nightly extendido debe abrir una
tarea de tests focales, no bloquear CI ni tocar produccion automaticamente.

Cuando haya varios runs, elevar el umbral solo con evidencia de estabilidad. Un
primer objetivo razonable tras cerrar los mutantes graves es `0.80` en
`orquesta-goal`, porque concentra validacion de cierre, write-set y resultados
durables.

## Top de mutantes supervivientes graves

El top operativo sale de `surviving_mutants_top` del JSON. Para convertirlo en
tests nuevos, priorizar en este orden:

1. `modulos/orquesta-goal/validation_v0.go`: mutantes que debiliten validacion
   de write-set, required tests o resultado durable.
2. `modulos/orquesta-goal/lifecycle_v0.go`: mutantes que acepten estados
   terminales contradictorios o incompletos.
3. `modulos/orquesta-estado-vivo/reglas_precedencia_v0.go`: mutantes que
   cambien precedencia entre proceso externo verificado, `wait_external` y
   outbox pendiente.
4. `modulos/orquesta-estado-vivo/proyeccion_v0.go`: mutantes que mezclen
   estados publicos que deben distinguir outbox pendiente, espera externa y
   proceso externo verificado.
5. `modulos/orquesta-goal/observe_active_skip_v0.go`: mutantes que omitan una
   observacion activa necesaria o marquen una observacion como saltable sin
   evidencia.

Estos cinco grupos son candidatos directos a nuevos tests si sobreviven en un
run real.

## Verificacion local focal

Sin instalar herramienta real:

```bash
bash -n scripts/orquesta_mutation_pilot.sh
scripts/test_orquesta_mutation_pilot.sh
git diff --check
```

El test usa una herramienta falsa, comprueba que falta de opt-in bloquea,
rechaza argumentos de paquetes y verifica que el JSON fechado contiene score por
paquete y top de mutantes supervivientes.
