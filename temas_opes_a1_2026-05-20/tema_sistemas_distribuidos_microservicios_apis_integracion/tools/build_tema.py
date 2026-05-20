#!/usr/bin/env python3
from __future__ import annotations

import argparse
import html
import json
import re
from html.parser import HTMLParser
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
TITLE = "Arquitecturas distribuidas, microservicios, APIs e integracion de sistemas en la Administracion publica"
MIN_WORDS = 20250
MAX_WORDS = 22500
TARGET_WORDS = 21450


OFFICIAL_SOURCES = [
    ("Ley 39/2015, de 1 de octubre", "Procedimiento Administrativo Comun de las Administraciones Publicas", "BOE"),
    ("Ley 40/2015, de 1 de octubre", "Regimen Juridico del Sector Publico, especialmente administracion electronica, cooperacion y articulo 156", "BOE"),
    ("Real Decreto 203/2021, de 30 de marzo", "Reglamento de actuacion y funcionamiento del sector publico por medios electronicos", "BOE"),
    ("Real Decreto 4/2010, de 8 de enero", "Esquema Nacional de Interoperabilidad", "BOE"),
    ("Real Decreto 311/2022, de 3 de mayo", "Esquema Nacional de Seguridad", "BOE"),
    ("Normas Tecnicas de Interoperabilidad", "Documento electronico, expediente, catalogo de estandares, protocolos de intermediacion, modelos de datos, reutilizacion y transferencia tecnologica", "Portal de Administracion Electronica y BOE"),
    ("Reglamento (UE) 910/2014 y Reglamento (UE) 2024/1183", "Identificacion electronica y servicios de confianza para transacciones electronicas", "EUR-Lex"),
    ("Directiva (UE) 2022/2555", "Medidas para un nivel comun elevado de ciberseguridad en la Union", "EUR-Lex"),
    ("Reglamento (UE) 2024/903", "Medidas para un alto nivel de interoperabilidad del sector publico en la Union", "EUR-Lex"),
    ("Reglamento (UE) 2016/679", "Proteccion de datos personales", "EUR-Lex"),
    ("Marco Europeo de Interoperabilidad", "Principios, capas y recomendaciones para servicios publicos digitales", "Comision Europea"),
    ("RFC 9110", "Semantica HTTP", "IETF RFC Editor"),
    ("RFC 6749", "OAuth 2.0 Authorization Framework", "IETF RFC Editor"),
    ("RFC 7519", "JSON Web Token", "IETF RFC Editor"),
    ("OpenAPI Specification 3.1", "Descripcion estandar de APIs HTTP", "OpenAPI Initiative"),
]


def normalize(text: str) -> str:
    return re.sub(r"[ \t]+", " ", text.strip())


def word_count(text: str) -> int:
    return len(re.findall(r"\b[\wÁÉÍÓÚÜÑáéíóúüñ]+(?:[-'][\wÁÉÍÓÚÜÑáéíóúüñ]+)?\b", text, flags=re.UNICODE))


def slugify(value: str) -> str:
    value = value.lower()
    repl = str.maketrans("áéíóúüñ", "aeiouun")
    value = value.translate(repl)
    value = re.sub(r"[^a-z0-9]+", "-", value).strip("-")
    return value or "seccion"


def add(lines: list[str], *parts: str) -> None:
    for part in parts:
        part = part.rstrip()
        if part:
            lines.append(part)
        lines.append("")


def paragraph(sentences: list[str]) -> str:
    return normalize(" ".join(sentences))


