# Instrucciones del director para Codex - cola F (mejoras de la app) - 2026-07-09

Autor: Claude (director/revisor). Orden del operador: pasar a fichero las
tareas de mejora que quedan ademas del despliegue.

Contexto: la cola D1-D7 (estructura del supervisor) y la cola E1-E6 (familias
estructurales de bugs) estan cerradas o en vuelo. Lo que queda YA NO son bugs
que impidan funcionar: es (1) la prueba funcional que nunca se ha hecho,
(2) deuda tecnica medida y (3) eficiencia. Este fichero es la cola F.

Reglas vigentes: un commit por tarea, bateria P5 (dependientes transitivos del
write-set), `git diff --check`, bitacora al cerrar. NO ejecutar tareas
gateadas antes de su gate.

Orden recomendado (y gates): F1 -> F2 -> F3 -> F4 -> F5 -> F6 -> F7.
F8 es independiente de OPES y puede hacerse en paralelo cuando no pise E3/E5
del wizard.

---

## F0 (precondicion, NO es tarea de Codex): despliegue

Nada de F1/F2 se verifica sin el despliegue atomico ejecutado en el servidor
(`scripts/orquesta_server_deploy.sh`, runbook
`docs/runbooks/orquesta_server_deploy_atomico_2026-07-09.md`). La reauth del
Codex remoto YA esta hecha y verificada (2026-07-09: `auth.json` de berserk
reescrito, probe real `codex exec` -> `ok`, 1447 tokens). El despliegue lo
lanza el operador o Claude con permiso explicito.

## F1 (GATEADA POR OPERADOR): test de campo OPES real

Estado 2026-07-09, orden del operador: NO ejecutar todavia creacion de
temarios ni escritura real en OPES. Primero debe quedar Orquesta perfecta en
nucleo, despliegue, control movil, nightly/avisos y observabilidad. Cuando
ese gate este cumplido, se probara con un temario que no exista ya o que este
claramente incompleto.

Es el unico examen que Orquesta no ha pasado nunca: el smoke de 24 fases pasa
con fixtures; produccion real jamas se ejecuto. Cierra BUG-058/066/075 y la
mision registrada en bitacora (commit 885e76b0).

Gate: F0 hecho, servidor con supervisor sano, control/avisos operativos y
autorizacion operativa expresa de reabrir OPES real.

Pasos:

1. Inventariar temarios/programas del OPES real y elegir uno incompleto o
   inexistente (no destructivo: preferir crear uno nuevo).
2. Conectar Orquesta al `opes-api` real con settings canonicas y scope duro.
   El goal DEBE transportar (el bloqueo `missing_required_settings` del
   2026-07-05 fue correcto, no reintentar sin esto):
   `ORQUESTA_OPES_BASE_URL`, `ORQUESTA_BASE_URL`,
   `ORQUESTA_OPES_TEMPORAL_CONFIRM=1`, `ORQUESTA_OPES_BRIDGE_CONFIRM=1`,
   `limit=1` y scope por `job_ref` o programa/tema/correlacion.
3. Cadena real de 24 fases hasta `finalize_temario_package`, observando por
   eventos (no por polling ciego).
4. Criterio de exito: temario completo materializado en OPES con settlement
   durable, calidad por tema aceptada, sin reescritura tardia y sin procesos
   residuales (`app_server_tmux_processes_alive=0`).
5. Cualquier fallo se registra como bug de campo con refs y se programa el fix
   por Orquesta, no a mano.

Precaucion: no tocar colas ajenas ni el OPES productivo fuera del scope
declarado. Si no hay instancia temporal/preprod, PARAR y pedir decision al
operador antes de escribir en produccion.

## F2: control movil real por Telegram

El endpoint no-LLM (`POST /api/v0/operator/telegram/update`) y el sender Bot
API directo YA existen y tienen tests. Lo que falta es configuracion,
transporte y validacion real.

Gate: F0 hecho.

1. Definir y documentar el `orquesta.config.json` canonico del perfil remoto
   con la seccion `telegram_operator.*` (enabled, bot_link_ref, token, chats
   autorizados, require_confirmation). El TOKEN Y LOS CHAT IDS LOS APORTA EL
   OPERADOR: dejar plantilla con placeholders y documentar el hueco; jamas
   commitear secretos ni imprimirlos en logs.
2. Montar webhook o poller (elegir y justificar; poller es mas simple sin TLS
   publico) como superficie opt-in.
3. Verificacion: enviar un comando desde el movil real, ver la respuesta y el
   recibo durable. Cerrar
   `BUG-ORQ-20260705-TELEGRAM-NOLLM-ACCEPTED-INVISIBLE` en su parte operativa.
