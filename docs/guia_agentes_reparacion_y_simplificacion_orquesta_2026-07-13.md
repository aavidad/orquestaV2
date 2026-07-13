# Guía para reparar y simplificar Orquesta

Fecha: 2026-07-13
Fuente de diagnóstico: [informe_auditoria_integral_orquesta_2026-07-13.md](informe_auditoria_integral_orquesta_2026-07-13.md)
Objetivo: cerrar los huecos reales sin seguir multiplicando código, rutas ni
autoridades.

## 1. Mandato para los agentes

No intentéis “arreglarlo todo” en un diff grande. Cada tarea debe restaurar un
invariante observable, declarar un write-set estrecho y producir una prueba que
falle si se retira el efecto. La integración es serial cuando dos tareas tocan
la misma autoridad, aunque el análisis y los adapters disjuntos se paralelicen.

Reglas de fuego:

1. Goal-first es la única ruta productiva que recibe features nuevas.
2. `legacy_director_loop` queda congelado para mantenimiento correctivo.
3. V2 offline no se conecta como otra ruta productiva sin retirar una autoridad
   equivalente.
4. Registrado, implementado, cableado, ejercitado y acreditado son estados
   distintos.
5. Un ACK de admisión no es un receipt de efecto.
6. El self-result del autor no acredita el cambio.
7. Ninguna identidad se acepta desde texto o JSON producido por el propio
   agente si existe un launch/receipt que puede acreditarla.
8. No se crean más lecturas de entorno, defaults o claves fuera del registro
   canónico.
9. No se copian primitivas de seguridad por provider.
10. No se borra legacy, documentos o stores hasta demostrar referencias,
    paridad, migración y replay.
11. No se añade un rail por palabras para compensar identidad o estado mal
    modelados.
12. Todo bug observado se enlaza desde el inventario de bugs.

Antes de editar, cada agente debe responder en su task:

```text
invariante que restaura:
autoridad que escribe:
write-set:
callers afectados:
estado previo y estado objetivo:
mutación que debe quedar roja sin el cambio:
E2E de cierre:
dependencias causales:
```

## 2. Arquitectura objetivo mínima

```text
API/MCP
  -> normalizador de intención
  -> IntentManifestStore (inmutable)
  -> GoalRepository (única autoridad de lifecycle)
  -> Scheduler/Observer residente único
  -> ProviderAdapter
       ├─ Codex
       ├─ Claude
       └─ Gemini
  -> Artifact/ReceiptStore
  -> Attestation/Review
  -> Closure/Promotion
```

### 2.1 Qué significa “única autoridad”

- `IntentManifest` manda sobre la intención original.
- `Goal` manda sobre identidad, generación, workspace asignado y lifecycle.
- El launch receipt manda sobre identidad del agente/provider.
- El artifact/attestation receipt manda sobre árbol, tests y efecto.
- El capability ledger manda sobre qué puede prometer el producto.
- `Run`, `PlanState`, `WorkflowTask`, dashboards y listados solo proyectan;
  nunca deciden un cierre paralelo.

Si un agente necesita añadir un store, debe decir qué autoridad reemplaza o por
qué es una proyección. “Hace más fácil el wiring” no basta.

### 2.2 Interfaces que sí merecen existir

```go
type IntentManifestStore interface { /* create once + read verified */ }
type GoalRepository interface { /* CAS, lease, generation, replay */ }
type ProviderAdapter interface { /* launch, observe, deliver, stop */ }
type ArtifactStore interface { /* immutable, secure, content-addressed */ }
type CredentialStore interface { /* resolve authorized ref, never expose */ }
type CapabilityLedger interface { /* declared -> accredited */ }
```

Los providers no deberían implementar de nuevo traversal seguro, hardlink
checks, manifest canonicalization, config parsing o políticas de cierre.

## 3. Orden causal de reparación

### Ola 0 — Contener los fallos inmediatos

