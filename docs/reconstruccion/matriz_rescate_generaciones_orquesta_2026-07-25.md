# Matriz de rescate de generaciones Orquesta — 2026-07-25

Estado: inventario físico de apoyo; no bloquea el avance de V23.

## Propósito y límite

Primer inventario durable de fuentes bajo `/home/alberto/Trabajo`, para rescatar
conocimiento o comportamiento útil sin volver a acoplar el producto nuevo al
árbol antiguo. El target es este worktree: `2881bea08982`, árbol
`0dcae8456d92` (`recuperacion/home-aislado`). Su autoridad sigue siendo
`AGENTS.md`, `product/roadmap.json`, `product/capabilities.json` y
`docs/reconstruccion/ruta_total_100.md`; este documento no promueve legado a
canon.

El inventario se hizo el 2026-07-25 con `find` de `.git`/worktrees hasta cinco
niveles, `git rev-parse HEAD HEAD^{tree}` y una inspección de forma para
directorios no Git. No se leyeron credenciales, configuraciones sensibles ni
estado de runtime. Es amplio y reproducible, pero **no afirma exhaustividad**:
los árboles no Git profundos, archivos comprimidos y workspaces creados o
eliminados después del barrido requieren el siguiente lote.

## Método de identidad y regla de uso

- Igual `HEAD^{tree}`: una sola fuente lógica; se listan ubicaciones como
  réplicas, sin revisar cada copia.
- Distinto árbol Git: candidato independiente; antes de rescatar se compara el
  diff acotado contra el capability/vertical vigente, nunca se copia un paquete
  entero.
- Sin Git: hash de manifiestos/módulos si contiene fuente; si solo hay runs,
  ACKs, artefactos, logs o estados, se clasifica como evidencia/runtime.
- Cualquier conducta rescatada se expresa primero como contrato, fixture o test
  de caracterización en la arquitectura nueva. No imports, bridges, dual-writes
  ni fallbacks hacia `/home/alberto/Trabajo/orquesta`.

**Regla operativa: revisar todo el inventario legacy antes de cada V.** Antes de
abrir una V se recorren las cuatro familias históricas, los árboles
independientes deduplicados y las ramas de reconstrucción, aunque a primera
vista una fuente parezca ajena. Para cada capacidad nueva o bug: (1) consultar
esta matriz y los cánones históricos, (2) comparar identidades, índices y diffs
de todas las familias, (3) profundizar por símbolo/contrato en los candidatos
aplicables, (4) extraer invariante, fallo conocido y evidencia útil, (5) decidir
`ya_integrado`, `adaptar` o `descartar`, y (6) registrar la decisión en el
capability/bug. La búsqueda no autoriza copiar arquitectura ni declarar una
capacidad acreditada. Al final del roadmap se repite además una revisión global
del producto completo.

## Fuentes de código deduplicadas