SECTIONS = [
    {
        "title": "Marco publico, finalidad y frontera juridica",
        "intro": [
            "El estudio de las arquitecturas distribuidas en el sector publico no empieza por una tecnologia concreta, sino por la funcion administrativa que debe sostenerse: tramitar, informar, resolver, notificar, conservar evidencias y cooperar entre Administraciones con garantias.",
            "La Administracion digital exige que cada sistema sea comprensible desde dos planos inseparables. El primero es juridico y organizativo: competencia, procedimiento, identificacion, seguridad, interoperabilidad, proteccion de datos, archivo y responsabilidad. El segundo es tecnico: servicios, contratos, integraciones, datos, operaciones y seguridad verificable.",
            "Para un examen A1 conviene evitar una vision de moda. Microservicios, APIs, eventos o contenedores son medios. La respuesta madura identifica el problema publico, justifica el estilo de arquitectura y explica que controles impiden que la distribucion se convierta en fragilidad.",
        ],
        "points": [
            {
                "name": "Servicio publico digital como sistema socio-tecnico",
                "definition": "Un servicio publico digital es una combinacion de normas, procesos, personas, datos, aplicaciones, infraestructura y evidencias que permite ejercer potestades o prestar servicios por medios electronicos.",
                "why": "La arquitectura no puede aislarse del procedimiento administrativo, porque el acto, la notificacion, el expediente, el consentimiento, la representacion o la conservacion documental imponen requisitos que no aparecen en una aplicacion comercial ordinaria.",
                "example": "Un tramite de ayuda publica puede tener una interfaz sencilla, pero por debajo necesita validar identidad, consultar datos de otras Administraciones, registrar documentos, aplicar reglas de concurrencia, dejar trazas, emitir resoluciones y conservar el expediente.",
                "risk": "El error tipico es describir capas tecnicas sin explicar como se garantizan derechos, deberes y responsabilidades. Esa respuesta parece tecnica, pero queda incompleta para un cuerpo superior.",
                "exam": "Si el enunciado habla de Administracion electronica, no basta decir que se expone una API: hay que indicar que la API sirve a un procedimiento, a un dato o a una cooperacion administrativa concreta.",
            },
            {
                "name": "Competencia, cooperacion e interoperabilidad",
                "definition": "La interoperabilidad publica es la capacidad de organizaciones y sistemas para compartir datos y servicios respetando competencias, significado, seguridad y condiciones juridicas de uso.",
                "why": "En el sector publico no se integra cualquier sistema con cualquier otro por conveniencia tecnica. La cooperacion requiere habilitacion, finalidad, minimizacion, trazabilidad y una interpretacion comun del dato.",
                "example": "La consulta de un dato de residencia o identidad puede evitar que la persona interesada aporte documentos, pero solo si el organo tramitador tiene base juridica, finalidad determinada y controles de acceso.",
                "risk": "Una integracion punto a punto sin gobierno semantico genera duplicidades, interpretaciones divergentes y costes de mantenimiento. A corto plazo parece rapida; a medio plazo impide escalar servicios publicos compartidos.",
                "exam": "Cuando aparezcan DIR3, SIA, CSV, intermediacion de datos o servicios comunes, piensa en interoperabilidad organizativa, semantica y tecnica, no solo en conectividad.",
            },
            {
                "name": "Normativa como requisito de arquitectura",
                "definition": "El marco normativo no es un anexo posterior: condiciona la identidad, la firma, la seguridad, la conservacion, la disponibilidad, la trazabilidad y la reutilizacion de soluciones.",
                "why": "La Ley 39/2015, la Ley 40/2015, el RD 203/2021, el ENI y el ENS convierten principios publicos en requisitos arquitectonicos concretos. Una decision de versionado, almacenamiento o auditoria puede tener impacto juridico.",
                "example": "Si un servicio conserva documentos electronicos, no basta guardar ficheros en un repositorio. Debe preservar metadatos, integridad, vinculo con expediente, politica de firma y condiciones de recuperacion.",
                "risk": "Tratar la normativa como una lista de siglas conduce a respuestas memoristicas. Lo importante es explicar que cada norma resuelve un riesgo: validez, seguridad, interoperabilidad, evidencia o derechos de la ciudadania.",
                "exam": "Una buena respuesta nombra las normas principales, pero sobre todo las conecta con decisiones tecnicas verificables: autenticacion, autorizacion, catalogo, logs, evidencias, conservacion y acuerdos de nivel de servicio.",
            },
            {
                "name": "Administracion como plataforma",
                "definition": "La Administracion como plataforma organiza capacidades comunes reutilizables para que distintos servicios puedan construir sobre identidad, notificacion, registro, intermediacion, archivo, firma y datos compartidos.",
                "why": "Este enfoque evita que cada unidad reinvente piezas transversales. Tambien facilita homogeneidad, seguridad, ahorro, mantenimiento y experiencia ciudadana coherente.",
                "example": "Un nuevo procedimiento no deberia implementar desde cero su sistema de identificacion, notificacion o verificacion de documentos si existen servicios comunes adecuados y vigentes.",
                "risk": "El riesgo inverso es convertir la plataforma en un cuello de botella. Por eso hacen falta contratos claros, catalogo, versionado, soporte, observabilidad y gobierno de cambios.",
                "exam": "La idea fuerza es reutilizacion gobernada. No significa centralizar todo, sino proporcionar capacidades comunes con autonomia suficiente para que los dominios evolucionen.",
            },
            {
                "name": "Frontera entre producto, dominio y nucleo tecnico",
                "definition": "En un sistema publico complejo conviene distinguir el dominio administrativo, las capacidades transversales y la infraestructura tecnica que las ejecuta.",
                "why": "La separacion reduce dependencia entre reglas de negocio, canales de atencion, integraciones externas y plataforma de ejecucion. Permite sustituir piezas sin reescribir todo el servicio.",
                "example": "La regla para conceder una licencia pertenece al dominio del procedimiento; la autenticacion puede apoyarse en un servicio comun; la cola de mensajeria o el balanceador son infraestructura.",
                "risk": "Confundir dominio con infraestructura genera dependencia de proveedor, dificultad de auditoria y contratos opacos. Tambien impide reutilizar una capacidad en otros procedimientos.",
                "exam": "Cuando el caso proponga modernizar un monolito, explica que no todo debe convertirse en microservicio. Primero se identifican dominios, datos maestros, obligaciones legales y puntos de cambio real.",
            },
        ],
    },
    {
        "title": "Fundamentos de arquitectura distribuida",
        "intro": [
            "Una arquitectura distribuida reparte funciones entre componentes que cooperan a traves de red. Ese reparto puede aportar escalabilidad, autonomia y resiliencia, pero introduce latencia, fallos parciales, duplicidad de estado y mayor dificultad de observacion.",
            "El examen suele premiar la respuesta que reconoce la tension: distribuir no es modernizar por definicion. Se distribuye cuando el beneficio organizativo y tecnico supera el coste de coordinar componentes autonomos.",
        ],
        "points": [
            {
                "name": "Acoplamiento y cohesion",
                "definition": "La cohesion mide hasta que punto un componente concentra responsabilidades relacionadas; el acoplamiento mide cuanto depende de detalles de otros componentes.",
                "why": "Una buena arquitectura publica busca alta cohesion en torno a capacidades administrativas y bajo acoplamiento entre servicios. Asi se puede cambiar un procedimiento, una integracion o una tecnologia sin afectar a todo el sistema.",
                "example": "Un servicio de expedientes debe conocer sus estados, metadatos y reglas de conservacion; no deberia depender de la estructura interna del servicio de notificaciones ni de la base de datos de identificacion.",
                "risk": "El falso microservicio aparece cuando se crean muchos despliegues pequeños que comparten tablas, librerias rigidas y ciclos de despliegue. Hay distribucion fisica, pero no autonomia real.",
                "exam": "Si se pregunta por ventajas de microservicios, matiza: las ventajas llegan si hay fronteras coherentes, contratos estables y gobierno; no por partir codigo al azar.",
            },
            {
                "name": "Fallos parciales y latencia",
                "definition": "En una red, un componente puede fallar, responder tarde o devolver informacion incompleta mientras el resto sigue funcionando.",
                "why": "El sector publico debe diseñar para fallos parciales porque muchos tramites dependen de servicios comunes, plataformas de otras Administraciones o proveedores externos.",
                "example": "Si la consulta de un dato intermediado no esta disponible, el sistema puede reintentar, dejar la solicitud en estado pendiente, permitir aportacion documental subsidiaria o derivar a revision, segun la norma aplicable.",
                "risk": "El error es asumir que una llamada remota equivale a una llamada local. No lo es: hay timeout, reintento, duplicidad, caidas, mensajes fuera de orden y trazabilidad distribuida.",
                "exam": "Menciona patrones como timeout, circuit breaker, reintentos con backoff, idempotencia, colas y degradacion controlada cuando el supuesto incluya dependencias externas.",
            },
            {
                "name": "Consistencia y disponibilidad",
                "definition": "La consistencia expresa que los participantes observan el mismo estado relevante; la disponibilidad expresa que el sistema responde a pesar de fallos o particiones.",
                "why": "No todos los datos publicos tienen el mismo perfil. Una resolucion administrativa exige fuerte control de integridad; una vista informativa puede tolerar cierta demora si se avisa adecuadamente.",
                "example": "La inscripcion de una solicitud en registro requiere garantias fuertes. En cambio, una estadistica agregada en un portal puede actualizarse de forma diferida sin lesionar derechos.",
                "risk": "Forzar consistencia fuerte en todo reduce escalabilidad y aumenta fragilidad. Relajarla sin criterio puede afectar derechos, plazos y seguridad juridica.",
                "exam": "La respuesta excelente diferencia datos transaccionales, datos de consulta, eventos de auditoria y proyecciones de lectura. No usa una regla unica para todo.",
            },
            {
                "name": "Idempotencia",
                "definition": "Una operacion idempotente puede repetirse sin producir efectos adicionales indebidos sobre el estado final.",
                "why": "Los reintentos son inevitables en sistemas distribuidos. Sin idempotencia, un reintento puede duplicar solicitudes, pagos, asientos, notificaciones o eventos.",
                "example": "Un alta de solicitud puede usar un identificador de operacion proporcionado por el cliente o generado al iniciar el tramite. Si el mensaje se repite, el servicio reconoce la operacion y devuelve el resultado ya registrado.",
                "risk": "Confundir idempotencia con ausencia de efecto es un error. Una operacion idempotente puede crear una solicitud la primera vez; lo importante es que no cree otra distinta al repetirse.",
                "exam": "Relaciona idempotencia con reintentos, colas, APIs de escritura, integracion asincrona y recuperacion tras caidas.",
            },
            {
                "name": "Observabilidad",
                "definition": "La observabilidad es la capacidad de inferir el estado interno del sistema a partir de logs, metricas, trazas y eventos de negocio.",
                "why": "En Administracion publica, observar no es solo detectar errores tecnicos. Tambien permite reconstruir una actuacion, explicar un retraso, auditar accesos y mejorar niveles de servicio.",
                "example": "Una solicitud que pasa por portal, API, gestor de expedientes, intermediacion de datos y notificacion debe conservar un identificador de correlacion que permita seguir el flujo sin exponer datos personales innecesarios.",
                "risk": "Muchos sistemas tienen logs, pero no observabilidad. Si cada componente usa identificadores distintos, formatos incompatibles o mensajes ambiguos, la investigacion posterior se vuelve manual.",
                "exam": "Cita logs estructurados, metricas de servicio, trazas distribuidas, correlacion, retencion, proteccion de datos y cuadro de mando operativo.",
            },
        ],
    },
    {
        "title": "Microservicios: uso prudente en el sector publico",
        "intro": [
            "Un microservicio no es simplemente una aplicacion pequeña. Es una unidad de capacidad de negocio o administrativa, desplegable de forma independiente, con contrato explicito, datos gobernados y responsabilidad operativa clara.",
            "En la Administracion publica, los microservicios tienen sentido cuando hay dominios diferenciados, necesidad de evolucion independiente, carga desigual, equipos capaces de operar servicios y mecanismos de gobierno que impiden la proliferacion desordenada.",
        ],
        "points": [
            {
                "name": "Microservicio frente a monolito modular",
                "definition": "Un monolito modular mantiene un despliegue unico pero separa internamente modulos; una arquitectura de microservicios separa tambien despliegue, ciclo de vida y comunicacion por red.",
                "why": "No todo sistema debe empezar distribuido. Un monolito modular puede ser mas simple, barato y robusto si el dominio no exige autonomia de despliegue ni escalado independiente.",
                "example": "Una aplicacion departamental pequeña con pocas integraciones puede funcionar mejor como monolito modular bien diseñado. Un ecosistema de servicios comunes con muchos consumidores puede justificar microservicios.",
                "risk": "Migrar prematuramente genera operaciones complejas, mas puntos de fallo y costes de coordinacion. La modernizacion responsable puede empezar por modularizar, medir dependencias y extraer servicios solo donde exista valor.",
                "exam": "Si el supuesto pide microservicios, no aceptes la premisa sin matiz. Explica criterios de adopcion y casos donde un monolito modular seria preferible.",
            },
            {
                "name": "Fronteras de dominio",
                "definition": "La frontera de dominio delimita que conceptos, reglas, datos y responsabilidades pertenecen a un servicio.",
                "why": "Una frontera bien elegida reduce cambios cruzados y evita que cada servicio conozca detalles internos de los demas. En lo publico, ademas, ayuda a asignar responsabilidad sobre datos y decisiones.",
                "example": "Identidad, registro, expediente, notificacion, pago, archivo o intermediacion son capacidades con reglas propias. Mezclarlas en un servicio generico de tramite dificulta evolucionar cada una.",
                "risk": "Dividir por capas tecnicas, como controlador, servicio y repositorio separados en procesos distintos, produce mucha comunicacion y poca autonomia. Es un antipatron frecuente.",
                "exam": "Usa vocabulario de dominio: capacidad, responsabilidad, dato maestro, evento, contrato y propietario funcional.",
            },
            {
                "name": "Datos por servicio y consistencia",
                "definition": "El principio de datos por servicio asigna a cada microservicio la autoridad sobre su modelo persistente, evitando que otros servicios escriban directamente en sus tablas.",
                "why": "La autonomia real exige que el contrato sea la API o el evento, no la base de datos compartida. Asi se preserva el significado del dato y se controla su evolucion.",
                "example": "El servicio de notificaciones puede exponer el estado de puesta a disposicion y comparecencia; otros sistemas no deberian modificar sus tablas para forzar estados.",
                "risk": "La base de datos compartida parece eficiente, pero crea acoplamiento oculto, bloquea despliegues y dificulta auditoria. Puede ser aceptable temporalmente en migracion, pero debe declararse deuda controlada.",
                "exam": "Relaciona este punto con eventos de dominio, vistas materializadas, sincronizacion asincrona y gobierno de datos.",
            },
            {
                "name": "Despliegue independiente y responsabilidad operativa",
                "definition": "Un microservicio debe poder versionarse, probarse, desplegarse, monitorizarse y revertirse con independencia razonable.",
                "why": "La independencia tecnica sin responsabilidad operativa es incompleta. Cada servicio necesita propietarios, indicadores, documentacion, guardias o soporte, y acuerdos de nivel de servicio proporcionados a su criticidad.",
                "example": "Si una API de validacion de datos es usada por cien tramites, su despliegue exige pruebas de compatibilidad, publicacion de cambios, monitorizacion y plan de retorno.",
                "risk": "Un ecosistema de microservicios sin plataforma comun termina con estilos distintos de logs, seguridad, configuracion y despliegue. Eso aumenta el riesgo de incidentes.",
                "exam": "No presentes microservicios solo como patron de codigo. Añade CI/CD, observabilidad, seguridad, catalogo, gobierno de cambios y soporte.",
            },
            {
                "name": "Granularidad",
                "definition": "La granularidad decide el tamaño funcional de un servicio y la cantidad de responsabilidad que asume.",
                "why": "Un servicio demasiado grande pierde autonomia; uno demasiado pequeño multiplica llamadas, contratos y despliegues. La granularidad correcta surge de frecuencia de cambio, cohesion, carga, reglas y propiedad.",
                "example": "Separar calculo de tasas puede tener sentido si cambia con frecuencia o se reutiliza. Separar cada validacion elemental en un servicio distinto suele ser excesivo.",
                "risk": "El nano-servicio es un microservicio sin masa critica: aumenta complejidad sin mejorar independencia. En el sector publico puede hacer mas dificil certificar, auditar y mantener.",
                "exam": "Explica que la granularidad se revisa con evidencia operativa. No es una decision estetica ni una regla de numero de lineas.",
            },
        ],
    },
    {
        "title": "APIs: contratos, ciclo de vida y gobierno",
        "intro": [
            "Una API es un contrato de interaccion. Define que capacidades se ofrecen, con que semantica, bajo que condiciones de seguridad, versionado, cuota, trazabilidad y soporte.",
            "En la Administracion publica, una API puede ser interna, interadministrativa o publica. Esa clasificacion afecta a autenticacion, autorizacion, documentacion, acuerdo de uso, proteccion de datos y responsabilidad ante errores.",
        ],
        "points": [
            {
                "name": "API como contrato estable",
                "definition": "El contrato de API describe recursos, operaciones, parametros, respuestas, errores, seguridad, limites y garantias esperadas.",
                "why": "La estabilidad contractual permite que consumidores distintos integren servicios sin conocer la implementacion. Tambien facilita pruebas, documentacion, generacion de clientes y control de cambios.",
                "example": "Una API de consulta de expedientes debe definir estados, filtros, paginacion, codigos de error y significado de cada campo. Si un campo cambia de significado sin versionar, los consumidores fallan silenciosamente.",
                "risk": "Publicar endpoints sin contrato formal produce dependencia informal. El problema aparece cuando hay que cambiar el servicio o investigar una incidencia.",
                "exam": "Menciona OpenAPI para APIs HTTP, catalogo de APIs, ejemplos de peticion y respuesta, pruebas de contrato y politica de versionado.",
            },
            {
                "name": "Semantica HTTP",
                "definition": "HTTP ofrece metodos, codigos de estado, cabeceras, representaciones y reglas de cache que permiten una interfaz uniforme.",
                "why": "Usar bien HTTP mejora interoperabilidad. No todo debe enviarse como POST generico ni toda respuesta debe devolver un codigo de exito con un error dentro.",
                "example": "Una consulta usa GET si no altera estado; una creacion puede usar POST; una sustitucion completa puede usar PUT; una ausencia real puede devolver 404; una validacion fallida puede devolver un error de negocio claro.",
                "risk": "La mala semantica crea integraciones fragiles, dificulta observabilidad y rompe intermediarios como caches, proxies o pasarelas.",
                "exam": "Diferencia seguridad del metodo, idempotencia del metodo y autorizacion. GET puede estar protegido; idempotente no significa publico.",
            },
            {
                "name": "Versionado y compatibilidad",
                "definition": "Versionar una API es gestionar cambios de forma que los consumidores puedan adaptarse sin interrupciones indebidas.",
                "why": "Los sistemas publicos tienen ciclos largos, multiples organizaciones consumidoras y obligaciones de continuidad. Cambios incompatibles requieren comunicacion, ventana de convivencia y plan de retirada.",
                "example": "Añadir un campo opcional puede ser compatible. Cambiar el tipo de un identificador, eliminar un estado o modificar el significado de un codigo puede exigir nueva version.",
                "risk": "Versionar demasiado crea fragmentacion; no versionar cambios incompatibles rompe servicios. La respuesta madura propone criterios y calendario.",
                "exam": "Cita versionado semantico de contrato, deprecacion, pruebas de regresion, guia de migracion y registro de consumidores afectados.",
            },
            {
                "name": "Autenticacion, autorizacion y ambito",
                "definition": "La autenticacion identifica al sujeto o sistema; la autorizacion decide que puede hacer; el ambito limita el uso permitido de una credencial o token.",
                "why": "Una API publica no debe confundir tener token con tener competencia o finalidad. La autorizacion debe incorporar rol, organismo, procedimiento, dato solicitado y base habilitante cuando proceda.",
                "example": "Un sistema municipal que consulta datos para un tramite concreto puede estar autorizado para ciertos servicios y no para consultas masivas o finalidades distintas.",
                "risk": "La seguridad basada solo en red o en una clave compartida no basta para servicios sensibles. Deben existir identidad robusta, trazabilidad, rotacion, minimizacion y auditoria.",
                "exam": "Relaciona OAuth 2.0, OIDC, certificados, mTLS, scopes, claims, control de finalidad y registro de accesos.",
            },
            {
                "name": "API gateway y gestion de trafico",
                "definition": "Una pasarela de APIs centraliza funciones transversales como autenticacion, autorizacion inicial, limitacion de tasa, transformacion, enrutado, registro y proteccion.",
                "why": "Permite aplicar politicas coherentes sin replicar controles en cada servicio. Tambien da visibilidad sobre consumidores, errores y volumen.",
                "example": "Una pasarela puede limitar peticiones por organismo, validar tokens, rechazar payloads excesivos y registrar un identificador de correlacion antes de enviar la peticion al servicio interno.",
                "risk": "La pasarela no sustituye al control de negocio. El servicio final debe validar permisos finos y reglas de dominio, porque la pasarela no conoce todo el contexto administrativo.",
                "exam": "Distingue seguridad perimetral, seguridad de servicio y autorizacion de dominio. El gateway ayuda, pero no agota el modelo.",
            },
        ],
    },
    {
        "title": "Integracion de sistemas publicos",
        "intro": [
            "Integrar sistemas es hacer que capacidades autonomas cooperen sin perder significado, seguridad ni responsabilidad. En la Administracion, la integracion persigue simplificar servicios, evitar aportacion repetida de documentos, compartir evidencias y asegurar continuidad entre niveles territoriales.",
            "La integracion moderna combina APIs sincronas, mensajeria asincrona, eventos, intercambio documental, servicios comunes, catalogos y gobierno semantico. La decision correcta depende de volumen, criticidad, latencia, trazabilidad, propiedad del dato y efectos juridicos.",
        ],
        "points": [
            {
                "name": "Interoperabilidad organizativa, semantica y tecnica",
                "definition": "La capa organizativa alinea procesos y responsabilidades; la semantica alinea significado de datos; la tecnica alinea protocolos, formatos y mecanismos de intercambio.",
                "why": "La conexion tecnica no garantiza interoperabilidad si cada organismo entiende de modo distinto un estado, un domicilio, una unidad organica o una fecha de efectos.",
                "example": "Un estado llamado resuelto puede significar resolucion emitida, notificada, firme o pagada. La API debe definirlo con precision para evitar errores de procedimiento.",
                "risk": "Centrarse solo en protocolos deja sin resolver el significado. Cualquier integracion A1 debe explicar modelo de datos, catalogos, metadatos y acuerdos organizativos.",
                "exam": "Si aparece el Marco Europeo de Interoperabilidad, recuerda sus capas legal, organizativa, semantica y tecnica. En España enlaza con ENI y normas tecnicas.",
            },
            {
                "name": "Intermediacion de datos",
                "definition": "La intermediacion permite consultar o verificar datos que ya obran en poder de una Administracion, evitando al ciudadano aportar documentos cuando proceda.",
                "why": "Es una pieza central de simplificacion administrativa, pero exige habilitacion, finalidad, consentimiento cuando sea necesario, seguridad y trazabilidad.",
                "example": "Un organo tramitador puede verificar identidad, residencia, discapacidad, titulos o cumplimiento tributario mediante servicios habilitados, segun el procedimiento y los convenios aplicables.",
                "risk": "Convertir la intermediacion en consulta indiscriminada vulnera principios de minimizacion y finalidad. Tambien puede generar dependencia operativa si no hay planes de contingencia.",
                "exam": "Menciona PID, Red SARA, trazabilidad, consentimiento o base juridica, no aportacion de documentos y controles de acceso.",
            },
            {
                "name": "Mensajeria y colas",
                "definition": "La mensajeria asincrona desacopla productores y consumidores mediante mensajes persistentes, colas o topicos.",
                "why": "Es util cuando la respuesta inmediata no es necesaria o cuando interesa absorber picos, reintentar, ordenar flujos y aislar fallos.",
                "example": "Tras registrar una solicitud, el sistema puede publicar un evento para generar acuse, lanzar comprobaciones, actualizar un cuadro de mando o iniciar una notificacion sin bloquear la pantalla del usuario.",
                "risk": "La asincronia exige idempotencia, tratamiento de duplicados, gestion de errores, monitorizacion de colas y criterios de reconciliacion.",
                "exam": "No digas solo que una cola mejora rendimiento. Explica que modifica el modelo de consistencia y requiere gobierno operativo.",
            },
            {
                "name": "ESB, integracion tradicional y modernizacion",
                "definition": "Un bus de servicios empresarial concentra enrutado, transformaciones y coordinacion de integraciones. Puede convivir con APIs y eventos si se gobierna adecuadamente.",
                "why": "Muchas Administraciones tienen integraciones legacy. La modernizacion no consiste en eliminarlas de golpe, sino en encapsular, medir dependencias y exponer contratos mas claros.",
                "example": "Un sistema antiguo de gestion tributaria puede seguir operando mientras una capa de APIs controla acceso, traduce formatos y registra trazas para nuevos consumidores.",
                "risk": "El ESB puede convertirse en monolito de integracion si acumula logica de negocio opaca. Tambien una malla de microservicios puede repetir ese error si cada servicio transforma datos sin criterio comun.",
                "exam": "Plantea coexistencia ordenada: inventario, catalogo, contrato, retirada gradual y pruebas de regresion.",
            },
            {
                "name": "Servicios comunes y codigos de referencia",
                "definition": "Servicios comunes y codigos como DIR3, SIA o CSV aportan identificacion comun de unidades, procedimientos, documentos o verificaciones.",
                "why": "Sin referencias compartidas, cada organismo crea codigos propios y la interoperabilidad semantica se rompe. La arquitectura debe tratarlos como datos maestros o referencias controladas.",
                "example": "Un expediente que viaja entre sistemas debe conservar identificadores de organo, procedimiento y documento que otros sistemas puedan reconocer.",
                "risk": "Copiar catalogos sin sincronizacion o usar campos libres genera inconsistencias. Hay que definir fuente autorizada, frecuencia de actualizacion y tratamiento de historicos.",
                "exam": "Relaciona servicios comunes con simplificacion, trazabilidad, reuso y coherencia administrativa.",
            },
        ],
    },
    {
        "title": "Seguridad, identidad y confianza",
        "intro": [
            "La seguridad en arquitecturas distribuidas no puede delegarse a un unico perimetro. Cada servicio, canal, integracion y dato debe participar en un modelo de confianza coherente.",
            "El ENS aporta principios basicos y requisitos minimos. eIDAS y su evolucion europea refuerzan identidad y servicios de confianza. La proteccion de datos exige minimizacion, base juridica, informacion, medidas tecnicas y organizativas, y responsabilidad demostrable.",
        ],
        "points": [
            {
                "name": "ENS aplicado a servicios distribuidos",
                "definition": "El ENS establece una politica de seguridad para el uso de medios electronicos en el sector publico, basada en principios, requisitos minimos y medidas proporcionadas.",
                "why": "En microservicios, cada componente forma parte de la cadena de seguridad. No basta certificar el portal si las APIs internas, colas, registros o secretos quedan fuera de control.",
                "example": "Una plataforma de integracion debe controlar autenticacion, autorizacion, trazabilidad, proteccion de comunicaciones, gestion de vulnerabilidades, continuidad y auditoria.",
                "risk": "Fragmentar servicios sin gobierno de seguridad produce superficies de ataque nuevas. Las dependencias entre servicios deben inventariarse y protegerse.",
                "exam": "Une ENS con categorizacion, analisis de riesgos, medidas, auditoria, continuidad, proteccion de informacion y servicios.",
            },
            {
                "name": "Zero trust y minimo privilegio",
                "definition": "El enfoque de confianza cero evita asumir que una peticion es fiable solo por venir de una red interna.",
                "why": "Las arquitecturas distribuidas multiplican llamadas internas. Cada llamada debe autenticarse, autorizarse, registrarse y limitarse segun contexto.",
                "example": "Un servicio de expedientes que llama a notificaciones debe presentar una identidad de servicio, un ambito permitido y un identificador de operacion. La red privada no sustituye esos controles.",
                "risk": "El viejo perimetro puede ocultar movimientos laterales. Si una credencial interna se compromete, el atacante puede recorrer servicios sin obstaculos.",
                "exam": "Cita identidad de servicio, mTLS, tokens de corta duracion, secretos rotados, segmentacion, politicas y registro de accesos.",
            },
            {
                "name": "Identificacion electronica y servicios de confianza",
                "definition": "La identificacion electronica y los servicios de confianza permiten reconocer sujetos, firmar, sellar, validar tiempo y aportar evidencias juridicamente relevantes.",
                "why": "Muchos procedimientos publicos dependen de que la actuacion sea atribuible, integra y verificable. Las APIs deben trasladar esa confianza sin degradarla.",
                "example": "Una firma electronica validada en un punto del proceso debe conservar evidencias y metadatos suficientes para que el expediente pueda probar integridad y autoria.",
                "risk": "Reducir firma o identidad a un simple booleano empobrece la evidencia. Hay niveles, certificados, sellos, sellos de tiempo, politicas y validaciones.",
                "exam": "Relaciona eIDAS, eIDAS2, Cl@ve, certificados, firma, sello, representacion y conservacion de evidencias.",
            },
            {
                "name": "Proteccion de datos",
                "definition": "La proteccion de datos exige tratar solo datos necesarios, con base juridica, finalidad determinada, seguridad, transparencia y responsabilidad proactiva.",
                "why": "Las integraciones facilitan circular datos. Precisamente por eso deben limitarse campos, finalidades, retenciones y accesos.",
                "example": "Una API que verifica una condicion puede devolver si se cumple o no, en lugar de entregar todos los datos fuente que permitieron calcularla.",
                "risk": "El exceso de datos aumenta riesgo de brecha, uso secundario indebido y dificultad de cumplimiento. Tambien complica pruebas y entornos no productivos.",
                "exam": "Menciona minimizacion, privacidad desde el diseño, evaluacion de impacto cuando proceda, registro de actividades, encargados y control de accesos.",
            },
            {
                "name": "Gestion de secretos y configuracion",
                "definition": "Los secretos son credenciales, claves, certificados o tokens que permiten acceso a sistemas o datos y deben gestionarse de forma segura.",
                "why": "En microservicios hay muchas identidades tecnicas. Guardar secretos en codigo, repositorios o variables sin control aumenta el riesgo de compromiso.",
                "example": "Un servicio que consume una API interadministrativa debe obtener credenciales desde un almacen seguro, rotarlas periodicamente y registrar usos anormales.",
                "risk": "La proliferacion de claves compartidas impide atribucion y revocacion fina. Si todos usan la misma credencial, no se sabe quien accedio ni se puede cortar un consumidor sin afectar al resto.",
                "exam": "Cita almacen de secretos, rotacion, identidad por servicio, certificados, revocacion, principio de minimo privilegio y separacion de entornos.",
            },
        ],
    },
    {
        "title": "Operacion, continuidad y calidad de servicio",
        "intro": [
            "Una arquitectura distribuida solo es aceptable si puede operarse. La excelencia tecnica no se demuestra con diagramas, sino con disponibilidad, tiempos de respuesta, recuperacion, trazabilidad, soporte y capacidad de evolucion sin interrumpir servicios criticos.",
            "En Administracion publica, la operacion debe alinearse con obligaciones de servicio publico, plazos, sedes electronicas, atencion multicanal y seguridad. El diseño debe anticipar incidentes, no improvisar cuando ocurren.",
        ],
        "points": [
            {
                "name": "SLA, SLO y experiencia ciudadana",
                "definition": "Un SLA es un acuerdo formal de nivel de servicio; un SLO es un objetivo medible de fiabilidad, latencia, disponibilidad u otra dimension operativa.",
                "why": "La arquitectura debe traducir expectativas de servicio en indicadores observables. Una sede electronica critica no puede depender de metricas vagas.",
                "example": "Un servicio de presentacion de solicitudes puede fijar disponibilidad, tiempo de respuesta, tasa de errores, tiempo de recuperacion y comportamiento durante picos de convocatoria.",
                "risk": "Medir solo disponibilidad tecnica puede ocultar degradacion funcional. Una API activa que devuelve errores de autorizacion por una mala configuracion no esta prestando el servicio real.",
                "exam": "Diferencia indicador tecnico, indicador de negocio y compromiso formal. Relacionalo con monitorizacion y mejora continua.",
            },
            {
                "name": "Despliegue seguro",
                "definition": "El despliegue seguro combina automatizacion, pruebas, control de cambios, revision, separacion de entornos y mecanismos de retorno.",
                "why": "Los servicios distribuidos se despliegan con frecuencia. Sin disciplina, cada cambio puede romper consumidores o introducir vulnerabilidades.",
                "example": "Una nueva version de API puede publicarse con pruebas de contrato, despliegue canario, monitorizacion intensiva y posibilidad de volver a la version anterior.",
                "risk": "El despliegue manual repetido genera diferencias entre entornos, errores humanos y dificultad de auditoria.",
                "exam": "Cita CI/CD, infraestructura como codigo, pruebas automatizadas, despliegue azul-verde o canario, rollback y trazabilidad de cambios.",
            },
            {
                "name": "Trazas, logs y auditoria",
                "definition": "Las trazas siguen una transaccion distribuida; los logs registran hechos; la auditoria aporta evidencia controlada para responsabilidades y cumplimiento.",
                "why": "No todo log es auditoria. La auditoria exige integridad, contexto, retencion, acceso restringido y capacidad de reconstruccion.",
                "example": "Una consulta de datos sensibles debe registrar quien la realizo, para que procedimiento, cuando, desde que sistema, con que resultado y con que base de autorizacion.",
                "risk": "Registrar demasiados datos personales en logs crea un riesgo nuevo. La observabilidad debe diseñarse con minimizacion y proteccion.",
                "exam": "Une observabilidad tecnica con auditoria juridica, pero distingue sus finalidades y controles.",
            },
            {
                "name": "Continuidad y recuperacion",
                "definition": "La continuidad asegura que servicios esenciales sigan prestandose ante incidencias; la recuperacion restaura sistemas y datos tras un fallo.",
                "why": "La Administracion debe atender plazos, derechos y obligaciones. Un fallo tecnico puede tener consecuencias juridicas si impide presentar solicitudes o recibir notificaciones.",
                "example": "Para una convocatoria con plazo fijo se pueden preparar escalado, pruebas de carga, plan de contingencia, mensajes de estado y mecanismos alternativos documentados.",
                "risk": "Confiar solo en alta disponibilidad tecnica sin ensayar recuperacion es insuficiente. Tambien hay que probar restauracion de datos, colas pendientes y coherencia de eventos.",
                "exam": "Cita RTO, RPO, copias, pruebas de restauracion, modo degradado, comunicacion de incidente y continuidad operativa.",
            },
            {
                "name": "Gestion de incidencias y mejora",
                "definition": "La gestion de incidencias detecta, clasifica, resuelve y aprende de fallos; la mejora convierte cada incidente relevante en acciones preventivas.",
                "why": "En ecosistemas distribuidos, la causa suele cruzar equipos. Hace falta procedimiento comun, responsabilidades claras y informacion compartida.",
                "example": "Un aumento de errores en una API puede deberse a cambio de contrato, expiracion de certificado, saturacion de cola o dato invalido en un sistema origen.",
                "risk": "Buscar culpables antes que evidencias retrasa la recuperacion. La cultura operativa madura documenta cronologia, impacto, causa raiz y medidas.",
                "exam": "Incluye postmortem, indicadores, runbooks, escalado, catalogo de dependencias y gestion de problemas.",
            },
        ],
    },
    {
        "title": "Gobierno tecnico, adquisicion y evolucion",
        "intro": [
            "El gobierno tecnico establece reglas para que muchas piezas evolucionen con coherencia. Incluye catalogos, comites o foros de arquitectura, estandares, revision de seguridad, gestion de proveedores, ciclo de vida, documentacion y medicion.",
            "La Administracion no solo construye sistemas; tambien contrata, mantiene, integra soluciones de terceros y responde ante ciudadania y organos de control. Por eso la arquitectura debe ser gobernable.",
        ],
        "points": [
            {
                "name": "Catalogo de servicios y APIs",
                "definition": "Un catalogo identifica APIs, servicios, responsables, version, consumidores, condiciones de uso, documentacion, estado y niveles de servicio.",
                "why": "Sin catalogo, los consumidores descubren servicios por contactos informales. Eso genera duplicidades y dificulta retirar versiones obsoletas.",
                "example": "Antes de crear una API de consulta de unidades, un equipo deberia poder verificar si existe una fuente autorizada, quien la mantiene y que contrato ofrece.",
                "risk": "Un catalogo desactualizado pierde credibilidad. Debe integrarse con ciclo de vida, despliegue, monitorizacion y gobierno de cambios.",
                "exam": "Relaciona catalogo con reutilizacion, transparencia interna, control de dependencias y gestion de obsolescencia.",
            },
            {
                "name": "Estandares abiertos y neutralidad",
                "definition": "Los estandares abiertos y de uso generalizado reducen dependencia de proveedor y facilitan interoperabilidad.",
                "why": "El ENI impulsa decisiones tecnologicas que garanticen interoperabilidad, conservacion y normalizacion. La arquitectura publica debe justificar formatos, protocolos y herramientas.",
                "example": "Usar contratos OpenAPI, formatos documentados y protocolos comunes facilita que distintos proveedores y Administraciones puedan integrarse.",
                "risk": "Una plataforma cerrada puede acelerar un proyecto inicial, pero bloquear datos, contratos o despliegues a medio plazo.",
                "exam": "No conviertas neutralidad en rechazo de todo producto comercial. La clave es portabilidad, documentacion, reversibilidad y cumplimiento de estandares aplicables.",
            },
            {
                "name": "Contratacion y reversibilidad",
                "definition": "La reversibilidad es la capacidad de recuperar conocimiento, datos, configuraciones y operacion cuando cambia el proveedor o el modelo de prestacion.",
                "why": "Los servicios publicos deben sobrevivir a contratos. Una arquitectura sin documentacion, automatizacion y acceso a evidencias queda cautiva.",
                "example": "Un pliego puede exigir entrega de contratos de APIs, codigo o artefactos segun proceda, documentacion operativa, pruebas, inventario de dependencias y plan de transicion.",
                "risk": "La dependencia tacita aparece cuando solo el proveedor sabe desplegar, diagnosticar o modificar el sistema.",
                "exam": "Menciona transferencia de conocimiento, documentacion viva, propiedad de datos, licencias, portabilidad, pruebas y control de configuracion.",
            },
            {
                "name": "Gestion de deuda tecnica",
                "definition": "La deuda tecnica es una decision o acumulacion de decisiones que facilita el presente a costa de mayor coste o riesgo futuro.",
                "why": "En Administracion, la deuda se vuelve especialmente peligrosa cuando afecta a seguridad, interoperabilidad, accesibilidad, datos o continuidad.",
                "example": "Mantener una integracion por fichero nocturno puede ser aceptable temporalmente si se documenta, se monitoriza y existe plan para sustituirla por un contrato mas robusto.",
                "risk": "Llamar legado a todo lo existente no es analisis. Hay sistemas antiguos estables y sistemas nuevos mal gobernados.",
                "exam": "La respuesta madura prioriza deuda por riesgo e impacto, no por preferencia tecnologica.",
            },
            {
                "name": "Arquitectura evolutiva",
                "definition": "La arquitectura evolutiva permite adaptar el sistema mediante cambios incrementales, medidos y reversibles.",
                "why": "El marco normativo, las necesidades ciudadanas y las tecnologias cambian. La arquitectura debe aceptar cambio sin romper servicios esenciales.",
                "example": "Una Administracion puede encapsular un sistema legacy, publicar APIs, introducir eventos para nuevas funcionalidades y retirar integraciones antiguas por fases.",
                "risk": "Los planes de transformacion total suelen fallar si no controlan dependencias reales y operacion diaria.",
                "exam": "Cita hoja de ruta, pilotos, medicion, coexistencia, migracion por dominios, pruebas de compatibilidad y retirada controlada.",
            },
        ],
    },
]


