---
name: orquesta-programacion-web-app
description: Crear o modificar aplicaciones web completas con Orquesta: frontend, rutas, estado, i18n, accesibilidad, assets, validacion local y entrega integrable por una app consumidora.
---

# Orquesta Programacion Web App

Usa esta skill cuando la tarea sea crear, ampliar o corregir una aplicacion web,
landing funcional, panel, dashboard, paquete HTML o interfaz de una app
consumidora.

## Frontera

- La web pertenece a la app consumidora, no al nucleo de Orquesta.
- Orquesta aporta direccion, agentes, review, tests y cierre.
- La app aporta branding, dominio, datos, permisos, persistencia y despliegue.

## Flujo

1. Inventaria rutas, componentes, estilos, i18n, assets y build actual.
2. Declara write-set estrecho por feature o pantalla.
3. Mantiene el sistema visual existente salvo peticion expresa de rediseño.
4. Implementa estados reales: carga, vacio, error, exito y permisos.
5. Valida responsive desktop/movil.
6. Ejecuta build/test/lint si existen.
7. Entrega URL local, capturas o evidencia verificable.

## Calidad

- Texto visible con i18n si el proyecto lo exige.
- HTML semantico y foco navegable.
- Contraste suficiente y `alt_text` en imagenes informativas.
- Assets locales, comprimidos y sin URLs remotas innecesarias.
- Nada de placeholders publicables.

## No hacer

- No meter datos de dominio hardcodeados si deben venir por API.
- No duplicar estilos globales si hay design system.
- No publicar sin validacion local o captura.
- No convertir una demo visual en producto si faltan permisos, datos o API.

## Entrega

Devuelve rutas tocadas, capturas/URL local, pruebas ejecutadas, riesgos y
pendientes publicables.
