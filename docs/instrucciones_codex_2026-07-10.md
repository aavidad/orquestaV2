# Instrucciones para Codex - 2026-07-10 (cuota recuperada)

De: Claude (revisor/director de continuacion). Contexto global:
`docs/guia_continuacion_agentes_2026-07-10.md` y
`docs/inventario_bugs_estado_vivo.md`.

## Tu tarea unica: cerrar 208H (atestacion de tests)

1. Trabaja SOLO en tu worktree
   `/home/alberto/Trabajo/orquesta-worktrees/` rama
   `wip/attestation-208h-20260710` (36 ficheros sin commitear a fecha de
   este documento). Nadie mas lo toca: esta reservado para ti.
2. Objetivo: dejar la atestacion de tests completa, commiteada y pusheada,
   con sus tests declarados ejecutables por un revisor externo.
3. Criterio de cierre (lo validara Claude reejecutando los tests, no se
   acepta verde autodeclarado; 208H ya tuvo un falso verde por fixtures
   inventados): cada test declarado en el receipt debe ejecutar con
   `go test -count=1` real sobre el paquete correspondiente y salir `ok`.
4. NUNCA ejecutes `go test ./...` global (mata sesiones por memoria).
   Verifica solo los paquetes tocados; para verificacion amplia usa
   `scripts/orquesta_test_batches.sh` (dos pases, aislado) al final.
5. Commits: espanol, prefijo `fix:`/`feat:`/`test:`/`docs:`, un paso por
   commit, push inmediato a tu rama.

## Que NO hacer

- No toques la rama `trabajo/plataforma-agentes` fuera de tu worktree:
  Claude esta ejecutando ahi las etapas A-C de la guia (veredicto causal
  F1 en superficies MCP, actuador runs/control F2, preparacion piloto).
- No lances smokes reales (`*_real.sh`) ni el piloto D2: son pasos
  posteriores coordinados por el operador con Claude como revisor.
- No conectes al servidor remoto `srv1651826` ni a `uso-app`.
- No commitees `checkpoint_started_*` ni `orquesta_goal_result_*`.
- No modifiques `scripts/orquesta_server_drain.sh`,
  `orquesta_server_deploy.sh` ni `orquesta_server_ctl.sh`.

## Al terminar

Deja una nota con la lista de tests declarados (paquete + patron `-run`)
en `docs/` de tu rama para que el revisor pueda reejecutarlos, y avisa al
operador. Despues de 208H aceptado se desbloquean D1 (smoke real) y D2
(piloto con el backlog de la etapa C2).
