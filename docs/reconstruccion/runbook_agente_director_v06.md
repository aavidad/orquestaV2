# Runbook: usar Orquesta V06 desde un agente director

Fecha: 2026-07-14. Aplica al checkpoint V06 y queda sustituido cuando V16
acredite workspace/Git o V22 acredite dirección autónoma completa.

## Objetivo y frontera honesta

Este procedimiento permite que un agente Codex externo —como la sesión que
opera el repositorio— use Orquesta como motor durable para ejecutar un DAG con
otros agentes Codex. Es una transición bootstrap gobernada, no la programación
externa completa prometida para V22.

La división actual es obligatoria:

| Responsable | Sí hace | Todavía no hace |
| --- | --- | --- |
| Agente director externo | inspecciona el proyecto, prepara contexto, compila el DAG, llama MCP y revisa artefactos; solo con autorización aplica cambios y tests | no crea otro lifecycle, no escribe SQLite ni infiere permiso de commit/merge/push/deploy |
| Orquesta V06 | persiste Goal/DAG, calcula ready-set, paraleliza write-sets disjuntos, lanza/observa Codex, reemplaza intentos y conserva artefactos/receipts | no crea worktrees, no aplica parches, no ejecuta tests del proyecto ni integra Git |
| Codex lanzado por Orquesta | produce un único artefacto textual desde el objetivo y metadata recibidos | no recibe workspace ni contexto del repositorio por contrato, no recibe artefactos previos automáticamente y no tiene permiso de escritura |

El último punto no es una recomendación: el adaptador vigente ejecuta Codex en
un subdirectorio privado de `runtime.codex.work_root`, con sandbox `read-only`,
`--ignore-user-config` y `--ignore-rules`. `write_set`, `skill_refs`,
`tool_refs`, `input_refs` y `criterion_refs` llegan como metadata del prompt;
no montan un workspace, no cargan una skill y no conceden una tool.

`read-only` limita escrituras, pero no es una frontera de confidencialidad del
sistema operativo. El proceso corre bajo el mismo usuario y recibe el entorno
allowlisted, normalmente `HOME` y `CODEX_HOME`; podría encontrar rutas del host
que ese usuario ya pueda leer. Ese acceso incidental no está soportado ni
acreditado: no pases rutas absolutas, no confíes en el sandbox para ocultar
secretos y no ejecutes material no confiable. Aislamiento de lectura real exige
un adaptador de contenedor/OS posterior.

Por tanto, hoy Orquesta puede encargar análisis, propuestas de código, parches
unificados, planes y revisiones autocontenidas. El director externo es quien
debe comprobar y aplicar el resultado en un worktree aislado. Nunca se debe
presentar un Goal `succeeded` como prueba de que el cambio está integrado.

## 1. Preflight del director

Antes de arrancar:

1. Lee `AGENTS.md` del rebuild y del proyecto objetivo.
2. Confirma autorización, repositorio, revisión base, rutas escribibles y tests.
3. Declara tareas con write-sets disjuntos; serializa solapamientos reales.
4. Comprueba espacio y que no existe otro servidor usando los roots elegidos.
5. No uses el árbol antiguo `/home/alberto/Trabajo/orquesta` como runtime ni
   como workspace del nuevo producto.

En el rebuild:

```bash
cd /home/alberto/Trabajo/orquesta-rebuild
git status --short
go test -mod=vendor -count=1 ./acceptance -run '^TestAcceptanceV0[1-6].*Receipt$'
```

Si se va a modificar un proyecto Git, el director crea el worktree; Orquesta
V06 no lo hace. Solo con autorización del operador:

```bash
git -C /ruta/al/proyecto worktree add \
  -b trabajo/orquesta-IDENTIFICADOR \
  /ruta/aislada/al/worktree REVISION_BASE
git -C /ruta/aislada/al/worktree status --short
```

No continúes sobre un worktree con cambios ajenos sin clasificarlos. La ruta
del worktree no debe usarse como `runtime.codex.work_root`: son superficies
distintas. El worker V06 no recibe ese workspace por contrato; escribir su ruta
en el objetivo no lo monta, no le aporta contexto y puede filtrar información
local innecesaria.

## 2. Preparar y arrancar el servidor

Crea un root operativo dedicado, privado y persistente. El siguiente bloque es
un ejemplo; sustituye la ruta antes de usarlo:

