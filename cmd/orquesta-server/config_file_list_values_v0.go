package main

import (
	"os"
	"strings"
)

func stringSliceProjectConfigOrEnvOrDefaultV0(key string, fileValue *[]string, fallback []string) []string {
	if strings.TrimSpace(os.Getenv(key)) != "" {
		return csvEnvOrDefaultV0(key, fallback)
	}
	if fileValue == nil {
		return append([]string(nil), fallback...)
	}
	out := make([]string, 0, len(*fileValue))
	for _, item := range *fileValue {
		value := strings.TrimSpace(item)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}