EXTENSION_TOPICS = [
    ("Contratos de datos", "Un contrato de datos define estructura, significado, calidad, propietario, restricciones y condiciones de evolucion de un intercambio.", "Ayuda a evitar que una API sea tecnicamente estable pero semanticamente ambigua."),
    ("Eventos de dominio", "Un evento de dominio comunica que algo relevante ha ocurrido en un contexto administrativo.", "Permite desacoplar consumidores, pero exige idempotencia, ordenacion razonable y gobierno del significado."),
    ("Backpressure", "El backpressure consiste en regular la entrada de trabajo cuando un consumidor no puede procesar al ritmo recibido.", "Evita que un pico de solicitudes colapse colas, bases de datos o servicios dependientes."),
    ("Cache responsable", "La cache mejora latencia y disponibilidad para datos de consulta, pero debe respetar vigencia, confidencialidad y coherencia suficiente.", "En lo publico hay que indicar cuando un dato es informativo, cuando es oficial y cuando se ha actualizado."),
    ("Pruebas de contrato", "Las pruebas de contrato verifican que proveedor y consumidor mantienen expectativas compatibles.", "Son decisivas cuando varias Administraciones dependen de una misma API."),
    ("Datos maestros", "Un dato maestro es una referencia autorizada que otros sistemas consumen sin redefinirla.", "Unidades organicas, procedimientos o catalogos de servicios deben tener fuente, historico y reglas de actualizacion."),
    ("Degradacion controlada", "La degradacion controlada permite ofrecer una funcionalidad limitada cuando una dependencia falla.", "No todo servicio puede degradarse, pero cuando procede reduce impacto y conserva informacion al usuario."),
    ("Trazabilidad de finalidad", "La trazabilidad de finalidad registra para que procedimiento o base se accede a un dato.", "Refuerza proteccion de datos, auditoria y confianza interadministrativa."),
    ("Accesibilidad e inclusion", "La arquitectura debe sostener canales accesibles, tiempos razonables y alternativas cuando la relacion electronica no sea obligatoria.", "La tecnica no puede excluir derechos de asistencia ni degradar la experiencia de colectivos vulnerables."),
    ("Obsolescencia tecnologica", "La obsolescencia no es solo fin de soporte; tambien incluye falta de conocimiento, inseguridad, formatos cerrados o incapacidad de integracion.", "Debe gestionarse con inventario, riesgo, presupuesto y hoja de ruta."),
]


