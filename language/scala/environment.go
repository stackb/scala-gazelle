package scala

import (
	"os"

	"github.com/stackb/scala-gazelle/pkg/procutil"
)

const (
	SCALA_GAZELLE_LOG_FILE      = procutil.EnvVar("SCALA_GAZELLE_LOG_FILE")
	SCALA_GAZELLE_SHOW_COVERAGE = procutil.EnvVar("SCALA_GAZELLE_SHOW_COVERAGE")
	SCALA_GAZELLE_SHOW_PROGRESS = procutil.EnvVar("SCALA_GAZELLE_SHOW_PROGRESS")
	SCALA_GAZELLE_WATCH_DIR     = procutil.EnvVar("SCALA_GAZELLE_WATCH_DIR")
	TEST_TMPDIR                 = procutil.EnvVar("TEST_TMPDIR")
)

func PrintEnv(printf func(format string, args ...any)) {
	for _, env := range os.Environ() {
		printf("env: %s", env)
	}
}
