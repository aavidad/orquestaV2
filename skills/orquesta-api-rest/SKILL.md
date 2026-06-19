---
name: orquesta-api-rest
description: Diseñar, implementar y revisar APIs REST para apps consumidoras de Orquesta con contratos claros, validacion, errores publicos, idempotencia, seguridad y pruebas.
---

# Orquesta API REST

Usa esta skill cuando una tarea cree o modifique endpoints HTTP/REST, clientes
HTTP, bridges, webhooks o adaptadores de dominio.

## Contrato

Cada endpoint debe declarar:

- metodo y ruta;
- request schema;
- response schema;
- codigos de error publicos;
- autenticacion/autorizacion;
- idempotency key si produce efectos;
- limites, timeout y paginacion;
- trazabilidad por refs opacas.

## Implementacion

- En apps generadas, handlers HTTP son adaptadores inbound: validan/serializan y
  llaman casos de uso. No contienen reglas de dominio, queries directas ni
  wiring de dependencias.
- Validar input antes de efectos externos.
- Separar handler, caso de uso, puerto y adaptador.
- No filtrar tokens, HOME, rutas privadas, query sensible ni cuerpos crudos en
  errores publicos.
- No usar strings libres como contrato si hay schema posible.
- En efectos externos, exigir confirmacion/allowlist/scope segun dominio.

## Pruebas

- Caso OK.
- Input invalido.
- Autorizacion denegada.
- Idempotencia/replay.
- Timeout/error de adaptador.
- Redaccion de datos sensibles.

## Entrega

Devuelve contrato, endpoints tocados, pruebas, compatibilidad y riesgos de
despliegue.
