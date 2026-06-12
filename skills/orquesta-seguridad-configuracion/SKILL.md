---
name: orquesta-seguridad-configuracion
description: Revisar y programar configuracion, env vars, seguridad operacional, allowlists, firewalls, rails advisory, redaccion de secretos y efectos externos opt-in.
---

# Orquesta Seguridad Configuracion

Usa esta skill cuando se toque configuracion de servidor, env vars, permisos,
firewall, rails, tokens, HOME, PATH, proveedores, URLs o efectos externos.

## Configuracion canonica

- Una variable por decision configurable.
- Defaults documentados y publicados en estado efectivo si afectan al runtime.
- Secretos solo como presencia/redaccion, nunca valor.
- Cambios que requieren reinicio deben indicarlo.

## Seguridad

- Cortes fuertes solo por seguridad real, causalidad rota, refs imposibles,
  datos sensibles efectivos o efectos externos no autorizados.
- Rails de detalle operativo son advisory: diagnostico o redaccion, no veto.
- HOME, PATH, tokens, OAuth y credenciales deben ser opt-in o aislados.
- URLs con efectos externos requieren clasificacion de destino y confirmacion.

## Operacion

- Preferir allowlist estrecha y automatizada.
- Probar cambios de firewall/SSH antes de cerrar.
- No dejar servicios de prueba vivos.
- Registrar evidencia compacta.

## Entrega

Devuelve variables tocadas, estado efectivo, pruebas, riesgos y rollback.