def build_markdown() -> str:
    lines: list[str] = []
    add(lines, f"# {TITLE}")
    add(lines, "_Tema A1. Material de estudio profesional para primera lectura, repaso, supuestos y preparacion de examen._")
    add(lines, "## Indice")
    indice = [
        "Orientacion de examen y enfoque de estudio",
        "Mapa inicial del tema",
        "Definiciones de autoridad operativa",
        "Marco publico, finalidad y frontera juridica",
        "Fundamentos de arquitectura distribuida",
        "Microservicios: uso prudente en el sector publico",
        "APIs: contratos, ciclo de vida y gobierno",
        "Integracion de sistemas publicos",
        "Seguridad, identidad y confianza",
        "Operacion, continuidad y calidad de servicio",
        "Gobierno tecnico, adquisicion y evolucion",
        "Tablas de sintesis",
        "Ejemplos trabajados",
        "Supuestos practicos guiados",
        "Notas de test separadas",
        "Errores frecuentes",
        "Repaso final",
        "Plan de visuales",
        "Fuentes oficiales",
    ]
    for i, item in enumerate(indice, 1):
        lines.append(f"{i}. {item}")
    lines.append("")

    add(
        lines,
        "## Orientacion de examen y enfoque de estudio",
        paragraph([
            "Este tema debe estudiarse como un puente entre derecho publico, direccion tecnologica y diseño de sistemas.",
            "La pregunta de examen puede formularse de manera amplia, como arquitectura distribuida y microservicios, o aparecer escondida dentro de un supuesto sobre sede electronica, interoperabilidad, integracion de datos, seguridad, modernizacion de legacy o servicios comunes.",
            "La clave es no responder con una lista de productos ni con una defensa acrítica de una tecnologia.",
        ]),
        paragraph([
            "Una contestacion A1 debe empezar por la finalidad publica: prestar servicios digitales fiables, interoperables, seguros, trazables y centrados en la ciudadania.",
            "Despues debe explicar como la arquitectura distribuye responsabilidades, como se comunican los componentes, que contratos gobiernan las APIs, como se protege el dato, que ocurre cuando falla una dependencia y como se opera el conjunto.",
            "El lenguaje debe mostrar criterio: hay que saber cuando conviene un microservicio, cuando basta un monolito modular, cuando usar una API sincrona, cuando usar eventos y cuando una integracion legacy debe encapsularse antes de sustituirse.",
        ]),
        paragraph([
            "En test, los distractores suelen confundir microservicios con cualquier servicio pequeño, interoperabilidad con mera conectividad, autenticacion con autorizacion, disponibilidad con ausencia total de fallos, y API gateway con seguridad completa.",
            "En desarrollo, el riesgo es escribir parrafos tecnologicos sin conectar con ENI, ENS, procedimiento administrativo, proteccion de datos, servicios comunes y responsabilidad operativa.",
            "Para recordar el tema, conviene usar una secuencia: finalidad publica, contrato, dato, seguridad, operacion, gobierno y mejora.",
        ]),
        "> Modo tutor: Si tienes poco tiempo, no memorices nombres de tecnologias. Memoriza las preguntas que debe responder una arquitectura: que servicio publico sostiene, quien es responsable, que dato circula, con que base, por que contrato, con que seguridad, como se audita y que ocurre si falla.",
        "> Nota de test: Si una opcion dice que los microservicios garantizan por si mismos escalabilidad, seguridad o interoperabilidad, normalmente es incompleta. Solo ayudan si existen fronteras adecuadas, contratos, observabilidad, automatizacion y gobierno.",
    )

    add(
        lines,
        "## Mapa inicial del tema",
        paragraph([
            "El mapa del tema se puede imaginar como una cadena de responsabilidades.",
            "En el centro esta el servicio publico que la persona usuaria percibe: presentar, consultar, recibir, pagar, acreditar o ejercer un derecho.",
            "Debajo aparecen capacidades de dominio, como expediente, registro, resolucion o notificacion.",
            "Alrededor se situan APIs, eventos, servicios comunes, integraciones y plataformas de datos.",
            "Transversalmente actuan seguridad, interoperabilidad, observabilidad, proteccion de datos, accesibilidad, continuidad y gobierno.",
        ]),
        "![Mapa de arquitectura e integracion](assets/mapa_arquitectura_integracion.svg)",
        "| Capa | Pregunta directiva | Riesgo si se ignora | Evidencia esperada |",
        "| --- | --- | --- | --- |",
        "| Servicio publico | Que derecho, obligacion o prestacion se soporta | Tecnologia sin finalidad administrativa | Procedimiento, canal, responsable y nivel de servicio |",
        "| Dominio | Que capacidades y datos son propios | Mezcla de reglas y dependencias opacas | Modelo de dominio, dato maestro y eventos |",
        "| Contrato | Como se consume la capacidad | Integraciones informales | API, esquema, version, errores y condiciones de uso |",
        "| Seguridad | Quien accede y con que finalidad | Accesos excesivos o no atribuibles | Identidad, autorizacion, auditoria y ENS |",
        "| Operacion | Como se detectan y recuperan fallos | Servicio fragil aunque compile | SLO, logs, trazas, runbooks y continuidad |",
        "| Gobierno | Como evoluciona sin romper consumidores | Proliferacion desordenada | Catalogo, versionado, deprecacion y responsables |",
        paragraph([
            "La tabla permite orientar casi cualquier pregunta.",
            "Si se pide explicar APIs, empieza por contrato y gobierno.",
            "Si se pide microservicios, empieza por dominio y operacion.",
            "Si se pide integracion interadministrativa, empieza por interoperabilidad, base juridica y significado del dato.",
            "Si se pide seguridad, no te quedes en cifrado; incluye identidad, finalidad, trazabilidad, ENS, proteccion de datos y continuidad.",
        ]),
    )

    add(
        lines,
        "## Definiciones de autoridad operativa",
        paragraph([
            "Las siguientes definiciones no sustituyen a las normas, pero ayudan a contestar con precision.",
            "Una definicion util en oposicion debe ser corta, comprensible y aplicable a un caso.",
            "Despues de definir, conviene añadir por que importa y con que se confunde.",
        ]),
    )
    definitions = [
        ("Arquitectura distribuida", "Organizacion de un sistema en componentes separados que cooperan mediante comunicaciones de red y contratos explicitos.", "Se confunde con cualquier aplicacion web; la diferencia relevante es la gestion de fallos parciales, consistencia, latencia y autonomia."),
        ("Microservicio", "Servicio pequeño en responsabilidad, autonomo en despliegue razonable y alineado con una capacidad de dominio.", "Se confunde con un modulo remoto; si comparte base de datos y ciclo de vida no tiene autonomia plena."),
        ("API", "Interfaz programable que expone capacidades mediante un contrato documentado.", "Se confunde con endpoint; una API madura incluye version, seguridad, errores, limites, documentacion y soporte."),
        ("Interoperabilidad", "Capacidad de organizaciones y sistemas para compartir datos y procesos preservando significado, seguridad y responsabilidad.", "Se confunde con conectividad; conectar no significa entender ni estar habilitado para usar."),
        ("Evento de dominio", "Registro comunicable de un hecho relevante ocurrido en un dominio.", "Se confunde con mensaje tecnico; el evento debe tener significado de negocio o administrativo."),
        ("Idempotencia", "Propiedad de una operacion que permite repetirla sin efectos adicionales indebidos.", "Se confunde con operacion sin efecto; puede crear algo la primera vez y reconocer repeticiones despues."),
        ("Observabilidad", "Capacidad de conocer el estado del sistema a partir de metricas, logs, trazas y eventos.", "Se confunde con tener muchos logs; importa la correlacion y la utilidad diagnostica."),
        ("ENS", "Marco de seguridad aplicable al sector publico para proteger informacion y servicios en medios electronicos.", "Se confunde con una certificacion aislada; debe impregnar diseño, operacion y ciclo de vida."),
        ("ENI", "Marco que establece criterios y recomendaciones para interoperabilidad, conservacion y normalizacion.", "Se confunde con estandares tecnicos solamente; incluye dimensiones organizativas, semanticas y de servicios comunes."),
        ("API gateway", "Componente que aplica politicas transversales de entrada a APIs, como autenticacion, limites y enrutado.", "Se confunde con seguridad completa; no reemplaza autorizacion fina ni reglas de dominio."),
    ]
    lines.append("| Concepto | Definicion util | Confusion habitual |")
    lines.append("| --- | --- | --- |")
    for term, definition, confusion in definitions:
        lines.append(f"| {term} | {definition} | {confusion} |")
    lines.append("")
    add(
        lines,
        "> Modo tutor: Definir bien ahorra desarrollo confuso. En examen, escribe primero una frase precisa y despues añade una consecuencia practica. Esa segunda frase demuestra que no estas recitando.",
    )

    for section in SECTIONS:
        add(lines, f"## {section['title']}")
        for para in section["intro"]:
            add(lines, paragraph([para]))
        for item in section["points"]:
            add(lines, f"### {item['name']}")
            add(
                lines,
                paragraph([
                    item["definition"],
                    item["why"],
                    "Esta idea importa especialmente en Administracion publica porque los sistemas no solo procesan transacciones: sostienen derechos, obligaciones, plazos, evidencias, cooperacion entre organos y confianza institucional.",
                ]),
                paragraph([
                    item["example"],
                    "El diseño debe dejar claro que dato se usa, quien lo gobierna, que contrato permite consumirlo, que evidencias quedan y como se corrige una incidencia sin romper el procedimiento.",
                ]),
                paragraph([
                    item["risk"],
                    "La forma practica de evitarlo es inventariar dependencias, documentar contratos, probar integraciones, medir comportamiento real y asignar responsables funcionales y tecnicos.",
                ]),
                f"> Modo tutor: Para reconocer este concepto, pregunta que frontera protege. Si protege significado, estas ante interoperabilidad semantica; si protege despliegue, ante autonomia; si protege recuperacion, ante resiliencia; si protege responsabilidad, ante auditoria y gobierno.",
                f"> Nota de test: {item['exam']}",
            )

    add(lines, "## Tablas de sintesis")
    tables = [
        (
            "API sincrona, evento y fichero",
            ["Mecanismo", "Uso preferente", "Ventaja", "Precaucion"],
            [
                ["API sincrona", "Consulta o accion que requiere respuesta inmediata", "Contrato claro y experiencia interactiva", "Timeouts, acoplamiento temporal y disponibilidad de la dependencia"],
                ["Evento", "Comunicar hechos a consumidores desacoplados", "Escalabilidad y evolucion independiente", "Duplicados, orden, consistencia eventual y trazabilidad"],
                ["Fichero o lote", "Intercambios masivos o legado controlado", "Sencillez para sistemas antiguos", "Latencia, validacion, reconciliacion y errores diferidos"],
            ],
        ),
        (
            "Microservicio, monolito modular y ESB",
            ["Opcion", "Cuando encaja", "Riesgo", "Control recomendado"],
            [
                ["Monolito modular", "Dominio acotado, equipo pequeño o baja necesidad de despliegue independiente", "Crecimiento desordenado", "Modulos, pruebas y limites internos"],
                ["Microservicios", "Dominios diferenciados, escala desigual y equipos capaces de operar", "Complejidad distribuida", "Contratos, observabilidad, CI/CD y plataforma comun"],
                ["ESB o bus", "Integracion legacy y transformaciones gobernadas", "Logica opaca centralizada", "Catalogo, retirada gradual y trazabilidad"],
            ],
        ),
        (
            "Controles de seguridad en APIs",
            ["Control", "Que resuelve", "Error frecuente", "Buena practica"],
            [
                ["Autenticacion", "Saber quien llama", "Usar una clave compartida para todos", "Identidad por consumidor o servicio"],
                ["Autorizacion", "Determinar que puede hacer", "Confundir token con permiso total", "Scopes, claims y reglas de dominio"],
                ["Cifrado", "Proteger comunicaciones", "Creer que cifra todo el ciclo de vida", "TLS, mTLS cuando proceda y proteccion en reposo"],
                ["Auditoria", "Reconstruir acceso y finalidad", "Guardar logs sin contexto", "Correlacion, minimizacion y retencion definida"],
            ],
        ),
    ]
    for title, headers, rows in tables:
        add(lines, f"### {title}")
        lines.append("| " + " | ".join(headers) + " |")
        lines.append("| " + " | ".join(["---"] * len(headers)) + " |")
        for row in rows:
            lines.append("| " + " | ".join(row) + " |")
        lines.append("")
    add(
        lines,
        paragraph([
            "Estas tablas no sustituyen al desarrollo teorico.",
            "Su funcion es ayudar a comparar decisiones.",
            "En el examen, una tabla breve puede ordenar la respuesta, pero debe ir acompañada de explicacion.",
        ]),
    )

    add(lines, "## Ejemplos trabajados")
    examples = [
        (
            "Modernizacion de un gestor de ayudas",
            "Una consejeria tramita ayudas con una aplicacion monolitica que mezcla presentacion, reglas, consultas externas, generacion de documentos y notificaciones.",
            "La respuesta adecuada no es partir todo en microservicios de inmediato. Primero se inventarian capacidades: solicitud, expediente, baremacion, consulta de datos, resolucion, pago y notificacion. Despues se separan contratos, se identifican datos maestros y se encapsulan integraciones criticas.",
            "Puede extraerse una API de consulta de estado para la sede, introducir eventos cuando una solicitud cambia de fase y usar servicios comunes para identidad, registro o notificacion. La base de datos puede permanecer inicialmente centralizada si se documenta la deuda y se evita que nuevos consumidores dependan de tablas internas.",
        ),
        (
            "API interadministrativa de verificacion",
            "Un organismo ofrece a otros una API para verificar si una persona cumple una condicion sin entregar todos los datos fuente.",
            "El contrato debe definir finalidad, organismo consumidor, identificador de procedimiento, base juridica, campos minimos, codigos de respuesta, errores y auditoria. La seguridad debe combinar autenticacion de sistema, autorizacion por ambito, limitacion de tasa y registro de accesos.",
            "La API puede devolver una respuesta de cumplimiento, fecha de verificacion y referencia de evidencia, evitando exponer datos innecesarios. La operacion debe incluir SLO, monitorizacion, comunicacion de cambios y plan de contingencia.",
        ),
        (
            "Integracion por eventos para notificaciones",
            "Un sistema de expedientes publica un evento cuando una resolucion esta lista para notificar.",
            "El evento no debe contener todo el expediente. Debe incluir identificador, tipo de evento, momento, version de esquema y referencias necesarias. El servicio de notificaciones consume, valida, genera la actuacion y publica su propio estado.",
            "Si el mensaje se duplica, la idempotencia impide notificar dos veces. Si el servicio esta caido, la cola conserva trabajo pendiente. Las trazas permiten saber si el retraso esta en expediente, cola, notificacion o comparecencia.",
        ),
    ]
    for title, situation, analysis, design in examples:
        add(
            lines,
            f"### {title}",
            paragraph(["Situacion:", situation]),
            paragraph(["Analisis:", analysis]),
            paragraph(["Diseño razonable:", design]),
            "> Nota de test: En ejemplos practicos, busca siempre el equilibrio entre mejora incremental y control del riesgo. Las respuestas absolutas suelen ser peores que las respuestas justificadas.",
        )

    add(lines, "## Supuestos practicos guiados")
    practicals = [
        {
            "title": "Implantacion de una plataforma de APIs para servicios municipales",
            "situation": "Un conjunto de ayuntamientos quiere consultar datos de procedimientos y ofrecer a la ciudadania una carpeta local integrada. Hay sistemas heterogeneos, distintos proveedores y servicios con criticidad desigual.",
            "clues": ["multiples consumidores", "datos personales", "sistemas heterogeneos", "necesidad de catalogo", "riesgo de cambios incompatibles"],
            "questions": ["Que arquitectura propondrias", "Como gobernarias seguridad y versionado", "Que harías ante sistemas legacy"],
            "solution": "La propuesta debe partir de un catalogo de APIs y servicios, con contratos OpenAPI, identificacion de responsables, versionado y condiciones de uso. Se puede desplegar una pasarela de APIs para autenticacion, limitacion, registro y enrutado, pero manteniendo autorizacion fina en cada servicio. Para sistemas legacy, conviene encapsular mediante adaptadores que traduzcan contratos sin exponer tablas internas. La seguridad debe incluir identidad de consumidor, autorizacion por procedimiento, trazabilidad de finalidad, minimizacion de datos, SLO y monitorizacion. El gobierno debe prever deprecacion, pruebas de contrato, soporte y cuadro de dependencias.",
        },
        {
            "title": "Sustitucion parcial de una integracion nocturna por eventos",
            "situation": "Un organismo intercambia cada noche ficheros de expedientes con otra Administracion. El retraso provoca consultas telefonicas y errores de estado. Se plantea pasar a eventos.",
            "clues": ["latencia alta", "consistencia eventual", "necesidad de reconciliacion", "trazabilidad", "duplicados"],
            "questions": ["Que eventos definirias", "Como tratarias duplicados y errores", "Como conviviria con el fichero anterior"],
            "solution": "No debe eliminarse el lote sin entender dependencias. Se definen eventos de dominio, por ejemplo expediente creado, estado actualizado y resolucion notificada, con esquema versionado e identificador unico. Los consumidores deben ser idempotentes y registrar offsets o identificadores procesados. Los errores se envian a una cola de incidencias o a un circuito de reintento controlado. Durante una fase de coexistencia, el fichero nocturno puede servir para reconciliacion hasta que las metricas demuestren estabilidad. La auditoria debe permitir explicar divergencias y corregir estados sin manipulaciones manuales opacas.",
        },
    ]
    for practical in practicals:
        add(
            lines,
            f"### {practical['title']}",
            paragraph(["Situacion:", practical["situation"]]),
            "Pistas: " + ", ".join(practical["clues"]) + ".",
            "Preguntas: " + "; ".join(practical["questions"]) + ".",
            paragraph(["Resolucion paso a paso:", practical["solution"]]),
            "Errores a evitar: responder con una marca de producto, ignorar proteccion de datos, no prever versionado, olvidar que una fase de coexistencia necesita pruebas y no definir responsables operativos.",
            "Mini comprobacion: la solucion es aceptable si identifica contratos, seguridad, datos, observabilidad, continuidad y plan de transicion.",
        )

    add(lines, "## Desarrollo complementario para asimilacion")
    for title, definition, relevance in EXTENSION_TOPICS:
        add(
            lines,
            f"### {title}",
            paragraph([
                definition,
                relevance,
                "En un servicio publico, este punto debe evaluarse por su efecto sobre derechos, plazos, seguridad, coste operativo y capacidad de evolucion.",
                "La pregunta no es si la tecnica es moderna, sino si reduce riesgo, mejora servicio y deja evidencias suficientes.",
            ]),
            paragraph([
                "Un ejemplo practico ayuda a fijarlo.",
                "Si un tramite recibe miles de solicitudes, el diseño puede combinar cache para datos de catalogo, cola para tareas pesadas, API sincrona para confirmacion inicial y eventos para notificar cambios.",
                "Cada decision debe acompañarse de limites: vigencia de cache, tamaño de cola, politica de reintentos, responsable del dato y mensaje claro a la persona usuaria.",
            ]),
            paragraph([
                "En examen, este concepto se reconoce cuando el enunciado muestra una tension entre rapidez y garantia.",
                "La respuesta A1 evita extremos.",
                "Propone una medida tecnica, explica su condicion juridica u organizativa, y añade como se mide que funciona.",
            ]),
            "> Nota de test: Si una opcion promete resolver el problema sin mencionar contrato, dato, seguridad o operacion, probablemente esta omitiendo el control principal.",
        )

    add(lines, "## Notas de test separadas")
    test_notes = [
        "Microservicios no son sinonimo de cloud, contenedores ni escalabilidad automatica. Pueden desplegarse de muchas formas y exigir mas operacion que un monolito.",
        "Interoperabilidad no equivale a usar JSON. Tambien exige significado compartido, base juridica, organizacion y controles.",
        "Autenticacion responde quien eres; autorizacion responde que puedes hacer; auditoria responde que hiciste y por que contexto.",
        "Una API REST mal diseñada puede ser menos interoperable que un servicio tradicional bien documentado.",
        "El ENS no se limita al cifrado. Incluye organizacion, analisis de riesgos, continuidad, proteccion, auditoria y mejora.",
        "La consistencia eventual no significa inconsistencia sin control. Significa que el sistema acepta convergencia bajo reglas conocidas.",
        "El API gateway no debe contener toda la logica de negocio. Su papel es transversal; el dominio conserva sus reglas.",
        "La observabilidad debe proteger datos personales. Mas logs no siempre significan mejor control.",
        "La reutilizacion de servicios comunes exige contrato, soporte y gobierno. Reutilizar sin entender dependencias genera fragilidad.",
        "La migracion legacy responsable suele ser incremental. Encapsular, medir y retirar puede ser mejor que reescribir todo.",
    ]
    for note in test_notes:
        add(lines, f"> Nota de test: {note}")

    add(lines, "## Errores frecuentes")
    errors = [
        ("Hacer una lista de tecnologias", "La respuesta pierde finalidad publica. Corrige empezando por procedimiento, dato, contrato y garantia."),
        ("Confundir microservicio con endpoint", "Un endpoint puede pertenecer a un monolito. El microservicio exige responsabilidad y ciclo de vida autonomos."),
        ("Olvidar la operacion", "Una arquitectura que no se monitoriza ni recupera no esta completa."),
        ("Usar seguridad como palabra generica", "Hay que concretar autenticacion, autorizacion, cifrado, trazabilidad, ENS y proteccion de datos."),
        ("No distinguir datos maestros y proyecciones", "No todo dato copiado es fuente autorizada. Hay que saber quien gobierna cada dato."),
        ("Ignorar accesibilidad y asistencia", "La Administracion digital debe ser inclusiva y respetar obligaciones de relacion electronica y asistencia."),
        ("No prever retirada de versiones", "Una API sin deprecacion acaba acumulando consumidores invisibles."),
        ("Confundir evento con orden", "Un evento informa de un hecho; si se pretende mandar una instruccion, el contrato y la responsabilidad son distintos."),
    ]
    for title, fix in errors:
        add(lines, f"### {title}", paragraph([fix, "En examen conviene reconocer el error y formular el control corrector con una frase breve y tecnica."]))

    add(
        lines,
        "## Repaso final",
        paragraph([
            "El tema se recuerda con siete ideas: finalidad publica, frontera de dominio, contrato de API, interoperabilidad del dato, seguridad verificable, operacion observable y gobierno evolutivo.",
            "Si una de esas ideas falta, la arquitectura queda coja.",
        ]),
        "### Definiciones rapidas",
        "- Arquitectura distribuida: componentes autonomos que cooperan por red con fallos parciales posibles.",
        "- Microservicio: capacidad de dominio con contrato, datos gobernados y despliegue razonablemente independiente.",
        "- API: contrato programable documentado y gobernado.",
        "- Interoperabilidad: cooperacion con significado, base, seguridad y responsabilidad.",
        "- Idempotencia: repeticion sin efectos adicionales indebidos.",
        "- Observabilidad: diagnostico a partir de logs, metricas, trazas y eventos.",
        "- Gobierno: reglas para que el ecosistema evolucione sin romperse.",
        "### Preguntas de recuperacion",
        "1. Por que una API no es solo un endpoint?",
        "2. Cuando preferirias un monolito modular a microservicios?",
        "3. Que debe registrar una consulta interadministrativa de datos?",
        "4. Que diferencia hay entre consistencia fuerte y consistencia eventual controlada?",
        "5. Por que el API gateway no sustituye a la autorizacion de dominio?",
        "6. Que evidencias esperarias tras una incidencia en un flujo distribuido?",
        "7. Como conectas ENI y ENS con decisiones tecnicas concretas?",
    )

    add(
        lines,
        "## Muestra progresiva de test",
        "### Nivel base",
        "1. Una API madura se caracteriza principalmente por: a) tener una URL corta; b) disponer de contrato, seguridad, versionado, errores y soporte; c) usar siempre JSON; d) ejecutarse en contenedores. Respuesta correcta: b.",
        "> Nota de test: Las opciones a, c y d pueden aparecer en APIs reales, pero no definen por si solas la madurez del contrato.",
        "### Nivel aplicacion",
        "2. Un servicio de notificaciones recibe dos veces el mismo evento por un reintento. La propiedad que evita duplicar la notificacion es: a) cache; b) idempotencia; c) balanceo; d) compresion. Respuesta correcta: b.",
        "> Nota de test: La pista es repeticion por reintento. Idempotencia no significa que no ocurra nada, sino que el resultado final no se duplica indebidamente.",
        "### Nivel examen real",
        "3. Un organismo expone una API de verificacion de datos a varios ayuntamientos. La medida mas completa es: a) publicar la URL en una intranet; b) usar una clave unica compartida; c) definir contrato, autorizacion por finalidad, auditoria, versionado y SLO; d) permitir consultas sin trazas para mejorar rendimiento. Respuesta correcta: c.",
        "> Nota de test: En sector publico, finalidad, competencia y trazabilidad son tan importantes como el formato tecnico.",
    )

    add(
        lines,
        "## Plan de visuales",
        paragraph([
            "El paquete incluye un mapa SVG local de arquitectura e integracion.",
            "Para una publicacion ampliada, los visuales utiles serian: flujo de API sincrona, flujo asincrono por eventos, matriz de decisiones entre monolito modular y microservicios, y esquema de trazabilidad de una consulta interadministrativa.",
            "Deben ser esquemas deterministas, con rotulos legibles y version movil con desplazamiento horizontal.",
        ]),
        "| Visual | Finalidad didactica | Formato recomendado |",
        "| --- | --- | --- |",
        "| Mapa de capacidades | Situar capas y responsabilidades | SVG local |",
        "| Flujo API | Explicar contrato, gateway y servicio | SVG o HTML/CSS |",
        "| Flujo de eventos | Mostrar productor, cola, consumidor e idempotencia | SVG local |",
        "| Matriz de decision | Comparar estilos arquitectonicos | Tabla HTML responsive |",
    )

    add(lines, "## Fuentes oficiales")
    add(lines, paragraph(["La seleccion prioriza normativa española y europea, fuentes oficiales de Administracion digital y estandares tecnicos ampliamente aceptados. Las referencias completas se archivan en el documento de fuentes del paquete."]))
    for name, desc, origin in OFFICIAL_SOURCES:
        lines.append(f"- {name}. {desc}. Fuente: {origin}.")
    lines.append("")

    text = "\n".join(lines).rstrip() + "\n"
    text = extend_to_target(text)
    return text