```bash
export ORQUESTA_RUNTIME_ROOT=/ruta/privada/orquesta-v06-runtime
install -d -m 700 \
  "$ORQUESTA_RUNTIME_ROOT" \
  "$ORQUESTA_RUNTIME_ROOT/state" \
  "$ORQUESTA_RUNTIME_ROOT/artifacts" \
  "$ORQUESTA_RUNTIME_ROOT/work" \
  "$ORQUESTA_RUNTIME_ROOT/secrets" \
  "$ORQUESTA_RUNTIME_ROOT/config"
cp config/orquesta.toml.example "$ORQUESTA_RUNTIME_ROOT/config/orquesta.toml"
chmod 600 "$ORQUESTA_RUNTIME_ROOT/config/orquesta.toml"
```

Edita el TOML y usa rutas absolutas bajo ese root para:

- `state.sqlite.path`;
- `artifact.filesystem.root`;
- `runtime.codex.work_root`;
- `identity.local_token_path`;
- `config.effective_path`.

Mantén `server.listen` en loopback. Comprueba además:

- `runtime.codex.command` resuelve al binario esperado;
- `runtime.codex.reasoning = "medium"` salvo riesgo justificado;
- `runtime.codex.timeout < scheduler.execution_timeout`;
- `runtime.codex.env_allowlist` contiene solo lo imprescindible;
- `runtime.codex.credential_ref` queda vacío hasta V08;
- los roots no se solapan y sus directorios existentes tienen modo `0700`.

Arranca en primer plano para conservar logs y parada cooperativa:

```bash
go run -mod=vendor ./cmd/orquesta serve \
  --config "$ORQUESTA_RUNTIME_ROOT/config/orquesta.toml"
```

No uses `curl` como si `/mcp` fuera REST. Es MCP Streamable HTTP con Bearer.
El servidor crea el token local una sola vez con modo privado.

## 3. Conectar el agente director mediante MCP

El cliente Codex instalado admite un servidor MCP HTTP y toma el Bearer desde
una variable de su propio proceso. Esa variable es bootstrap del cliente, no
una nueva clave de configuración del servidor Orquesta: no se añade a
`config/registry.json`, TOML ni `effective_config`.

Registra una vez el conector sin copiar el secreto a la configuración:

```bash
codex mcp add orquesta-v06 \
  --url http://127.0.0.1:8080/mcp \
  --bearer-token-env-var ORQUESTA_MCP_TOKEN
codex mcp get orquesta-v06
```

Antes de iniciar la sesión directora, carga el token sin imprimirlo ni ponerlo
como argumento:

```bash
ORQUESTA_MCP_TOKEN="$(<"$ORQUESTA_RUNTIME_ROOT/secrets/local-owner.token")"
export ORQUESTA_MCP_TOKEN
codex
unset ORQUESTA_MCP_TOKEN
```

Una sesión que ya estaba abierta puede no descubrir el conector recién
registrado; en ese caso inicia otra sesión con la variable disponible. No
pegues el token en prompts, logs, Goals, artefactos o mensajes de handoff.
El token concede identidad `local-owner` sobre las seis tools y no protege
frente a otros procesos del mismo usuario. Mientras esté exportado, usa solo un
director local confiable: no ejecutes hooks/scripts no revisados ni vuelques el
entorno. Si se expone, detén el servidor, rota el fichero mediante un
procedimiento controlado y registra la incidencia antes de reanudar.

El director debe ver exactamente estas seis tools:

- `orquesta.system.status`;
- `orquesta.goals.create`;
- `orquesta.goals.get`;
- `orquesta.goals.list`;
- `orquesta.artifacts.read`;
- `orquesta.goals.amend`, que en V06 solo registra un sucesor causal pendiente:
  no le añade plan, outbox ni ejecución.

La primera llamada siempre es `orquesta.system.status` con `{}`. Exige
`ready=true`. Si el conector no aparece, no simules llamadas MCP con comandos
ad hoc: corrige registro, URL, proceso o Bearer.

## 4. Compilar el trabajo antes de crear el Goal

El director externo inspecciona el proyecto con lecturas acotadas y prepara
para cada WorkItem un objetivo autocontenido. Debe incluir solo el contexto
necesario:

- revisión Git y hashes relevantes;
- fragmentos exactos de código o contratos que el worker necesita;
- comportamiento actual y esperado;
- rutas permitidas;
- formato de salida, normalmente diff unificado o informe estructurado;
- tests y criterios que el director ejecutará después.

Reglas del DAG V06:

- `key` identifica el item dentro de esa petición; `dependencies` y `parent`
  referencian esas keys, no refs durables inventadas por el cliente;
- `write_set` contiene rutas relativas limpias, sin rutas absolutas ni globs;
- scopes padre/hijo se solapan: `internal/a` solapa
  `internal/a/fichero.go` y se serializan;
- tareas sin dependencia y con write-sets disjuntos pueden ejecutarse en
  paralelo;
