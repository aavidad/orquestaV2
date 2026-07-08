# Auditoria: programacion minima y ahorro de tokens

Fecha: 2026-07-08.

Objetivo: localizar recursos externos y medidas internas para reducir consumo
de tokens y evitar codigo innecesario en agentes de programacion, sin activar
nuevas reglas globales sin prueba empirica.

## Resumen ejecutivo

Orquesta ya tiene una base buena:

- `skills/orquesta-programacion-minima/SKILL.md` cubre diff minimo,
  anti-overengineering, no helpers/abstracciones/archivos sin necesidad,
  reutilizacion de repo/stdlib/plataforma y medicion A/B antes de default.
- `AGENTS.md` ya exige comunicacion compacta, contexto acotado, `rg`/refs antes
  de lecturas amplias, write-set estrecho y `medium` por defecto.
- `docs/runbooks/orquesta_golden_evals_2026-07-04.md` ya define banco A/B para
  comparar reglas de agente.

La mejora no debe ser "meter mas texto" en `AGENTS.md`. Hay evidencia externa
de que los context files grandes pueden aumentar pasos, lecturas, escrituras y
coste. La mejora correcta es:

1. mantener `AGENTS.md` estable y corto;
2. usar `orquesta-programacion-minima` como skill/skill_ref opt-in;
3. medirla en golden tasks con tokens reales, diff stats, rework y score;
4. solo activarla como default amplio si no baja calidad ni sube rework.

## Recursos externos utiles