4. Aviso: sin `orquesta.config.json` el servidor NO monta la ruta opt-in y
   devuelve 404 (causa del fallo del 2026-07-08). El deploy debe pasar a
   `ORQUESTA_DEPLOY_REQUIRE_CONFIG=1` en cuanto la config exista.

## F3: nightly verde y con aviso (continuacion de E2)

Gate: F0 hecho (el nightly debe correr sobre el arbol desplegado, no sobre uno
congelado: llevaba 3 noches en rojo sobre el arbol del dia 6).

1. Diagnosticar y resolver `code_audit_ratchet_failed` con el arbol actual.
   Metricas del 20260709: `deadcode_candidates=1226` (baja, bien),
   `helper_duplicate_definitions=299`, `helper_family_definitions=302`,
   `large_files_over_800=18`, `orphan_modules=1`. Si la subida es legitima,
   justificarla por el mecanismo documentado; si es real, podarla (ver F4).
2. Terminar el aviso terminal por el canal operador (verde Y rojo) con status,
   phase, ref de git y ruta del resultado. Un rojo no puede volver a pasar
   inadvertido tres dias.
3. Reabrir la cuenta de la ventana §9 (7 nightlies verdes consecutivos) desde
   el primer verde y anotar la fecha de inicio en bitacora.

## F4: poda grande (deuda medida)

Autorizacion permanente del operador (2026-07-05) para podar sin consultar,
con los guardas tecnicos obligatorios: clasificacion a/b/c en JSON durable
ANTES de borrar, build global verde, suites verdes, smoke vivo readiness+status,
ratchet de auditoria bajando, tests huerfanos borrados en el mismo commit, y
revertir la ola entera si algo falla.

Orden dentro de F4 (olas pequenas, una por commit):

1. `orphan_modules=1`: identificarlo y retirarlo o cablearlo (decision
   razonada en bitacora).
2. `helper_duplicate_definitions=299` sobre 302 familias: es el hallazgo mas
   grave de calidad. Consolidar por familia (un helper canonico por familia,
   el resto borrado), empezando por las familias con mas duplicados.
3. `large_files_over_800=18`: dividir por responsabilidad, no por lineas. Ojo:
   la segmentacion existe porque llegamos a ficheros de 20000 lineas; no
   deshacerla, solo cortar los que superan el umbral.
4. `deadcode_candidates=1226`: olas por modulo, con el JSON de clasificacion.

RETIRADA DEL LEGACY (`orchestration-core/runtime` clase C, ~221 simbolos):
GATEADA a la ventana §9 (7 nightlies verdes). Autorizacion ya dada; no
ejecutar antes del gate. Al cumplirse, retirar y firmar autonomia en bitacora.

## F5: retirar los alias deprecated de variables de entorno

Hoy `env_vars_orquesta=512` con la config canonica ya mandando y los envs
como alias deprecated que emiten `deprecated_env_used`. El plan acordado era
retirarlos en dos versiones. Esta es la tarea que responde a la preocupacion
del operador de "no tener 500 variables".

1. Confirmar por telemetria/diagnostico que ningun alias sigue en uso real en
   los perfiles vivos (local, remoto, smokes, nightly).
2. Ola 1: retirar los alias sin uso, bajar el ratchet MEJ-106 al nuevo valor
   (bajarlo, nunca subirlo) y borrar sus tests.
3. Ola 2: los que queden, con aviso previo en bitacora.
4. Criterio: `scripts/orquesta_metricas_deuda.sh --json` muestra
   `env_vars_orquesta` bajando de forma sostenida; ningun smoke depende de un
   alias.

## F6: eficiencia y tokens decidida por datos (MEJ-205, ya en curso)

Continuar los golden evals con metricas e ingesta de usage. Regla vinculante
ya escrita en bitacora y que se mantiene: las medidas de ahorro
(comunicacion compacta, write-set estrecho, contexto acotado, programacion
minima) son OPT-IN y se aceptan solo si el A/B en el banco de tareas doradas
reduce tokens/diff SIN aumentar fallos. Si suben los fallos, se revierten.

Entregable: informe corto en docs con la tabla A/B (tokens, diff, fallos) y la
decision por medida. Nada de doctrina nueva sin numeros.

## F7 (AUTORIZADA por el operador 2026-07-09, gateada por prudencia): reactivar el lazo de automejora

Hoy `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_DISABLED=true` por la orden previa
de congelar la cola. Reactivarlo convierte a Orquesta de "autonoma ejecutando
trabajo encolado" a "autonoma generandose trabajo".

Decision del operador (2026-07-09, literal): "y activa la automejora" /
"pero cuando sea prudente". Es decir: la autorizacion esta DADA; el juicio del
momento queda delegado en Claude (director). No se pide nueva confirmacion
para activarla una vez cumplidos los gates; SI se avisa al operador al
activarla.

