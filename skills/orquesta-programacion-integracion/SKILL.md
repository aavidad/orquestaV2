---
name: orquesta-programacion-integracion
description: Implementar integraciones de software con arquitectura hexagonal: nucleo neutral, puertos, adaptadores opt-in, configuracion canonica y tests de frontera.
---

# Orquesta Programacion Integracion

Usa esta skill al anadir conectores, runtimes, APIs, colas, modelos, DBs o apps
externas.

## Hexagonal

- Nucleo: contratos neutrales, refs opacas, sin proveedor.
- Puerto: interfaz pequena y testable.
- Adaptador: tecnologia concreta opt-in.
- Composicion: wiring, env/config, defaults y reinicio si aplica.
- Tests: frontera, opt-in, error sin puerto y caso fake.
- En apps generadas, toda integracion entra por puerto + adaptador. El adaptador
  se registra solo desde bootstrap/cmd; el nucleo de aplicacion no importa SDKs,
  DBs, HTTP clients concretos ni proveedores.

## Configuracion

- Centralizar env/defaults/metadata en una superficie canonica.
- No duplicar variables con nombres distintos.
- No aceptar URLs, tokens, HOME o credenciales como payload publico si deben ser
  configuracion de servidor.

## Reglas

- OPES, Codex, Gemini, Claude, Ollama o cualquier proveedor no entran en core.
- Si hay alias recuperable, normalizar en adaptador.
- Si falta autorizacion para efecto externo, bloquear con error tecnico claro.

## Entrega

Contrato, adaptador, wiring, docs locales y pruebas focales.