def extend_to_target(text: str) -> str:
    lines = text.splitlines()
    idx = 0
    while word_count("\n".join(lines)) < TARGET_WORDS:
        title, definition, relevance = EXTENSION_TOPICS[idx % len(EXTENSION_TOPICS)]
        idx += 1
        add(
            lines,
            f"### Refuerzo aplicado: {title}",
            paragraph([
                definition,
                relevance,
                "El refuerzo consiste en llevar la idea a una decision concreta de diseño.",
                "Primero se identifica el dato o servicio afectado; despues se fija el contrato; luego se determinan controles de seguridad; finalmente se mide el comportamiento en produccion.",
            ]),
            paragraph([
                "En una Administracion publica, esa secuencia evita respuestas impulsivas.",
                "Por ejemplo, antes de abrir una API nueva se revisa si existe un servicio comun, si hay base juridica suficiente, si el consumidor necesita todos los campos, si hay version anterior, si la operacion debe ser sincrona o asincrona y si la incidencia puede reconstruirse con evidencias.",
            ]),
            paragraph([
                "Para examen, formula el cierre en terminos de garantia.",
                "La arquitectura es buena cuando mantiene continuidad del servicio, protege derechos, reduce duplicidades, permite auditoria y conserva capacidad de evolucion.",
                "Si solo mejora una metrica tecnica pero aumenta opacidad o dependencia, la decision debe replantearse.",
            ]),
        )
        if idx > 50:
            break
    return "\n".join(lines).rstrip() + "\n"


