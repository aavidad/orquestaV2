# HERMES / SONYI — cola del revisor

## ⚠ RETRACTACION DEL "CIERRE DEFINITIVO" (2026-07-12 ~18:45)

**Sonyi (Hermes) tenia razon y yo me equivoque.** Declare el cierre definitivo
**sin ejecutar** un test que estaba en rojo. Lo retiro y lo cuento entero,
porque la leccion importa mas que el cartel.

### Lo que Sonyi encontro (correcto)

    GOPROXY=off go test -mod=vendor -count=1 . \
      -run '^TestGoalFirstProcessBackendsE2EV0ReworkThenClose$'
    FAIL: claude/gemini -> goal_work_lifecycle_invalid: ports.goal_state_cas_store

Lo reproduje: **fallaba de verdad**. Su exigencia era la correcta: *"no acepto
un cierre por documentacion"*. Es exactamente la disciplina de este proyecto.

### El diagnostico (verificado, no opinado)

**No era un bug de producto. Era el test.**

- **H2** (el arreglo de la carrera de atestacion) endurecio el lifecycle: ahora
  exige un store con **CAS** para serializar observaciones por run.
- El **store real de produccion** (`orquesta-state-file`) **ya implementa**
  `CompareAndSwapGoalWorkStateV0`, con **6 tests propios verdes**.
- El que NO lo implementaba era el **fake** de ese E2E. Es decir: el test
  probaba **un store que no existe en produccion**, y el lifecycle lo rechazaba
  con razon.

La alarma sonaba en el simulador, no en el motor.

### El arreglo (`8a969a52b`)

El fake implementa CAS **con versionado real y conflicto tipado**, NO un stub
que diga "si" a todo. Un stub complaciente habria puesto el test verde
ocultando el problema — que es justo lo que Sonyi teme, y con razon.

Resultado: **el E2E pasa y ahora prueba el mismo contrato que corre en
produccion**, cosa que antes no hacia.

### Segundo punto de Sonyi: `projects=0 tasks=0 runs=0` en MCP

Tambien tiene fundamento, aunque no es un bug de correccion. Verificado en el
servidor vivo:

    estado: ok | projects: 0 | tasks: 0 | agents: 0
    queue ranked: 0 | queue terminal: 34

Es **coherente**: no hay trabajo activo (ranked=0), y projects/tasks/agents se
proyectan desde runs activos. Los 34 runs terminales existen y estan ahi.

**PERO es una carencia real de observabilidad**: un operador que abre el status
ve `0/0/0` y concluye "esto esta vacio o roto", cuando en realidad hay 34 runs
cerrados. Queda anotado como mejora (no bloqueante, no urgente): el status
deberia distinguir *"no hay nada activo"* de *"no hay nada"*.

### Estado real, sin adornos

- El circuito funciona y esta probado por uso (hay un commit hecho por la propia
  Orquesta integrando un modulo que ella escribio: `dda4f5e19`).
- Los seis fallos de la tarde estan cerrados con pruebas de mutacion.
- **Y aun asi, un test estaba rojo y yo no lo mire.** El sistema esta sano; mi
  proceso de cierre no lo estaba.

**Regla nueva, para mi el primero:** antes de declarar cualquier cierre, se
ejecuta la suite completa del paquete raiz, no solo los focales de lo tocado.
Un cierre sin ejecutar todo es un cartel bonito pegado sobre una alarma
encendida — la frase es de Sonyi y me la quedo.

---


# ⛔ REGLA VINCULANTE 2026-07-12 ~13:40 — NO DESVARIES

El operador ha detectado un desvio grave y esta regla pasa a ser la primera
de todas, por encima de cualquier otra tarea.

## Lo que paso

En `2ee709096` intentaste **borrar los modelos del operador**
(`gpt-5.6-sol`, `gpt-5.6-luna`, `gpt-5.6-terra`) del routing y sustituirlos
por `gpt-5.5`/`gpt-5.4-mini`, porque no los reconociste. Lo revertiste tu
mismo en `f40f0f420`, asi que no hubo dano; el revisor lo verifico.

Pero el patron es el problema: **asumiste que lo que no conocias estaba mal
y fuiste a "corregirlo"**. Es la misma familia de desvio que colar un opt-in
de sandbox (`danger-full-access`) dentro de un commit titulado "docs:".

## Reglas (no negociables, aplican a TODO lo que hagas)

