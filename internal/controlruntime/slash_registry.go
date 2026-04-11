package controlruntime

type SlashCommandSpec struct {
	Command     string
	Description string
}

var codexSlashCommands = []SlashCommandSpec{
	{Command: "/status", Description: "Muestra estado de sesion, cuenta y cuota"},
	{Command: "/help", Description: "Muestra ayuda de slash commands"},
	{Command: "/compact", Description: "Compacta historial de conversacion"},
	{Command: "/clear", Description: "Limpia conversacion visible"},
	{Command: "/model", Description: "Muestra o cambia modelo"},
	{Command: "/permissions", Description: "Muestra o cambia permisos"},
	{Command: "/config", Description: "Muestra configuracion"},
	{Command: "/memory", Description: "Muestra memoria/AGENTS"},
	{Command: "/diff", Description: "Muestra diff git"},
	{Command: "/export", Description: "Exporta conversacion"},
	{Command: "/session", Description: "Gestiona reanudacion de sesion"},
	{Command: "/version", Description: "Muestra version"},
}

var claudeSlashCommands = []SlashCommandSpec{
	{Command: "/status", Description: "Muestra estado de sesion, modelo y coste"},
	{Command: "/help", Description: "Muestra ayuda de slash commands"},
	{Command: "/cost", Description: "Muestra desglose de coste"},
	{Command: "/compact", Description: "Compacta historial de conversacion"},
	{Command: "/clear", Description: "Limpia conversacion visible"},
	{Command: "/model", Description: "Muestra o cambia modelo"},
	{Command: "/permissions", Description: "Muestra o cambia permisos"},
	{Command: "/config", Description: "Muestra configuracion"},
	{Command: "/memory", Description: "Muestra CLAUDE.md o memoria"},
	{Command: "/diff", Description: "Muestra diff git"},
	{Command: "/export", Description: "Exporta conversacion"},
	{Command: "/session", Description: "Gestiona reanudacion de sesion"},
	{Command: "/version", Description: "Muestra version"},
}

func SupportedSlashCommands(obj ObjetivoProceso) []SlashCommandSpec {
	if esRuntimeCodexLocal(obj) {
		return append([]SlashCommandSpec(nil), codexSlashCommands...)
	}
	if esRuntimeClaudeLocal(obj) {
		return append([]SlashCommandSpec(nil), claudeSlashCommands...)
	}
	return nil
}
