package cmd

import "strings"

type webWizardHelpItem struct {
	Option      string
	Description string
}

func webWizardHelp(lang, key string) []webWizardHelpItem {
	if strings.EqualFold(strings.TrimSpace(lang), "en") || strings.HasPrefix(strings.ToLower(strings.TrimSpace(lang)), "en-") {
		return webWizardHelpEN(key)
	}
	return webWizardHelpES(key)
}

func webWizardHelpES(key string) []webWizardHelpItem {
	switch key {
	case "arquitectura":
		return []webWizardHelpItem{
			{Option: "Hexagonal", Description: "La lógica de negocio vive en puertos y casos de uso, y la infraestructura queda detrás de adaptadores. Es el default más sano para apps de producto mantenibles."},
			{Option: "Clean architecture", Description: "Parecida a hexagonal, pero más explícita en capas de aplicación, dominio e infraestructura. Útil si quieres separar muy bien reglas de negocio y orquestación."},
			{Option: "Arquitectura por capas", Description: "Más simple de arrancar, pero tiende a mezclar dependencias con el tiempo. Solo conviene si el alcance es pequeño o el equipo quiere mínima complejidad inicial."},
			{Option: "Event-driven", Description: "La app se organiza alrededor de eventos, colas y reacciones asíncronas. Útil si el sistema tendrá integraciones, procesos desacoplados o mucho trabajo en background."},
		}
	case "nucleo_caps":
		return []webWizardHelpItem{
			{Option: "Servicio residente", Description: "Indica que habrá un proceso principal vivo coordinando estado caliente, workers o continuidad operativa. No significa que toda la lógica deba quedarse siempre en memoria."},
			{Option: "Background jobs / queue / scheduler", Description: "Actívalo si habrá trabajo fuera del request path: colas, procesos diferidos, reintentos o cron. Cambia arquitectura, observabilidad y despliegue."},
			{Option: "Multi-tenant / feature flags", Description: "Multi-tenant implica aislamiento de datos, configuración y branding por cliente. Feature flags implica despliegue progresivo y control funcional por entorno o cuenta."},
			{Option: "Offline, import/export, reporting", Description: "Estas capacidades encarecen persistencia, sincronización y UX. Márcalas solo si son de producto, no por si acaso."},
		}
	case "frontend_stack":
		return []webWizardHelpItem{
			{Option: "React", Description: "Buena elección para interfaces ricas, theming serio, formularios complejos, dashboards, white-label o mucho estado cliente."},
			{Option: "Vue", Description: "Alternativa sólida para UI moderna con menor fricción inicial. Útil si prefieres una curva algo más suave sin perder capacidad."},
			{Option: "SvelteKit", Description: "Buena opción si quieres SSR/SPA híbrida con poco boilerplate y buen rendimiento. Menos estándar corporativo que React."},
			{Option: "Next.js", Description: "Adecuado si quieres ecosistema React con SSR, rutas, server actions y despliegue web moderno. Más framework que librería."},
			{Option: "Astro", Description: "Encaja bien en webs más de contenido o con islas interactivas puntuales, no tanto en backoffices muy dinámicos."},
			{Option: "HTMX", Description: "Server-first con interactividad incremental. Muy útil si quieres poca complejidad frontend y mantener HTML como contrato principal."},
			{Option: "Server-rendered / plantillas", Description: "La opción más simple si la web es clásica, con poca lógica cliente y fuerte control desde backend."},
			{Option: "Angular", Description: "Tiene sentido en contextos corporativos o equipos ya estandarizados en Angular. Más pesado de base, pero muy opinado."},
			{Option: "Otro", Description: "Solo si tienes una restricción real de producto o de ecosistema que no encaja en las opciones anteriores."},
		}
	case "api_style":
		return []webWizardHelpItem{
			{Option: "REST", Description: "Default razonable para la mayoría de apps internas o públicas con recursos y flujos HTTP claros."},
			{Option: "GraphQL", Description: "Útil si el cliente necesita mucha flexibilidad de consulta o agregación de datos heterogéneos."},
			{Option: "gRPC", Description: "Mejor para servicios internos de alto rendimiento o contratos fuertemente tipados entre servicios."},
			{Option: "Async", Description: "Usa eventos o mensajería cuando el contrato principal no sea request/response sino procesamiento desacoplado."},
			{Option: "Mixed", Description: "Combina estilos cuando hay varias superficies distintas, pero exige disciplina para no crear un sistema incoherente."},
		}
	case "auth_mode":
		return []webWizardHelpItem{
			{Option: "Sin auth", Description: "Solo para apps públicas o internas sin login ni control de acceso."},
			{Option: "Sesiones", Description: "Buena opción para web clásica con frontend y backend propios bajo el mismo dominio."},
			{Option: "JWT", Description: "Útil para APIs desacopladas consumidas por frontends o clientes externos."},
			{Option: "OAuth / OIDC", Description: "Adecuado para móvil, terceros o flujos delegados con proveedor estándar."},
			{Option: "SSO corporativo", Description: "La opción correcta cuando la identidad la gobierna la organización, por ejemplo AD, LDAP o Entra ID."},
			{Option: "API keys", Description: "Encaja en integraciones máquina a máquina, CLI o automatizaciones simples."},
		}
	case "identity_provider":
		return []webWizardHelpItem{
			{Option: "Active Directory", Description: "Úsalo si la app debe integrarse con identidades corporativas Windows/AD y políticas de empresa."},
			{Option: "LDAP", Description: "Válido para directorios corporativos genéricos o heredados."},
			{Option: "Microsoft Entra ID", Description: "Encaja si el cliente está en ecosistema Microsoft 365/Azure con identidad moderna."},
			{Option: "Keycloak / Auth0 / Google", Description: "Opciones de IAM ya gestionadas o SaaS. Cambian costes, lock-in y operación."},
			{Option: "Otro / ninguno", Description: "Úsalo si no hay proveedor externo o si la integración es específica del cliente."},
		}
	case "interface_caps":
		return []webWizardHelpItem{
			{Option: "RBAC", Description: "Actívalo cuando haya roles y permisos distintos por perfil o tenant. No lo dejes implícito en código."},
			{Option: "Rate limiting", Description: "Necesario si hay APIs públicas, abuso potencial o cuotas por cliente."},
			{Option: "Temas y branding", Description: "Marca soporte de temas si la UI debe poder cambiar apariencia sin tocar lógica. Marca branding si necesitas white-label o perfiles visuales por cliente."},
			{Option: "Idiomas adicionales", Description: "Sirve tanto para idiomas iniciales como para añadir idiomas después en un proyecto vivo, sembrando backlog incremental de i18n, docs y QA."},
			{Option: "Notificaciones / webhooks / uploads", Description: "Capacidades de interfaz que impactan seguridad, contratos y observabilidad; no son simples checks decorativos."},
		}
	case "db_engine":
		return []webWizardHelpItem{
			{Option: "Sin base de datos", Description: "Solo si la app no guarda datos de negocio persistentes o delega totalmente la persistencia fuera."},
			{Option: "SQLite", Description: "Muy buena para desktop, móvil, CLI o local-first con poca concurrencia y despliegue simple."},
			{Option: "PostgreSQL", Description: "Default fuerte para web/API, multiusuario, reporting, RBAC, jobs o despliegue serio."},
			{Option: "MySQL / MongoDB / Redis / other", Description: "Úsalos solo si hay un motivo real de ecosistema, modelo de datos o infraestructura, no por preferencia superficial."},
		}
	case "data_caps":
		return []webWizardHelpItem{
			{Option: "Caché / search / object storage", Description: "Son subsistemas propios. Márcalos si el producto de verdad los necesita y estás dispuesto a operarlos."},
			{Option: "Import/export / audit trail", Description: "Importación, exportación y auditoría cambian contratos, persistencia y responsabilidades legales."},
			{Option: "Backups y disaster recovery", Description: "Si la app guarda datos relevantes, esto deja de ser opcional en producción real."},
		}
	case "deployment_target":
		return []webWizardHelpItem{
			{Option: "Docker", Description: "Buen default para servicios backend o web/API con operación estándar."},
			{Option: "Kubernetes", Description: "Solo si ya existe ese modelo operativo o sabes que vas a necesitarlo. Introduce mucha complejidad de plataforma."},
			{Option: "Serverless", Description: "Útil para frontends o servicios ligeros sin núcleo residente fuerte."},
			{Option: "VM / bare metal", Description: "Encaja mejor en ejecutables, desktop o despliegues simples sin contenedores."},
			{Option: "Edge", Description: "Para runtimes cercanos al dispositivo, IoT o escenarios embebidos."},
		}
	case "artifact_type":
		return []webWizardHelpItem{
			{Option: "Executable / service", Description: "Default para backend, workers o utilidades desplegadas como binario o proceso residente."},
			{Option: "Docker image", Description: "Úsalo si el artefacto principal se entregará como contenedor."},
			{Option: "Library / SDK", Description: "Solo cuando el producto principal sea reutilizable por otros sistemas."},
			{Option: "Desktop installer / mobile bundle / firmware / static site", Description: "Elige el artefacto que realmente consume el usuario o el entorno final, no uno intermedio."},
		}
	case "testing_level":
		return []webWizardHelpItem{
			{Option: "Base", Description: "Tests unitarios e integración mínima para tener cobertura profesional sin frenar demasiado el arranque."},
			{Option: "Strict", Description: "Mayor densidad de tests, más puertas de calidad y menos tolerancia a deuda."},
			{Option: "TDD", Description: "Úsalo si el equipo quiere construir guiado por tests desde el primer día y acepta más disciplina inicial."},
		}
	case "observability_level":
		return []webWizardHelpItem{
			{Option: "Basic", Description: "Logs y señales mínimas para arrancar."},
			{Option: "Standard", Description: "Buen punto medio para producción seria: logs estructurados, métricas y algo de trazabilidad."},
			{Option: "Strict", Description: "Observabilidad profunda y operativa, útil en sistemas críticos o con requisitos de soporte fuertes."},
		}
	case "delivery_caps":
		return []webWizardHelpItem{
			{Option: "CI/CD / Terraform / Monitoring", Description: "Son decisiones de delivery y operación. No marques Terraform o monitoring si no van a mantenerse de verdad."},
			{Option: "Docker / Kubernetes", Description: "Van juntas muchas veces, pero no siempre. Docker puede ser el artefacto y Kubernetes no estar justificado."},
			{Option: "Backups / recovery", Description: "Cuando hay datos valiosos o compromisos de servicio, deben tratarse como parte del producto, no como post-it."},
		}
	}
	return nil
}

