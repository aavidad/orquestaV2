# Instrucciones para Hermes - cola viva del revisor (2026-07-12)

De: Claude (director/revisor residente de Orquesta). Este fichero es la cola
VIVA de trabajo de Hermes. Al completar un item, marca su checkbox y anota el
commit; el revisor la reejecuta y la actualiza en cada despertar.

Complementa, NO sustituye, a `docs/runbooks/hermes_orquesta_aislado_2026-07-11.md`
(arranque, aislamiento y cierre de la instalacion). Si algo de aqui contradice
ese runbook, manda el runbook y avisa al revisor.

## Quien es quien (no invadir carriles)

- **Orquesta** ejecuta los goals (backend `app_server_tmux`) y es la unica que
  materializa cambios dentro del write-set que ella misma entrega.
- **Hermes** es agente de trabajo dentro del contenedor aislado: opera sobre
  `/workspace`, dentro del write-set del goal, y devuelve el resultado a
  Orquesta. No lanza servidores fuera del contenedor ni toca el host.
- **Codex** trabaja su cola en `docs/instrucciones_codex_2026-07-11.md`.
- **Claude (revisor)** valida cierres reejecutando los tests declarados; ningun
  cierre se acepta autodeclarado.

## Reglas de oro (vinculantes)

1. **Write-set**: solo escribes lo que el goal declara. Nada fuera de `/workspace`
   ni fuera del scope. Un cambio fuera de scope invalida el goal entero.
2. **Nada de verdes autodeclarados**: ejecuta los `Tests:` declarados y muestra
   la salida real. Si no puedes ejecutarlos, dilo y devuelve `blocked`.
3. **No inventes evidencia**: ni fixtures, ni refs, ni resultados. Este proyecto
   ya sufrio dos falsos verdes (F3-R2 y 208H); la atestacion independiente esta
   activa y bloquea el lanzamiento si no hay atestador configurado.
4. **Discrepancia > obediencia ciega**: si el enunciado del goal no cuadra con
   el codigo real, documenta la discrepancia y propon la correccion SIN
   aplicarla (esto es exactamente lo que hizo bien el goal T9104).
5. **Cierres controlados**: nunca `kill`. Shutdown gobernado y verificacion de
   residuos (procesos, paneles tmux `orquesta-goal-*`, sockets).
6. **No `go test ./...` global**: mata sesiones por memoria. Focales del paquete
   tocado; lotes solo con `scripts/orquesta_test_batches.sh`.
7. **No commitees artefactos de ejecucion** (`checkpoint_started_*`,
   `orquesta_goal_result_*`): no son fuente.
8. **No subas ratchets ni presupuestos** para ponerte en verde (envs 425/103):
   se consolida, no se eleva el limite.

## Estado del proyecto (contexto minimo)

- **NUCLEO: CERRADO SIN CONDICIONES** (BUG-226 cerrado en `befb3707b` con
  prueba empirica real: Orquesta lanzo y completo un goal por su API nativa,
  closure accepted, shutdown limpio).
- **Frente unico vigente: CONECTORES** (runtime-codex-*, state-file,
  required-test, superficies MCP/HTTP, wiring stack/`cmd/orquesta-server`).
- **AUXILIARES CONGELADOS** hasta cerrar conectores: tools/CLI extra, transporte
  stdio, ingesta, presentaciones, web/telegram.
- Bugs abiertos restantes: solo residuales de campo (remoto/OPES/proveedor real).
  Ninguno bloquea el trabajo local.

## REPARTO DE CARRILES (orden del operador 2026-07-12)

Hermes NO trabaja el mismo frente que Codex. Reparto vinculante:

- **Hermes: BUGS + PROGRAMACION DE TOOLS.** Es tu carril propio. Las tools
  dejan de estar congeladas PARA TI (siguen congeladas para Codex, que no
  debe abrir frente auxiliar mientras cierra conectores).
- **Codex: CONECTORES** (runtime-codex-*, state-file, required-test,
  superficies MCP/HTTP, wiring stack/cmd). No entres en esos ficheros: si un
  bug tuyo obliga a tocarlos, avisa al revisor por la senal y espera.
- Si dos carriles chocan en un mismo fichero, gana quien lo tenga asignado y
  el otro espera; nunca se edita el write-set activo del otro agente.

## Cola de trabajo

- [ ] H1: BUGS. Coge los bugs abiertos del inventario
  (`docs/inventario_bugs_orquesta_2026-06-30.md`) de uno en uno, empezando por
  los reproducibles en local. Para cada uno: reproducir, arreglar, test que
  falle sin el fix, y cierre con evidencia real. Los residuales de campo
  (remoto/OPES/proveedor real) NO son tuyos: dejalos abiertos y anotados.
- [ ] H1b: TOOLS. Programacion de tools nuevas de Orquesta en tu carril. Cada
  tool: contrato claro, tests focales, sin logica de dominio duplicada (el
  nucleo ya es autoridad unica: no reimplementes reconciliacion causal ni
  gobierno de progreso material).
- [ ] H1c: Preferentemente trabaja DENTRO de goals que Orquesta te entregue
  (write-set gobernado). Si trabajas fuera de un goal, respeta igual el
  write-set del bug/tool y no toques el carril de Codex.
- [ ] H2: Si un goal se te queda sin progreso material (sin diff, test, result
  ni receipt), NO sigas quemando contexto: devuelvelo con causa concreta. El
  gobierno de progreso material (BUG-226) esta activo y cortara igualmente.
- [ ] H3: Al terminar un hito, avisar al revisor con la senal (una linea):

      echo "H<n> cerrado en <commit>: <resumen>" > /home/alberto/Trabajo/orquesta/.orquesta-revisor-wake

  Usa la misma senal si te BLOQUEAS ("BLOQUEADO: <causa>"). El fichero senal no
  se commitea.

## Trampas ya conocidas (no las repitas)

- El **guard de identidad** exige binario reproducible: cualquier fichero suelto
  en la raiz del checkout degrada el servidor (`runtime_build_not_reproducible`).
  Arbol limpio antes de compilar.
- `worktree_ref` y `branch_ref` de un request de autoprogramacion son **refs
  opacas** (sin `/`): no rutas ni nombres de rama git.
- La **atestacion independiente** exige fichero de config con permisos 0600 y el
  env `ORQUESTA_GOAL_REQUIRED_TEST_ATTESTATION_CONFIG_FILE`; sin el, Orquesta
  rechaza el goal (y hace bien).
- El **shutdown** solo alcanza `ready` con `cleanup_goal_backends: true` cuando
  queda backend vivo.
- Los goals escriben su resultado durable bajo el **primer scope directorio** del
  write-set: pon `docs` primero para no ensuciar `scripts/`.
- **tmux 3.6**: no "simplifiques" selectores (`=sesion:` con `:`).

## Hallazgo del arranque (2026-07-12, revisor)

La entrada canonica arranca y el aislamiento se aplica, pero Hermes pide
`hermes setup` (proveedor OpenAI Codex, modelo gpt-5.5): la instalacion no
tiene la configuracion/credencial completada. El operador debe ejecutar
`hermes setup` (y `hermes doctor` para diagnostico) desde la sesion TUI antes
de que Hermes pueda ejecutar trabajo real. Sin ese paso, el contenedor
levanta, muestra status y sale.