Estas tareas son pequeñas y pueden prepararse en paralelo, pero deben integrarse
en workspaces separados y sobre una fotografía estable.

#### ORQ-FIX-0A — Autorización real de shutdown

- Invariante: solo el principal/rol autorizado puede parar el servidor.
- Write-set: módulo `orquesta-server-shutdown`, wiring de identidad y tests.
- Cambio: retirar cualquier decisión basada en substrings de `requested_by`.
- Conservar `requested_by` únicamente como metadata auditiva.
- Tests: caller escribe “director” y es rechazado; operador real es aceptado;
  ausencia de principal falla cerrado; replay no duplica shutdown.
- E2E: shutdown público deja cero procesos propios sin fallback externo.

#### ORQ-FIX-0B — Paridad de schema DomainWork

- Invariante: toda acción aceptada por el handler aparece en schema y catálogo.
- Write-set: schema MCP/capability spec/tests.
- Cambio mínimo: exponer `evaluate_external_capabilities`.
- Cambio estructural: generar enum, descripción, dispatcher test y catálogo
  desde una misma `CapabilitySpec`.
- Mutación: añadir una acción al handler sin spec debe romper el test.

#### ORQ-FIX-0C — Ledger de capacidades

- Invariante: ninguna capacidad se declara cerrada sin receipt proporcional.
- Write-set: nuevo ledger pequeño, generador/reportes; no tocar lifecycle.
- Estados permitidos:

```text
declared
implemented
wired
exercised
accredited
deprecated
revoked
```

- Cada fila lleva revisión/imagen, evidence refs, alcance, provider y fecha.
- Los documentos históricos enlazan el ledger; no duplican su estado vigente.
- H0d entra como `revoked`; 056/V1-B como `declared`; 057 como `implemented`
  o `exercised` solo según receipts reales.

### Ola 1 — Cerrar 057 sin ampliar arquitectura

#### ORQ-057-A — Fijar manifest canónico

- Mantener la captura completa antes de generar `GoalSpec` compactado.
- El manifest se crea una vez, tiene hash canónico y no se reescribe.
- Aclarar si representa toda la request pública o solo la request de dominio.
  Si existe metadata externa que cambia semántica, debe incluirse o tener
  receipt enlazado; no crear un segundo manifest competidor.
- Añadir límites explícitos y error tipado para oversize.

#### ORQ-057-B — Completar store seguro

- Extraer una única primitiva neutral después de congelar los tests actuales.
- Exigir, como mínimo: no symlink, no FIFO/device/socket, owner correcto,
  permisos, tamaño, create-once, fsync, traversal seguro y `Nlink == 1`.
- Añadir ataque real con hardlink, intercambio concurrente y restart.
- No aceptar “ruta bajo root” como sustituto de `openat`/descriptor seguro.

#### ORQ-057-C — Una asignación de workspace

- `Goal` persiste `workspace_ref`, generación y manifest SHA una sola vez.
- Launch, process, observe, stop y attestation resuelven esa asignación.
- Los routers provider traducen; no eligen otra autoridad.
- Colisión por sanitización/truncado debe fallar o desambiguarse con identidad
  estable, nunca compartir carpeta.

#### ORQ-057-D — Matriz E2E común

Ejecutar la misma suite parametrizada para Codex, Claude y Gemini:

1. intención extensa preservada;
2. manifest inmutable;
3. hash estable;
4. create-once concurrente;
5. replay no reescribe;
6. restart recupera;
7. workspace lógico y físico coinciden;
8. launch usa ese workspace;
9. process usa ese workspace;
10. observe usa ese workspace;
11. attestation usa ese árbol;
12. stop usa la misma generación;
13. dos goals no colisionan;
14. symlink rechazado;
15. hardlink rechazado;
16. FIFO/device rechazado;
17. owner/permisos inseguros rechazados;
18. oversize rechazado;
19. prompt/result no filtran datos sensibles;
20. receipt liga manifest, goal, workspace, provider y generación.

