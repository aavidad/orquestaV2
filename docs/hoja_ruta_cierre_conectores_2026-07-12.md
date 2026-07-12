# ✅ FRENTE CONECTORES CERRADO — ORQUESTA TERMINADA COMO PLATAFORMA

Declarado por el revisor el 2026-07-12 ~12:20, tras verificacion adversarial
de los cuatro hitos. Estado final:

| Hito | Estado | Evidencia verificada por el revisor |
|---|---|---|
| Diagnostico T9201 | ACEPTADO | Hecho por Orquesta via API nativa; conclusion "no faltan adaptadores, falta evidencia de integracion" |
| H0a backend `app_server_tmux` | ACREDITADO | 4 goals reales complete+accepted, artefacto materializado, required tests atestiguados, shutdown limpio |
| H0b bootstrap MCP/HTTP | ACREDITADO (`e82736fc3`) | Pasa PRUEBA DE MUTACION del revisor: desactive `RunControl` y se puso rojo. tools/list real, llamadas HTTP por grupo, deteccion de `port_unavailable` |
| H0c delivery->review->closure | ACREDITADO (`fd2e18f7d`) | Pasa PRUEBA DE MUTACION del revisor: rompi el registro del closure causal EN CODIGO DE PRODUCCION (`close_run_v0.go`) y el smoke se puso rojo. Fuentes reales Codex (no fakes), ACK, review gate, runner de tests con evidencia, cadena de eventos causales |
| H0d canal operador-director | CERRADO | Buzon durable opt-in (`c66b9e009`) + test de `tools/call` durable (`862c1a6a2`) |

Guards al cierre: `TestEnvVarsBudgetMEJ106V0` verde (426, justificado), suites
de `orquesta-runtime-codex-delivery`, `orquesta-app-codex-stack` y
`cmd/orquesta-server` verdes. Sin envs nuevas ni relajaciones de seguridad en
el ultimo tramo.

**Siguiente fase (requiere visto bueno del operador): APP REAL.** Que Orquesta
cree una app de verdad — modulo nuevo o tool nueva — lanzada por su API nativa
con goal acotado y el revisor validando el cierre. Ademas queda pendiente el
informe del Baremador que pidio el operador.

---

# Hoja de ruta: cierre del frente CONECTORES

Autor: Claude (director/revisor). Fecha: 2026-07-12.
Destinatario: Codex. Esta hoja es **vinculante** y sustituye a cualquier
interpretacion previa de "cerrar conectores".

## Estado de partida historico (superado; ver estado final al inicio)

- **NUCLEO: cerrado sin condiciones** (BUG-226 cerrado con prueba empirica
  real; `befb3707b`).
- **Frente CONECTORES: abierto por la mitad.**
  - [x] Diagnostico T9201: hecho por Orquesta y aceptado
    (`docs/diagnostico_frente_conectores_T9201_2026-07-12.md`).
    Conclusion: **NO faltan adaptadores; falta EVIDENCIA DE INTEGRACION**.
  - [x] **H0a** (backend `app_server_tmux`): ACREDITADO. Cuatro goals reales
    cerraron `complete` + `accepted` con artefacto materializado, required
    tests atestiguados `passed` y shutdown gobernado sin residuos.
  - [x] **H0d** (canal operador-director): cerrado. Buzon durable opt-in
    (`c66b9e009`) + test de `tools/call` durable (`862c1a6a2`).
  - [x] **H0b** (bootstrap MCP/HTTP): ACREDITADO en `e82736fc3`.
  - [x] **H0c** (ciclo delivery -> review -> closure): ACREDITADO en
    `fd2e18f7d`.

H0b y H0c quedaron acreditados por el revisor; Orquesta termino su frente de
plataforma y completo despues la fase de app real documentada en
`CODEX_LEEME.md`.

## Reglas que aplican a los dos hitos (no negociables)

1. **Nada de verdes autodeclarados.** Ejecuta y pega la salida real. El
   revisor reejecuta todo antes de aceptar.
2. **Reejecuta tus propios guards antes de cerrar**: como minimo
   `go test -count=1 -run 'TestEnvVarsBudgetMEJ106V0' .` y los focales de lo
   tocado. Esta es tu debilidad historica: dejaste el guard rojo dos veces.
3. **No subas presupuestos ni ratchets** para ponerte en verde: consolida.
   (El limite esta en 426 y esa subida ya esta gastada y justificada.)
4. **Relajaciones de seguridad: se anuncian, no se disimulan.** Nada de
   colar un opt-in de sandbox dentro de un commit titulado "docs:" o "fix:
   runner". Si hace falta, commit propio y aviso explicito al revisor.
5. **Un hito = commits pequenos + senal al revisor al cerrar**:
   `echo "H0b cerrado en <commit>: <resumen>" > /home/alberto/Trabajo/orquesta/.orquesta-revisor-wake`
6. No `go test ./...` global. Focales del paquete tocado.

---

## H0b — ACREDITADO por el revisor (2026-07-12 ~12:20, commit `e82736fc3`)

Verificacion adversarial superada:
- Test reejecutado por el revisor: verde real.
- **Prueba de mutacion del revisor**: desactive `RunControl` (un binding
  DISTINTO al que Codex probo) y el test se puso ROJO. No es decorativo:
  caza bindings muertos de verdad.