| Familia / ubicaciones representativas | Identidad observada | Clasificación frente al target | Uso de rescate |
| --- | --- | --- | --- |
| Target V23: `orquestaV2-v23-home-aislado`, `orquestaV2-v23-recuperacion-agent`, `orquestaV2-cierre-v22` | mismo árbol `0dcae8456d92` (HEAD `2881bea08982`) | **Ya integrado/equivalente** entre las tres copias | Base única; no tratar las réplicas como tres fuentes. |
| V22 especializados: `orquestaV2-v22-{config-mcp,mailbox,e2e,inventario,presupuesto-*,rbac,revocacion}` y `wt-alberto-agente-b-sec-rbac-core` | 9 árboles distintos; ver commits en worktrees | **Ramas parciales superadas o divergentes** | Ninguna aporta un fichero ausente del cierre V22. No hacer cherry-pick; extraer solo un test puntual si demuestra un hueco actual. |
| Reconstrucción principal: `orquesta-rebuild` | `98789e291b5` (HEAD `6f244a759414`) | **V07--V22 integradas por descendencia; resto histórico** | Los commits terminales V07--V22 son ancestros del target. Consultar contratos y decisiones posteriores sin asumir que el HEAD completo sea equivalente. |
| Olas de reconstrucción V07: `orquesta-rebuild-v07-{bootstrap,manager,registry,store}` | 4 árboles distintos | **Ramas parciales superadas** | No portar. Conservan casos de caracterización para bootstrap, registry y persistencia si aparece una regresión concreta. |
| Worktrees de reconstrucción: `orquesta-rebuild-worktrees/v19-*` … `v37-*` | V19--V22 con implementación; V23--V37, salvo V30, con contrato/fixture/test | V19--V22: **integradas por descendencia**; V23--V37: **especificación planificada, sin producto** | V19--V22 no se portan. De V23--V37 se reutilizan requisitos y negativos sobre la arquitectura actual; nunca los commits completos. No se localizó worktree V30. |
| Fuentes V22 aisladas: `.orquesta-v22-e-20260724/source{,2}` | commits `64f8f38d30dd`, `b02f7a6ab3ef`, ancestros del target | **Evidencia redundante ya integrada** | Conservar como trazabilidad; no recuperar código desde ellas. |
| Workspaces Goal efímeros: `.orquesta-goal-workspaces-09275a87a782765b/workspaces/*` | 83 rutas, **33 árboles** distintos; repeticiones mayores: `6c25ae0eb070` y `00c111d3aa03` (10 cada una) | **Histórico/conocimiento** hasta asociarlos a Goal/receipt | Agrupar por árbol y metadata causal, no leer/copiar 83 veces. Muestrear solo los árboles que correspondan a una incidencia o capability abierta. |
| Línea antigua principal: `orquesta`, `.worktrees/orquesta-vec-director`, y sus sub-worktrees runtime | árbol `e0079e9217f3`, commit `7576f60bd3b5` (muchas réplicas); `.worktrees/orquesta-vec-canonical` es `d5922837f4d8` | **Histórico/conocimiento**; congelada por el target | Fuente de contratos, smokes y errores reales del Director/goal-first. Prohibido usar como dependencia o restaurar su control-plane. |
| Variantes antiguas: `orquesta-autonomia-clean`, `orquesta-autoprogramacion-*`, `orquesta-goal-worktree-*`, `orquesta-wt-lease-generation-*`, `orquesta.bk(sin VM Berserk)` | 6 árboles distintos | **Histórico/conocimiento** | Rescate dirigido para autonomía, autoprogramación, Goal/lease y wizard; siempre contrastar con el lifecycle único actual. |
| Adaptadores o consumidores externos: `orquesta-app-codex-stack`, `hermes-orquesta-bridge`, `PlataformaMunicipal/orquestador`, VEC/OPES runtime worktrees | árboles/forma heterogéneos | **Falta o divergente** para conectores; **runtime/artefacto** cuando sea `.orquesta-runtime` | Útiles para contratos de borde, nunca para incorporar dominio, filesystem interno o estado de otra app. |
| Backups y snapshots sin fuente Git: `backups-orquesta/*.tar.gz`, `orquesta-e2e-*`, `orquesta_smokes/*` | un backup comprimido y proyectos sueltos con `go.mod`; hash de módulos pendiente | `orquesta-e2e-*`/`orquesta_smokes`: **histórico/conocimiento**; backup: **pendiente de clasificación** | Abrir solo bajo caso concreto, en área aislada y sin extraer secretos. |
| `orquesta-{runs,real-runs,smokes,state-*,runtime-*,artifacts-archive}`, `.orquesta-*`, OPES `.orquesta-runtime/*` | no fuente Git o worktrees internos repetidos | **Runtime/artefacto no fuente** | Conservar como evidencia de operación; no usar para implementación ni limpieza automática desde este trabajo. |

## Comparación por áreas/módulos