Cierre: los 20 invariantes pasan en los tres providers sobre la misma revisión
e imagen; un revisor independiente confirma los receipts.

### Ola 2 — Entrega causal H0d

#### ORQ-H0D-A — Modelo neutral de intervención

El mailbox debe ser una cola, no un log decorativo. Cada orden necesita:

```text
message_ref
goal_ref
target_generation
created_by_principal
expected_delivery_mode
claim/lease
delivery_attempt
delivery_receipt_ref
effect_or_rework_ref
```

- Goal interactivo: entregar un turn a la generación activa.
- Backend no interactivo: crear successor/bundle causal.
- Si la generación cambió, replan explícito; no entregar silenciosamente al
  proceso nuevo.
- ACK de admisión, ACK de claim y receipt de delivery son eventos distintos.

Tests de cierre:

- tool pública→mailbox→turn con nonce visible en prompt→delivery ACK;
- dos consumidores en carrera, una sola entrega;
- crash entre claim y ACK, recuperación por lease;
- replay no redelivery;
- goal terminal produce rechazo/rework tipado;
- provider no interactivo crea successor ligado al mensaje.

### Ola 3 — Stop selectivo 056

#### ORQ-056-A — Ownership de recursos

- Registrar PID/session/backend por `goal_ref` y `generation`.
- Diferenciar recurso exclusivo de backend compartido.
- Stop de un goal termina solo recursos exclusivos y desregistra su suscripción
  del compartido.
- El receipt enumera recursos intentados, detenidos, preservados y pendientes.

#### ORQ-056-B — E2E concurrente obligatorio

```text
Launch A
Launch B
Observe A vivo
Stop B
Observe A sigue vivo y avanza
Observe B terminal
Restart
Replay Stop B sin efectos nuevos
```

No cerrar 056 con mocks de process registry ni con shutdown global.

### Ola 4 — Configuración e idle en paralelo

La fundación de configuración y la reducción de loops pueden avanzar en
paralelo si no comparten `cmd/orquesta-server` en el mismo workspace. Integrar
una y rebasar la otra antes de tocar wiring común.

#### ORQ-CFG-A — Las tres superficies

No crear `public.env`, `private.env` y otro `.env`. Eso mantendría strings y
tres fuentes editables. Crear/consolidar estas superficies:

##### 1. `orquesta.config.json`

- Editable y no secreto.
- Schema/version explícitos.
- Valores tipados: bool, duration, integer, enum, URL y listas.
- Puede contener referencias como `credential_ref`.
- No contiene tokens, passwords, private keys ni material OAuth.

Ejemplo orientativo:

```json
{
  "schema_version": "orquesta.config.v1",
  "server": {"listen_address": "127.0.0.1:8080"},
  "runtime": {
    "default_provider": "codex",
    "provider_credentials": {"codex": "credential-ref-codex-default"}
  },
  "supervisor": {"idle_backoff": "30s"}
}
```

##### 2. `orquesta.credentials.json` o `CredentialStore`

- No versionado, directorio privado, ficheros `0600`, owner comprobado.
- Store local solo como adapter; el puerto debe permitir secret manager.
- Campos mínimos: `credential_ref`, `provider`, `kind`, `owner_ref`, `scope`,
  `created_at`, `expires_at`, `revoked_at` y secreto cifrado/restringido.
- El Goal guarda solo `credential_ref`.
- Ni prompts, resultados, logs, receipts ni `effective_config` incluyen el
  secreto.

Ejemplo de forma lógica, no para Git:

```json
{
  "schema_version": "orquesta.credentials.v1",
  "credentials": [{
    "credential_ref": "credential-ref-codex-default",
    "provider": "codex",
    "kind": "api_token",
    "owner_ref": "principal-ref-operator",
    "scope": ["goal.launch"],
    "secret": "<material privado>"
  }]
}
```