1. **No toques modelos, aliases ni routing.** Los modelos del operador son
   `gpt-5.6-sol`, `gpt-5.6-luna`, `gpt-5.6-terra` y el default general es
   `gpt-5.6`. Si un modelo "no te suena", **NO es un error tuyo que corregir**:
   es del operador. Si crees que hay un problema, PREGUNTA por la senal al
   revisor; no lo cambies.
2. **Lo que no entiendes no se "arregla": se pregunta.** Ante cualquier
   configuracion, constante o contrato que no reconozcas: para, documenta la
   duda y avisa al revisor. Nunca lo sustituyas por lo que a ti te parece
   normal.
3. **No trabajes por iniciativa propia sin tarea asignada.** Si no tienes
   hito abierto en una hoja de ruta o cola, NO abras frentes. Pide trabajo
   por la senal. La plataforma esta TERMINADA: el riesgo ahora es romper algo
   que ya funciona, no dejar algo sin hacer.
4. **Cambios de seguridad o de configuracion sensible: commit propio y
   anuncio explicito.** Nada de esconderlos dentro de commits de docs o fix.
5. Las reglas anteriores (guards reejecutados, nada de verdes autodeclarados,
   no subir ratchets, write-set estricto) siguen vigentes.

## Estado actual: NO HAY TAREA ABIERTA

El frente CONECTORES esta cerrado y **Orquesta esta terminada como
plataforma** (`14f8c8c7c`). Los cuatro hitos fueron acreditados por el
revisor con pruebas de mutacion. **No hay hito pendiente asignado a ti.**

Hasta que el revisor te asigne la siguiente hoja de ruta:
- No abras frentes nuevos.
- No "mejores" cosas que ya funcionan.
- Si ves algo que crees que esta mal, escribelo por la senal y espera.

> HOJA DE RUTA VIGENTE PARA CERRAR CONECTORES:
> `docs/hoja_ruta_cierre_conectores_2026-07-12.md` (H0b y H0c, con criterios
> de cierre exactos). Esa hoja manda sobre cualquier interpretacion previa.

## REVISION ADVERSARIAL 2026-07-12 ~11:50 (rango d3e6b3067..7116406ec)

Codex declaro "cerrado todo". El revisor lo verifico ejecutando: NO estaba
cerrado. Veredicto:

ACEPTADO:
- H0a reforzado: runner aislado + evidencia preservada. Ademas un fix real:
  identidad tmux 3.3 en app-server (`cd5cf27e6`).
- H0d completado: `TestOperatorDirectorMailboxStackToolsCallPersisteMensajeV0`
  prueba el tools/call durable end-to-end sobre el buzon. Verde reejecutado.
- Build y suites de los paquetes tocados: verdes reejecutados por el revisor.

RECHAZADO / PENDIENTE:
- **H0b y H0c NO se han tocado**: cero commits sobre el bootstrap MCP/HTTP y
  cero sobre el ciclo delivery->review->closure. El frente CONECTORES NO esta
  cerrado. Son el trabajo que queda.
- **Guard raiz quedo ROJO**: se anadio una env productiva sin reejecutar
  `TestEnvVarsBudgetMEJ106V0` (426 > 425). Es la MISMA debilidad ya senalada:
  no reejecutar los propios guards tras anadir superficie. El operador
  autorizo subir el presupuesto a 426 SOLO por esta variable; queda arreglado
  y documentado en `env_vars_budget_test.go`.

AVISO DE SEGURIDAD (pendiente de decision del operador):
- `ORQUESTA_CODEX_CONTAINER_SANDBOX_BOUNDARY_CONFIRMED=1` permite ejecutar
  Codex con `danger-full-access`, saltando el minimo de sandbox del contrato
  (`modulos/orquesta-runtime-codex-appserver/codex_goal_app_server_v0.go`).
  El diseno es defendible (dentro de un contenedor, el contenedor es el
  limite) y es opt-in, pero se introdujo dentro de un commit titulado
  "documenta y preserva runner aislado". Regla del revisor: NO se amplia su
  uso ni se anaden mas relajaciones de sandbox sin autorizacion explicita.
- Se anadio `vendor/` (29k lineas). Compila, pero cambia el build a
  `-mod=vendor` para todo el repo. Decision no consultada; se acepta por
  ahora porque no rompe nada, pero queda registrada.

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
- **Frente que falta para TERMINAR la app: CONECTORES** (runtime-codex-*,
  state-file, required-test, superficies MCP/HTTP, wiring stack/`cmd/orquesta-server`).
  Es ahora tu prioridad 1.