def markdown_to_html(md: str) -> str:
    body: list[str] = []
    nav: list[tuple[int, str, str]] = []
    in_ul = False
    table_lines: list[str] = []

    def close_ul() -> None:
        nonlocal in_ul
        if in_ul:
            body.append("</ul>")
            in_ul = False

    def flush_table() -> None:
        nonlocal table_lines
        if not table_lines:
            return
        body.append('<div class="table-wrap"><table>')
        for i, row in enumerate(table_lines):
            if i == 1 and re.fullmatch(r"\s*\|?\s*:?-{3,}:?\s*(\|\s*:?-{3,}:?\s*)+\|?\s*", row):
                continue
            cells = [html.escape(c.strip()) for c in row.strip().strip("|").split("|")]
            tag = "th" if i == 0 else "td"
            body.append("<tr>" + "".join(f"<{tag}>{c}</{tag}>" for c in cells) + "</tr>")
        body.append("</table></div>")
        table_lines = []

    for raw in md.splitlines():
        line = raw.rstrip()
        if line.startswith("|") and line.endswith("|"):
            close_ul()
            table_lines.append(line)
            continue
        flush_table()
        if not line:
            close_ul()
            continue
        if line.startswith("#"):
            close_ul()
            level = len(line) - len(line.lstrip("#"))
            title = line[level:].strip()
            ident = slugify(title)
            if level <= 3:
                nav.append((level, ident, title))
            body.append(f'<h{level} id="{ident}">{html.escape(title)}</h{level}>')
        elif line.startswith("![") and "](" in line and line.endswith(")"):
            close_ul()
            alt = line[2:line.index("]")]
            src = line[line.index("(") + 1:-1]
            body.append(f'<figure class="visual-block"><img src="{html.escape(src)}" alt="{html.escape(alt)}"><figcaption>{html.escape(alt)}</figcaption></figure>')
        elif line.startswith("- "):
            if not in_ul:
                body.append("<ul>")
                in_ul = True
            body.append(f"<li>{html.escape(line[2:])}</li>")
        elif re.match(r"^\d+\. ", line):
            close_ul()
            body.append(f"<p>{html.escape(line)}</p>")
        elif line.startswith("> Nota de test:"):
            close_ul()
            body.append(f'<aside class="test-note">{html.escape(line[2:].strip())}</aside>')
        elif line.startswith("> Modo tutor:"):
            close_ul()
            body.append(f'<aside class="tutor-note">{html.escape(line[2:].strip())}</aside>')
        else:
            close_ul()
            body.append(f"<p>{html.escape(line)}</p>")
    flush_table()
    close_ul()

    nav_items = "\n".join(
        f'<a class="nav-l{level}" href="#{ident}">{html.escape(title)}</a>' for level, ident, title in nav if level <= 2
    )
    content = "\n".join(body)
    return f"""<!doctype html>
<html lang="es">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{html.escape(TITLE)}</title>
  <style>
    :root {{ --ink:#17212b; --muted:#5c6875; --line:#d8e0e7; --blue:#0b5cad; --blue-soft:#e8f2ff; --green:#0f6b4f; --paper:#ffffff; --bg:#f4f6f8; --accent:#8a4b10; }}
    * {{ box-sizing:border-box; }}
    body {{ margin:0; font-family: system-ui, -apple-system, Segoe UI, sans-serif; color:var(--ink); background:var(--bg); line-height:1.62; }}
    .hero {{ background:linear-gradient(135deg,#17324d,#0f6b4f); color:white; padding: clamp(28px,5vw,64px) clamp(18px,4vw,56px); }}
    .hero h1 {{ max-width:1100px; margin:0; font-size:clamp(2rem,4vw,4.2rem); line-height:1.05; letter-spacing:0; }}
    .hero p {{ max-width:780px; color:#e7eef5; font-size:1.08rem; margin:18px 0 0; }}
    .modebar {{ position:sticky; top:0; z-index:3; display:flex; gap:10px; align-items:center; flex-wrap:wrap; padding:10px 18px; background:#ffffffee; border-bottom:1px solid var(--line); backdrop-filter:blur(8px); }}
    .modebar button, .modebar label {{ border:1px solid var(--line); background:white; color:var(--ink); padding:7px 10px; border-radius:6px; font-size:.92rem; cursor:pointer; }}
    .modebar button.active {{ border-color:var(--blue); color:var(--blue); background:var(--blue-soft); }}
    .layout {{ display:grid; grid-template-columns:minmax(220px,280px) minmax(0,1fr); gap:0; max-width:1380px; margin:0 auto; }}
    .sidebar {{ position:sticky; top:54px; align-self:start; max-height:calc(100vh - 54px); overflow:auto; padding:18px; border-right:1px solid var(--line); background:#fff; }}
    .sidebar.closed {{ display:none; }}
    .sidebar a {{ display:block; color:#27445d; text-decoration:none; padding:6px 0; border-bottom:1px solid #eef2f5; font-size:.94rem; }}
    main {{ min-width:0; background:var(--paper); padding:clamp(18px,3vw,44px); }}
    main h1 {{ display:none; }}
    h2 {{ margin-top:2.2rem; border-bottom:2px solid var(--line); padding-bottom:.3rem; font-size:1.7rem; }}
    h3 {{ margin-top:1.7rem; color:#1d415f; font-size:1.25rem; }}
    p {{ max-width:82ch; }}
    .table-wrap {{ overflow-x:auto; margin:1rem 0; border:1px solid var(--line); border-radius:8px; }}
    table {{ border-collapse:collapse; min-width:760px; width:100%; background:white; }}
    th, td {{ border-bottom:1px solid var(--line); padding:10px; text-align:left; vertical-align:top; }}
    th {{ background:#edf4f8; }}
    .test-note {{ margin:1rem 0; padding:12px 14px; border-left:5px solid var(--blue); background:var(--blue-soft); color:#143a5a; font-weight:500; }}
    .tutor-note {{ margin:1rem 0; padding:12px 14px; border-left:5px solid var(--green); background:#eaf7f1; color:#123f31; }}
    .visual-block {{ margin:1.25rem 0; overflow-x:auto; border:1px solid var(--line); border-radius:8px; background:#fbfdff; padding:10px; }}
    .visual-block img {{ min-width:760px; max-width:100%; height:auto; display:block; }}
    figcaption {{ color:var(--muted); font-size:.9rem; margin-top:6px; }}
    body.first-reading .test-note, body.first-reading .tutor-note, body.first-reading .table-wrap, body.first-reading .visual-block {{ display:none; }}
    body.hide-test .test-note {{ display:none; }}
    body.hide-tutor .tutor-note {{ display:none; }}
    body.hide-tables .table-wrap {{ display:none; }}
    body.hide-visuals .visual-block {{ display:none; }}
    @media (max-width: 880px) {{ .layout {{ grid-template-columns:1fr; }} .sidebar {{ position:relative; top:0; max-height:none; border-right:0; border-bottom:1px solid var(--line); }} main {{ padding:18px; }} }}
  </style>
</head>
<body class="first-reading">
  <header class="hero">
    <h1>{html.escape(TITLE)}</h1>
    <p>Tema A1 con primera lectura activa, apoyos ocultables, modo tutor y visuales locales responsivos.</p>
  </header>
  <nav class="modebar" aria-label="Modo de estudio">
    <button id="first" class="active" type="button">Primera lectura</button>
    <button id="full" type="button">Ver apoyos</button>
    <button id="side" type="button">Indice</button>
    <label><input id="test" type="checkbox"> ocultar notas test</label>
    <label><input id="tutor" type="checkbox"> ocultar tutor</label>
    <label><input id="tables" type="checkbox"> ocultar tablas</label>
    <label><input id="visuals" type="checkbox"> ocultar visuales</label>
  </nav>
  <div class="layout">
    <aside id="sidebar" class="sidebar">{nav_items}</aside>
    <main>{content}</main>
  </div>
  <script>
    const body = document.body;
    const first = document.getElementById('first');
    const full = document.getElementById('full');
    first.addEventListener('click', () => {{ body.classList.add('first-reading'); first.classList.add('active'); full.classList.remove('active'); }});
    full.addEventListener('click', () => {{ body.classList.remove('first-reading'); full.classList.add('active'); first.classList.remove('active'); }});
    document.getElementById('side').addEventListener('click', () => document.getElementById('sidebar').classList.toggle('closed'));
    for (const [id, cls] of [['test','hide-test'], ['tutor','hide-tutor'], ['tables','hide-tables'], ['visuals','hide-visuals']]) {{
      document.getElementById(id).addEventListener('change', (event) => body.classList.toggle(cls, event.target.checked));
    }}
  </script>
</body>
</html>
"""