##### 3. `effective_config.json`

- Generado tras resolver config; nunca editable.
- Redactado y apto para diagnóstico/receipts.
- Incluye valor efectivo no sensible, fuente, hash/revisión y restart.
- Para secretos muestra solo `credential_ref` y estado
  `available|missing|revoked`, nunca material.

Ejemplo orientativo:

```json
{
  "schema_version": "orquesta.effective_config.v1",
  "revision": "sha256:...",
  "entries": {
    "/runtime/default_provider": {
      "value": "codex",
      "source": "file",
      "restart_required": true
    },
    "/runtime/provider_credentials/codex": {
      "credential_ref": "credential-ref-codex-default",
      "credential_status": "available",
      "source": "file",
      "redacted": true
    }
  }
}
```

La “tercera” pieza que se buscaba es, con alta probabilidad, este snapshot de
configuración efectiva. Debe confirmarse con la decisión histórica si aparece
un documento posterior, pero es la división que resuelve operación y seguridad
sin crear otra autoridad editable.

#### ORQ-CFG-B — Registro único de claves

Un solo `ConfigKeySpec` contiene toda la metadata:

```go
type ConfigKeySpec struct {
    JSONPointer      string
    EnvName          string   // solo bootstrap o alias de migración
    Aliases          []string
    ValueType        ValueType
    Default          any
    Sensitive        bool
    Scope            ConfigScope
    RestartRequired  bool
    Validate         func(any) error
}
```

No mantener por separado un env registry, policies de default/secret/restart y
un mapper de effective config que repitan la misma clave. Se pueden generar
vistas desde `ConfigKeySpec`, pero no editarlas como nuevas autoridades.

#### ORQ-CFG-C — Un único ingreso por ejecutable

```text
LoadServerConfig(environ, configFile, credentialStore) -> ResolvedServerConfig
LoadGuardianConfig(environ, configFile, credentialStore) -> ResolvedGuardianConfig
```

- `cmd/orquesta-server` y `cmd/orquesta-guardian` pueden tener loaders distintos
  porque son apps distintas, pero comparten parser/registry primitives.
- Después del bootstrap, ningún módulo llama a `os.Getenv`/`os.LookupEnv`.
- Procesos hijos reciben una proyección explícita desde `ResolvedConfig`, no un
  barrido de prefijo `ORQUESTA_*`.
- Defaults se aplican una sola vez en el loader.
- Aliases temporales se declaran en el registry, emiten warning/receipt y llevan
  fecha/versión de retirada.

Precedencia recomendada:

```text
valor explícito canónico de bootstrap permitido
    > orquesta.config.json
    > default del ConfigKeySpec
```

Los secretos no participan en esa precedencia como valores: se resuelven por
`credential_ref` y autorización. Cualquier override de env debe estar declarado
clave por clave; nunca por prefijo amplio.

#### ORQ-CFG-D — Migración sin big bang

1. Congelar el número actual con un inventario repo-wide por AST.
2. Clasificar cada lectura: bootstrap, config no secreta, secreto, alias,
   test-only o inválida.
3. Crear spec canónico antes de mover el primer caller.
4. Migrar una familia semántica por PR: server, goals, providers, guardian,
   rails y procesos hijos.
5. Conservar alias máximo dos releases con warning y métrica.
6. Bajar el baseline en cada PR; nunca aumentarlo.
7. Cuando llegue a cero, prohibir llamadas directas fuera de los dos loaders.
8. Solo entonces implementar escritura V1-A con CAS, audit receipt y
   `effective_config` nuevo.

No renombrar variables “para ordenarlas” sin migrar semántica: eso duplica
claves. Antes de crear una variable nueva, buscar JSON pointer, env canónico,
aliases y salida efectiva.

#### ORQ-CFG-E — Tests de cierre