| Área target / vertical | Fuente anterior más útil | Estado de rescate | Decisión actual |
| --- | --- | --- | --- |
| Intent, Goal, WorkItem, escritor único, evidencia | `orquesta-rebuild` y V07 store/manager | **Ya integrado/equivalente a nivel de capability** (`CORE-*` aceptadas); equivalencia por símbolo aún no auditada | No migrar código; usar para tests de caracterización ante regresión. |
| Configuración, credenciales, backup/restore | cierres V07--V09 | **Integrado y acreditado por descendencia** | No crear stores ni registros paralelos. Reutilizar negativos de revocación, recuperación por descriptor y CAS causal. |
| Identidad, proyectos, RBAC, OIDC | cierres V10--V12 | **Integrado y acreditado por descendencia** | No portar middleware alterno. Conservar aislamiento, auditoría, lease y fencing como invariantes. |
| Director, scheduler, mailbox, waits, control | árbol antiguo `orquesta`, mapa de generaciones y cierres V13--V14 | Mailbox/controles V13--V14 **integrados**; generaciones antiguas del Director son **conocimiento histórico** | Mantener lifecycle y escritor único del target. Traducir solo invariantes aún aplicables, nunca restaurar el control-plane legado. |
| Agent/Codex, efectos, workspace/Git, test attestor, review/consejo | cierres V15--V22 y `orquesta-app-codex-stack` | **Integrado y acreditado por descendencia**, con regresiones puntuales BUG-454/455/456 | Reparar los bugs sobre el target. Única extracción inmediata: semántica del detector Go legacy para BUG-454, no su módulo. |
| i18n, command registry | cierres V20--V21 | **Integrado y acreditado por descendencia** | Extender el registro y catálogos únicos existentes; no importar ramas ni catálogos paralelos. |
| Wizard, web, providers, tools/skills, contexto, plugins, deploy | `v23`--`v29` | **Falta o divergente / planned** | Son candidatos ordenados por roadmap; no confundir worktree existente con capacidad terminada. |
| OPES, multihost, operaciones, apps externas, cutover, videojuegos | roadmap V30--V37, contratos V31--V37, conectores OPES y artefactos de runtime | **Planificado**; no se localizó worktree V30 y V31--V37 no contienen producto | Mantener dominio fuera del núcleo; rescatar sólo puertos, fixtures y criterios de efecto. |

## Canones e incidencias históricas reutilizables

| Documento | Valor reutilizable | Límite |
| --- | --- | --- |
| `orquesta/docs/mapa_generaciones_director_2026-07-03.md` | Mapa de generaciones, módulos congelados y sunset de `legacy_director_loop` | Describe el árbol antiguo; no define lifecycle del target. |
| `orquesta/docs/autoprogramacion_orquesta_pendientes_2026-05-23.md` | Backlog, T259/T260 y trazabilidad de deuda operativa | Convertir cada ítem aplicable a capability/bug nuevo; no ejecutar su plan literal. |
| `orquesta/docs/inventario_bugs_orquesta_2026-06-30.md` | Evidencia, falsos verdes y cierres previos | Fuente de hipótesis y regresiones, no prueba de acreditación actual. |
| `orquesta/docs/analisis_fallos_estructurales_orquesta_2026-07-10.md` | F1--F6: autoridades fragmentadas, stop real, verificación contaminada, write-set y cierres narrativos | Aplicar como anti-patrones al diseño nuevo. |
| `orquesta/docs/informe_auditoria_integral_orquesta_2026-07-13.md` | Inventario top-down, deuda del director y matriz requisito/evidencia | Auditoría de una composición anterior; revalidar cada hallazgo. |
| `orquesta/docs/auditoria_limpieza_worktrees_ramas_2026-07-06.md` | Qué worktrees retener como evidencia y cómo no limpiar a ciegas | No autoriza borrar nada en este rescate. |
| `orquesta/docs/auditoria_runtime_orquesta_2026-05-24.md` y auditorías 2026-07 | Fronteras runtime, env y dead-code | Advisory/documentación: no reintroducir filtros o rails automáticos del legado. |

## Próximos lotes verificables

1. Generar un manifiesto no sensible `ruta, HEAD, tree, branch, hash-go.mod` y
   actualizar sólo cuando cambie el conjunto de rutas; excluir explícitamente
   `.orquesta-runtime`, states, logs y credenciales.
2. Para V07--V22, mantener el resultado de descendencia como línea base y
   comparar ramas antiguas solo cuando un bug actual identifique un hueco
   concreto; no reabrir verticales verdes por existir copias divergentes.
3. Asociar los 33 árboles de workspaces Goal a `GoalRef`/receipt disponibles;
   clasificar el resto como efímero y no abrirlo masivamente.
4. Inspeccionar backup comprimido y proyectos sueltos únicamente ante una
   pregunta concreta, con extracción temporal aislada y hash previo.
5. Antes de cualquier implementación, añadir a la tarea la fila de esta matriz
   consultada y la decisión `equivalente/adaptar/descartar`.

Comandos de repetición seguros (metadatos únicamente):

```bash
find /home/alberto/Trabajo -maxdepth 5 \( -type d -name .git -o -type f -name .git \) -printf '%h\n' | sort -u
git -C <ruta> rev-parse HEAD HEAD^{tree}
find <ruta> -maxdepth 3 -name go.mod -print0 | sort -z | xargs -0 sha256sum
```
