# Instrucciones del director para Codex - 2026-07-09: cierre de familias estructurales

Autor: Claude (director/revisor). Orden del operador: "documenta para codex
que cambios o mejoras debe hacer para arreglar todos los problemas que nos
estamos encontrando".

Fuente del analisis: corpus completo de bugs 2026-06-30 a 2026-07-09
(`docs/inventario_bugs_orquesta_2026-06-30.md`) agrupado por CAUSA RAIZ, no
por sintoma. Seis familias estructurales; cada tarea de abajo cierra una
familia entera, no un bug suelto. Continua la doctrina de
`docs/auditoria_diseno_estructural_2026-07-06.md` (P1-P8) y de la cola
D1-D7 (cerrada).

Reglas vigentes: un commit por tarea, bateria P5 (dependientes transitivos
del write-set), `git diff --check`, bitacora al cerrar. Orden = prioridad.

---

## TAREA-E1 (familia E5, la MAS URGENTE): pipeline de despliegue atomico

Evidencia de la familia (6 incidentes en 5 dias): servidor arrancado como
root (estado con dueno roto), servidor arrancado SIN `orquesta.config.json`
(rutas opt-in sin montar, 404 Telegram del 2026-07-08), binario vivo sin los
fixes del arbol (reapertura del payload T137 el 2026-07-08 con 1333 parejas),
nightly 3 noches en rojo sobre arbol congelado del dia 6, servidor 50+
commits por detras de local/GitHub, ratchet raiz divergente del de cmd/.

Causa estructural: codigo, binario, config y arbol del servidor se mueven
POR SEPARADO; no existe una operacion atomica de despliegue.

Cambios requeridos:

1. `scripts/orquesta_server_deploy.sh` (o subcomando `deploy` en
   `orquesta_server_ctl.sh`): una sola operacion que haga, en orden y
   abortando al primer fallo con reason_code:
   a. sync del repo del servidor a la ref pedida (ff-only; rechazar si no es
      ff con `deploy_not_fast_forward`),
   b. reset del worktree de pilotos a la misma ref,
   c. build del binario DESDE ese arbol con sha256 registrado,
   d. backup del binario anterior + swap,
   e. verificacion de config canonica presente (`deploy_config_missing` si
      falta `orquesta.config.json` donde el perfil lo exige),
   f. arranque via `orquesta_server_ctl.sh start` (que ya valida usuario y
      duenos de estado),
   g. verificacion post-arranque: `startup_ready`, `supervisor` sin error en
      N muestras, y `runtime_identity.binary_sha256` == sha del build,
   h. recibo durable JSON del despliegue (ref, sha, resultado, evidencias)
      en el state dir.
2. Guard test tipo scripts-guard: ningun script del repo puede copiar el
   binario del servidor o arrancarlo fuera del ctl/deploy (misma tecnica que
   `TestScriptsQueArrancanServidorTemporalUsanShutdownComunV0`).
3. Runbook corto en docs/runbooks con el comando unico y el rollback
   (restaurar backup + start).

Criterio de cierre: un despliegue real completo en el servidor con recibo
durable, y el reason_code de cada aborto probado por test (config ausente,
no-ff, sha mismatch).

## TAREA-E2 (familia E5): nightly verde y que avise solo

Evidencia: resultados 20260707/08/09 `status=failed`,
`phase_reached=code_audit_ratchet_failed`, y NADIE lo vio durante 3 dias.
La ventana §9 (7 nightlies verdes para retirar legacy) esta a cero.

Cambios requeridos:

1. Diagnosticar el ratchet que falla con el ARBOL ACTUAL (tras TAREA-E1):
   deadcode bajo de 1241 a 1226 (bien); sospechosos `orphan_modules=1` y
   `helper_duplicate_definitions=299`. Si la subida es legitima, justificarla
   por el mecanismo documentado (como env_vars); si es real, podarla.
2. Notificacion terminal del nightly por el canal operador ya existente
   (Hermes send / Bot API directo): un mensaje SIEMPRE (verde o rojo) con
   status, phase y ruta del resultado. Un rojo no puede volver a pasar
   inadvertido 3 dias.
3. El nightly debe registrar tambien la ref de git del arbol sobre el que
   corrio (hoy no se sabe sin entrar a mano).

Criterio de cierre: primer nightly verde de la nueva cuenta §9 + mensaje
recibido en Telegram.

## TAREA-E3 (familia E1): paridad de contratos por catalogo, no por parches

Evidencia (5 bugs de la misma clase en 4 dias): MCP-WIZARD-BOT-SCHEMA-STALE
(tool registrado sin campos), WIZARD-MCP-CONTRACT-PARITY (campos web ausentes
en DTO MCP), WIZARD-MCP-BOT-TOOL (bot sin tool MCP), WIZARD-WEB-CHAT-PANEL
(tool MCP sin superficie web), D1 accepted-invisible (estado en una
proyeccion y no en otra).

