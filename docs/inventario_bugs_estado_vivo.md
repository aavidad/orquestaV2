# Bugs vivos de Orquesta - indice canonico

Actualizado: 2026-07-10 (post-integracion del fix del lease, HEAD `1f24eb9f2`).
Mantenedor: Claude (revisor). Regla: UNA fila por bug vivo con su residual
exacto; el historial completo vive en
`docs/inventario_bugs_orquesta_2026-06-30.md` y NO se cuenta desde alli.
No reutilizar IDs. Al cerrar o abrir un bug, actualizar este indice en el
mismo commit.

## Vivos (nucleo)

| ID | Residual exacto que lo mantiene vivo | Siguiente accion |
| --- | --- | --- |
| BUG-ORQ-20260710-208A-D | patches focales (write-sets solapados, 504, workdir, rework) verdes en local; sin verificar por API contra servidor desplegado | tras deploy: repro/API de prepare-run, observe y runs/control |
| BUG-ORQ-20260710-208E | tooling drain/harness listo (F3-R2); falta receipt real `clean` de drain + dos pases amplios verdes. El perfil aislado exige `ORQUESTA_TEST_CACHE_ROOT` y `ORQUESTA_TEST_BATCH_ROOT` fuera de `/srv`; una revision local 2026-07-10 confirmo que Go 1.25 deja `GOMODCACHE` readonly y un runner desligado puede dejar helpers Unix bajo esa raiz | operador ejecuta drain + `orquesta_test_batches.sh` con ambas rutas aisladas; consolidar config y restaurar `chmod -R u+w` antes del cleanup, verificando que no quedan helpers por identidad de runtime |
| F5/identidad runtime | integrado en `2fe12f658` y completado localmente por D1 con `5f30973d7`: identidad de servidor y proyecto externo ya son distintos; closure accepted y shutdown listo | ejecutar drain/deploy gobernados y verificar por API el binario remoto, sin tocar `uso-app` |
| BUG-ORQ-20260710-208H | integrado en `7444dcf8a`: D1 normal acepto el resultado y una reejecucion externa posterior paso, pero el servidor no llevaba atestador independiente configurado | probar atestador real integrado y sus rutas de fallo antes de cerrar el bug |
| BUG-ORQ-20260710-208J | integrado en `7444dcf8a`: fallo del atestador persiste claim `failed` y cierre `blocked/rework`, sin reintento por polling | D1 debe confirmar comportamiento con atestador real; conservar la [incidencia 208J](incidencias/incidencia_orquesta_208h_claim_pending_sin_reintento_2026-07-10.md) como evidencia |
| BUG-ORQ-20260710-208K | la revision del WIP confirma que `codexModelRoutingFromProjectConfigFileV0` y `claudeModelRoutingFromProjectConfigFileV0` crean politica por defecto pero dejan `ModelAlias` vacio si falta `*_model_routing`; el resolver rechaza por tanto todo launch legacy normal | [incidencia 208K](incidencias/incidencia_orquesta_model_routing_fail_closed_legacy_2026-07-10.md): materializar aliases canonicos de composicion solo cuando la seccion esta ausente, mantener fail-closed para configuracion explicita parcial y reejecutar paquetes |
| BUG-ORQ-20260710-208I | timeout parcial de observe coexistio con `invalid` durable; causa raiz no demostrada | correlacionar refs/tiempos tras adoptar veredicto causal F1 en observe |
| SUBFALLO routing/modelos 20260710 | fail-open en defaults/args vacios, herencia xhigh, PATH ambiguo; D3 local mide 538 variables `ORQUESTA_*` frente al limite 513 | WIP en `fix/model-routing-p0-p1-sol-20260710`; consolidar/fusionar configuracion sin subir ratchet y repetir D3 |
| BUG-ORQ-20260701-079 | solo frontera proveedor: cap duro pre-tool ante stdout crudo sin redireccion | esperar enforcement del proveedor o probe adversarial nuevo; no bloquea local |
| BUG-ORQ-20260704-165 / 20260701-065 | residual amplio de observabilidad/control lento con proveedor real; nucleo local cerrado | se paga con la adopcion completa del veredicto F1 + repro 208 tras deploy |

## Vivos (operativos, no de codigo)

| ID | Residual | Siguiente accion |
| --- | --- | --- |
| S14/deploy remoto | binario remoto vivo `9541e2f0...` anterior al codigo; flujo nuevo: remoto = solo destino de deploy | drain gobernado + `orquesta_server_deploy.sh` cuando el operador decida subir |
| S13/F4 artefactos versionados | 61 ficheros de ejecucion en el arbol fuente (37 patron exacto S13 + 24 familia backlog) | auditoria gobernada de retencion con JSON de clasificacion |
| S12 limpieza envs remota | perfil remoto/secretos/defaults sin corte gobernado | corte separado tras deploy; no mezclar con drain |
| CODEX-HOME-TOKEN-INVALIDADO | auth Codex remota caducada | reauth del operador en servidor; hoy ademas cuota local agotada |
| BUG-ORQ-20260701-058/066/075 (familia OPES) | solo residuales de campo: OPES temporal/preproduccion y proveedor real; local/fake cerrado | field test OPES temporal cuando se retome ese frente |

## Cerrados hoy (referencia rapida)

- Incidencia lease generation conflict primer lanzamiento: cerrada con
  `01cb27d77` (selector tmux `=sesion:` + verificacion tri-estado +
  cleanup/shutdown degradan a evidencia residual). Smoke real local verde
  end-to-end verificado por revisor.
- BUG-ORQ-20260710-210 (guard textual stale del smoke compuesto).
- 208G (perdida causal de `checkpoint_started` en materializador) - reducido
  local, pendiente confirmacion independiente.
- BUG-ORQ-20260710-208L: cerrado por `5f30973d7` y D1 local retenido en
  `/tmp/orquesta-goal-first-app-server.ZAorvf`; el target externo no Git ya
  no se usa como identidad del binario.
- D3 local 2026-07-10: detenido tras el primer lote determinista para no
  gastar un segundo pase imposible. `TestEnvVarsBudgetMEJ106V0` fallo con
  538 variables frente a 513; recibo y log retenidos en
  `/tmp/orquesta-test-batches/receipt.json` y
  `/tmp/orquesta-test-batches/logs/pass-001-batch-001.log`. No hubo cambios
  de fuente ni procesos residuales.
- Revision routing 2026-07-10: los focales de capacity, runtimes y stack
  fueron verdes, pero la bateria amplia del worktree WIP se desligo antes de
  devolver resultado y dejo dos helpers Unix bajo
  `/tmp/orquesta-routing-review-env`; se pararon por identidad de esa ruta y
  se limpio despues de restaurar permisos de `GOMODCACHE`. No acredita el
  test amplio ni la integracion del routing; pertenece a 208E/F3.

## Regla de conteo

Bugs vivos de codigo del nucleo: los de la primera tabla (7 entradas, de las
cuales 208A-E son el mismo frente con residuales distintos). Todo lo demas es
operativo o de campo. Si una lectura antigua del inventario historico
contradice este indice, prevalece este indice.
