# Política Arquitectónica por Tipo de Proyecto

Orquesta no impone una única estructura inamovible para cualquier situación técnica, sino que establece directrices estrictas basadas en el tipo de proyecto:

1. **Servicios y APIs Core:** Deben seguir una arquitectura hexagonal estricta (Separación en puertos y adaptadores). Esto es un requisito vital para el crecimiento del propio `Orquestador` y `PlataformaMunicipal`, para evitar acoplar la lógica de negocio a la base de datos (ej. SQLite).
2. **Scripts y Herramientas Operativas Rápidas:** Pueden emplear una estructura plana o monolítica si su propósito es puntual y no gestionan estado persistente complejo.
3. **Controladores de Infraestructura:** Pueden acoplarse de forma directa a librerías y SDKs si su único fin es el despliegue o el puenteo de APIs.

*(Nota: La hexagonalización actuará como "gate" previo al avance de versiones en módulos centrales para prevenir la deuda técnica futura).*
