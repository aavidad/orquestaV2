---
name: orquesta-artefacto-modular
description: Crear artefactos enchufables para apps consumidoras de Orquesta, como juegos, paneles, manuales, tutores, paquetes web o assets, sin acoplarlos al nucleo ni al dominio equivocado.
---

# Orquesta Artefacto Modular

Usa esta skill cuando una app necesite anadir un artefacto reutilizable sin
reescribir el producto base: juegos, retos, tutor, ayuda, web, infografias,
manuales, paquetes o integraciones similares.

## Principio

El modulo debe poder pegarse, quitarse o versionarse sin tocar el nucleo de
Orquesta ni forzar cambios en otros dominios.

## Manifest minimo

Cada artefacto debe declarar:

- `artifact_ref`;
- `artifact_type`;
- `schema_version`;
- app o composicion consumidora;
- inputs por refs opacas;
- outputs por refs opacas;
- assets;
- i18n si hay texto visible;
- validaciones;
- estado publicable.

## Integracion

- Consumir datos por contratos/manifests, no por DB interna ni rutas privadas.
- Mantener assets locales y comprimidos si van a web.
- No mostrar trazabilidad tecnica a usuarios finales.
- Probar en local antes de produccion.
- Si falta contexto, entregar plantilla o tarea derivada en vez de bloquear el
  paquete completo.

## Calidad

- El artefacto debe aportar valor real, no ser decoracion.
- Debe tener captura o prueba funcional cuando sea UI/web.
- Debe ser accesible: texto alternativo, foco, contraste e i18n cuando aplique.

## Entrega

Devuelve manifest, artefactos, pruebas, capturas si aplica y notas de reuso.
