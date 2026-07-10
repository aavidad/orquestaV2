# Backlog acotado para piloto de autonomia

Fecha: 2026-07-10.

Este backlog es el primer piloto local de Orquesta. Las tres tareas son
documentales, tienen write-set unico y no autorizan remoto, OPES, credenciales,
procesos persistentes ni cambios de codigo. Se ejecutan de una en una durante
el piloto supervisado de la etapa D2.

## T9101 revisar-evidencia-208h

Objetivo: revisar la rama publicada `wip/attestation-208h-20260710`, reejecutar
la lista de pruebas declaradas y registrar en el handoff si la evidencia es
suficiente para que un revisor humano decida integrar o devolver rework.

Estado: pendiente.

Alcance:

- `docs`

Criterios:

- no se afirma cierre de 208H sin ejecutar los comandos declarados
- el resultado distingue pruebas verdes, fallos y evidencia no disponible
- no se modifica codigo ni se integra ninguna rama

Tests:

- `git diff --check -- docs`
- revisar `docs/pruebas_revisor_208h_2026-07-10.md` desde la rama publicada

## T9102 actualizar-indice-bugs-vivos

Objetivo: contrastar el indice canonico de bugs vivos con los commits y
handoffs del corte, corrigiendo solo estados, refs y siguientes acciones que
esten documentalmente desfasados.

Estado: pendiente.

Alcance:

- `docs`

Criterios:

- cada fila conserva un residual verificable y una siguiente accion concreta
- no se crean IDs duplicados ni se marcan bugs como cerrados por texto libre
- los cambios enlazan al commit, test o receipt que los respalda

Tests:

- `git diff --check -- docs`
- `rg -n 'BUG-ORQ-20260710-208H|BUG-ORQ-20260710-208E' docs/inventario_bugs_estado_vivo.md`

## T9103 clasificar-retencion-s13

Objetivo: preparar una clasificacion JSON documentada de los artefactos S13
versionados para que una ola posterior pueda retener, archivar o borrar solo
con evidencia, sin eliminar ficheros en este piloto.

Estado: hecha fuera del piloto (revisor, 2026-07-10). Artefactos:
`docs/clasificacion_retencion_s13_2026-07-10.json` (jq empty OK) y
`docs/clasificacion_retencion_s13_2026-07-10.md`. Sustituida en el piloto
por T9104.

Alcance:

- `docs`

Criterios:

- cada artefacto queda clasificado como retener, archivar o candidato a borrar
- los candidatos incluyen motivo, referencias encontradas y riesgo de borrado
- el piloto no borra, mueve ni modifica artefactos historicos

Tests:

- `git diff --check -- docs`
- el JSON de clasificacion es parseable por `jq empty`

## T9104 verificar-muestra-clasificacion-s13

Objetivo: verificar una muestra de 10 entradas de
`docs/clasificacion_retencion_s13_2026-07-10.json` (al menos 3 de cada
clase): comprobar que el fichero existe, que las referencias listadas son
reales (grep) y que la clase asignada respeta el criterio documentado en
`docs/clasificacion_retencion_s13_2026-07-10.md`. Registrar discrepancias.

Estado: pendiente.

Alcance:

- `docs`

Criterios:

- cada entrada de la muestra queda anotada como correcta o discrepante con
  la evidencia del comando ejecutado
- no se borra, mueve ni modifica ningun artefacto historico
- si hay discrepancias, se proponen correcciones sin aplicarlas

Tests:

- `jq empty docs/clasificacion_retencion_s13_2026-07-10.json`
- `git diff --check -- docs`
