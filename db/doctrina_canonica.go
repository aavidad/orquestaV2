package db

const RutaDoctrinaCanonica = "docs/BIBLIA_APP_ORQUESTA.md"

func InstruccionDoctrinaCanonica() string {
	return "Lectura obligatoria antes de actuar: " + RutaDoctrinaCanonica + ". Es la doctrina canonica unica de Orquesta. Si hay conflicto con otros documentos, manda este archivo y el estado vivo consultado en Orquesta."
}

func ResumenDoctrinaCanonica() string {
	return "Doctrina canonica: " + RutaDoctrinaCanonica
}
