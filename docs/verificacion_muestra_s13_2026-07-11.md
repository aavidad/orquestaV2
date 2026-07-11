# Verificación de muestra S13 (T9104)

Fecha de verificación: 2026-07-11. Manifiesto revisado:
`docs/clasificacion_retencion_s13_2026-07-10.json`.

## Alcance y discrepancia de clases

El manifiesto contiene 103 entradas: `retener=3`, `archivar=75` y
`archivar_condicionado=25`. No existe la clase literal `candidato_borrar`.
La nota `docs/clasificacion_retencion_s13_2026-07-10.md` documenta que los 25
antiguos candidatos a borrar fueron reconciliados como `archivar_condicionado`
y que ninguno se declara libre de referencias ni apto para borrado. Por ello,
la muestra equilibrada de 10 usa 3 `retener`, 3 `archivar` y 4
`archivar_condicionado` como representación vigente de `candidato_borrar`.

La discrepancia de nomenclatura queda registrada; no se modifica el manifiesto.
Corrección propuesta (no aplicada): actualizar el criterio de T9104 para pedir
`archivar_condicionado` como clase reconciliada, o añadir una tabla explícita
`candidato_borrar -> archivar_condicionado`.

## Evidencia reproducible

Comandos eje ejecutados desde la raíz del repositorio:

```text
python3 - <<'PY' ... json.load(...); os.path.isfile(path); rg --files -g basename ...
Salida resumida: 10/10 ficheros existen; 38/38 referencias listadas de las
dos entradas con referencias se resolvieron por basename; las otras 8 entradas
no listan referencias.
```

Además, la distribución se obtuvo con `json.load` y `Counter`:
`{'retener': 3, 'archivar': 75, 'archivar_condicionado': 25}`.

## Entradas verificadas

Todas las entradas siguientes son **correctas** respecto al criterio vigente y
al estado del árbol. `refs=19/19` significa que cada basename de las 19
referencias del manifiesto tuvo al menos una coincidencia con `rg --files -g`;
`refs=0` significa que el registro no declara referencias.

| # | Clase | Entrada | Existencia | Referencias | Juicio |
|---:|---|---|---|---|---|
| 1 | retener | `docs/historico/checkpoint_started_t272_limpieza_raiz_repo.md` | sí | 0 | correcta: ya archivada bajo `docs/historico` |
| 2 | retener | `docs/historico/docs/orquesta_goal_result_goal-ref-autoprogramming-backlog-t272-limpieza-raiz-repo-df0c2f74.json` | sí | 0 | correcta: evidencia histórica ya archivada |
| 3 | retener | `docs/incidencias/incidencia_orquesta_goal_result_canonico_compartido_2026-07-02.md` | sí | 0 | correcta: documentación de incidencia |
| 4 | archivar | `cmd/orquesta-server/checkpoint_started.txt` | sí | 19/19 | correcta: marcador citado; archivar en ola gobernada |
| 5 | archivar | `cmd/orquesta-server/docs/checkpoint_started.txt` | sí | 19/19 | correcta: marcador citado; archivar en ola gobernada |
| 6 | archivar | `cmd/orquesta-server/docs/orquesta_goal_result_goal-ref-autoprogramming-backlog-t281-diagnostico-etiquetas-backlog-d23bb3aa.json` | sí | 0 | correcta: receipt histórico, no fuente |
| 7 | archivar_condicionado | `cmd/orquesta-server/docs/checkpoint_started_goal-ref-task-autoprogramming-6153eceb442b-g01.txt` | sí | 0 | correcta: marcador condicionado, no borrar |
| 8 | archivar_condicionado | `cmd/orquesta-server/docs/checkpoint_started_goal-ref-task-autoprogramming-93e0439fa334-g01.txt` | sí | 0 | correcta: marcador condicionado, no borrar |
| 9 | archivar_condicionado | `cmd/orquesta-server/docs/checkpoint_started_goal-ref-task-autoprogramming-9fa50e72c7ef-g01.txt` | sí | 0 | correcta: marcador condicionado, no borrar |
| 10 | archivar_condicionado | `cmd/orquesta-server/docs/checkpoint_started_t281_diagnostico_etiquetas_backlog.txt` | sí | 0 | correcta: marcador condicionado, no borrar |

No hubo discrepancias de existencia, referencias o criterio en las 10 entradas.
No se borró, movió ni modificó ningún artefacto histórico. Cualquier archivado
queda como propuesta futura de ola gobernada.

## Test requerido

Ejecutado: `jq empty docs/clasificacion_retencion_s13_2026-07-10.json`.
Resultado: **pasó** (JSON válido).
