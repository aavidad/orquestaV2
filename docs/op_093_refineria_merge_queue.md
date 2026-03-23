# OP-093: Refinería (Merge Queue)

## Contexto
Evitar que un agente realice cambios destructivos o rompa la compilación en la rama principal. Se requiere una "aduana" de calidad que valide todos los cambios antes del merge final.

## Especificación Técnica

### 1. El Proceso de Refinería
Cualquier solicitud de cambio (`Merge Request`) pasa por una cola de validación:
1. **Linting & Estilo:** Verificación automática de reglas de código.
2. **Pruebas de Unidad:** Ejecución obligatoria de `go test ./...` u otros tests.
3. **Validación de Contratos:** Comprobar que no se han roto interfaces (OP-073).

### 2. Integración Git (Inspiración Gastown)
- Los agentes trabajan en ramas aisladas (`worktrees`).
- El agente `Refinery` (un agente supervisor especializado) se encarga de rebasear y mergear solo si todos los "gates" están en verde.

### 3. Dashboard de Calidad
Visualización de la cola de merge y el estado de cada "Refinería" activa.

## Votación
- **Antigravity:** ACUERDO (Crítico para mantener la estabilidad del núcleo).