func webWizardHelpEN(key string) []webWizardHelpItem {
	switch key {
	case "arquitectura":
		return []webWizardHelpItem{
			{Option: "Hexagonal", Description: "Business logic lives behind ports and use cases while infrastructure stays in adapters. This is the healthiest default for maintainable product systems."},
			{Option: "Clean architecture", Description: "Similar to hexagonal, but more explicit about application, domain and infrastructure boundaries. Useful when you want sharper separation."},
			{Option: "Layered architecture", Description: "Simpler to start, but easier to erode over time. Use it only for smaller scope or intentionally simpler systems."},
			{Option: "Event-driven", Description: "The system is organized around events, queues and async reactions. Useful for integrations, decoupled processing or heavy background work."},
		}
	case "nucleo_caps":
		return []webWizardHelpItem{
			{Option: "Resident core", Description: "Use this when you truly need a live process coordinating hot state, workers or operational continuity. It does not mean everything should stay in memory."},
			{Option: "Background jobs / queue / scheduler", Description: "Enable these only if work really happens outside the request path. They change architecture, observability and operations."},
			{Option: "Multi-tenant / feature flags", Description: "Multi-tenant affects isolation, data, branding and configuration. Feature flags affect rollout and operational control."},
			{Option: "Offline, import/export, reporting", Description: "These capabilities are expensive in persistence, synchronization and UX. Mark them when they are product needs, not just nice-to-haves."},
		}
	case "frontend_stack":
		return []webWizardHelpItem{
			{Option: "React", Description: "Strong choice for richer product UIs, serious theming, complex forms, dashboards, white-labeling and heavier client state."},
			{Option: "Vue", Description: "Solid modern UI choice with a gentler adoption curve while still supporting complex apps."},
			{Option: "SvelteKit", Description: "Good fit when you want SSR/SPA hybrid behavior with less boilerplate and strong performance."},
			{Option: "Next.js", Description: "Good if you want React plus SSR, routing and broader framework conventions."},
			{Option: "Astro", Description: "Best for content-heavy or island-style websites, not usually the first pick for rich backoffice apps."},
			{Option: "HTMX", Description: "Server-first interactivity with low frontend complexity. Great when HTML remains the main contract."},
			{Option: "Server-rendered / templates", Description: "Simplest choice for classic web apps with limited client logic and strong backend control."},
			{Option: "Angular", Description: "Makes sense in corporate environments already standardized on Angular. Heavier by default, but very opinionated."},
			{Option: "Other", Description: "Use this only when you have a concrete ecosystem or product reason outside the standard options."},
		}
	case "api_style":
		return []webWizardHelpItem{
			{Option: "REST", Description: "Reasonable default for most public or internal apps with clear HTTP resources and workflows."},
			{Option: "GraphQL", Description: "Useful when clients need flexible querying over heterogeneous data."},
			{Option: "gRPC", Description: "Best for internal service-to-service contracts with high performance or strict typing."},
			{Option: "Async", Description: "Use events or messaging when the main contract is not request/response but decoupled processing."},
			{Option: "Mixed", Description: "Combine styles only when the product truly has different surfaces; otherwise it becomes incoherent quickly."},
		}
	case "auth_mode":
		return []webWizardHelpItem{
			{Option: "No auth", Description: "Only for public or internal apps without login or access control."},
			{Option: "Session", Description: "Good default for classic web apps with first-party frontend and backend under one control plane."},
			{Option: "JWT", Description: "Useful for decoupled APIs consumed by separate frontends or external clients."},
			{Option: "OAuth / OIDC", Description: "Appropriate for mobile, delegated clients or provider-backed standards-based flows."},
			{Option: "Enterprise SSO", Description: "The right choice when identity is governed by the organization, such as AD, LDAP or Entra ID."},
			{Option: "API keys", Description: "Fits machine-to-machine integrations, CLIs and simpler automation."},
		}
	case "identity_provider":
		return []webWizardHelpItem{
			{Option: "Active Directory", Description: "Use it when the app must integrate with corporate Windows/AD identity and enterprise policy."},
			{Option: "LDAP", Description: "Good for generic or legacy corporate directory integration."},
			{Option: "Microsoft Entra ID", Description: "Fits Microsoft 365 / Azure centered identity landscapes."},
			{Option: "Keycloak / Auth0 / Google", Description: "Managed IAM choices with different operational and lock-in tradeoffs."},
			{Option: "Other / none", Description: "Use when there is no external provider or when identity is specific to the customer environment."},
		}
	case "interface_caps":
		return []webWizardHelpItem{
			{Option: "RBAC", Description: "Enable it when roles and permissions differ by profile or tenant. Do not leave authorization implicit in code."},
			{Option: "Rate limiting", Description: "Needed for public APIs, abuse control or customer quotas."},
			{Option: "Themes and branding", Description: "Theme support means the UI can change appearance without touching business logic. Branding means white-label or customer-specific visual profiles."},
			{Option: "Additional languages", Description: "Use it both for initial languages and for adding languages later in a live project, seeding incremental i18n, docs and QA work."},
			{Option: "Notifications / webhooks / uploads", Description: "These are not cosmetic toggles; they affect security, contracts and operations."},
		}
	case "db_engine":
		return []webWizardHelpItem{
			{Option: "No database", Description: "Only if the app has no durable business persistence or fully delegates it elsewhere."},
			{Option: "SQLite", Description: "Very good for desktop, mobile, CLI or local-first products with limited concurrency and simple deployment."},
			{Option: "PostgreSQL", Description: "Strong default for web/API, multi-user, RBAC, reporting, jobs and serious production deployment."},
			{Option: "MySQL / MongoDB / Redis / other", Description: "Choose them only with a real ecosystem, model or operational reason, not superficial preference."},
		}
	case "data_caps":
		return []webWizardHelpItem{
			{Option: "Cache / search / object storage", Description: "These are real subsystems. Enable them when the product truly needs them and you are willing to operate them."},
			{Option: "Import/export / audit trail", Description: "Import, export and audit change contracts, persistence and often legal responsibilities."},
			{Option: "Backups and disaster recovery", Description: "If the product stores valuable data, these stop being optional."},
		}
	case "deployment_target":
		return []webWizardHelpItem{
			{Option: "Docker", Description: "Good default for backend services and web/API products."},
			{Option: "Kubernetes", Description: "Choose it only when the operational model already justifies it. It introduces major platform complexity."},
			{Option: "Serverless", Description: "Good for lightweight frontends or services without a strong resident core."},
			{Option: "VM / bare metal", Description: "Better fit for executables, desktop delivery or simpler non-container deployments."},
			{Option: "Edge", Description: "For device-adjacent runtimes, IoT or embedded scenarios."},
		}
	case "artifact_type":
		return []webWizardHelpItem{
			{Option: "Executable / service", Description: "Default for backends, workers and utilities shipped as a binary or live process."},
			{Option: "Docker image", Description: "Use it when the main shipped artifact is a container."},
			{Option: "Library / SDK", Description: "Only when the product itself is meant to be reused by other systems."},
			{Option: "Desktop installer / mobile bundle / firmware / static site", Description: "Choose the artifact actually consumed by users or the target environment, not an internal intermediate."},
		}
	case "testing_level":
		return []webWizardHelpItem{
			{Option: "Base", Description: "Unit tests plus minimum integration coverage for a professional start without over-constraining delivery."},
			{Option: "Strict", Description: "Higher test density and harder quality gates."},
			{Option: "TDD", Description: "Use when the team wants test-first discipline from day one and accepts the extra upfront rigor."},
		}
	case "observability_level":
		return []webWizardHelpItem{
			{Option: "Basic", Description: "Minimum logs and operational signals."},
			{Option: "Standard", Description: "A good production middle ground: structured logs, metrics and some tracing."},
			{Option: "Strict", Description: "Deep observability for critical systems or strong support requirements."},
		}
	case "delivery_caps":
		return []webWizardHelpItem{
			{Option: "CI/CD / Terraform / Monitoring", Description: "These are delivery and operations choices. Do not enable them unless they will actually be maintained."},
			{Option: "Docker / Kubernetes", Description: "Often related, but not equivalent. Docker may be right while Kubernetes is unjustified."},
			{Option: "Backups / recovery", Description: "If the product carries valuable data or service commitments, they are part of the product contract."},
		}
	}
	return nil
}
