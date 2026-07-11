# Handoff para Hermes/Sonyi — 2026-07-12

Retoma el trabajo en el punto exacto **H0a**. El núcleo de Orquesta está
**cerrado sin condiciones**; el único frente abierto es **CONECTORES**.

El diagnóstico **T9201 ya está hecho** y su conclusión aceptada es clara:
**NO faltan adaptadores; falta EVIDENCIA DE INTEGRACIÓN**. Por tanto, ejecuta
y conserva evidencia de composición; no abras una nueva implementación de
puertos ni reactives el loop histórico.

## Cola, en este orden

1. **H0a — prioridad inmediata:** smoke real del backend
   `app_server_tmux`, cubriendo `launch -> observe -> closure` con receipt
   durable. Conserva refs compactas de probe, launch, observe y stop, y cierra
   sin residuos.
2. **H0b:** smoke del bootstrap MCP/HTTP: enumera resources y tools registrados
   en el servidor arrancado y ejecuta una llamada representativa por grupo;
   debe fallar si falta un binding.
3. **H0c:** smoke causal del ciclo `delivery -> review -> closure`, enlazando
   ACK, delivery y review en evidencia durable.

Después de esos tres hitos, continúa con BUGS locales reproducibles y luego
TOOLS, según la cola viva.

## Fuentes que debes leer

- Cola viva y reparto operativo: [`docs/instrucciones_hermes_2026-07-12.md`](instrucciones_hermes_2026-07-12.md).
- Diagnóstico T9201 y matriz de huecos de evidencia:
  [`docs/diagnostico_frente_conectores_T9201_2026-07-12.md`](diagnostico_frente_conectores_T9201_2026-07-12.md).

El motivo de empezar por H0a es que es la primera evidencia pendiente y valida
el backend real que sostiene el flujo goal-first completo; H0b y H0c dependen
de cerrar después las otras dos superficies de integración.