| Recurso | Enlace | Para que sirve | Agentes | Reduce tokens / codigo innecesario | Reglas relevantes | Adaptable a Orquesta | Valoracion |
|---|---|---|---|---|---|---|---|
| AGENTS.md standard | https://agents.md/ | Formato comun de instrucciones repo/directorio. | Codex, Jules, Cursor, Windsurf/Cascade, otros. | Ambos, si se mantiene enfocado. | Instrucciones por repo, comandos, tests, convenciones, AGENTS anidados. | Usar solo para verdad estable del repo; mover reglas especiales a skills. | Util |
| OpenAI Codex: AGENTS.md | https://developers.openai.com/codex/guides/agents-md | Guia oficial de carga global/repo/override. | Codex. | Indirecto: evita repetir instrucciones; puede subir coste si se abusa. | Global `~/.codex/AGENTS.md`, repo `AGENTS.md`, overrides cercanos. | Confirmar que Orquesta no duplica reglas globales/locales; preferir AGENTS locales solo por modulo. | Util |
| OpenAI Codex Skills | https://developers.openai.com/codex/skills | Skills con progressive disclosure. | Codex. | Tokens: carga completa solo al activar skill. | Descripciones concisas, skills para workflows reutilizables. | Mantener `orquesta-programacion-minima` como skill activable, no como bloque siempre-on largo. | Muy util |
| Claude Code: Manage costs | https://code.claude.com/docs/en/costs | Guia oficial de costes/contexto. | Claude Code. | Tokens. | Gestionar contexto, modelo, MCP, hooks/skills, instrucciones fuera de CLAUDE.md si son especificas. | Aplicar a Orquesta: skills y herramientas diferidas; no MCP amplio por defecto. | Muy util |
| Anthropic: Agent Skills | https://www.anthropic.com/engineering/equipping-agents-for-the-real-world-with-agent-skills | Skills con scripts para trabajo determinista. | Claude/Claude Code. | Tokens y calidad: ejecutar scripts en vez de generar texto. | Offload a codigo determinista; no cargar documentos grandes en contexto. | Potenciar harness/scripts para medicion de diff/tokens en vez de pedir resumen manual. | Util |
| Karpathy-inspired CLAUDE.md | https://github.com/multica-ai/andrej-karpathy-skills/blob/main/CLAUDE.md | Prompt anti errores comunes de agentes. | Claude; portable a Codex/Cursor. | Principalmente codigo innecesario; algo de tokens por menos rework. | Think before coding, simplicity first, surgical changes, goal-driven execution. | Ya cubierto parcialmente. Copiar solo frases compactas: "no features beyond asked", "no single-use abstraction", "touch only what you must". | Util con cuidado |
| Caveman | https://github.com/JuliusBrussee/caveman | Skill/plugin de comunicacion tersa. | Claude, Codex, Gemini, Cursor, Windsurf, Cline, Copilot. | Tokens de salida. | Quitar filler, preservar comandos/codigo/errores exactos. | Ya instalado. Usar en updates/finales y subagentes, no para decisiones donde la compresion cree ambiguedad. | Parcialmente util |
| claude-token-efficient | https://github.com/drona23/claude-token-efficient | CLAUDE.md para respuestas tersas. | Claude Code. | Tokens de salida. | Respuestas cortas, menos narracion. | Redundante con Caveman; no copiar completo para no inflar instrucciones. | Parcial |
| Azure SDK for Python AGENTS.md | https://github.com/Azure/azure-sdk-for-python/blob/main/AGENTS.md | Ejemplo real de repo grande. | AGENTS-compatible. | Codigo innecesario/diff. | Cambios minimos para warnings concretos; no dependencias nuevas ni refactors grandes; revalidar tras cada fix. | Adoptar como criterio en skill: "warning concreto => cambio concreto". | Util |
| claude-howto CLAUDE.md | https://github.com/luongnv89/claude-howto/blob/main/CLAUDE.md | Ejemplo real con seccion Token Efficiency. | Claude Code. | Ambos. | No re-leer lo recien editado; no eco de bloques grandes; batch edits; no tool calls de mas. | Copiar al runbook de medicion, no a AGENTS global sin A/B. | Util |
| Lich skills / build-until-pass | https://github.com/LichAmnesia/lich-skills | Skills de loops acotados. | Claude/Gemini extension. | Codigo innecesario y rework. | Leer primer error, fix mas pequeno, re-run, cap de intentos, no fake green. | Excelente para Orquesta: skill de bugfix minimo con attempt cap y primer error. | Muy util |
| Koroqe claude-code-sdlc | https://github.com/Koroqe/claude-code-sdlc | Pipeline multiagente/SDLC. | Claude Code. | Parcial: controla drift; puede aumentar coste. | Plan ejecutable con `Files/Changes/Verify/Done when`, waves por file overlap, minimal diff salvo items estructurales. | Adaptar formato de slices para Orquesta y paralelizacion por write-set. | Parcialmente util |
| Evaluating AGENTS.md paper | https://arxiv.org/html/2602.11988v1 | Evidencia empirica sobre context files. | Codex/Claude-style coding agents en benchmark. | Advierte contra sobrecargar contexto. | Context files aumentan pasos/coste; developer-written ayuda poco; LLM-generated puede empeorar. | Regla clave: ningun prompt/AGENTS nuevo sin golden eval A/B. | Muy util |
| Cursor Rules docs | https://cursor.com/docs/rules | Documentacion oficial de reglas persistentes. | Cursor. | Indirecto. | Project/team/user rules; scoping. | Si exportamos reglas para Cursor, deben ser dinamicas por tarea, no todo always-on. | Parcial |
| Trigger.dev Cursor Rules | https://trigger.dev/blog/cursor-rules | Guia practica de reglas Cursor. | Cursor. | Codigo innecesario via convenciones/validacion. | Rule type correcto, ejemplos concretos, verificacion, pruebas de reglas. | Aplicar a Orquesta: reglas por skill_ref y tests de regla antes de default. | Util |
| Devin/Cascade AGENTS.md y Rules | https://docs.devin.ai/desktop/cascade/agents-md / https://docs.devin.ai/desktop/cascade/memories | Scoping AGENTS y reglas/memorias. | Devin Desktop / Windsurf Cascade. | Tokens por scoping; menos repeticion. | AGENTS por directorio; reglas versionadas; memoria local no compartida. | Evitar memorias locales no auditables; usar docs/skills versionados. | Util |
| Firecrawl token efficiency | https://www.firecrawl.dev/blog/claude-code-token-efficiency | Practicas de coste para Claude Code. | Claude Code, transferible. | Tokens. | Separar plan/build/verify; snippets; controlar MCP overhead; prompts estructurados. | Orquesta ya tiene varias. Medir MCP/tool schema overhead y separar modos en golden tasks. | Util |
| MindStudio benchmark skills | https://www.mindstudio.ai/blog/5-claude-code-skills-cut-token-costs-70-percent-benchmarked | Ejemplo de benchmark comparativo de skills. | Claude Code. | Tokens, coste, calidad. | Comparar sesiones identicas con/sin plugin. | Patron para Orquesta: baseline/variante, mismas tareas, mismo modelo, score + coste. | Util |

## Medidas internas ya existentes