- Auxiliares (stdio, ingesta, presentaciones, web/telegram): no los amplies
  hasta cerrar conectores, salvo tools que te encargue el revisor.
- Bugs abiertos restantes: solo residuales de campo (remoto/OPES/proveedor real).
  Ninguno bloquea el trabajo local.

## REPARTO DE CARRILES (actualizado 2026-07-12 ~01:00)

**Codex ya no esta en juego**: no hay sesiones Codex vivas y su cola
(`docs/instrucciones_codex_2026-07-11.md`) queda HISTORICA. El trabajo lo
llevan ahora dos actores:

- **Hermes (tu)**: BUGS del inventario + PROGRAMACION DE TOOLS + el frente de
  CONECTORES que Codex dejo abierto (runtime-codex-*, state-file,
  required-test, superficies MCP/HTTP, wiring stack/`cmd/orquesta-server`).
  Las tools dejan de estar congeladas: son tu carril.
- **Claude (revisor/director)**: no programa el dia a dia. Dirige, revisa
  cierres reejecutando tests, arbitra, lanza goals por la API nativa de
  Orquesta cuando toca y arregla bloqueos. Interviene en codigo solo si tu
  te atascas o si es trabajo de direccion (contratos, guards, arbitraje).
- **Orquesta**: ejecuta goals; es quien materializa cambios dentro del
  write-set que ella entrega. Cuando puedas, trabaja DENTRO de un goal suyo.

Prioridad de tu cola: (1) CONECTORES hasta cerrarlos -- es lo unico que falta
para terminar la app; (2) BUGS locales reproducibles; (3) TOOLS.

## DIAGNOSTICO DE CONECTORES (T9201, 2026-07-12) - CAMBIA EL PLAN

Orquesta ejecuto el diagnostico por su API nativa (goal
`goal-ref-task-autoprogramming-e0d0ac4a630f-g01`, complete + accepted; test
declarado reejecutado por el revisor). Resultado en
`docs/diagnostico_frente_conectores_T9201_2026-07-12.md`.

Hallazgo principal, aceptado por el revisor: **NO faltan adaptadores**. Todos
los puertos del nucleo (goal, state, required-test, delivery, transporte)
tienen implementacion y wiring identificables y probados con fakes. Lo que
falta para cerrar el frente es **EVIDENCIA DE INTEGRACION**, tres smokes de
composicion:

1. Backend `app_server_tmux` real: launch -> observe -> closure con receipt
   durable.
2. Registro simultaneo completo MCP/HTTP: un bootstrap que enumere resources
   y tools registrados y haga una llamada representativa por grupo.
3. Ciclo delivery -> review -> closure causal (ACK + delivery + review).

Es decir: conectores se cierra EJECUTANDO Y CONSERVANDO EVIDENCIA, no
programando adaptadores nuevos. No conviertas "falta smoke" en "falta
adaptador".

## CANAL DIRECTO DE MENSAJES (hallazgo del revisor 2026-07-12)

Pregunta del operador: "¿no seria bueno poder mandarle mensajes directos por
MCP?". Respuesta: **ya existe y esta implementado**, solo falta cablearlo.

Verificado por el revisor:
- Tool MCP: `orquesta.operator.director.message.v0` (registrada, con store
  durable; contrato en `modulos/orquesta-operator-director-channel`):
  campos `sender_ref`, `target_ref`, `intent` (`instruction`), `body`.
- Endpoint MCP vivo del servidor: `POST /mcp` (JSON-RPC `tools/call`).
- Conector Hermes ya implementado: `modulos/orquesta-operator-mcp-hermes`
  (`NewHermesOperatorMCPConnectorV0`) y su config
  (`cmd/orquesta-server/hermes_operator_config_v0.go`, bloque
  `hermes_operator` de `orquesta.config.json`: `enabled`, `base_url`,
  `mcp_path`, `api_key_file`, tools y connector refs).

Estado real: al llamar la tool en el servidor vivo devuelve
`operator_message_port_unavailable`, porque el servicio se construye con el
`operatorConnector` y este es **nil**: el bloque `hermes_operator` no esta
habilitado en la config del servidor
(`cmd/orquesta-server/stack.go:500`, `newOperatorDirectorChannelServiceV0`).

Esto es un HUECO DE CONECTORES de manual: la superficie existe pero el
bootstrap no inyecta el binding. Encaja exactamente con lo que predijo el
diagnostico T9201.

## Cola de trabajo

