package scala

import "github.com/stackb/scala-gazelle/pkg/procutil"

const (
	SCALA_GAZELLE_LOG_FILE      = procutil.EnvVar("SCALA_GAZELLE_LOG_FILE")
	SCALA_GAZELLE_SHOW_COVERAGE = procutil.EnvVar("SCALA_GAZELLE_SHOW_COVERAGE")
	SCALA_GAZELLE_SHOW_PROGRESS = procutil.EnvVar("SCALA_GAZELLE_SHOW_PROGRESS")
	TEST_TMPDIR                 = procutil.EnvVar("TEST_TMPDIR")
)