| Medida | Ubicacion | Tipo | Estado |
|---|---|---|---|
| Comunicacion compacta y `caveman` para agentes/subagentes | `AGENTS.md`, `skills/orquesta-director-agentes/SKILL.md` | Tokens | Activa como norma operativa |
| `rg`/lecturas acotadas, no contexto bruto, no codebase-memory por defecto | `AGENTS.md` | Tokens | Activa |
| ContextBundle por refs y limites de bytes | `modulos/orquesta-context/docs/contratos.md` | Tokens | Existe como subsistema |
| Broker de contexto con snippets compactos, cache/dedupe | `modulos/orquesta-context/docs/contratos.md` | Tokens | Existe |
| Write-set estrecho y paralelizacion solo si no se pisan | `AGENTS.md` | Diff/seguridad | Activa |
| `orquesta-programacion-minima` | `skills/orquesta-programacion-minima/SKILL.md` | Ambos | Existe; debe seguir opt-in hasta A/B |
| ACK con `changed_files`, `new_files_count`, helpers y abstracciones | `skills/orquesta-programacion-minima/SKILL.md` | Diff/overengineering | Existe; falta contrastarlo contra diff real |
| Config canonica | `AGENTS.md` y runbooks | Evita duplicidad | Activa, aun con residuales |
| Razonamiento `medium` por defecto y `xhigh` solo justificado | `AGENTS.md`, `capacity_reasoning_policy_v0.go` | Tokens | Activa |
| Presupuesto idle y tokens estimados/cache | `idle_self_improvement_v0.go` | Tokens | Existe |
| Benchmark opt-in de compresion documental | `scripts/benchmark_context_compression_optin.sh` | Tokens | Existe, no productivo |
| Golden tasks A/B | `scripts/orquesta_golden_evals.sh`, `docs/evals/orquesta_golden_tasks_v0.json` | Medicion | Existe; falta token/diff real completo |
| Tests congelados | `docs/plan_mejora_continua_orquesta_2026-07-04.md` | Evita fake green/diff tramposo | Diseñado |
| Efficiency summary MCP | `autoprogramming_efficiency_summary_v0.go` | Observabilidad | Existe; no mide tokens por si solo |

## Reglas candidatas para copiar/adaptar

No copiar bloques largos. Adaptar como reglas cortas dentro de la skill
`orquesta-programacion-minima` o como `skill_ref` de tarea:

```text
No features beyond asked. No single-use abstraction. No configurability unless requested.
Touch only files needed for current write-set. Do not improve adjacent code.
Before adding helper/file/dependency, prove existing repo/stdlib/platform cannot cover it.
Bugfix loop: reproduce or identify first failing evidence -> smallest root-cause fix -> rerun.
Green is command exit/evidence, not agent opinion. Never fake green.
If structural fix truly exceeds minimal diff, declare STRUCTURAL with files/changes/verify/done_when.
```

## Plan empirico antes de activar nuevas medidas

### Dataset

Usar `docs/evals/orquesta_golden_tasks_v0.json` como banco inicial. Ejecutar
minimo dos brazos:

- A: baseline actual, sin `skill-ref-orquesta-programacion-minima-v0`.
- B: misma tarea/modelo/presupuesto con `skill-ref-orquesta-programacion-minima-v0`.

Si se prueba Caveman/token-efficient, hacerlo como brazo separado. No mezclar
skill de diff minimo y compresion de salida en el mismo experimento inicial.

### Metricas obligatorias

Por `run_ref`/`goal_ref`:

- score del evaluador;
- tests requeridos pasados/fallidos;
- ficheros tocados y ficheros nuevos;
- lineas anadidas/eliminadas;
- escrituras fuera de write-set;
- `helpers_added`, `abstractions_added`, `scope_expansion_reason`;
- tool calls/pasos si el backend lo reporta;
- tiempo total;
- tokens reales si el proveedor los publica: input, output, reasoning,
  cached/prompt-cache;
- rework count y causa;
- bugs/incidencias generadas.

### Criterio de activacion

Una medida solo puede pasar de opt-in a default si:

- no baja score ni rompe tests respecto a baseline;
- no aumenta rework ni incidencias;
- reduce tokens netos despues de contar el coste de la propia regla/skill;
- reduce o mantiene ficheros nuevos y lineas tocadas;
- no aumenta falsos bloqueos por "minimalismo" en fixes estructurales reales.

Si reduce diff pero baja calidad, queda opt-in. Si reduce tokens de salida pero
aumenta ambiguedad/rework, queda solo para updates/finales compactos, no para
contratos tecnicos.

## Cambios recomendados

1. No tocar `AGENTS.md` de momento. Ya es largo y tiene reglas suficientes.
2. Mantener `orquesta-programacion-minima` como skill canonica opt-in.
3. Crear una tarea separada para extender `scripts/orquesta_golden_evals.sh` y
   `result.json` con metricas reales de tokens/diff/rework por `run_ref`.
4. Crear un launcher A/B aislado para comparar baseline vs
   `skill-ref-orquesta-programacion-minima-v0`.
5. Si el A/B sale bien, entonces evaluar si una version ultracorta de la skill
   entra como default en la composicion de autoprogramacion, no en `AGENTS.md`.

Avance 2026-07-08: el evaluador ya acepta `result.json.metrics`, agrega
`summary.metrics` y conserva `tasks[].metrics`. Falta que un launcher aislado
rellene tokens reales de proveedor y diff stats reales por brazo A/B.

## Decision actual

No se activa ninguna regla nueva global en este commit. Este documento deja
evidencia para Claude y para el Director: la direccion correcta es medir antes
de endurecer instrucciones, porque el remedio puede empeorar coste, pasos o
calidad.