- AST guard repo-wide, incluidos guardian, runtimes y rails.
- Una clave nueva fuera del registry rompe el build.
- Unknown JSON keys fallan con path útil.
- Precedencia se prueba para cada fuente permitida.
- Env alias y clave canónica simultáneos fallan por ambigüedad o siguen regla
  explícita probada.
- Defaults aparecen idénticos en runtime y `effective_config`.
- Restart-required se proyecta correctamente.
- Secretos conocidos no aparecen en prompt, logs, JSON público ni receipts.
- Owner/scope/revocation de credenciales tienen negativos.
- Server y daemon/guardian ven la misma revisión de config.
- Escritura futura usa CAS/atomic replace/fsync y genera receipt.

#### ORQ-OPS-A — Unificar observación residente

- Inventariar todos los tickers de 250 ms/5 s y su caller.
- Elegir un scheduler/observer residente como dueño.
- Usar wakeups por cambio durable y backoff cuando no hay activos.
- Listados “active” filtran terminales antes de cargar artefactos extensos.
- Reconciliación stale tiene presupuesto y marca de progreso.
- Watchdog observa telemetría; no crea un loop de negocio paralelo.

Cierre medido:

- imagen nueva, state vacío y state con historial;
- 10 minutos sin trabajo, muestras CPU/memoria/PIDs;
- wakeup inmediato al insertar un goal;
- vuelta a backoff tras terminal;
- shutdown público limpia todos los procesos;
- comparación before/after guardada como receipt.

### Ola 5 — Credenciales V1-B

#### ORQ-V1B-A — Store, autorización y recibo de uso

- Implementar el puerto `CredentialStore` antes de la UI de configuración.
- Resolver por principal acreditado, owner, scope, provider y estado.
- Entregar el secreto solo al adapter que ejecuta; no al Goal ni al agente.
- Receipt de uso: referencia, versión, provider, owner autorizado, purpose y
  resultado redacted.
- Revocación impide nuevos launches; define qué ocurre con uno ya activo.
- Rotación conserva trazabilidad de versión sin conservar secreto en receipts.

E2E: dos propietarios, misma clase de provider; A usa su ref, B no puede usarla;
rotación/revocación y búsqueda de leaks en todos los artefactos.

### Ola 6 — Escritura gobernada V1-A/A2

- API/MCP modifica solo `orquesta.config.json`, nunca el store de credenciales
  mediante el mismo endpoint genérico.
- Lectura→expected revision→validate→atomic write→new revision→effective
  snapshot→audit receipt.
- Cambios de restart quedan `pending_restart`, no se presentan como aplicados.
- Routing/model catalog consume `ResolvedConfig`, no env ad hoc.
- UI es adapter; toda regla vive en el servicio de configuración.

### Ola 7 — Consejo 058 y dos reviews independientes

#### ORQ-058-A — Consejo opcional

Política exacta:

- `auto`: el Director decide con receipt razonado.
- `required`: no puede cerrar sin convocatoria/votos válidos.
- `skip_by_operator`: bypass explícito y durable con principal acreditado.

El bypass del consejo no elimina la revisión de implementación.

`voter_ref` se deriva del launch/ACK, no del contenido del voto. Cada voto liga
invitation, launch, delivery, goal, generación y árbol/decisión revisada.

#### ORQ-REV-B — Dos revisores reales

- Dos launches distintos, identidad/familia conforme a la política.
- Ambos revisan el mismo commit/tree y los mismos tests requeridos.
- No vale que un agregador produzca dos nombres.
- Si el árbol cambia tras una review, ambas quedan stale.
- El consejo puede omitirse; estas reviews no.

### Ola 8 — Simplificar después de fijar comportamiento

#### ORQ-SIM-A — Primitivas provider compartidas

Extraer del código 057 solo después de que la matriz común esté verde:

