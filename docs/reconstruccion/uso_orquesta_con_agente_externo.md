# Uso de Orquesta con un agente externo

Fecha de corte: 2026-07-16. Rama: `reconstruccion/orquesta-total-20260714`.
Repositorio operativo: `/home/alberto/Trabajo/orquesta-rebuild`.

## Respuesta corta y alcance real

Sí: el checkpoint acreditado V01-V14 ya sirve para que un Codex externo use
Orquesta por MCP, cree un Goal con un DAG, lance uno o varios workers Codex,
consulte su estado y recupere artefactos durables. Hay un binario productivo
único, autenticación, autorización por proyecto, SQLite, artefactos, scheduler,
backup/recovery y cierre cooperativo.

V14 está cerrado por receipt V3 `PASS`: P=`6dbc0d808de63973305914b002c3bc2b8a806bb0`,
S=`e3e7c28e669ccd7e67a8661c40333d649ab82dd5` y
E=`5b97545ad14a40fd0063fc3671f6e79d9978ec09`. Este runbook público conserva
las mismas seis tools: V14 no añade bindings HTTP/MCP/CLI para sus controles;
estos pertenecen al registro único V20.

V13 acredita el mailbox causal interno `child_delivery`, pero no añade bindings
públicos. Este runbook no atribuye al agente externo claim, delivery, consume o
ACK por mailbox. `Parent` sigue siendo solo linaje: la entrada MCP pública no
expone `HandoffRequired`, su valor queda `false` y el DAG normal cierra sin
mailbox ni requeue.

No es todavía la Orquesta total. Hasta V22, el agente externo sigue siendo el
director práctico: inspecciona el proyecto, aporta contexto autocontenido,
compila el DAG, revisa resultados y, si el operador autorizó cambios, aplica y
prueba los parches fuera de Orquesta. Un Goal `succeeded` acredita la ejecución
y sus artefactos; no acredita por sí solo que exista commit, merge o deploy.
V12 es coordinación operativa limitada, V22 es el primer MVP de programación
extremo a extremo y solo V34 cierra la aplicación total.

Fuente de verdad del estado:

- `product/roadmap.json`: capacidades y estados canónicos;
- `product/evidence/v01_*.json` a `product/evidence/v14_*.json`: receipts
  reproducibles;
- `product/evidence/v13_mailbox.json`: receipt V3 `PASS` desde checkout
  `detached_clean` sobre el candidato sellado V13;
- `product/evidence/v14_controls.json`: receipt V3 `PASS` desde checkout
  `detached_clean` sobre el candidato V14 sellado;
- `docs/reconstruccion/estado_y_handoff_rebuild.md`: último handoff humano.

V01-V14 representan 14 de 34 verticales, 41,18 %, y 48 de 257 capacidades,
18,68 %, con 14/14 receipts válidos. V14 suma exclusivamente `GOV-07`,
`STG-15`, `ORC-03` y `ORC-16`; V11 sigue siendo una vertical transversal sin
IDs nuevos.

Resumen funcional:

| Vertical | Resultado disponible |
| --- | --- |
| V01-V03 | catálogo, autoridad del rebuild, trazabilidad y lecciones |
| V04 | intención, AppSpec y amendments causales |
| V05 | DAG, fases, dependencias, parentesco y write-sets |
| V06 | SQLite, eventos, outbox, scheduler, leases/fencing worker y replay |
| V07 | registro único de configuración, TOML y effective-config redactado |
| V08 | CredentialStore y secretos solo por referencia |
| V09 | recovery, backup y restore verificables |
| V10 | proyectos, multiusuario, RBAC, aislamiento y auditoría |
| V11 | proveedor neutral de identidad, `local_token` y OIDC; interoperabilidad AD mediante Dex/LDAP-LDAPS probada en entorno aislado |
| V12 | Director neutral con claim, renew, takeover, lease/fence y propuesta causal sobre el mismo Goal/SQLite/outbox |
| V13 | mailbox interno acreditado solo para `child_delivery` opt-in: destinatario exacto, lifecycle causal y retiro sistémico; `Parent` público permanece no contractual y sin bindings mailbox |
| V14 | controles acreditados application-only: pause/resume, cancel, stop selectivo cooperativo/forzado, retry de Execution y replan causal; sin bindings públicos hasta V20 |

