package orquestaweb

import (
	"sort"
	"strings"
)

func nuevaAppWizardHelpI18nKeysV0() []string {
	keys := make([]string, 0, len(nuevaAppWizardHelpI18nSpanishV0()))
	for key := range nuevaAppWizardHelpI18nSpanishV0() {
		keys = append(keys, key)
	}
	for key := range nuevaAppWizardUniversalI18nSpanishV0() {
		keys = append(keys, key)
	}
	for key := range nuevaAppWizardUniversalGeneratedI18nSpanishV0() {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func nuevaAppWizardOptionExampleKeysV0() map[string]bool {
	return map[string]bool{
		"nueva_app.wizard.example.option.tipo_app.api":                        true,
		"nueva_app.wizard.example.option.deploy.contenedor":                   true,
		"nueva_app.wizard.example.option.deploy.paas":                         true,
		"nueva_app.wizard.example.option.deploy.serverless":                   true,
		"nueva_app.wizard.example.option.deploy.kubernetes":                   true,
		"nueva_app.wizard.example.option.integracion.calendar.caldav":         true,
		"nueva_app.wizard.example.option.integracion.ecommerce.payments_sync": true,
		"nueva_app.wizard.example.option.integracion.maps.public_sources":     true,
		"nueva_app.wizard.example.option.storage.relacional":                  true,
		"nueva_app.wizard.example.option.storage.documental":                  true,
		"nueva_app.wizard.example.option.storage.mixta":                       true,
		"nueva_app.wizard.example.option.integracion.gobierno.simple":         true,
		"nueva_app.wizard.example.option.integracion.gobierno.oauth_alta":     true,
	}
}

func wizardGlossaryMarkdownV0() string {
	catalog := NewNuevaAppI18nCatalogV0()
	keys := nuevaAppWizardHelpI18nKeysV0()
	var builder strings.Builder
	builder.WriteString("# Glosario del wizard de nueva app\n\n")
	builder.WriteString("Fuente generada desde el catalogo i18n es/en del wizard.\n\n")
	for _, key := range keys {
		if !strings.Contains(key, ".help.") {
			continue
		}
		es := catalog.lookupExact(NuevaAppI18nDefaultLocaleV0, key)
		en := catalog.lookupExact(NuevaAppI18nEnglishLocaleV0, key)
		builder.WriteString("## ")
		builder.WriteString(key)
		builder.WriteString("\n\n")
		builder.WriteString("- es: ")
		builder.WriteString(es)
		builder.WriteString("\n")
		builder.WriteString("- en: ")
		builder.WriteString(en)
		builder.WriteString("\n\n")
		exampleKey := strings.Replace(key, ".help.", ".example.", 1)
		if exampleES := catalog.lookupExact(NuevaAppI18nDefaultLocaleV0, exampleKey); exampleES != "" {
			builder.WriteString("- ejemplo es: ")
			builder.WriteString(exampleES)
			builder.WriteString("\n")
			builder.WriteString("- example en: ")
			builder.WriteString(catalog.lookupExact(NuevaAppI18nEnglishLocaleV0, exampleKey))
			builder.WriteString("\n\n")
		}
	}
	return builder.String()
}

func nuevaAppWizardHelpI18nSpanishV0() map[string]string {
	return map[string]string{
		"nueva_app.wizard.question.wizard-q-locale.help":                      "Pregunta por el idioma principal de la app. Conviene elegir el idioma que usaran primero las personas reales. Ejemplo: si tu equipo trabaja en espanol, la app empieza con textos y documentos en espanol.",
		"nueva_app.wizard.question.wizard-q-nombre.help":                      "Pregunta por el nombre publico o interno de la app. Conviene fijarlo pronto para que documentos y tareas usen la misma referencia. Ejemplo: una app de turnos puede llamarse Turnos Clinica.",
		"nueva_app.wizard.question.wizard-q-objetivo.help":                    "Pregunta que resultado concreto debe conseguir la app. Conviene responder con algo que luego se pueda comprobar. Ejemplo: permitir reservar una cita y recibir confirmacion.",
		"nueva_app.wizard.question.wizard-q-tipo-app.help":                    "Pregunta cual sera la forma principal de uso. Conviene elegir la superficie donde estara la mayor parte del valor. Ejemplo: web si se usara desde navegador.",
		"nueva_app.wizard.question.wizard-q-datos.help":                       "Pregunta si la app crea, guarda o solo consulta informacion. Conviene aclararlo para no construir almacenamiento innecesario. Ejemplo: una calculadora simple puede no guardar nada.",
		"nueva_app.wizard.question.wizard-q-deploy.help":                      "Pregunta donde se ejecutara la primera version. Conviene elegir el destino mas cercano al uso real. Ejemplo: local para probar en un ordenador antes de publicar.",
		"nueva_app.wizard.question.wizard-r1-uso-personal-compartido.help":    "Pregunta si la app la usara una sola persona o varias. Conviene distinguirlo porque compartir suele pedir cuentas y permisos. Ejemplo: agenda personal frente a agenda de recepcion.",
		"nueva_app.wizard.question.wizard-r2-plataformas.help":                "Pregunta en que dispositivos se usara. Conviene elegir los dispositivos habituales de los usuarios. Ejemplo: movil y PC para una agenda que se mira en oficina y fuera.",
		"nueva_app.wizard.question.wizard-r3-integracion-agenda.help":         "Pregunta si la app debe conectarse con calendarios externos. Conviene activarlo cuando ya existen citas en otra herramienta. Ejemplo: leer huecos de una agenda de empresa.",
		"nueva_app.wizard.question.wizard-r3-integracion-tienda.help":         "Pregunta si habra catalogo, carrito o cobros. Conviene activarlo cuando el flujo incluye vender o cobrar. Ejemplo: reservar una clase y pagarla.",
		"nueva_app.wizard.question.wizard-r3-integracion-mapa.help":           "Pregunta si la ubicacion forma parte del uso. Conviene activarlo cuando hay direcciones, rutas o busqueda por cercania. Ejemplo: mostrar centros cercanos.",
		"nueva_app.wizard.question.wizard-r3-integracion-inventario.help":     "Pregunta si hay existencias que suben y bajan. Conviene activarlo cuando importa saber cantidad disponible. Ejemplo: controlar material en almacen.",
		"nueva_app.wizard.question.wizard-r3-integracion-notas.help":          "Pregunta si la app manejara textos, documentos o conocimiento interno. Conviene activarlo cuando se escriben y consultan contenidos. Ejemplo: una wiki de procedimientos.",
		"nueva_app.wizard.question.wizard-r3-integracion-tareas.help":         "Pregunta si hay trabajos con responsables y estados. Conviene activarlo cuando se organiza trabajo pendiente. Ejemplo: tablero de tareas de un equipo.",
		"nueva_app.wizard.question.wizard-r3-integracion-finanzas.help":       "Pregunta si se manejan gastos, presupuestos o extractos. Conviene activarlo cuando el dinero necesita seguimiento. Ejemplo: registrar gastos mensuales.",
		"nueva_app.wizard.question.wizard-r3-integracion-crm.help":            "Pregunta si se gestionan clientes o contactos. Conviene activarlo cuando hay seguimiento comercial o de atencion. Ejemplo: ver llamadas pendientes de cada cliente.",
		"nueva_app.wizard.question.wizard-r3-integracion-reservas.help":       "Pregunta si se reservan plazas, citas o turnos. Conviene activarlo cuando hay disponibilidad limitada. Ejemplo: pedir hora en una consulta.",
		"nueva_app.wizard.question.wizard-r3-integracion-salud.help":          "Pregunta si hay datos de salud, habitos o actividad fisica. Conviene activarlo cuando la privacidad es importante. Ejemplo: registrar mediciones diarias.",
		"nueva_app.wizard.question.wizard-r3-integracion-educacion.help":      "Pregunta si hay cursos, alumnos o aprendizaje. Conviene activarlo cuando se mide progreso. Ejemplo: marcar lecciones completadas.",
		"nueva_app.wizard.question.wizard-r3-integracion-comunidad.help":      "Pregunta si habra conversaciones o participacion de usuarios. Conviene activarlo cuando se necesita moderar. Ejemplo: foro de vecinos.",
		"nueva_app.wizard.question.wizard-r3-integracion-iot.help":            "Pregunta si la app hablara con sensores o dispositivos. Conviene activarlo cuando hay lecturas automaticas. Ejemplo: ver temperatura enviada por un sensor.",
		"nueva_app.wizard.question.wizard-r3-integracion-media.help":          "Pregunta si se subiran o procesaran fotos, videos u otros archivos visuales. Conviene activarlo cuando el contenido pesado es parte central. Ejemplo: galeria de obras terminadas.",
		"nueva_app.wizard.question.wizard-r3-integracion-facturacion.help":    "Pregunta si se emitiran facturas o documentos legales. Conviene activarlo cuando hay requisitos fiscales. Ejemplo: generar una factura PDF numerada.",
		"nueva_app.wizard.question.wizard-r3-dominio-abierto.help":            "Pregunta por el trabajo cotidiano cuando el wizard no reconoce el dominio. Conviene describir un dia normal para evitar suposiciones. Ejemplo: abrir ficha, anotar avance y cerrar tarea.",
		"nueva_app.wizard.question.wizard-r4-storage-db.help":                 "Pregunta que tipo de guardado encaja con los datos. Conviene elegir segun si la informacion es ordenada, flexible o mixta. Ejemplo: clientes en tabla y documentos adjuntos.",
		"nueva_app.wizard.question.wizard-r5-integracion-gobierno.help":       "Pregunta como se controla una conexion con otra herramienta. Conviene responder segun el riesgo y si necesita acceso protegido. Ejemplo: una API publica no exige lo mismo que datos de clientes.",
		"nueva_app.wizard.question.wizard-r6-movil-plataformas.help":          "Pregunta que sistemas moviles debe cubrir. Conviene elegir donde estan los usuarios reales. Ejemplo: iOS y Android si el equipo usa ambos.",
		"nueva_app.wizard.question.wizard-r7-deploy-compatible.help":          "Pregunta por un destino alternativo cuando el elegido no encaja. Conviene corregirlo para poder construir y probar. Ejemplo: una app movil necesita paquete movil, no solo escritorio.",
		"nueva_app.wizard.question.wizard-r8-usuarios-compartido.help":        "Pregunta que grupo usara la app compartida. Conviene saberlo para ajustar permisos y pantallas. Ejemplo: equipo pequeno con administrador y lectores.",
		"nueva_app.wizard.help.option.locale.es":                              "Espanol significa que los textos iniciales salen en espanol. Conviene si la mayoria del equipo trabaja en espanol.",
		"nueva_app.wizard.help.option.locale.en":                              "Ingles significa que los textos iniciales salen en ingles. Conviene si usuarios o documentacion principal estaran en ingles.",
		"nueva_app.wizard.help.option.nombre.inferido":                        "Nombre inferido usa el objetivo para proponer un nombre. Conviene si aun no tienes marca definida.",
		"nueva_app.wizard.help.option.nombre.generico":                        "Nueva app es un nombre temporal. Conviene solo para empezar rapido y cambiarlo despues.",
		"nueva_app.wizard.help.option.objetivo.describir":                     "Describir objetivo verificable obliga a contar que debe pasar al final. Conviene cuando aun falta cerrar el resultado esperado.",
		"nueva_app.wizard.help.option.tipo_app.web":                           "Web es una app usada desde navegador. Conviene cuando quieres acceso facil desde varios ordenadores o moviles.",
		"nueva_app.wizard.help.option.tipo_app.api":                           "API es una puerta para que otros programas pidan o envien datos. Conviene cuando no necesitas pantalla principal.",
		"nueva_app.wizard.help.option.tipo_app.mobile":                        "Movil es una app pensada para telefono. Conviene cuando el uso pasa fuera de una mesa.",
		"nueva_app.wizard.help.option.tipo_app.desktop":                       "Escritorio es una app instalada en un ordenador. Conviene para trabajo local u offline.",
		"nueva_app.wizard.help.option.datos.gestion":                          "Gestionar datos propios significa crear y guardar informacion dentro de la app. Conviene si los usuarios registran cosas nuevas.",
		"nueva_app.wizard.help.option.datos.consulta":                         "Consultar datos existentes significa leer informacion que ya vive en otro sitio. Conviene si la app es una ventana a datos externos.",
		"nueva_app.wizard.help.option.datos.sin_persistencia":                 "Sin persistencia propia significa que la app no guarda datos duraderos. Conviene para calculos, asistentes o pantallas temporales.",
		"nueva_app.wizard.help.option.deploy.local":                           "Local significa ejecutar en una maquina concreta. Conviene para pruebas o uso individual sin publicar.",
		"nueva_app.wizard.help.option.deploy.contenedor":                      "Contenedor empaqueta la app para moverla entre servidores de forma repetible. Conviene para web o API que deben desplegarse con control.",
		"nueva_app.wizard.help.option.deploy.paas":                            "PaaS es una plataforma que aloja la app y simplifica operacion. Conviene si prefieres menos mantenimiento de servidores.",
		"nueva_app.wizard.help.option.deploy.mobile_store":                    "Tiendas moviles significa publicar como app de iOS o Android. Conviene si los usuarios la instalaran en el telefono.",
		"nueva_app.wizard.help.option.deploy.desktop":                         "Paquete desktop significa instalador o ejecutable de ordenador. Conviene para uso de oficina o offline.",
		"nueva_app.wizard.help.option.deploy.serverless":                      "Serverless ejecuta partes de la app solo cuando hace falta. Conviene para tareas eventuales o carga irregular.",
		"nueva_app.wizard.help.option.deploy.kubernetes":                      "Kubernetes coordina muchos contenedores en servidores. Conviene solo si ya tienes operacion y escala que lo justifiquen.",
		"nueva_app.wizard.help.option.uso.compartir":                          "Compartida con usuarios significa que varias personas entran a la misma app. Conviene si hay colaboracion, invitaciones o permisos.",
		"nueva_app.wizard.help.option.uso.personal":                           "Personal significa que la app es para una sola persona o uso privado. Conviene si no hay cuentas ni colaboracion.",
		"nueva_app.wizard.help.option.plataformas.ambas":                      "Movil y PC cubre telefono y ordenador. Conviene cuando el trabajo empieza en oficina y sigue fuera.",
		"nueva_app.wizard.help.option.plataformas.pc":                         "PC web prioriza pantallas grandes o navegador de ordenador. Conviene para tareas largas o administrativas.",
		"nueva_app.wizard.help.option.plataformas.movil":                      "Movil prioriza telefono. Conviene para consultas rapidas o trabajo en movimiento.",
		"nueva_app.wizard.help.option.integracion.calendar.enterprise":        "Agenda empresarial configurable conecta calendarios sin fijar proveedor. Conviene si aun no sabes si sera Google, Microsoft u otro.",
		"nueva_app.wizard.help.option.integracion.calendar.google":            "Google Calendar o Workspace conecta con calendarios de Google. Conviene si tus citas ya viven alli.",
		"nueva_app.wizard.help.option.integracion.calendar.microsoft":         "Microsoft 365 conecta con calendarios de Microsoft. Conviene si tu empresa usa Outlook o Teams.",
		"nueva_app.wizard.help.option.integracion.calendar.caldav":            "CalDAV es una forma comun de hablar con calendarios compatibles. Conviene si tu servidor de calendario no es Google ni Microsoft.",
		"nueva_app.wizard.help.option.integracion.ecommerce.capability":       "Catalogo, carrito y pagos agrupa lo necesario para vender. Conviene si el usuario elige productos y paga.",
		"nueva_app.wizard.help.option.integracion.ecommerce.payments_sync":    "Pagos y stock sincronizados mantiene cobros y existencias alineados. Conviene si vender cambia el inventario.",
		"nueva_app.wizard.help.option.integracion.domain.capability":          "Capacidad principal del dominio representa la funcion central del negocio. Conviene cuando quieres construir esa funcion dentro de la app.",
		"nueva_app.wizard.help.option.integracion.domain.sync":                "Sincronizar datos del dominio copia o actualiza datos con otra herramienta. Conviene si ya trabajas con un sistema externo.",
		"nueva_app.wizard.help.option.integracion.deferred":                   "Decidir mas tarde deja la integracion para otra fase. Conviene si todavia no sabes proveedor, datos o permisos.",
		"nueva_app.wizard.help.option.integracion.maps.public_sources":        "Mapas con fuentes publicas usa datos cartograficos accesibles sin atarse a un proveedor privado. Conviene para ubicaciones generales.",
		"nueva_app.wizard.help.option.dominio.flujo_diario":                   "Describir el dia normal cuenta los pasos reales del usuario. Conviene cuando el nombre del dominio no basta.",
		"nueva_app.wizard.help.option.dominio.integraciones":                  "Describir herramientas o datos a conectar enumera sistemas externos. Conviene cuando el valor esta en unir piezas existentes.",
		"nueva_app.wizard.help.option.storage.relacional":                     "Relacional guarda datos ordenados en tablas relacionadas. Conviene para clientes, pedidos, citas o permisos.",
		"nueva_app.wizard.help.option.storage.documental":                     "Documental guarda registros flexibles con formas distintas. Conviene para fichas variables, notas o configuraciones cambiantes.",
		"nueva_app.wizard.help.option.storage.mixta":                          "Mixta combina formas de guardado. Conviene cuando hay datos ordenados y documentos flexibles a la vez.",
		"nueva_app.wizard.help.option.storage.sin_preferencia":                "Sin preferencia deja que el diseno elija el guardado. Conviene si te importa el resultado pero no la tecnologia.",
		"nueva_app.wizard.help.option.integracion.gobierno.simple":            "Auth simple y criticidad media significa acceso controlado sin asumir alto riesgo. Conviene para empezar sin secretos complejos.",
		"nueva_app.wizard.help.option.integracion.gobierno.publica":           "Publica y criticidad baja significa que la conexion lee datos abiertos o poco sensibles. Conviene si no hay cuentas ni informacion privada.",
		"nueva_app.wizard.help.option.integracion.gobierno.oauth_alta":        "OAuth con criticidad alta significa que el usuario autoriza acceso protegido y el fallo importa. Conviene para cuentas, pagos o datos sensibles.",
		"nueva_app.wizard.help.option.mobile.ios_android":                     "iOS y Android cubre los dos sistemas moviles principales. Conviene si no controlas que telefono tendra cada usuario.",
		"nueva_app.wizard.help.option.mobile.ios":                             "Solo iOS cubre iPhone y iPad. Conviene si todos los usuarios usan dispositivos Apple.",
		"nueva_app.wizard.help.option.mobile.android":                         "Solo Android cubre telefonos y tabletas Android. Conviene si el parque de dispositivos es Android.",
		"nueva_app.wizard.help.option.mobile.web_responsive":                  "Web responsive es una web que se adapta al movil. Conviene si quieres evitar instalar una app nativa al principio.",
		"nueva_app.wizard.help.option.usuarios.equipo":                        "Equipo pequeno significa pocos usuarios con roles simples. Conviene para empezar con permisos claros sin complejidad.",
		"nueva_app.wizard.help.option.usuarios.internos":                      "Profesionales internos son usuarios de la organizacion. Conviene si la app es herramienta de trabajo interna.",
		"nueva_app.wizard.help.option.usuarios.clientes":                      "Clientes invitados son personas externas con acceso limitado. Conviene si usuarios de fuera consultan o aportan informacion.",
		"nueva_app.wizard.default.architecture.help":                          "Arquitectura hexagonal separa reglas de negocio de pantallas, base de datos y servicios externos. Conviene para poder cambiar adaptadores sin rehacer el nucleo.",
		"nueva_app.wizard.default.i18n.help":                                  "i18n por catalogo guarda textos por idioma en una fuente comun. Conviene para traducir sin buscar frases por todo el codigo.",
		"nueva_app.wizard.default.tests.help":                                 "Tests unitarios y de arquitectura comprueban piezas pequenas y reglas de estructura. Conviene para detectar roturas antes de entregar.",
		"nueva_app.wizard.default.linters.help":                               "Linters y formato revisan estilo y errores mecanicos. Conviene para que la verificacion sea repetible.",
		"nueva_app.wizard.default.errors.help":                                "Errores tipados usan codigos publicos estables. Conviene para que UI y operadores entiendan fallos sin leer detalles internos.",
		"nueva_app.wizard.default.observability.help":                         "Logging estructurado registra hechos operativos con campos claros. Conviene para investigar problemas sin depender de frases sueltas.",
		"nueva_app.wizard.default.persistence.help":                           "Migraciones, backups y restore probado hacen recuperables los datos. Conviene definirlos aunque el motor exacto se elija despues.",
		"nueva_app.wizard.default.resilience.help":                            "Timeouts, reintentos, paginacion y limites evitan que una app se bloquee o consuma recursos sin control.",
		"nueva_app.wizard.default.docs.help":                                  "Documentacion de handoff deja estructura, arbol de ficheros y stack tecnico. Conviene para que otro equipo pueda continuar.",
		"nueva_app.wizard.rich.explain_all":                                   "Que significa todo esto?",
		"nueva_app.wizard.rich.full_glossary":                                 "Ver glosario completo",
		"nueva_app.wizard.example.option.tipo_app.api":                        "Ejemplo: una app de reservas puede ofrecer una puerta para que la web y el movil consulten los mismos horarios.",
		"nueva_app.wizard.example.option.deploy.contenedor":                   "Ejemplo: como una caja cerrada con la app y sus piezas, lista para abrirse igual en varios servidores.",
		"nueva_app.wizard.example.option.deploy.paas":                         "Ejemplo: como alquilar un local con luz y mantenimiento incluidos en vez de construir el edificio.",
		"nueva_app.wizard.example.option.deploy.serverless":                   "Ejemplo: como encender una luz solo cuando alguien entra en la habitacion y apagarla despues.",
		"nueva_app.wizard.example.option.deploy.kubernetes":                   "Ejemplo: como un encargado que reparte muchas cajas de la app entre varios servidores y las sustituye si fallan.",
		"nueva_app.wizard.example.option.integracion.calendar.caldav":         "Ejemplo: como usar un enchufe comun para calendarios compatibles aunque sean de marcas distintas.",
		"nueva_app.wizard.example.option.integracion.ecommerce.payments_sync": "Ejemplo: cuando alguien compra la ultima entrada, el cobro y el contador de entradas bajan a la vez.",
		"nueva_app.wizard.example.option.integracion.maps.public_sources":     "Ejemplo: usar un mapa publico para ubicar centros sin guardar claves privadas al principio.",
		"nueva_app.wizard.example.option.storage.relacional":                  "Ejemplo: como hojas de calculo conectadas, una de clientes y otra de pedidos.",
		"nueva_app.wizard.example.option.storage.documental":                  "Ejemplo: como carpetas con fichas que no siempre tienen los mismos apartados.",
		"nueva_app.wizard.example.option.storage.mixta":                       "Ejemplo: clientes en tablas y contratos como documentos adjuntos.",
		"nueva_app.wizard.example.option.integracion.gobierno.simple":         "Ejemplo: una llave sencilla para una conexion interna de bajo riesgo.",
		"nueva_app.wizard.example.option.integracion.gobierno.oauth_alta":     "Ejemplo: el usuario pulsa Autorizar para que la app acceda a su cuenta sin entregar su contrasena.",
		"nueva_app.wizard.default.architecture.example":                       "Ejemplo: cambiar de base de datos sin cambiar las reglas de reservas.",
		"nueva_app.wizard.default.i18n.example":                               "Ejemplo: traducir Guardar a Save cambiando el catalogo, no cada pantalla.",
		"nueva_app.wizard.default.tests.example":                              "Ejemplo: una prueba falla si reservar una cita deja de guardar la confirmacion.",
		"nueva_app.wizard.default.linters.example":                            "Ejemplo: el verificador avisa si un fichero no tiene el formato esperado.",
		"nueva_app.wizard.default.errors.example":                             "Ejemplo: mostrar formulario_incompleto en vez de una traza interna.",
		"nueva_app.wizard.default.observability.example":                      "Ejemplo: registrar usuario, accion y resultado en campos separados.",
		"nueva_app.wizard.default.persistence.example":                        "Ejemplo: restaurar una copia en un entorno de prueba antes de confiar en el backup.",
		"nueva_app.wizard.default.resilience.example":                         "Ejemplo: cortar una llamada externa lenta y devolver una respuesta controlada.",
		"nueva_app.wizard.default.docs.example":                               "Ejemplo: dejar una guia con comandos de prueba y mapa de carpetas.",
	}
}

func nuevaAppWizardHelpI18nEnglishV0() map[string]string {
	out := map[string]string{
		"nueva_app.wizard.question.wizard-q-locale.help":                   "Asks for the main language of the app. Choose the language real users will use first. Example: if your team works in English, the first texts and docs start in English.",
		"nueva_app.wizard.question.wizard-q-nombre.help":                   "Asks for the public or internal app name. It is useful early because docs and tasks can use one stable reference. Example: a scheduling app can be called Clinic Slots.",
		"nueva_app.wizard.question.wizard-q-objetivo.help":                 "Asks what concrete result the app must achieve. It is best answered with something that can be checked later. Example: book an appointment and receive confirmation.",
		"nueva_app.wizard.question.wizard-q-tipo-app.help":                 "Asks for the main way people will use the app. Choose the surface where most of the value will happen. Example: web when people will use a browser.",
		"nueva_app.wizard.question.wizard-q-datos.help":                    "Asks whether the app creates, stores, or only reads information. This avoids building storage that is not needed. Example: a simple calculator may store nothing.",
		"nueva_app.wizard.question.wizard-q-deploy.help":                   "Asks where the first version will run. Choose the target closest to real use. Example: local to test on one computer before publishing.",
		"nueva_app.wizard.question.wizard-r1-uso-personal-compartido.help": "Asks whether one person or several people will use it. Shared use often needs accounts and permissions. Example: a personal calendar versus a reception calendar.",
		"nueva_app.wizard.question.wizard-r2-plataformas.help":             "Asks which devices will be used. Choose the devices users normally have. Example: mobile and desktop for a calendar checked in the office and outside.",
		"nueva_app.wizard.question.wizard-r3-integracion-agenda.help":      "Asks whether the app must connect to external calendars. Use it when appointments already live in another tool. Example: read free slots from a company calendar.",
		"nueva_app.wizard.question.wizard-r3-integracion-tienda.help":      "Asks whether there will be a catalog, cart, or charges. Use it when the flow includes selling or collecting money. Example: book a class and pay for it.",
		"nueva_app.wizard.question.wizard-r3-integracion-mapa.help":        "Asks whether location is part of the workflow. Use it for addresses, routes, or nearby search. Example: show nearby centers.",
		"nueva_app.wizard.question.wizard-r3-integracion-inventario.help":  "Asks whether stock goes up and down. Use it when available quantity matters. Example: track supplies in a storeroom.",
		"nueva_app.wizard.question.wizard-r3-integracion-notas.help":       "Asks whether the app will handle text, documents, or internal knowledge. Use it when people write and consult content. Example: a procedures wiki.",
		"nueva_app.wizard.question.wizard-r3-integracion-tareas.help":      "Asks whether there is work with owners and states. Use it when pending work must be organized. Example: a team task board.",
		"nueva_app.wizard.question.wizard-r3-integracion-finanzas.help":    "Asks whether expenses, budgets, or statements are handled. Use it when money needs tracking. Example: record monthly expenses.",
		"nueva_app.wizard.question.wizard-r3-integracion-crm.help":         "Asks whether customers or contacts are managed. Use it for sales or support follow-up. Example: see pending calls for each customer.",
		"nueva_app.wizard.question.wizard-r3-integracion-reservas.help":    "Asks whether slots, appointments, or bookings are reserved. Use it when availability is limited. Example: request a clinic appointment.",
		"nueva_app.wizard.question.wizard-r3-integracion-salud.help":       "Asks whether health, habit, or activity data exists. Use it when privacy matters. Example: record daily measurements.",
		"nueva_app.wizard.question.wizard-r3-integracion-educacion.help":   "Asks whether there are courses, students, or learning. Use it when progress is tracked. Example: mark lessons completed.",
		"nueva_app.wizard.question.wizard-r3-integracion-comunidad.help":   "Asks whether users will participate in discussions. Use it when moderation is needed. Example: a neighborhood forum.",
		"nueva_app.wizard.question.wizard-r3-integracion-iot.help":         "Asks whether the app talks to sensors or devices. Use it when readings arrive automatically. Example: view temperature sent by a sensor.",
		"nueva_app.wizard.question.wizard-r3-integracion-media.help":       "Asks whether photos, videos, or visual files are uploaded or processed. Use it when heavy content is central. Example: a gallery of finished work.",
		"nueva_app.wizard.question.wizard-r3-integracion-facturacion.help": "Asks whether invoices or legal documents will be issued. Use it when tax rules apply. Example: create a numbered PDF invoice.",
		"nueva_app.wizard.question.wizard-r3-dominio-abierto.help":         "Asks for everyday work when the wizard does not recognize the domain. Describe a normal day to avoid assumptions. Example: open a record, note progress, and close a task.",
		"nueva_app.wizard.question.wizard-r4-storage-db.help":              "Asks which kind of storage fits the data. Choose based on whether information is structured, flexible, or mixed. Example: customer tables plus attached documents.",
		"nueva_app.wizard.question.wizard-r5-integracion-gobierno.help":    "Asks how a connection to another tool is controlled. Answer based on risk and protected access. Example: a public API is not the same as customer data.",
		"nueva_app.wizard.question.wizard-r6-movil-plataformas.help":       "Asks which mobile systems must be covered. Choose where real users are. Example: iOS and Android when the team uses both.",
		"nueva_app.wizard.question.wizard-r7-deploy-compatible.help":       "Asks for another target when the selected one does not fit. Correcting it allows build and test. Example: a mobile app needs a mobile package, not just desktop.",
		"nueva_app.wizard.question.wizard-r8-usuarios-compartido.help":     "Asks which group will use the shared app. This shapes permissions and screens. Example: a small team with an admin and readers.",
		"nueva_app.wizard.rich.explain_all":                                "Explain everything",
		"nueva_app.wizard.rich.full_glossary":                              "Open full glossary",
	}
	for key, value := range nuevaAppWizardHelpOptionEnglishV0() {
		out[key] = value
	}
	for key, value := range nuevaAppWizardHelpDefaultEnglishV0() {
		out[key] = value
	}
	for key, value := range nuevaAppWizardExampleEnglishV0() {
		out[key] = value
	}
	return out
}

func nuevaAppWizardHelpOptionEnglishV0() map[string]string {
	return map[string]string{
		"nueva_app.wizard.help.option.locale.es":                           "Spanish means the first texts are in Spanish. Use it when most of the team works in Spanish.",
		"nueva_app.wizard.help.option.locale.en":                           "English means the first texts are in English. Use it when users or the main documentation will be in English.",
		"nueva_app.wizard.help.option.nombre.inferido":                     "Inferred name uses the goal to suggest a name. Use it when you do not have a brand yet.",
		"nueva_app.wizard.help.option.nombre.generico":                     "New app is a temporary name. Use it only to start quickly and rename later.",
		"nueva_app.wizard.help.option.objetivo.describir":                  "Describe a verifiable objective means saying what must be true at the end. Use it when the expected result is still unclear.",
		"nueva_app.wizard.help.option.tipo_app.web":                        "Web is an app used from a browser. Use it when you want easy access from different computers or phones.",
		"nueva_app.wizard.help.option.tipo_app.api":                        "API is a doorway for other programs to request or send data. Use it when you do not need a main screen.",
		"nueva_app.wizard.help.option.tipo_app.mobile":                     "Mobile is an app designed for phones. Use it when work happens away from a desk.",
		"nueva_app.wizard.help.option.tipo_app.desktop":                    "Desktop is an app installed on a computer. Use it for local or offline work.",
		"nueva_app.wizard.help.option.datos.gestion":                       "Manage own data means creating and saving information inside the app. Use it when users record new things.",
		"nueva_app.wizard.help.option.datos.consulta":                      "Read existing data means information already lives somewhere else. Use it when the app is a window into external data.",
		"nueva_app.wizard.help.option.datos.sin_persistencia":              "No own persistence means the app does not keep long-lived data. Use it for calculators, helpers, or temporary screens.",
		"nueva_app.wizard.help.option.deploy.local":                        "Local means running on one specific machine. Use it for tests or individual use without publishing.",
		"nueva_app.wizard.help.option.deploy.contenedor":                   "Container packages the app so it can move between servers repeatably. Use it for controlled web or API deploys.",
		"nueva_app.wizard.help.option.deploy.paas":                         "PaaS is a hosted platform that simplifies operations. Use it when you want less server maintenance.",
		"nueva_app.wizard.help.option.deploy.mobile_store":                 "Mobile stores means publishing as an iOS or Android app. Use it when users will install it on phones.",
		"nueva_app.wizard.help.option.deploy.desktop":                      "Desktop package means an installer or computer executable. Use it for office or offline use.",
		"nueva_app.wizard.help.option.deploy.serverless":                   "Serverless runs parts of the app only when needed. Use it for occasional tasks or uneven load.",
		"nueva_app.wizard.help.option.deploy.kubernetes":                   "Kubernetes coordinates many containers on servers. Use it only when operations and scale justify it.",
		"nueva_app.wizard.help.option.uso.compartir":                       "Shared with users means several people enter the same app. Use it for collaboration, invitations, or permissions.",
		"nueva_app.wizard.help.option.uso.personal":                        "Personal means one person or private use. Use it when there are no accounts or collaboration.",
		"nueva_app.wizard.help.option.plataformas.ambas":                   "Mobile and desktop covers phone and computer. Use it when work starts in the office and continues outside.",
		"nueva_app.wizard.help.option.plataformas.pc":                      "Desktop web prioritizes large screens or computer browsers. Use it for long or administrative tasks.",
		"nueva_app.wizard.help.option.plataformas.movil":                   "Mobile prioritizes phones. Use it for quick checks or work on the move.",
		"nueva_app.wizard.help.option.integracion.calendar.enterprise":     "Configurable company calendar connects calendars without locking a provider. Use it when Google, Microsoft, or another service is still undecided.",
		"nueva_app.wizard.help.option.integracion.calendar.google":         "Google Calendar or Workspace connects to Google calendars. Use it when your appointments already live there.",
		"nueva_app.wizard.help.option.integracion.calendar.microsoft":      "Microsoft 365 connects to Microsoft calendars. Use it when your company uses Outlook or Teams.",
		"nueva_app.wizard.help.option.integracion.calendar.caldav":         "CalDAV is a common way to talk to compatible calendars. Use it when your calendar server is not Google or Microsoft.",
		"nueva_app.wizard.help.option.integracion.ecommerce.capability":    "Catalog, cart, and payments group what is needed to sell. Use it when users choose products and pay.",
		"nueva_app.wizard.help.option.integracion.ecommerce.payments_sync": "Synchronized payments and stock keeps charges and inventory aligned. Use it when a sale changes availability.",
		"nueva_app.wizard.help.option.integracion.domain.capability":       "Main domain capability represents the core business function. Use it when you want that function built inside the app.",
		"nueva_app.wizard.help.option.integracion.domain.sync":             "Synchronize domain data copies or updates data with another tool. Use it when you already work in an external system.",
		"nueva_app.wizard.help.option.integracion.deferred":                "Decide later leaves the integration for another phase. Use it when provider, data, or permissions are unknown.",
		"nueva_app.wizard.help.option.integracion.maps.public_sources":     "Maps with public sources uses map data without locking to a private provider. Use it for general locations.",
		"nueva_app.wizard.help.option.dominio.flujo_diario":                "Describe the normal day tells the user's real steps. Use it when the domain name alone is not enough.",
		"nueva_app.wizard.help.option.dominio.integraciones":               "Describe tools or data to connect lists external systems. Use it when value comes from joining existing pieces.",
		"nueva_app.wizard.help.option.storage.relacional":                  "Relational stores ordered data in linked tables. Use it for customers, orders, appointments, or permissions.",
		"nueva_app.wizard.help.option.storage.documental":                  "Document storage keeps flexible records with different shapes. Use it for variable files, notes, or changing settings.",
		"nueva_app.wizard.help.option.storage.mixta":                       "Mixed combines storage styles. Use it when ordered data and flexible documents coexist.",
		"nueva_app.wizard.help.option.storage.sin_preferencia":             "No preference lets design choose storage. Use it when the result matters more than the technology.",
		"nueva_app.wizard.help.option.integracion.gobierno.simple":         "Simple auth and medium criticality means controlled access without assuming high risk. Use it to start without complex secrets.",
		"nueva_app.wizard.help.option.integracion.gobierno.publica":        "Public and low criticality means the connection reads open or low-risk data. Use it when there are no accounts or private information.",
		"nueva_app.wizard.help.option.integracion.gobierno.oauth_alta":     "OAuth with high criticality means the user authorizes protected access and failure matters. Use it for accounts, payments, or sensitive data.",
		"nueva_app.wizard.help.option.mobile.ios_android":                  "iOS and Android covers the two main mobile systems. Use it when you cannot control each user's phone.",
		"nueva_app.wizard.help.option.mobile.ios":                          "iOS only covers iPhone and iPad. Use it when all users have Apple devices.",
		"nueva_app.wizard.help.option.mobile.android":                      "Android only covers Android phones and tablets. Use it when the device fleet is Android.",
		"nueva_app.wizard.help.option.mobile.web_responsive":               "Responsive web is a website that adapts to mobile. Use it when you want to avoid a native install at first.",
		"nueva_app.wizard.help.option.usuarios.equipo":                     "Small team means a few users with simple roles. Use it to start with clear permissions and little complexity.",
		"nueva_app.wizard.help.option.usuarios.internos":                   "Internal professionals are users from the organization. Use it when the app is an internal work tool.",
		"nueva_app.wizard.help.option.usuarios.clientes":                   "Invited clients are external people with limited access. Use it when outside users read or provide information.",
	}
}

func nuevaAppWizardHelpDefaultEnglishV0() map[string]string {
	return map[string]string{
		"nueva_app.wizard.default.architecture.help":  "Hexagonal architecture separates business rules from screens, database, and external services. Use it so adapters can change without rewriting the core.",
		"nueva_app.wizard.default.i18n.help":          "Catalog i18n keeps text by language in one shared source. Use it to translate without hunting phrases across the code.",
		"nueva_app.wizard.default.tests.help":         "Unit and architecture tests check small pieces and structure rules. Use them to catch breakage before delivery.",
		"nueva_app.wizard.default.linters.help":       "Linters and formatting check style and mechanical errors. Use them so verification is repeatable.",
		"nueva_app.wizard.default.errors.help":        "Typed errors use stable public codes. Use them so UI and operators understand failures without internal details.",
		"nueva_app.wizard.default.observability.help": "Structured logging records operational facts with clear fields. Use it to investigate issues without relying on loose text.",
		"nueva_app.wizard.default.persistence.help":   "Migrations, backups, and tested restore make data recoverable. Use them even when the exact engine is chosen later.",
		"nueva_app.wizard.default.resilience.help":    "Timeouts, retries, pagination, and limits prevent stalls and unbounded resource use.",
		"nueva_app.wizard.default.docs.help":          "Handoff documentation records structure, file tree, and technical stack. Use it so another team can continue.",
	}
}

func nuevaAppWizardExampleEnglishV0() map[string]string {
	return map[string]string{
		"nueva_app.wizard.example.option.tipo_app.api":                        "Example: a booking app can offer one doorway so web and mobile read the same schedules.",
		"nueva_app.wizard.example.option.deploy.contenedor":                   "Example: like a sealed box with the app and its pieces, ready to open the same way on different servers.",
		"nueva_app.wizard.example.option.deploy.paas":                         "Example: like renting a place with power and maintenance included instead of building the building.",
		"nueva_app.wizard.example.option.deploy.serverless":                   "Example: like turning on a light only when someone enters the room and turning it off afterward.",
		"nueva_app.wizard.example.option.deploy.kubernetes":                   "Example: like a coordinator that spreads many app boxes across servers and replaces them if they fail.",
		"nueva_app.wizard.example.option.integracion.calendar.caldav":         "Example: like a common plug for compatible calendars from different brands.",
		"nueva_app.wizard.example.option.integracion.ecommerce.payments_sync": "Example: when someone buys the last ticket, the payment and ticket counter change together.",
		"nueva_app.wizard.example.option.integracion.maps.public_sources":     "Example: use a public map to place centers without storing private keys at first.",
		"nueva_app.wizard.example.option.storage.relacional":                  "Example: like connected spreadsheets, one for customers and one for orders.",
		"nueva_app.wizard.example.option.storage.documental":                  "Example: like folders with records that do not always have the same sections.",
		"nueva_app.wizard.example.option.storage.mixta":                       "Example: customers in tables and contracts as attached documents.",
		"nueva_app.wizard.example.option.integracion.gobierno.simple":         "Example: a simple key for a low-risk internal connection.",
		"nueva_app.wizard.example.option.integracion.gobierno.oauth_alta":     "Example: the user clicks Authorize so the app can access the account without receiving the password.",
		"nueva_app.wizard.default.architecture.example":                       "Example: change the database without changing booking rules.",
		"nueva_app.wizard.default.i18n.example":                               "Example: translate Guardar to Save by changing the catalog, not every screen.",
		"nueva_app.wizard.default.tests.example":                              "Example: a test fails if booking an appointment stops saving confirmation.",
		"nueva_app.wizard.default.linters.example":                            "Example: the checker warns when a file does not use the expected format.",
		"nueva_app.wizard.default.errors.example":                             "Example: show form_incomplete instead of an internal trace.",
		"nueva_app.wizard.default.observability.example":                      "Example: record user, action, and result in separate fields.",
		"nueva_app.wizard.default.persistence.example":                        "Example: restore a backup in a test environment before trusting it.",
		"nueva_app.wizard.default.resilience.example":                         "Example: cut off a slow external call and return a controlled response.",
		"nueva_app.wizard.default.docs.example":                               "Example: leave a guide with test commands and folder map.",
	}
}
