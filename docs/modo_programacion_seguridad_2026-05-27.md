# Modo programacion y seguridad gradual

Fecha: 2026-05-27.

Decision operativa vigente: durante desarrollo Orquesta debe arrancar en
`ORQUESTA_SECURITY_MODE=programming`.

En este modo se quitan los railes que estaban retrasando la autoprogramacion:

- detalle prohibido por palabras genericas;
- bloqueo por secretos/proveedor/modelo cuando el operador ya esta en entorno
  local controlado;
- politicas cerradas de `safety`;
- prohibicion de shell;
- allowlist estrecha de entorno.

Los dos limites que siguen siendo estrictos incluso en modo programacion son:

- no borrar sin orden explicita revisada;
- no operar fuera del directorio de trabajo autorizado.

Produccion queda preparada por niveles:

- `production_low`;
- `production`;
- `production_high`.

La regla para cerrar seguridad en el futuro es progresiva: cada rail que vuelva
a activarse debe tener tests externos con matrices amplias de falsos positivos y
no debe cortar trabajos validos por vocabulario operativo normal.