V12–V14 no añaden tools públicas: Director, mailbox y controles están en
aplicación/composición, pero la superficie MCP pública vigente sigue teniendo
seis tools. Para operar hoy, el Codex externo declara el plan completo en
`orquesta.goals.create`; no puede invocar `Control` ni `ProposeDirectorPlan` por
MCP.

## 1. Preflight obligatorio

Trabajar solo en el rebuild nuevo:

```bash
cd /home/alberto/Trabajo/orquesta-rebuild
test "$(git branch --show-current)" = "reconstruccion/orquesta-total-20260714"
git status --short --branch
codex --version
```

Usar el `HEAD` limpio que entregue el handoff de cierre. No arrancar desde un
commit intermedio de producto, sellado o evidencia. No ejecutar ni modificar:

```text
/home/alberto/Trabajo/orquesta
/home/alberto/Trabajo/orquesta-rebuild/modulos
/home/alberto/Trabajo/orquesta-rebuild/cmd/* salvo cmd/orquesta
```

El árbol antiguo es solo evidencia histórica. No se usa como servidor,
workspace, fallback, bridge ni fuente de configuración.

Requisitos locales:

- Go y dependencias vendorizadas del repo;
- `codex` autenticado para el worker; esta autenticación es distinta del Bearer
  que usa el agente externo contra Orquesta;
- puerto `127.0.0.1:8080` libre, o cambiarlo por otro loopback;
- directorios privados, espacio suficiente y ningún servidor usando los mismos
  roots.

El comando MCP de Codex descrito abajo fue comprobado con `codex-cli 0.144.3`.

## 2. Preparar runtime y configuración `local_token`

Este equipo usa `local_token`. El ejemplo crea una instancia aislada fuera del
repo. Si se elige otra raíz, todas las sustituciones deben apuntar a rutas
absolutas equivalentes y no solapadas.

```bash
export ORQUESTA_RUNTIME_ROOT=/home/alberto/.local/state/orquesta-rebuild/v12-external
umask 077
install -d -m 700 \
  "$ORQUESTA_RUNTIME_ROOT" \
  "$ORQUESTA_RUNTIME_ROOT/bin" \
  "$ORQUESTA_RUNTIME_ROOT/config" \
  "$ORQUESTA_RUNTIME_ROOT/state" \
  "$ORQUESTA_RUNTIME_ROOT/artifacts" \
  "$ORQUESTA_RUNTIME_ROOT/work" \
  "$ORQUESTA_RUNTIME_ROOT/secrets"
cp config/orquesta.toml.example "$ORQUESTA_RUNTIME_ROOT/config/orquesta.toml"
chmod 600 "$ORQUESTA_RUNTIME_ROOT/config/orquesta.toml"
sed -i \
  -e "s|./var/state/orquesta.sqlite|$ORQUESTA_RUNTIME_ROOT/state/orquesta.sqlite|" \
  -e "s|./var/artifacts|$ORQUESTA_RUNTIME_ROOT/artifacts|" \
  -e "s|./var/secrets/credentials.json|$ORQUESTA_RUNTIME_ROOT/secrets/credentials.json|" \
  -e "s|./var/work|$ORQUESTA_RUNTIME_ROOT/work|" \
  -e "s|./var/secrets/local-owner.token|$ORQUESTA_RUNTIME_ROOT/secrets/local-owner.token|" \
  -e "s|./var/effective_config.json|$ORQUESTA_RUNTIME_ROOT/effective_config.json|" \
  "$ORQUESTA_RUNTIME_ROOT/config/orquesta.toml"
```

Comprobar en el TOML:

```toml
[server]
listen = "127.0.0.1:8080"
mcp_path = "/mcp"

[runtime]
provider = "codex"

[runtime.codex]
command = "codex"
reasoning = "medium"
max_concurrent_executions = 70
env_allowlist = ["PATH", "HOME", "CODEX_HOME"]
credential_ref = ""

[identity]
provider = "local_token"
local_actor = "actor:local-owner"
local_token_path = "/home/alberto/.local/state/orquesta-rebuild/v12-external/secrets/local-owner.token"

[project]
default = "project:default"
```

No crear otra variable de servidor ni otro fichero de secretos. El registro
canónico es `config/registry.json`; el TOML contiene valores no sensibles y
refs. `effective_config.json` es una salida redactada, nunca una entrada.

Construir el binario desde el checkpoint elegido:

```bash
VERSION="$(git rev-parse --short=12 HEAD)"
go build -mod=vendor -trimpath \
  -ldflags "-X main.version=$VERSION" \
  -o "$ORQUESTA_RUNTIME_ROOT/bin/orquesta" \
  ./cmd/orquesta
"$ORQUESTA_RUNTIME_ROOT/bin/orquesta" version
```

## 3. Arrancar servidor

Terminal A, en primer plano:

```bash
cd /home/alberto/Trabajo/orquesta-rebuild
export ORQUESTA_RUNTIME_ROOT=/home/alberto/.local/state/orquesta-rebuild/v12-external
"$ORQUESTA_RUNTIME_ROOT/bin/orquesta" serve \
  --config "$ORQUESTA_RUNTIME_ROOT/config/orquesta.toml"
```

El servidor solo admite listeners TCP loopback. En el primer arranque crea el
Bearer en `identity.local_token_path`, con directorio `0700` y fichero `0600`.
Un fichero existente inválido, enlazado o con permisos incorrectos bloquea el
arranque; no se sustituye silenciosamente.

`/mcp` es MCP Streamable HTTP, no un endpoint REST. No probarlo con llamadas
`curl` inventadas.

## 4. Registrar Orquesta en Codex sin mostrar el Bearer

Terminal B. Registrar una vez el nombre de la variable, nunca su contenido:

```bash
codex mcp add orquesta-rebuild-v12 \
  --url http://127.0.0.1:8080/mcp \
  --bearer-token-env-var ORQUESTA_MCP_TOKEN
codex mcp get --json orquesta-rebuild-v12
```

Si el conector ya existe, verificar URL y nombre de variable en vez de borrarlo
o sobrescribirlo a ciegas.

Cargar el secreto sin imprimirlo, iniciar una sesión nueva y retirarlo al
salir:

```bash
export ORQUESTA_RUNTIME_ROOT=/home/alberto/.local/state/orquesta-rebuild/v12-external
test -r "$ORQUESTA_RUNTIME_ROOT/secrets/local-owner.token"
ORQUESTA_MCP_TOKEN="$(<"$ORQUESTA_RUNTIME_ROOT/secrets/local-owner.token")"
export ORQUESTA_MCP_TOKEN
codex
unset ORQUESTA_MCP_TOKEN
```

No usar `echo`, `printenv`, `env`, `set -x` ni pasar el token como argumento.
Una sesión Codex ya abierta puede no descubrir un conector añadido después;
abrir otra con `ORQUESTA_MCP_TOKEN` disponible.

## 5. Seis tools reales y flujo recomendado

El servidor expone exactamente:

1. `orquesta.system.status` — readiness y contadores del proyecto.
2. `orquesta.goals.create` — crea de forma idempotente intención, AppSpec,
   Goal y plan/DAG confirmado.
3. `orquesta.goals.get` — devuelve Goal, WorkItems, ejecuciones, artifacts y
   attestations.
4. `orquesta.goals.list` — lista Goals visibles del proyecto.
5. `orquesta.artifacts.read` — lee un artifact inmutable como Base64, con
   digest, media type y tamaño.
6. `orquesta.goals.amend` — crea un sucesor causal de un Goal terminal usando
   revisión y AppSpec hash exactos.

Todas exigen `project_ref`. Actor, principal y tiempo proceden del transporte
autenticado y del servidor; nunca se envían en el payload.

### 5.1 Estado primero

Llamar mediante la tool MCP, no por shell:

```json
{
  "project_ref": "project:default"
}
```

Exigir `ready: true`. Revisar `running_goals`, `pending_actions` y
`quarantined_actions` antes de crear trabajo nuevo.

### 5.2 Crear un Goal sencillo

Para una primera prueba, omitir `plan` crea un WorkItem por defecto:

```json
{
  "project_ref": "project:default",
  "request_ref": "request:external:smoke:001",
  "statement": "Devuelve exactamente ORQUESTA_EXTERNAL_AGENT_OK y una frase breve.",
  "normalized_objective": "Producir una evidencia textual mínima.",
  "confirm": true
}
```

Conservar `goal_ref`, `revision`, `app_spec.hash`, `app_spec.generation` y
`plan_generation`. Repetir la misma `request_ref` con idéntico payload devuelve
el mismo Goal con `created:false`; cambiar su significado exige una ref nueva.