- secure file/openat validation;
- manifest/artifact store;
- workspace assignment/lookup;
- receipt envelope;
- ataque/test suite parametrizada.

El adapter de provider debe quedar reducido a launch, observe, deliver y stop,
más la traducción de errores propia del proveedor.

#### ORQ-SIM-B — Retirada de generaciones

Para cada ruta legacy:

1. inventariar callers reales y smokes;
2. mapear a Goal-first;
3. ejecutar paridad y replay;
4. marcar deprecated en capability ledger;
5. eliminar wiring productivo;
6. conservar migrador/reader si hay estado durable antiguo;
7. borrar solo en una tarea posterior con evidencia de cero referencias.

Objetivo verificable: una sola ruta productiva, no “menos nombres”.

#### ORQ-SIM-C — Reducir superficie pública

No conectar automáticamente toda tool histórica. Clasificar:

- productiva acreditada;
- experimental fuera de promesa;
- compatibilidad/deprecated;
- interna no pública;
- inválida/solapada.

Generar schema, dispatcher, catálogo y tests desde el mismo spec. Una tool
deprecada debe comunicar sucesora y fecha, no seguir pareciendo principal.

#### ORQ-SIM-D — Documentación como proyección

- `AGENTS.md` conserva reglas estables y frontera conceptual.
- Capability ledger conserva estado actual y evidencias.
- Inventario de bugs conserva incidencias y cierres.
- Runbooks explican operación.
- `CODEX_LEEME.md` y cortes históricos son log, no snapshot vigente.

No seguir añadiendo “cerrado” al final de documentos que contienen “abierto”
sin actualizar una cabecera canónica.

### Ola 9 — Acreditación final de producto

Ejecutar sobre un commit limpio, una imagen identificada y stores nuevos:

#### E2E APP-A — Consejo omitido explícitamente

```text
API/MCP create app
-> manifest íntegro
-> credential_ref autorizado
-> skip_by_operator receipt
-> implementación real
-> dos reviews independientes
-> tests requeridos
-> closure/promoción
-> shutdown limpio
-> restart/replay sin efectos duplicados
```

#### E2E APP-B — Consejo obligatorio

Mismo flujo, con `required`, launches/votos reales y decisión durable. Debe
usar una app/goal/workspace distinto para evitar contaminación del primer caso.

Un revisor independiente valida ambos sobre el mismo commit/imagen y confirma
las mutaciones críticas. Solo entonces puede actualizarse la promesa pública.

## 4. Paralelización segura

Orquesta debe dirigir estos frentes con goals y write-sets disjuntos. Propuesta:

| Ola | Frentes paralelos | Integración serial necesaria |
|---|---|---|
| 0 | auth, MCP schema, ledger docs | wiring común de server |
| 1 | tests provider Codex/Claude/Gemini | store y Goal workspace |
| 4 | config server, config guardian, perfil idle | `cmd/orquesta-server` |
| 5 | store local, auth policy, leak tests | wiring provider |
| 7 | consejo, reviews, identity receipts | promotion gate |
| 8 | shared file store, docs, tool registry | retirada de wiring legacy |

Cada padre conserva `child_task_refs`, ACKs y bloqueos. Ningún padre cierra si
falta un hijo causal. Los agentes no deben usar indexadores ni leer todo el
repo; `rg`, tests focales y refs compactas bastan para estos frentes.

## 5. Política de tamaño y código

Antes de añadir un fichero o abstracción, responder:

1. ¿Qué bug/invariante reproducible lo exige?
2. ¿Existe ya un puerto/store con la misma responsabilidad?
3. ¿Aumenta o reduce el número de autoridades?
4. ¿Aumenta o reduce forks por provider?
5. ¿Qué test queda rojo si se elimina?
6. ¿Puede resolverse eliminando wiring o generando una vista?

Límites prácticos:

