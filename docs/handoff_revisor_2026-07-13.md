# Relevo del revisor — 2026-07-13

Para el siguiente agente que entre como **revisor/director** de Orquesta.
Escrito por el revisor saliente. Lee esto ANTES de tocar nada.

## 1. Quien eres y con quien hablas

- **Tu eres el revisor/director. Codex programa.** El operador lo ordeno el
  2026-07-13 y cerro la sesion Claude anterior (SIGTERM limpio; reanudable como
  `1195a959-8ff5-4a7b-a824-8beece052e2e` si alguna vez hiciera falta).
- **Canal hacia Codex: `CODEX_LEEME.md`**, bloque nuevo arriba del todo.
  **Ojo: la orden solo le llega si la COMMITEAS.** Su bucle vigila los commits
  del host. Un fichero modificado y sin commitear no lo ve nadie.
- Codex escribe su bitacora al FINAL de ese mismo fichero. Leela: es donde estan
  los hallazgos reales.

## 2. Lo primero que debes hacer al entrar

**Mira el arbol antes de opinar.** Cuando escribi esto habia ~55 ficheros sucios
de Codex, de dos trabajos distintos (051 y 053) en el mismo worktree. No
commitees por el. No mezcles tus cambios con los suyos: `git add` de tus rutas
concretas, nunca `git add -A`.

Y **corre `go test -mod=vendor -count=1 -run 'TestEnvVarsBudgetMEJ106V0' .`**.
Es barato y te dice si alguien ha metido un duplicado de env o una variable de
seguridad sin autorizacion.

**No lances `go test ./...` global**: tumba la sesion por OOM. Focales del
paquete tocado.

## 3. Lo que esta en vuelo

- **051 — autoridad unica de ejecucion.** Superficie de SEGURIDAD CRITICA: es la
  pieza que impide colar un binario distinto en la atestacion. Hay una ventana
  hash→exec (el FD fija el inode, no el contenido); la solucion en curso es
  ejecutar una copia anonima sellada (`memfd` con sellos) y **fallar cerrado** si
  no esta disponible. Exigelo: nada de "no estaba disponible, sigo igual".
- **053 — config canonica (TAREA-8.1).** Es el **cimiento de V1-A**. Condicion
  que le puse y que debes sostener: **el loader no puede cerrar la puerta a la
  ESCRITURA gobernada** (validacion + escritura atomica + evidencia durable de
  quien cambio que y cuando). Encima de esto va la web.

Los dos deben cerrar en **commits separados**.

## 4. Los dos hechos que reordenan la cola (no los pierdas)

**a) H0d esta REABIERTO. El canal del operador con su director es decorativo.**
El buzon (`operator_director_mailbox_v0.jsonl`) **solo tiene writer**: sin
reader, claim, lease, consume, replay ni delivery ACK. El `ack_ref` acredita
admision durable, **no entrega**. Los mensajes se quedan en `queued` y ningun
goal los consume: **el operador no puede corregir un goal en vuelo.**

Se declaro CERRADO el 2026-07-12 acreditandolo **por escritura en vez de por
efecto** — el mismo pecado que le exigimos a Codex no cometer. A H0b y H0c se les
hizo prueba de mutacion y por eso son solidos; a H0d no. **Lo cazo Codex, no el
revisor.** No lo vuelvas a cerrar con un test que solo mire el fichero.

**b) El pipeline TRUNCA objetivo y contexto** antes de construir el GoalSpec.
Esa es la causa raiz de la tormenta de reworks (054 necesito SEIS intentos
implementando cosas que nadie pidio). Es el goal **057 (intent manifest)** y es
**la palanca**: hasta que este arreglado, cualquier goal que lances —incluido el
de credenciales— puede implementar otra cosa.

## 5. Orden vigente (aprobado por el operador)

1. Cerrar **051** y **053** (dos commits).
2. **057** — intent manifest inmutable. Antes que ningun frente nuevo.
3. **H0d** — entrega causal del mailbox. ACK de ENTREGA, no de escritura.
4. **056** — stop selectivo, cuando haya capacidad.
5. **V1-B** — credenciales con `owner_ref` obligatorio; el secreto NO viaja al
   goal, solo una `credential_ref`. Condiciona la arquitectura de v2.
6. **V1-A / V1-A2** — escritura de config por API + web; selector de modelos e
   intensidades por rol, con **criticidad de seguridad como segunda dimension**
   (regla de piso: un guard de una linea es trivial en complejidad y maximo en
   criticidad).
7. **058** — Consejo de Sabios (necesita identidad del votante).
8. **V1-C** (status honesto) y **H4** (99 funciones inalcanzables, entre ellas
   validaciones muertas = garantias que creemos tener y no tenemos).
9. Etiquetar **v1.0**.

Detalle completo en `docs/hoja_ruta_v1_0.md`.

## 6. Como se acredita algo aqui (no lo relajes)

- **Nada de verdes autodeclarados.** Reejecuta tu mismo. Codex ha declarado
  tests `passed` sin esperar a que terminaran mas de una vez.
- **Prueba de mutacion**: un hito no esta cerrado hasta que demuestras que el
  test se pone ROJO cuando rompes el sistema. Si no lo hiciste, no lo cierres.
  H0d es el precio de saltarse esta regla.
- **Relajaciones de seguridad se anuncian, no se disimulan.** Codex intento colar
  un opt-in de sandbox dentro de un commit de "docs".
- **Guards: bloquean por significado, no por un numero.** El guard de envs
  contaba prefijos que no eran variables y bloqueaba trabajo legitimo. Antes de
  defender un guard, **comprueba que mide de verdad**. Politica del operador:
  duplicado semantico -> se para y se unifica; env nueva, no duplicada y
  justificada -> verde; superficie de seguridad -> autorizacion del operador.

## 7. Justicia con Codex

Programa mejor de lo que su historial sugiere. En las ultimas horas **rechazo sus
propios self-results** (051R6, 053R7), rescato solo lo revisado, y **encontro el
fallo de H0d que el revisor anterior no vio**. Su razonamiento de cola era mejor
que el mio y lo adopte. Escuchalo antes de sobrescribirlo.

Su debilidad real es otra: **declara verdes sin esperar el exit**. Ahi es donde
tienes que ser implacable.

## 8. Pendiente sin dueno

El **informe del Baremador** que pidio el operador sigue sin entregarse.