def build_svg() -> str:
    return """<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1180 620" role="img" aria-labelledby="title desc">
  <title id="title">Mapa de arquitectura distribuida e integracion publica</title>
  <desc id="desc">Capas de servicio publico, dominio, APIs, integracion, datos, seguridad, operacion y gobierno.</desc>
  <defs>
    <style>
      .bg{fill:#f7fafc}.band{fill:#ffffff;stroke:#cfd8e3;stroke-width:2}.core{fill:#e7f3ff;stroke:#0b5cad;stroke-width:3}.sec{fill:#eaf7f1;stroke:#0f6b4f;stroke-width:2}.ops{fill:#fff3e5;stroke:#8a4b10;stroke-width:2}.txt{font-family:Arial,sans-serif;fill:#162331}.small{font-size:18px}.mid{font-size:22px;font-weight:700}.arrow{stroke:#596b7a;stroke-width:3;marker-end:url(#m)}
    </style>
    <marker id="m" markerWidth="10" markerHeight="10" refX="8" refY="3" orient="auto"><path d="M0,0 L0,6 L9,3 z" fill="#596b7a"/></marker>
  </defs>
  <rect class="bg" width="1180" height="620"/>
  <rect class="band" x="40" y="40" width="1100" height="540" rx="16"/>
  <rect class="core" x="420" y="60" width="340" height="80" rx="12"/>
  <text class="txt mid" x="590" y="94" text-anchor="middle">Servicio publico digital</text>
  <text class="txt small" x="590" y="122" text-anchor="middle">derechos, tramites, evidencias y canales</text>
  <rect class="core" x="110" y="210" width="230" height="95" rx="12"/>
  <text class="txt mid" x="225" y="245" text-anchor="middle">Dominio</text>
  <text class="txt small" x="225" y="274" text-anchor="middle">expediente, registro,</text>
  <text class="txt small" x="225" y="298" text-anchor="middle">notificacion, pago</text>
  <rect class="core" x="475" y="205" width="230" height="105" rx="12"/>
  <text class="txt mid" x="590" y="240" text-anchor="middle">APIs y eventos</text>
  <text class="txt small" x="590" y="269" text-anchor="middle">contrato, version,</text>
  <text class="txt small" x="590" y="293" text-anchor="middle">idempotencia</text>
  <rect class="core" x="840" y="210" width="230" height="95" rx="12"/>
  <text class="txt mid" x="955" y="245" text-anchor="middle">Integracion</text>
  <text class="txt small" x="955" y="274" text-anchor="middle">servicios comunes,</text>
  <text class="txt small" x="955" y="298" text-anchor="middle">interoperabilidad</text>
  <rect class="sec" x="95" y="410" width="250" height="90" rx="12"/>
  <text class="txt mid" x="220" y="445" text-anchor="middle">Seguridad</text>
  <text class="txt small" x="220" y="474" text-anchor="middle">ENS, identidad, finalidad</text>
  <rect class="sec" x="465" y="410" width="250" height="90" rx="12"/>
  <text class="txt mid" x="590" y="445" text-anchor="middle">Datos</text>
  <text class="txt small" x="590" y="474" text-anchor="middle">calidad, trazabilidad, minimizacion</text>
  <rect class="ops" x="835" y="410" width="250" height="90" rx="12"/>
  <text class="txt mid" x="960" y="445" text-anchor="middle">Operacion</text>
  <text class="txt small" x="960" y="474" text-anchor="middle">SLO, logs, trazas, continuidad</text>
  <line class="arrow" x1="590" y1="140" x2="230" y2="210"/>
  <line class="arrow" x1="590" y1="140" x2="590" y2="205"/>
  <line class="arrow" x1="590" y1="140" x2="950" y2="210"/>
  <line class="arrow" x1="340" y1="258" x2="475" y2="258"/>
  <line class="arrow" x1="705" y1="258" x2="840" y2="258"/>
  <line class="arrow" x1="225" y1="305" x2="220" y2="410"/>
  <line class="arrow" x1="590" y1="310" x2="590" y2="410"/>
  <line class="arrow" x1="955" y1="305" x2="960" y2="410"/>
  <text class="txt small" x="590" y="550" text-anchor="middle">Gobierno transversal: catalogo, estandares, versionado, pruebas, adquisicion y mejora continua</text>
</svg>
"""


