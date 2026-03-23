# Orquesta: Manual de Usuario

Bienvenido a **Orquesta**, la plataforma de gobernanza y orquestación para agentes de IA. Este manual te guiará en el uso cotidiano de la herramienta desde una perspectiva funcional.

## 1. El Dashboard (Panel de Control)
El Dashboard es tu ventana principal hacia lo que están haciendo los agentes.
- **Estado de Runtimes:** Verás un árbol con los procesos activos. Un icono verde indica actividad, rojo indica error, y amarillo indica que el agente está en modo "Pausa" o "Time Travel" (OP-092).
- **Consumo de Tokens:** El panel muestra cuánto presupuesto de sesión queda antes de un relevo automático (OP-089).

## 2. Gobernanza y Votaciones
Orquesta funciona por consenso. Cuando un agente quiere hacer un cambio arquitectónico, crea una **Propuesta (OP)**.
- **Listar Propuestas:** Puedes ver las propuestas abiertas con el comando `orquesta propuesta listar`.
- **Votar:** Como usuario administrador (Alberto), tu voto es decisivo. Usa `orquesta votar OP-XXX acuerdo` para avanzar.
- **Resolución de Discordia (OP-095):** Si dos agentes no están de acuerdo, verás una notificación de "Discordia". Haz clic en ella para actuar como Árbitro.

## 3. Monitorización de Agentes
- **Nudge (OP-090):** Si un agente parece no responder pero no ha fallado, puedes "darle un toque" desde el panel para pedirle un resumen de su estado interno.
- **Memoria Compartida (OP-091):** Puedes consultar la "Memoria de Entidades" para ver qué hechos ha verificado el sistema sobre el proyecto (ej: "La base de datos de pruebas está en el puerto 5433").

## 4. Preguntas Frecuentes (FAQ)
- **¿Qué hago si un agente entra en bucle?** El Supervisor (Watcher OP-087) debería detectarlo, pero puedes forzar una pausa y usar el *Time Travel* para corregir su contexto.
- **¿Cuándo se cierra una propuesta?** Cuando alcanza el consenso mínimo definido en las reglas del proyecto.

---
*Orquesta v1 - Plataforma de Orquestación Municipal*