Gates que Claude debe comprobar antes de activar (todos, no algunos):

1. F0 desplegado y `runtime_identity.binary_sha256` del binario vivo == arbol.
2. Supervisor sano en remoto de forma sostenida: `supervisor_error_ticks` sin
   crecer durante al menos 24h de observacion, cola drenando y T137 (o
   cualquier run sobredimensionado) aparcado por `run_oversized`, no
   congelando.
3. F1 verde: temario OPES real completo con settlement durable.
4. F3 operativo: nightly verde y AVISO por Telegram funcionando, para que el
   operador pueda ver un rojo y parar el lazo desde el movil.
5. Ventana §9 cumplida (7 nightlies verdes consecutivos).

Como activar cuando los gates esten verdes:

- Retirar `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_DISABLED=true` del perfil de
  arranque (`scripts/orquesta_server_ctl.sh`) via config canonica, no via env
  suelta.
- Activar con limites EXPLICITOS y conservadores en la primera ventana:
  concurrencia de agentes acotada, presupuesto de tokens por ola, y
  `checkpoint_only_high_consumption` ya vigente.
- Kill switch probado ANTES de activar: comando de parada desde Telegram y
  `scripts/orquesta_server_ctl.sh stop` verificados en esa misma sesion.
- Primera ventana en observacion: revisar la primera ola generada por el lazo
  (que se auto-encolo, que programo y con que evidencia) antes de dejarlo
  correr solo. Registrar en bitacora la firma de autonomia.

Si algun gate falla, NO activar y anotar el motivo en bitacora.

## F8: dossier final pre-lanzamiento del wizard de programacion

Orden del operador 2026-07-10: el wizard debe guiar la definicion de la app y,
antes de aceptar crearla, entregar un resumen final muy completo de lo que se
va a programar, con documentacion extensa e infografias/diagramas que expliquen
la solucion elegida: arquitectura, i18n, datos, conectores, seguridad,
operacion, plan de trabajo, pruebas y riesgos.

Fuente vinculante: seccion 14 de
`docs/diseno_wizard_programacion_2026-07-04.md`.

Estado observado: el wizard ya guia, recomienda opciones, explica ayudas,
aplica defaults de ingenieria y expone `SpecPreview`/`LaunchReady`, pero no
hay contrato completo de `WizardLaunchDossierV0` ni confirmacion final por
`dossier_ref`.

Cambios requeridos:

1. Anadir `WizardLaunchDossierV0` con secciones estructuradas y diagramas
   verificables (`mermaid`, `diagram_blueprint` o brief infografico) generado
   desde la sesion/spec del wizard, sin LLM obligatorio.
2. Exponer el dossier en HTTP, web y MCP cuando `LaunchReady=true`.
3. Cambiar el cierre: no lanzar la app solo por `LaunchReady`; exigir
   confirmacion explicita `confirm_launch_dossier_ref`/`dossier_ref` vigente.
4. Incluir contenido minimo: objetivo, alcance, usuarios/roles, decisiones,
   desviaciones, arquitectura hexagonal/puertos/adaptadores, i18n/l10n, datos,
   integraciones/conectores, seguridad/RGPD, UI/UX, deploy/operacion, plan de
   Orquesta, tests requeridos, riesgos y dudas abiertas.
5. Incluir diagramas obligatorios: arquitectura hexagonal, flujo de usuario,
   flujo de datos/integraciones, mapa i18n y despliegue si aplica.
6. Guardas: sin secretos, tokens, HOME, rutas locales, credenciales ni detalles
   internos de runtime/proveedor en el dossier.
7. Tests: los definidos en la seccion 14.5 del diseno, mas paridad HTTP/MCP/web.

Criterio de cierre: una sesion tipo "quiero una app para una agenda" aceptando
recomendaciones produce dossier completo y no permite crear la app hasta
confirmar el `dossier_ref`; cambiar una respuesta regenera el dossier y anula
la confirmacion anterior.

---

## Residuales menores (hacer al pasar por la zona, no abrir ola propia)

- E4: enforcement pre-tool real de BUG-079 (depende del app-server del
  proveedor; BUG-200 ya bajado a nivel protocolo).
- E3: campos web del inventario de contratos.
- E6: tabla unica `script -> contratos exigidos` para los guards de scripts.
- BUG-165/065: smoke amplio con proveedor lento/stale.

## Nota final

Claude revisa cada cierre (rol observador: diseno y estructura). Ante duda de
diseno, anotar la pregunta en bitacora y seguir con la siguiente tarea en vez
de bloquearse. Ninguna tarea de esta cola justifica tocar produccion OPES
fuera del scope declarado en F1.