- Hace `tools/list` real por JSON-RPC contra `POST /mcp`, llamadas HTTP
  reales por grupo, y **detecta `port_unavailable`** (la trampa exacta que
  nos mordio con el canal de mensajes): registrado != cableado, probado.
- Guard de envs (426) y suite completa de `cmd/orquesta-server`: verdes
  reejecutados.

Queda H0c. Enunciado original abajo, conservado como referencia.

## H0b — Smoke de bootstrap MCP/HTTP (ACREDITADO, enunciado original)

**Problema real que resuelve.** La superficie MCP/HTTP depende de que el
bootstrap **inyecte cada binding**. Hoy nada garantiza que, en un servidor
arrancado de verdad, todos los tools/resources esten registrados. Ya nos
mordio: `orquesta.operator.director.message.v0` estaba registrada pero
devolvia `operator_message_port_unavailable` porque su dispatcher era `nil`.
**Un binding puede existir y estar muerto.** Eso es lo que hay que cazar.

**Entregable.** Un test/smoke de composicion que, contra un servidor
arrancado con la config canonica:

1. Enumere los tools y resources realmente registrados (`tools/list` por
   JSON-RPC en `POST /mcp`) y los contraste con
   `MCPTransportBindingsV0` (`modulos/orquesta-mcp/mcp_transport_bindings_v0.go`):
   **si un binding declarado no aparece registrado, el test falla**.
2. Haga **una llamada representativa por grupo** (MCP via `POST /mcp` y HTTP
   via su ruta) y verifique que **no responde `*_port_unavailable`** ni error
   de puerto sin cablear. Grupos minimos: goal/observe, autoprogramming
   status, run control, director stats, operator director message, codebase.
3. Falle explicitamente si un tool responde con `estado: error` por puerto
   ausente, aunque el registro exista. **Registrado != cableado.**

**Punto de entrada.** `cmd/orquesta-server/mcp_real_transport_protocol_v0_test.go`
ya monta transporte real JSON-RPC: extiende ese patron, no inventes otro.
Endpoint vivo: `POST /mcp`.

**Criterio de cierre (lo comprobara el revisor).**
- El test falla si comento el cableado de cualquier binding (demuestralo:
  ejecutalo con un binding desactivado y ensena el rojo).
- Verde real con todos los bindings.
- Focales de `cmd/orquesta-server` y guard de envs verdes.

---

## H0c — Smoke del ciclo delivery -> review -> closure causal

**Problema real que resuelve.** El diagnostico T9201 dice que existen las
fuentes (`CodexDeliveryObservationSourceV0`,
`CodexProgressObservationSourceV0`, `CodexReviewGateObservationSourceV0` en
`modulos/orquesta-runtime-codex-delivery`) y estan probadas con fakes, pero
**no hay una prueba que recorra el ciclo entero en composicion**: ACK ->
delivery observada -> review gate -> closure causal aceptado.

**Entregable.** Un smoke/test de composicion que recorra el ciclo completo y
conserve evidencia durable:

1. Un trabajo que produzca **ACK** y entrega observable.
2. La **delivery** se observa por su fuente real (no fake) y queda con refs.
3. El **review gate** se evalua y su resultado entra en la decision.
4. El **closure** se acepta o bloquea **desde el veredicto causal**, no por
   inferencia de strings ni por autodeclaracion del agente.
5. Evidencia durable enlazada (receipt/refs), reejecutable por el revisor.

**Reglas de dominio que NO puedes romper.**
- La promocion sin ACK sigue siendo **recuperacion advisory**, no veto.
- `test` solo se acredita desde **atestacion independiente** (208H), nunca
  desde resultados autodeclarados por el agente.
- Nada de logica de dominio nueva en los adaptadores: el nucleo es autoridad
  unica (reconciliacion causal y gobierno de progreso material ya existen).

**Criterio de cierre.**
- El ciclo recorrido de verdad, con salida real pegada.
- El test falla si se rompe un eslabon (ej.: sin ACK, o review gate no
  consultado).
- Focales de `orquesta-runtime-codex-delivery`, `orquesta-app-codex-stack` y
  `cmd/orquesta-server` verdes, mas guard de envs.

---

## Al terminar los dos

1. Marca los checkboxes en `docs/instrucciones_hermes_2026-07-12.md`.
2. Actualiza el inventario de bugs si algun residual cae.
3. Senala al revisor. El revisor **reejecuta todo** y, si pasa, declara el
   frente CONECTORES cerrado y **Orquesta terminada** como plataforma.
4. Entonces —y solo entonces— se abre la fase de app real: el operador quiere
   que Orquesta cree una app de verdad (modulo nuevo o tool nueva), lanzada
   por su API nativa con goal acotado y revisor validando el cierre.

## Lo que NO hay que hacer

- No abrir frentes auxiliares (stdio, ingesta, presentaciones, web).
- No "cerrar" un hito escribiendo documentacion sobre el: los hitos se cierran
  con **pruebas que fallan cuando el sistema esta roto**.
- No tocar el carril del Baremador ni el material de `/tmp/orquesta-t9104`.