- no añadir lógica nueva a controladores de más de 1.000 líneas;
- separar wiring, dominio, filesystem y HTTP antes de crecer el fichero;
- una primitiva de seguridad transversal no vive en tres adapters;
- un registry produce vistas; no se copian sus datos a tres mapas editables;
- cualquier feature nueva debe retirar o deprecar una ruta solapada antes de
  declararse terminada.

Estos límites son señales de revisión, no rails automáticos de contenido.

## 6. Checklist de entrega de cada agente

### Antes

- [ ] HEAD/dirty hash e imagen declarados.
- [ ] `AGENTS.md` local y documentos vigentes leídos.
- [ ] Write-set sin solape declarado.
- [ ] Autoridad afectada identificada.
- [ ] Bug enlazado al inventario.
- [ ] Mutación y E2E definidos antes del código.

### Durante

- [ ] No nueva env/default fuera de registry.
- [ ] No identidad por texto libre.
- [ ] No ACK de admisión presentado como efecto.
- [ ] No provider fork de una primitiva neutral.
- [ ] No feature en legacy.
- [ ] Replay, CAS/lease e idempotencia conservados.
- [ ] Secretos ausentes de prompts/logs/receipts.

### Cierre

- [ ] Unitarios y composición focal verdes.
- [ ] `-race` en stores/lifecycle concurrente.
- [ ] Mutación crítica queda roja sin el efecto.
- [ ] E2E público produce receipt causal.
- [ ] Revisor distinto confirma mismo tree/revisión.
- [ ] `git diff --check` y suite proporcional verdes.
- [ ] Capability ledger actualizado con evidencia, no con afirmación.
- [ ] Inventario conserva la fila y añade prueba/commit de cierre.
- [ ] Cero procesos/caches/temporales creados por la tarea.

Formato final compacto del agente:

```text
Hecho:
Invariante restaurado:
Autoridad final:
Tests/mutaciones/E2E:
Receipts:
Riesgos o bloqueo:
Código/ruta retirada o consolidada:
Siguiente dependencia:
```

## 7. Definición de terminado

No usar porcentajes. Orquesta estará terminada para la promesa auditada cuando:

- no haya P0/P1 abiertos;
- Goal e IntentManifest sean autoridades únicas;
- H0d y 056 tengan E2E causal/replay/race;
- V1-B acredite owner/scope/revoke/redaction;
- consejo `auto|required|skip_by_operator` y dos reviews reales estén separados;
- tres providers pasen la misma matriz 057;
- configuración tenga registro único, store privado y snapshot efectivo;
- no haya lecturas env ad hoc fuera de loaders;
- idle y shutdown estén medidos y limpios;
- schemas, handlers y capability ledger coincidan;
- dos apps reales cierren por API/MCP en la misma release;
- una revisión independiente confirme receipts y mutaciones.

Hasta entonces, el estado debe expresarse por capacidades concretas, no como
“99 %” ni “casi terminado”.

## 8. Handoff vigente tras el cierre 057 Codex

Hasta nueva orden del operador:

1. no integrar ni ampliar Claude, Gemini, Ollama o local; conservar su trabajo
   aislado;
2. usar `388c1bd09f` como base mínima del camino Codex 057;
3. no reabrir permisos de workspace, binding durable, autoridad concurrente ni
   lease tmux salvo regresión reproducida;
4. para observar un Goal en vuelo, exigir `estado=ok`, `goal_status=running` y
   `observe_later`; `invalid` o error genérico es regresión;
5. cerrar solo cuando test independiente, artefacto, integración, commit,
   archive, replay y shutdown público tengan receipts causales;
6. mantener la promoción opt-in: un Goal completo sin puerto de promoción no
   acredita integración canónica;
7. al preparar smokes, crear raíz, state, runtime y stores sensibles con modo
   `0700` antes del primer request.

Referencia de cierre: `request-ref-e2e-057-004`, revisión
`388c1bd09f696fa3938a66e49cf739525c7b44f5`.
