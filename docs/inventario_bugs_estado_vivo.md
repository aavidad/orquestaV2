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
| BUG-ORQ-20260710-208E | tooling drain/harness listo (F3-R2); falta receipt real `clean` de drain + dos pases amplios verdes. El perfil aislado exige `ORQUESTA_TEST_CACHE_ROOT` y `ORQUESTA_TEST_BATCH_ROOT` fuera de `/srv`; su cleanup debe restaurar permisos readonly de `GOMODCACHE` | operador ejecuta drain + `orquesta_test_batches.sh` con ambas rutas aisladas; consolidar config y `chmod -R u+w` previo al cleanup en F3 |
| F5/identidad runtime | integrado en `2fe12f658`: valida ctl/worktree, degrada identidad y limpia backend por identidad; focales verdes | ejecutar drain/deploy gobernados y verificar por API el binario remoto, sin tocar `uso-app` |
| BUG-ORQ-20260710-208H | atestacion independiente de required tests: candidato con focales y dos lotes aislados verdes, pero no aceptable mientras exista `208J` | reparar y revisar `208J` antes de integrar; BLOQUEA D1/D2 y autonomia sin supervision |
| BUG-ORQ-20260710-208J | candidata rebasada `wip/attestation-208j-20260710` (`063feeae7`) persiste `failed` y rework tras error del atestador; focales y dos lotes aislados verdes, no integrada ni revisada sobre principal | [incidencia 208J](incidencias/incidencia_orquesta_208h_claim_pending_sin_reintento_2026-07-10.md): reejecutar focales y conservar cierre bloqueado hasta entonces |
| BUG-ORQ-20260710-208K | routing de modelos en rama WIP rechaza configuraciones legacy sin `model_routing`, rompiendo Codex, Claude y stack | [incidencia 208K](incidencias/incidencia_orquesta_model_routing_fail_closed_legacy_2026-07-10.md): normalizar ausencia a politica conservadora y reejecutar paquetes |
| BUG-ORQ-20260710-208I | timeout parcial de observe coexistio con `invalid` durable; causa raiz no demostrada | correlacionar refs/tiempos tras adoptar veredicto causal F1 en observe |
| SUBFALLO routing/modelos 20260710 | fail-open en defaults/args vacios, herencia xhigh, PATH ambiguo; ratchet 521 vs limite 513 | WIP en `fix/model-routing-p0-p1-sol-20260710`; fail-closed y consolidar sin subir ratchet |
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

## Regla de conteo

Bugs vivos de codigo del nucleo: los de la primera tabla (7 entradas, de las
cuales 208A-E son el mismo frente con residuales distintos). Todo lo demas es
operativo o de campo. Si una lectura antigua del inventario historico
contradice este indice, prevalece este indice.
