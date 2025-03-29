package procutil

import (
	"os"
	"strings"
)

type EnvVar string

func LookupBoolEnv(name EnvVar, defaultValue bool) bool {
	if val, ok := os.LookupEnv(string(name)); ok {
		switch strings.ToLower(val) {
		case "true", "1":
			return true
		case "false", "0":
			return false
		}
	}
	return defaultValue
}

func LookupEnv(name EnvVar) (string, bool) {
	return os.LookupEnv(string(name))
}