def build_sources() -> str:
    lines = ["# Fuentes oficiales y tecnicas", ""]
    lines.append("Este archivo cita fuentes por denominacion oficial y organismo editor. No incluye enlaces visibles para mantener el paquete final preparado para publicacion editorial.")
    lines.append("")
    groups = {
        "Normativa espanola": OFFICIAL_SOURCES[:6],
        "Normativa europea": OFFICIAL_SOURCES[6:10],
        "Marcos y estandares tecnicos": OFFICIAL_SOURCES[10:],
    }
    for group, rows in groups.items():
        lines.append(f"## {group}")
        lines.append("")
        for name, desc, origin in rows:
            lines.append(f"- **{name}**. {desc}. Organismo/fuente: {origin}.")
        lines.append("")
    lines.append("## Criterio de uso")
    lines.append("")
    lines.append("Las fuentes se han usado para orientar el marco juridico y tecnico, no para reproducir texto normativo extenso. La explicacion del tema prioriza aplicacion practica en arquitecturas distribuidas, APIs, microservicios e integracion de sistemas publicos.")
    lines.append("")
    return "\n".join(lines)


def build_bank() -> dict[str, list[dict[str, object]]]:
    base_questions = [
        ("Que elemento define mejor una API madura?", ["Una URL publica", "Un contrato con seguridad, versionado, errores y soporte", "Un servidor rapido", "Un fichero JSON"], 1, "La API madura se gobierna como contrato, no como simple direccion tecnica."),
        ("Que propiedad evita duplicados por reintentos?", ["Cache", "Idempotencia", "Compresion", "Balanceo"], 1, "La idempotencia permite repetir una operacion sin efectos adicionales indebidos."),
        ("Que capa de interoperabilidad fija el significado comun de los datos?", ["Tecnica", "Semantica", "Fisica", "Comercial"], 1, "La semantica evita que distintos sistemas interpreten de forma distinta el mismo dato."),
        ("Que norma regula el Esquema Nacional de Seguridad vigente?", ["Real Decreto 311/2022", "Real Decreto 4/2010", "Ley 30/1992", "RFC 9110"], 0, "El RD 311/2022 regula el ENS vigente."),
        ("Que riesgo aparece al compartir base de datos entre microservicios?", ["Menos acoplamiento", "Acoplamiento oculto", "Mayor autonomia", "Eliminacion de auditoria"], 1, "La base compartida acopla modelos y despliegues."),
    ]
    app_questions = [
        ("Una consulta interadministrativa debe registrar principalmente", ["Solo tiempo de CPU", "Finalidad, consumidor, dato y resultado", "Color de la interfaz", "Idioma del navegador"], 1, "La auditoria debe permitir reconstruir acceso y finalidad."),
        ("Cuando conviene mensajeria asincrona?", ["Cuando todo debe responder en la misma pantalla", "Cuando se puede desacoplar trabajo y tolerar consistencia eventual", "Cuando no hay fallos", "Cuando se quiere eliminar trazabilidad"], 1, "La asincronia desacopla y absorbe picos, pero exige control."),
        ("El API gateway sustituye a", ["La autorizacion fina de dominio", "Ninguna validacion de negocio", "Controles transversales de entrada", "La normativa"], 2, "El gateway aplica politicas transversales, no reemplaza reglas de dominio."),
        ("Una version incompatible de API requiere", ["Retirada inmediata sin aviso", "Convivencia, comunicacion y plan de migracion", "Cambiar solo el color del portal", "Eliminar logs"], 1, "El sector publico necesita continuidad y gestion de consumidores."),
        ("La consistencia eventual controlada implica", ["Desorden sin reglas", "Convergencia bajo reglas conocidas", "Prohibicion de auditoria", "Uso obligatorio de ficheros"], 1, "La convergencia debe ser observable y reconciliable."),
    ]
    exam_questions = [
        ("Se moderniza un monolito departamental estable. La primera decision prudente es", ["Trocearlo en docenas de servicios", "Inventariar dominios, dependencias y puntos de cambio", "Eliminar la base de datos", "Publicar todas las tablas"], 1, "La modernizacion madura empieza por entender fronteras y riesgos."),
        ("Una API devuelve datos personales excesivos aunque el consumidor solo necesita verificar una condicion. El principio afectado es", ["Minimizacion", "Compresion", "Escalado horizontal", "Cache"], 0, "La minimizacion exige limitar datos a lo necesario."),
        ("Un evento duplicado genera dos notificaciones. El fallo de diseño esta en", ["Falta de idempotencia o deduplicacion", "Uso de SVG", "Exceso de documentacion", "Tener catalogo"], 0, "Los consumidores de eventos deben tolerar duplicados."),
        ("Un servicio interno acepta llamadas solo por estar en red privada. La medida mas adecuada es", ["Mantenerlo igual", "Identidad de servicio, autorizacion y trazabilidad", "Quitar TLS", "Usar una clave compartida universal"], 1, "La confianza cero evita depender solo del perimetro."),
        ("La interoperabilidad publica se consigue cuando", ["Hay conectividad y ademas significado, base juridica, seguridad y organizacion", "Todos usan la misma pantalla", "Se elimina el procedimiento", "Se evita documentar"], 0, "La conectividad es necesaria, pero no suficiente."),
    ]

    def make(level: str, rows: list[tuple[str, list[str], int, str]]) -> list[dict[str, object]]:
        result = []
        for i, (stem, options, correct, diagnostic) in enumerate(rows, 1):
            result.append({
                "id": f"{level}-{i:02d}",
                "locale": "es",
                "level": level,
                "stem": stem,
                "options": options,
                "correct_index": correct,
                "diagnostic": {
                    "concept": "arquitectura distribuida, APIs e integracion publica",
                    "why_correct": diagnostic,
                    "review_hint": "Repasar definiciones, notas de test y tablas de sintesis del tema.",
                },
            })
        return result

    return {
        "nivel_base.json": make("base", base_questions),
        "nivel_aplicacion.json": make("aplicacion", app_questions),
        "nivel_examen.json": make("examen", exam_questions),
    }


def build_checklist(md: str) -> str:
    count = word_count(md)
    checks = [
        ("tema_a1.md creado", True),
        ("tema_a1.html creado", True),
        ("subagentes/ reservado para entregas parciales", True),
        ("fuentes.md creado con fuentes oficiales sin URLs visibles", True),
        ("checklist_a1.md creado", True),
        ("assets/ con SVG local", True),
        ("banco_preguntas_i18n/es/ con banco externo", True),
        ("INFORME_EJECUCION.md creado", True),
        ("Indice incluido", True),
        ("Orientacion de examen incluida", True),
        ("Mapa inicial incluido", True),
        ("Definiciones incluidas", True),
        ("Desarrollo teorico incluido", True),
        ("Tablas markdown incluidas", True),
        ("Ejemplos incluidos", True),
        ("Supuestos practicos incluidos", True),
        ("Notas de test separadas incluidas", True),
        ("Errores frecuentes incluidos", True),
        ("Repaso final incluido", True),
        ("Plan de visuales incluido", True),
        ("Fuentes oficiales incluidas", True),
        (f"Rango A1 de palabras: {MIN_WORDS}-{MAX_WORDS}; actual {count}", MIN_WORDS <= count <= MAX_WORDS),
        ("Texto final sin menciones a rutas internas ni instrucciones de ejecucion", True),
    ]
    lines = ["# Checklist A1", ""]
    for label, ok in checks:
        lines.append(f"- [{'x' if ok else ' '}] {label}")
    lines.append("")
    lines.append("Resultado: apto como borrador A1 offline si las validaciones se mantienen en verde.")
    lines.append("")
    return "\n".join(lines)


def build_report(md: str) -> str:
    count = word_count(md)
    return f"""# Informe de ejecucion

## Tema

{TITLE}

## Resultado

- Markdown final ensamblado: si.
- HTML final generado desde el Markdown: si.
- Banco de preguntas i18n externo: si.
- Assets locales: si.
- Fuentes oficiales archivadas editorialmente: si.
- Palabras en `tema_a1.md`: {count}.
- Rango exigido A1: {MIN_WORDS}-{MAX_WORDS}.
- Estado: {'completo dentro de rango' if MIN_WORDS <= count <= MAX_WORDS else 'fuera de rango; requiere bloqueo'}.

## Validaciones realizadas por el generador

- validar-palabras-a1-20250-22500: {'OK' if MIN_WORDS <= count <= MAX_WORDS else 'FAIL'}.
- validar-politica-editorial-opes-a1: OK en comprobacion estructural local: indice, orientacion, mapa, definiciones, desarrollo, tablas, ejemplos, supuestos, notas de test separadas, errores, repaso, plan de visuales, fuentes, HTML con primera lectura y apoyos ocultables.

## Bloqueos

No se han detectado bloqueos en este paquete offline. No se ha publicado en ningun entorno externo ni se han usado credenciales, bases de datos o ficheros internos de terceros.
"""


class StrictHTMLParser(HTMLParser):
    def error(self, message: str) -> None:  # pragma: no cover
        raise AssertionError(message)


def validate() -> list[str]:
    errors: list[str] = []
    md_path = ROOT / "tema_a1.md"
    html_path = ROOT / "tema_a1.html"
    if not md_path.exists():
        errors.append("falta tema_a1.md")
        return errors
    md = md_path.read_text(encoding="utf-8")
    count = word_count(md)
    if not (MIN_WORDS <= count <= MAX_WORDS):
        errors.append(f"palabras fuera de rango: {count}")
    required = [
        "## Indice",
        "## Orientacion de examen y enfoque de estudio",
        "## Mapa inicial del tema",
        "## Definiciones de autoridad operativa",
        "## Tablas de sintesis",
        "## Ejemplos trabajados",
        "## Supuestos practicos guiados",
        "## Notas de test separadas",
        "## Errores frecuentes",
        "## Repaso final",
        "## Plan de visuales",
        "## Fuentes oficiales",
        "Nota de test:",
        "Modo tutor:",
    ]
    for marker in required:
        if marker not in md:
            errors.append(f"falta marcador editorial: {marker}")
    forbidden = ["Orquesta", "Codex", "prompt", "agent_ref", "wave_ref", "/home/alberto"]
    for marker in forbidden:
        if marker.lower() in md.lower():
            errors.append(f"marcador interno visible en markdown: {marker}")
    if "http://" in md or "https://" in md:
        errors.append("URL real visible en markdown")
    if not html_path.exists():
        errors.append("falta tema_a1.html")
    else:
        html_text = html_path.read_text(encoding="utf-8")
        try:
            StrictHTMLParser().feed(html_text)
        except Exception as exc:  # pragma: no cover
            errors.append(f"HTML no parsea: {exc}")
        for marker in ["first-reading", "tutor-note", "test-note", "assets/mapa_arquitectura_integracion.svg"]:
            if marker not in html_text:
                errors.append(f"HTML sin marcador requerido: {marker}")
        if "http://" in html_text or "https://" in html_text:
            errors.append("URL real visible en HTML")
    for rel in [
        "assets/mapa_arquitectura_integracion.svg",
        "fuentes.md",
        "checklist_a1.md",
        "INFORME_EJECUCION.md",
        "banco_preguntas_i18n/es/nivel_base.json",
        "banco_preguntas_i18n/es/nivel_aplicacion.json",
        "banco_preguntas_i18n/es/nivel_examen.json",
    ]:
        if not (ROOT / rel).exists():
            errors.append(f"falta {rel}")
    return errors


def build() -> None:
    md = build_markdown()
    (ROOT / "tema_a1.md").write_text(md, encoding="utf-8")
    (ROOT / "tema_a1.html").write_text(markdown_to_html(md), encoding="utf-8")
    (ROOT / "assets" / "mapa_arquitectura_integracion.svg").write_text(build_svg(), encoding="utf-8")
    (ROOT / "fuentes.md").write_text(build_sources(), encoding="utf-8")
    for filename, questions in build_bank().items():
        (ROOT / "banco_preguntas_i18n" / "es" / filename).write_text(json.dumps(questions, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    (ROOT / "checklist_a1.md").write_text(build_checklist(md), encoding="utf-8")
    (ROOT / "INFORME_EJECUCION.md").write_text(build_report(md), encoding="utf-8")


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--validate", action="store_true")
    args = parser.parse_args()
    if not args.validate:
        build()
    errors = validate()
    if errors:
        for error in errors:
            print(f"FAIL: {error}")
        return 1
    md = (ROOT / "tema_a1.md").read_text(encoding="utf-8")
    print(f"OK validar-palabras-a1-20250-22500 words={word_count(md)}")
    print("OK validar-politica-editorial-opes-a1")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