- `parent` expresa lineage; no sustituye a `dependencies`;
- usa `output_contract: "evidence_bundle"` salvo contrato distinto explícito;
- `confirm:true`, `request_ref` estable y `statement` exacto son obligatorios;
- repetir el mismo `request_ref` y el mismo payload es idempotente; si cambia
  el significado, crea otra ref;
- el DAG es estático después de `create`: V06 no añade items, replantea ni
  propaga automáticamente resultados durante la ejecución.

Una dependencia solo controla orden. V06 no incorpora el artefacto del item A
al prompt del item B. Si B necesita el resultado de A, usa dos Goals:

1. Goal A produce análisis o propuesta.
2. El director lee y valida su artefacto.
3. Goal B recibe explícitamente el contexto aprobado en sus objetivos.

No cargues repositorios completos en el prompt. Resume, incluye fragmentos
acotados y divide por fronteras reales. `input_refs` son referencias opacas, no
un RAG ni un loader de ficheros.

## 5. Crear un Goal con trabajo paralelo

Ejemplo de argumentos para `orquesta.goals.create`. Sustituye contexto, hashes,
rutas y criterios; no envíes los marcadores `<...>` literalmente:

```json
{
  "request_ref": "request:programacion:PROYECTO:TAREA:01",
  "statement": "Preparar propuestas de cambio independientes para TAREA sobre REVISION_BASE.",
  "normalized_objective": "Producir parches revisables sin modificar el repositorio objetivo.",
  "confirm": true,
  "plan": {
    "phases": [
      {
        "ref": "phase-instance:proposal",
        "key": "phase:proposal",
        "template_ref": "phase-template:code-proposal",
        "input_refs": ["input:project-context:REVISION_BASE"],
        "criterion_refs": ["criterion:unified-diff-and-tests"]
      }
    ],
    "work_items": [
      {
        "key": "patch-domain",
        "objective": "Contexto exacto: <fragmentos y contratos>. Propón un diff unificado limitado a internal/domain para <resultado>. No inventes APIs. Incluye tests que debe ejecutar el director.",
        "phase": "phase:proposal",
        "role": "role:implementer",
        "dependencies": [],
        "write_set": ["internal/domain"],
        "skill_refs": [],
        "tool_refs": [],
        "capability_refs": ["capability:code-proposal"],
        "output_contract": "evidence_bundle"
      },
      {
        "key": "patch-adapter",
        "objective": "Contexto exacto: <fragmentos y puerto estable>. Propón un diff unificado limitado a internal/adapters/x para <resultado>. Incluye negativos y tests; no modifiques dominio.",
        "phase": "phase:proposal",
        "role": "role:implementer",
        "dependencies": [],
        "write_set": ["internal/adapters/x"],
        "skill_refs": [],
        "tool_refs": [],
        "capability_refs": ["capability:code-proposal"],
        "output_contract": "evidence_bundle"
      }
    ]
  }
}
```

Conserva de la respuesta `goal_ref`, `app_spec.hash`, `app_spec.generation`,
`plan_generation`, WorkItem refs y revisión. `created=false` con el mismo Goal
es replay correcto, no un fallo.

## 6. Supervisar y recuperar resultados

Consulta `orquesta.goals.get` con `goal_ref`. No hagas polling agresivo; uno o
dos segundos es suficiente para una sesión interactiva. Usa
`orquesta.system.status` para pending/quarantine global sin leer SQLite.

Estados terminales:

- `succeeded`: todos los WorkItems terminaron y tienen la evidencia mínima;
- `failed`: se agotó una política de ejecución o un fallo terminal cerró el
  Goal; revisa `executions[].failure_code` y estados skipped;
- una acción en cuarentena aparece en `system.status`; V06 no tiene tool de
  reparación manual. No edites tablas para desbloquearla.

Tampoco existen todavía tools operativas de pausa, cancelación, rework, replan
o cierre manual. El lifecycle termina por las transiciones automáticas del
motor; no lo suplas escribiendo estado o inventando una tool lateral.

Tras `succeeded`, toma cada `artifact_ref` de `goals.get` y llama
`orquesta.artifacts.read` con su `goal_ref`. El contenido viene como Base64 y
su digest debe coincidir con la evidencia del Goal. Decodifica a un temporal
privado; no lo apliques todavía.

Checklist del director para cada artefacto:

1. pertenece al WorkItem y write-set esperados;
2. no contiene secretos, paths fuera de scope ni cambios no solicitados;
3. aplica contra la revisión declarada;
4. respeta arquitectura y contratos existentes;
5. incluye o permite construir tests proporcionales;
6. no confunde texto plausible con efecto ya realizado.

## 7. Validar y, si está autorizado, aplicar/probar fuera de Orquesta

