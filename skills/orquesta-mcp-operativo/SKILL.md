---
name: orquesta-mcp-operativo
description: Crear y revisar tools/resources/prompts MCP de Orquesta con contratos compactos, puertos inyectados, refs opacas, presupuestos de salida y sin acoplar runtime ni producto al nucleo.
---

# Orquesta MCP Operativo

Usa esta skill cuando la tarea exponga una capacidad de Orquesta por MCP:
tools, resources, prompts, transporte JSON-RPC, descriptor o bridge operador.

## Reglas

- MCP es adaptador, no dominio.
- El tool llama a puertos publicos; no lee stores internos directamente salvo
  composicion autorizada.
- Inputs publicos no aceptan tokens, HOME, base_url sensible, rutas privadas ni
  credenciales.
- Outputs compactos, con refs opacas y presupuestos.
- Si una accion produce efecto externo, debe ser opt-in y auditable.

## Resource

Un resource debe ser:

- pequeño;
- versionado;
- estable;
- sin dumps de DB, logs, prompts o transcripts;
- con freshness si representa estado vivo.

## Tool

Un tool debe declarar:

- schema de entrada;
- schema de salida;
- errores publicos;
- puerto ejecutor;
- presupuesto de respuesta;
- pruebas de registro en transporte.

## Pruebas

Ejecuta pruebas focales del modulo MCP y del adaptador que registra el tool. Si
se toca transporte real, usa smoke opt-in y salida redactada.

## Entrega

Devuelve tool/resource, contrato, puerto usado, pruebas y limites.
