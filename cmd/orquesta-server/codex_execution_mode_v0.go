package main

const (
	codexExecutionModeParallelV0 = "parallel"
	codexExecutionModeSerialV0   = "serial"
)

func codexExecutionModeCapPositiveV0(mode string, value int) int {
	if value <= 0 {
		value = 1
	}
	if mode == codexExecutionModeSerialV0 && value > 1 {
		return 1
	}
	return value
}

func codexExecutionModeCapIntEnvOrDefaultV0(mode string, key string, fallback int) int {
	return codexExecutionModeCapPositiveV0(mode, intEnvOrDefaultV0(key, fallback))
}