Para trabajo paralelo, usar un DAG explícito. Este ejemplo lanza dos items con
write-sets disjuntos:

```json
{
  "project_ref": "project:default",
  "request_ref": "request:external:review:001",
  "statement": "Preparar dos análisis independientes sobre REVISION_BASE.",
  "normalized_objective": "Entregar propuestas revisables sin modificar el repositorio.",
  "confirm": true,
  "plan": {
    "phases": [
      {
        "ref": "phase-instance:analysis",
        "key": "phase:analysis",
        "template_ref": "phase-template:analysis",
        "input_refs": ["input:revision-base"],
        "criterion_refs": ["criterion:scope-and-tests"]
      }
    ],
    "work_items": [
      {
        "key": "domain-review",
        "objective": "Contexto autocontenido y fragmentos exactos aquí. Revisa solo internal/domain y entrega hallazgos o diff unificado con tests.",
        "phase": "phase:analysis",
        "role": "role:reviewer",
        "dependencies": [],
        "write_set": ["internal/domain"],
        "skill_refs": [],
        "tool_refs": [],
        "capability_refs": [],
        "output_contract": "evidence_bundle"
      },
      {
        "key": "adapter-review",
        "objective": "Contexto autocontenido y puerto exacto aquí. Revisa solo internal/adapters/x y entrega hallazgos o diff unificado con negativos y tests.",
        "phase": "phase:analysis",
        "role": "role:reviewer",
        "dependencies": [],
        "write_set": ["internal/adapters/x"],
        "skill_refs": [],
        "tool_refs": [],
        "capability_refs": [],
        "output_contract": "evidence_bundle"
      }
    ]
  }
}
```

Claves de `dependencies` y `parent` son locales a la petición. Los write-sets
son rutas relativas limpias, sin globs ni rutas absolutas. Items listos con
write-sets solapados se serializan; los disjuntos forman la cohorte máxima.

### 5.3 Consultar hasta terminalidad

Llamar `orquesta.goals.get` cada uno o dos segundos:

```json
{
  "project_ref": "project:default",
  "goal_ref": "GOAL_REF_DEVUELTA_POR_CREATE"
}
```

`succeeded` exige la evidencia mínima declarada por cada WorkItem. `failed`
exige revisar `executions[].failure_code`; no editar SQLite. Si hay acciones en
cuarentena, conservar refs y registrar bloqueo: todavía no existe tool pública
de rescate.

Para enumerar los Goals visibles del proyecto, usar `orquesta.goals.list`:

```json
{
  "project_ref": "project:default",
  "limit": 50
}
```

`limit` es opcional. No usar la enumeración como sustituto de
`orquesta.goals.get` para comprobar artefactos, ejecuciones o cierre causal.

### 5.4 Leer y validar artifacts

Tomar cada `artifact_ref` de `goals.get`:

```json
{
  "project_ref": "project:default",
  "goal_ref": "GOAL_REF",
  "artifact_ref": "ARTIFACT_REF"
}
```

La respuesta declara `encoding:"base64"`, `content_base64`, `digest`,
`media_type` y `size`. El director externo decodifica a un temporal privado y
verifica digest, scope, revisión base, secretos, arquitectura y tests antes de
usar el contenido. No aplicar automáticamente texto porque el Goal cerró.

### 5.5 Amendment causal

Solo sobre un Goal terminal. Copiar de `goals.get` la `revision` y
`app_spec.hash` exactas. El hash son 64 caracteres hexadecimales minúsculos,
sin prefijo `sha256:`:

```json
{
  "project_ref": "project:default",
  "request_ref": "request:external:amend:001",
  "source_goal_ref": "GOAL_REF_TERMINAL",
  "expected_source_revision": 6,
  "expected_source_spec_hash": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
  "statement": "Corregir la intención preservando la historia anterior.",
  "normalized_objective": "Crear un sucesor causal para la corrección.",
  "reason": "El operador pidió ajustar el alcance tras revisar la evidencia.",
  "confirm": true
}
```

Sustituir el hash de ejemplo por el valor exacto devuelto por `goals.get`.

En el checkpoint actual, `amend` crea el sucesor causal pendiente, pero no le
añade plan, outbox ni ejecución. No usarlo como continuación automática de
programación; para ejecutar una corrección, crear un Goal nuevo autocontenido y
con otra `request_ref`, conservando la relación en el informe del director.