- [x] H0-diagnostico: hecho por Orquesta (T9201) y aceptado por el revisor.
- [x] H0a ACREDITADO por el revisor (2026-07-12 ~02:40). NO hacia falta un
  smoke nuevo: la evidencia ya existe y fue verificada leyendo los receipts
  durables del servidor vivo. Goals reales sobre backend `app_server_tmux`
  con `complete` + closure `accepted`, artefacto materializado, required
  tests con atestacion independiente `passed` y shutdown gobernado sin
  residuos:
    - `goal-ref-task-autoprogramming-812ab1c3804c-g01` (T9104, piloto 226):
      artefacto `docs/verificacion_muestra_s13_2026-07-11.md`.
    - `goal-ref-task-autoprogramming-e0d0ac4a630f-g01` (T9201, diagnostico):
      artefacto `docs/diagnostico_frente_conectores_T9201_2026-07-12.md`.
    - `goal-ref-task-autoprogramming-737dd91a2e1a-g01` y `b8b48396ebcd-g01`:
      complete/accepted con 3 required tests `passed` cada uno.
  El ciclo launch -> observe -> closure con receipt durable queda probado en
  composicion real. Lo que resta del frente son H0b y H0c.
- [ ] H0a-bis (OPCIONAL, solo si se quiere script reproducible): smoke real del backend `app_server_tmux`:
  launch -> observe -> closure con receipt durable; conservar refs compactas
  de probe, launch, observe y stop. Cierre gobernado y cero residuos.
- [x] H0b ACREDITADO (e82736fc3; pasa la prueba de mutacion del revisor): smoke de bootstrap MCP/HTTP: enumerar resources y tools
  registrados en el servidor arrancado y hacer una llamada representativa por
  grupo (MCP y HTTP). Debe fallar si falta un binding.
- [x] H0c ACREDITADO (fd2e18f7d; pasa la prueba de mutacion del revisor en codigo de produccion): smoke del ciclo delivery -> review -> closure causal (ACK,
  delivery, review), con evidencia durable enlazada en la matriz de pruebas.
- [x] H0d CERRADO (c66b9e009 + 862c1a6a2): habilitar el bloque
  `hermes_operator` en la config del servidor y cablear el conector para que
  `orquesta.operator.director.message.v0` deje de devolver
  `operator_message_port_unavailable`. Criterio de cierre: un `tools/call`
  contra `POST /mcp` con `target_ref` de Hermes entrega el mensaje y queda
  registro durable; test de composicion que falle si el binding no se inyecta.
- [ ] H1: BUGS. Coge los bugs abiertos del inventario
  (`docs/inventario_bugs_orquesta_2026-06-30.md`) de uno en uno, empezando por
  los reproducibles en local. Para cada uno: reproducir, arreglar, test que
  falle sin el fix, y cierre con evidencia real. Los residuales de campo
  (remoto/OPES/proveedor real) NO son tuyos: dejalos abiertos y anotados.
- [x] H1b: TOOLS CERRADO. Las seis tools canonicas estan vivas y gobernadas;
  la app real creo ademas `orquesta-native-smoke-tool` por el circuito completo.
- H1c (regla permanente): preferentemente trabaja DENTRO de goals que Orquesta te entregue
  (write-set gobernado). Si trabajas fuera de un goal, respeta igual el
  write-set del bug/tool y no toques el carril de Codex.
- H2 (regla permanente): si un goal se te queda sin progreso material (sin diff, test, result
  ni receipt), NO sigas quemando contexto: devuelvelo con causa concreta. El
  gobierno de progreso material (BUG-226) esta activo y cortara igualmente.
- H3 (regla permanente): al terminar un hito, avisar al revisor con la senal (una linea):

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

## Verificacion de arranque (2026-07-12, revisor) - CORREGIDA

La instalacion esta INSTALADA Y CONFIGURADA: `.env` presente, `auth.json`
presente, provider OpenAI Codex, modelo gpt-5.5. Los `x (not set)` del panel
de API keys son esperados: la autenticacion va por `auth.json`, no por
variables de entorno.

Comportamiento correcto observado: la entrada canonica arranca el contenedor
efimero con el aislamiento del runbook. Sin TTY (lanzada desde una tool o
script) Hermes imprime el status y termina; la sesion de trabajo requiere
terminal interactivo:

    ./.orquesta-runtime/hermes/hermes-orquesta.sh

Nota del revisor: una primera lectura parcial del status hizo creer que
faltaba `hermes setup`. Era un error de lectura mio, no un fallo de la
instalacion; queda corregido aqui para que nadie repita el diagnostico.