Causa estructural: cada contrato se re-declara a mano en N superficies
(web, MCP, http-gateway, stack) y la paridad depende de que alguien note el
hueco.

Cambios requeridos:

1. Inventario de contratos multi-superficie (wizard, wizard-bot, intake,
   autoprogramming status/prepare, operator-director, telegram) con su DTO
   canonico declarado UNA vez.
2. Test de paridad GENERICO que recorra el inventario y verifique que cada
   superficie declarada expone los mismos campos (la tecnica ya existe en
   los tests de paridad puntuales del wizard; generalizarla y que el
   inventario sea la unica lista).
3. Guard: anadir un tool/route nuevo sin entrada en el inventario debe
   fallar en test (`contract_surface_not_inventoried`).

Criterio de cierre: los 5 bugs de la clase quedan reproducidos como casos
del test generico (que pasarian hoy) y un caso sintetico de des-paridad
falla.

## TAREA-E4 (familia E4): frontera de proveedor que avisa y degrada

Evidencia: CODEX-HOME-TOKEN-INVALIDADO (abierto operativo desde el dia 5),
SMOKE-GOAL-FIRST-USAGE-LIMITED, Hermes 429, BUG-200 (agente real no ejecuta
el probe). La clasificacion `codex_app_server_provider_unauthorized` ya
existe (bien); falta que el sistema ACTUE.

Cambios requeridos:

1. Cuando el runtime clasifique `provider_unauthorized` o
   `usage_limit_reached`: notificacion inmediata al operador por el canal
   Telegram existente (con reason_code y el comando de reauth), una sola vez
   por episodio (dedupe por clave estable, no spam por tick).
2. Politica de degradacion documentada y aplicada: con proveedor caido, el
   supervisor pausa lanzamientos nuevos con reason_code
   (`provider_unavailable_paused`) en vez de intentar y fallar en bucle;
   reanuda solo cuando un probe barato (login status real o exec minimo)
   vuelve a pasar.
3. BUG-200: bajar la prueba adversarial de tool-output al nivel
   app-server/protocolo (inyectar la salida gigante directamente) para no
   depender de que el agente decida ejecutar el probe. Cierra la
   verificacion pre-tool de BUG-079 sin LLM en el lazo.

Criterio de cierre: simulacion de 401 en test produce pausa gobernada +
notificacion unica; BUG-079 con prueba de enforcement pre-tool verde.

## TAREA-E5 (familia E6): tests de contenido, no de presencia

Evidencia: WIZARD-I18N-PLACEHOLDER (cobertura verde con textos de relleno),
U12-AUTONOMIA-PISADA (dos preguntas escribiendo el mismo campo), parser de
paridad que no entendia arrays.

Cambios requeridos:

1. Regla de catalogo para el wizard: test que verifique unicidad de campo
   destino por pregunta (generaliza el fix de U6/U12 a TODAS las preguntas,
   no solo ese par).
2. Extender la prohibicion de placeholders a cualquier catalogo i18n nuevo
   (lista de patrones prohibidos centralizada, no copiada por test).
3. Donde un test valide "existe", preguntarse si debe validar "significa":
   regla de revision, aplicarla al menos a los catalogos del wizard y a las
   ayudas/glosario.

## TAREA-E6 (familia E2, mantenimiento): guards de scripts consolidados

La familia (shutdown sin contrato, POSTs stale, rutas durables bajo fichero)
esta CONTENIDA por los guards que ya anadiste; esta tarea es solo
consolidacion: mover los N guards de scripts a una tabla unica
(script -> contratos exigidos) para que anadir un contrato nuevo requiera
UNA linea y no un test nuevo. Prioridad baja; hacerla al pasar por esa zona.

## Familia E3 (historial sin presupuesto): SIN TAREA

Cerrada en codigo con 5 capas (payload, cargador, parking, dedupe en origen,
compactacion sin carril activo). Lo unico pendiente es VERIFICACION VIVA
sostenida en el servidor tras TAREA-E1, que ya esta cubierta por el criterio
de cierre de esa tarea. No tocar mas este frente salvo regresion con
evidencia.

---

## Nota final

El orden importa: E1 desbloquea E2 (nightly sobre arbol correcto) y la
verificacion viva de todo lo demas. E1+E2 son ademas la condicion para
reabrir la cuenta de la ventana §9. Claude revisa cada cierre; ante duda de
diseno, anotar la pregunta en bitacora y seguir con la siguiente tarea en
vez de bloquearse.