## 6. Qué debe hacer el agente externo

Instrucción lista para otra sesión:

```text
Trabaja como director externo de la nueva Orquesta. Lee AGENTS.md y
docs/reconstruccion/uso_orquesta_con_agente_externo.md completos. Usa solo
/home/alberto/Trabajo/orquesta-rebuild y la rama
reconstruccion/orquesta-total-20260714; no uses runtime, código ni estado de
/home/alberto/Trabajo/orquesta y no uses codebase-memory.

Primero llama orquesta.system.status con project_ref=project:default. Inspecciona
el proyecto objetivo con rg y lecturas acotadas. Declara revisión base,
write-sets, dependencias y tests. Compila Goals/DAG autocontenidos; paraleliza
solo write-sets disjuntos. Los workers actuales son read-only, no reciben el
workspace del proyecto ni contexto implícito: incluye fragmentos y contratos
mínimos en cada objective y pide artifacts/diffs, no efectos.

Consulta goals.get, lee cada artifact por artifacts.read y valida digest,
scope, secretos, base y tests. Solo aplica cambios si el encargo lo autoriza;
hazlo en worktree aislado con apply_patch y prueba fuera de Orquesta. Commit,
merge, push, deploy y publicación requieren autorización separada. Goal
succeeded no equivale a cambio integrado. Entrega refs de Goal/AppSpec,
executions/artifacts, tests, riesgos y bloqueos.
```

## 7. Limitaciones tras V14

- V13 cerrado: existe mailbox durable interno para `child_delivery`, con
  `admitted → claimed → delivered → consumed → acknowledged|blocked` y
  `retired` sistémico. `Parent` y `HandoffRequired` son contratos distintos:
  el segundo es explícito, `false` por defecto y no puede ser `true` sin padre.
  Sigue sin superficie HTTP/MCP/CLI: las seis tools públicas no exponen ese
  opt-in ni permiten claim, deliver, consume o ACK. Por tanto el DAG V05
  público conserva `false` y cierra sin mailbox/requeue. Mensajes genéricos,
  sesiones reanudables y handoff entre proveedores (`ORC-15`) quedan en V27.
- V14 acreditado: los casos internos de aplicación para pause/resume, cancel,
  stop selectivo cooperativo/forzado, retry de Execution y replan causal están
  cerrados por receipt V3 `PASS`. Sus bindings HTTP/MCP/CLI públicos se
  incorporarán mediante el registro único de V20; V14 no añade tools ad hoc.
- V15 no está abierto: faltan presupuestos completos, fairness, riesgo y
  effects/approvals. La próxima sesión empieza por su análisis, no por código.
- V16: faltan workspace, worktree, aplicación de patch y receipts Git; los
  workers actuales no editan el proyecto objetivo.
- V17-V19: faltan atestador independiente completo, autor/reviewers/refinery y
  Consejo de Sabios gobernados.
- V20-V21: faltan registro único de comandos y paridad/i18n total entre HTTP,
  MCP, CLI y futuras superficies.
- V22: falta el E2E de autodirección completa. Hasta entonces Codex externo
  prepara contexto, integra y verifica.
- V23-V24: faltan Wizard y web/PWA administrativa.
- V25+: faltan Hermes, Claude/Gemini/Ollama/local, tools/skills/plugins, RAG,
  deploy, OPES, PostgreSQL/S3/multihost y operación completa.

No simular estas capacidades con scripts laterales ni meterlas en el núcleo.
El protocolo V12 de Director, el mailbox V13 y los controles V14 existen
internamente, pero aún no tienen bindings en las seis tools MCP actuales.

## 8. Verificación y E2E

Desde un checkout limpio, validar el último cierre acreditado V01-V14:

```bash
cd /home/alberto/Trabajo/orquesta-rebuild
go test -mod=vendor -count=1 ./acceptance \
  -run '^TestAcceptanceV(0[1-9]|1[0-4]).*Receipt$'
```

Validar el E2E interno de controles mediante la composición productiva:

```bash
go test -mod=vendor -count=1 ./internal/bootstrap \
  -run '^(TestRealCodexControlsThroughProductionComposition|TestRealCodexCooperativeStopLeavesResidentSchedulerLive)$'
```