Por defecto el director entrega el diff revisado y los tests propuestos sin
mutar el proyecto. Modificar el worktree exige que el encargo incluya
implementación. Commit, merge, push, deploy o publicación son efectos separados
y requieren orden explícita; nunca se infieren de `Goal succeeded` ni de la
autorización para editar.

Cuando la implementación esté autorizada, valida primero el artefacto aprobado
en el worktree aislado. Para un diff unificado:

```bash
git -C /ruta/aislada/al/worktree apply --check /ruta/privada/propuesta.diff
```

`git apply --check` solo valida y no autoriza ni aplica el cambio. Usa
`apply_patch` para toda edición manual y conserva cambios ajenos; las
reescrituras mecánicas siguen únicamente las excepciones exactas de `AGENTS.md`.
Ejecuta los tests del proyecto, revisa el diff y registra:

- `goal_ref`, AppSpec hash/generación y execution refs;
- artifact refs y digests utilizados o descartados;
- commit base y commit resultante;
- comandos de test y resultados;
- desviaciones, trabajo parcial reutilizado y P0/P1.

Una revisión externa puede crear un nuevo Goal autocontenido con el diff y
criterios exactos. Hasta V18, esa revisión no es todavía el workflow de review
independiente acreditado de Orquesta. Hasta V16, el commit y la integración no
son receipts de Orquesta.

Si se quiere delegar también la escritura antes de V16, el director puede usar
un subagente Codex directo dentro del worktree como excepción bootstrap
declarada en `AGENTS.md`. Esa ejecución no la gobierna Orquesta: debe recibir
write-set y tests exactos, trabajar en rama/worktree aislados y devolver diff,
tests y riesgos al director. No se debe cambiar el worker V06 a
`workspace-write` ni llamar “adaptador temporal” a este atajo.

Si el artefacto es inválido pero recuperable, no borres el Goal ni reescribas
su estado. Conserva la evidencia y crea otro `orquesta.goals.create` con DAG,
contexto corregido y nueva `request_ref`. No uses `orquesta.goals.amend` para
continuar programación en V06: aunque exige Goal terminal y CAS exacto, su
sucesor queda pendiente y no recibe plan ni ejecución. Esa tool solo conserva
la relación causal para una vertical futura.

## 8. Parada y reanudación

Antes de parar, consulta `orquesta.system.status` y guarda las refs necesarias.
Envía `SIGINT` o `SIGTERM` al servidor y espera su cierre cooperativo. No mates
procesos hijos ni borres roots mientras el servidor está vivo.

Para reanudar, usa el mismo TOML y los mismos roots. SQLite recupera estado,
outbox, receipts y ejecuciones; trabajo terminal no se relanza. Después del
arranque llama de nuevo a `orquesta.system.status` y `orquesta.goals.get`.

Elimina un runtime solo cuando el operador haya decidido que no se necesita
reanudación y se hayan preservado evidencias. Comprueba antes procesos, PID,
worktrees y status; nunca limpies por patrón amplio.

## 9. Instrucción lista para entregar al siguiente agente

```text
Actúa como director externo de Orquesta V06. Lee AGENTS.md y
docs/reconstruccion/runbook_agente_director_v06.md completos. No uses el runtime
legacy ni codebase-memory. Inspecciona el proyecto objetivo con rg y lecturas
acotadas, declara revisión base, write-set y tests, y crea worktree aislado solo
si el operador lo autorizó. Comprueba orquesta.system.status. Compila Goals/DAG
autocontenidos y paraleliza únicamente write-sets disjuntos. Los workers V06 son
read-only y no reciben workspace/contexto del repo por contrato: aporta contexto
mínimo exacto y pídeles artefactos o diffs, no efectos. Lee artifacts por MCP y
valida scope/digest. Solo si el encargo autoriza implementación, edita con
apply_patch y ejecuta tests; commit, merge, push, deploy o publicación requieren
orden explícita separada.
Registra Goal/AppSpec/execution/artifact refs y resultados. No afirmes que Goal
succeeded equivale a commit, review o integración. Si falta workspace/Git,
propagación de contexto, tools reales o review gobernada, informa la frontera;
no la simules ni la metas en el núcleo.
```

## 10. Cuándo retirar este bootstrap

- V16 reemplaza worktree, aplicación de patch e integración Git manuales por
  puertos y receipts de workspace/Git.
- V18 reemplaza la revisión externa informal por autor/revisores/refinery
  acreditados sobre el mismo diff y revisión.
- V22 permite que la petición abierta, dirección, ejecución, revisión, tests,
  integración y cierre se gobiernen completamente desde Orquesta con Codex.

Hasta entonces, este runbook es el camino honesto: Orquesta gobierna la
ejecución durable; el agente externo gobierna contexto, escrituras y efectos.