Este E2E construye y arranca la composición, crea una Execution viva, solicita
stop forzado exacto y exige control confirmado, receipt durable, Execution
`stopped`, WorkItem `interrupted` y Goal abierto; el segundo demuestra progreso
del scheduler único ante `SIGTERM` ignorado y escalada forced posterior. No
usan binding público de control: forman parte del argv sellado ya acreditado.

Validar API MCP pública, DAG y shutdown con composición aislada de test:

```bash
go test -mod=vendor -count=1 ./internal/bootstrap \
  -run '^(TestRealMCPAPIClosesDurableGoalThroughSQLiteAndArtifactStore|TestMCPCreatesAndExecutesDiamondDAGAtomically|TestShutdownTerminatesAndReapsOwnedAgentProcess)$'
```

Gate proporcional del producto nuevo:

```bash
git diff --check
scripts/check_rebuild_write_set.sh
go test -mod=vendor -count=1 .
go test -mod=vendor -count=1 ./internal/... ./cmd/orquesta
GOFLAGS=-mod=vendor go vet ./internal/... ./cmd/orquesta
```

No usar `go test ./...`: atraviesa superficies congeladas que no forman parte
del gate del producto nuevo.

E2E opt-in con Codex real: detener antes cualquier servidor que use esa
configuración, usar un TOML dedicado con `server.listen="127.0.0.1:0"` y roots
vacíos/aislados, y ejecutar:

```bash
go test -mod=vendor -v -count=1 ./internal/bootstrap \
  -run '^TestRealCodexAdapterClosesGoalThroughProductionMCPServer$' \
  -args -orquesta-real-codex-config=/ruta/absoluta/orquesta-real-e2e.toml
```

La prueba debe cerrar un Goal por servidor MCP de producción, leer un artifact
con marcador único y apagar todos sus procesos. No apuntarla al runtime legacy
ni al estado operativo que se quiera conservar.

Última comprobación real: `PASS` en `6.15s`, ejecutada
`2026-07-16T12:12:56+02:00` con configuración aislada
`/home/alberto/Trabajo/.orquesta-rebuild-real-e2e-v13-20260716T115042/orquesta.toml`,
source digest
`sha256:d3bac625fa52b76fdc2c0007292577751ca8931a6ca890ecd2751a8976fa8d36`
y marcador `ORQUESTA_CODEX_E2E_OK_70c61a97f86425f3464a7e12e9b9829a`.
Prueba composición productiva, MCP, Codex, SQLite y CAS sin regresión; no activa
mailbox ni acredita por sí sola el contrato V13. Esa acreditación procede del
receipt V3 separado `product/evidence/v13_mailbox.json`.

## 9. Seguridad, parada y limpieza

Reglas de seguridad:

- mantener MCP en loopback; no exponer `8080` a red;
- `local_token` concede `project_owner` local: tratarlo como secreto de alto
  privilegio, no copiarlo a TOML, effective-config, prompts, logs, artifacts,
  commits o handoffs;
- el Bearer autentica al director contra Orquesta; la credencial que usa el
  worker Codex es independiente;
- mantener allowlist del hijo mínima. `read-only` evita escrituras soportadas,
  pero no es una frontera de confidencialidad OS frente a otro proceso del
  mismo usuario; no ejecutar material no confiable ni pasar rutas sensibles;
- no editar SQLite, artifacts CAS ni receipts a mano;
- OIDC es opt-in y requiere issuer/audience/grupos y membresía provisionada;
  para este quickstart usar exactamente un proveedor: `local_token`.

Parar Terminal A con `Ctrl-C` o enviar `SIGTERM` al PID propio. El servidor
cancela scheduler y workers, recolecta procesos, cierra HTTP, artifacts y
SQLite. No usar `kill -9` salvo incidente documentado.

Después:

```bash
unset ORQUESTA_MCP_TOKEN
pgrep -af '/orquesta.*serve' || true
git -C /home/alberto/Trabajo/orquesta-rebuild status --short --branch
```

`pgrep` solo diagnostica: no matar por patrón amplio. Conservar el runtime para
reanudar con el mismo TOML; restart recupera SQLite/outbox y no relanza trabajo
terminal. Borrar roots únicamente tras confirmar que no hay proceso vivo y que
Goals, artifact refs, receipts y evidencia ya no se necesitan. Retirar el
conector Codex solo por decisión explícita:

```bash
codex mcp remove orquesta-rebuild-v12
```
